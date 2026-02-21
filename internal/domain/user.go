package domain

import "time"

type User struct {
	ID          int64     `json:"id"`
	Name        *string   `json:"name"`
	Surname     *string   `json:"surname"`
	Gender      *int      `json:"gender"`
	Avatar      *string   `json:"avatar"`
	Height      *int      `json:"height"`
	BirthOfDate *string   `json:"birthOfDate"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
