package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/yusufkecer/body-metrics-backend/internal/config"
	"github.com/yusufkecer/body-metrics-backend/internal/db"
	"github.com/yusufkecer/body-metrics-backend/internal/domain"
	"github.com/yusufkecer/body-metrics-backend/internal/handler"
	"github.com/yusufkecer/body-metrics-backend/internal/middleware"
	"github.com/yusufkecer/body-metrics-backend/internal/repository"
	"github.com/yusufkecer/body-metrics-backend/internal/service"
)

func setupTestDB(t *testing.T) *sql.DB {
	cfg := &config.Config{
		DBHost:     "localhost",
		DBPort:     "3306",
		DBUser:     "bodymetrics",
		DBPassword: "bodymetrics_pass",
		DBName:     "test_bodymetrics",
	}

	database, err := db.Connect(cfg)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}

	if err := db.RunMigrations(database); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Clean up tables
	tables := []string{"user_metrics", "users", "password_reset_tokens", "accounts"}
	for _, table := range tables {
		_, err := database.Exec("DELETE FROM " + table)
		if err != nil {
			t.Fatalf("failed to clean up table %s: %v", table, err)
		}
	}

	return database
}

func setupRouter(database *sql.DB) *mux.Router {
	accountRepo := repository.NewAccountRepository(database)
	userRepo := repository.NewUserRepository(database)
	metricRepo := repository.NewMetricRepository(database)
	resetTokenRepo := repository.NewResetTokenRepository(database)
	emailService := service.NewEmailService("", "noreply@test.local")

	jwtSecret := "test-secret"

	authHandler := handler.NewAuthHandler(jwtSecret, accountRepo, resetTokenRepo, emailService)
	userHandler := handler.NewUserHandler(userRepo)
	metricHandler := handler.NewMetricHandler(metricRepo, userRepo)

	r := mux.Router{}
	api := r.PathPrefix("/api/v1").Subrouter()

	api.Handle("/auth/register", http.HandlerFunc(authHandler.Register)).Methods(http.MethodPost)
	api.Handle("/auth/login", http.HandlerFunc(authHandler.Login)).Methods(http.MethodPost)

	protected := api.NewRoute().Subrouter()
	protected.Use(middleware.AuthMiddleware(jwtSecret))

	protected.HandleFunc("/users", userHandler.Create).Methods(http.MethodPost)
	protected.HandleFunc("/users", userHandler.GetAll).Methods(http.MethodGet)
	protected.HandleFunc("/users/{id}", userHandler.Update).Methods(http.MethodPatch)
	protected.HandleFunc("/users/{id}", userHandler.GetByID).Methods(http.MethodGet)
	protected.HandleFunc("/users/{id}/metrics", metricHandler.Create).Methods(http.MethodPost)
	protected.HandleFunc("/users/{id}/metrics", metricHandler.GetByUserID).Methods(http.MethodGet)

	return &r
}

func TestRegisterAndLogin(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	router := setupRouter(database)

	email := "test@example.com"
	password := "password123"

	// Register
	reqBody, _ := json.Marshal(domain.TokenRequest{
		Email:    email,
		Password: password,
	})

	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201 Created, got %d", rr.Code)
	}

	var res domain.TokenResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal register response: %v", err)
	}

	if res.Token == "" {
		t.Errorf("expected token, got empty string")
	}

	// Login
	loginReq, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(reqBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginRr := httptest.NewRecorder()
	router.ServeHTTP(loginRr, loginReq)

	if loginRr.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", loginRr.Code)
	}

	var loginRes domain.TokenResponse
	if err := json.Unmarshal(loginRr.Body.Bytes(), &loginRes); err != nil {
		t.Fatalf("failed to unmarshal login response: %v", err)
	}

	if loginRes.Token == "" {
		t.Errorf("expected token, got empty string")
	}

	// Test protected route
	token := loginRes.Token
	getReq, _ := http.NewRequest("GET", "/api/v1/users", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)

	getRr := httptest.NewRecorder()
	router.ServeHTTP(getRr, getReq)

	if getRr.Code != http.StatusOK {
		t.Errorf("expected status 200 OK for GET /users, got %d", getRr.Code)
	}
}

// Keep the compiler happy

