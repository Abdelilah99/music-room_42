package service

import (
	"context"
	"errors"

	"music-room/internal/model"
	"music-room/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrPlaylistNotFound = errors.New("playlist not found")

type PlaylistService interface {
	Create(ctx context.Context, ownerID uuid.UUID, req model.CreatePlaylistRequest) (*model.Playlist, error)
	List(ctx context.Context, callerID uuid.UUID, f model.PlaylistListFilter) ([]model.Playlist, error)
	Get(ctx context.Context, playlistID, callerID uuid.UUID) (*model.PlaylistWithTracks, error)
	Update(ctx context.Context, playlistID, callerID uuid.UUID, req model.UpdatePlaylistRequest) (*model.Playlist, error)
	Delete(ctx context.Context, playlistID, callerID uuid.UUID) error
	Invite(ctx context.Context, playlistID, callerID, targetUserID uuid.UUID) error
}

type playlistService struct {
	repo repository.PlaylistRepository
}

func NewPlaylistService(repo repository.PlaylistRepository) PlaylistService {
	return &playlistService{repo: repo}
}

func (s *playlistService) Create(ctx context.Context, ownerID uuid.UUID, req model.CreatePlaylistRequest) (*model.Playlist, error) {
	return s.repo.Create(ctx, ownerID, req)
}

func (s *playlistService) List(ctx context.Context, callerID uuid.UUID, f model.PlaylistListFilter) ([]model.Playlist, error) {
	return s.repo.List(ctx, callerID, f)
}

func (s *playlistService) Get(ctx context.Context, playlistID, callerID uuid.UUID) (*model.PlaylistWithTracks, error) {
	p, err := s.repo.GetAccessible(ctx, playlistID, callerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPlaylistNotFound
		}
		return nil, err
	}

	tracks, err := s.repo.ListTracks(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if tracks == nil {
		tracks = []model.PlaylistTrack{}
	}

	return &model.PlaylistWithTracks{Playlist: *p, Tracks: tracks}, nil
}

// Update, Delete and Invite go through GetByIDForOwner first: a non-owner (or a
// missing playlist) yields pgx.ErrNoRows, which maps to ErrPlaylistNotFound so
// the endpoint returns 404 without leaking whether the playlist exists.
func (s *playlistService) Update(ctx context.Context, playlistID, callerID uuid.UUID, req model.UpdatePlaylistRequest) (*model.Playlist, error) {
	if _, err := s.requireOwner(ctx, playlistID, callerID); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, playlistID, req)
}

func (s *playlistService) Delete(ctx context.Context, playlistID, callerID uuid.UUID) error {
	if _, err := s.requireOwner(ctx, playlistID, callerID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, playlistID)
}

func (s *playlistService) Invite(ctx context.Context, playlistID, callerID, targetUserID uuid.UUID) error {
	if _, err := s.requireOwner(ctx, playlistID, callerID); err != nil {
		return err
	}
	if err := s.repo.AddInvite(ctx, playlistID, targetUserID); err != nil {
		if isPgFKViolation(err) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *playlistService) requireOwner(ctx context.Context, playlistID, callerID uuid.UUID) (*model.Playlist, error) {
	p, err := s.repo.GetByIDForOwner(ctx, playlistID, callerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPlaylistNotFound
		}
		return nil, err
	}
	return p, nil
}
