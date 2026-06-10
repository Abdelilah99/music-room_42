package repository

import (
	"context"
	"fmt"

	"music-room/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TrackRepository interface {
	Add(ctx context.Context, eventID, suggestedBy uuid.UUID, req model.SuggestTrackRequest) (*model.Track, error)
	GetQueue(ctx context.Context, eventID uuid.UUID) ([]model.Track, error)
	GetByID(ctx context.Context, trackID, eventID uuid.UUID) (*model.Track, error)
	Vote(ctx context.Context, trackID, userID uuid.UUID) error
	Delete(ctx context.Context, trackID uuid.UUID) error
}

type trackRepository struct {
	pool *pgxpool.Pool
}

func NewTrackRepository(pool *pgxpool.Pool) TrackRepository {
	return &trackRepository{pool: pool}
}

const trackCols = `id, event_id, external_id, title, artist, suggested_by, vote_count, created_at`

func scanTrack(row interface{ Scan(...any) error }) (*model.Track, error) {
	var t model.Track
	err := row.Scan(&t.ID, &t.EventID, &t.ExternalID, &t.Title, &t.Artist, &t.SuggestedBy, &t.VoteCount, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *trackRepository) Add(ctx context.Context, eventID, suggestedBy uuid.UUID, req model.SuggestTrackRequest) (*model.Track, error) {
	query := fmt.Sprintf(`
		INSERT INTO tracks (event_id, external_id, title, artist, suggested_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING %s`, trackCols)

	return scanTrack(r.pool.QueryRow(ctx, query, eventID, req.ExternalID, req.Title, req.Artist, suggestedBy))
}

func (r *trackRepository) GetQueue(ctx context.Context, eventID uuid.UUID) ([]model.Track, error) {
	query := fmt.Sprintf(`SELECT %s FROM tracks WHERE event_id = $1 ORDER BY vote_count DESC, created_at ASC`, trackCols)

	rows, err := r.pool.Query(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []model.Track
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, *t)
	}
	return tracks, rows.Err()
}

func (r *trackRepository) GetByID(ctx context.Context, trackID, eventID uuid.UUID) (*model.Track, error) {
	query := fmt.Sprintf(`SELECT %s FROM tracks WHERE id = $1 AND event_id = $2`, trackCols)
	return scanTrack(r.pool.QueryRow(ctx, query, trackID, eventID))
}

// Vote inserts a vote row and atomically increments vote_count in a single transaction.
// The UPDATE uses vote_count = vote_count + 1 — never a read-then-write.
func (r *trackRepository) Vote(ctx context.Context, trackID, userID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx, `INSERT INTO votes (track_id, user_id) VALUES ($1, $2)`, trackID, userID); err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, `UPDATE tracks SET vote_count = vote_count + 1 WHERE id = $1`, trackID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *trackRepository) Delete(ctx context.Context, trackID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM tracks WHERE id = $1`, trackID)
	return err
}
