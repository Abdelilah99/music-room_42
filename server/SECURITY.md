# Server Data Isolation & Ownership Audit

This document records the findings of the ticket-45 ownership audit across all server resources.
Every handler file was reviewed against three questions:

1. Is the JWT `user_id` (callerID) extracted and passed to every service call?
2. Does the service enforce ownership / access before mutating or returning data?
3. Are WebSocket upgrade endpoints gated by the same access checks as their REST equivalents?

---

## Auth middleware coverage

All routes under `/api/v1` (except the public groups below) are protected by
`jwtMiddleware.Authenticate()` or `jwtMiddleware.AuthenticateWS()` applied at
the route-group level in `cmd/main.go`.

Public routes (no auth by design):

| Path | Reason |
|------|--------|
| `GET /health` | Infrastructure probe |
| `GET /api/v1/docs/*` | Swagger UI |
| `POST /api/v1/auth/register` | Onboarding |
| `GET /api/v1/auth/verify-email` | Email confirmation link |
| `POST /api/v1/auth/resend-verification` | Email confirmation |
| `POST /api/v1/auth/forgot-password` | Password reset |
| `POST /api/v1/auth/reset-password` | Password reset |
| `POST /api/v1/auth/login` | Token issuance |
| `POST /api/v1/auth/refresh` | Token rotation |
| `POST /api/v1/auth/logout` | Token revocation |
| `POST /api/v1/auth/google` | OAuth sign-in |

---

## Resource-by-resource findings

### Users / profiles

| Operation | Handler | Service enforcement |
|-----------|---------|---------------------|
| `GET /users/me` | `profile.go` - callerID from JWT | `GetMyProfile` scoped to callerID only |
| `PATCH /users/me` | callerID from JWT | `UpdateMyProfile` scoped to callerID only |
| `GET /users/search` | callerID from JWT | Returns only public-visibility fields |
| `GET /users/:id` | callerID + targetID both passed | `GetUserProfile` applies visibility tiers |

Result: no cross-user data exposure.

---

### Friends

| Operation | Ownership check |
|-----------|----------------|
| `POST /friends/request` | service validates `caller != addressee` |
| `POST /friends/accept/:id` | service: caller must be the addressee of the request (`ErrNotAddresseeOp` -> 403) |
| `DELETE /friends/reject/:id` | service: caller must be a participant (`ErrNotParticipantOp` -> 403) |
| `DELETE /friends/:id` | service: caller must be a participant |
| `GET /friends` / `/requests` / `/outgoing` | scoped to callerID in the query |

Result: no cross-user mutation possible.

---

### Events

| Operation | Ownership check |
|-----------|----------------|
| `POST /events` | callerID set as owner |
| `GET /events` | `List` returns only events accessible to callerID |
| `GET /events/:id` | `GetAccessible` - 404 for inaccessible events |
| `PUT /events/:id` | `GetByIDForOwner` - 404 if caller is not the owner |
| `DELETE /events/:id` | `GetByIDForOwner` |
| `POST /events/:id/invites` | `GetByIDForOwner` |
| `GET /events/:id/ws` | `Get` (GetAccessible) before upgrade - see note below |

Note: the `/events/:id/ws` WebSocket endpoint was added in this ticket to replace
the former generic `/ws/:entityID` hub. The old endpoint accepted any authenticated
user to any hub with no access check, which meant a non-invited user of a
license-1 (invite-only) event could subscribe to its live queue. The generic
endpoint has been removed; all event WS connections now go through
`EventHandler.ServeWS`, which calls `eventSvc.Get` before upgrading.

---

### Tracks (event queue)

| Operation | Ownership check |
|-----------|----------------|
| `POST /events/:id/tracks` | `GetAccessible` (any participant can suggest) |
| `GET /events/:id/queue` | `GetAccessible` |
| `POST /events/:id/tracks/:trackId/vote` | `GetAccessible` + `enforceLicense` (invite-only or geo-fence) |
| `DELETE /events/:id/tracks/:trackId` | `GetByIDForOwner` (only event owner can delete) |

Result: license enforcement at the service layer; callerID passed to every call.

---

### Devices

| Operation | Ownership check |
|-----------|----------------|
| `POST /devices` | callerID set as owner |
| `GET /devices` | `List` scoped to callerID |
| `GET /devices/:id` | `Get(deviceID, callerID)` |
| `DELETE /devices/:id` | `Delete(deviceID, callerID)` |
| `POST /devices/:id/command` | `RequireDelegateOrOwner()` middleware applied before handler |
| `GET /devices/:id/ws` | `deviceSvc.Get` before WS upgrade |

Result: all device operations scoped to owner or explicit delegate.

---

### Delegation

| Operation | Ownership check |
|-----------|----------------|
| `POST /devices/:id/delegate` | `Grant(deviceID, callerID, friendID)` - service checks device ownership |
| `DELETE /devices/:id/delegate` | `Revoke(deviceID, callerID)` - service checks device ownership |
| `GET /devices/delegated` | `ListDelegated(callerID)` - returns only devices delegated to caller |

Result: delegation can only be granted/revoked by the device owner.

---

### Playlists

| Operation | Ownership check |
|-----------|----------------|
| `POST /playlists` | callerID set as owner |
| `GET /playlists` | `List` scoped to callerID (owned + invited) |
| `GET /playlists/:id` | `GetAccessible` - 404 for inaccessible |
| `PUT /playlists/:id` | `requireOwner` in service |
| `DELETE /playlists/:id` | `requireOwner` |
| `POST /playlists/:id/invites` | `requireOwner` |
| `POST /playlists/:id/tracks` | `requireEditAccess` (owner or invited on license-1) |
| `DELETE /playlists/:id/tracks/:trackId` | `requireEditAccess` |
| `PATCH /playlists/:id/tracks/:trackId/position` | `requireEditAccess` |
| `GET /playlists/:id/ws` | `Get` (GetAccessible) before WS upgrade |

Note: `GET /playlists/:id` and `GET /playlists/:id/ws` both return 404 for
playlists the caller cannot access - existence is not leaked.

Result: no cross-owner mutation possible; license enforcement correct.

---

## Issues fixed in this audit

| Severity | Issue | Fix |
|----------|-------|-----|
| High | Generic `GET /ws/:entityID` hub accepted any authenticated user to any entity hub without access checks. A non-invited user could subscribe to a license-1 event's live queue. | Removed the generic endpoint. Added `GET /events/:id/ws` on `EventHandler.ServeWS`, which calls `eventSvc.Get` (enforces `GetAccessible`) before upgrading the connection. |

---

## No issues found

- Profile endpoints: correctly scoped
- Friend mutation endpoints: all enforce participant/addressee constraints
- Event REST mutations: all use `GetByIDForOwner`
- Track mutations: license enforced at service layer
- Device endpoints: all scoped to callerID or authorized delegate
- Playlist mutations: `requireOwner` / `requireEditAccess` enforced consistently
- Device WS: ownership check before upgrade
- Playlist WS: access check before upgrade
