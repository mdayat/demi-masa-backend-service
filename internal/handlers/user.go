package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/httputil"
	"github.com/mdayat/demi-masa/internal/retryutil"
	"github.com/mdayat/demi-masa/internal/services"
	"github.com/mdayat/demi-masa/repository"
	"github.com/rs/zerolog/log"
)

type UserHandler interface {
	GetMe(res http.ResponseWriter, req *http.Request)
	GetActiveSubscription(res http.ResponseWriter, req *http.Request)
	DeleteUser(res http.ResponseWriter, req *http.Request)
	UpdateUserCoordinates(res http.ResponseWriter, req *http.Request)
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

func (u user) GetMe(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	userId := ctx.Value(userIdKey{}).(string)
	if userId == "" {
		logger.Error().Err(errors.New("user not found")).Caller().Int("status_code", http.StatusNotFound).Send()
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	user, err := retryutil.RetryWithData(func() (repository.User, error) {
		return u.configs.Db.Queries.SelectUserById(ctx, userId)
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

	resBody := struct {
		UserId string `json:"user_id"`
	}{
		UserId: user.ID,
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

func (u user) GetActiveSubscription(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	userId := ctx.Value(userIdKey{}).(string)
	subscription, err := retryutil.RetryWithData(func() (repository.Subscription, error) {
		return u.configs.Db.Queries.SelectActiveSubscription(ctx, userId)
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select active subscription")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	params := httputil.SendSuccessResponseParams{StatusCode: http.StatusOK}
	if err == nil {
		resBody := struct {
			Id        string `json:"id"`
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
		}{
			Id:        subscription.ID.String(),
			StartDate: subscription.StartDate.Time.Format(time.RFC3339),
			EndDate:   subscription.EndDate.Time.Format(time.RFC3339),
		}

		params.ResBody = resBody
	}

	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully got active subscription")
}

func (u user) DeleteUser(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	userId := chi.URLParam(req, "userId")
	err := retryutil.RetryWithoutData(func() error {
		return u.configs.Db.Queries.DeleteUserByID(ctx, userId)
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

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully deleted user")
}

func (u user) UpdateUserCoordinates(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	var reqBody struct {
		Latitude  float64 `json:"latitude" validate:"required"`
		Longitude float64 `json:"longitude" validate:"required"`
	}

	if err := httputil.DecodeAndValidate(req, u.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	result, err := u.service.ReverseGeocode(ctx, reqBody.Latitude, reqBody.Longitude)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to reverse geocode")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if result.City == "" || result.Timezone == "" {
		logger.Error().Err(errors.New("empty reverse geocode result")).Caller().Int("status_code", http.StatusInternalServerError).Send()
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	userId := chi.URLParam(req, "userId")
	err = retryutil.RetryWithoutData(func() error {
		return u.configs.Db.Queries.UpdateUserCoordinatesById(ctx, repository.UpdateUserCoordinatesByIdParams{
			ID:          userId,
			Coordinates: pgtype.Point{P: pgtype.Vec2{X: reqBody.Longitude, Y: reqBody.Latitude}, Valid: true},
		})
	})

	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to update user coordinates")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := struct {
		TimeZone string `json:"time_zone"`
		City     string `json:"city"`
	}{
		TimeZone: result.Timezone,
		City:     result.City,
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

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully updated user coordinates")
}
