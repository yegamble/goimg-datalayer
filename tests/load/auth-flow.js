// auth-flow.js
// Load test for user authentication flow
// Tests: Register → Login → Fetch profile → Logout

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { endpoint, getThinkTime } from './helpers/config.js';
import { registerUser, login, logout, getAuthHeaders } from './helpers/auth.js';

// Custom metrics
const authFlowDuration = new Trend('auth_flow_duration', true);
const registrationCount = new Counter('registration_count');
const loginCount = new Counter('login_count');
const logoutCount = new Counter('logout_count');

// Test configuration
export const options = {
  // Staged ramp-up with 50 concurrent VUs over 5 minutes
  stages: [
    { duration: '2m', target: 13 },  // Ramp 0 → 25% (12.5 VUs)
    { duration: '3m', target: 50 },  // Ramp 25% → 100% (50 VUs)
    { duration: '5m', target: 50 },  // Hold at 100% for test duration
    { duration: '1m', target: 0 },   // Ramp down
  ],

  // Performance thresholds
  thresholds: {
    // Standard endpoints (non-upload)
    'http_req_duration{endpoint:register}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:login}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:profile}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:logout}': ['p(95)<200', 'p(99)<500'],

    // Overall error rate
    'http_req_failed': ['rate<0.01'], // Less than 1% errors

    // Full flow duration
    'auth_flow_duration': ['p(95)<2000'], // Entire flow under 2s at p95
  },

  // Test metadata
  tags: {
    test_type: 'load',
    flow: 'authentication',
  },
};

// Main test function - executed by each VU repeatedly
export default function () {
  const flowStart = Date.now();

  // Step 1: Register new user
  const user = registerUser();
  if (!user) {
    console.error('Registration failed, skipping iteration');
    sleep(5); // Back off on failure
    return;
  }
  registrationCount.add(1);

  // Think time after registration (user reading confirmation)
  sleep(getThinkTime(1000, 3000) / 1000);

  // Step 2: Login with credentials
  const tokens = login(user.email, user.password);
  if (!tokens) {
    console.error('Login failed, skipping iteration');
    sleep(5);
    return;
  }
  loginCount.add(1);

  // Think time after login (user navigating)
  sleep(getThinkTime(1000, 3000) / 1000);

  // Step 3: Fetch own profile
  const profileResponse = http.get(
    endpoint(`/users/${user.id}`),
    {
      headers: getAuthHeaders(tokens.accessToken),
      tags: { endpoint: 'profile' },
    }
  );

  check(profileResponse, {
    'profile fetch successful': (r) => r.status === 200,
    'profile has username': (r) => {
      if (r.status === 200) {
        const body = JSON.parse(r.body);
        return body.username === user.username;
      }
      return false;
    },
  });

  // Think time (user viewing profile)
  sleep(getThinkTime(2000, 4000) / 1000);

  // Step 4: Logout
  const loggedOut = logout(tokens.accessToken);
  if (loggedOut) {
    logoutCount.add(1);
  }

  // Record total flow duration
  const flowDuration = Date.now() - flowStart;
  authFlowDuration.add(flowDuration);

  // Think time before next iteration
  sleep(getThinkTime(1000, 3000) / 1000);
}

// Setup function (runs once at start)
export function setup() {
  console.log('Starting auth-flow load test');
  console.log('Configuration: 50 VUs, 5 minutes duration');

  // Verify API is accessible
  const healthCheck = http.get(endpoint('/health'));
  if (healthCheck.status !== 200) {
    throw new Error('API health check failed - is the server running?');
  }

  return {};
}

// Teardown function (runs once at end)
export function teardown(data) {
  console.log('Auth-flow load test completed');
}

// Handle summary to include custom metrics
export function handleSummary(data) {
  return {
    'stdout': JSON.stringify(data, null, 2),
    'auth-flow-summary.json': JSON.stringify(data),
  };
}
