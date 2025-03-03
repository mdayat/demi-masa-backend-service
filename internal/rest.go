package internal

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/mdayat/demi-masa/configs"

	"github.com/mdayat/demi-masa/internal/handlers"
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
	authService := services.NewAuthService(r.configs)
	customMiddleware := handlers.NewMiddlewareHandler(r.configs, authService)

	r.router.Use(chiMiddleware.CleanPath)
	r.router.Use(chiMiddleware.RealIP)
	r.router.Use(customMiddleware.Logger)
	r.router.Use(chiMiddleware.Recoverer)
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
	r.router.Use(chiMiddleware.Heartbeat("/ping"))

	authHandler := handlers.NewAuthHandler(r.configs, authService)
	r.router.Post("/auth/register", authHandler.Register)
	r.router.Post("/auth/login", authHandler.Login)
	r.router.Post("/auth/refresh", authHandler.Refresh)

	r.router.Group(func(router chi.Router) {
		router.Use(customMiddleware.Authenticate)

		userService := services.NewUserService(r.configs)
		userHandler := handlers.NewUserHandler(r.configs, authService, userService)
		router.Get("/users/me", userHandler.GetMe)
	})

	if err := http.ListenAndServe(":8080", r.router); err != nil {
		return err
	}

	return nil
}
