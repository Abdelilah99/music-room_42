# Playlist Track Management

Endpoints for adding, removing, and reordering tracks within a playlist.

## Endpoints

### Add a track

```
POST /api/v1/playlists/:id/tracks
```

**Body**
```json
{ "external_id": "138547415", "title": "Creep", "artist": "Radiohead" }
```

**Responses**
| Status | Meaning |
|--------|---------|
| 201 | Track added, returns the new `PlaylistTrack` object |
| 400 | Missing required field |
| 403 | `EDIT_NOT_ALLOWED` - caller is not owner or invited (License 1 playlist) |
| 404 | Playlist not found |
| 409 | Track already in this playlist (duplicate `external_id`) |

---

### Remove a track

```
DELETE /api/v1/playlists/:id/tracks/:trackId
```

After deletion, remaining tracks are recompacted so positions are gapless (1, 2, 3...).

**Responses**
| Status | Meaning |
|--------|---------|
| 200 | Track removed |
| 403 | `EDIT_NOT_ALLOWED` |
| 404 | Playlist or track not found |

---

### Move a track

```
PATCH /api/v1/playlists/:id/tracks/:trackId/position
```

**Body**
```json
{ "position": 2 }
```

Moves the track to the given position and shifts all affected tracks to maintain a contiguous order. Uses a PostgreSQL serializable transaction so two concurrent reorder requests on the same playlist cannot interleave and corrupt positions.

**Responses**
| Status | Meaning |
|--------|---------|
| 200 | Track moved |
| 400 | `position` is missing, less than 1, or greater than the track count |
| 403 | `EDIT_NOT_ALLOWED` |
| 404 | Playlist or track not found |

---

## License enforcement

| License | Who can add / remove / reorder |
|---------|-------------------------------|
| 0 | Any authenticated user who can see the playlist |
| 1 | Owner and invited users only (returns 403 with `EDIT_NOT_ALLOWED` otherwise) |

---

## Real-time WebSocket

```
GET /api/v1/playlists/:id/ws?token=<jwt>
```

Upgrades to a WebSocket connection scoped to a single playlist. After a successful upgrade, the server pushes an event frame to every connected client whenever a track is added, removed, or moved.

**Authentication:** pass the JWT as the `token` query parameter (browsers cannot set `Authorization` headers on WebSocket upgrades). The server also accepts a `Bearer` header for native clients. Connections for playlists the caller cannot access are rejected with `404` before the upgrade — this intentionally does not distinguish "not found" from "access denied" to avoid leaking whether a private playlist exists.

**Events received by clients**

| Event type | Payload |
|------------|---------|
| `track_added` | `{ "type": "track_added", "track": { ...PlaylistTrack } }` |
| `track_removed` | `{ "type": "track_removed", "track_id": "<uuid>" }` |
| `track_moved` | `{ "type": "track_moved", "track_id": "<uuid>", "position": N }` |

**Handling `track_moved` on the client:** moving a track to a new position reflows every track between the old and new positions — it is not a single-field patch. The correct client behaviour is to remove the track from its current index in the local list and reinsert it at `position - 1` (0-based). Patching only the moved track's field without reinserting it will cause the client's position order to drift from the server's.

**Self-echo:** the client that performs a mutation (add, remove, move) is connected to the same hub, so it will receive the broadcast event in addition to the REST response. Clients must deduplicate by `track_id` (and for `track_added`, by `external_id`) to avoid rendering the same track twice.

The hub for a playlist is created on the first client connection and destroyed when the last client disconnects, so there are no idle goroutine leaks.

---

## Concurrency

The `PATCH position` endpoint runs inside a `SERIALIZABLE` transaction. Postgres will serialise concurrent moves on the same playlist and retry any transaction that would produce a non-serializable result. This guarantees the final position order is always consistent with no skipped or duplicate positions, regardless of how many clients reorder simultaneously.
