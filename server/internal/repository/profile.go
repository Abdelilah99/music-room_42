package repository

import (
	"context"
	"fmt"
	"strings"

	"music-room/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProfileRepository interface {
	GetProfileByID(ctx context.Context, id string) (*model.UserProfile, error)
	UpdateProfile(ctx context.Context, id string, req model.UpdateProfileRequest) error
	GetFriendshipStatus(ctx context.Context, userID, targetID string) (string, error)
}

type profileRepository struct {
	pool *pgxpool.Pool
}

func NewProfileRepository(pool *pgxpool.Pool) ProfileRepository {
	return &profileRepository{pool: pool}
}

func (r *profileRepository) GetProfileByID(ctx context.Context, id string) (*model.UserProfile, error) {
	var p model.UserProfile
	query := `SELECT id, email, public_info, friends_info, private_info, music_preferences 
	          FROM users WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Email, &p.PublicInfo, &p.FriendsInfo, &p.PrivateInfo, &p.MusicPreferences,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *profileRepository) GetFriendshipStatus(ctx context.Context, userID, targetID string) (string, error) {
	var status string
	query := `SELECT status FROM friendships 
	          WHERE (requester_id = $1 AND addressee_id = $2) 
	             OR (requester_id = $2 AND addressee_id = $1)`
	
	err := r.pool.QueryRow(ctx, query, userID, targetID).Scan(&status)
	if err == pgx.ErrNoRows {
		return "", nil // No relationship record exists
	}
	return status, err
}

func (r *profileRepository) UpdateProfile(ctx context.Context, id string, req model.UpdateProfileRequest) error {
	// Execute mutations inside a single database transaction to guarantee atomicity
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var setClauses []string
	var args []interface{}
	argPos := 1

	if req.PublicInfo != nil {
		setClauses = append(setClauses, fmt.Sprintf("public_info = public_info || $%d", argPos))
		args = append(args, *req.PublicInfo)
		argPos++
	}
	if req.FriendsInfo != nil {
		setClauses = append(setClauses, fmt.Sprintf("friends_info = friends_info || $%d", argPos))
		args = append(args, *req.FriendsInfo)
		argPos++
	}
	if req.PrivateInfo != nil {
		setClauses = append(setClauses, fmt.Sprintf("private_info = private_info || $%d", argPos))
		args = append(args, *req.PrivateInfo)
		argPos++
	}
	if req.MusicPreferences != nil {
		setClauses = append(setClauses, fmt.Sprintf("music_preferences = music_preferences || $%d", argPos))
		args = append(args, *req.MusicPreferences)
		argPos++
	}

	// If no patch values are provided, return early with zero mutations executed
	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argPos)

	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
