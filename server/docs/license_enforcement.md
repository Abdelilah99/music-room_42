# License enforcement and geolocation check

Covers issue #27: per-event voting access control based on license level, including invite checks and GPS-based geofencing.

## What was built

- **`server/internal/repository/event.go`** - Added `IsInvited(ctx, eventID, userID)` to `EventRepository` interface and Postgres implementation.
- **`server/internal/service/track.go`** - Added `enforceLicense` helper, `HaversineKm` function, and four new sentinel errors: `ErrNotInvited`, `ErrOutOfRange`, `ErrVotingClosed`, `ErrMissingCoords`.
- **`server/internal/service/track_test.go`** - License enforcement tests for every scenario, plus three Haversine unit tests using known reference distances.
- **`server/internal/handler/track.go`** - Maps new errors to `403`/`400` responses with machine-readable codes.

No new migration. The `event_invites` table and license/geolocation columns on `events` already exist in `000002_full_schema.up.sql`.

## License levels

| License | Name | Who can vote |
|---------|------|--------------|
| `0` | Open | Any authenticated user |
| `1` | Invite-only | Users listed in `event_invites` for this event |
| `2` | Geofenced | Invited users within `radius` km of the event and within the `vote_start`/`vote_end` window |

The license is set when creating or updating an event. See `event_management.md` for event CRUD details.

## Vote endpoint

```
POST /api/v1/events/:id/tracks/:trackId/vote
```

The request body is optional for license 0 and 1. License 2 requires `lat` and `lng`.

```bash
# License 0 or 1 — no body needed
curl -X POST http://localhost:8081/api/v1/events/<eventId>/tracks/<trackId>/vote \
  -H "Authorization: Bearer <token>"

# License 2 — GPS coordinates required
curl -X POST http://localhost:8081/api/v1/events/<eventId>/tracks/<trackId>/vote \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"lat": 48.8566, "lng": 2.3522}'
```

## Response codes

| Status | Body | Meaning |
|--------|------|---------|
| `200` | `{"message": "vote cast"}` | Vote accepted |
| `400` | `{"error": "lat and lng are required to vote on this event"}` | License 2 event but no coordinates sent |
| `403` | `{"error": "NOT_INVITED"}` | License 1 or 2: caller is not in `event_invites` |
| `403` | `{"error": "OUT_OF_RANGE"}` | License 2: Haversine distance exceeds event radius |
| `403` | `{"error": "VOTING_CLOSED"}` | License 2: current time is outside `vote_start`/`vote_end` |
| `409` | `{"error": "already voted on this track"}` | Caller already voted on this track |

## How the geofence works

The server receives the user's coordinates and computes the great-circle distance to the event coordinates using the Haversine formula. No GPS happens server-side — the client is responsible for obtaining and sending accurate coordinates.

The formula, implemented in pure Go with no external library:

```go
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
    const R = 6371.0
    dLat := (lat2 - lat1) * math.Pi / 180
    dLng := (lng2 - lng1) * math.Pi / 180
    a := math.Sin(dLat/2)*math.Sin(dLat/2) +
        math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
            math.Sin(dLng/2)*math.Sin(dLng/2)
    return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
```

`event.radius` is stored in kilometres. If the distance exceeds the radius, the vote is rejected with `OUT_OF_RANGE`.

## Setting up test events

### License 1 — invite-only

```bash
# Create the event
curl -X POST http://localhost:8081/api/v1/events \
  -H "Authorization: Bearer <owner_token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "Invite Only", "visibility": "public", "license": 1}'

# Invite a user
curl -X POST http://localhost:8081/api/v1/events/<eventId>/invites \
  -H "Authorization: Bearer <owner_token>" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "<target_user_id>"}'
```

### License 2 — geofenced with time window

```bash
curl -X POST http://localhost:8081/api/v1/events \
  -H "Authorization: Bearer <owner_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Geo Event",
    "visibility": "public",
    "license": 2,
    "lat": 48.8566,
    "lng": 2.3522,
    "radius": 1,
    "vote_start": "2026-07-01T18:00:00Z",
    "vote_end": "2026-07-01T22:00:00Z"
  }'
```

`radius` is in kilometres. `vote_start` and `vote_end` are ISO 8601 UTC timestamps. All three (`lat`, `lng`, `radius`, `vote_start`, `vote_end`) are required when `license` is `2`.

## Run the unit tests

```bash
cd server
go test ./internal/service/... -v -run "TestTrackService_Vote_License|TestHaversine"
```

Expected output:

```
--- PASS: TestTrackService_Vote_License1_NotInvited_Returns403
--- PASS: TestTrackService_Vote_License1_Invited_Succeeds
--- PASS: TestTrackService_Vote_License2_MissingCoords_Returns400
--- PASS: TestTrackService_Vote_License2_OutOfRange_Returns403
--- PASS: TestTrackService_Vote_License2_VotingClosed_Returns403
--- PASS: TestTrackService_Vote_License2_AllConditionsMet_Succeeds
--- PASS: TestHaversineKm_ParisToBerlin
--- PASS: TestHaversineKm_SamePoint_ReturnsZero
--- PASS: TestHaversineKm_ParisToLondon
```

What each test covers:

| Test | What it proves |
|------|----------------|
| `License1_NotInvited_Returns403` | Non-invited user gets `ErrNotInvited` on license 1 event |
| `License1_Invited_Succeeds` | Invited user can vote on license 1 event |
| `License2_MissingCoords_Returns400` | No coordinates on license 2 event returns `ErrMissingCoords` |
| `License2_OutOfRange_Returns403` | London coordinates rejected on Paris 1 km radius event |
| `License2_VotingClosed_Returns403` | Vote outside the time window returns `ErrVotingClosed` |
| `License2_AllConditionsMet_Succeeds` | All license 2 conditions met — vote accepted |
| `HaversineKm_ParisToBerlin` | Known distance ~878 km within 10 km tolerance |
| `HaversineKm_SamePoint_ReturnsZero` | Identical coordinates return 0 |
| `HaversineKm_ParisToLondon` | Known distance ~340 km within 10 km tolerance |

## Run the full test suite

```bash
cd server
go test ./...
```

## Start the stack

```bash
docker compose up --build
```
