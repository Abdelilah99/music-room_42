package repository

import (
	"context"
	"fmt"

	"music-room/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepository interface {
	Create(ctx context.Context, ownerID uuid.UUID, req model.CreateEventRequest) (*model.Event, error)
	List(ctx context.Context, callerID uuid.UUID, f model.EventListFilter) ([]model.Event, error)
	GetAccessible(ctx context.Context, eventID, callerID uuid.UUID) (*model.Event, error)
	GetByIDForOwner(ctx context.Context, eventID, ownerID uuid.UUID) (*model.Event, error)
	Update(ctx context.Context, eventID uuid.UUID, req model.UpdateEventRequest) (*model.Event, error)
	Delete(ctx context.Context, eventID uuid.UUID) error
	AddInvite(ctx context.Context, eventID, userID uuid.UUID) error
	IsInvited(ctx context.Context, eventID, userID uuid.UUID) (bool, error)
}

type eventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) EventRepository {
	return &eventRepository{pool: pool}
}

const eventCols = `id, owner_id, name, visibility, lat, lng, radius, vote_start, vote_end, license, created_at`

func scanEvent(row interface {
	Scan(...any) error
}) (*model.Event, error) {
	var e model.Event
	err := row.Scan(
		&e.ID, &e.OwnerID, &e.Name, &e.Visibility,
		&e.Lat, &e.Lng, &e.Radius,
		&e.VoteStart, &e.VoteEnd, &e.License, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *eventRepository) Create(ctx context.Context, ownerID uuid.UUID, req model.CreateEventRequest) (*model.Event, error) {
	query := fmt.Sprintf(`
		INSERT INTO events (owner_id, name, visibility, lat, lng, radius, vote_start, vote_end, license)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING %s`, eventCols)

	return scanEvent(r.pool.QueryRow(ctx, query,
		ownerID, req.Name, req.Visibility, req.Lat, req.Lng,
		req.Radius, req.VoteStart, req.VoteEnd, req.License,
	))
}

func (r *eventRepository) List(ctx context.Context, callerID uuid.UUID, f model.EventListFilter) ([]model.Event, error) {
	// Base visibility filter: public OR owned by caller OR caller is invited.
	baseWhere := `
		(e.visibility = 'public' OR e.owner_id = $1
		 OR EXISTS (SELECT 1 FROM event_invites ei WHERE ei.event_id = e.id AND ei.user_id = $1))`

	args := []any{callerID}
	paramIdx := 2

	if f.Q != "" {
		baseWhere += fmt.Sprintf(` AND e.name ILIKE $%d`, paramIdx)
		args = append(args, "%"+f.Q+"%")
		paramIdx++
	}

	// Location filter: only applied when all three params are present.
	// $N=lat, $N+1=lng, $N+2=radius (meters). Lat is referenced twice at $N.
	if f.Lat != nil && f.Lng != nil && f.Radius != nil {
		baseWhere += fmt.Sprintf(`
			AND e.lat IS NOT NULL AND e.lng IS NOT NULL
			AND 6371000 * 2 * asin(sqrt(
				pow(sin(radians((e.lat - $%d) / 2)), 2) +
				cos(radians($%d)) * cos(radians(e.lat)) * pow(sin(radians((e.lng - $%d) / 2)), 2)
			)) <= $%d`, paramIdx, paramIdx, paramIdx+1, paramIdx+2)
		args = append(args, *f.Lat, *f.Lng, *f.Radius)
		paramIdx += 3
	}

	query := fmt.Sprintf(`SELECT %s FROM events e WHERE %s ORDER BY e.created_at DESC`, eventCols, baseWhere)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *e)
	}
	return events, rows.Err()
}

// GetAccessible returns the event only if the caller can see it
// (public, owner, or invited). Returns pgx.ErrNoRows if not accessible.
func (r *eventRepository) GetAccessible(ctx context.Context, eventID, callerID uuid.UUID) (*model.Event, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM events e
		WHERE e.id = $1
		AND (e.visibility = 'public' OR e.owner_id = $2
		     OR EXISTS (SELECT 1 FROM event_invites ei WHERE ei.event_id = e.id AND ei.user_id = $2))`,
		eventCols)

	return scanEvent(r.pool.QueryRow(ctx, query, eventID, callerID))
}

// GetByIDForOwner returns the event only if callerID is the owner.
// Used for write operations (update, delete, invite).
func (r *eventRepository) GetByIDForOwner(ctx context.Context, eventID, ownerID uuid.UUID) (*model.Event, error) {
	query := fmt.Sprintf(`SELECT %s FROM events e WHERE e.id = $1 AND e.owner_id = $2`, eventCols)
	return scanEvent(r.pool.QueryRow(ctx, query, eventID, ownerID))
}

func (r *eventRepository) Update(ctx context.Context, eventID uuid.UUID, req model.UpdateEventRequest) (*model.Event, error) {
	query := fmt.Sprintf(`
		UPDATE events SET
			name       = COALESCE($2, name),
			visibility = COALESCE($3, visibility),
			lat        = CASE WHEN $4::float8 IS NOT NULL THEN $4 ELSE lat END,
			lng        = CASE WHEN $5::float8 IS NOT NULL THEN $5 ELSE lng END,
			radius     = CASE WHEN $6::float8 IS NOT NULL THEN $6 ELSE radius END,
			vote_start = CASE WHEN $7::timestamptz IS NOT NULL THEN $7 ELSE vote_start END,
			vote_end   = CASE WHEN $8::timestamptz IS NOT NULL THEN $8 ELSE vote_end END,
			license    = COALESCE($9, license)
		WHERE id = $1
		RETURNING %s`, eventCols)

	return scanEvent(r.pool.QueryRow(ctx, query,
		eventID, req.Name, req.Visibility,
		req.Lat, req.Lng, req.Radius,
		req.VoteStart, req.VoteEnd, req.License,
	))
}

func (r *eventRepository) Delete(ctx context.Context, eventID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM events WHERE id = $1`, eventID)
	return err
}

func (r *eventRepository) AddInvite(ctx context.Context, eventID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO event_invites (event_id, user_id) VALUES ($1, $2)`,
		eventID, userID,
	)
	return err
}

func (r *eventRepository) IsInvited(ctx context.Context, eventID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM event_invites WHERE event_id=$1 AND user_id=$2)`,
		eventID, userID,
	).Scan(&exists)
	return exists, err
}
