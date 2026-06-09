# License Enforcement and Geolocation Check

## Overview

The vote endpoint enforces access rules based on the event's license level.
Three license levels exist:

| License | Name | Voting restriction |
|---------|------|--------------------|
| 0 | Open | Anyone who can see the event can vote |
| 1 | Invite-only | Caller must be in `event_invites` for the event |
| 2 | Geofence + time window | Caller must supply GPS coords within the event radius and vote within the configured window |

## Endpoint

```
POST /api/v1/events/:id/tracks/:trackId/vote
Authorization: Bearer <JWT>
Content-Type: application/json  (optional for license 0 and 1)

Body (required only for license 2):
{
  "lat": 48.8566,
  "lng": 2.3522
}
```

## Error codes

| HTTP | Body `error` | Meaning |
|------|-------------|---------|
| 400 | `lat and lng are required to vote on this event` | License 2 event, no GPS provided |
| 403 | `NOT_INVITED` | License 1 event, caller not in invite list |
| 403 | `OUT_OF_RANGE` | License 2 event, caller too far from venue |
| 403 | `VOTING_CLOSED` | License 2 event, current time outside vote_start/vote_end window |
| 409 | `already voted on this track` | Caller already voted for this track |

## Haversine distance

The geofence check uses the Haversine formula to compute great-circle distance in kilometres between the caller's GPS position and the event's stored `lat`/`lng`. The `radius` field on the event is in kilometres.

```go
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64
```

## Creating events with license 2

A license 2 event requires all five fields: `lat`, `lng`, `radius` (km), `vote_start`, `vote_end`.
Missing any of these returns `400 ErrInvalidLicenseConfig`.

```json
POST /api/v1/events
{
  "name": "Concert",
  "visibility": "public",
  "license": 2,
  "lat": 48.8566,
  "lng": 2.3522,
  "radius": 0.5,
  "vote_start": "2026-06-08T19:00:00Z",
  "vote_end": "2026-06-08T22:00:00Z"
}
```

## Test setup (live Docker)

All commands assume the server is running on port 8081.

### Register and login

```bash
curl -s -X POST http://localhost:8081/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"testuser","email":"test@example.com","password":"pass1234"}'

# verify email via mailpit at http://localhost:8025, then:
curl -s -X POST http://localhost:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"pass1234"}'
# -> {"access_token":"<TOKEN>", ...}
```

### License 1: invite-only voting

```bash
# Create event
EVENT=$(curl -s -X POST http://localhost:8081/api/v1/events \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Invite Party","visibility":"public","license":1}')
EVENT_ID=$(echo $EVENT | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])")

# Suggest a track
TRACK=$(curl -s -X POST "http://localhost:8081/api/v1/events/$EVENT_ID/tracks" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"external_id":"dz:123","title":"My Song","artist":"Artist"}')
TRACK_ID=$(echo $TRACK | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])")

# Vote without invite -> 403 NOT_INVITED
curl -s -X POST "http://localhost:8081/api/v1/events/$EVENT_ID/tracks/$TRACK_ID/vote" \
  -H "Authorization: Bearer $OTHER_TOKEN"

# Invite the user, then vote -> 200
curl -s -X POST "http://localhost:8081/api/v1/events/$EVENT_ID/invites" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"<other-user-uuid>"}'
curl -s -X POST "http://localhost:8081/api/v1/events/$EVENT_ID/tracks/$TRACK_ID/vote" \
  -H "Authorization: Bearer $OTHER_TOKEN"
```

### License 2: geofence + time window

```bash
# Create event (Paris, 10km radius, open window)
EVENT=$(curl -s -X POST http://localhost:8081/api/v1/events \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name":"Paris Gig","visibility":"public","license":2,
    "lat":48.8566,"lng":2.3522,"radius":10,
    "vote_start":"2026-06-08T18:00:00Z","vote_end":"2026-06-08T23:00:00Z"
  }')
EVENT_ID=$(echo $EVENT | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])")

TRACK_ID=...  # suggest as above

# No GPS body -> 400
curl -s -X POST "http://localhost:8081/api/v1/events/$EVENT_ID/tracks/$TRACK_ID/vote" \
  -H "Authorization: Bearer $TOKEN"

# London coords -> 403 OUT_OF_RANGE
curl -s -X POST "http://localhost:8081/api/v1/events/$EVENT_ID/tracks/$TRACK_ID/vote" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"lat":51.5074,"lng":-0.1278}'

# Paris coords -> 200
curl -s -X POST "http://localhost:8081/api/v1/events/$EVENT_ID/tracks/$TRACK_ID/vote" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"lat":48.8566,"lng":2.3522}'
```
