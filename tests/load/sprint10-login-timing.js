// sprint10-login-timing.js
// Load test for Sprint 10: Random Login Delay (Timing Attack Mitigation)
//
// Purpose: Verify that the random 100-300ms delay:
// 1. Is consistently applied to all login attempts (success and failure)
// 2. Doesn't cause p95 latency to exceed 500ms under load
// 3. Effectively masks timing differences between valid/invalid credentials
//
// Security Gate: S10-PERF-001 - <500ms p95 login latency

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';
import { endpoint, getThinkTime } from './helpers/config.js';
import { generateRandomEmail, generateRandomPassword } from './helpers/data.js';

// Custom metrics for timing analysis
const loginSuccessDuration = new Trend('login_success_duration', true);
const loginFailureDuration = new Trend('login_failure_duration', true);
const loginDelayConsistency = new Rate('login_delay_consistency');
const loginAttemptsSuccess = new Counter('login_attempts_success');
const loginAttemptsFailure = new Counter('login_attempts_failure');

// Pre-created test users (created during setup)
let testUsers = [];

// Test configuration
export const options = {
  scenarios: {
    // Scenario 1: Gradual ramp-up to test p95 latency under increasing load
    timing_verification: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 10 },   // Warm-up to 10 VUs
        { duration: '3m', target: 50 },   // Ramp to 50 VUs
        { duration: '5m', target: 100 },  // Peak load at 100 VUs
        { duration: '2m', target: 10 },   // Ramp down to 10 VUs
        { duration: '1m', target: 0 },    // Cool down
      ],
      gracefulRampDown: '30s',
      exec: 'testLoginTiming',
    },

    // Scenario 2: Constant load to measure timing consistency
    consistency_check: {
      executor: 'constant-vus',
      vus: 20,
      duration: '10m',
      startTime: '2m',  // Start after warm-up
      exec: 'testTimingConsistency',
    },
  },

  // Performance thresholds (Security Gate S10-PERF-001)
  thresholds: {
    // Primary requirement: p95 login latency < 500ms (including 100-300ms delay)
    'http_req_duration{endpoint:login_success}': [
      'p(95)<500',    // CRITICAL: Must pass for security gate
      'p(99)<750',    // Additional safety margin
      'avg<300',      // Average should be reasonable
    ],
    'http_req_duration{endpoint:login_failure}': [
      'p(95)<500',    // Failed logins should have same latency profile
      'p(99)<750',
      'avg<300',
    ],

    // Timing consistency: success vs failure should not be statistically different
    // We measure this by ensuring both have delays in the 100-300ms range
    'login_success_duration': [
      'p(50)>200',    // Median should be ~200ms (middle of 100-300ms range)
      'p(95)<500',    // p95 within gate requirement
    ],
    'login_failure_duration': [
      'p(50)>200',    // Should match success timing
      'p(95)<500',
    ],

    // Delay consistency: at least 95% of requests should have delay in range
    'login_delay_consistency': ['rate>0.95'],

    // Error rate should be minimal (excluding intentional failures)
    'http_req_failed{endpoint:login_success}': ['rate<0.01'],
  },

  // Test metadata
  tags: {
    test_type: 'load',
    sprint: 'sprint-10',
    feature: 'timing-attack-mitigation',
    security_gate: 'S10-PERF-001',
  },
};

