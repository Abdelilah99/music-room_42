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

## Concurrency

The `PATCH position` endpoint runs inside a `SERIALIZABLE` transaction. Postgres will serialise concurrent moves on the same playlist and retry any transaction that would produce a non-serializable result. This guarantees the final position order is always consistent with no skipped or duplicate positions, regardless of how many clients reorder simultaneously.
