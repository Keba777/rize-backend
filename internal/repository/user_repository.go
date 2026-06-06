package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"rize-api/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindOrCreate(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, email, name, sit_limit_min, move_goal_min, report_time, push_enabled, timezone, created_at
		 FROM users WHERE email = $1`, email,
	).Scan(&user.ID, &user.Email, &user.Name, &user.SitLimitMin, &user.MoveGoalMin,
		&user.ReportTime, &user.PushEnabled, &user.Timezone, &user.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		err = r.db.QueryRow(ctx,
			`INSERT INTO users (email) VALUES ($1)
			 RETURNING id, email, name, sit_limit_min, move_goal_min, report_time, push_enabled, timezone, created_at`,
			email,
		).Scan(&user.ID, &user.Email, &user.Name, &user.SitLimitMin, &user.MoveGoalMin,
			&user.ReportTime, &user.PushEnabled, &user.Timezone, &user.CreatedAt)
	}
	return user, err
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, email, name, sit_limit_min, move_goal_min, report_time, push_enabled, timezone, created_at
		 FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.Email, &user.Name, &user.SitLimitMin, &user.MoveGoalMin,
		&user.ReportTime, &user.PushEnabled, &user.Timezone, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

func (r *UserRepository) UpdateSettings(ctx context.Context, userID string, in *models.UpdateSettingsInput) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(ctx,
		`UPDATE users SET
			sit_limit_min = COALESCE($2, sit_limit_min),
			move_goal_min = COALESCE($3, move_goal_min),
			report_time   = COALESCE($4::TIME, report_time),
			timezone      = COALESCE($5, timezone)
		 WHERE id = $1
		 RETURNING id, email, name, sit_limit_min, move_goal_min, report_time, push_enabled, timezone, created_at`,
		userID, in.SitLimitMin, in.MoveGoalMin, in.ReportTime, in.Timezone,
	).Scan(&user.ID, &user.Email, &user.Name, &user.SitLimitMin, &user.MoveGoalMin,
		&user.ReportTime, &user.PushEnabled, &user.Timezone, &user.CreatedAt)
	return user, err
}

func (r *UserRepository) SetPushEnabled(ctx context.Context, userID string, enabled bool) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET push_enabled = $2 WHERE id = $1`, userID, enabled)
	return err
}

func (r *UserRepository) GetAllWithPushEnabled(ctx context.Context) ([]*models.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, email, name, sit_limit_min, move_goal_min, report_time, push_enabled, timezone, created_at
		 FROM users WHERE push_enabled = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.SitLimitMin, &u.MoveGoalMin,
			&u.ReportTime, &u.PushEnabled, &u.Timezone, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Magic link methods

func (r *UserRepository) CreateMagicLink(ctx context.Context, userID string, ttlMin int) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	expiresAt := time.Now().Add(time.Duration(ttlMin) * time.Minute)
	_, err := r.db.Exec(ctx,
		`INSERT INTO magic_links (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		userID, token, expiresAt,
	)
	return token, err
}

func (r *UserRepository) ConsumeMagicLink(ctx context.Context, token string) (string, error) {
	var userID string
	var expiresAt time.Time
	var used bool
	err := r.db.QueryRow(ctx,
		`SELECT user_id, expires_at, used FROM magic_links WHERE token = $1`, token,
	).Scan(&userID, &expiresAt, &used)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("invalid token")
	}
	if err != nil {
		return "", err
	}
	if used {
		return "", errors.New("token already used")
	}
	if time.Now().After(expiresAt) {
		return "", errors.New("token expired")
	}
	_, err = r.db.Exec(ctx, `UPDATE magic_links SET used = true WHERE token = $1`, token)
	return userID, err
}
