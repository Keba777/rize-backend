package models

import "time"

type DailyReport struct {
	ID                int64     `json:"id"`
	UserID            string    `json:"user_id"`
	ReportDate        string    `json:"report_date"`
	TotalSitMin       int       `json:"total_sit_min"`
	TotalMoveMin      int       `json:"total_move_min"`
	LongestSitMin     int       `json:"longest_sit_min"`
	SitSessionsCount  int       `json:"sit_sessions_count"`
	MoveSessionsCount int       `json:"move_sessions_count"`
	HealthScore       int       `json:"health_score"`
	Advice            *string   `json:"advice"`
	GeneratedAt       time.Time `json:"generated_at"`
}
