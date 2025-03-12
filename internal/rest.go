package internal

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/mdayat/demi-masa-backend-service/internal/handlers"
	"github.com/mdayat/demi-masa-backend-service/internal/services"
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
	r.router.Post("/auth/logout", authHandler.Logout)
	r.router.Get("/auth/refresh", authHandler.Refresh)

	paymentService := services.NewPaymentService(r.configs)
	paymentHandler := handlers.NewPaymentHandler(r.configs, paymentService)
	r.router.Post("/payments/callback", paymentHandler.TripayCallback)

	r.router.Group(func(router chi.Router) {
		router.Use(customMiddleware.Authenticate)

		userService := services.NewUserService(r.configs)
		userHandler := handlers.NewUserHandler(r.configs, userService)
		router.Get("/users/me", userHandler.GetMe)
		router.Delete("/users/{userId}", userHandler.DeleteUser)
		router.Put("/users/{userId}/coordinates", userHandler.UpdateUserCoordinates)
		router.Get("/subscriptions/active", userHandler.GetActiveSubscription)

		prayerService := services.NewPrayerService(r.configs)
		prayerHandler := handlers.NewPrayerHandler(r.configs, prayerService)
		router.Get("/prayers", prayerHandler.GetPrayers)
		router.Put("/prayers/{prayerId}", prayerHandler.UpdatePrayerStatus)

		router.Get("/invoices/active", paymentHandler.GetActiveInvoice)
		router.Post("/invoices", paymentHandler.CreateInvoice)
		router.Get("/payments", paymentHandler.GetPayments)

		planHandler := handlers.NewPlanHandler(r.configs)
		router.Get("/plans", planHandler.GetPlans)
		router.Get("/plans/{planId}", planHandler.GetPlan)

		taskService := services.NewTaskService(r.configs)
		taskHandler := handlers.NewTaskHandler(r.configs, taskService)
		router.Get("/tasks", taskHandler.GetTasks)
		router.Post("/tasks", taskHandler.CreateTask)
		router.Put("/tasks/{taskId}", taskHandler.UpdateTask)
		router.Delete("/tasks/{taskId}", taskHandler.DeleteTask)
	})

	if err := http.ListenAndServe(":8080", r.router); err != nil {
		return err
	}

	return nil
}
