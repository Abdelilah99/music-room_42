# WebSocket hub infrastructure - run and test guide

Covers issue #16: generic WebSocket hub used by Track Vote, Control Delegation, and Playlist Editor.

## What was built

Three files in `server/internal/hub/`:

- **`hub.go`** - `Hub` struct. Holds connected clients and routes messages through channels (register, unregister, broadcast). The clients map is only ever touched by one goroutine, so no mutex is needed on it. Also handles slow clients - if a client's buffer is full during a broadcast it is evicted immediately without affecting others.
- **`manager.go`** - `HubManager`. Maps an entity ID (string) to its `Hub`. Creates a hub on first connection, removes it automatically when the last client disconnects. Safe to call from multiple goroutines.
- **`handler.go`** - `ServeWS` function. Upgrades an HTTP connection to WebSocket and registers the client with the correct hub.

The hub is wired into the server in `cmd/main.go` and exposed on:

```
GET /api/v1/ws/:entityID   (JWT protected)
```

`:entityID` is whatever ID the calling service passes - an event ID, playlist ID, device ID, etc. All three real-time services will call `ServeWS` with their own ID and reuse the same hub code.

## Requirements

Only Docker is needed. No Go, no local setup.

## Start the stack

```bash
docker compose up --build
```

Server is on port **8081**.

## Run migrations

```bash
docker compose run --rm server go run ./cmd/migrate/main.go up
```

## Run the unit tests

```bash
cd server
go test ./internal/hub/... -v
```

Expected output:

```
--- PASS: TestBroadcastReachesAllClients
--- PASS: TestDisconnectDoesNotAffectOtherClients
--- PASS: TestHubCleansUpWhenEmpty
--- PASS: TestSlowClientEvictionCleansUpHub
--- PASS: TestClientsInDifferentHubsAreIsolated
```

What each test covers:

| Test | What it proves |
|------|---------------|
| `TestBroadcastReachesAllClients` | A message from one client reaches every client in the same room |
| `TestDisconnectDoesNotAffectOtherClients` | Closing one client does not crash the hub or cut off others |
| `TestHubCleansUpWhenEmpty` | HubManager removes the entry when the last client leaves - no memory leak |
| `TestSlowClientEvictionCleansUpHub` | Slow clients (full buffer) are evicted and the hub still cleans up correctly |
| `TestClientsInDifferentHubsAreIsolated` | Two different entity IDs get separate hubs - messages do not bleed between rooms |

## Test the live endpoint manually

### 1. Get a JWT

Register and verify a user first (see `auth.md`), then:

```bash
curl -s -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"YourPassword1!"}' | jq .access_token
```

Copy the token.

### 2. Install a WebSocket client

```bash
npm install -g wscat
```

### 3. Connect two terminals to the same room

**Terminal 1:**

```bash
wscat -c "ws://localhost:8081/api/v1/ws/my-room-123" \
  -H "Authorization: Bearer <your_token>"
```

**Terminal 2:**

```bash
wscat -c "ws://localhost:8081/api/v1/ws/my-room-123" \
  -H "Authorization: Bearer <your_token>"
```

Both are now in room `my-room-123`.

### 4. Broadcast

Type anything in Terminal 1 and press Enter. You should see the message appear in **both** terminals - that is the hub broadcasting to all connected clients.

### 5. Test isolation between rooms

Open a third terminal connected to a **different** room ID:

```bash
wscat -c "ws://localhost:8081/api/v1/ws/other-room" \
  -H "Authorization: Bearer <your_token>"
```

Messages sent in `my-room-123` must **not** appear here.

### 6. Test disconnect recovery

Close Terminal 1 (Ctrl+C). Send a message from Terminal 2. It should still work - the hub keeps running as long as at least one client is connected.

### 7. Test unauthorized access

```bash
wscat -c "ws://localhost:8081/api/v1/ws/my-room-123"
```

No Authorization header - server should reject the upgrade with `401`.

## How to reuse the hub in a new service

Import the package and call `ServeWS` from your Gin handler:

```go
import "music-room/internal/hub"

// In your handler (hubManager comes from main.go via dependency injection):
func (h *YourHandler) JoinRoom(c *gin.Context) {
    roomID := c.Param("id")   // or whatever your route param is
    hub.ServeWS(h.hubManager, roomID, c)
}
```

Register the route in `setupRouter` (JWT protected):

```go
yourGroup.GET("/:id/ws", yourHandler.JoinRoom)
```

That is all. The hub creation, cleanup, and broadcast are handled automatically.
