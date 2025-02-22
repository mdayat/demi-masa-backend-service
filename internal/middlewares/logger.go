package middlewares

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func Logger(next http.Handler) http.Handler {
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
