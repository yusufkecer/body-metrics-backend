package repository

import (
	"database/sql"
	"errors"
	"fmt"
)

var ErrAccountNotFound = errors.New("account not found")

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

func (r *AccountRepository) UpdatePassword(accountID int64, passwordHash string) error {
	_, err := r.db.Exec(
		`UPDATE accounts SET password_hash = ? WHERE id = ?`,
		passwordHash, accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

func (r *AccountRepository) ExistsByID(accountID int64) (bool, error) {
	var exists int
	err := r.db.QueryRow(
		`SELECT 1 FROM accounts WHERE id = ? LIMIT 1`,
		accountID,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check account existence: %w", err)
	}
	return true, nil
}

func (r *AccountRepository) Delete(accountID int64) error {
	result, err := r.db.Exec(
		`DELETE FROM accounts WHERE id = ?`,
		accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to inspect delete account result: %w", err)
	}
	if rowsAffected == 0 {
		return ErrAccountNotFound
	}

	return nil
}
