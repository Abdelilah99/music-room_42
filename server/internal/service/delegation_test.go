package service_test

import (
	"context"
	"errors"
	"testing"

	"music-room/internal/model"
	"music-room/internal/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type mockDelegRepo struct {
	grantFn        func(ctx context.Context, deviceID, ownerID, delegateID uuid.UUID) error
	revokeFn       func(ctx context.Context, deviceID, ownerID uuid.UUID) error
	isAuthorizedFn func(ctx context.Context, deviceID, callerID uuid.UUID) (bool, error)
	listDelegatedFn func(ctx context.Context, delegateID uuid.UUID) ([]model.DelegatedDevice, error)
	isFriendFn     func(ctx context.Context, userA, userB uuid.UUID) (bool, error)
}

func (m *mockDelegRepo) Grant(ctx context.Context, deviceID, ownerID, delegateID uuid.UUID) error {
	return m.grantFn(ctx, deviceID, ownerID, delegateID)
}
func (m *mockDelegRepo) Revoke(ctx context.Context, deviceID, ownerID uuid.UUID) error {
	return m.revokeFn(ctx, deviceID, ownerID)
}
func (m *mockDelegRepo) IsAuthorized(ctx context.Context, deviceID, callerID uuid.UUID) (bool, error) {
	return m.isAuthorizedFn(ctx, deviceID, callerID)
}
func (m *mockDelegRepo) ListDelegated(ctx context.Context, delegateID uuid.UUID) ([]model.DelegatedDevice, error) {
	return m.listDelegatedFn(ctx, delegateID)
}
func (m *mockDelegRepo) IsFriend(ctx context.Context, userA, userB uuid.UUID) (bool, error) {
	return m.isFriendFn(ctx, userA, userB)
}

func TestDelegationService_Grant_DeviceNotOwned_Returns404(t *testing.T) {
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Device, error) {
			return nil, pgx.ErrNoRows
		},
	}
	delegRepo := &mockDelegRepo{}

	svc := service.NewDelegationService(deviceRepo, delegRepo)
	err := svc.Grant(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrDeviceNotFound) {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestDelegationService_Grant_NotFriends_Returns403(t *testing.T) {
	ownerID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Device, error) {
			return newDevice(ownerID), nil
		},
	}
	delegRepo := &mockDelegRepo{
		isFriendFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := service.NewDelegationService(deviceRepo, delegRepo)
	err := svc.Grant(context.Background(), uuid.New(), ownerID, uuid.New())
	if !errors.Is(err, service.ErrNotFriends) {
		t.Errorf("expected ErrNotFriends, got %v", err)
	}
}

func TestDelegationService_Grant_Friend_Succeeds(t *testing.T) {
	ownerID := uuid.New()
	friendID := uuid.New()
	granted := false

	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Device, error) {
			return newDevice(ownerID), nil
		},
	}
	delegRepo := &mockDelegRepo{
		isFriendFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		grantFn: func(_ context.Context, _, _, dID uuid.UUID) error {
			if dID != friendID {
				t.Errorf("expected friendID %s, got %s", friendID, dID)
			}
			granted = true
			return nil
		},
	}

	svc := service.NewDelegationService(deviceRepo, delegRepo)
	if err := svc.Grant(context.Background(), uuid.New(), ownerID, friendID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !granted {
		t.Error("expected Grant to be called on repository")
	}
}

func TestDelegationService_Revoke_DeviceNotOwned_Returns404(t *testing.T) {
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Device, error) {
			return nil, pgx.ErrNoRows
		},
	}
	delegRepo := &mockDelegRepo{}

	svc := service.NewDelegationService(deviceRepo, delegRepo)
	err := svc.Revoke(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrDeviceNotFound) {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestDelegationService_Revoke_Owner_Succeeds(t *testing.T) {
	ownerID := uuid.New()
	revoked := false

	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Device, error) {
			return newDevice(ownerID), nil
		},
	}
	delegRepo := &mockDelegRepo{
		revokeFn: func(_ context.Context, _, _ uuid.UUID) error {
			revoked = true
			return nil
		},
	}

	svc := service.NewDelegationService(deviceRepo, delegRepo)
	if err := svc.Revoke(context.Background(), uuid.New(), ownerID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !revoked {
		t.Error("expected Revoke to be called on repository")
	}
}

func TestDelegationService_IsAuthorized_Owner_ReturnsTrue(t *testing.T) {
	deviceRepo := &mockDeviceRepo{}
	delegRepo := &mockDelegRepo{
		isAuthorizedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
	}

	svc := service.NewDelegationService(deviceRepo, delegRepo)
	ok, err := svc.IsAuthorized(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected authorized=true")
	}
}

func TestDelegationService_IsAuthorized_Stranger_ReturnsFalse(t *testing.T) {
	deviceRepo := &mockDeviceRepo{}
	delegRepo := &mockDelegRepo{
		isAuthorizedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := service.NewDelegationService(deviceRepo, delegRepo)
	ok, err := svc.IsAuthorized(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected authorized=false")
	}
}

func TestDelegationService_ListDelegated_ReturnsDevices(t *testing.T) {
	delegateID := uuid.New()
	ownerID := uuid.New()

	want := []model.DelegatedDevice{
		{
			Device: *newDevice(ownerID),
			Owner:  model.ActiveDelegate{UserID: ownerID, Email: "owner@example.com"},
		},
	}

	deviceRepo := &mockDeviceRepo{}
	delegRepo := &mockDelegRepo{
		listDelegatedFn: func(_ context.Context, id uuid.UUID) ([]model.DelegatedDevice, error) {
			if id != delegateID {
				t.Errorf("expected delegateID %s, got %s", delegateID, id)
			}
			return want, nil
		},
	}

	svc := service.NewDelegationService(deviceRepo, delegRepo)
	got, err := svc.ListDelegated(context.Background(), delegateID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 device, got %d", len(got))
	}
	if got[0].Owner.Email != "owner@example.com" {
		t.Errorf("unexpected owner email: %s", got[0].Owner.Email)
	}
}
