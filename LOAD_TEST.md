# Load Test Results

## Tool

[k6](https://k6.io) v0.57.0

## Test environment

These tests were run on a development machine, not the production target. Results should be treated as conservative lower bounds.

| Component | Spec |
|-----------|------|
| CPU | AMD Ryzen 7 5800X, 8 cores / 16 threads @ 3.8 GHz |
| RAM | 32 GB DDR4 @ 3200 MHz |
| OS | Ubuntu Server 22.04.4 LTS |
| Server | Go binary built locally, pointing to Docker Postgres on port 5437 |
| Rate limits | `RATE_LIMIT_GLOBAL=10000-S`, `RATE_LIMIT_AUTH=1000-S` (overridden via `docker-compose.loadtest.yml`) |

## Scripts

| Script | What it tests |
|--------|---------------|
| `load_tests/track_vote.js` | Concurrent voting on event tracks (unique per user/track constraint) |
| `load_tests/delegation.js` | Concurrent playback command relay (play/pause via device) |
| `load_tests/playlist_editor.js` | Concurrent track add + reorder in a shared playlist |

## Results

### 1. Track vote (`track_vote.js`)

Ramp: 50 -> 100 -> 150 VUs over 3m30s

Each VU authenticates as a distinct user (`loadtest1..150@musicroom.test`) so votes
actually INSERT into the database and contend on the unique(user, track) constraint
rather than bouncing off it on every request. 409 responses only appear after a VU
has exhausted all 50 tracks in the pool (one vote per user per track per run).

| Metric | Value |
|--------|-------|
| Total iterations | 50,051 |
| Throughput | ~238 req/s |
| http_req_duration avg | 353 ms |
| http_req_duration p50 | 387 ms |
| http_req_duration p90 | 566 ms |
| http_req_duration p95 | 606 ms |
| Check pass rate | 100% (200 OK or 409 Conflict) |

**Threshold result:** FAILED -- p95 (606 ms) exceeded the 500 ms target.

**Breaking point:** ~100 VUs. At 100+ concurrent voters the p95 response time crosses 500 ms.

---

### 2. Command delegation (`delegation.js`)

Ramp: 50 -> 100 -> 200 VUs over 3m30s

| Metric | Value |
|--------|-------|
| Total iterations | 216,076 |
| Throughput | ~1,028 req/s |
| http_req_duration avg | 64 ms |
| http_req_duration p50 | 56 ms |
| http_req_duration p90 | 133 ms |
| http_req_duration p95 | 152 ms |
| http_req_failed | 0.00% |

**Threshold result:** PASSED -- both thresholds met at peak 200 VUs.

**Breaking point:** Not reached in this test. The delegation endpoint is a stateless relay with no DB writes under concurrent load, which explains the high throughput and low latency.

---

### 3. Playlist editor (`playlist_editor.js`)

Ramp: 50 -> 100 -> 150 VUs over 3m30s (70% add track, 30% move track)

| Metric | Value |
|--------|-------|
| Total iterations | 47,691 |
| Throughput | ~227 req/s |
| http_req_duration avg | 374 ms |
| http_req_duration p50 | 327 ms |
| http_req_duration p90 | 740 ms |
| http_req_duration p95 | 783 ms |
| http_req_failed | 10.40% (4,964 failures) |
| Add track check pass rate | 100% |
| Move track check pass rate | ~64% |

**Threshold result:** FAILED -- both thresholds exceeded (error rate 10.4% > 1%, p95 783 ms > 500 ms).

**Breaking point:** ~150 VUs. Failures are concentrated in the move-track operation (PATCH `/position`). Two factors contribute: (1) the test requests positions 1-10 regardless of current track count, which can exceed the count early in the ramp before enough tracks have been added; (2) serializable transaction conflicts under concurrent writes cause aborts that are not retried at the service layer (see issue #139). Add-track operations remained error-free throughout.

---

## Running the tests

Install k6 if not already present:

```bash
# Linux (x86_64)
wget -q https://github.com/grafana/k6/releases/download/v0.57.0/k6-v0.57.0-linux-amd64.tar.gz \
  -O - | tar -xz --strip-components=1 -C ~/.local/bin k6-v0.57.0-linux-amd64/k6
```

Then run everything:

```bash
# Start the stack with elevated rate limits and seed all test users
make load-test
```

The `load-test` target:
1. Starts the server with `docker-compose.loadtest.yml` overrides
2. Seeds `loadtest@musicroom.test` and `loadtest1..150@musicroom.test` into the DB (all with password `loadtest123`)
3. Runs all three k6 scripts sequentially

To run a single script manually:

```bash
k6 run --env BASE_URL=http://localhost:8081 load_tests/track_vote.js
```

> Note: if the Docker server OOMs during Go compilation (container memory < 2 GB), build and run the binary locally:
> ```bash
> cd server && go build -o /tmp/music-room-server ./cmd/main.go
> POSTGRES_HOST=localhost POSTGRES_PORT=5437 /tmp/music-room-server
> ```
> Then point k6 at `http://localhost:8082` (or whichever port the local binary binds to).
