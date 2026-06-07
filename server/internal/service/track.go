package service

import (
	"context"
	"errors"

	"music-room/internal/model"
	"music-room/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrTrackNotFound      = errors.New("track not found")
	ErrTrackAlreadyExists = errors.New("track already in this event")
	ErrAlreadyVoted       = errors.New("already voted on this track")
)

type TrackService interface {
	Suggest(ctx context.Context, eventID, callerID uuid.UUID, req model.SuggestTrackRequest) (*model.Track, error)
	GetQueue(ctx context.Context, eventID, callerID uuid.UUID) ([]model.Track, error)
	Vote(ctx context.Context, eventID, trackID, callerID uuid.UUID, gps model.VoteRequest) error
	DeleteTrack(ctx context.Context, eventID, trackID, callerID uuid.UUID) error
}

type trackService struct {
	eventRepo repository.EventRepository
	trackRepo repository.TrackRepository
}

func NewTrackService(eventRepo repository.EventRepository, trackRepo repository.TrackRepository) TrackService {
	return &trackService{eventRepo: eventRepo, trackRepo: trackRepo}
}

func (s *trackService) Suggest(ctx context.Context, eventID, callerID uuid.UUID, req model.SuggestTrackRequest) (*model.Track, error) {
	if _, err := s.eventRepo.GetAccessible(ctx, eventID, callerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	track, err := s.trackRepo.Add(ctx, eventID, callerID, req)
	if err != nil {
		if isPgUniqueViolation(err) {
			return nil, ErrTrackAlreadyExists
		}
		return nil, err
	}
	return track, nil
}

func (s *trackService) GetQueue(ctx context.Context, eventID, callerID uuid.UUID) ([]model.Track, error) {
	if _, err := s.eventRepo.GetAccessible(ctx, eventID, callerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	return s.trackRepo.GetQueue(ctx, eventID)
}

// gps is accepted for future geofence enforcement (license 2, tracked in #27).
// It is threaded through now so callers do not need a breaking change when #27 lands.
func (s *trackService) Vote(ctx context.Context, eventID, trackID, callerID uuid.UUID, gps model.VoteRequest) error {
	if _, err := s.eventRepo.GetAccessible(ctx, eventID, callerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEventNotFound
		}
		return err
	}

	if _, err := s.trackRepo.GetByID(ctx, trackID, eventID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTrackNotFound
		}
		return err
	}

	if err := s.trackRepo.Vote(ctx, trackID, callerID); err != nil {
		if isPgUniqueViolation(err) {
			return ErrAlreadyVoted
		}
		return err
	}
	return nil
}

func (s *trackService) DeleteTrack(ctx context.Context, eventID, trackID, callerID uuid.UUID) error {
	if _, err := s.eventRepo.GetByIDForOwner(ctx, eventID, callerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEventNotFound
		}
		return err
	}

	if _, err := s.trackRepo.GetByID(ctx, trackID, eventID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTrackNotFound
		}
		return err
	}

	return s.trackRepo.Delete(ctx, trackID)
}

func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
