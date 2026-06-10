package repository

import (
	"context"
	"fmt"

	"music-room/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlaylistRepository interface {
	Create(ctx context.Context, ownerID uuid.UUID, req model.CreatePlaylistRequest) (*model.Playlist, error)
	List(ctx context.Context, callerID uuid.UUID, f model.PlaylistListFilter) ([]model.Playlist, error)
	GetAccessible(ctx context.Context, playlistID, callerID uuid.UUID) (*model.Playlist, error)
	GetByIDForOwner(ctx context.Context, playlistID, ownerID uuid.UUID) (*model.Playlist, error)
	Update(ctx context.Context, playlistID uuid.UUID, req model.UpdatePlaylistRequest) (*model.Playlist, error)
	Delete(ctx context.Context, playlistID uuid.UUID) error
	AddInvite(ctx context.Context, playlistID, userID uuid.UUID) error
	ListTracks(ctx context.Context, playlistID uuid.UUID) ([]model.PlaylistTrack, error)
}

type playlistRepository struct {
	pool *pgxpool.Pool
}

func NewPlaylistRepository(pool *pgxpool.Pool) PlaylistRepository {
	return &playlistRepository{pool: pool}
}

const playlistCols = `id, owner_id, name, visibility, license, created_at`

func scanPlaylist(row interface {
	Scan(...any) error
}) (*model.Playlist, error) {
	var p model.Playlist
	err := row.Scan(&p.ID, &p.OwnerID, &p.Name, &p.Visibility, &p.License, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *playlistRepository) Create(ctx context.Context, ownerID uuid.UUID, req model.CreatePlaylistRequest) (*model.Playlist, error) {
	query := fmt.Sprintf(`
		INSERT INTO playlists (owner_id, name, visibility, license)
		VALUES ($1, $2, $3, $4)
		RETURNING %s`, playlistCols)

	return scanPlaylist(r.pool.QueryRow(ctx, query, ownerID, req.Name, req.Visibility, req.License))
}

func (r *playlistRepository) List(ctx context.Context, callerID uuid.UUID, f model.PlaylistListFilter) ([]model.Playlist, error) {
	// Base visibility filter: public OR owned by caller OR caller is invited.
	baseWhere := `
		(p.visibility = 'public' OR p.owner_id = $1
		 OR EXISTS (SELECT 1 FROM playlist_invites pi WHERE pi.playlist_id = p.id AND pi.user_id = $1))`

	args := []any{callerID}
	paramIdx := 2

	if f.Q != "" {
		baseWhere += fmt.Sprintf(` AND p.name ILIKE $%d`, paramIdx)
		args = append(args, "%"+f.Q+"%")
		paramIdx++
	}

	query := fmt.Sprintf(`SELECT %s FROM playlists p WHERE %s ORDER BY p.created_at DESC`, playlistCols, baseWhere)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []model.Playlist
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			return nil, err
		}
		playlists = append(playlists, *p)
	}
	return playlists, rows.Err()
}

// GetAccessible returns the playlist only if the caller can see it
// (public, owner, or invited). Returns pgx.ErrNoRows if not accessible.
func (r *playlistRepository) GetAccessible(ctx context.Context, playlistID, callerID uuid.UUID) (*model.Playlist, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM playlists p
		WHERE p.id = $1
		AND (p.visibility = 'public' OR p.owner_id = $2
		     OR EXISTS (SELECT 1 FROM playlist_invites pi WHERE pi.playlist_id = p.id AND pi.user_id = $2))`,
		playlistCols)

	return scanPlaylist(r.pool.QueryRow(ctx, query, playlistID, callerID))
}

// GetByIDForOwner returns the playlist only if callerID is the owner.
// Used for write operations (update, delete, invite).
func (r *playlistRepository) GetByIDForOwner(ctx context.Context, playlistID, ownerID uuid.UUID) (*model.Playlist, error) {
	query := fmt.Sprintf(`SELECT %s FROM playlists p WHERE p.id = $1 AND p.owner_id = $2`, playlistCols)
	return scanPlaylist(r.pool.QueryRow(ctx, query, playlistID, ownerID))
}

func (r *playlistRepository) Update(ctx context.Context, playlistID uuid.UUID, req model.UpdatePlaylistRequest) (*model.Playlist, error) {
	query := fmt.Sprintf(`
		UPDATE playlists SET
			name       = COALESCE($2, name),
			visibility = COALESCE($3, visibility),
			license    = COALESCE($4, license)
		WHERE id = $1
		RETURNING %s`, playlistCols)

	return scanPlaylist(r.pool.QueryRow(ctx, query, playlistID, req.Name, req.Visibility, req.License))
}

func (r *playlistRepository) Delete(ctx context.Context, playlistID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM playlists WHERE id = $1`, playlistID)
	return err
}

func (r *playlistRepository) AddInvite(ctx context.Context, playlistID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO playlist_invites (playlist_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		playlistID, userID,
	)
	return err
}

func (r *playlistRepository) ListTracks(ctx context.Context, playlistID uuid.UUID) ([]model.PlaylistTrack, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, playlist_id, external_id, title, artist, position, added_by, created_at
		FROM playlist_tracks
		WHERE playlist_id = $1
		ORDER BY position ASC`, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []model.PlaylistTrack
	for rows.Next() {
		var t model.PlaylistTrack
		err := rows.Scan(
			&t.ID, &t.PlaylistID, &t.ExternalID, &t.Title,
			&t.Artist, &t.Position, &t.AddedBy, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}
