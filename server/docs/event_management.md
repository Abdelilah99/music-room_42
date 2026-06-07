# Event management - run and test guide

Covers issue #25: event CRUD endpoints and invite-based access control.

## What was built

- **`server/internal/model/event.go`** - `Event`, `CreateEventRequest`, `UpdateEventRequest`, `EventListFilter` structs.
- **`server/internal/repository/event.go`** - `EventRepository` interface and Postgres implementation. Handles all SQL including Haversine-based location filtering.
- **`server/internal/service/event.go`** - `EventService` interface and implementation with access control logic.
- **`server/internal/handler/event.go`** - Six Gin handlers wired to the routes below.
- **`server/cmd/main.go`** - Event handler wired into `setupRouter`.

No new migration was added. The `events` and `event_invites` tables already exist in migration `000002_full_schema.up.sql`.

## Endpoints

All endpoints require a valid `Authorization: Bearer <token>` header.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/events` | Create an event |
| `GET` | `/api/v1/events` | List accessible events (with optional filters) |
| `GET` | `/api/v1/events/:id` | Get a single event |
| `PUT` | `/api/v1/events/:id` | Update an event (owner only) |
| `DELETE` | `/api/v1/events/:id` | Delete an event (owner only) |
| `POST` | `/api/v1/events/:id/invites` | Invite a user to an event (owner only) |

## Create an event

```bash
curl -X POST http://localhost:8081/api/v1/events \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "name": "Summer Jam",
    "visibility": "public",
    "lat": 48.8566,
    "lng": 2.3522,
    "radius": 5000,
    "vote_start": "2026-07-01T18:00:00Z",
    "vote_end": "2026-07-01T22:00:00Z",
    "license": 1
  }'
```

Fields:

| Field | Required | Type | Notes |
|-------|----------|------|-------|
| `name` | yes | string | Min 1 character |
| `visibility` | yes | string | `public` or `private` |
| `lat` | no | float | Event latitude |
| `lng` | no | float | Event longitude |
| `radius` | no | float | Event coverage radius in meters |
| `vote_start` | no | RFC3339 | Start of voting window |
| `vote_end` | no | RFC3339 | End of voting window |
| `license` | no | int | `0`, `1`, or `2` (defaults to `0`) |

## List events with filters

Returns all public events plus private events the caller owns or is invited to.

```bash
# No filter - all accessible events
curl http://localhost:8081/api/v1/events \
  -H "Authorization: Bearer <token>"

# Filter by name
curl "http://localhost:8081/api/v1/events?q=summer" \
  -H "Authorization: Bearer <token>"

# Filter by location - all three params required (radius in meters)
curl "http://localhost:8081/api/v1/events?lat=48.85&lng=2.35&radius=10000" \
  -H "Authorization: Bearer <token>"

# Both filters can be combined
curl "http://localhost:8081/api/v1/events?q=jam&lat=48.85&lng=2.35&radius=10000" \
  -H "Authorization: Bearer <token>"
```

The location filter uses the Haversine formula. It returns events whose stored coordinates fall within `radius` meters of the given `lat`/`lng`. Events with no coordinates are excluded when a location filter is active.

## Update an event

Only the owner can update. Non-owners receive `404` (does not reveal existence of private events).

All fields are optional in the request body.

```bash
curl -X PUT http://localhost:8081/api/v1/events/<id> \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"name": "Summer Jam 2026", "visibility": "private"}'
```

## Delete an event

Only the owner can delete.

```bash
curl -X DELETE http://localhost:8081/api/v1/events/<id> \
  -H "Authorization: Bearer <token>"
```

## Invite a user to a private event

Only the owner can invite. Inviting an already-invited user is a no-op (idempotent).

```bash
curl -X POST http://localhost:8081/api/v1/events/<id>/invites \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"user_id": "<target-user-uuid>"}'
```

## Access control rules

| Action | Rule |
|--------|------|
| See event in list | Public event, or owner, or invited |
| Get event by ID | Public event, or owner, or invited — otherwise `404` |
| Update / Delete | Owner only — non-owners get `404` |
| Invite | Owner only — non-owners get `404` |

Using `404` instead of `403` for private events is intentional: it avoids leaking whether a private event exists to users who have no access to it.

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
go test ./internal/service/... -v -run TestEvent
```

Expected output:

```
--- PASS: TestEventService_Create
--- PASS: TestEventService_Get_PublicEvent
--- PASS: TestEventService_Get_PrivateNotInvited_Returns404
--- PASS: TestEventService_Update_NotOwner_Returns404
--- PASS: TestEventService_Delete_Owner
--- PASS: TestEventService_Delete_NotOwner_Returns404
--- PASS: TestEventService_Invite_NotOwner_Returns404
--- PASS: TestEventService_Invite_Owner
```

What each test covers:

| Test | What it proves |
|------|----------------|
| `TestEventService_Create` | Owner ID is correctly passed to the repository |
| `TestEventService_Get_PublicEvent` | Accessible event is returned |
| `TestEventService_Get_PrivateNotInvited_Returns404` | `pgx.ErrNoRows` from the repo maps to `ErrEventNotFound` |
| `TestEventService_Update_NotOwner_Returns404` | Non-owner update attempt returns `ErrEventNotFound` |
| `TestEventService_Delete_Owner` | Owner can delete and repository `Delete` is called |
| `TestEventService_Delete_NotOwner_Returns404` | Non-owner delete returns `ErrEventNotFound` |
| `TestEventService_Invite_NotOwner_Returns404` | Non-owner invite returns `ErrEventNotFound` |
| `TestEventService_Invite_Owner` | Owner invite calls `AddInvite` with correct IDs |
