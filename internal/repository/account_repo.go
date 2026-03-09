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

func (r *AccountRepository) Delete(accountID int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete metrics for the user associated with this account
	_, err = tx.Exec(
		`DELETE um FROM user_metrics um
		 JOIN users u ON um.user_id = u.id
		 WHERE u.account_id = ?`,
		accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete user metrics: %w", err)
	}

	// Delete user profiles associated with this account
	_, err = tx.Exec(
		`DELETE FROM users WHERE account_id = ?`,
		accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete users: %w", err)
	}

	// Delete password reset tokens
	_, err = tx.Exec(
		`DELETE FROM password_reset_tokens WHERE account_id = ?`,
		accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete password reset tokens: %w", err)
	}

	// Delete the account
	_, err = tx.Exec(
		`DELETE FROM accounts WHERE id = ?`,
		accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
