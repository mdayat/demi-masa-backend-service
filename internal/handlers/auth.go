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
	"github.com/mdayat/demi-masa-backend-service/internal/dtos"
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

func (a auth) Register(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	var reqBody dtos.RegisterRequest
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

	resBody := dtos.AuthResponse{
		RefreshToken: result.RefreshToken,
		AccessToken:  result.AccessToken,
		User: dtos.UserResponse{
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

	var reqBody dtos.LoginRequest
	if err := httputil.DecodeAndValidate(req, a.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	result, err := a.service.AuthenticateUser(ctx, services.AuthenticateUserParams{
		Email:    reqBody.Email,
		Password: reqBody.Password,
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

	resBody := dtos.AuthResponse{
		RefreshToken: result.RefreshToken,
		AccessToken:  result.AccessToken,
		User: dtos.UserResponse{
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

	authHeader := req.Header.Get("Authorization")
	splittedAuthHeader := strings.Split(authHeader, "Bearer ")
	if authHeader == "" || len(splittedAuthHeader) != 2 {
		logger.Error().Err(errors.New("invalid authorization header")).Caller().Int("status_code", http.StatusUnauthorized).Send()
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	claims, err := a.service.ValidateRefreshToken(splittedAuthHeader[1])
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusUnauthorized).Msg("invalid refresh token")
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	_, err = retryutil.RetryWithData(func() (repository.RefreshToken, error) {
		refreshTokenUUID, err := uuid.Parse(claims.ID)
		if err != nil {
			return repository.RefreshToken{}, fmt.Errorf("failed to parse JTI to UUID: %w", err)
		}

		userUUID, err := uuid.Parse(claims.Subject)
		if err != nil {
			return repository.RefreshToken{}, fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		return a.configs.Db.Queries.RevokeUserRefreshToken(ctx, repository.RevokeUserRefreshTokenParams{
			ID:     pgtype.UUID{Bytes: refreshTokenUUID, Valid: true},
			UserID: pgtype.UUID{Bytes: userUUID, Valid: true},
		})
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusUnauthorized).Msg("invalid refresh token")
			http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		} else {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to revoke user refresh token")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully revoked refresh token")
}

func (a auth) Refresh(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	authHeader := req.Header.Get("Authorization")
	splittedAuthHeader := strings.Split(authHeader, "Bearer ")
	if authHeader == "" || len(splittedAuthHeader) != 2 {
		logger.Error().Err(errors.New("invalid authorization header")).Caller().Int("status_code", http.StatusUnauthorized).Send()
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	claims, err := a.service.ValidateRefreshToken(splittedAuthHeader[1])
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

		return a.configs.Db.Queries.SelectUserRefreshToken(ctx, repository.SelectUserRefreshTokenParams{
			ID:     pgtype.UUID{Bytes: refreshTokenUUID, Valid: true},
			UserID: pgtype.UUID{Bytes: userUUID, Valid: true},
		})
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select user refresh token")
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

	resBody := dtos.RefreshResponse{
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
