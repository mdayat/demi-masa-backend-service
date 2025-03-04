package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/dbutil"
	"github.com/mdayat/demi-masa/internal/httputil"
	"github.com/mdayat/demi-masa/internal/services"
	"github.com/mdayat/demi-masa/repository"
	"github.com/rs/zerolog/log"
)

type AuthHandler interface {
	Register(res http.ResponseWriter, req *http.Request)
	Login(res http.ResponseWriter, req *http.Request)
	Refresh(res http.ResponseWriter, req *http.Request)
}

type auth struct {
	configs configs.Configs
	service services.AuthServicer
}

func NewAuthHandler(configs configs.Configs, service services.AuthServicer) AuthHandler {
	return &auth{
		configs: configs,
		service: service,
	}
}

type registrationResult struct {
	user         repository.User
	refreshToken string
	accessToken  string
}

func (a auth) Register(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	var reqBody struct {
		IdToken string `json:"id_token" validate:"required,jwt"`
	}

	if err := httputil.DecodeAndValidate(req, a.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	payload, err := a.service.ValidateIDToken(ctx, reqBody.IdToken)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusUnauthorized).Msg("invalid Id token")
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	isUserExist, err := a.service.CheckUserExistence(ctx, payload.Subject)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to check user existence")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if isUserExist {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusConflict).Msg("user already exist")
		http.Error(res, http.StatusText(http.StatusConflict), http.StatusConflict)
		return
	}

	retryableFunc := func(qtx *repository.Queries) (registrationResult, error) {
		user, err := qtx.InsertUser(ctx, payload.Subject)
		if err != nil {
			return registrationResult{}, fmt.Errorf("failed to insert user: %w", err)
		}

		now := time.Now()
		refreshTokenClaims := services.RefreshTokenClaims{
			Type: services.Refresh,
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        uuid.NewString(),
				ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(now),
				Issuer:    a.configs.Env.OriginURL,
				Subject:   user.ID,
			},
		}

		refreshToken, err := a.service.CreateRefreshToken(refreshTokenClaims)
		if err != nil {
			return registrationResult{}, fmt.Errorf("failed to create refresh token: %w", err)
		}

		accessTokenClaims := services.AccessTokenClaims{
			Type: services.Access,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(now),
				Issuer:    a.configs.Env.OriginURL,
				Subject:   user.ID,
			},
		}

		accessToken, err := a.service.CreateAccessToken(accessTokenClaims)
		if err != nil {
			return registrationResult{}, fmt.Errorf("failed to create access token: %w", err)
		}

		refreshTokenId, err := uuid.Parse(refreshTokenClaims.ID)
		if err != nil {
			return registrationResult{}, fmt.Errorf("failed to parse JTI to UUID: %w", err)
		}

		err = qtx.InsertRefreshToken(ctx, repository.InsertRefreshTokenParams{
			ID:        pgtype.UUID{Bytes: refreshTokenId, Valid: true},
			UserID:    user.ID,
			ExpiresAt: pgtype.Timestamptz{Time: refreshTokenClaims.ExpiresAt.Time, Valid: true},
		})

		if err != nil {
			return registrationResult{}, fmt.Errorf("failed to insert refresh token: %w", err)
		}

		registrationResult := registrationResult{
			user:         user,
			refreshToken: refreshToken,
			accessToken:  accessToken,
		}

		return registrationResult, nil
	}

	result, err := dbutil.RetryableTxWithData(ctx, a.configs.Db.Conn, a.configs.Db.Queries, retryableFunc)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to register user")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := struct {
		UserId       string `json:"user_id"`
		RefreshToken string `json:"refresh_token"`
		AccessToken  string `json:"access_token"`
	}{
		UserId:       result.user.ID,
		RefreshToken: result.refreshToken,
		AccessToken:  result.accessToken,
	}

	params := httputil.SendSuccessResponseParams{
		StatusCode: http.StatusCreated,
		ResBody:    resBody,
	}

	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusCreated).Msg("successfully registered user")
}

type authenticateUserResult struct {
	refreshToken string
	accessToken  string
}

