package handlers

import (
	"net/http"

	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/httputil"
	"github.com/mdayat/demi-masa/internal/services"
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

	user, err := a.authService.CreateUser(ctx, services.CreateUserParams{
		UserId: payload.Subject,
	})

	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to create user")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := struct {
		UserId string `json:"user_id"`
	}{
		UserId: user.ID,
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
