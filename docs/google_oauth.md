# Google OAuth - run and test guide

Covers issue #8: Google Sign-In token verification and account linking on the server.

The mobile app uses the Google Sign-In SDK to obtain an ID token on-device and sends it to the server. The server verifies that token with Google's API, upserts the user, and returns a JWT pair. No browser redirect or server-side OAuth flow.

## Requirements

Only Docker is needed. No Go, no local database setup.

## Start the stack

```bash
docker compose up --build
```

## Set GOOGLE_CLIENT_ID

Open `server/.env` and fill in your Google OAuth client ID:

```
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
```

Get this from the Google Cloud Console under **APIs & Services > Credentials**. If this is left empty the server still runs but accepts tokens from any Google app (fine for local dev, never for production).

## Endpoints

### POST /api/v1/auth/google

Public. Receives an `id_token` from the mobile app. Verifies it with Google, creates or links the user, returns a JWT pair.

**Request body:**
```json
{
  "id_token": "<Google ID token from mobile SDK>"
}
```

**Response 200:**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "a3f2..."
}
```

**Account linking behaviour:**
- If the Google `sub` (unique Google user ID) is already in `user_providers` the existing user is returned directly.
- If the Google email matches an existing email/password account, the Google provider is linked to that account. No duplicate user is created.
- Otherwise a new user is created (pre-verified, no password) and the provider row is inserted.

### POST /api/v1/auth/link/google

JWT protected. Links a Google account to an already logged-in user.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request body:**
```json
{
  "id_token": "<Google ID token from mobile SDK>"
}
```

**Response 200:**
```json
{
  "message": "Google account linked successfully"
}
```

## Testing with curl

Since these endpoints require a real Google ID token produced by the mobile SDK, the error paths below can be tested immediately. The happy path requires a token from Akram's mobile integration.

### Invalid token returns 401

```bash
curl -X POST http://localhost:8081/api/v1/auth/google \
  -H "Content-Type: application/json" \
  -d '{"id_token":"this.is.not.valid"}'
```

Expected response `401`:
```json
{"error": "invalid or expired Google ID token"}
```

### Missing field returns 400

```bash
curl -X POST http://localhost:8081/api/v1/auth/google \
  -H "Content-Type: application/json" \
  -d '{}'
```

Expected response `400`:
```json
{"error": "id_token is required"}
```

### Link endpoint without JWT returns 401

```bash
curl -X POST http://localhost:8081/api/v1/auth/link/google \
  -H "Content-Type: application/json" \
  -d '{"id_token":"anything"}'
```

Expected response `401`:
```json
{"error": "Authorization header is required"}
```

### Happy path (requires mobile token)

Once the mobile app produces a real ID token, the full flow can be tested:

```bash
curl -X POST http://localhost:8081/api/v1/auth/google \
  -H "Content-Type: application/json" \
  -d '{"id_token":"PASTE-REAL-TOKEN-HERE"}'
```

Expected response `200` with `access_token` and `refresh_token`. The returned `access_token` can then be used on any JWT-protected endpoint exactly like a regular login token.

## Running the unit tests

All happy paths and edge cases are covered by unit tests using mocks - no real Google token needed:

```bash
docker compose run --rm server go test ./internal/auth/... -v
```

Tests covered:

| Test | What it checks |
|---|---|
| `TestGoogleSignIn_NewUser` | New user gets created and receives a JWT pair |
| `TestGoogleSignIn_ReturnsValidJWT` | Returned access token is valid and contains the correct email |
| `TestGoogleSignIn_ExistingProvider_ReusesUser` | Same Google user signs in twice, same user ID returned both times |
| `TestGoogleSignIn_AccountLinking_NoNewUser` | Google email matches existing account, no duplicate user created |
| `TestGoogleSignIn_InvalidToken_Returns401` | Invalid token returns 401 |
| `TestGoogleSignIn_MissingToken_Returns400` | Missing field returns 400 |
| `TestLinkGoogle_Success` | Logged-in user links their Google account |
| `TestLinkGoogle_AlreadyLinkedSameUser_Returns409` | Linking the same Google account twice returns 409 |
| `TestLinkGoogle_AlreadyLinkedOtherUser_Returns409` | Google account already linked to someone else returns 409 |
| `TestLinkGoogle_NoJWT_Returns401` | Link endpoint without JWT returns 401 |

## Error responses

| Status | Meaning |
|---|---|
| 400 | `id_token` field missing from request body |
| 401 | Token invalid, expired, wrong audience, or email not verified by Google |
| 409 | Google account already linked (to this user or another) |

## Database tables used

- `users` - OAuth users are inserted with an empty password hash and `is_verified = true`
- `user_providers` - one row per linked provider: `(user_id, provider='google', provider_id=<Google sub>)`. Unique on `(provider, provider_id)`.

## Stop the stack

```bash
docker compose down        # stop, keep data
docker compose down -v     # stop and wipe the database
```
