package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"music-room/internal/model"
)

// MusicService defines the interface for music searches
type MusicService interface {
	SearchTracks(ctx context.Context, query string) ([]model.TrackDTO, error)
	GetTrack(ctx context.Context, id string) (*model.TrackDTO, error)
}

type deezerMusicService struct {
	client *http.Client
}

// NewMusicService returns a proxy service that hits the public Deezer API
func NewMusicService() MusicService {
	return &deezerMusicService{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Structs representing Deezer's JSON structure
type deezerResponse struct {
	Data  []deezerTrack `json:"data"`
	Error *deezerError  `json:"error"`
}

type deezerError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type deezerTrack struct {
	ID      int64        `json:"id"`
	Title   string       `json:"title"`
	Preview string       `json:"preview"`
	Artist  deezerArtist `json:"artist"`
	Album   deezerAlbum  `json:"album"`
}

type deezerArtist struct {
	Name string `json:"name"`
}

type deezerAlbum struct {
	Title       string `json:"title"`
	CoverMedium string `json:"cover_medium"`
}

// deezerTrackResponse is the GET /track/{id} payload: the track fields sit at
// the top level, with an optional error object Deezer returns for a bad id.
type deezerTrackResponse struct {
	deezerTrack
	Error *deezerError `json:"error"`
}

func (s *deezerMusicService) SearchTracks(ctx context.Context, query string) ([]model.TrackDTO, error) {
	// Construct URL
	reqURL := fmt.Sprintf("https://api.deezer.com/search?q=%s", url.QueryEscape(query))
	
	// Prepare request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	// Do request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("deezer API returned status %d", resp.StatusCode)
	}

	// Decode JSON
	var payload deezerResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode deezer response: %w", err)
	}

	// Deezer sometimes returns 200 with an "error" nested payload
	if payload.Error != nil {
		return nil, fmt.Errorf("deezer API error: %s", payload.Error.Message)
	}

	// Map to our DTO
	results := make([]model.TrackDTO, 0, len(payload.Data))
	for _, track := range payload.Data {
		results = append(results, model.TrackDTO{
			ExternalID: strconv.FormatInt(track.ID, 10),
			Title:      track.Title,
			Artist:     track.Artist.Name,
			Album:      track.Album.Title,
			PreviewURL: track.Preview,
			CoverURL:   track.Album.CoverMedium,
		})
	}

	return results, nil
}

// GetTrack resolves a single track by its Deezer id, returning a fresh preview
// URL for playback. Deezer preview URLs are short-lived, so the client resolves
// them just before playing rather than storing them.
func (s *deezerMusicService) GetTrack(ctx context.Context, id string) (*model.TrackDTO, error) {
	reqURL := fmt.Sprintf("https://api.deezer.com/track/%s", url.PathEscape(id))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("deezer API returned status %d", resp.StatusCode)
	}

	var payload deezerTrackResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode deezer response: %w", err)
	}

	// Deezer replies 200 with a nested "error" payload for an unknown id.
	if payload.Error != nil {
		return nil, fmt.Errorf("deezer API error: %s", payload.Error.Message)
	}

	return &model.TrackDTO{
		ExternalID: strconv.FormatInt(payload.ID, 10),
		Title:      payload.Title,
		Artist:     payload.Artist.Name,
		Album:      payload.Album.Title,
		PreviewURL: payload.Preview,
		CoverURL:   payload.Album.CoverMedium,
	}, nil
}