func authenticateUser(ctx context.Context, auth auth, userId string) (authenticateUserResult, error) {
	now := time.Now()
	refreshTokenClaims := services.RefreshTokenClaims{
		Type: services.Refresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    auth.configs.Env.OriginURL,
			Subject:   userId,
		},
	}

	refreshToken, err := auth.service.CreateRefreshToken(refreshTokenClaims)
	if err != nil {
		return authenticateUserResult{}, fmt.Errorf("failed to create refresh token: %w", err)
	}

	accessTokenClaims := services.AccessTokenClaims{
		Type: services.Access,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    auth.configs.Env.OriginURL,
			Subject:   userId,
		},
	}

	accessToken, err := auth.service.CreateAccessToken(accessTokenClaims)
	if err != nil {
		return authenticateUserResult{}, fmt.Errorf("failed to create access token: %w", err)
	}

	refreshTokenId, err := uuid.Parse(refreshTokenClaims.ID)
	if err != nil {
		return authenticateUserResult{}, fmt.Errorf("failed to parse JTI to UUID: %w", err)
	}

	err = auth.service.InsertRefreshToken(ctx, repository.InsertRefreshTokenParams{
		ID:        pgtype.UUID{Bytes: refreshTokenId, Valid: true},
		UserID:    userId,
		ExpiresAt: pgtype.Timestamptz{Time: refreshTokenClaims.ExpiresAt.Time, Valid: true},
	})

	if err != nil {
		return authenticateUserResult{}, fmt.Errorf("failed to insert refresh token: %w", err)
	}

	authenticateUserResult := authenticateUserResult{
		refreshToken: refreshToken,
		accessToken:  accessToken,
	}

	return authenticateUserResult, nil
}

func (a auth) Login(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	var reqBody struct {
		IdToken string `json:"id_token" validate:"required,jwt"`
	}

	if err := httputil.DecodeAndValidate(req, a.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	payload, err := a.service.ValidateIDToken(ctx, reqBody.IdToken)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusUnauthorized).Msg("invalid Id token")
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	user, err := a.service.SelectUserById(ctx, payload.Subject)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusNotFound).Msg("user not found")
			http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select user")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	result, err := authenticateUser(ctx, a, user.ID)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to authenticate user")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := struct {
		UserId       string `json:"user_id"`
		RefreshToken string `json:"refresh_token"`
		AccessToken  string `json:"access_token"`
	}{
		UserId:       user.ID,
		RefreshToken: result.refreshToken,
		AccessToken:  result.accessToken,
	}

	params := httputil.SendSuccessResponseParams{
		StatusCode: http.StatusOK,
		ResBody:    resBody,
	}

	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully authenticated user")
}

func (a auth) Refresh(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	bearerToken := req.Header.Get("Authorization")
	if bearerToken == "" || !strings.Contains(bearerToken, "Bearer") {
		logger.Error().Err(errors.New("invalid authorization header")).Caller().Int("status_code", http.StatusUnauthorized).Send()
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	claims, err := a.service.ValidateRefreshToken(bearerToken)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusUnauthorized).Msg("invalid refresh token")
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	refreshToken, err := a.service.SelectRefreshTokenById(ctx, services.SelectRefreshTokenByIdParams{Jti: claims.ID, UserId: claims.Subject})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select refresh token")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if (err != nil && errors.Is(err, pgx.ErrNoRows)) || refreshToken.Revoked || refreshToken.ExpiresAt.Time.Before(time.Now()) {
		logger.Error().Err(errors.New("invalid refresh token")).Caller().Int("status_code", http.StatusUnauthorized).Send()
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	result, err := a.service.RotateRefreshToken(ctx, services.RotateRefreshTokenParams{
		Jti:       refreshToken.ID.String(),
		UserId:    refreshToken.UserID,
		ExpiresAt: refreshToken.ExpiresAt.Time,
	})

	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to rotate refresh token")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := struct {
		RefreshToken string `json:"refresh_token"`
		AccessToken  string `json:"access_token"`
	}{
		RefreshToken: result.RefreshToken,
		AccessToken:  result.AccessToken,
	}

	params := httputil.SendSuccessResponseParams{
		StatusCode: http.StatusCreated,
		ResBody:    resBody,
	}

	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusCreated).Msg("successfully rotated refresh token")
}
