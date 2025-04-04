package handlers

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/rs/zerolog"
)

var testServer *httptest.Server
var testClient *http.Client

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	env, err := configs.LoadEnv("../../.test.env")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	db, err := configs.NewDb(ctx, env.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	configs := configs.Configs{
		Env:      env,
		Db:       db,
		Validate: configs.NewValidate(),
	}

	authenticator := NewTestAuthenticator(configs)
	customMiddleware := NewMiddlewareHandler(configs, authenticator)
	rest := NewRestHandler(configs, customMiddleware)

	testServer = httptest.NewServer(rest.Router)
	defer testServer.Close()
	testClient = testServer.Client()

	exitCode := m.Run()
	os.Exit(exitCode)
}
