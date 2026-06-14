/**
 * playlist_editor.js
 *
 * Simulates concurrent users adding and reordering tracks in a shared playlist.
 * setup() creates a public (license=0) playlist and one seeded track.
 * Each iteration: 70 % add a new track, 30 % move the seeded track to a random position.
 * The playlist is public so all VUs can edit without ownership checks.
 */
import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';
const EMAIL    = __ENV.TEST_EMAIL    || 'loadtest@musicroom.test';
const PASSWORD = __ENV.TEST_PASSWORD || 'loadtest123';

export const options = {
  stages: [
    { duration: '30s', target: 50  },
    { duration: '1m',  target: 100 },
    { duration: '30s', target: 150 },
    { duration: '1m',  target: 150 },
    { duration: '30s', target: 0   },
  ],
  thresholds: {
    http_req_failed:   ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};

const JSON_HEADERS = { 'Content-Type': 'application/json' };

function authHeaders(token) {
  return { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` };
}

export function setup() {
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email: EMAIL, password: PASSWORD }),
    { headers: JSON_HEADERS },
  );
  if (loginRes.status !== 200) {
    throw new Error(`login failed: ${loginRes.status} ${loginRes.body}`);
  }
  const token = JSON.parse(loginRes.body).access_token;

  // Create public playlist (license=0: any authenticated user can add/remove tracks)
  const plRes = http.post(
    `${BASE_URL}/api/v1/playlists`,
    JSON.stringify({ name: 'Load Test Playlist', visibility: 'public', license: 0 }),
    { headers: authHeaders(token) },
  );
  if (plRes.status !== 201) {
    throw new Error(`create playlist failed: ${plRes.status} ${plRes.body}`);
  }
  const playlistId = JSON.parse(plRes.body).id;

  // Seed one track (used as the target for move operations)
  const trackRes = http.post(
    `${BASE_URL}/api/v1/playlists/${playlistId}/tracks`,
    JSON.stringify({ external_id: 'deezer-seed-0', title: 'Seed Track', artist: 'Load Test' }),
    { headers: authHeaders(token) },
  );
  if (trackRes.status !== 201 && trackRes.status !== 200) {
    throw new Error(`seed track failed: ${trackRes.status} ${trackRes.body}`);
  }
  const trackId = JSON.parse(trackRes.body).id;

  return { token, playlistId, trackId };
}

export default function (data) {
  let res;

  if (__ITER % 10 < 7) {
    // 70 %: add a new track with a unique external_id
    res = http.post(
      `${BASE_URL}/api/v1/playlists/${data.playlistId}/tracks`,
      JSON.stringify({
        external_id: `deezer-lt-${__VU}-${__ITER}`,
        title:       `Track VU${__VU} iter${__ITER}`,
        artist:      'Load Test',
      }),
      { headers: authHeaders(data.token) },
    );
    check(res, { 'add track (2xx)': r => r.status >= 200 && r.status < 300 });
  } else {
    // 30 %: move the seeded track to a new position (1-based, must be >= 1)
    const pos = (__ITER % 10) + 1;
    res = http.patch(
      `${BASE_URL}/api/v1/playlists/${data.playlistId}/tracks/${data.trackId}/position`,
      JSON.stringify({ position: pos }),
      { headers: authHeaders(data.token) },
    );
    check(res, { 'move track (2xx)': r => r.status >= 200 && r.status < 300 });
  }

  sleep(0.05);
}

export function teardown(data) {
  http.del(
    `${BASE_URL}/api/v1/playlists/${data.playlistId}`,
    null,
    { headers: authHeaders(data.token) },
  );
}
