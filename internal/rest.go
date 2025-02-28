package internal

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/mdayat/demi-masa/configs"

	"github.com/mdayat/demi-masa/internal/handlers"
	"github.com/mdayat/demi-masa/internal/middlewares"
	"github.com/mdayat/demi-masa/internal/services"
)

type RestServicer interface {
	Start() error
}

type rest struct {
	configs configs.Configs
	router  *chi.Mux
}

func NewRestService(configs configs.Configs) RestServicer {
	return &rest{
		configs: configs,
		router:  chi.NewRouter(),
	}
}

func (r rest) Start() error {
	r.router.Use(middleware.CleanPath)
	r.router.Use(middleware.RealIP)
	r.router.Use(middlewares.Logger)
	r.router.Use(middleware.Recoverer)
	r.router.Use(httprate.LimitByIP(100, 1*time.Minute))

	options := cors.Options{
		AllowedOrigins:   strings.Split(r.configs.Env.AllowedOrigins, ","),
		AllowedMethods:   []string{"GET", "PUT", "POST", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders:   []string{"User-Agent", "Content-Type", "Accept", "Accept-Encoding", "Accept-Language", "Cache-Control", "Connection", "Host", "Origin", "Referer", "Authorization"},
		ExposedHeaders:   []string{"Content-Length", "Location"},
		AllowCredentials: true,
		MaxAge:           300,
	}
	r.router.Use(cors.Handler(options))
	r.router.Use(middleware.Heartbeat("/ping"))

	authService := services.NewAuthService(r.configs)
	authHandler := handlers.NewAuthHandler(r.configs, authService)
	r.router.Post("/auth/register", authHandler.Register)
	r.router.Post("/auth/login", authHandler.Login)
	r.router.Post("/auth/refresh", authHandler.Refresh)

	if err := http.ListenAndServe(":8080", r.router); err != nil {
		return err
	}

	return nil
}
