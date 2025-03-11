package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/mdayat/demi-masa-backend-service/internal/httputil"
	"github.com/mdayat/demi-masa-backend-service/internal/retryutil"
	"github.com/mdayat/demi-masa-backend-service/internal/services"
	"github.com/mdayat/demi-masa-backend-service/repository"
	"github.com/rs/zerolog/log"
)

type AuthHandler interface {
	Register(res http.ResponseWriter, req *http.Request)
	Login(res http.ResponseWriter, req *http.Request)
	Logout(res http.ResponseWriter, req *http.Request)
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

type userResponse struct {
	Id        string  `json:"id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      string  `json:"city"`
	Timezone  string  `json:"timezone"`
	CreatedAt string  `json:"created_at"`
}

func (a auth) Register(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	var reqBody struct {
		Username string `json:"username" validate:"required"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	if err := httputil.DecodeAndValidate(req, a.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	userUUID := uuid.New()
	result, err := a.service.RegisterUser(ctx, services.RegisterUserParams{
		UserUUID:  pgtype.UUID{Bytes: userUUID, Valid: true},
		Username:  reqBody.Username,
		UserEmail: reqBody.Email,
		Password:  reqBody.Password,
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusConflict).Msg("user already exist")
			http.Error(res, http.StatusText(http.StatusConflict), http.StatusConflict)
		} else {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to register user")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	resBody := struct {
		RefreshToken string       `json:"refresh_token"`
		AccessToken  string       `json:"access_token"`
		User         userResponse `json:"user"`
	}{
		RefreshToken: result.RefreshToken,
		AccessToken:  result.AccessToken,
		User: userResponse{
			Id:        result.User.ID.String(),
			Email:     result.User.Email,
			Name:      result.User.Name,
			Latitude:  result.User.Coordinates.P.Y,
			Longitude: result.User.Coordinates.P.X,
			City:      result.User.City,
			Timezone:  result.User.Timezone,
			CreatedAt: result.User.CreatedAt.Time.Format(time.RFC3339),
		},
	}

	params := httputil.SendSuccessResponseParams{
		StatusCode: http.StatusCreated,
		ResBody:    resBody,
	}

	res.Header().Set("Location", fmt.Sprintf("%s/users/me", a.configs.Env.OriginURL))
	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusCreated).Msg("successfully registered user")
}

func (a auth) Login(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	var reqBody struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	if err := httputil.DecodeAndValidate(req, a.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	user, err := retryutil.RetryWithData(func() (repository.User, error) {
		return a.configs.Db.Queries.SelectUserByEmailAndPassword(ctx, repository.SelectUserByEmailAndPasswordParams{
			Email:    reqBody.Email,
			Password: reqBody.Password,
		})
	})

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

	result, err := a.service.AuthenticateUser(ctx, user.ID)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to authenticate user")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := struct {
		RefreshToken string       `json:"refresh_token"`
		AccessToken  string       `json:"access_token"`
		User         userResponse `json:"user"`
	}{
		RefreshToken: result.RefreshToken,
		AccessToken:  result.AccessToken,
		User: userResponse{
			Id:        user.ID.String(),
			Email:     user.Email,
			Name:      user.Name,
			Latitude:  user.Coordinates.P.Y,
			Longitude: user.Coordinates.P.X,
			City:      user.City,
			Timezone:  user.Timezone,
			CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
		},
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

func (a auth) Logout(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	var reqBody struct {
		UserId string `json:"user_id" validate:"required"`
	}

	if err := httputil.DecodeAndValidate(req, a.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	bearerToken := req.Header.Get("Authorization")
	if bearerToken == "" || !strings.Contains(bearerToken, "Bearer") {
		logger.Error().Err(errors.New("invalid authorization header")).Caller().Int("status_code", http.StatusUnauthorized).Send()
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	refreshToken := strings.Split(bearerToken, "Bearer ")[1]
	claims, err := a.service.ValidateRefreshToken(refreshToken)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusUnauthorized).Msg("invalid refresh token")
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	err = retryutil.RetryWithoutData(func() error {
		refreshTokenUUID, err := uuid.Parse(claims.ID)
		if err != nil {
			return fmt.Errorf("failed to parse JTI to UUID: %w", err)
		}

		userUUID, err := uuid.Parse(reqBody.UserId)
		if err != nil {
			return fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		return a.configs.Db.Queries.RevokeRefreshToken(ctx, repository.RevokeRefreshTokenParams{
			ID:     pgtype.UUID{Bytes: refreshTokenUUID, Valid: true},
			UserID: pgtype.UUID{Bytes: userUUID, Valid: true},
		})
	})

	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to revoke refresh token")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully revoked refresh token")
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

	claims, err := a.service.ValidateRefreshToken(strings.Split(bearerToken, "Bearer ")[1])
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusUnauthorized).Msg("invalid refresh token")
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	refreshToken, err := retryutil.RetryWithData(func() (repository.RefreshToken, error) {
		refreshTokenUUID, err := uuid.Parse(claims.ID)
		if err != nil {
			return repository.RefreshToken{}, fmt.Errorf("failed to parse JTI to UUID: %w", err)
		}

		userUUID, err := uuid.Parse(claims.Subject)
		if err != nil {
			return repository.RefreshToken{}, fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		return a.configs.Db.Queries.SelectRefreshTokenById(ctx, repository.SelectRefreshTokenByIdParams{
			ID:     pgtype.UUID{Bytes: refreshTokenUUID, Valid: true},
			UserID: pgtype.UUID{Bytes: userUUID, Valid: true},
		})
	})

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
		UserUUID:  refreshToken.UserID,
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
