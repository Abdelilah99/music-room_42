package repository

import (
	"context"
	"fmt"

	"music-room/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeviceRepository interface {
	Create(ctx context.Context, userID uuid.UUID, req model.CreateDeviceRequest) (*model.Device, error)
	List(ctx context.Context, userID uuid.UUID) ([]model.DeviceWithDelegate, error)
	GetByID(ctx context.Context, deviceID, userID uuid.UUID) (*model.Device, error)
	Delete(ctx context.Context, deviceID, userID uuid.UUID) (bool, error)
}

type deviceRepository struct {
	pool *pgxpool.Pool
}

func NewDeviceRepository(pool *pgxpool.Pool) DeviceRepository {
	return &deviceRepository{pool: pool}
}

const deviceCols = `id, user_id, name, platform, model, created_at`

func scanDevice(row interface{ Scan(...any) error }) (*model.Device, error) {
	var d model.Device
	if err := row.Scan(&d.ID, &d.UserID, &d.Name, &d.Platform, &d.Model, &d.CreatedAt); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *deviceRepository) Create(ctx context.Context, userID uuid.UUID, req model.CreateDeviceRequest) (*model.Device, error) {
	query := fmt.Sprintf(`
		INSERT INTO devices (user_id, name, platform, model)
		VALUES ($1, $2, $3, $4)
		RETURNING %s`, deviceCols)
	return scanDevice(r.pool.QueryRow(ctx, query, userID, req.Name, req.Platform, req.Model))
}

func (r *deviceRepository) List(ctx context.Context, userID uuid.UUID) ([]model.DeviceWithDelegate, error) {
	query := `
		SELECT d.id, d.user_id, d.name, d.platform, d.model, d.created_at,
		       del.delegate_id, u.email
		FROM devices d
		LEFT JOIN delegations del ON del.device_id = d.id AND del.revoked_at IS NULL
		LEFT JOIN users u ON u.id = del.delegate_id
		WHERE d.user_id = $1
		ORDER BY d.created_at ASC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.DeviceWithDelegate
	for rows.Next() {
		var dwd model.DeviceWithDelegate
		var delegateID *uuid.UUID
		var delegateEmail *string
		if err := rows.Scan(
			&dwd.ID, &dwd.UserID, &dwd.Name, &dwd.Platform, &dwd.Model, &dwd.CreatedAt,
			&delegateID, &delegateEmail,
		); err != nil {
			return nil, err
		}
		if delegateID != nil && delegateEmail != nil {
			dwd.Delegate = &model.ActiveDelegate{UserID: *delegateID, Email: *delegateEmail}
		}
		result = append(result, dwd)
	}
	return result, rows.Err()
}

func (r *deviceRepository) GetByID(ctx context.Context, deviceID, userID uuid.UUID) (*model.Device, error) {
	query := fmt.Sprintf(`SELECT %s FROM devices WHERE id = $1 AND user_id = $2`, deviceCols)
	return scanDevice(r.pool.QueryRow(ctx, query, deviceID, userID))
}

func (r *deviceRepository) Delete(ctx context.Context, deviceID, userID uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM devices WHERE id = $1 AND user_id = $2`, deviceID, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
