package handler

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/yusufkecer/body-metrics-backend/internal/domain"
	"github.com/yusufkecer/body-metrics-backend/internal/middleware"
	"github.com/yusufkecer/body-metrics-backend/internal/repository"
	"github.com/yusufkecer/body-metrics-backend/internal/service"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	jwtSecret      string
	repo           *repository.AccountRepository
	resetTokenRepo *repository.ResetTokenRepository
	emailService   *service.EmailService
}

func NewAuthHandler(
	jwtSecret string,
	repo *repository.AccountRepository,
	resetTokenRepo *repository.ResetTokenRepository,
	emailService *service.EmailService,
) *AuthHandler {
	return &AuthHandler{
		jwtSecret:      jwtSecret,
		repo:           repo,
		resetTokenRepo: resetTokenRepo,
		emailService:   emailService,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	if !strings.Contains(email, "@") || strings.Index(email, "@") == 0 || !strings.Contains(email[strings.Index(email, "@"):], ".") {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	accountID, err := h.repo.Create(email, string(passwordHash))
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			writeError(w, http.StatusConflict, "email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	token, err := middleware.GenerateToken(accountID, email, h.jwtSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusCreated, domain.TokenResponse{Token: token})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	if !strings.Contains(email, "@") || strings.Index(email, "@") == 0 || !strings.Contains(email[strings.Index(email, "@"):], ".") {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	account, err := h.repo.GetByEmail(email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to login")
		return
	}
	if account == nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(account.PasswordHash),
		[]byte(req.Password),
	)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := middleware.GenerateToken(account.ID, account.Email, h.jwtSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, domain.TokenResponse{Token: token})
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req domain.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Always return 200 to prevent email enumeration
		writeJSON(w, http.StatusOK, map[string]string{"message": "if the email exists, a code has been sent"})
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))

	// Always return OK regardless of whether the email exists
	go func() {
		account, err := h.repo.GetByEmail(email)
		if err != nil || account == nil {
			return
		}

		// Delete all existing tokens for this account
		_ = h.resetTokenRepo.DeleteExpiredByAccountID(account.ID)

		// Generate 6-digit OTP
		otp, err := generateOTP()
		if err != nil {
			return
		}

		expiresAt := time.Now().Add(15 * time.Minute)
		if err := h.resetTokenRepo.Create(account.ID, otp, expiresAt); err != nil {
			return
		}

		_ = h.emailService.SendPasswordReset(email, otp)
	}()

	writeJSON(w, http.StatusOK, map[string]string{"message": "if the email exists, a code has been sent"})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req domain.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Token == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email, token and password are required")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	resetToken, err := h.resetTokenRepo.GetValidByEmailAndToken(email, req.Token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify token")
		return
	}
	if resetToken == nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := h.repo.UpdatePassword(resetToken.AccountID, string(passwordHash)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	if err := h.resetTokenRepo.MarkUsed(resetToken.ID); err != nil {
		// Non-critical error — password was already updated
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password reset successful"})
}

func generateOTP() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	return fmt.Sprintf("%06d", n%1000000), nil
}
