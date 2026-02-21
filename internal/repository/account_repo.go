package repository

import (
	"database/sql"
	"fmt"
)

type Account struct {
	ID           int64
	Email        string
	PasswordHash string
}

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(
	email string,
	passwordHash string,
) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO accounts (email, password_hash) VALUES (?, ?)`,
		email,
		passwordHash,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create account: %w", err)
	}
	return result.LastInsertId()
}

func (r *AccountRepository) GetByEmail(email string) (*Account, error) {
	var account Account
	err := r.db.QueryRow(
		`SELECT id, email, password_hash FROM accounts WHERE email = ?`,
		email,
	).Scan(&account.ID, &account.Email, &account.PasswordHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	return &account, nil
}
