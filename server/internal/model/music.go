package model

// TrackDTO represents a normalized track object returned to the mobile app
type TrackDTO struct {
	ExternalID string `json:"external_id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	PreviewURL string `json:"preview_url"`
	CoverURL   string `json:"cover_url"`
}
