package service

import (
	"context"
	"errors"
	"time"

	"music-room/internal/model"
	"music-room/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrEventNotFound        = errors.New("event not found")
	ErrNotEventOwner        = errors.New("only the event owner can perform this action")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidLicenseConfig = errors.New("license 2 requires lat, lng, radius, vote_start and vote_end")
)

type EventService interface {
	Create(ctx context.Context, ownerID uuid.UUID, req model.CreateEventRequest) (*model.Event, error)
	List(ctx context.Context, callerID uuid.UUID, f model.EventListFilter) ([]model.Event, error)
	Get(ctx context.Context, eventID, callerID uuid.UUID) (*model.Event, error)
	Update(ctx context.Context, eventID, callerID uuid.UUID, req model.UpdateEventRequest) (*model.Event, error)
	Delete(ctx context.Context, eventID, callerID uuid.UUID) error
	Invite(ctx context.Context, eventID, callerID, targetUserID uuid.UUID) error
}

type eventService struct {
	repo repository.EventRepository
}

func NewEventService(repo repository.EventRepository) EventService {
	return &eventService{repo: repo}
}

func (s *eventService) Create(ctx context.Context, ownerID uuid.UUID, req model.CreateEventRequest) (*model.Event, error) {
	if err := validateLicense(req.License, req.Lat, req.Lng, req.Radius, req.VoteStart, req.VoteEnd); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, ownerID, req)
}

func (s *eventService) List(ctx context.Context, callerID uuid.UUID, f model.EventListFilter) ([]model.Event, error) {
	return s.repo.List(ctx, callerID, f)
}

func (s *eventService) Get(ctx context.Context, eventID, callerID uuid.UUID) (*model.Event, error) {
	e, err := s.repo.GetAccessible(ctx, eventID, callerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}
	return e, nil
}

func (s *eventService) Update(ctx context.Context, eventID, callerID uuid.UUID, req model.UpdateEventRequest) (*model.Event, error) {
	_, err := s.repo.GetByIDForOwner(ctx, eventID, callerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}
	if req.License != nil {
		if err := validateLicense(*req.License, req.Lat, req.Lng, req.Radius, req.VoteStart, req.VoteEnd); err != nil {
			return nil, err
		}
	}
	return s.repo.Update(ctx, eventID, req)
}

func (s *eventService) Delete(ctx context.Context, eventID, callerID uuid.UUID) error {
	_, err := s.repo.GetByIDForOwner(ctx, eventID, callerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEventNotFound
		}
		return err
	}
	return s.repo.Delete(ctx, eventID)
}

func (s *eventService) Invite(ctx context.Context, eventID, callerID, targetUserID uuid.UUID) error {
	_, err := s.repo.GetByIDForOwner(ctx, eventID, callerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEventNotFound
		}
		return err
	}
	if err := s.repo.AddInvite(ctx, eventID, targetUserID); err != nil {
		if isPgFKViolation(err) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}

func validateLicense(license int, lat, lng, radius *float64, voteStart, voteEnd *time.Time) error {
	if license == 2 {
		if lat == nil || lng == nil || radius == nil || voteStart == nil || voteEnd == nil {
			return ErrInvalidLicenseConfig
		}
	}
	return nil
}

func isPgFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
