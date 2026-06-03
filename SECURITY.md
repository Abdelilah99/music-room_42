# Security

This document covers the threat model for the Music Room API and explains how each attack vector is mitigated. It is required by the project subject and is reviewed during peer evaluation.

---

## 1. JWT expiry — session theft window

**Threat:** A stolen access token can be reused indefinitely by an attacker.

**Mitigation:** Access tokens are short-lived (default 15 minutes, configurable via `JWT_ACCESS_TTL`). Even if an access token is intercepted, it becomes useless after expiry without the refresh token.

**Enforced in:** `server/internal/auth/jwt.go` — `GenerateAccessToken` sets `exp` claim. `server/internal/auth/middleware.go` — `ValidateAccessToken` rejects expired tokens with 401.

---

## 2. Refresh token rotation — stolen refresh token invalidation

**Threat:** A stolen refresh token can be reused to generate new access tokens indefinitely.

**Mitigation:** Every use of a refresh token issues a new one and immediately revokes the old one (rotation). If a previously-rotated token is presented again (reuse detected), **all** tokens for that user are revoked, forcing a full re-login.

**Enforced in:** `server/internal/auth/handler.go` — `Refresh` revokes the presented token and detects reuse via `RevokeAllForUser`. `server/internal/repository/token.go` — `Revoke` and `RevokeAllForUser` persist revocations.

---

## 3. Brute-force — credential stuffing mitigation

**Threat:** An attacker submits thousands of login attempts to guess credentials.

**Mitigation:** All `/api/v1/auth/*` routes are rate-limited (default 10 requests per minute per IP, configurable via `RATE_LIMIT_AUTH`). A global rate limit also applies to all other routes (`RATE_LIMIT_GLOBAL`, default 100/min). Exceeding either limit returns `429 Too Many Requests`.

**Enforced in:** `server/cmd/main.go` — `authGroup.Use(middleware.NewRateLimiter(authLimitRate))`. `server/internal/middleware/ratelimit.go` — in-memory store keyed by client IP.

> Note: rate limiting middleware is implemented in issue #17. Until that branch is merged, this is enforced at the infrastructure level (reverse proxy / load balancer).

---

## 4. CSRF — Cross-Site Request Forgery

**Threat:** A malicious site tricks an authenticated user's browser into making requests to the API using the user's session.

**Mitigation:** The API authenticates exclusively via the `Authorization: Bearer <token>` header. Browsers never automatically attach the `Authorization` header to cross-site requests (unlike cookies), so CSRF attacks cannot forge authenticated requests. Additionally, CORS is configured to reject requests from unlisted origins (`ALLOWED_ORIGINS`), providing a second layer.

**Enforced in:** `server/internal/auth/middleware.go` — all protected routes require `Authorization` header. `server/internal/middleware/cors.go` — `AllowCredentials: false`, origins whitelist.

---

## 5. Replay attacks — token expiry and one-time use

**Threat:** An attacker captures a valid token and reuses it after the legitimate session ends.

**Mitigation:** Access tokens expire (15 min TTL). Refresh tokens are one-time-use — each rotation revokes the previous token. A captured refresh token that has already been rotated is therefore useless; its hash is marked revoked in the database.

**Enforced in:** `server/internal/auth/jwt.go` — `exp` claim on access tokens. `server/internal/repository/token.go` — `GetByHash` checks `revoked_at IS NULL` before accepting a refresh token.

---

## 6. User enumeration — uniform error messages on login

**Threat:** Differences in error responses between "wrong email" and "wrong password" let attackers discover which emails are registered.

**Mitigation:** The login endpoint returns the same error message (`"Invalid email or password"`) regardless of whether the email exists or the password is wrong. The response time is also consistent because bcrypt comparison runs even when the user is not found (no short-circuit).

**Enforced in:** `server/internal/auth/handler.go` — `Login` returns identical 401 for both failure cases.

---

## 7. SQL injection — parameterized queries

**Threat:** Malicious input in fields like email or display name is interpreted as SQL, allowing data extraction or destruction.

**Mitigation:** Every database query uses `pgx` with numbered parameter placeholders (`$1`, `$2`, ...). User input is never concatenated into query strings. Input validation (`binding:"required,email"` etc.) also prevents the most obviously malformed inputs from reaching the database layer at all.

**Enforced in:** All files under `server/internal/repository/` — every `pool.QueryRow`, `pool.Exec`, and `pool.Query` call uses placeholder arguments. `server/internal/middleware/validate.go` — `BindAndValidate` rejects malformed input before it reaches any handler logic.

---

## 8. Data isolation — owner verification on every handler

**Threat:** An authenticated user accesses or modifies another user's private data by guessing resource IDs.

**Mitigation:** Every handler that operates on user-owned data reads the authenticated user's ID from the JWT claims set by the middleware (`c.Get("user_id")`) and checks it against the resource owner before proceeding. Friendship operations verify the caller is a participant. Profile writes are scoped to the caller's own record.

**Enforced in:** `server/internal/auth/middleware.go` — sets `user_id` from verified JWT. `server/internal/handler/profile.go` — `UpdateMyProfile` uses `myID` from context, never from the request body. `server/internal/service/friend.go` — `Unfriend`, `AcceptRequest`, `RejectRequest` all verify `callerID` against the friendship record before mutating state.

---

## Reporting a vulnerability

If you discover a security issue in this project, open a private GitHub issue or contact the repository owner directly. Do not open a public issue for unpatched vulnerabilities.
