package handlers

import (
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/mdayat/demi-masa-backend-service/internal/services"
)

func NewRestHandler(configs configs.Configs, customMiddleware MiddlewareHandler) *chi.Mux {
	router := chi.NewRouter()

	router.Use(chiMiddleware.CleanPath)
	router.Use(chiMiddleware.RealIP)
	router.Use(customMiddleware.Logger)
	router.Use(chiMiddleware.Recoverer)
	router.Use(httprate.LimitByIP(100, 1*time.Minute))

	options := cors.Options{
		AllowedOrigins:   strings.Split(configs.Env.AllowedOrigins, ","),
		AllowedMethods:   []string{"GET", "PUT", "POST", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders:   []string{"User-Agent", "Content-Type", "Accept", "Accept-Encoding", "Accept-Language", "Cache-Control", "Connection", "Host", "Origin", "Referer", "Authorization"},
		ExposedHeaders:   []string{"Content-Length", "Location"},
		AllowCredentials: true,
		MaxAge:           300,
	}
	router.Use(cors.Handler(options))
	router.Use(chiMiddleware.Heartbeat("/ping"))

	authService := services.NewAuthService(configs)
	authHandler := NewAuthHandler(configs, authService)
	router.Post("/auth/register", authHandler.Register)
	router.Post("/auth/login", authHandler.Login)
	router.Post("/auth/logout", authHandler.Logout)
	router.Get("/auth/refresh", authHandler.Refresh)

	paymentService := services.NewPaymentService(configs)
	paymentHandler := NewPaymentHandler(configs, paymentService)
	router.Post("/payments/callback", paymentHandler.TripayCallback)

	router.Group(func(r chi.Router) {
		r.Use(customMiddleware.Authenticate)

		userService := services.NewUserService(configs)
		userHandler := NewUserHandler(configs, userService)
		r.Get("/users/me", userHandler.GetUser)
		r.Delete("/users/me", userHandler.DeleteUser)
		r.Put("/users/me", userHandler.UpdateUser)

		prayerService := services.NewPrayerService(configs)
		prayerHandler := NewPrayerHandler(configs, prayerService)
		r.Get("/prayers", prayerHandler.GetPrayers)
		r.Put("/prayers/{prayerId}", prayerHandler.UpdatePrayer)

		r.Get("/invoices/active", paymentHandler.GetActiveInvoice)
		r.Post("/invoices", paymentHandler.CreateInvoice)
		r.Get("/payments", paymentHandler.GetPayments)

		planHandler := NewPlanHandler(configs)
		r.Get("/plans", planHandler.GetPlans)
		r.Get("/plans/{planId}", planHandler.GetPlan)

		taskHandler := NewTaskHandler(configs)
		r.Get("/tasks", taskHandler.GetTasks)
		r.Post("/tasks", taskHandler.CreateTask)
		r.Put("/tasks/{taskId}", taskHandler.UpdateTask)
		r.Delete("/tasks/{taskId}", taskHandler.DeleteTask)

		couponHandler := NewCouponHandler(configs)
		r.Get("/coupons/{couponCode}", couponHandler.GetCoupon)
	})

	return router
}
