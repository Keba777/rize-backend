package models

import "time"

type SitSession struct {
	ID          int64      `json:"id"`
	UserID      string     `json:"user_id"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at"`
	DurationMin int        `json:"duration_min"`
	CreatedAt   time.Time  `json:"created_at"`
}
