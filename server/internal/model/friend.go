package model

import (
	"time"

	"github.com/google/uuid"
)

type Friendship struct {
	ID          uuid.UUID
	RequesterID uuid.UUID
	AddresseeID uuid.UUID
	Status      string
	CreatedAt   time.Time
}

type FriendEntry struct {
	FriendshipID uuid.UUID              `json:"friendship_id"`
	UserID       uuid.UUID              `json:"user_id"`
	Email        string                 `json:"email"`
	PublicInfo   map[string]interface{} `json:"public_info"`
	CreatedAt    time.Time              `json:"created_at"`
}
