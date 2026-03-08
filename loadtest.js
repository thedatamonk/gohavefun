import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const predictLatency = new Trend('predict_latency', true);
const registryLatency = new Trend('registry_latency', true);
const writeLatency = new Trend('write_latency', true);

export const options = {
  scenarios: {
    // Simulate read-heavy production traffic
    read_heavy: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 100 },
        { duration: '20s', target: 1000 },
        { duration: '20s', target: 1000 },
        { duration: '10s', target: 0 },
      ],
      exec: 'readWorkload'
    },
    // Simulate write traffic (lower volume, validated)
    write_traffic: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 10 },
        { duration: '20s', target: 100 },
        { duration: '20s', target: 100 },
        { duration: '10s', target: 0 },
      ],
      exec: 'writeWorkload',
    },
    // Registry browsing (metadata lookups)
    registry_reads: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 20 },
        { duration: '20s', target: 200 },
        { duration: '20s', target: 200 },
        { duration: '10s', target: 0 },
      ],
      exec: 'registryWorkload',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    errors: ['rate<0.01'],
    predict_latency: ['p(95)<600'],
    registry_latency: ['p(95)<300'],
    write_latency: ['p(95)<500'],
  },
};

const BASE_URL = 'http://localhost:8080';
const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } };

const FEATURE_VIEWS = [
  'customer_profile',
  'usage_metrics',
  'billing',
  'support',
];

function randomCustomerID() {
  const n = Math.floor(Math.random() * 5000) + 1;
  return `cust-${String(n).padStart(4, '0')}`;
}

// Read-heavy workload: feature lookups, batch reads, predictions
export function readWorkload() {
  const id = randomCustomerID();

  group('health check', () => {
    const res = http.get(`${BASE_URL}/health`);
    const ok = check(res, { 'status 200': (r) => r.status === 200 });
    errorRate.add(!ok);
  });

  group('get single feature', () => {
    const view = FEATURE_VIEWS[Math.floor(Math.random() * FEATURE_VIEWS.length)];
    const res = http.get(`${BASE_URL}/features/${view}/${id}`);
    const ok = check(res, {
      'status 200': (r) => r.status === 200,
      'has data': (r) => r.body.length > 2,
    });
    errorRate.add(!ok);
  });

  group('get all customer features', () => {
    const res = http.get(`${BASE_URL}/customers/${id}/features`);
    const ok = check(res, {
      'status 200': (r) => r.status === 200,
      'has all groups': (r) =>
        r.body.includes('customer_profile') &&
        r.body.includes('usage_metrics') &&
        r.body.includes('billing') &&
        r.body.includes('support'),
    });
    errorRate.add(!ok);
  });

  group('batch features', () => {
    const payload = JSON.stringify({
      keys: FEATURE_VIEWS.map((v) => ({
        entity_type: v,
        entity_id: id,
      })),
    });
    const res = http.post(`${BASE_URL}/features/batch`, payload, JSON_HEADERS);
    const ok = check(res, { 'status 200': (r) => r.status === 200 });
    errorRate.add(!ok);
  });

  group('predict churn', () => {
    const res = http.get(`${BASE_URL}/predict/${id}`);
    predictLatency.add(res.timings.duration);
    const ok = check(res, {
      'status 200': (r) => r.status === 200,
      'has churn_probability': (r) => r.body.includes('churn_probability'),
      'has risk_level': (r) => r.body.includes('risk_level'),
      'has top_risk_factors': (r) => r.body.includes('top_risk_factors'),
    });
    errorRate.add(!ok);
  });

  sleep(0.1);
}

