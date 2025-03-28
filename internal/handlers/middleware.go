package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/mdayat/demi-masa-backend-service/internal/services"
	"github.com/rs/zerolog/log"
)

type MiddlewareHandler interface {
	Logger(next http.Handler) http.Handler
	Authenticate(next http.Handler) http.Handler
}

type middleware struct {
	configs     configs.Configs
	authService services.AuthServicer
}

func NewMiddlewareHandler(configs configs.Configs, authService services.AuthServicer) MiddlewareHandler {
	return &middleware{
		configs:     configs,
		authService: authService,
	}
}

func (m middleware) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		subLogger := log.
			With().
			Str("request_id", uuid.New().String()).
			Str("method", req.Method).
			Str("path", req.URL.Path).
			Str("client_ip", req.RemoteAddr).
			Logger()

		req = req.WithContext(subLogger.WithContext(req.Context()))
		next.ServeHTTP(res, req)
	})
}

type userIdKey struct{}

func (m middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		logger := log.Ctx(ctx).With().Logger()

		authHeader := req.Header.Get("Authorization")
		splittedAuthHeader := strings.Split(authHeader, "Bearer ")
		if authHeader == "" || len(splittedAuthHeader) != 2 {
			logger.Error().Err(errors.New("invalid authorization header")).Caller().Int("status_code", http.StatusUnauthorized).Send()
			http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		claims, err := m.authService.ValidateAccessToken(splittedAuthHeader[1])
		if err != nil {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusUnauthorized).Msg("invalid access token")
			http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		req = req.WithContext(context.WithValue(req.Context(), userIdKey{}, claims.Subject))
		next.ServeHTTP(res, req)
	})
}
