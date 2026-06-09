# Device Management

## Overview

Devices are physical units (phones, tablets) linked to a user account. Each device can
independently receive a delegation grant, allowing another user to control playback on
that device. A user can own multiple devices with separate delegation states.

## Endpoints

All endpoints require `Authorization: Bearer <JWT>`.

### Register a device

```
POST /api/v1/devices
Content-Type: application/json

{
  "name": "My Pixel",
  "platform": "android",
  "model": "Pixel 8"
}
```

Returns `201` with the created device. Returns `409` if a device with the same
`model` is already registered under this user account.

### List devices

```
GET /api/v1/devices
```

Returns `200` with the authenticated user's devices, each including the active
delegate (if one exists) or `null`.

```json
{
  "devices": [
    {
      "id": "...",
      "user_id": "...",
      "name": "My Pixel",
      "platform": "android",
      "model": "Pixel 8",
      "created_at": "...",
      "delegate": null
    },
    {
      "id": "...",
      "user_id": "...",
      "name": "My iPad",
      "platform": "ios",
      "model": "iPad Air",
      "created_at": "...",
      "delegate": {
        "user_id": "...",
        "email": "friend@example.com"
      }
    }
  ]
}
```

### Get a single device

```
GET /api/v1/devices/:id
```

Returns `200` with the device. Returns `404` if not found or not owned by the caller.

### Delete a device

```
DELETE /api/v1/devices/:id
```

Unregisters the device. Any active delegation on that device is also removed
(cascade delete). Returns `200` on success, `404` if not found or not owned.

## Error codes

| HTTP | Body `error` | Meaning |
|------|-------------|---------|
| 404 | `device not found` | Device does not exist or belongs to another user |
| 409 | `device already registered` | Same `model` already registered for this user |
| 401 | `unauthorized` | Missing or invalid JWT |

## Uniqueness constraint

Registering the same `model` value twice for the same user returns `409`. This
prevents duplicate entries for the same physical hardware. Different users can
register devices with the same model string independently.

## Test setup (live Docker)

All commands assume the server is running on port 8081.

```bash
# Register and login (see auth docs)
TOKEN=<your-jwt>

# Register a device
DEV=$(curl -s -X POST http://localhost:8081/api/v1/devices \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"My Pixel","platform":"android","model":"Pixel 8"}')
DEVICE_ID=$(echo $DEV | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])")

# Duplicate -> 409
curl -s -X POST http://localhost:8081/api/v1/devices \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Other","platform":"android","model":"Pixel 8"}'

# List devices
curl -s http://localhost:8081/api/v1/devices \
  -H "Authorization: Bearer $TOKEN"

# Get single device
curl -s "http://localhost:8081/api/v1/devices/$DEVICE_ID" \
  -H "Authorization: Bearer $TOKEN"

# Delete device (also removes active delegation)
curl -s -X DELETE "http://localhost:8081/api/v1/devices/$DEVICE_ID" \
  -H "Authorization: Bearer $TOKEN"
```
