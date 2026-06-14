/**
 * track_vote.js
 *
 * Simulates concurrent users voting on tracks in a shared event.
 * setup() creates one event and 250 tracks under a single test user.
 * Each VU votes on a different track per iteration (stride-based spread).
 * A second vote on the same track returns 409 -- accepted as correct behaviour.
 */
import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';
const EMAIL    = __ENV.TEST_EMAIL    || 'loadtest@musicroom.test';
const PASSWORD = __ENV.TEST_PASSWORD || 'loadtest123';
const TRACKS   = parseInt(__ENV.TRACK_COUNT || '50', 10);

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
  // Login
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email: EMAIL, password: PASSWORD }),
    { headers: JSON_HEADERS },
  );
  if (loginRes.status !== 200) {
    throw new Error(`login failed: ${loginRes.status} ${loginRes.body}`);
  }
  const token = JSON.parse(loginRes.body).access_token;

  // Create event
  const evRes = http.post(
    `${BASE_URL}/api/v1/events`,
    JSON.stringify({ name: 'Load Test Event', visibility: 'public', license: 0 }),
    { headers: authHeaders(token) },
  );
  if (evRes.status !== 201) {
    throw new Error(`create event failed: ${evRes.status} ${evRes.body}`);
  }
  const eventId = JSON.parse(evRes.body).id;

  // Seed tracks
  const trackIds = [];
  for (let i = 0; i < TRACKS; i++) {
    const res = http.post(
      `${BASE_URL}/api/v1/events/${eventId}/tracks`,
      JSON.stringify({
        external_id: `deezer-lt-${i}`,
        title:       `Load Track ${i}`,
        artist:      'Load Test',
      }),
      { headers: authHeaders(token) },
    );
    if (res.status === 201 || res.status === 200) {
      trackIds.push(JSON.parse(res.body).id);
    }
  }

  if (trackIds.length === 0) {
    throw new Error('no tracks were created; check rate limits and server status');
  }

  return { token, eventId, trackIds };
}

export default function (data) {
  // Stride-based spread: each VU starts at a different track and advances by 1 each iteration.
  const idx = (__VU * 17 + __ITER) % data.trackIds.length;
  const trackId = data.trackIds[idx];

  const res = http.post(
    `${BASE_URL}/api/v1/events/${data.eventId}/tracks/${trackId}/vote`,
    '{}',
    { headers: authHeaders(data.token) },
  );

  // 200 = vote accepted; 409 = already voted (same user/track) -- both are valid server responses.
  check(res, { 'vote ok (200) or already cast (409)': r => r.status === 200 || r.status === 409 });
  sleep(0.05);
}

export function teardown(data) {
  http.del(
    `${BASE_URL}/api/v1/events/${data.eventId}`,
    null,
    { headers: authHeaders(data.token) },
  );
}
