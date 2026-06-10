package service_test

import (
	"context"
	"errors"
	"testing"

	"music-room/internal/model"
	"music-room/internal/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type mockDeviceRepo struct {
	createFn  func(ctx context.Context, userID uuid.UUID, req model.CreateDeviceRequest) (*model.Device, error)
	listFn    func(ctx context.Context, userID uuid.UUID) ([]model.DeviceWithDelegate, error)
	getByIDFn func(ctx context.Context, deviceID, userID uuid.UUID) (*model.Device, error)
	deleteFn  func(ctx context.Context, deviceID, userID uuid.UUID) (bool, error)
}

func (m *mockDeviceRepo) Create(ctx context.Context, userID uuid.UUID, req model.CreateDeviceRequest) (*model.Device, error) {
	return m.createFn(ctx, userID, req)
}
func (m *mockDeviceRepo) List(ctx context.Context, userID uuid.UUID) ([]model.DeviceWithDelegate, error) {
	return m.listFn(ctx, userID)
}
func (m *mockDeviceRepo) GetByID(ctx context.Context, deviceID, userID uuid.UUID) (*model.Device, error) {
	return m.getByIDFn(ctx, deviceID, userID)
}
func (m *mockDeviceRepo) Delete(ctx context.Context, deviceID, userID uuid.UUID) (bool, error) {
	return m.deleteFn(ctx, deviceID, userID)
}

func newDevice(userID uuid.UUID) *model.Device {
	return &model.Device{
		ID:       uuid.New(),
		UserID:   userID,
		Name:     "My Phone",
		Platform: "android",
		Model:    "Pixel 8",
	}
}

func TestDeviceService_Register_Success(t *testing.T) {
	userID := uuid.New()
	want := newDevice(userID)

	repo := &mockDeviceRepo{
		createFn: func(_ context.Context, id uuid.UUID, req model.CreateDeviceRequest) (*model.Device, error) {
			if id != userID {
				t.Errorf("expected userID %s, got %s", userID, id)
			}
			return want, nil
		},
	}

	svc := service.NewDeviceService(repo)
	got, err := svc.Register(context.Background(), userID, model.CreateDeviceRequest{
		Name:     "My Phone",
		Platform: "android",
		Model:    "Pixel 8",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected device ID %s, got %s", want.ID, got.ID)
	}
}

func TestDeviceService_Register_DuplicateModel_Returns409(t *testing.T) {
	uniqueErr := &pgconn.PgError{Code: "23505"}

	repo := &mockDeviceRepo{
		createFn: func(_ context.Context, _ uuid.UUID, _ model.CreateDeviceRequest) (*model.Device, error) {
			return nil, uniqueErr
		},
	}

	svc := service.NewDeviceService(repo)
	_, err := svc.Register(context.Background(), uuid.New(), model.CreateDeviceRequest{
		Name: "Phone", Platform: "android", Model: "Pixel 8",
	})
	if !errors.Is(err, service.ErrDeviceAlreadyExists) {
		t.Errorf("expected ErrDeviceAlreadyExists, got %v", err)
	}
}

func TestDeviceService_List_ReturnsDevices(t *testing.T) {
	userID := uuid.New()
	delegateID := uuid.New()
	email := "delegate@example.com"

	want := []model.DeviceWithDelegate{
		{Device: *newDevice(userID), Delegate: nil},
		{
			Device:   *newDevice(userID),
			Delegate: &model.ActiveDelegate{UserID: delegateID, Email: email},
		},
	}

	repo := &mockDeviceRepo{
		listFn: func(_ context.Context, id uuid.UUID) ([]model.DeviceWithDelegate, error) {
			if id != userID {
				t.Errorf("expected userID %s, got %s", userID, id)
			}
			return want, nil
		},
	}

	svc := service.NewDeviceService(repo)
	got, err := svc.List(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 devices, got %d", len(got))
	}
	if got[0].Delegate != nil {
		t.Error("expected first device to have no delegate")
	}
	if got[1].Delegate == nil || got[1].Delegate.Email != email {
		t.Errorf("expected second device delegate email %s", email)
	}
}

func TestDeviceService_Get_NotOwner_Returns404(t *testing.T) {
	repo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Device, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewDeviceService(repo)
	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrDeviceNotFound) {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestDeviceService_Get_Owner_ReturnsDevice(t *testing.T) {
	userID := uuid.New()
	want := newDevice(userID)

	repo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Device, error) {
			return want, nil
		},
	}

	svc := service.NewDeviceService(repo)
	got, err := svc.Get(context.Background(), want.ID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected device ID %s, got %s", want.ID, got.ID)
	}
}

func TestDeviceService_Delete_NotFound_Returns404(t *testing.T) {
	repo := &mockDeviceRepo{
		deleteFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := service.NewDeviceService(repo)
	err := svc.Delete(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrDeviceNotFound) {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestDeviceService_Delete_Owner_Succeeds(t *testing.T) {
	deleted := false

	repo := &mockDeviceRepo{
		deleteFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			deleted = true
			return true, nil
		},
	}

	svc := service.NewDeviceService(repo)
	if err := svc.Delete(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected Delete to be called on repository")
	}
}
