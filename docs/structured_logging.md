# Structured logging - run and test guide

Covers issue #19: JSON structured logging with mobile metadata headers using Go's built-in `log/slog`.

## What was built

- **`server/internal/middleware/logger.go`** - Gin middleware that emits one JSON log line per HTTP request. Captures method, path, status, latency, user ID, and the three mobile headers. Log level is set automatically: INFO for 2xx/3xx, WARN for 4xx, ERROR for 5xx.
- **`server/cmd/main.go`** - slog JSON handler set up at startup. All startup, DB connection, and shutdown events are logged at INFO/ERROR through slog. Gin's default logger is replaced so there is only one log line per request.
- **`server/internal/hub/handler.go`** - WebSocket upgrade errors now go through slog at ERROR level.
- **`server/cmd/migrate/main.go`** - Migration tool logs through slog.
- **`server/cmd/seed/main.go`** - Seed tool logs through slog.

No external library added. `log/slog` is part of Go's standard library since Go 1.21.

## What a log line looks like

Every request produces one line on stdout:

```json
{"time":"2026-06-04T17:21:57Z","level":"WARN","msg":"request","method":"POST","path":"/api/v1/auth/login","status":401,"latency":59852333,"user_id":"","platform":"android","device_model":"Pixel 8","app_version":"1.0.0"}
```

Fields:

| Field | Description |
|-------|-------------|
| `time` | UTC timestamp |
| `level` | INFO / WARN / ERROR |
| `msg` | Always `"request"` for HTTP log lines |
| `method` | HTTP method |
| `path` | Matched route pattern (e.g. `/api/v1/users/:id`) |
| `status` | HTTP response status code |
| `latency` | Request duration in nanoseconds |
| `user_id` | JWT user ID if the request was authenticated, empty string otherwise |
| `platform` | Value of `X-Platform` header (e.g. `android`) |
| `device_model` | Value of `X-Device-Model` header (e.g. `Pixel 8`) |
| `app_version` | Value of `X-App-Version` header (e.g. `1.0.3`) |

If the mobile headers are absent the fields are logged as empty strings. No error is thrown.

## Mobile client headers

The mobile app must send these three headers on every request so the server can log them:

```
X-Platform: android
X-Device-Model: Pixel 8
X-App-Version: 1.0.3
```

These are read from `DeviceInfoService` (issue #22) and should be added to the Dio interceptor in the mobile app.

## Requirements

Only Docker is needed.

## Start the stack

```bash
docker compose up --build
```

## Watch logs in real time

```bash
docker logs -f music-room_42-server-1
```

Output will be one JSON object per line. Use `jq` to pretty-print:

```bash
docker logs -f music-room_42-server-1 | jq .
```

Filter by level:

```bash
docker logs -f music-room_42-server-1 | jq 'select(.level == "WARN" or .level == "ERROR")'
```

## Run the unit tests

```bash
cd server
go test ./internal/middleware/... -v -run TestLogger
```

Expected output:

```
--- PASS: TestLogger_InfoOn2xx
--- PASS: TestLogger_WarnOn4xx
--- PASS: TestLogger_ErrorOn5xx
--- PASS: TestLogger_MobileHeaders
--- PASS: TestLogger_MissingHeadersAreEmpty
--- PASS: TestLogger_UserIDLogged
```

What each test covers:

| Test | What it proves |
|------|----------------|
| `TestLogger_InfoOn2xx` | 2xx responses are logged at INFO |
| `TestLogger_WarnOn4xx` | 4xx responses are logged at WARN |
| `TestLogger_ErrorOn5xx` | 5xx responses are logged at ERROR |
| `TestLogger_MobileHeaders` | X-Platform, X-Device-Model, X-App-Version are captured correctly |
| `TestLogger_MissingHeadersAreEmpty` | Missing mobile headers produce empty strings, not errors |
| `TestLogger_UserIDLogged` | Authenticated user ID is captured from the Gin context |

## How to add logging to a new handler

You do not need to add anything to individual handlers. The middleware runs globally for every route and logs automatically.

For one-off events outside of request handling (e.g. a background job, a startup check), use slog directly:

```go
import "log/slog"

slog.Info("job started", "job", "cleanup", "items", 42)
slog.Error("job failed", "error", err)
```

Never use `fmt.Println`, `fmt.Printf`, or the standard `log` package. All output must go through `slog` so logs stay structured and parseable.
