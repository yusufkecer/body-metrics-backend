package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/yusufkecer/body-metrics-backend/internal/domain"
	"github.com/yusufkecer/body-metrics-backend/internal/middleware"
	"github.com/yusufkecer/body-metrics-backend/internal/repository"
)

type MetricHandler struct {
	repo     *repository.MetricRepository
	userRepo *repository.UserRepository
}

func NewMetricHandler(
	repo *repository.MetricRepository,
	userRepo *repository.UserRepository,
) *MetricHandler {
	return &MetricHandler{repo: repo, userRepo: userRepo}
}

func (h *MetricHandler) Create(w http.ResponseWriter, r *http.Request) {
	accountID, ok := r.Context().Value(middleware.AccountIDKey).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid account context")
		return
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := h.userRepo.GetByIDAndAccountID(userID, accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to validate user ownership")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	var metric domain.UserMetric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	metric.UserID = userID

	id, err := h.repo.Create(&metric)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create metric")
		return
	}

	metric.ID = id
	writeJSON(w, http.StatusCreated, metric)
}

func (h *MetricHandler) GetByUserID(w http.ResponseWriter, r *http.Request) {
	accountID, ok := r.Context().Value(middleware.AccountIDKey).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid account context")
		return
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := h.userRepo.GetByIDAndAccountID(userID, accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to validate user ownership")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	metrics, err := h.repo.GetByUserID(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list metrics")
		return
	}
	if metrics == nil {
		metrics = []domain.UserMetric{}
	}

	writeJSON(w, http.StatusOK, metrics)
}
