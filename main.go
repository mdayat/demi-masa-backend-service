package main

import (
	"context"
	"path/filepath"
	"strconv"

	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal"

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

	rest := internal.NewRestService(env, db)
	if err := rest.Start(); err != nil {
		logger.Fatal().Err(err).Send()
	}
}
