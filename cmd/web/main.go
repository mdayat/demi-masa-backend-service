package main

import (
	"context"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/mdayat/demi-masa-backend-service/internal/handlers"
	"github.com/mdayat/demi-masa-backend-service/internal/services"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
		return filepath.Base(file) + ":" + strconv.Itoa(line)
	}
	logger := log.With().Caller().Logger()

	env, err := configs.LoadEnv()
	if err != nil {
		logger.Fatal().Err(err).Send()
	}

	ctx := context.TODO()
	db, err := configs.NewDb(ctx, env.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Send()
	}

	configs := configs.Configs{
		Env:      env,
		Db:       db,
		Validate: configs.NewValidate(),
	}

	authService := services.NewAuthService(configs)
	authenticator := handlers.NewProdAuthenticator(authService)
	customMiddleware := handlers.NewMiddlewareHandler(configs, authenticator)
	rest := handlers.NewRestHandler(configs, customMiddleware)

	if err := http.ListenAndServe(":8080", rest.Router); err != nil {
		logger.Fatal().Err(err).Send()
	}
}
