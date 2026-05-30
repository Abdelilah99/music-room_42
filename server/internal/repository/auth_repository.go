package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"music-room/internal/model"
)

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

func (r *AuthRepository) CreateUser(ctx context.Context, email, passwordHash string) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)
		 RETURNING id, email, password_hash, is_verified, subscription_tier, created_at`,
		email, passwordHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.IsVerified, &user.SubscriptionTier, &user.CreatedAt)
	return user, err
}

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, is_verified, subscription_tier, created_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.IsVerified, &user.SubscriptionTier, &user.CreatedAt)
	return user, err
}

func (r *AuthRepository) CreateUserWithVerification(ctx context.Context, email, passwordHash string, token uuid.UUID) (*model.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	user := &model.User{}
	err = tx.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)
		 RETURNING id, email, password_hash, is_verified, subscription_tier, created_at`,
		email, passwordHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.IsVerified, &user.SubscriptionTier, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO email_verifications (user_id, token) VALUES ($1, $2)`,
		user.ID, token,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) CreateEmailVerification(ctx context.Context, userID, token uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_verifications (user_id, token) VALUES ($1, $2)`,
		userID, token,
	)
	return err
}

func (r *AuthRepository) DeleteEmailVerificationsForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM email_verifications WHERE user_id = $1`,
		userID,
	)
	return err
}

func (r *AuthRepository) GetAndDeleteEmailVerification(ctx context.Context, token uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.pool.QueryRow(ctx,
		`DELETE FROM email_verifications WHERE token = $1 RETURNING user_id`,
		token,
	).Scan(&userID)
	return userID, err
}

func (r *AuthRepository) VerifyUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET is_verified = true WHERE id = $1`,
		userID,
	)
	return err
}

func (r *AuthRepository) CreatePasswordResetToken(ctx context.Context, userID, token uuid.UUID, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		userID, token, expiresAt,
	)
	return err
}

func (r *AuthRepository) GetPasswordResetToken(ctx context.Context, token uuid.UUID) (*model.PasswordResetToken, error) {
	t := &model.PasswordResetToken{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token, expires_at, used_at
		 FROM password_reset_tokens WHERE token = $1`,
		token,
	).Scan(&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.UsedAt)
	return t, err
}

func (r *AuthRepository) ResetPassword(ctx context.Context, tokenID, userID uuid.UUID, newPasswordHash string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`,
		newPasswordHash, userID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`,
		tokenID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
