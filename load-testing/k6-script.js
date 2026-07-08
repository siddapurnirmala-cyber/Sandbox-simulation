import http from 'k6/http';
import { sleep, check } from 'k6';

export const options = {
  stages: [
    { duration: '15s', target: 10 }, // Ramp up to 10 users
    { duration: '30s', target: 10 }, // Stay at 10 users (normal load)
    { duration: '15s', target: 50 }, // Ramp up to 50 users (spike/stress)
    { duration: '30s', target: 50 }, // Stay at 50 users (high load)
    { duration: '15s', target: 0 },  // Ramp down to 0 users
  ],
};

const BASE_URL = `http://${__ENV.BACKEND_HOST || 'localhost'}:8080`;

export default function () {
  // 1. Hit Health endpoint
  let res = http.get(`${BASE_URL}/health`);
  check(res, {
    'status is 200': (r) => r.status === 200,
    'status is 500 (simulated failure)': (r) => r.status === 500,
  });
  sleep(1);

  // 2. Fetch list of sandboxes
  res = http.get(`${BASE_URL}/sandbox`);
  check(res, {
    'sandbox query success': (r) => r.status === 200,
  });
  sleep(1);

  // 3. Simulating CRUD activity
  // Only 5% of requests will create a new sandbox to prevent database bloat
  if (Math.random() < 0.05) {
    const payload = JSON.stringify({
      sandbox_name: `k6-sandbox-${Math.floor(Math.random() * 10000)}`,
      owner: 'k6-load-tester@enterprise.com',
    });
    
    const params = {
      headers: {
        'Content-Type': 'application/json',
      },
    };

    res = http.post(`${BASE_URL}/sandbox`, payload, params);
    if (check(res, { 'sandbox created successfully': (r) => r.status === 201 })) {
      const sandbox = JSON.parse(res.body);
      const id = sandbox.id;

      // Connect to sandbox VSI (triggers simulated latency & fail rates)
      res = http.post(`${BASE_URL}/sandbox/${id}/connect`);
      check(res, {
        'vsi connection completed': (r) => r.status === 200 || r.status === 500,
      });
      sleep(2);

      // If connected, execute a command on the terminal
      if (res.status === 200) {
        const cmdPayload = JSON.stringify({ command: 'uname -a' });
        res = http.post(`${BASE_URL}/sandbox/${id}/run-command`, cmdPayload, params);
        check(res, {
          'command executed successfully': (r) => r.status === 200,
        });
        sleep(2);

        // Disconnect VSI
        http.post(`${BASE_URL}/sandbox/${id}/disconnect`);
      }

      // Delete sandbox environment
      http.del(`${BASE_URL}/sandbox/${id}`);
    }
  }

  // 4. Query Database Event Logs
  res = http.get(`${BASE_URL}/logs?limit=20`);
  check(res, {
    'logs query success': (r) => r.status === 200,
  });
  sleep(1);

  // 5. Query Prometheus metrics
  res = http.get(`${BASE_URL}/metrics`);
  check(res, {
    'metrics query success': (r) => r.status === 200,
  });
  sleep(1);

  // 6. Inject failures dynamically (only run occasionally by 1 user thread to show graphs updates)
  if (__VU === 1 && Math.random() < 0.02) {
    // Inject DB Failure
    http.post(`${BASE_URL}/simulate/db-failure`, JSON.stringify({ enable: true }), { headers: { 'Content-Type': 'application/json' } });
    sleep(5); // Fail queries for 5 seconds
    http.post(`${BASE_URL}/simulate/db-failure`, JSON.stringify({ enable: false }), { headers: { 'Content-Type': 'application/json' } });

    // Inject CPU workload
    http.post(`${BASE_URL}/simulate/high-cpu`, JSON.stringify({ enable: true }), { headers: { 'Content-Type': 'application/json' } });
    sleep(10); // Stress CPU for 10 seconds
    http.post(`${BASE_URL}/simulate/high-cpu`, JSON.stringify({ enable: false }), { headers: { 'Content-Type': 'application/json' } });

    // Inject API Delay
    http.post(`${BASE_URL}/simulate/api-delay`, JSON.stringify({ delay_ms: 2000 }), { headers: { 'Content-Type': 'application/json' } });
    sleep(8); // Inject API latency for 8 seconds
    http.post(`${BASE_URL}/simulate/api-delay`, JSON.stringify({ delay_ms: 0 }), { headers: { 'Content-Type': 'application/json' } });
  }
}
