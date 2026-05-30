package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID               uuid.UUID
	Email            string
	PasswordHash     string
	IsVerified       bool
	SubscriptionTier string
	CreatedAt        time.Time
}

type EmailVerification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     uuid.UUID
	CreatedAt time.Time
}

type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     uuid.UUID
	ExpiresAt time.Time
	UsedAt    *time.Time
}
