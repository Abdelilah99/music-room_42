# Swagger API documentation - run and test guide

Covers issue #21: auto-generated OpenAPI docs for all server endpoints.

## What was built

- `swaggo/swag` parses annotations on handler functions and generates `docs/swagger.json`, `docs/swagger.yaml`, and `docs/docs.go`
- `swaggo/gin-swagger` serves the Swagger UI at `GET /api/v1/docs/index.html`
- All 21 endpoints are annotated: auth, users, friends, music
- Protected routes show the `BearerAuth` security requirement
- `make docs` regenerates everything from scratch without any manual steps

## Requirements

Only Docker is needed to run the server. To regenerate docs you need Go installed locally.

## Start the stack

```bash
docker compose up --build
```

## View the Swagger UI

Open in your browser:

```
http://localhost:8081/api/v1/docs/index.html
```

You should see the interactive Swagger UI listing all endpoints grouped by tag:
- **auth** — register, verify-email, resend-verification, forgot-password, reset-password, login, refresh, logout, google sign-in, link google
- **users** — get my profile, update my profile, search users, get user profile
- **friends** — send request, accept, reject, unfriend, list friends, list incoming requests, list outgoing requests
- **music** — search tracks

## Regenerate docs after adding a new endpoint

```bash
cd server
make docs
```

This runs `swag init` against all handler files and overwrites `docs/`. Commit the updated `docs/` folder so the generated files stay in sync with the code.

## Authenticate in the Swagger UI

1. Click **Authorize** (top right)
2. Enter `Bearer <your_access_token>` in the `BearerAuth` field
3. Click **Authorize** — all subsequent requests will include the header

To get a token: use the `POST /auth/login` endpoint directly in the UI.

## How to annotate a new handler

Every new handler added in Phase 4-6 must be annotated before its branch is merged. Copy this template and fill in the fields:

```go
// MyHandler godoc
// @Summary      One-line description
// @Tags         tag-name
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body RequestStructName true "Description"
// @Param        id   path string            true "Resource UUID"
// @Success      200 {object} ResponseStructName
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /your/path [post]
func (h *Handler) MyHandler(c *gin.Context) {
```

Rules:
- `@Router` path must match the path registered in `setupRouter` in `cmd/main.go`, without the `/api/v1` prefix
- Use `{object}` for a single JSON object, `{array}` for a JSON array
- For response types defined in a different package, use the swag-mangled name (e.g. `music-room_internal_model.TrackDTO`). For types in the same package, just use the type name.
- Add `@Security BearerAuth` on every JWT-protected route
- After adding annotations run `make docs` and commit the updated `docs/` folder

## Run the unit tests

```bash
cd server
go test ./... -v
```

Expected: all tests pass across `auth`, `handler`, `hub`, and `middleware` packages.
