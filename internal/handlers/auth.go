package handlers

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
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
}

type auth struct {
	configs     configs.Configs
	authService services.AuthServicer
}

func NewAuthHandler(configs configs.Configs, authService services.AuthServicer) AuthHandler {
	return &auth{
		configs:     configs,
		authService: authService,
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

	payload, err := a.authService.ValidateIDToken(ctx, reqBody.IdToken)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusUnauthorized).Msg("invalid Id token")
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	isUserExist, err := a.authService.CheckUserExistence(ctx, payload.Subject)
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
		user, err := qtx.CreateUser(ctx, payload.Subject)
		if err != nil {
			return registrationResult{}, fmt.Errorf("failed to create user: %w", err)
		}

		result, err := a.authService.CreateRefreshToken(user.ID)
		if err != nil {
			return registrationResult{}, fmt.Errorf("failed to create refresh token: %w", err)
		}

		accessToken, err := a.authService.CreateAccessToken(user.ID)
		if err != nil {
			return registrationResult{}, fmt.Errorf("failed to create access token: %w", err)
		}

		refreshTokenId, err := uuid.Parse(result.Claims.ID)
		if err != nil {
			return registrationResult{}, fmt.Errorf("failed to parse JTI to UUID: %w", err)
		}

		err = qtx.CreateRefreshToken(ctx, repository.CreateRefreshTokenParams{
			ID:        pgtype.UUID{Bytes: refreshTokenId, Valid: true},
			UserID:    user.ID,
			ExpiresAt: pgtype.Timestamptz{Time: result.Claims.ExpiresAt.Time, Valid: true},
		})

		if err != nil {
			return registrationResult{}, fmt.Errorf("failed to create refresh token: %w", err)
		}

		return registrationResult{user, result.TokenString, accessToken}, nil
	}

	result, err := dbutil.RetryableTxWithData(ctx, a.configs.Db.Conn, a.configs.Db.Queries, retryableFunc)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to register a new user")
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

	logger.Info().Int("status_code", http.StatusCreated).Msg("successfully registered a new user")
}

func (a auth) Login(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("Login"))
}
