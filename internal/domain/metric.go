package domain

type UserMetric struct {
	ID         int64    `json:"id"`
	UserID     int64    `json:"user_id"`
	Date       string   `json:"date"`
	Weight     *float64 `json:"weight"`
	Height     int      `json:"height"`
	BMI        float64  `json:"bmi"`
	WeightDiff *float64 `json:"weight_diff"`
	BodyMetric *string  `json:"body_metric"`
	CreatedAt  *string  `json:"created_at"`
}
