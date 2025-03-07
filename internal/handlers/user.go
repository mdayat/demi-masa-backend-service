package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
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
