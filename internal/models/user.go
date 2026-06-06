package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         *string   `json:"name"`
	SitLimitMin  int       `json:"sit_limit_min"`
	MoveGoalMin  int       `json:"move_goal_min"`
	ReportTime   string    `json:"report_time"`
	PushEnabled  bool      `json:"push_enabled"`
	Timezone     string    `json:"timezone"`
	CreatedAt    time.Time `json:"created_at"`
}

type MagicLink struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateSettingsInput struct {
	SitLimitMin *int    `json:"sit_limit_min" validate:"omitempty,min=15,max=120"`
	MoveGoalMin *int    `json:"move_goal_min" validate:"omitempty,min=10,max=120"`
	ReportTime  *string `json:"report_time"   validate:"omitempty"`
	Timezone    *string `json:"timezone"      validate:"omitempty"`
}
