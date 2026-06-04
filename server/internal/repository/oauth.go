package repository

import (
	"context"

	"music-room/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OAuthRepository interface {
	GetProviderByProviderID(ctx context.Context, provider, providerID string) (*model.UserProvider, error)
	CreateProvider(ctx context.Context, userID uuid.UUID, provider, providerID string) error
	CreateOAuthUser(ctx context.Context, email string) (*model.User, error)
}

type oauthRepository struct {
	pool *pgxpool.Pool
}

func NewOAuthRepository(pool *pgxpool.Pool) OAuthRepository {
	return &oauthRepository{pool: pool}
}

func (r *oauthRepository) GetProviderByProviderID(ctx context.Context, provider, providerID string) (*model.UserProvider, error) {
	var p model.UserProvider
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, provider, provider_id, created_at
		 FROM user_providers WHERE provider = $1 AND provider_id = $2`,
		provider, providerID,
	).Scan(&p.ID, &p.UserID, &p.Provider, &p.ProviderID, &p.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *oauthRepository) CreateProvider(ctx context.Context, userID uuid.UUID, provider, providerID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_providers (user_id, provider, provider_id) VALUES ($1, $2, $3)`,
		userID, provider, providerID,
	)
	return err
}

// CreateOAuthUser creates a user without a password. The account is pre-verified
// because the OAuth provider has already confirmed the email.
func (r *oauthRepository) CreateOAuthUser(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, is_verified)
		 VALUES ($1, '', TRUE)
		 RETURNING id, email, password_hash, is_verified, subscription_tier, created_at`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsVerified, &u.SubscriptionTier, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
