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

type UserProvider struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Provider   string
	ProviderID string
	CreatedAt  time.Time
}

type UserSearchResult struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}
