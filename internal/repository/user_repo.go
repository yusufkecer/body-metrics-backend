package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/yusufkecer/body-metrics-backend/internal/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(u *domain.User) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO users (name, surname, gender, avatar, height, birth_of_date)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		u.Name, u.Surname, u.Gender, u.Avatar, u.Height, u.BirthOfDate,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}
	return result.LastInsertId()
}

func (r *UserRepository) GetByID(id int64) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(
		`SELECT id, name, surname, gender, avatar, height, birth_of_date, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Name, &u.Surname, &u.Gender, &u.Avatar, &u.Height, &u.BirthOfDate, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetAll() ([]domain.User, error) {
	rows, err := r.db.Query(
		`SELECT id, name, surname, gender, avatar, height, birth_of_date, created_at, updated_at
		 FROM users ORDER BY id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Surname, &u.Gender, &u.Avatar, &u.Height, &u.BirthOfDate, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Update(id int64, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	allowed := map[string]bool{
		"name": true, "surname": true, "gender": true,
		"avatar": true, "height": true, "birth_of_date": true,
	}

	var setClauses []string
	var args []interface{}
	for k, v := range fields {
		if !allowed[k] {
			continue
		}
		setClauses = append(setClauses, k+" = ?")
		args = append(args, v)
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, id)
	query := "UPDATE users SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}
