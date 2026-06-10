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
	ErrDeviceNotFound      = errors.New("device not found")
	ErrDeviceAlreadyExists = errors.New("device already registered")
)

type DeviceService interface {
	Register(ctx context.Context, userID uuid.UUID, req model.CreateDeviceRequest) (*model.Device, error)
	List(ctx context.Context, userID uuid.UUID) ([]model.DeviceWithDelegate, error)
	Get(ctx context.Context, deviceID, userID uuid.UUID) (*model.Device, error)
	Delete(ctx context.Context, deviceID, userID uuid.UUID) error
}

type deviceService struct {
	repo repository.DeviceRepository
}

func NewDeviceService(repo repository.DeviceRepository) DeviceService {
	return &deviceService{repo: repo}
}

func (s *deviceService) Register(ctx context.Context, userID uuid.UUID, req model.CreateDeviceRequest) (*model.Device, error) {
	device, err := s.repo.Create(ctx, userID, req)
	if err != nil {
		if isPgUniqueViolation(err) {
			return nil, ErrDeviceAlreadyExists
		}
		return nil, err
	}
	return device, nil
}

func (s *deviceService) List(ctx context.Context, userID uuid.UUID) ([]model.DeviceWithDelegate, error) {
	return s.repo.List(ctx, userID)
}

func (s *deviceService) Get(ctx context.Context, deviceID, userID uuid.UUID) (*model.Device, error) {
	device, err := s.repo.GetByID(ctx, deviceID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}
	return device, nil
}

func (s *deviceService) Delete(ctx context.Context, deviceID, userID uuid.UUID) error {
	deleted, err := s.repo.Delete(ctx, deviceID, userID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrDeviceNotFound
	}
	return nil
}
