package handlers

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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

	ctx := context.TODO()
	db, err := configs.NewDb(ctx, env.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	configs := configs.NewConfigs(env, db)
	authenticator := NewTestAuthenticator(configs)

	customMiddleware := NewMiddlewareHandler(configs, authenticator)
	router := NewRestHandler(configs, customMiddleware)

	testServer = httptest.NewServer(router)
	defer testServer.Close()
	testClient = testServer.Client()

	exitCode := m.Run()
	os.Exit(exitCode)
}
