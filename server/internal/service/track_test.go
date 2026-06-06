package service_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"music-room/internal/model"
	"music-room/internal/repository"
	"music-room/internal/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// --- mock event repository (only methods used by trackService) ---

type mockEventRepoForTrack struct {
	getAccessibleFn  func(ctx context.Context, eventID, callerID uuid.UUID) (*model.Event, error)
	getByIDOwnerFn   func(ctx context.Context, eventID, ownerID uuid.UUID) (*model.Event, error)
}

func (m *mockEventRepoForTrack) Create(ctx context.Context, ownerID uuid.UUID, req model.CreateEventRequest) (*model.Event, error) {
	panic("not expected")
}
func (m *mockEventRepoForTrack) List(ctx context.Context, callerID uuid.UUID, f model.EventListFilter) ([]model.Event, error) {
	panic("not expected")
}
func (m *mockEventRepoForTrack) GetAccessible(ctx context.Context, eventID, callerID uuid.UUID) (*model.Event, error) {
	return m.getAccessibleFn(ctx, eventID, callerID)
}
func (m *mockEventRepoForTrack) GetByIDForOwner(ctx context.Context, eventID, ownerID uuid.UUID) (*model.Event, error) {
	return m.getByIDOwnerFn(ctx, eventID, ownerID)
}
func (m *mockEventRepoForTrack) Update(ctx context.Context, eventID uuid.UUID, req model.UpdateEventRequest) (*model.Event, error) {
	panic("not expected")
}
func (m *mockEventRepoForTrack) Delete(ctx context.Context, eventID uuid.UUID) error {
	panic("not expected")
}
func (m *mockEventRepoForTrack) AddInvite(ctx context.Context, eventID, userID uuid.UUID) error {
	panic("not expected")
}

// --- mock track repository ---

type mockTrackRepo struct {
	addFn      func(ctx context.Context, eventID, suggestedBy uuid.UUID, req model.SuggestTrackRequest) (*model.Track, error)
	getQueueFn func(ctx context.Context, eventID uuid.UUID) ([]model.Track, error)
	getByIDFn  func(ctx context.Context, trackID, eventID uuid.UUID) (*model.Track, error)
	voteFn     func(ctx context.Context, trackID, userID uuid.UUID) error
	deleteFn   func(ctx context.Context, trackID uuid.UUID) error
}

func (m *mockTrackRepo) Add(ctx context.Context, eventID, suggestedBy uuid.UUID, req model.SuggestTrackRequest) (*model.Track, error) {
	return m.addFn(ctx, eventID, suggestedBy, req)
}
func (m *mockTrackRepo) GetQueue(ctx context.Context, eventID uuid.UUID) ([]model.Track, error) {
	return m.getQueueFn(ctx, eventID)
}
func (m *mockTrackRepo) GetByID(ctx context.Context, trackID, eventID uuid.UUID) (*model.Track, error) {
	return m.getByIDFn(ctx, trackID, eventID)
}
func (m *mockTrackRepo) Vote(ctx context.Context, trackID, userID uuid.UUID) error {
	return m.voteFn(ctx, trackID, userID)
}
func (m *mockTrackRepo) Delete(ctx context.Context, trackID uuid.UUID) error {
	return m.deleteFn(ctx, trackID)
}

// --- helpers ---

func accessibleEvent(eventID uuid.UUID) *model.Event {
	return &model.Event{ID: eventID, Visibility: "public", CreatedAt: time.Now()}
}

func sampleTrack(eventID, trackID uuid.UUID) *model.Track {
	return &model.Track{ID: trackID, EventID: eventID, ExternalID: "dz:123", Title: "Song", Artist: "Artist", VoteCount: 0, CreatedAt: time.Now()}
}

func uniqueViolation() error {
	return &pgconn.PgError{Code: "23505"}
}

// --- tests ---

