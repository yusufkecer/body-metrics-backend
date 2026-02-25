package domain

import "time"

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Email    string `json:"email"`
	Token    string `json:"token"`
	Password string `json:"password"`
}

type PasswordResetToken struct {
	ID        int64
	AccountID int64
	Token     string
	ExpiresAt time.Time
	Used      bool
}
