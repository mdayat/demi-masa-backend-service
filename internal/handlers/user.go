package handlers

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/httputil"
	"github.com/mdayat/demi-masa/internal/services"
	"github.com/rs/zerolog/log"
)

type UserHandler interface {
	GetMe(res http.ResponseWriter, req *http.Request)
}

type user struct {
	configs     configs.Configs
	authService services.AuthServicer
	userService services.UserServicer
}

func NewUserHandler(configs configs.Configs, authService services.AuthServicer, userService services.UserServicer) UserHandler {
	return &user{
		configs:     configs,
		authService: authService,
		userService: userService,
	}
}

func (u user) GetMe(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	userId := req.Context().Value(userIdKey{}).(string)
	if userId == "" {
		logger.Error().Err(errors.New("user not found")).Caller().Int("status_code", http.StatusNotFound).Send()
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	user, err := u.authService.SelectUserById(ctx, userId)
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

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully get me")
}
