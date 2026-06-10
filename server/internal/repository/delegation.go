package repository

import (
	"context"

	"music-room/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DelegationRepository interface {
	Grant(ctx context.Context, deviceID, ownerID, delegateID uuid.UUID) error
	Revoke(ctx context.Context, deviceID, ownerID uuid.UUID) error
	IsAuthorized(ctx context.Context, deviceID, callerID uuid.UUID) (bool, error)
	ListDelegated(ctx context.Context, delegateID uuid.UUID) ([]model.DelegatedDevice, error)
	IsFriend(ctx context.Context, userA, userB uuid.UUID) (bool, error)
}

type delegationRepository struct {
	pool *pgxpool.Pool
}

func NewDelegationRepository(pool *pgxpool.Pool) DelegationRepository {
	return &delegationRepository{pool: pool}
}

// Grant revokes any existing active delegation for the device then inserts a new one.
func (r *delegationRepository) Grant(ctx context.Context, deviceID, ownerID, delegateID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE delegations SET revoked_at = NOW() WHERE device_id = $1 AND revoked_at IS NULL`,
		deviceID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO delegations (device_id, owner_id, delegate_id) VALUES ($1, $2, $3)`,
		deviceID, ownerID, delegateID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Revoke sets revoked_at on the active delegation for the device owned by ownerID.
func (r *delegationRepository) Revoke(ctx context.Context, deviceID, ownerID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE delegations SET revoked_at = NOW() WHERE device_id = $1 AND owner_id = $2 AND revoked_at IS NULL`,
		deviceID, ownerID,
	)
	return err
}

// IsAuthorized returns true if callerID is the device owner or the active delegate.
func (r *delegationRepository) IsAuthorized(ctx context.Context, deviceID, callerID uuid.UUID) (bool, error) {
	var authorized bool
	err := r.pool.QueryRow(ctx, `
		SELECT
			EXISTS(SELECT 1 FROM devices WHERE id = $1 AND user_id = $2) OR
			EXISTS(SELECT 1 FROM delegations WHERE device_id = $1 AND delegate_id = $2 AND revoked_at IS NULL)`,
		deviceID, callerID,
	).Scan(&authorized)
	return authorized, err
}

// ListDelegated returns all devices for which delegateID currently holds an active delegation.
func (r *delegationRepository) ListDelegated(ctx context.Context, delegateID uuid.UUID) ([]model.DelegatedDevice, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT d.id, d.user_id, d.name, d.platform, d.model, d.created_at,
		       u.id AS owner_id, u.email AS owner_email
		FROM devices d
		JOIN delegations del ON del.device_id = d.id AND del.revoked_at IS NULL AND del.delegate_id = $1
		JOIN users u ON u.id = d.user_id
		ORDER BY d.created_at ASC`,
		delegateID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.DelegatedDevice
	for rows.Next() {
		var dd model.DelegatedDevice
		if err := rows.Scan(
			&dd.ID, &dd.UserID, &dd.Name, &dd.Platform, &dd.Model, &dd.CreatedAt,
			&dd.Owner.UserID, &dd.Owner.Email,
		); err != nil {
			return nil, err
		}
		result = append(result, dd)
	}
	return result, rows.Err()
}

// IsFriend returns true if an accepted friendship exists between userA and userB.
func (r *delegationRepository) IsFriend(ctx context.Context, userA, userB uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM friendships
			WHERE ((requester_id = $1 AND addressee_id = $2) OR (requester_id = $2 AND addressee_id = $1))
			AND status = 'accepted'
		)`,
		userA, userB,
	).Scan(&exists)
	return exists, err
}
