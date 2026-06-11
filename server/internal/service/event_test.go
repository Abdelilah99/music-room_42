package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"music-room/internal/model"
	"music-room/internal/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// --- mock repository ---

type mockEventRepo struct {
	createFn        func(ctx context.Context, ownerID uuid.UUID, req model.CreateEventRequest) (*model.Event, error)
	listFn          func(ctx context.Context, callerID uuid.UUID, f model.EventListFilter) ([]model.Event, error)
	getAccessibleFn func(ctx context.Context, eventID, callerID uuid.UUID) (*model.Event, error)
	getByIDOwnerFn  func(ctx context.Context, eventID, ownerID uuid.UUID) (*model.Event, error)
	updateFn        func(ctx context.Context, eventID uuid.UUID, req model.UpdateEventRequest) (*model.Event, error)
	deleteFn        func(ctx context.Context, eventID uuid.UUID) error
	addInviteFn     func(ctx context.Context, eventID, userID uuid.UUID) error
}

func (m *mockEventRepo) Create(ctx context.Context, ownerID uuid.UUID, req model.CreateEventRequest) (*model.Event, error) {
	return m.createFn(ctx, ownerID, req)
}
func (m *mockEventRepo) List(ctx context.Context, callerID uuid.UUID, f model.EventListFilter) ([]model.Event, error) {
	return m.listFn(ctx, callerID, f)
}
func (m *mockEventRepo) GetAccessible(ctx context.Context, eventID, callerID uuid.UUID) (*model.Event, error) {
	return m.getAccessibleFn(ctx, eventID, callerID)
}
func (m *mockEventRepo) GetByIDForOwner(ctx context.Context, eventID, ownerID uuid.UUID) (*model.Event, error) {
	return m.getByIDOwnerFn(ctx, eventID, ownerID)
}
func (m *mockEventRepo) Update(ctx context.Context, eventID uuid.UUID, req model.UpdateEventRequest) (*model.Event, error) {
	return m.updateFn(ctx, eventID, req)
}
func (m *mockEventRepo) Delete(ctx context.Context, eventID uuid.UUID) error {
	return m.deleteFn(ctx, eventID)
}
func (m *mockEventRepo) AddInvite(ctx context.Context, eventID, userID uuid.UUID) error {
	return m.addInviteFn(ctx, eventID, userID)
}
func (m *mockEventRepo) IsInvited(_ context.Context, _, _ uuid.UUID) (bool, error) {
	panic("not expected")
}

// --- helpers ---

func newEvent(ownerID uuid.UUID) *model.Event {
	return &model.Event{
		ID:         uuid.New(),
		OwnerID:    ownerID,
		Name:       "Test Event",
		Visibility: "public",
		License:    0,
		CreatedAt:  time.Now(),
	}
}

// --- tests ---

