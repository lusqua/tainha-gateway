import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const mappingLatency = new Trend('mapping_latency', true);
const authLatency = new Trend('auth_latency', true);
const proxyLatency = new Trend('proxy_latency', true);

const GATEWAY = __ENV.GATEWAY_URL || 'http://localhost:8080';
const TOKEN = __ENV.JWT_TOKEN || '';

export const options = {
  scenarios: {
    // Smoke test: low load, check everything works
    smoke: {
      executor: 'constant-vus',
      vus: 5,
      duration: '10s',
      startTime: '0s',
      tags: { scenario: 'smoke' },
    },
    // Load test: normal expected traffic
    load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '15s', target: 50 },   // ramp up
        { duration: '30s', target: 50 },   // sustain
        { duration: '10s', target: 0 },    // ramp down
      ],
      startTime: '12s',
      tags: { scenario: 'load' },
    },
    // Stress test: push beyond normal
    stress: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 100 },
        { duration: '20s', target: 200 },
        { duration: '10s', target: 0 },
      ],
      startTime: '70s',
      tags: { scenario: 'stress' },
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],   // 95th < 500ms, 99th < 1s
    errors: ['rate<0.05'],                               // <5% errors
    http_req_failed: ['rate<0.05'],
  },
};

const authHeaders = {
  headers: { Authorization: `Bearer ${TOKEN}` },
};

export default function () {
  group('Public Routes', () => {
    // Simple proxy — no auth, no mapping
    const categories = http.get(`${GATEWAY}/api/categories`);
    check(categories, { 'categories 200': (r) => r.status === 200 });
    proxyLatency.add(categories.timings.duration);
    errorRate.add(categories.status !== 200);

    // Proxy with path param
    const product = http.get(`${GATEWAY}/api/products/1`);
    check(product, { 'product 200': (r) => r.status === 200 });
    proxyLatency.add(product.timings.duration);
    errorRate.add(product.status !== 200);

    // Proxy + response mapping (most expensive)
    const products = http.get(`${GATEWAY}/api/products`);
    check(products, {
      'products 200': (r) => r.status === 200,
      'has mapping': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body[0] && body[0].category !== undefined;
        } catch { return false; }
      },
    });
    mappingLatency.add(products.timings.duration);
    errorRate.add(products.status !== 200);
  });

  group('Auth Routes', () => {
    // Protected route without token
    const noAuth = http.get(`${GATEWAY}/api/users`);
    check(noAuth, { 'no auth 401': (r) => r.status === 401 });

    // Protected route with token
    const users = http.get(`${GATEWAY}/api/users`, authHeaders);
    check(users, { 'users 200': (r) => r.status === 200 });
    authLatency.add(users.timings.duration);
    errorRate.add(users.status !== 200);

    // Protected + path param + mapping
    const user = http.get(`${GATEWAY}/api/users/1`, authHeaders);
    check(user, {
      'user 200': (r) => r.status === 200,
      'user has orders': (r) => {
        try {
          return JSON.parse(r.body).orders !== undefined;
        } catch { return false; }
      },
    });
    authLatency.add(user.timings.duration);
    mappingLatency.add(user.timings.duration);
    errorRate.add(user.status !== 200);
  });

  group('Health & Metrics', () => {
    const health = http.get(`${GATEWAY}/health`);
    check(health, { 'health 200': (r) => r.status === 200 });
  });

  sleep(0.1); // small pause between iterations
}

export function handleSummary(data) {
  const m = data.metrics;
  const dur = m.http_req_duration ? m.http_req_duration.values : {};
  const reqs = m.http_reqs ? m.http_reqs.values : {};
  const durationMs = data.state ? data.state.testRunDurationMs : 1;

  const fmt = (v) => v !== undefined && v !== null ? v.toFixed(1) : 'N/A';
  const fmtMs = (v) => v !== undefined && v !== null ? `${v.toFixed(1)}ms` : 'N/A';

  const summary = [
    `  Total requests:  ${reqs.count || 0}`,
    `  RPS:             ${fmt((reqs.count || 0) / (durationMs / 1000))}`,
    `  Avg latency:     ${fmtMs(dur.avg)}`,
    `  P95 latency:     ${fmtMs(dur['p(95)'])}`,
    `  P99 latency:     ${fmtMs(dur['p(99)'])}`,
    `  Error rate:      ${m.errors ? fmt(m.errors.values.rate * 100) : '0'}%`,
    `  Mapping avg:     ${m.mapping_latency ? fmtMs(m.mapping_latency.values.avg) : 'N/A'}`,
    `  Auth avg:        ${m.auth_latency ? fmtMs(m.auth_latency.values.avg) : 'N/A'}`,
    `  Proxy avg:       ${m.proxy_latency ? fmtMs(m.proxy_latency.values.avg) : 'N/A'}`,
  ];

  const output = '\n=== TAINHA GATEWAY LOAD TEST RESULTS ===\n' +
    summary.join('\n') +
    '\n========================================\n';

  return { stdout: output };
}
