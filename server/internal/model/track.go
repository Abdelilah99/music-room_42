package model

import (
	"time"

	"github.com/google/uuid"
)

type Track struct {
	ID          uuid.UUID  `json:"id"`
	EventID     uuid.UUID  `json:"event_id"`
	ExternalID  string     `json:"external_id"`
	Title       string     `json:"title"`
	Artist      string     `json:"artist"`
	SuggestedBy *uuid.UUID `json:"suggested_by"`
	VoteCount   int        `json:"vote_count"`
	CreatedAt   time.Time  `json:"created_at"`
}

type SuggestTrackRequest struct {
	ExternalID string `json:"external_id" binding:"required"`
	Title      string `json:"title" binding:"required"`
	Artist     string `json:"artist" binding:"required"`
}
