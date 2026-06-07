# Track suggestion and voting

Covers issue #26: per-event track queue, voting system, and concurrent vote safety.

## What was built

- **`server/internal/model/track.go`** - `Track` and `SuggestTrackRequest` structs.
- **`server/internal/repository/track.go`** - `TrackRepository` interface and Postgres implementation with atomic vote increment.
- **`server/internal/service/track.go`** - `TrackService` interface with access control and error mapping.
- **`server/internal/service/track_test.go`** - 10 unit tests including a concurrent vote test.
- **`server/internal/handler/track.go`** - Four Gin handlers wired to the routes below.
- **`server/cmd/main.go`** - Track handler wired into `setupRouter`.

No new migration was added. The `tracks` and `votes` tables already exist in migration `000002_full_schema.up.sql`.

## Endpoints

All endpoints require a valid `Authorization: Bearer <token>` header.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/events/:id/tracks` | Suggest a track for an event queue |
| `GET` | `/api/v1/events/:id/queue` | Get the event queue ordered by votes |
| `POST` | `/api/v1/events/:id/tracks/:trackId/vote` | Cast a vote on a track |
| `DELETE` | `/api/v1/events/:id/tracks/:trackId` | Remove a track (event owner only) |

## Suggest a track

```bash
curl -X POST http://localhost:8081/api/v1/events/<eventId>/tracks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "external_id": "dz:1234567",
    "title": "Blinding Lights",
    "artist": "The Weeknd"
  }'
```

Fields:

| Field | Required | Notes |
|-------|----------|-------|
| `external_id` | yes | Deezer track ID (e.g. `dz:1234567`) |
| `title` | yes | Track title |
| `artist` | yes | Artist name |

Returns `409` if the same `external_id` is already in this event.

## Get the queue

Returns all tracks for the event sorted by `vote_count` descending, then `created_at` ascending for ties.

```bash
curl http://localhost:8081/api/v1/events/<eventId>/queue \
  -H "Authorization: Bearer <token>"
```

## Vote on a track

```bash
curl -X POST http://localhost:8081/api/v1/events/<eventId>/tracks/<trackId>/vote \
  -H "Authorization: Bearer <token>"
```

Returns `200 {"message": "vote cast"}` on success.

- Returns `409 {"error": "already voted on this track"}` if the caller already voted.
- The vote increment is fully atomic: `UPDATE tracks SET vote_count = vote_count + 1 WHERE id = $1`. No read-then-write.

## Remove a track

Only the event owner can remove tracks. Associated votes are deleted by cascade.

```bash
curl -X DELETE http://localhost:8081/api/v1/events/<eventId>/tracks/<trackId> \
  -H "Authorization: Bearer <token>"
```

Returns `200 {"message": "track removed"}` on success. Non-owners receive `404`.

## Access control rules

| Action | Rule |
|--------|------|
| Suggest / view queue / vote | Event must be accessible (public, owner, or invited) |
| Remove track | Event owner only |

Non-accessible private events and non-existent resources both return `404`.

## Concurrency

Two users voting on the same track simultaneously results in `vote_count` being incremented by exactly 2. This is guaranteed by PostgreSQL's row-level locking on the UPDATE — no application-level read-then-write occurs.

## Requirements

Only Docker is needed.

## Start the stack

```bash
docker compose up --build
```

## Run migrations

```bash
docker compose run --rm server go run ./cmd/migrate/main.go up
```

## Run the unit tests

```bash
cd server
go test ./internal/service/... -v -run TestTrack
```

Expected output:

```
--- PASS: TestTrackService_Suggest_Success
--- PASS: TestTrackService_Suggest_EventNotFound
--- PASS: TestTrackService_Suggest_Duplicate_Returns409
--- PASS: TestTrackService_GetQueue_Success
--- PASS: TestTrackService_Vote_Success
--- PASS: TestTrackService_Vote_AlreadyVoted_Returns409
--- PASS: TestTrackService_Vote_TrackNotFound
--- PASS: TestTrackService_Vote_Concurrent
--- PASS: TestTrackService_DeleteTrack_Owner
--- PASS: TestTrackService_DeleteTrack_NotOwner_Returns404
```

What each test covers:

| Test | What it proves |
|------|----------------|
| `Suggest_Success` | Correct eventID and callerID passed to repository |
| `Suggest_EventNotFound` | `pgx.ErrNoRows` from event repo maps to `ErrEventNotFound` |
| `Suggest_Duplicate_Returns409` | Unique violation maps to `ErrTrackAlreadyExists` |
| `GetQueue_Success` | Correct eventID forwarded to repository |
| `Vote_Success` | Happy path: access check, track check, vote recorded |
| `Vote_AlreadyVoted_Returns409` | Unique violation on votes maps to `ErrAlreadyVoted` |
| `Vote_TrackNotFound` | Non-existent track in event returns `ErrTrackNotFound` |
| `Vote_Concurrent` | Two goroutines vote simultaneously, both succeed, counter = 2 |
| `DeleteTrack_Owner` | Owner can delete and repository Delete is called |
| `DeleteTrack_NotOwner_Returns404` | Non-owner returns `ErrEventNotFound` |

## Run with race detector

```bash
cd server
go test -race ./internal/service/... -run TestTrackService_Vote_Concurrent
```
