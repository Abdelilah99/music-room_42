# Playlist management - run and test guide

Covers issue #39: playlist CRUD endpoints and invite-based access control.

## What was built

- **`server/internal/model/playlist.go`** - `Playlist`, `PlaylistTrack`, `PlaylistWithTracks`, `CreatePlaylistRequest`, `UpdatePlaylistRequest`, `PlaylistListFilter` structs.
- **`server/internal/repository/playlist.go`** - `PlaylistRepository` interface and Postgres implementation. Handles all SQL including the public/owner/invited visibility filter and the read-only track listing.
- **`server/internal/service/playlist.go`** - `PlaylistService` interface and implementation with ownership and access control logic.
- **`server/internal/handler/playlist.go`** - Six Gin handlers wired to the routes below.
- **`server/cmd/main.go`** - Playlist handler wired into `setupRouter`.

No new migration was added. The `playlists`, `playlist_invites` and `playlist_tracks` tables already exist in migration `000002_full_schema.up.sql`.

## Endpoints

All endpoints require a valid `Authorization: Bearer <token>` header.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/playlists` | Create a playlist |
| `GET` | `/api/v1/playlists` | List accessible playlists (optional `?q=` search) |
| `GET` | `/api/v1/playlists/:id` | Get a single playlist with its tracks |
| `PUT` | `/api/v1/playlists/:id` | Update a playlist (owner only) |
| `DELETE` | `/api/v1/playlists/:id` | Delete a playlist and its tracks (owner only) |
| `POST` | `/api/v1/playlists/:id/invites` | Invite a user to a private playlist (owner only) |

## Create a playlist

```bash
curl -X POST http://localhost:8081/api/v1/playlists \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "name": "Road Trip",
    "visibility": "public",
    "license": 0
  }'
```

Fields:

| Field | Required | Type | Notes |
|-------|----------|------|-------|
| `name` | yes | string | Min 1 character |
| `visibility` | yes | string | `public` or `private` |
| `license` | no | int | `0` (anyone edits) or `1` (invited only). Defaults to `0` |

License `2` (geofence + time window) does not apply to playlists and is rejected with `400`.

## List playlists

Returns all public playlists plus private playlists the caller owns or is invited to.

```bash
# All accessible playlists
curl http://localhost:8081/api/v1/playlists \
  -H "Authorization: Bearer <token>"

# Filter by name
curl "http://localhost:8081/api/v1/playlists?q=road" \
  -H "Authorization: Bearer <token>"
```

## Get a playlist with its tracks

```bash
curl http://localhost:8081/api/v1/playlists/<id> \
  -H "Authorization: Bearer <token>"
```

The response embeds the playlist fields plus a `tracks` array ordered by `position`
(empty until tracks are added in issue #40). A caller with no access to a private
playlist receives `404`.

## Update a playlist

Only the owner can update. Non-owners receive `404`. All fields are optional.

```bash
curl -X PUT http://localhost:8081/api/v1/playlists/<id> \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"name": "Road Trip 2026", "visibility": "private", "license": 1}'
```

## Delete a playlist

Only the owner can delete. Deleting a playlist cascades to its invites and tracks.

```bash
curl -X DELETE http://localhost:8081/api/v1/playlists/<id> \
  -H "Authorization: Bearer <token>"
```

## Invite a user to a private playlist

Only the owner can invite. Inviting an already-invited user is a no-op (idempotent).
Inviting a non-existent user returns `404`.

```bash
curl -X POST http://localhost:8081/api/v1/playlists/<id>/invites \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"user_id": "<target-user-uuid>"}'
```

## Access control rules

| Action | Rule |
|--------|------|
| See playlist in list | Public playlist, or owner, or invited |
| Get playlist by ID | Public playlist, or owner, or invited — otherwise `404` |
| Update / Delete | Owner only — non-owners get `404` |
| Invite | Owner only — non-owners get `404` |

Using `404` instead of `403` for private playlists is intentional: it avoids leaking
whether a private playlist exists to users who have no access to it.

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
go test ./internal/service/... -v -run TestPlaylist
```

What each test covers:

| Test | What it proves |
|------|----------------|
| `TestPlaylistService_Create` | Owner ID is correctly passed to the repository |
| `TestPlaylistService_Get_Accessible_IncludesTracks` | An accessible playlist is returned with its tracks attached |
| `TestPlaylistService_Get_NoTracks_ReturnsEmptySlice` | Tracks default to a non-nil empty slice |
| `TestPlaylistService_Get_PrivateNotInvited_Returns404` | `pgx.ErrNoRows` from the repo maps to `ErrPlaylistNotFound` |
| `TestPlaylistService_Update_NotOwner_Returns404` | Non-owner update attempt returns `ErrPlaylistNotFound` |
| `TestPlaylistService_Delete_Owner` | Owner can delete and repository `Delete` is called |
| `TestPlaylistService_Delete_NotOwner_Returns404` | Non-owner delete returns `ErrPlaylistNotFound` |
| `TestPlaylistService_Invite_Owner` | Owner invite calls `AddInvite` with correct IDs |
| `TestPlaylistService_Invite_NotOwner_Returns404` | Non-owner invite returns `ErrPlaylistNotFound` |
| `TestPlaylistService_Invite_NonExistentUser_Returns404` | A foreign-key violation maps to `ErrUserNotFound` |
