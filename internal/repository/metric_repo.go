package repository

import (
	"database/sql"
	"fmt"

	"github.com/yusufkecer/body-metrics-backend/internal/domain"
)

type MetricRepository struct {
	db *sql.DB
}

func NewMetricRepository(db *sql.DB) *MetricRepository {
	return &MetricRepository{db: db}
}

func (r *MetricRepository) Create(m *domain.UserMetric) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO user_metrics (user_id, date, weight, height, bmi, weight_diff, body_metric, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.UserID, m.Date, m.Weight, m.Height, m.BMI, m.WeightDiff, m.BodyMetric, m.CreatedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create metric: %w", err)
	}
	return result.LastInsertId()
}

func (r *MetricRepository) GetByUserID(userID int64) ([]domain.UserMetric, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, date, weight, height, bmi, weight_diff, body_metric, created_at
		 FROM user_metrics
		 WHERE user_id = ?
		 ORDER BY created_at ASC, id ASC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list metrics: %w", err)
	}
	defer rows.Close()

	var metrics []domain.UserMetric
	for rows.Next() {
		var m domain.UserMetric
		if err := rows.Scan(&m.ID, &m.UserID, &m.Date, &m.Weight, &m.Height, &m.BMI, &m.WeightDiff, &m.BodyMetric, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}
