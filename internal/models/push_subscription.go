package models

import "time"

type PushSubscription struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	CreatedAt time.Time `json:"created_at"`
}

type SavePushInput struct {
	Endpoint string `json:"endpoint" validate:"required,url"`
	P256dh   string `json:"p256dh"   validate:"required"`
	Auth     string `json:"auth"     validate:"required"`
}

type DeletePushInput struct {
	Endpoint string `json:"endpoint" validate:"required"`
}
