package model

import (
	"time"

	"github.com/google/uuid"
)

type Playlist struct {
	ID         uuid.UUID `json:"id"`
	OwnerID    uuid.UUID `json:"owner_id"`
	Name       string    `json:"name"`
	Visibility string    `json:"visibility"`
	License    int       `json:"license"`
	CreatedAt  time.Time `json:"created_at"`
}

type PlaylistTrack struct {
	ID         uuid.UUID  `json:"id"`
	PlaylistID uuid.UUID  `json:"playlist_id"`
	ExternalID string     `json:"external_id"`
	Title      string     `json:"title"`
	Artist     string     `json:"artist"`
	Position   int        `json:"position"`
	AddedBy    *uuid.UUID `json:"added_by"`
	CreatedAt  time.Time  `json:"created_at"`
}

type PlaylistWithTracks struct {
	Playlist
	Tracks []PlaylistTrack `json:"tracks"`
}

type CreatePlaylistRequest struct {
	Name       string `json:"name"       binding:"required,min=1"`
	Visibility string `json:"visibility" binding:"required,oneof=public private"`
	License    int    `json:"license"    binding:"min=0,max=1"`
}

type UpdatePlaylistRequest struct {
	Name       *string `json:"name"       binding:"omitempty,min=1"`
	Visibility *string `json:"visibility" binding:"omitempty,oneof=public private"`
	License    *int    `json:"license"    binding:"omitempty,min=0,max=1"`
}

type PlaylistListFilter struct {
	Q string
}

type AddTrackRequest struct {
	ExternalID string `json:"external_id" binding:"required"`
	Title      string `json:"title"       binding:"required"`
	Artist     string `json:"artist"      binding:"required"`
}

type MoveTrackRequest struct {
	Position int `json:"position" binding:"required,min=1"`
}
