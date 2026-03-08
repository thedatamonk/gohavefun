import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '10s', target: 100 },  // ramp up to 10 users
    { duration: '20s', target: 1000 },  // ramp up to 50 users
    { duration: '20s', target: 1000 },  // stay at 50 users
    { duration: '10s', target: 0 },   // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(99)<100'],  // 99% of requests must finish within 100ms
    errors: ['rate<0.01'],             // error rate must be below 1%
  },
};

const BASE_URL = 'http://localhost:8080';

function randomCustomerID() {
  const n = Math.floor(Math.random() * 75) + 1;
  return `cust-${String(n).padStart(4, '0')}`;
}

export default function () {
  const id = randomCustomerID();

  group('health check', () => {
    const res = http.get(`${BASE_URL}/health`);
    const ok = check(res, { 'status 200': (r) => r.status === 200 });
    errorRate.add(!ok);
  });

  group('get single feature', () => {
    const res = http.get(`${BASE_URL}/features/customer_profile/${id}`);
    const ok = check(res, {
      'status 200': (r) => r.status === 200,
      'has tenure_months': (r) => r.body.includes('tenure_months'),
    });
    errorRate.add(!ok);
  });

  group('get batch features', () => {
    const payload = JSON.stringify({
      keys: [
        { entity_type: 'customer_profile', entity_id: id },
        { entity_type: 'usage_metrics', entity_id: id },
        { entity_type: 'billing', entity_id: id },
        { entity_type: 'support', entity_id: id },
      ],
    });
    const res = http.post(`${BASE_URL}/features/batch`, payload, {
      headers: { 'Content-Type': 'application/json' },
    });
    const ok = check(res, { 'status 200': (r) => r.status === 200 });
    errorRate.add(!ok);
  });

  group('predict churn', () => {
    const res = http.get(`${BASE_URL}/predict/${id}`);
    const ok = check(res, {
      'status 200': (r) => r.status === 200,
      'has churn_probability': (r) => r.body.includes('churn_probability'),
      'has risk_level': (r) => r.body.includes('risk_level'),
    });
    errorRate.add(!ok);
  });

  sleep(0.1);
}