func TestEventService_Create(t *testing.T) {
	ownerID := uuid.New()
	want := newEvent(ownerID)

	repo := &mockEventRepo{
		createFn: func(_ context.Context, id uuid.UUID, req model.CreateEventRequest) (*model.Event, error) {
			if id != ownerID {
				t.Errorf("expected ownerID %s, got %s", ownerID, id)
			}
			return want, nil
		},
	}

	svc := service.NewEventService(repo)
	got, err := svc.Create(context.Background(), ownerID, model.CreateEventRequest{Name: "Test Event", Visibility: "public"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected event ID %s, got %s", want.ID, got.ID)
	}
}

func TestEventService_Get_PublicEvent(t *testing.T) {
	ownerID := uuid.New()
	callerID := uuid.New()
	want := newEvent(ownerID)

	repo := &mockEventRepo{
		getAccessibleFn: func(_ context.Context, eventID, cID uuid.UUID) (*model.Event, error) {
			if cID != callerID {
				t.Errorf("expected callerID %s, got %s", callerID, cID)
			}
			return want, nil
		},
	}

	svc := service.NewEventService(repo)
	got, err := svc.Get(context.Background(), want.ID, callerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected event ID %s, got %s", want.ID, got.ID)
	}
}

func TestEventService_Get_PrivateNotInvited_Returns404(t *testing.T) {
	repo := &mockEventRepo{
		getAccessibleFn: func(_ context.Context, _, _ uuid.UUID) (*model.Event, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewEventService(repo)
	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrEventNotFound) {
		t.Errorf("expected ErrEventNotFound, got %v", err)
	}
}

func TestEventService_Update_NotOwner_Returns404(t *testing.T) {
	repo := &mockEventRepo{
		getByIDOwnerFn: func(_ context.Context, _, _ uuid.UUID) (*model.Event, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewEventService(repo)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateEventRequest{})
	if !errors.Is(err, service.ErrEventNotFound) {
		t.Errorf("expected ErrEventNotFound, got %v", err)
	}
}

func TestEventService_Delete_Owner(t *testing.T) {
	ownerID := uuid.New()
	eventID := uuid.New()
	deleted := false

	repo := &mockEventRepo{
		getByIDOwnerFn: func(_ context.Context, eID, oID uuid.UUID) (*model.Event, error) {
			if eID != eventID || oID != ownerID {
				return nil, pgx.ErrNoRows
			}
			return newEvent(ownerID), nil
		},
		deleteFn: func(_ context.Context, id uuid.UUID) error {
			deleted = true
			return nil
		},
	}

	svc := service.NewEventService(repo)
	if err := svc.Delete(context.Background(), eventID, ownerID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected Delete to be called on repository")
	}
}

func TestEventService_Delete_NotOwner_Returns404(t *testing.T) {
	repo := &mockEventRepo{
		getByIDOwnerFn: func(_ context.Context, _, _ uuid.UUID) (*model.Event, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewEventService(repo)
	err := svc.Delete(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrEventNotFound) {
		t.Errorf("expected ErrEventNotFound, got %v", err)
	}
}

func TestEventService_Invite_NotOwner_Returns404(t *testing.T) {
	repo := &mockEventRepo{
		getByIDOwnerFn: func(_ context.Context, _, _ uuid.UUID) (*model.Event, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewEventService(repo)
	err := svc.Invite(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrEventNotFound) {
		t.Errorf("expected ErrEventNotFound, got %v", err)
	}
}

func TestEventService_Invite_Owner(t *testing.T) {
	ownerID := uuid.New()
	eventID := uuid.New()
	targetID := uuid.New()
	invited := false

	repo := &mockEventRepo{
		getByIDOwnerFn: func(_ context.Context, eID, oID uuid.UUID) (*model.Event, error) {
			if eID != eventID || oID != ownerID {
				return nil, pgx.ErrNoRows
			}
			return newEvent(ownerID), nil
		},
		addInviteFn: func(_ context.Context, eID, uID uuid.UUID) error {
			if eID != eventID || uID != targetID {
				t.Errorf("unexpected invite params: event=%s user=%s", eID, uID)
			}
			invited = true
			return nil
		},
	}

	svc := service.NewEventService(repo)
	if err := svc.Invite(context.Background(), eventID, ownerID, targetID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !invited {
		t.Error("expected AddInvite to be called on repository")
	}
}

func TestEventService_Invite_AlreadyInvited_Returns409(t *testing.T) {
	ownerID := uuid.New()
	eventID := uuid.New()
	uniqueErr := &pgconn.PgError{Code: "23505"}

	repo := &mockEventRepo{
		getByIDOwnerFn: func(_ context.Context, _, _ uuid.UUID) (*model.Event, error) {
			return newEvent(ownerID), nil
		},
		addInviteFn: func(_ context.Context, _, _ uuid.UUID) error {
			return uniqueErr
		},
	}

	svc := service.NewEventService(repo)
	err := svc.Invite(context.Background(), eventID, ownerID, uuid.New())
	if !errors.Is(err, service.ErrAlreadyInvited) {
		t.Errorf("expected ErrAlreadyInvited, got %v", err)
	}
}

func TestEventService_Invite_NonExistentUser_Returns404(t *testing.T) {
	ownerID := uuid.New()
	eventID := uuid.New()
	fkErr := &pgconn.PgError{Code: "23503"}

	repo := &mockEventRepo{
		getByIDOwnerFn: func(_ context.Context, _, _ uuid.UUID) (*model.Event, error) {
			return newEvent(ownerID), nil
		},
		addInviteFn: func(_ context.Context, _, _ uuid.UUID) error {
			return fkErr
		},
	}

	svc := service.NewEventService(repo)
	err := svc.Invite(context.Background(), eventID, ownerID, uuid.New())
	if !errors.Is(err, service.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestEventService_Create_License1_NoCoords_Succeeds(t *testing.T) {
	ownerID := uuid.New()
	want := newEvent(ownerID)

	repo := &mockEventRepo{
		createFn: func(_ context.Context, _ uuid.UUID, _ model.CreateEventRequest) (*model.Event, error) {
			return want, nil
		},
	}
	svc := service.NewEventService(repo)

	got, err := svc.Create(context.Background(), ownerID, model.CreateEventRequest{
		Name:       "Test",
		Visibility: "invite_only",
		License:    1,
	})
	if err != nil {
		t.Fatalf("license 1 without coords should succeed, got: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected event ID %s, got %s", want.ID, got.ID)
	}
}

func TestEventService_Create_License2_MissingCoords_Returns400(t *testing.T) {
	repo := &mockEventRepo{}
	svc := service.NewEventService(repo)

	now := time.Now()
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateEventRequest{
		Name:       "Test",
		Visibility: "public",
		License:    2,
		VoteStart:  &now,
		VoteEnd:    &now,
	})
	if !errors.Is(err, service.ErrInvalidLicenseConfig) {
		t.Errorf("expected ErrInvalidLicenseConfig, got %v", err)
	}
}

func TestEventService_Create_License2_MissingTimeWindow_Returns400(t *testing.T) {
	repo := &mockEventRepo{}
	svc := service.NewEventService(repo)

	lat, lng, radius := 48.8566, 2.3522, 1000.0
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateEventRequest{
		Name:       "Test",
		Visibility: "public",
		License:    2,
		Lat:        &lat,
		Lng:        &lng,
		Radius:     &radius,
	})
	if !errors.Is(err, service.ErrInvalidLicenseConfig) {
		t.Errorf("expected ErrInvalidLicenseConfig, got %v", err)
	}
}

func TestEventService_Create_License2_AllFields_Succeeds(t *testing.T) {
	ownerID := uuid.New()
	want := newEvent(ownerID)
	lat, lng, radius := 48.8566, 2.3522, 1000.0
	now := time.Now()

	repo := &mockEventRepo{
		createFn: func(_ context.Context, _ uuid.UUID, _ model.CreateEventRequest) (*model.Event, error) {
			return want, nil
		},
	}

	svc := service.NewEventService(repo)
	got, err := svc.Create(context.Background(), ownerID, model.CreateEventRequest{
		Name:       "Test",
		Visibility: "public",
		License:    2,
		Lat:        &lat,
		Lng:        &lng,
		Radius:     &radius,
		VoteStart:  &now,
		VoteEnd:    &now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected event ID %s, got %s", want.ID, got.ID)
	}
}
