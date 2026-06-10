# Delegation Endpoints

## Overview

A device owner can grant control of one of their devices to exactly one friend at
a time. The delegate can then issue playback commands on behalf of the owner. Only
one active delegation exists per device at any time -- granting a second delegation
automatically revokes the first.

## Endpoints

All endpoints require `Authorization: Bearer <JWT>`.

### Grant delegation

```
POST /api/v1/devices/:id/delegate
Content-Type: application/json

{
  "friend_user_id": "<uuid>"
}
```

- Only the device owner can call this.
- The target user must be an accepted friend.
- If an active delegation already exists for this device, it is revoked first.
- Returns `200` on success.

### Revoke delegation

```
DELETE /api/v1/devices/:id/delegate
```

- Only the device owner can call this.
- Sets `revoked_at` on the active delegation row.
- Idempotent: returns `200` even if no active delegation exists.

### List devices delegated to me

```
GET /api/v1/devices/delegated
```

Returns all devices where the authenticated user currently holds an active
delegation, including the device owner info.

```json
{
  "devices": [
    {
      "id": "...",
      "user_id": "...",
      "name": "Living Room",
      "platform": "android",
      "model": "Shield TV",
      "created_at": "...",
      "owner": {
        "user_id": "...",
        "email": "owner@example.com"
      }
    }
  ]
}
```

## Error codes

| HTTP | Body `error` | Meaning |
|------|-------------|---------|
| 403 | `NOT_FRIENDS` | Target user is not an accepted friend of the device owner |
| 403 | `NOT_AUTHORIZED` | Caller is neither the device owner nor the active delegate |
| 404 | `device not found` | Device does not exist or belongs to another user |
| 401 | `unauthorized` | Missing or invalid JWT |

## Delegation enforcement middleware

The `RequireDelegateOrOwner` middleware (in `handler/delegation.go`) is ready
to be applied to any route that should allow both the device owner and the active
delegate. It will be wired to `POST /api/v1/devices/:id/command` in issue #35.

Apply it like this:

```go
devices.POST("/:id/command", delegHandler.RequireDelegateOrOwner(), commandHandler.Send)
```

## Auto-revoke on re-grant

Granting a second delegation does not return an error. The previous delegation
row gets `revoked_at = NOW()` in the same transaction before the new row is inserted.
This means there is always at most one active delegation per device.

## Test setup (live Docker)

```bash
TOKEN_OWNER=<jwt>
TOKEN_FRIEND=<jwt>
FRIEND_ID=<uuid>
DEVICE_ID=<uuid>

# Grant
curl -s -X POST "http://localhost:8081/api/v1/devices/$DEVICE_ID/delegate" \
  -H "Authorization: Bearer $TOKEN_OWNER" \
  -H 'Content-Type: application/json' \
  -d "{\"friend_user_id\":\"$FRIEND_ID\"}"

# List devices delegated to friend
curl -s "http://localhost:8081/api/v1/devices/delegated" \
  -H "Authorization: Bearer $TOKEN_FRIEND"

# Revoke
curl -s -X DELETE "http://localhost:8081/api/v1/devices/$DEVICE_ID/delegate" \
  -H "Authorization: Bearer $TOKEN_OWNER"
```