func TestCreateAndGetUser(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	router := setupRouter(database)

	email := "user@example.com"
	password := "password123"

	// Register
	reqBody, _ := json.Marshal(domain.TokenRequest{
		Email:    email,
		Password: password,
	})

	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var res domain.TokenResponse
	json.Unmarshal(rr.Body.Bytes(), &res)
	token := res.Token

	// Create User
	userBody, _ := json.Marshal(domain.User{
		Name:    ptrString("John"),
		Surname: ptrString("Doe"),
		Gender:  ptrInt(0),
		Height:  ptrInt(180),
	})

	createReq, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(userBody))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")

	createRr := httptest.NewRecorder()
	router.ServeHTTP(createRr, createReq)

	if createRr.Code != http.StatusCreated {
		t.Errorf("expected status 201 Created for POST /users, got %d", createRr.Code)
		t.Logf("Response body: %s", createRr.Body.String())
	}

	var createdUser domain.User
	if err := json.Unmarshal(createRr.Body.Bytes(), &createdUser); err != nil {
		t.Fatalf("failed to unmarshal user response: %v", err)
	}

	if *createdUser.Name != "John" {
		t.Errorf("expected user name John, got %s", *createdUser.Name)
	}

	// Try creating user again, it should fail
	createReq2, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(userBody))
	createReq2.Header.Set("Authorization", "Bearer "+token)
	createReq2.Header.Set("Content-Type", "application/json")

	createRr2 := httptest.NewRecorder()
	router.ServeHTTP(createRr2, createReq2)

	if createRr2.Code != http.StatusConflict {
		t.Errorf("expected status 409 Conflict for second POST /users, got %d", createRr2.Code)
	}

	// Get Users
	getReq, _ := http.NewRequest("GET", "/api/v1/users", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)

	getRr := httptest.NewRecorder()
	router.ServeHTTP(getRr, getReq)

	if getRr.Code != http.StatusOK {
		t.Errorf("expected status 200 OK for GET /users, got %d", getRr.Code)
	}

	var users []domain.User
	if err := json.Unmarshal(getRr.Body.Bytes(), &users); err != nil {
		t.Fatalf("failed to unmarshal users response: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}

	if users[0].ID != createdUser.ID {
		t.Errorf("expected user id %d, got %d", createdUser.ID, users[0].ID)
	}
}

func TestUpdateUser(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	router := setupRouter(database)

	email := "update@example.com"
	password := "password123"

	// Register
	reqBody, _ := json.Marshal(domain.TokenRequest{
		Email:    email,
		Password: password,
	})

	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var res domain.TokenResponse
	json.Unmarshal(rr.Body.Bytes(), &res)
	token := res.Token

	// Create User
	userBody, _ := json.Marshal(domain.User{
		Name: ptrString("OldName"),
	})

	createReq, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(userBody))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")

	createRr := httptest.NewRecorder()
	router.ServeHTTP(createRr, createReq)

	var createdUser domain.User
	json.Unmarshal(createRr.Body.Bytes(), &createdUser)
	userID := createdUser.ID

	// Update User
	updateBody, _ := json.Marshal(map[string]interface{}{
		"name": "NewName",
	})

	url := fmt.Sprintf("/api/v1/users/%d", userID)

	updateReq, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateReq.Header.Set("Content-Type", "application/json")

	updateRr := httptest.NewRecorder()
	router.ServeHTTP(updateRr, updateReq)

	if updateRr.Code != http.StatusOK {
		t.Errorf("expected status 200 OK for PATCH /users, got %d", updateRr.Code)
	}

	var updatedUser domain.User
	json.Unmarshal(updateRr.Body.Bytes(), &updatedUser)

	if *updatedUser.Name != "NewName" {
		t.Errorf("expected user name NewName, got %s", *updatedUser.Name)
	}
}

func TestMetricCreateAndGet(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	router := setupRouter(database)

	email := "metric@example.com"
	password := "password123"

	// Register
	reqBody, _ := json.Marshal(domain.TokenRequest{
		Email:    email,
		Password: password,
	})

	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var res domain.TokenResponse
	json.Unmarshal(rr.Body.Bytes(), &res)
	token := res.Token

	// Create User
	userBody, _ := json.Marshal(domain.User{
		Name: ptrString("MetricUser"),
	})

	createReq, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(userBody))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")

	createRr := httptest.NewRecorder()
	router.ServeHTTP(createRr, createReq)

	var createdUser domain.User
	json.Unmarshal(createRr.Body.Bytes(), &createdUser)
	userID := createdUser.ID

	url := fmt.Sprintf("/api/v1/users/%d/metrics", userID)

	metricBody, _ := json.Marshal(domain.UserMetric{
		Date:   "01-01-2023",
		Weight: ptrFloat(75.5),
		Height: 180,
		BMI:    23.3,
	})

	createMetricReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(metricBody))
	createMetricReq.Header.Set("Authorization", "Bearer "+token)
	createMetricReq.Header.Set("Content-Type", "application/json")

	createMetricRr := httptest.NewRecorder()
	router.ServeHTTP(createMetricRr, createMetricReq)

	if createMetricRr.Code != http.StatusCreated {
		t.Errorf("expected status 201 Created for POST /metrics, got %d", createMetricRr.Code)
	}

	var createdMetric domain.UserMetric
	json.Unmarshal(createMetricRr.Body.Bytes(), &createdMetric)

	if createdMetric.ID == 0 {
		t.Errorf("expected metric ID, got 0")
	}

	getMetricsReq, _ := http.NewRequest("GET", url, nil)
	getMetricsReq.Header.Set("Authorization", "Bearer "+token)

	getMetricsRr := httptest.NewRecorder()
	router.ServeHTTP(getMetricsRr, getMetricsReq)

	if getMetricsRr.Code != http.StatusOK {
		t.Errorf("expected status 200 OK for GET /metrics, got %d", getMetricsRr.Code)
	}

	var metrics []domain.UserMetric
	json.Unmarshal(getMetricsRr.Body.Bytes(), &metrics)

	if len(metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(metrics))
	}
	//

}
