package repository_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"music-room/internal/model"
	"music-room/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// TestTrackRepository_Vote_AtomicIncrement proves that concurrent votes from
// two users each increment vote_count exactly once via vote_count = vote_count + 1,
// with no lost-update from a read-then-write pattern.
func TestTrackRepository_Vote_AtomicIncrement(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	// seed: owner user
	ownerID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, username, email, password_hash) VALUES ($1,$2,$3,'x')`,
		ownerID, "testowner_"+ownerID.String()[:8], ownerID.String()+"@test.local",
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, ownerID) })

	// seed: voter users
	voter1, voter2 := uuid.New(), uuid.New()
	for _, uid := range []uuid.UUID{voter1, voter2} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO users (id, username, email, password_hash) VALUES ($1,$2,$3,'x')`,
			uid, "voter_"+uid.String()[:8], uid.String()+"@test.local",
		); err != nil {
			t.Fatalf("seed voter: %v", err)
		}
		t.Cleanup(func(id uuid.UUID) func() {
			return func() { pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, id) }
		}(uid))
	}

	// seed: event
	eventID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO events (id, owner_id, name, visibility) VALUES ($1,$2,'test-event','public')`,
		eventID, ownerID,
	); err != nil {
		t.Fatalf("seed event: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `DELETE FROM events WHERE id=$1`, eventID) })

	// seed: track
	trackRepo := repository.NewTrackRepository(pool)
	track, err := trackRepo.Add(ctx, eventID, ownerID, model.SuggestTrackRequest{
		ExternalID: "dz:atomic-test",
		Title:      "Atomic Test",
		Artist:     "Test Artist",
	})
	if err != nil {
		t.Fatalf("seed track: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `DELETE FROM tracks WHERE id=$1`, track.ID) })

	// two concurrent votes
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, uid := range []uuid.UUID{voter1, voter2} {
		wg.Add(1)
		go func(userID uuid.UUID) {
			defer wg.Done()
			errs <- trackRepo.Vote(ctx, track.ID, userID)
		}(uid)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("concurrent vote failed: %v", err)
		}
	}

	// verify vote_count is exactly 2
	var count int
	if err := pool.QueryRow(ctx, `SELECT vote_count FROM tracks WHERE id=$1`, track.ID).Scan(&count); err != nil {
		t.Fatalf("read vote_count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected vote_count=2, got %d", count)
	}
}
