package repository

import (
	"context"

	"rize-api/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PushRepository struct {
	db *pgxpool.Pool
}

func NewPushRepository(db *pgxpool.Pool) *PushRepository {
	return &PushRepository{db: db}
}

func (r *PushRepository) Save(ctx context.Context, userID string, in *models.SavePushInput) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (endpoint) DO UPDATE SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth`,
		userID, in.Endpoint, in.P256dh, in.Auth,
	)
	return err
}

func (r *PushRepository) Delete(ctx context.Context, userID, endpoint string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`,
		userID, endpoint,
	)
	return err
}

func (r *PushRepository) GetByUserID(ctx context.Context, userID string) ([]*models.PushSubscription, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, endpoint, p256dh, auth, created_at
		 FROM push_subscriptions WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []*models.PushSubscription
	for rows.Next() {
		s := &models.PushSubscription{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}
