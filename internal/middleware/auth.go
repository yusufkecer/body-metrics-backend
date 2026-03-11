package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const AccountIDKey contextKey = "account_id"

type AccountChecker interface {
	ExistsByID(accountID int64) (bool, error)
}

func GenerateToken(accountID int64, email, secret string) (string, error) {
	claims := jwt.MapClaims{
		"account_id": accountID,
		"email":      email,
		"iat":        time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func AuthMiddleware(secret string, checker AccountChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeAuthError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			if tokenStr == header {
				writeAuthError(w, http.StatusUnauthorized, "invalid authorization format")
				return
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				writeAuthError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeAuthError(w, http.StatusUnauthorized, "invalid token claims")
				return
			}

			accountIDFloat, ok := claims["account_id"].(float64)
			if !ok {
				writeAuthError(w, http.StatusUnauthorized, "invalid account id in token")
				return
			}
			accountID := int64(accountIDFloat)

			if checker != nil {
				exists, err := checker.ExistsByID(accountID)
				if err != nil {
					writeAuthError(w, http.StatusInternalServerError, "failed to validate account")
					return
				}
				if !exists {
					writeAuthError(w, http.StatusUnauthorized, "account no longer exists")
					return
				}
			}

			ctx := context.WithValue(r.Context(), AccountIDKey, accountID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
