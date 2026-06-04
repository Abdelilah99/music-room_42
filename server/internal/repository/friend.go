package repository

import (
	"context"

	"music-room/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FriendRepository interface {
	GetExisting(ctx context.Context, userA, userB uuid.UUID) (*model.Friendship, error)
	Create(ctx context.Context, requesterID, addresseeID uuid.UUID) (*model.Friendship, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Friendship, error)
	Accept(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListFriends(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error)
	ListIncomingRequests(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error)
	ListOutgoingRequests(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error)
}

type friendRepository struct {
	pool *pgxpool.Pool
}

func NewFriendRepository(pool *pgxpool.Pool) FriendRepository {
	return &friendRepository{pool: pool}
}

func (r *friendRepository) GetExisting(ctx context.Context, userA, userB uuid.UUID) (*model.Friendship, error) {
	var f model.Friendship
	query := `
		SELECT id, requester_id, addressee_id, status, created_at
		FROM friendships
		WHERE (requester_id = $1 AND addressee_id = $2)
		   OR (requester_id = $2 AND addressee_id = $1)`

	err := r.pool.QueryRow(ctx, query, userA, userB).Scan(
		&f.ID, &f.RequesterID, &f.AddresseeID, &f.Status, &f.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *friendRepository) Create(ctx context.Context, requesterID, addresseeID uuid.UUID) (*model.Friendship, error) {
	var f model.Friendship
	query := `
		INSERT INTO friendships (requester_id, addressee_id, status)
		VALUES ($1, $2, 'pending')
		RETURNING id, requester_id, addressee_id, status, created_at`

	err := r.pool.QueryRow(ctx, query, requesterID, addresseeID).Scan(
		&f.ID, &f.RequesterID, &f.AddresseeID, &f.Status, &f.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *friendRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Friendship, error) {
	var f model.Friendship
	query := `
		SELECT id, requester_id, addressee_id, status, created_at
		FROM friendships WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&f.ID, &f.RequesterID, &f.AddresseeID, &f.Status, &f.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *friendRepository) Accept(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE friendships SET status = 'accepted' WHERE id = $1`, id)
	return err
}

func (r *friendRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM friendships WHERE id = $1`, id)
	return err
}

func (r *friendRepository) ListFriends(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error) {
	query := `
		SELECT
			f.id,
			CASE WHEN f.requester_id = $1 THEN f.addressee_id ELSE f.requester_id END AS friend_id,
			u.email,
			u.public_info,
			f.created_at
		FROM friendships f
		JOIN users u ON u.id = CASE WHEN f.requester_id = $1 THEN f.addressee_id ELSE f.requester_id END
		WHERE (f.requester_id = $1 OR f.addressee_id = $1) AND f.status = 'accepted'
		ORDER BY f.created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.FriendEntry
	for rows.Next() {
		var e model.FriendEntry
		if err := rows.Scan(&e.FriendshipID, &e.UserID, &e.Email, &e.PublicInfo, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *friendRepository) ListIncomingRequests(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error) {
	query := `
		SELECT f.id, f.requester_id, u.email, u.public_info, f.created_at
		FROM friendships f
		JOIN users u ON u.id = f.requester_id
		WHERE f.addressee_id = $1 AND f.status = 'pending'
		ORDER BY f.created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.FriendEntry
	for rows.Next() {
		var e model.FriendEntry
		if err := rows.Scan(&e.FriendshipID, &e.UserID, &e.Email, &e.PublicInfo, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *friendRepository) ListOutgoingRequests(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error) {
	query := `
		SELECT f.id, f.addressee_id, u.email, u.public_info, f.created_at
		FROM friendships f
		JOIN users u ON u.id = f.addressee_id
		WHERE f.requester_id = $1 AND f.status = 'pending'
		ORDER BY f.created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.FriendEntry
	for rows.Next() {
		var e model.FriendEntry
		if err := rows.Scan(&e.FriendshipID, &e.UserID, &e.Email, &e.PublicInfo, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
