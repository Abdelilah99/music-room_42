# Rate limiting & input validation - run and test guide

Covers issue #17: global rate limiting, stricter auth rate limiting, and consistent input validation across all endpoints.

## What was built

Three files in `server/internal/middleware/`:

- **`ratelimit.go`** - `NewRateLimiter(rateStr)` returns a Gin middleware. Takes a rate string in ulule/limiter format (`"100-M"` = 100 per minute). Uses an in-memory store keyed by client IP. Panics on invalid format so misconfiguration is caught at startup, not at runtime.
- **`validate.go`** - `BindAndValidate(c, &req)` binds the JSON body and runs validation in one step. Returns `false` and writes a 400 response with a human-readable field list if validation fails. Also exposes `RegisterJSONTagNames()` which makes error messages use json tag names (`"email"`) instead of Go field names (`"Email"`).

**Rate limits (configurable via `.env`):**

| Variable | Default | Applied to |
|----------|---------|-----------|
| `RATE_LIMIT_GLOBAL` | `100-M` | Every route |
| `RATE_LIMIT_AUTH` | `10-M` | All `/api/v1/auth/*` routes (stricter, brute-force mitigation) |

The auth rate limit is cumulative across all auth endpoints per IP. Hitting `/auth/register` 8 times and `/auth/login` 3 times is 11 total, which exceeds the 10-M auth limit.

**Validation tags added to all request structs:**

| Struct | Field | Rule |
|--------|-------|------|
| `RegisterRequest` | email | `required,email` |
| `RegisterRequest` | password | `required,min=8` |
| `ResendVerificationRequest` | email | `required,email` |
| `ForgotPasswordRequest` | email | `required,email` |
| `ResetPasswordRequest` | password | `required,min=8` |
| `LoginRequest` | email | `required,email` |
| `SendRequest` (friends) | addressee_id | `required,uuid` |

## Requirements

Only Docker is needed.

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
go test ./internal/middleware/... -v
```

Expected output:

```
--- PASS: TestRateLimitReturns429WhenExceeded
--- PASS: TestRateLimitDifferentIPsAreIndependent
--- PASS: TestInvalidRatePanics
--- PASS: TestBindAndValidatePassesValidBody
--- PASS: TestBindAndValidateMissingRequiredField
--- PASS: TestBindAndValidateInvalidEmail
--- PASS: TestBindAndValidatePasswordTooShort
--- PASS: TestBindAndValidateMalformedJSON
--- PASS: TestBindAndValidateEmptyBody
```

## Test validation manually

### Missing required field

```bash
curl -s -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com"}'
```

Expected response (`400`):
```json
{
  "error": "validation failed",
  "fields": ["password is required"]
}
```

### Invalid email format

```bash
curl -s -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"notanemail","password":"password123"}'
```

Expected response (`400`):
```json
{
  "error": "validation failed",
  "fields": ["email must be a valid email address"]
}
```

### Password too short

```bash
curl -s -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"short"}'
```

Expected response (`400`):
```json
{
  "error": "validation failed",
  "fields": ["password must be at least 8 characters"]
}
```

## Test rate limiting manually

### Auth rate limit (default: 10 per minute)

Run this in a terminal - the 11th request should return 429:

```bash
for i in $(seq 1 12); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST http://localhost:8081/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"x@x.com","password":"wrong"}')
  echo "Request $i: HTTP $STATUS"
done
```

Expected: first 10 return 401, requests 11+ return 429.

### Global rate limit (default: 100 per minute)

```bash
for i in $(seq 1 105); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/health)
  [ "$STATUS" = "429" ] && echo "429 at request $i" && break
done
```

Expected: `429 at request 101`.

## Configure rate limits

Add to your `.env` file (format: `<count>-<S|M|H|D>`):

```env
RATE_LIMIT_GLOBAL=100-M
RATE_LIMIT_AUTH=10-M
```

To tighten for production or loosen for development, just change the values and restart the server. No code changes needed.

## How to use BindAndValidate in a new handler

```go
import "music-room/internal/middleware"

type CreateRoomRequest struct {
    Name string `json:"name" binding:"required,min=3,max=50"`
}

func (h *Handler) CreateRoom(c *gin.Context) {
    var req CreateRoomRequest
    if !middleware.BindAndValidate(c, &req) {
        return // 400 already written
    }
    // req.Name is guaranteed valid here
}
```

Common binding tags: `required`, `email`, `min=N`, `max=N`, `uuid`, `url`.