// Setup function - creates test users
export function setup() {
  console.log('='.repeat(60));
  console.log('Sprint 10 Load Test: Login Timing Attack Mitigation');
  console.log('='.repeat(60));
  console.log('Security Gate: S10-PERF-001');
  console.log('Requirement: p95 login latency < 500ms');
  console.log('Expected Delay: 100-300ms random per login attempt');
  console.log('');

  // Verify API is accessible
  const healthCheck = http.get(endpoint('/health'));
  if (healthCheck.status !== 200) {
    throw new Error('API health check failed - is the server running?');
  }
  console.log('✓ API health check passed');

  // Create pool of test users for consistent timing tests
  const userPool = [];
  const poolSize = 50; // Enough users to avoid rate limiting

  console.log(`Creating ${poolSize} test users for timing verification...`);

  for (let i = 0; i < poolSize; i++) {
    const email = `timing-test-${Date.now()}-${i}@loadtest.local`;
    const password = generateRandomPassword(16);
    const username = `timinguser${Date.now()}${i}`;

    const payload = JSON.stringify({
      email: email,
      username: username,
      password: password,
    });

    const response = http.post(
      endpoint('/auth/register'),
      payload,
      {
        headers: { 'Content-Type': 'application/json' },
        tags: { endpoint: 'setup_register' },
      }
    );

    if (response.status === 201 || response.status === 200) {
      userPool.push({
        email: email,
        password: password,
        username: username,
        id: JSON.parse(response.body).user?.id || null,
      });

      if ((i + 1) % 10 === 0) {
        console.log(`  Created ${i + 1}/${poolSize} users...`);
      }
    } else {
      console.error(`Failed to create user ${i}: ${response.status}`);
    }

    // Small delay to avoid overwhelming the server during setup
    sleep(0.1);
  }

  console.log(`✓ Created ${userPool.length} test users`);
  console.log('');
  console.log('Starting load test...');
  console.log('='.repeat(60));

  return { users: userPool };
}

// Scenario 1: Test login timing under increasing load
export function testLoginTiming(data) {
  const users = data.users;
  if (!users || users.length === 0) {
    console.error('No test users available');
    return;
  }

  // Randomly select a valid user for successful login
  const validUser = users[Math.floor(Math.random() * users.length)];

  // Alternate between successful and failed login attempts
  const testSuccess = Math.random() < 0.5;

  if (testSuccess) {
    // Test successful login with valid credentials
    const startTime = Date.now();

    const payload = JSON.stringify({
      identifier: validUser.email,
      password: validUser.password,
    });

    const response = http.post(
      endpoint('/auth/login'),
      payload,
      {
        headers: { 'Content-Type': 'application/json' },
        tags: { endpoint: 'login_success' },
      }
    );

    const duration = Date.now() - startTime;

    const success = check(response, {
      'successful login: status 200': (r) => r.status === 200,
      'successful login: has access token': (r) => {
        if (r.status === 200) {
          const body = JSON.parse(r.body);
          return body.tokens && body.tokens.accessToken;
        }
        return false;
      },
      'successful login: delay in range (100-500ms)': (r) => {
        // The delay should be 100-300ms + processing time
        // Total should be < 500ms at p95
        return duration >= 100 && duration <= 500;
      },
    });

    loginSuccessDuration.add(duration);
    loginAttemptsSuccess.add(1);

    if (duration >= 100 && duration <= 500) {
      loginDelayConsistency.add(true);
    } else {
      loginDelayConsistency.add(false);
    }

  } else {
    // Test failed login with invalid credentials
    const startTime = Date.now();

    const payload = JSON.stringify({
      identifier: generateRandomEmail(),  // Random non-existent email
      password: 'InvalidPassword123!',
    });

    const response = http.post(
      endpoint('/auth/login'),
      payload,
      {
        headers: { 'Content-Type': 'application/json' },
        tags: { endpoint: 'login_failure' },
      }
    );

    const duration = Date.now() - startTime;

    check(response, {
      'failed login: status 401': (r) => r.status === 401,
      'failed login: error message present': (r) => {
        if (r.status === 401) {
          const body = JSON.parse(r.body);
          return body.title || body.detail;
        }
        return false;
      },
      'failed login: delay in range (100-500ms)': (r) => {
        // Failed logins should have SAME timing as successful ones
        return duration >= 100 && duration <= 500;
      },
    });

    loginFailureDuration.add(duration);
    loginAttemptsFailure.add(1);

    if (duration >= 100 && duration <= 500) {
      loginDelayConsistency.add(true);
    } else {
      loginDelayConsistency.add(false);
    }
  }

  // Think time between requests
  sleep(getThinkTime(500, 1500) / 1000);
}