func TestTrackService_Suggest_Success(t *testing.T) {
	eventID := uuid.New()
	callerID := uuid.New()
	want := sampleTrack(eventID, uuid.New())

	eventRepo := &mockEventRepoForTrack{
		getAccessibleFn: func(_ context.Context, eID, cID uuid.UUID) (*model.Event, error) {
			return accessibleEvent(eID), nil
		},
	}
	trackRepo := &mockTrackRepo{
		addFn: func(_ context.Context, eID, sID uuid.UUID, req model.SuggestTrackRequest) (*model.Track, error) {
			if eID != eventID || sID != callerID {
				t.Errorf("unexpected Add params")
			}
			return want, nil
		},
	}

	svc := service.NewTrackService(eventRepo, trackRepo)
	got, err := svc.Suggest(context.Background(), eventID, callerID, model.SuggestTrackRequest{ExternalID: "dz:123", Title: "Song", Artist: "Artist"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected track ID %s, got %s", want.ID, got.ID)
	}
}

func TestTrackService_Suggest_EventNotFound(t *testing.T) {
	eventRepo := &mockEventRepoForTrack{
		getAccessibleFn: func(_ context.Context, _, _ uuid.UUID) (*model.Event, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewTrackService(eventRepo, &mockTrackRepo{})
	_, err := svc.Suggest(context.Background(), uuid.New(), uuid.New(), model.SuggestTrackRequest{})
	if !errors.Is(err, service.ErrEventNotFound) {
		t.Errorf("expected ErrEventNotFound, got %v", err)
	}
}

func TestTrackService_Suggest_Duplicate_Returns409(t *testing.T) {
	eventRepo := &mockEventRepoForTrack{
		getAccessibleFn: func(_ context.Context, eID, _ uuid.UUID) (*model.Event, error) {
			return accessibleEvent(eID), nil
		},
	}
	trackRepo := &mockTrackRepo{
		addFn: func(_ context.Context, _, _ uuid.UUID, _ model.SuggestTrackRequest) (*model.Track, error) {
			return nil, uniqueViolation()
		},
	}

	svc := service.NewTrackService(eventRepo, trackRepo)
	_, err := svc.Suggest(context.Background(), uuid.New(), uuid.New(), model.SuggestTrackRequest{ExternalID: "dz:123", Title: "Song", Artist: "Artist"})
	if !errors.Is(err, service.ErrTrackAlreadyExists) {
		t.Errorf("expected ErrTrackAlreadyExists, got %v", err)
	}
}

func TestTrackService_GetQueue_Success(t *testing.T) {
	eventID := uuid.New()
	want := []model.Track{*sampleTrack(eventID, uuid.New()), *sampleTrack(eventID, uuid.New())}

	eventRepo := &mockEventRepoForTrack{
		getAccessibleFn: func(_ context.Context, eID, _ uuid.UUID) (*model.Event, error) {
			return accessibleEvent(eID), nil
		},
	}
	trackRepo := &mockTrackRepo{
		getQueueFn: func(_ context.Context, eID uuid.UUID) ([]model.Track, error) {
			if eID != eventID {
				t.Errorf("unexpected eventID: %s", eID)
			}
			return want, nil
		},
	}

	svc := service.NewTrackService(eventRepo, trackRepo)
	got, err := svc.GetQueue(context.Background(), eventID, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 tracks, got %d", len(got))
	}
}

func TestTrackService_Vote_Success(t *testing.T) {
	eventID := uuid.New()
	trackID := uuid.New()

	eventRepo := &mockEventRepoForTrack{
		getAccessibleFn: func(_ context.Context, eID, _ uuid.UUID) (*model.Event, error) {
			return accessibleEvent(eID), nil
		},
	}
	trackRepo := &mockTrackRepo{
		getByIDFn: func(_ context.Context, tID, eID uuid.UUID) (*model.Track, error) {
			return sampleTrack(eID, tID), nil
		},
		voteFn: func(_ context.Context, _, _ uuid.UUID) error {
			return nil
		},
	}

	svc := service.NewTrackService(eventRepo, trackRepo)
	if err := svc.Vote(context.Background(), eventID, trackID, uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTrackService_Vote_AlreadyVoted_Returns409(t *testing.T) {
	eventRepo := &mockEventRepoForTrack{
		getAccessibleFn: func(_ context.Context, eID, _ uuid.UUID) (*model.Event, error) {
			return accessibleEvent(eID), nil
		},
	}
	trackRepo := &mockTrackRepo{
		getByIDFn: func(_ context.Context, tID, eID uuid.UUID) (*model.Track, error) {
			return sampleTrack(eID, tID), nil
		},
		voteFn: func(_ context.Context, _, _ uuid.UUID) error {
			return uniqueViolation()
		},
	}

	svc := service.NewTrackService(eventRepo, trackRepo)
	err := svc.Vote(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrAlreadyVoted) {
		t.Errorf("expected ErrAlreadyVoted, got %v", err)
	}
}

func TestTrackService_Vote_TrackNotFound(t *testing.T) {
	eventRepo := &mockEventRepoForTrack{
		getAccessibleFn: func(_ context.Context, eID, _ uuid.UUID) (*model.Event, error) {
			return accessibleEvent(eID), nil
		},
	}
	trackRepo := &mockTrackRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Track, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewTrackService(eventRepo, trackRepo)
	err := svc.Vote(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrTrackNotFound) {
		t.Errorf("expected ErrTrackNotFound, got %v", err)
	}
}

// TestTrackService_Vote_Concurrent verifies that two simultaneous votes
// by different users both succeed and vote is counted exactly twice.
// The actual DB-level atomicity relies on vote_count = vote_count + 1 — never a read-then-write.
func TestTrackService_Vote_Concurrent(t *testing.T) {
	eventID := uuid.New()
	trackID := uuid.New()
	var voteCount int32

	eventRepo := &mockEventRepoForTrack{
		getAccessibleFn: func(_ context.Context, eID, _ uuid.UUID) (*model.Event, error) {
			return accessibleEvent(eID), nil
		},
	}
	trackRepo := &mockTrackRepo{
		getByIDFn: func(_ context.Context, tID, eID uuid.UUID) (*model.Track, error) {
			return sampleTrack(eID, tID), nil
		},
		voteFn: func(_ context.Context, _, _ uuid.UUID) error {
			atomic.AddInt32(&voteCount, 1)
			return nil
		},
	}

	svc := service.NewTrackService(eventRepo, trackRepo)

	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			errs <- svc.Vote(context.Background(), eventID, trackID, uuid.New())
		}()
	}

	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Errorf("goroutine vote failed: %v", err)
		}
	}

	if got := atomic.LoadInt32(&voteCount); got != 2 {
		t.Errorf("expected voteCount=2, got %d", got)
	}
}

func TestTrackService_DeleteTrack_Owner(t *testing.T) {
	eventID := uuid.New()
	trackID := uuid.New()
	ownerID := uuid.New()
	deleted := false

	eventRepo := &mockEventRepoForTrack{
		getByIDOwnerFn: func(_ context.Context, eID, oID uuid.UUID) (*model.Event, error) {
			if eID != eventID || oID != ownerID {
				return nil, pgx.ErrNoRows
			}
			return accessibleEvent(eID), nil
		},
	}
	trackRepo := &mockTrackRepo{
		getByIDFn: func(_ context.Context, tID, eID uuid.UUID) (*model.Track, error) {
			return sampleTrack(eID, tID), nil
		},
		deleteFn: func(_ context.Context, tID uuid.UUID) error {
			if tID != trackID {
				t.Errorf("unexpected trackID: %s", tID)
			}
			deleted = true
			return nil
		},
	}

	svc := service.NewTrackService(eventRepo, trackRepo)
	if err := svc.DeleteTrack(context.Background(), eventID, trackID, ownerID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected Delete to be called on repository")
	}
}

func TestTrackService_DeleteTrack_NotOwner_Returns404(t *testing.T) {
	eventRepo := &mockEventRepoForTrack{
		getByIDOwnerFn: func(_ context.Context, _, _ uuid.UUID) (*model.Event, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewTrackService(eventRepo, &mockTrackRepo{})
	err := svc.DeleteTrack(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrEventNotFound) {
		t.Errorf("expected ErrEventNotFound, got %v", err)
	}
}

// Ensure mockEventRepoForTrack implements repository.EventRepository at compile time.
var _ repository.EventRepository = (*mockEventRepoForTrack)(nil)

// Ensure mockTrackRepo implements repository.TrackRepository at compile time.
var _ repository.TrackRepository = (*mockTrackRepo)(nil)
