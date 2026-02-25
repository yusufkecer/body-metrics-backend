package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/yusufkecer/body-metrics-backend/internal/domain"
)

type ResetTokenRepository struct {
	db *sql.DB
}

func NewResetTokenRepository(db *sql.DB) *ResetTokenRepository {
	return &ResetTokenRepository{db: db}
}

func (r *ResetTokenRepository) Create(accountID int64, token string, expiresAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO password_reset_tokens (account_id, token, expires_at) VALUES (?, ?, ?)`,
		accountID, token, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}
	return nil
}

func (r *ResetTokenRepository) GetValidByEmailAndToken(email, token string) (*domain.PasswordResetToken, error) {
	var t domain.PasswordResetToken
	var usedInt int
	err := r.db.QueryRow(`
		SELECT prt.id, prt.account_id, prt.token, prt.expires_at, prt.used
		FROM password_reset_tokens prt
		JOIN accounts a ON a.id = prt.account_id
		WHERE a.email = ? AND prt.token = ? AND prt.used = 0 AND prt.expires_at > NOW()
		ORDER BY prt.id DESC
		LIMIT 1`,
		email, token,
	).Scan(&t.ID, &t.AccountID, &t.Token, &t.ExpiresAt, &usedInt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reset token: %w", err)
	}
	t.Used = usedInt != 0
	return &t, nil
}

func (r *ResetTokenRepository) MarkUsed(id int64) error {
	_, err := r.db.Exec(
		`UPDATE password_reset_tokens SET used = 1 WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}
	return nil
}

func (r *ResetTokenRepository) DeleteExpiredByAccountID(accountID int64) error {
	_, err := r.db.Exec(
		`DELETE FROM password_reset_tokens WHERE account_id = ?`,
		accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete old tokens: %w", err)
	}
	return nil
}
