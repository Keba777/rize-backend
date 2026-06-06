package models

import "time"

type MoveSession struct {
	ID          int64      `json:"id"`
	UserID      string     `json:"user_id"`
	Category    string     `json:"category"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     time.Time  `json:"ended_at"`
	DurationMin int        `json:"duration_min"`
	Notes       *string    `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
}

// used for timed (live) move sessions stored in-flight
type ActiveMoveSession struct {
	SessionID int64     `json:"session_id"`
	StartedAt time.Time `json:"started_at"`
}

type LogMoveInput struct {
	Category    string  `json:"category"     validate:"required,oneof=walk stretch exercise stand"`
	DurationMin int     `json:"duration_min" validate:"required,min=1,max=480"`
	Notes       *string `json:"notes"        validate:"omitempty,max=500"`
}

type StartMoveInput struct {
	Category string `json:"category" validate:"required,oneof=walk stretch exercise stand"`
}
