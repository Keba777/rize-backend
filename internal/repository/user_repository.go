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

const userCols = `id, email, name, sit_limit_min, move_goal_min, report_time, push_enabled, timezone, created_at`

func scanUser(row interface{ Scan(...any) error }, u *models.User) error {
	return row.Scan(&u.ID, &u.Email, &u.Name, &u.SitLimitMin, &u.MoveGoalMin,
		&u.ReportTime, &u.PushEnabled, &u.Timezone, &u.CreatedAt)
}

// FindOrCreate is used by the magic-link flow.
func (r *UserRepository) FindOrCreate(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	err := scanUser(r.db.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE email = $1`, email), user)

	if errors.Is(err, pgx.ErrNoRows) {
		err = scanUser(r.db.QueryRow(ctx,
			`INSERT INTO users (email, email_verified) VALUES ($1, true)
			 RETURNING `+userCols, email), user)
	}
	return user, err
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}
	err := scanUser(r.db.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE id = $1`, id), user)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

// FindByEmailWithAuth returns the user plus private auth fields needed for password login.
func (r *UserRepository) FindByEmailWithAuth(ctx context.Context, email string) (*models.User, *string, bool, error) {
	user := &models.User{}
	var passwordHash *string
	var emailVerified bool
	err := r.db.QueryRow(ctx,
		`SELECT `+userCols+`, password_hash, email_verified FROM users WHERE email = $1`, email,
	).Scan(&user.ID, &user.Email, &user.Name, &user.SitLimitMin, &user.MoveGoalMin,
		&user.ReportTime, &user.PushEnabled, &user.Timezone, &user.CreatedAt,
		&passwordHash, &emailVerified)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, false, nil
	}
	return user, passwordHash, emailVerified, err
}

// CreateWithPassword creates a new user with a bcrypt password hash and email_verified=false.
func (r *UserRepository) CreateWithPassword(ctx context.Context, email, passwordHash string) (*models.User, error) {
	user := &models.User{}
	err := scanUser(r.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, email_verified) VALUES ($1, $2, false)
		 RETURNING `+userCols, email, passwordHash), user)
	return user, err
}

// FindOrCreateByGoogle finds by google_id, links to existing email, or creates a new verified user.
func (r *UserRepository) FindOrCreateByGoogle(ctx context.Context, googleID, email string, name *string) (*models.User, error) {
	user := &models.User{}

	// 1. Find by google_id
	err := scanUser(r.db.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE google_id = $1`, googleID), user)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// 2. Link google_id to existing email account
	err = scanUser(r.db.QueryRow(ctx,
		`UPDATE users SET google_id = $2, email_verified = true, name = COALESCE(name, $3)
		 WHERE email = $1 RETURNING `+userCols, email, googleID, name), user)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// 3. Create new user
	err = scanUser(r.db.QueryRow(ctx,
		`INSERT INTO users (email, google_id, name, email_verified) VALUES ($1, $2, $3, true)
		 RETURNING `+userCols, email, googleID, name), user)
	return user, err
}

func (r *UserRepository) SetEmailVerified(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET email_verified = true WHERE id = $1`, userID)
	return err
}

func (r *UserRepository) UpdateSettings(ctx context.Context, userID string, in *models.UpdateSettingsInput) (*models.User, error) {
	user := &models.User{}
	err := scanUser(r.db.QueryRow(ctx,
		`UPDATE users SET
			sit_limit_min = COALESCE($2, sit_limit_min),
			move_goal_min = COALESCE($3, move_goal_min),
			report_time   = COALESCE($4::TIME, report_time),
			timezone      = COALESCE($5, timezone)
		 WHERE id = $1 RETURNING `+userCols,
		userID, in.SitLimitMin, in.MoveGoalMin, in.ReportTime, in.Timezone), user)
	return user, err
}

func (r *UserRepository) SetPushEnabled(ctx context.Context, userID string, enabled bool) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET push_enabled = $2 WHERE id = $1`, userID, enabled)
	return err
}

func (r *UserRepository) GetAllWithPushEnabled(ctx context.Context) ([]*models.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+userCols+` FROM users WHERE push_enabled = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := scanUser(rows, u); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Magic link methods

func (r *UserRepository) CreateMagicLink(ctx context.Context, userID string, ttlMin int) (string, error) {
	token, err := randomHex(32)
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(time.Duration(ttlMin) * time.Minute)
	_, err = r.db.Exec(ctx,
		`INSERT INTO magic_links (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		userID, token, expiresAt)
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
	if err != nil {
		return "", err
	}
	// Magic link click = email verified
	_, _ = r.db.Exec(ctx, `UPDATE users SET email_verified = true WHERE id = $1`, userID)
	return userID, nil
}

// Email verify token methods

func (r *UserRepository) CreateEmailVerifyToken(ctx context.Context, userID string, ttlMin int) (string, error) {
	token, err := randomHex(32)
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(time.Duration(ttlMin) * time.Minute)
	_, err = r.db.Exec(ctx,
		`INSERT INTO email_verify_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		userID, token, expiresAt)
	return token, err
}

func (r *UserRepository) ConsumeEmailVerifyToken(ctx context.Context, token string) (string, error) {
	var userID string
	var expiresAt time.Time
	var used bool
	err := r.db.QueryRow(ctx,
		`SELECT user_id, expires_at, used FROM email_verify_tokens WHERE token = $1`, token,
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
	_, err = r.db.Exec(ctx, `UPDATE email_verify_tokens SET used = true WHERE token = $1`, token)
	return userID, err
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
