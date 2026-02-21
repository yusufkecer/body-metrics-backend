package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/yusufkecer/body-metrics-backend/internal/config"
	"github.com/yusufkecer/body-metrics-backend/internal/db"
	"github.com/yusufkecer/body-metrics-backend/internal/handler"
	"github.com/yusufkecer/body-metrics-backend/internal/middleware"
	"github.com/yusufkecer/body-metrics-backend/internal/repository"
)

func main() {
	// 1. Config
	cfg := config.Load()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable must be set")
	}

	// 2. Database
	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer database.Close()

	// 3. Migrations
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	// 4. Repositories
	accountRepo := repository.NewAccountRepository(database)
	userRepo := repository.NewUserRepository(database)
	metricRepo := repository.NewMetricRepository(database)

	// 5. Handlers
	authHandler := handler.NewAuthHandler(cfg.JWTSecret, accountRepo)
	userHandler := handler.NewUserHandler(userRepo)
	metricHandler := handler.NewMetricHandler(metricRepo)

	// 6. Router
	r := mux.NewRouter()

	// Global: request body size limit (1 MB)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			next.ServeHTTP(w, r)
		})
	})

	// Health check (API key gerektirmez)
	r.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods(http.MethodGet)

	api := r.PathPrefix("/api/v1").Subrouter()

	// API Key middleware — tüm route'lara uygulanır
	api.Use(middleware.APIKeyMiddleware(cfg.APIKey))

	// Public routes
	api.HandleFunc("/auth/register", authHandler.Register).Methods(http.MethodPost)
	api.HandleFunc("/auth/login", authHandler.Login).Methods(http.MethodPost)

	// Protected routes — JWT middleware
	protected := api.NewRoute().Subrouter()
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	protected.HandleFunc("/users", userHandler.Create).Methods(http.MethodPost)
	protected.HandleFunc("/users", userHandler.GetAll).Methods(http.MethodGet)
	protected.HandleFunc("/users/{id}", userHandler.GetByID).Methods(http.MethodGet)
	protected.HandleFunc("/users/{id}", userHandler.Update).Methods(http.MethodPatch)
	protected.HandleFunc("/users/{id}/metrics", metricHandler.Create).Methods(http.MethodPost)
	protected.HandleFunc("/users/{id}/metrics", metricHandler.GetByUserID).Methods(http.MethodGet)

	// 7. Server
	addr := ":" + cfg.Port
	log.Printf("server starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
