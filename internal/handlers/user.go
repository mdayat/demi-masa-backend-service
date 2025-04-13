package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/mdayat/demi-masa-backend-service/internal/dtos"
	"github.com/mdayat/demi-masa-backend-service/internal/httputil"
	"github.com/mdayat/demi-masa-backend-service/internal/retryutil"
	"github.com/mdayat/demi-masa-backend-service/internal/services"
	"github.com/mdayat/demi-masa-backend-service/repository"
	"github.com/rs/zerolog/log"
)

type UserHandler interface {
	GetUser(res http.ResponseWriter, req *http.Request)
	DeleteUser(res http.ResponseWriter, req *http.Request)
	UpdateUser(res http.ResponseWriter, req *http.Request)
}

type user struct {
	configs configs.Configs
	service services.UserServicer
}

func NewUserHandler(configs configs.Configs, service services.UserServicer) UserHandler {
	return &user{
		configs: configs,
		service: service,
	}
}

func (u user) GetUser(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	userId := ctx.Value(userIdKey{}).(string)
	user, err := retryutil.RetryWithData(func() (repository.SelectUserRow, error) {
		userUUID, err := uuid.Parse(userId)
		if err != nil {
			return repository.SelectUserRow{}, fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		return u.configs.Db.Queries.SelectUser(ctx, pgtype.UUID{Bytes: userUUID, Valid: true})
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

	var userSubscription *dtos.UserSubscription
	if len(user.Subscription) != 0 {
		if err := json.Unmarshal(user.Subscription, &userSubscription); err != nil {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to unmarshal user subscription")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	resBody := dtos.UserResponse{
		Id:           user.ID.String(),
		Email:        user.Email,
		Name:         user.Name,
		Latitude:     user.Coordinates.P.Y,
		Longitude:    user.Coordinates.P.X,
		City:         user.City,
		Timezone:     user.Timezone,
		CreatedAt:    user.CreatedAt.Time.Format(time.RFC3339),
		Subscription: userSubscription,
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

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully got me")
}

func (u user) DeleteUser(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	userId := ctx.Value(userIdKey{}).(string)
	err := retryutil.RetryWithoutData(func() error {
		userUUID, err := uuid.Parse(userId)
		if err != nil {
			return fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		return u.configs.Db.Queries.DeleteUser(ctx, pgtype.UUID{Bytes: userUUID, Valid: true})
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusNotFound).Msg("user not found")
			http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to delete user")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	res.WriteHeader(http.StatusNoContent)
	logger.Info().Int("status_code", http.StatusNoContent).Msg("successfully deleted user")
}

func (u user) UpdateUser(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	var reqBody dtos.UserRequest
	if err := httputil.DecodeAndValidate(req, u.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if reqBody.Email == "" && reqBody.Password == "" && reqBody.Username == "" && reqBody.Latitude == "" && reqBody.Longitude == "" {
		res.WriteHeader(http.StatusNoContent)
		logger.Info().Int("status_code", http.StatusNoContent).Msg("no update performed")
		return
	}

	var email pgtype.Text
	if reqBody.Email != "" {
		email = pgtype.Text{String: reqBody.Email, Valid: true}
	}

	var password pgtype.Text
	if reqBody.Password != "" {
		password = pgtype.Text{String: reqBody.Password, Valid: true}
	}

	var name pgtype.Text
	if reqBody.Username != "" {
		name = pgtype.Text{String: reqBody.Username, Valid: true}
	}

	var city pgtype.Text
	var timezone pgtype.Text

	if reqBody.Latitude != "" && reqBody.Longitude != "" {
		result, err := u.service.ReverseGeocode(ctx, reqBody.Latitude, reqBody.Longitude)
		if err != nil {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to reverse geocode")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		city = pgtype.Text{String: result.City, Valid: true}
		timezone = pgtype.Text{String: result.Timezone, Valid: true}
	}

	userId := ctx.Value(userIdKey{}).(string)
	user, err := retryutil.RetryWithData(func() (repository.User, error) {
		userUUID, err := uuid.Parse(userId)
		if err != nil {
			return repository.User{}, fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		var coordinates pgtype.Point
		if reqBody.Latitude != "" && reqBody.Longitude != "" {
			latitude, longitude, err := u.service.ParseStringCoordinates(reqBody.Latitude, reqBody.Longitude)
			if err != nil {
				return repository.User{}, fmt.Errorf("failed to parse string coordinates: %w", err)
			}

			coordinates = pgtype.Point{P: pgtype.Vec2{X: longitude, Y: latitude}, Valid: true}
		}

		return u.configs.Db.Queries.UpdateUser(ctx, repository.UpdateUserParams{
			ID:          pgtype.UUID{Bytes: userUUID, Valid: true},
			Email:       email,
			Password:    password,
			Name:        name,
			Coordinates: coordinates,
			City:        city,
			Timezone:    timezone,
		})
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusNotFound).Msg("user not found")
			http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to update user")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	resBody := dtos.UserResponse{
		Id:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		Latitude:  user.Coordinates.P.Y,
		Longitude: user.Coordinates.P.X,
		City:      user.City,
		Timezone:  user.Timezone,
		CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
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

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully updated user")
}
