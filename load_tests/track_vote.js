/**
 * track_vote.js
 *
 * Simulates concurrent users voting on tracks in a shared event.
 * setup() creates one event and 50 tracks under loadtest1, then logs in
 * one user per VU (loadtest1..loadtest150) via http.batch so each VU holds
 * its own JWT. This ensures votes actually INSERT into the DB and contend on
 * the unique(user, track) constraint only after a VU exhausts all tracks.
 */
import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL  = __ENV.BASE_URL  || 'http://localhost:8081';
const PASSWORD  = __ENV.TEST_PASSWORD || 'loadtest123';
const TRACKS    = parseInt(__ENV.TRACK_COUNT || '50', 10);
const MAX_VUS   = 150;

export const options = {
  setupTimeout: '120s',
  stages: [
    { duration: '30s', target: 50  },
    { duration: '1m',  target: 100 },
    { duration: '30s', target: 150 },
    { duration: '1m',  target: 150 },
    { duration: '30s', target: 0   },
  ],
  thresholds: {
    checks:            ['rate>0.99'],
    http_req_duration: ['p(95)<500'],
  },
};

const JSON_HEADERS = { 'Content-Type': 'application/json' };

function authHeaders(token) {
  return { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` };
}

export function setup() {
  // Login user 1 to create the shared event and seed tracks.
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email: 'loadtest1@musicroom.test', password: PASSWORD }),
    { headers: JSON_HEADERS },
  );
  if (loginRes.status !== 200) {
    throw new Error(`login failed: ${loginRes.status} ${loginRes.body}`);
  }
  const adminToken = JSON.parse(loginRes.body).access_token;

  // Create event.
  const evRes = http.post(
    `${BASE_URL}/api/v1/events`,
    JSON.stringify({ name: 'Load Test Event', visibility: 'public', license: 0 }),
    { headers: authHeaders(adminToken) },
  );
  if (evRes.status !== 201) {
    throw new Error(`create event failed: ${evRes.status} ${evRes.body}`);
  }
  const eventId = JSON.parse(evRes.body).id;

  // Seed tracks.
  const trackIds = [];
  for (let i = 0; i < TRACKS; i++) {
    const res = http.post(
      `${BASE_URL}/api/v1/events/${eventId}/tracks`,
      JSON.stringify({ external_id: `deezer-lt-${i}`, title: `Load Track ${i}`, artist: 'Load Test' }),
      { headers: authHeaders(adminToken) },
    );
    if (res.status === 201 || res.status === 200) {
      trackIds.push(JSON.parse(res.body).id);
    }
  }
  if (trackIds.length === 0) {
    throw new Error('no tracks created; check rate limits and server status');
  }

  // Login all VU users in parallel so each VU gets its own JWT.
  const loginBatch = [];
  for (let i = 1; i <= MAX_VUS; i++) {
    loginBatch.push([
      'POST',
      `${BASE_URL}/api/v1/auth/login`,
      JSON.stringify({ email: `loadtest${i}@musicroom.test`, password: PASSWORD }),
      { headers: JSON_HEADERS },
    ]);
  }
  const responses = http.batch(loginBatch);
  const tokens = responses
    .filter(r => r.status === 200)
    .map(r => JSON.parse(r.body).access_token);

  if (tokens.length === 0) {
    throw new Error('failed to log in any VU user; run "make load-test" to seed them first');
  }

  return { tokens, eventId, trackIds };
}

export default function (data) {
  // Each VU uses its own token; wraps around if fewer tokens than VUs.
  const token   = data.tokens[(__VU - 1) % data.tokens.length];
  // Stride-based spread: each VU starts at a different track.
  const idx     = (__VU * 17 + __ITER) % data.trackIds.length;
  const trackId = data.trackIds[idx];

  const res = http.post(
    `${BASE_URL}/api/v1/events/${data.eventId}/tracks/${trackId}/vote`,
    '{}',
    { headers: authHeaders(token) },
  );

  // 200 = vote inserted; 409 = this user already voted this track this run.
  check(res, { 'vote ok (200) or already cast (409)': r => r.status === 200 || r.status === 409 });
  sleep(0.05);
}

export function teardown(data) {
  if (!data.tokens || !data.tokens.length) return;
  http.del(
    `${BASE_URL}/api/v1/events/${data.eventId}`,
    null,
    { headers: authHeaders(data.tokens[0]) },
  );
}
