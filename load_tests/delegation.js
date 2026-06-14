/**
 * delegation.js
 *
 * Simulates concurrent users sending playback commands via the delegation service.
 * setup() registers a device under the test user.
 * All VUs share the same token and alternate play/pause commands each iteration.
 * Commands are stateless relays -- the same action can be sent repeatedly.
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
    { duration: '30s', target: 200 },
    { duration: '1m',  target: 200 },
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

  const devRes = http.post(
    `${BASE_URL}/api/v1/devices`,
    JSON.stringify({ name: 'Load Test Device', platform: 'android', model: 'Pixel 8' }),
    { headers: authHeaders(token) },
  );
  if (devRes.status !== 201) {
    throw new Error(`register device failed: ${devRes.status} ${devRes.body}`);
  }
  const deviceId = JSON.parse(devRes.body).id;

  return { token, deviceId };
}

export default function (data) {
  const action = __ITER % 2 === 0 ? 'play' : 'pause';

  const res = http.post(
    `${BASE_URL}/api/v1/devices/${data.deviceId}/command`,
    JSON.stringify({ action }),
    { headers: authHeaders(data.token) },
  );

  check(res, { 'command accepted (2xx)': r => r.status >= 200 && r.status < 300 });
  sleep(0.05);
}

export function teardown(data) {
  http.del(
    `${BASE_URL}/api/v1/devices/${data.deviceId}`,
    null,
    { headers: authHeaders(data.token) },
  );
}
