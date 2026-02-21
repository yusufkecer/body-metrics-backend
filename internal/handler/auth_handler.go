package handler

import (
	"encoding/json"
	"net/http"

	"github.com/yusufkecer/body-metrics-backend/internal/domain"
	"github.com/yusufkecer/body-metrics-backend/internal/middleware"
)

type AuthHandler struct {
	jwtSecret string
}

func NewAuthHandler(jwtSecret string) *AuthHandler {
	return &AuthHandler{jwtSecret: jwtSecret}
}

func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	var req domain.TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	token, err := middleware.GenerateToken(req.DeviceID, h.jwtSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, domain.TokenResponse{Token: token})
}