// Scenario 2: Test timing consistency (statistical analysis)
export function testTimingConsistency(data) {
  const users = data.users;
  if (!users || users.length === 0) {
    console.error('No test users available');
    return;
  }

  const validUser = users[Math.floor(Math.random() * users.length)];

  // Perform both successful and failed login in same iteration
  // to compare timing profiles

  // 1. Successful login
  const successStart = Date.now();
  const successPayload = JSON.stringify({
    identifier: validUser.email,
    password: validUser.password,
  });

  const successResponse = http.post(
    endpoint('/auth/login'),
    successPayload,
    {
      headers: { 'Content-Type': 'application/json' },
      tags: { endpoint: 'login_success', test_type: 'consistency' },
    }
  );

  const successDuration = Date.now() - successStart;
  loginSuccessDuration.add(successDuration);

  sleep(0.5); // Brief pause between attempts

  // 2. Failed login
  const failureStart = Date.now();
  const failurePayload = JSON.stringify({
    identifier: validUser.email,
    password: 'WrongPassword123!',
  });

  const failureResponse = http.post(
    endpoint('/auth/login'),
    failurePayload,
    {
      headers: { 'Content-Type': 'application/json' },
      tags: { endpoint: 'login_failure', test_type: 'consistency' },
    }
  );

  const failureDuration = Date.now() - failureStart;
  loginFailureDuration.add(failureDuration);

  // Statistical check: timing difference should be minimal
  // (random delay should mask the difference)
  const timingDiff = Math.abs(successDuration - failureDuration);

  check(null, {
    'timing difference < 50ms': () => timingDiff < 50,
  });

  // Both should be in the expected range
  if (successDuration >= 100 && successDuration <= 500) {
    loginDelayConsistency.add(true);
  } else {
    loginDelayConsistency.add(false);
  }

  if (failureDuration >= 100 && failureDuration <= 500) {
    loginDelayConsistency.add(true);
  } else {
    loginDelayConsistency.add(false);
  }

  sleep(getThinkTime(1000, 2000) / 1000);
}

// Teardown function
export function teardown(data) {
  console.log('');
  console.log('='.repeat(60));
  console.log('Sprint 10 Load Test Complete');
  console.log('='.repeat(60));
  console.log('');
  console.log('Review the metrics above to verify:');
  console.log('1. p95 login latency < 500ms (Security Gate S10-PERF-001)');
  console.log('2. Timing consistency between success/failure > 95%');
  console.log('3. Both success and failure delays in 100-500ms range');
  console.log('');
  console.log('Cleanup: Test users will remain for manual inspection');
  console.log('To clean up, run: make clean-test-users');
}

// Handle summary for detailed reporting
export function handleSummary(data) {
  const successP95 = data.metrics.login_success_duration?.values?.['p(95)'] || 0;
  const failureP95 = data.metrics.login_failure_duration?.values?.['p(95)'] || 0;
  const consistency = data.metrics.login_delay_consistency?.values?.rate || 0;

  const summary = {
    'Sprint 10 - Login Timing Load Test Summary': {
      security_gate: 'S10-PERF-001',
      requirement: 'p95 login latency < 500ms',
      results: {
        success_p95_ms: successP95.toFixed(2),
        failure_p95_ms: failureP95.toFixed(2),
        delay_consistency_rate: (consistency * 100).toFixed(2) + '%',
        gate_status: successP95 < 500 && failureP95 < 500 ? 'PASS ✓' : 'FAIL ✗',
      },
    },
    raw_data: data,
  };

  return {
    'stdout': JSON.stringify(summary, null, 2),
    'sprint10-login-timing-results.json': JSON.stringify(data, null, 2),
  };
}
