package service

import (
	"context"
	"errors"

	"music-room/internal/model"
	"music-room/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFriends    = errors.New("NOT_FRIENDS")
	ErrNotAuthorized = errors.New("NOT_AUTHORIZED")
)

type DelegationService interface {
	Grant(ctx context.Context, deviceID, ownerID, friendID uuid.UUID) error
	Revoke(ctx context.Context, deviceID, ownerID uuid.UUID) error
	IsAuthorized(ctx context.Context, deviceID, callerID uuid.UUID) (bool, error)
	ListDelegated(ctx context.Context, delegateID uuid.UUID) ([]model.DelegatedDevice, error)
}

type delegationService struct {
	deviceRepo repository.DeviceRepository
	delegRepo  repository.DelegationRepository
}

func NewDelegationService(deviceRepo repository.DeviceRepository, delegRepo repository.DelegationRepository) DelegationService {
	return &delegationService{deviceRepo: deviceRepo, delegRepo: delegRepo}
}

func (s *delegationService) Grant(ctx context.Context, deviceID, ownerID, friendID uuid.UUID) error {
	if _, err := s.deviceRepo.GetByID(ctx, deviceID, ownerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrDeviceNotFound
		}
		return err
	}

	isFriend, err := s.delegRepo.IsFriend(ctx, ownerID, friendID)
	if err != nil {
		return err
	}
	if !isFriend {
		return ErrNotFriends
	}

	return s.delegRepo.Grant(ctx, deviceID, ownerID, friendID)
}

func (s *delegationService) Revoke(ctx context.Context, deviceID, ownerID uuid.UUID) error {
	if _, err := s.deviceRepo.GetByID(ctx, deviceID, ownerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrDeviceNotFound
		}
		return err
	}

	return s.delegRepo.Revoke(ctx, deviceID, ownerID)
}

func (s *delegationService) IsAuthorized(ctx context.Context, deviceID, callerID uuid.UUID) (bool, error) {
	return s.delegRepo.IsAuthorized(ctx, deviceID, callerID)
}

func (s *delegationService) ListDelegated(ctx context.Context, delegateID uuid.UUID) ([]model.DelegatedDevice, error) {
	return s.delegRepo.ListDelegated(ctx, delegateID)
}
