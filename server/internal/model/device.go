package model

import (
	"time"

	"github.com/google/uuid"
)

type Device struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Platform  string    `json:"platform"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
}

type ActiveDelegate struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
}

type DeviceWithDelegate struct {
	Device
	Delegate *ActiveDelegate `json:"delegate"`
}

type CreateDeviceRequest struct {
	Name     string `json:"name"     binding:"required"`
	Platform string `json:"platform" binding:"required"`
	Model    string `json:"model"    binding:"required"`
}
