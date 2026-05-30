package repository

import (
	"context"
	"errors"
	"music-room/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id string) (*model.User, error)
	Create(ctx context.Context, email, passwordHash string) (*model.User, error)
}

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT id, email, password_hash, is_verified, subscription_tier, created_at FROM users WHERE email = $1`
	var u model.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.IsVerified,
		&u.SubscriptionTier,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	query := `SELECT id, email, password_hash, is_verified, subscription_tier, created_at FROM users WHERE id = $1`
	var u model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.IsVerified,
		&u.SubscriptionTier,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *PostgresUserRepository) Create(ctx context.Context, email, passwordHash string) (*model.User, error) {
	query := `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, email, password_hash, is_verified, subscription_tier, created_at`
	var u model.User
	err := r.pool.QueryRow(ctx, query, email, passwordHash).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.IsVerified,
		&u.SubscriptionTier,
		&u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
