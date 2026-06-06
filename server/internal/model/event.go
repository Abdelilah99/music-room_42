package model

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID         uuid.UUID  `json:"id"`
	OwnerID    uuid.UUID  `json:"owner_id"`
	Name       string     `json:"name"`
	Visibility string     `json:"visibility"`
	Lat        *float64   `json:"lat"`
	Lng        *float64   `json:"lng"`
	Radius     *float64   `json:"radius"`
	VoteStart  *time.Time `json:"vote_start"`
	VoteEnd    *time.Time `json:"vote_end"`
	License    int        `json:"license"`
	CreatedAt  time.Time  `json:"created_at"`
}

type CreateEventRequest struct {
	Name       string     `json:"name"       binding:"required,min=1"`
	Visibility string     `json:"visibility" binding:"required,oneof=public private"`
	Lat        *float64   `json:"lat"`
	Lng        *float64   `json:"lng"`
	Radius     *float64   `json:"radius"`
	VoteStart  *time.Time `json:"vote_start"`
	VoteEnd    *time.Time `json:"vote_end"`
	License    int        `json:"license"    binding:"min=0,max=2"`
}

type UpdateEventRequest struct {
	Name       *string    `json:"name"       binding:"omitempty,min=1"`
	Visibility *string    `json:"visibility" binding:"omitempty,oneof=public private"`
	Lat        *float64   `json:"lat"`
	Lng        *float64   `json:"lng"`
	Radius     *float64   `json:"radius"`
	VoteStart  *time.Time `json:"vote_start"`
	VoteEnd    *time.Time `json:"vote_end"`
	License    *int       `json:"license"    binding:"omitempty,min=0,max=2"`
}

type EventListFilter struct {
	Q      string
	Lat    *float64
	Lng    *float64
	Radius *float64
}
