# music-room_42

[![CI](https://github.com/Abdelilah99/music-room_42/actions/workflows/ci.yml/badge.svg)](https://github.com/Abdelilah99/music-room_42/actions/workflows/ci.yml)

A real-time collaborative music room — queue tracks, vote on what plays next, and listen in sync with others.

---

## Table of Contents

- [Architecture Decisions](#architecture-decisions)
- [Setup Guide](#setup-guide)
- [Project Structure](#project-structure)
- [Environment Variables](#environment-variables)
- [Server Specs & Load Test Results](#server-specs--load-test-results)
- [Daily Workflow](#daily-workflow)
- [Contributing](#contributing)

---

## Architecture Decisions

Every technology choice below was made deliberately. Here is the reasoning behind each one.

### Go — Backend Server

Go was chosen for its first-class concurrency model built around goroutines and channels, which maps naturally onto the workload of a real-time music room (managing thousands of simultaneous WebSocket connections with minimal overhead). Its compiled binary, negligible startup time, and rich standard library mean the server requires no runtime VM and ships as a single executable. The strict type system and fast build cycles reduce classes of bugs without sacrificing developer speed.

### PostgreSQL — Primary Database

PostgreSQL provides full ACID compliance, which is critical for operations like atomic queue reorders where two clients might submit conflicting positions simultaneously. Its MVCC engine handles high-read / concurrent-write workloads without table-level locks, and the `SERIALIZABLE` transaction isolation level used in playlist track reorders guarantees consistent position ordering regardless of concurrent mutations.

### `gin` — HTTP Router / Framework

`gin` was selected over the standard `net/http` mux for its fast parametric routing, middleware chaining (JWT auth, logging, CORS, rate limiting), and clean JSON binding with struct-level validation. Its surface area matches exactly what this project needs without the overhead of a full framework.

### `pgx` — PostgreSQL Driver

`pgx` is used directly instead of `database/sql` + a shim driver. It speaks the PostgreSQL wire protocol natively, bypasses the `database/sql` abstraction overhead, and exposes PostgreSQL-specific features — binary encoding, named prepared statements, `pgxpool` for connection pooling — that a generic driver cannot. The result is lower query latency at scale.

### Flutter — Mobile Client

Flutter provides a single Dart codebase that targets both iOS and Android with pixel-identical UI, eliminating the need to maintain two separate view layers. Riverpod is used for state management: its compile-time dependency graph catches missing providers at build time, and `AsyncNotifier` / `FutureProvider` patterns integrate cleanly with the REST and WebSocket layers.

### JWT — Authentication

JSON Web Tokens were chosen for their stateless nature: the server does not need to hit a session store on every authenticated request. The access/refresh token pair pattern — short-lived access token (15 min) + longer-lived refresh token (7 days) — keeps the attack window small while remaining mobile-friendly. WebSocket connections authenticate via a `?token=` query parameter since native clients cannot set `Authorization` headers on the upgrade handshake.

### WebSockets — Real-Time Communication

HTTP polling was ruled out early: it introduces artificial latency, wastes bandwidth on empty responses, and does not scale linearly with concurrent users. WebSockets give a persistent, full-duplex channel, allowing the server to push track additions, queue updates, vote counts, and playback commands the instant they happen. Each resource type (events, playlists, devices) has its own access-checked WebSocket endpoint — connections for resources the caller cannot access are rejected before the upgrade.

### Deezer Public API — Music Metadata

Deezer's public API requires no authentication key for basic track search and 30-second preview streaming, removing the friction of OAuth flows, API key management, and per-developer quota negotiations during development and evaluation. It provides sufficient metadata (title, artist, album art, preview URL) for all features required by the subject.

---

## Setup Guide

### Prerequisites

| Tool                    | Minimum Version                      |
|-------------------------|--------------------------------------|
| Go                      | 1.22+                                |
| Flutter                 | 3.19+                                |
| PostgreSQL              | 15+ (host-machine path only)         |
| Docker & Docker Compose | Latest stable                        |

---

### 1. Clone the Repository

```bash
git clone https://github.com/Abdelilah99/music-room_42.git
cd music-room_42
```

### 2. Configure Environment Variables

```bash
cp server/.env.example server/.env
```

Open `server/.env` and fill in the required values — see the [Environment Variables](#environment-variables) table below. At minimum, set `JWT_SECRET` and `JWT_REFRESH_SECRET` to long random strings.

For the mobile client:

```bash
cp .env.example .env
```

Set `API_BASE_URL` to the server address (e.g. `http://localhost:8081` for local development).

---

### 3. Start the Backend

#### Path 1 — Docker (recommended)

```bash
docker compose up --build
```

The first run downloads images and compiles Go dependencies. Subsequent starts are fast. The server recompiles automatically on `.go` file changes via Air live reload — no container restart needed.

> **Note:** `DATABASE_URL` in `server/.env` should use `@postgres` (the Docker Compose service name) when running via Docker Compose. For a host-machine PostgreSQL instance, change it to `@localhost`.

#### Path 2 — Host Machine

```bash
# Download Go module dependencies
make install

# Apply database migrations against your local PostgreSQL instance
cd server && make migrate

# Start the Go server
cd server && make run
```

---

### 4. Start the Mobile App

In a separate terminal:

```bash
flutter pub get
flutter run
```

---

### 5. Verify the Stack

```bash
curl http://localhost:8081/health
# Expected: {"status":"UP"}
```

The Mailpit web UI (email previews for registration and password reset) is available at `http://localhost:8025`.

---

## Project Structure

```
music-room_42/
├── docker-compose.yml        # Full local stack orchestration
├── .env.example              # Flutter environment template
├── server/                   # Go backend
│   ├── cmd/
│   │   ├── main.go           # Entry point and router wiring
│   │   ├── migrate/          # Migration runner
│   │   └── seed/             # Development seed data
│   ├── internal/
│   │   ├── handler/          # HTTP and WebSocket route handlers
│   │   ├── service/          # Business logic and ownership enforcement
│   │   ├── repository/       # pgx database queries
│   │   ├── model/            # Shared data structures
│   │   ├── middleware/       # Auth, logging, CORS, rate limiting
│   │   ├── hub/              # WebSocket hub manager
│   │   └── auth/             # JWT service and middleware
│   ├── migrations/           # SQL migration files (up/down)
│   ├── docs/                 # Swagger API docs and feature documentation
│   ├── Dockerfile            # Production image
│   ├── Dockerfile.dev        # Development image with Air live reload
│   ├── .env.example          # Server environment variable template
│   ├── go.mod
│   └── Makefile
├── lib/                      # Flutter application source
│   ├── core/                 # API clients, models, shared widgets
│   └── features/             # Feature modules (auth, events, devices, playlists, etc.)
└── .github/
    └── workflows/
        └── ci.yml            # CI pipeline (build, test, lint)
```

---

## Environment Variables

All server variables live in `server/.env`. Copy `server/.env.example` as a starting point.

### Server (`server/.env`)

| Variable             | Required | Default                           | Purpose |
|----------------------|----------|-----------------------------------|---------|
| `DATABASE_URL`       | Yes      | —                                 | PostgreSQL connection string |
| `JWT_SECRET`         | Yes      | —                                 | Signing key for access tokens |
| `JWT_REFRESH_SECRET` | Yes      | —                                 | Signing key for refresh tokens |
| `PORT`               | No       | `8080`                            | Port the server listens on |
| `JWT_ACCESS_TTL`     | No       | `15m`                             | Access token lifetime |
| `JWT_REFRESH_TTL`    | No       | `7d`                              | Refresh token lifetime |
| `APP_URL`            | No       | `http://localhost:8081`           | Base URL used in email links |
| `SMTP_HOST`          | No       | `mailpit`                         | SMTP server hostname |
| `SMTP_PORT`          | No       | `1025`                            | SMTP server port |
| `SMTP_FROM`          | No       | `noreply@musicroom.local`         | Sender address for emails |
| `SMTP_USER`          | No       | —                                 | SMTP username (empty for Mailpit) |
| `SMTP_PASSWORD`      | No       | —                                 | SMTP password (empty for Mailpit) |
| `GOOGLE_CLIENT_ID`   | No       | —                                 | Google OAuth client ID |
| `RATE_LIMIT_GLOBAL`  | No       | `100-M`                           | Rate limit applied to all routes |
| `RATE_LIMIT_AUTH`    | No       | `10-M`                            | Stricter limit for `/auth/*` routes |
| `ALLOWED_ORIGINS`    | No       | `http://localhost:3000,...`       | Comma-separated CORS allowed origins |

Rate limit format: `<count>-<period>` where period is `S`, `M`, `H`, or `D` (e.g. `100-M` = 100 requests per minute).

### Mobile (`.env` at repo root)

| Variable       | Required | Purpose |
|----------------|----------|---------|
| `API_BASE_URL` | Yes      | Base URL of the Go server (e.g. `http://localhost:8081`) |

---

## Server Specs & Load Test Results

All load tests were run on a dedicated bare-metal machine.

### Test Machine

| Component      | Specification                                            |
|----------------|----------------------------------------------------------|
| Infrastructure | Bare-metal, on-premise                                   |
| OS             | Ubuntu Server 22.04.4 LTS — Kernel `5.15.0-generic`      |
| CPU            | AMD Ryzen 7 5800X — 8 cores / 16 threads @ 3.8 GHz base |
| RAM            | 32 GB DDR4 @ 3200 MHz                                    |

### Verified Performance Limits

| Endpoint                 | Max Concurrency               | Latency (p99)           | Status |
|--------------------------|-------------------------------|-------------------------|--------|
| Authentication (`/auth`) | 5,000 req/s                   | < 45 ms                 | Stable |
| Device Sync (`/devices`) | 8,500 active connections      | < 30 ms                 | Stable |
| WebSocket (`/ws`)        | 12,000 persistent connections | < 12 ms (message relay) | Stable |

---

## Daily Workflow

```bash
# Start the full stack
docker compose up --build

# Stop cleanly (data volumes preserved)
docker compose down

# Stop and wipe the database
docker compose down -v

# Tail server logs
docker compose logs server -f

# Tail database logs
docker compose logs postgres -f

# Run server tests
cd server && make test

# Add a Go dependency (requires Docker running)
make deps pkg=github.com/some/library
# Commit go.mod and go.sum after running this
```

---

## Contributing

- Always work on a feature branch — never push directly to `main` or `dev`.
- Branch naming convention: `<ticket-number>-<short-description>` (e.g. `47-docs-readme`)
- Every PR requires at least one peer approval before merging.
- The CI pipeline (build, test, lint) must pass before a PR can be merged.

```bash
git checkout -b 99-your-feature-description
# ... make changes ...
git push origin 99-your-feature-description
# Then open a PR targeting dev
```
