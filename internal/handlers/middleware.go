package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/mdayat/demi-masa-backend-service/internal/retryutil"
	"github.com/mdayat/demi-masa-backend-service/internal/services"
	"github.com/mdayat/demi-masa-backend-service/repository"
	"github.com/rs/zerolog/log"
)

type Authenticator interface {
	Authenticate(next http.Handler) http.Handler
}

type userIdKey struct{}

type prodAuthenticator struct {
	authService services.AuthServicer
}

func NewProdAuthenticator(authService services.AuthServicer) Authenticator {
	return &prodAuthenticator{
		authService: authService,
	}
}

func (p prodAuthenticator) Authenticate(next http.Handler) http.Handler {
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

		claims, err := p.authService.ValidateAccessToken(splittedAuthHeader[1])
		if err != nil {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusUnauthorized).Msg("invalid access token")
			http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		req = req.WithContext(context.WithValue(req.Context(), userIdKey{}, claims.Subject))
		next.ServeHTTP(res, req)
	})
}

type testAuthenticator struct {
	configs configs.Configs
}

func NewTestAuthenticator(configs configs.Configs) Authenticator {
	return &testAuthenticator{
		configs: configs,
	}
}

func (t testAuthenticator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		logger := log.Ctx(ctx).With().Logger()

		testUser, err := retryutil.RetryWithData(func() (repository.User, error) {
			return t.configs.Db.Queries.SelectUserByEmail(ctx, "example@gmail.com")
		})

		if err != nil {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Send()
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		req = req.WithContext(context.WithValue(ctx, userIdKey{}, testUser.ID.String()))
		next.ServeHTTP(res, req)
	})
}

type MiddlewareHandler interface {
	Logger(next http.Handler) http.Handler
	Authenticate(next http.Handler) http.Handler
}

type middleware struct {
	configs       configs.Configs
	authenticator Authenticator
}

func NewMiddlewareHandler(configs configs.Configs, authenticator Authenticator) MiddlewareHandler {
	return &middleware{
		configs:       configs,
		authenticator: authenticator,
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

func (m middleware) Authenticate(next http.Handler) http.Handler {
	return m.authenticator.Authenticate(next)
}
