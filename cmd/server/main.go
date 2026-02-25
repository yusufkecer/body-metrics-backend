package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/yusufkecer/body-metrics-backend/internal/config"
	"github.com/yusufkecer/body-metrics-backend/internal/db"
	"github.com/yusufkecer/body-metrics-backend/internal/handler"
	"github.com/yusufkecer/body-metrics-backend/internal/middleware"
	"github.com/yusufkecer/body-metrics-backend/internal/repository"
	"github.com/yusufkecer/body-metrics-backend/internal/service"
)

func main() {
	cfg := config.Load()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable must be set")
	}

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer database.Close()

	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	accountRepo := repository.NewAccountRepository(database)
	userRepo := repository.NewUserRepository(database)
	metricRepo := repository.NewMetricRepository(database)
	resetTokenRepo := repository.NewResetTokenRepository(database)

	emailService := service.NewEmailService(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPass,
		cfg.SMTPFrom,
	)

	authHandler := handler.NewAuthHandler(cfg.JWTSecret, accountRepo, resetTokenRepo, emailService)
	userHandler := handler.NewUserHandler(userRepo)
	metricHandler := handler.NewMetricHandler(metricRepo)

	// Rate limiters
	loginRL := middleware.NewRateLimiter(5, 15*time.Minute)
	forgotPasswordRL := middleware.NewRateLimiter(3, 60*time.Minute)

	r := mux.NewRouter()

	// Global middleware: CORS → Security Headers → MaxBytesReader
	r.Use(middleware.CORSMiddleware(cfg.AllowedOrigins))
	r.Use(middleware.SecurityHeaders)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			next.ServeHTTP(w, r)
		})
	})

	r.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods(http.MethodGet, http.MethodOptions)

	api := r.PathPrefix("/api/v1").Subrouter()

	api.Use(middleware.APIKeyMiddleware(cfg.APIKey))

	api.Handle("/auth/register", http.HandlerFunc(authHandler.Register)).Methods(http.MethodPost, http.MethodOptions)
	api.Handle("/auth/login", loginRL.Middleware(http.HandlerFunc(authHandler.Login))).Methods(http.MethodPost, http.MethodOptions)
	api.Handle("/auth/forgot-password", forgotPasswordRL.Middleware(http.HandlerFunc(authHandler.ForgotPassword))).Methods(http.MethodPost, http.MethodOptions)
	api.Handle("/auth/reset-password", http.HandlerFunc(authHandler.ResetPassword)).Methods(http.MethodPost, http.MethodOptions)

	protected := api.NewRoute().Subrouter()
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	protected.HandleFunc("/users", userHandler.Create).Methods(http.MethodPost, http.MethodOptions)
	protected.HandleFunc("/users", userHandler.GetAll).Methods(http.MethodGet, http.MethodOptions)
	protected.HandleFunc("/users/{id}", userHandler.GetByID).Methods(http.MethodGet, http.MethodOptions)
	protected.HandleFunc("/users/{id}", userHandler.Update).Methods(http.MethodPatch, http.MethodOptions)
	protected.HandleFunc("/users/{id}/metrics", metricHandler.Create).Methods(http.MethodPost, http.MethodOptions)
	protected.HandleFunc("/users/{id}/metrics", metricHandler.GetByUserID).Methods(http.MethodGet, http.MethodOptions)

	addr := ":" + cfg.Port
	log.Printf("server starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
