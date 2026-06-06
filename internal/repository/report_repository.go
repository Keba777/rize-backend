package repository

import (
	"context"
	"errors"

	"rize-api/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportRepository struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Upsert(ctx context.Context, report *models.DailyReport) (*models.DailyReport, error) {
	out := &models.DailyReport{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO daily_reports
		   (user_id, report_date, total_sit_min, total_move_min, longest_sit_min,
		    sit_sessions_count, move_sessions_count, health_score, advice)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 ON CONFLICT (user_id, report_date) DO UPDATE SET
		   total_sit_min       = EXCLUDED.total_sit_min,
		   total_move_min      = EXCLUDED.total_move_min,
		   longest_sit_min     = EXCLUDED.longest_sit_min,
		   sit_sessions_count  = EXCLUDED.sit_sessions_count,
		   move_sessions_count = EXCLUDED.move_sessions_count,
		   health_score        = EXCLUDED.health_score,
		   advice              = EXCLUDED.advice,
		   generated_at        = NOW()
		 RETURNING id, user_id, report_date, total_sit_min, total_move_min, longest_sit_min,
		           sit_sessions_count, move_sessions_count, health_score, advice, generated_at`,
		report.UserID, report.ReportDate, report.TotalSitMin, report.TotalMoveMin,
		report.LongestSitMin, report.SitSessionsCount, report.MoveSessionsCount,
		report.HealthScore, report.Advice,
	).Scan(&out.ID, &out.UserID, &out.ReportDate, &out.TotalSitMin, &out.TotalMoveMin,
		&out.LongestSitMin, &out.SitSessionsCount, &out.MoveSessionsCount,
		&out.HealthScore, &out.Advice, &out.GeneratedAt)
	return out, err
}

func (r *ReportRepository) GetByDate(ctx context.Context, userID, date string) (*models.DailyReport, error) {
	out := &models.DailyReport{}
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, report_date, total_sit_min, total_move_min, longest_sit_min,
		        sit_sessions_count, move_sessions_count, health_score, advice, generated_at
		 FROM daily_reports WHERE user_id = $1 AND report_date = $2::DATE`,
		userID, date,
	).Scan(&out.ID, &out.UserID, &out.ReportDate, &out.TotalSitMin, &out.TotalMoveMin,
		&out.LongestSitMin, &out.SitSessionsCount, &out.MoveSessionsCount,
		&out.HealthScore, &out.Advice, &out.GeneratedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return out, err
}

func (r *ReportRepository) GetHistory(ctx context.Context, userID string, days int) ([]*models.DailyReport, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, report_date, total_sit_min, total_move_min, longest_sit_min,
		        sit_sessions_count, move_sessions_count, health_score, advice, generated_at
		 FROM daily_reports
		 WHERE user_id = $1 AND report_date >= CURRENT_DATE - ($2::INT - 1)
		 ORDER BY report_date DESC`,
		userID, days,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reports []*models.DailyReport
	for rows.Next() {
		out := &models.DailyReport{}
		if err := rows.Scan(&out.ID, &out.UserID, &out.ReportDate, &out.TotalSitMin, &out.TotalMoveMin,
			&out.LongestSitMin, &out.SitSessionsCount, &out.MoveSessionsCount,
			&out.HealthScore, &out.Advice, &out.GeneratedAt); err != nil {
			return nil, err
		}
		reports = append(reports, out)
	}
	return reports, rows.Err()
}