// Write workload: validated feature writes (mix of valid and invalid)
export function writeWorkload() {
  const id = randomCustomerID();

  group('write customer_profile', () => {
    const payload = JSON.stringify({
      tenure_months: Math.floor(Math.random() * 72) + 1,
      plan_tier: Math.floor(Math.random() * 3) + 1,
      monthly_charge: Math.random() * 60 + 9.99,
    });
    const res = http.post(
      `${BASE_URL}/features/customer_profile/${id}`,
      payload,
      JSON_HEADERS
    );
    writeLatency.add(res.timings.duration);
    const ok = check(res, { 'status 201': (r) => r.status === 201 });
    errorRate.add(!ok);
  });

  group('write usage_metrics', () => {
    const payload = JSON.stringify({
      logins_last_30d: Math.floor(Math.random() * 30),
      avg_session_minutes: Math.floor(Math.random() * 60),
      days_since_last_login: Math.floor(Math.random() * 30),
      feature_adoption_pct: Math.floor(Math.random() * 100),
    });
    const res = http.post(
      `${BASE_URL}/features/usage_metrics/${id}`,
      payload,
      JSON_HEADERS
    );
    writeLatency.add(res.timings.duration);
    const ok = check(res, { 'status 201': (r) => r.status === 201 });
    errorRate.add(!ok);
  });

  group('write billing', () => {
    const payload = JSON.stringify({
      total_spend: Math.random() * 3000 + 50,
      late_payments_count: Math.floor(Math.random() * 5),
      avg_monthly_spend: Math.random() * 60 + 10,
    });
    const res = http.post(
      `${BASE_URL}/features/billing/${id}`,
      payload,
      JSON_HEADERS
    );
    writeLatency.add(res.timings.duration);
    const ok = check(res, { 'status 201': (r) => r.status === 201 });
    errorRate.add(!ok);
  });

  group('write support', () => {
    const payload = JSON.stringify({
      tickets_last_90d: Math.floor(Math.random() * 10),
      avg_resolution_hours: Math.floor(Math.random() * 72),
      escalation_count: Math.floor(Math.random() * 4),
    });
    const res = http.post(
      `${BASE_URL}/features/support/${id}`,
      payload,
      JSON_HEADERS
    );
    writeLatency.add(res.timings.duration);
    const ok = check(res, { 'status 201': (r) => r.status === 201 });
    errorRate.add(!ok);
  });

  // ~20% of writes attempt invalid features (should be rejected)
  if (Math.random() < 0.2) {
    group('invalid feature write (expected 400)', () => {
      const payload = JSON.stringify({ bad_field: 99, another_bad: 42 });
      const res = http.post(
        `${BASE_URL}/features/customer_profile/${id}`,
        payload,
        JSON_HEADERS
      );
      const ok = check(res, {
        'status 400': (r) => r.status === 400,
        'has error message': (r) => r.body.includes('unknown feature'),
      });
      errorRate.add(!ok);
    });
  }

  // ~10% of writes attempt unknown views (should be rejected)
  if (Math.random() < 0.1) {
    group('unknown view write (expected 400)', () => {
      const payload = JSON.stringify({ x: 1 });
      const res = http.post(
        `${BASE_URL}/features/nonexistent_view/${id}`,
        payload,
        JSON_HEADERS
      );
      const ok = check(res, {
        'status 400': (r) => r.status === 400,
        'has error message': (r) => r.body.includes('unknown feature view'),
      });
      errorRate.add(!ok);
    });
  }

  sleep(0.2);
}

// Registry workload: metadata reads
export function registryWorkload() {
  group('list all views', () => {
    const res = http.get(`${BASE_URL}/registry/feature-views`);
    registryLatency.add(res.timings.duration);
    const ok = check(res, {
      'status 200': (r) => r.status === 200,
      'has views': (r) => {
        try {
          const views = JSON.parse(r.body);
          return Array.isArray(views) && views.length >= 4;
        } catch {
          return false;
        }
      },
    });
    errorRate.add(!ok);
  });

  group('get specific view', () => {
    const view = FEATURE_VIEWS[Math.floor(Math.random() * FEATURE_VIEWS.length)];
    const res = http.get(`${BASE_URL}/registry/feature-views/${view}`);
    registryLatency.add(res.timings.duration);
    const ok = check(res, {
      'status 200': (r) => r.status === 200,
      'has features array': (r) => r.body.includes('"features"'),
      'has owner': (r) => r.body.includes('"owner"'),
    });
    errorRate.add(!ok);
  });

  group('get missing view (expected 404)', () => {
    const res = http.get(`${BASE_URL}/registry/feature-views/does_not_exist`);
    const ok = check(res, { 'status 404': (r) => r.status === 404 });
    errorRate.add(!ok);
  });

  sleep(0.2);
}
