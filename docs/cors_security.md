# CORS configuration & SECURITY.md - run and test guide

Covers issue #18: CORS origin whitelisting and security documentation.

## What was built

**`server/internal/middleware/cors.go`** - `NewCORS(allowedOriginsEnv)` returns a Gin middleware. Takes a comma-separated list of allowed origins from the `ALLOWED_ORIGINS` env var. Configures allowed methods (`GET POST PUT PATCH DELETE OPTIONS`) and headers (`Authorization`, `Content-Type`). When the env var is empty, no origins are whitelisted and all cross-origin requests are blocked by the browser.

**`SECURITY.md`** at the repo root - required by the project subject. Covers 8 attack vectors: JWT expiry, refresh token rotation, brute-force, CSRF, replay attacks, user enumeration, SQL injection, and data isolation. Each entry names the threat, the mitigation, and the exact file and function where it is enforced.

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
--- PASS: TestCORSAllowsWhitelistedOrigin
--- PASS: TestCORSBlocksUnlistedOrigin
--- PASS: TestCORSPreflightReturns204
--- PASS: TestCORSMultipleOriginsAllowed
--- PASS: TestCORSEmptyOriginsBlocksAll
```

## Configure allowed origins

Add to `server/.env`:

```env
ALLOWED_ORIGINS=https://your-frontend.com,https://your-admin.com
```

Multiple origins are comma-separated. Restart the server after changing this value. No code changes needed.

## Test CORS manually

### 1. Whitelisted origin receives correct headers

```bash
curl -s -D - \
  -H "Origin: https://your-frontend.com" \
  http://localhost:8081/health | grep -i access-control
```

Expected:
```
Access-Control-Allow-Origin: https://your-frontend.com
Access-Control-Expose-Headers: Content-Length
```

### 2. Unlisted origin gets no CORS header

```bash
curl -s -D - \
  -H "Origin: https://evil.com" \
  http://localhost:8081/health | grep -i access-control
```

Expected: no `Access-Control-Allow-Origin` header. The browser will block the response.

### 3. Preflight from whitelisted origin returns 204

```bash
curl -s -o /dev/null -w "HTTP %{http_code}\n" -X OPTIONS \
  -H "Origin: https://your-frontend.com" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Authorization,Content-Type" \
  http://localhost:8081/api/v1/auth/login
```

Expected: `HTTP 204`

### 4. Verify all methods and headers are exposed

```bash
curl -s -D - -X OPTIONS \
  -H "Origin: https://your-frontend.com" \
  -H "Access-Control-Request-Method: PATCH" \
  http://localhost:8081/api/v1/users/me | grep -i access-control
```

Expected headers to include:
```
Access-Control-Allow-Methods: GET,POST,PUT,PATCH,DELETE,OPTIONS
Access-Control-Allow-Headers: Authorization,Content-Type
```

## How to use NewCORS in a new service

The middleware is already wired globally in `cmd/main.go` via `r.Use(middleware.NewCORS(allowedOrigins))`. No action needed in individual handlers — it applies to all routes automatically.

If you need a different CORS policy for a specific route group, call `NewCORS` again with a different origins string:

```go
adminGroup := v1.Group("/admin")
adminGroup.Use(middleware.NewCORS(os.Getenv("ADMIN_ALLOWED_ORIGINS")))
```

## SECURITY.md

`SECURITY.md` lives at the repo root. It is required by the project subject and is evaluated during peer review. It covers:

1. JWT expiry
2. Refresh token rotation
3. Brute-force protection
4. CSRF mitigation
5. Replay attack prevention
6. User enumeration prevention
7. SQL injection prevention
8. Data isolation

Each entry includes the threat description, the mitigation used, and the exact file/function where it is enforced. Keep it up to date when adding new endpoints or changing auth logic.
