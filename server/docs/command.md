# Playback Command Relay

## Overview

A device owner or their active delegate can send playback commands to a device.
The command is broadcast instantly over the device's WebSocket hub to the owner's
connected app, which executes the action on the local player.

## Endpoints

All endpoints require `Authorization: Bearer <JWT>` unless noted otherwise.

### Send a command

```
POST /api/v1/devices/:id/command
Content-Type: application/json

{
  "action": "play" | "pause" | "next" | "volume",
  "value": <integer 0-100>   // required only when action is "volume"
}
```

- Caller must be the device owner or the active delegate (`RequireDelegateOrOwner` middleware).
- `value` is required and must be 0-100 when `action` is `"volume"`; ignored for all other actions.
- Returns `200` on success.

### Connect to the device WebSocket

```
GET /api/v1/devices/:id/ws?token=<JWT>
```

- Only the device owner can connect. Non-owners receive `403`.
- Token must be passed as the `token` query parameter (browsers cannot set
  `Authorization` headers on WebSocket handshakes).
- The server keeps the connection alive with ping/pong frames.
- The hub is removed automatically when the owner disconnects.

## Broadcast message schema

Every accepted command is broadcast to the device's hub as:

```json
{
  "type":    "command",
  "action":  "play" | "pause" | "next" | "volume",
  "value":   null | 0-100,
  "sent_by": "<user_id>"
}
```

`value` is `null` for non-volume actions.

## Error codes

| HTTP | Body `error` | Meaning |
|------|-------------|---------|
| 400 | `action must be one of: play, pause, next, volume` | Unknown action |
| 400 | `volume requires a value between 0 and 100` | Missing or invalid volume |
| 403 | `NOT_AUTHORIZED` | Caller is neither the device owner nor the active delegate |
| 401 | `unauthorized` | Missing or invalid JWT |

## Live test example

```bash
TOKEN_OWNER=<jwt>
TOKEN_DELEGATE=<jwt>
DEVICE_ID=<uuid>

# Owner opens the device WebSocket (receives commands)
wscat -c "ws://localhost:8081/api/v1/devices/$DEVICE_ID/ws?token=$TOKEN_OWNER"

# Delegate sends a play command (in another terminal)
curl -s -X POST "http://localhost:8081/api/v1/devices/$DEVICE_ID/command" \
  -H "Authorization: Bearer $TOKEN_DELEGATE" \
  -H 'Content-Type: application/json' \
  -d '{"action":"play"}'

# Delegate sends a volume command
curl -s -X POST "http://localhost:8081/api/v1/devices/$DEVICE_ID/command" \
  -H "Authorization: Bearer $TOKEN_DELEGATE" \
  -H 'Content-Type: application/json' \
  -d '{"action":"volume","value":80}'
```
