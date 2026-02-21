package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/yusufkecer/body-metrics-backend/internal/domain"
	"github.com/yusufkecer/body-metrics-backend/internal/repository"
)

type UserHandler struct {
	repo *repository.UserRepository
}

func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var user domain.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	id, err := h.repo.Create(&user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	user.ID = id
	writeJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := h.repo.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.GetAll()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	if users == nil {
		users = []domain.User{}
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var fields map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.repo.Update(id, fields); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	user, err := h.repo.GetByID(id)
	if err != nil || user == nil {
		writeError(w, http.StatusInternalServerError, "failed to get updated user")
		return
	}

	writeJSON(w, http.StatusOK, user)
}
