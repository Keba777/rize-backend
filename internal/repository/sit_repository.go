package repository

import (
	"context"
	"errors"
	"time"

	"rize-api/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SitRepository struct {
	db *pgxpool.Pool
}

func NewSitRepository(db *pgxpool.Pool) *SitRepository {
	return &SitRepository{db: db}
}

func scanSit(row pgx.Row) (*models.SitSession, error) {
	s := &models.SitSession{}
	err := row.Scan(&s.ID, &s.UserID, &s.StartedAt, &s.EndedAt, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	s.DurationMin = durationMin(s.StartedAt, s.EndedAt)
	return s, nil
}

func durationMin(start time.Time, end *time.Time) int {
	if end == nil {
		return int(time.Since(start).Minutes())
	}
	return int(end.Sub(start).Minutes())
}

func (r *SitRepository) Start(ctx context.Context, userID string) (*models.SitSession, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO sit_sessions (user_id, started_at) VALUES ($1, NOW())
		 RETURNING id, user_id, started_at, ended_at, created_at`,
		userID,
	)
	return scanSit(row)
}

func (r *SitRepository) End(ctx context.Context, userID string) (*models.SitSession, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE sit_sessions SET ended_at = NOW()
		 WHERE user_id = $1 AND ended_at IS NULL
		 RETURNING id, user_id, started_at, ended_at, created_at`,
		userID,
	)
	s, err := scanSit(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func (r *SitRepository) GetActive(ctx context.Context, userID string) (*models.SitSession, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, started_at, ended_at, created_at FROM sit_sessions
		 WHERE user_id = $1 AND ended_at IS NULL
		 ORDER BY started_at DESC LIMIT 1`,
		userID,
	)
	s, err := scanSit(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func (r *SitRepository) GetToday(ctx context.Context, userID string) ([]*models.SitSession, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, started_at, ended_at, created_at FROM sit_sessions
		 WHERE user_id = $1 AND started_at >= CURRENT_DATE
		 ORDER BY started_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []*models.SitSession
	for rows.Next() {
		s := &models.SitSession{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.StartedAt, &s.EndedAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.DurationMin = durationMin(s.StartedAt, s.EndedAt)
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *SitRepository) GetByDate(ctx context.Context, userID, date string) ([]*models.SitSession, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, started_at, ended_at, created_at FROM sit_sessions
		 WHERE user_id = $1 AND started_at::DATE = $2::DATE
		 ORDER BY started_at ASC`,
		userID, date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []*models.SitSession
	for rows.Next() {
		s := &models.SitSession{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.StartedAt, &s.EndedAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.DurationMin = durationMin(s.StartedAt, s.EndedAt)
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// GetUsersOverSitLimit returns user IDs with an active sit session exceeding their sit_limit_min
func (r *SitRepository) GetUsersOverSitLimit(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT s.user_id FROM sit_sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.ended_at IS NULL
		   AND EXTRACT(EPOCH FROM (NOW() - s.started_at)) / 60 > u.sit_limit_min
		   AND u.push_enabled = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
