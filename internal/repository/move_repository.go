package repository

import (
	"context"
	"errors"
	"time"

	"rize-api/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MoveRepository struct {
	db *pgxpool.Pool
}

func NewMoveRepository(db *pgxpool.Pool) *MoveRepository {
	return &MoveRepository{db: db}
}

func (r *MoveRepository) Log(ctx context.Context, userID string, in *models.LogMoveInput) (*models.MoveSession, error) {
	now := time.Now()
	startedAt := now.Add(-time.Duration(in.DurationMin) * time.Minute)
	s := &models.MoveSession{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO move_sessions (user_id, category, started_at, ended_at, duration_min, notes)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, category, started_at, ended_at, duration_min, notes, created_at`,
		userID, in.Category, startedAt, now, in.DurationMin, in.Notes,
	).Scan(&s.ID, &s.UserID, &s.Category, &s.StartedAt, &s.EndedAt, &s.DurationMin, &s.Notes, &s.CreatedAt)
	return s, err
}

func (r *MoveRepository) Start(ctx context.Context, userID string, in *models.StartMoveInput) (*models.MoveSession, error) {
	s := &models.MoveSession{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO move_sessions (user_id, category, started_at, ended_at, duration_min)
		 VALUES ($1, $2, NOW(), NOW(), 0)
		 RETURNING id, user_id, category, started_at, ended_at, duration_min, notes, created_at`,
		userID, in.Category,
	).Scan(&s.ID, &s.UserID, &s.Category, &s.StartedAt, &s.EndedAt, &s.DurationMin, &s.Notes, &s.CreatedAt)
	return s, err
}

func (r *MoveRepository) End(ctx context.Context, userID string, sessionID int64) (*models.MoveSession, error) {
	s := &models.MoveSession{}
	err := r.db.QueryRow(ctx,
		`UPDATE move_sessions
		 SET ended_at = NOW(),
		     duration_min = GREATEST(1, ROUND(EXTRACT(EPOCH FROM (NOW() - started_at)) / 60)::INT)
		 WHERE id = $1 AND user_id = $2
		 RETURNING id, user_id, category, started_at, ended_at, duration_min, notes, created_at`,
		sessionID, userID,
	).Scan(&s.ID, &s.UserID, &s.Category, &s.StartedAt, &s.EndedAt, &s.DurationMin, &s.Notes, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func (r *MoveRepository) GetToday(ctx context.Context, userID string) ([]*models.MoveSession, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, category, started_at, ended_at, duration_min, notes, created_at
		 FROM move_sessions
		 WHERE user_id = $1 AND started_at >= CURRENT_DATE
		 ORDER BY started_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMoves(rows)
}

func (r *MoveRepository) GetByDate(ctx context.Context, userID, date string) ([]*models.MoveSession, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, category, started_at, ended_at, duration_min, notes, created_at
		 FROM move_sessions
		 WHERE user_id = $1 AND started_at::DATE = $2::DATE
		 ORDER BY started_at ASC`,
		userID, date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMoves(rows)
}

func (r *MoveRepository) Delete(ctx context.Context, userID string, sessionID int64) (bool, error) {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM move_sessions WHERE id = $1 AND user_id = $2`,
		sessionID, userID,
	)
	return tag.RowsAffected() > 0, err
}

func (r *MoveRepository) GetNoMovementUsers(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id FROM users WHERE push_enabled = true
		 AND id NOT IN (
			SELECT DISTINCT user_id FROM move_sessions
			WHERE started_at >= CURRENT_DATE
		 )`)
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

func scanMoves(rows pgx.Rows) ([]*models.MoveSession, error) {
	var sessions []*models.MoveSession
	for rows.Next() {
		s := &models.MoveSession{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.Category, &s.StartedAt, &s.EndedAt,
			&s.DurationMin, &s.Notes, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}
