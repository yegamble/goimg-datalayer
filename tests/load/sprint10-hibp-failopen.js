// sprint10-hibp-failopen.js
// Load test for Sprint 10: HIBP Fail-Open Behavior
//
// Purpose: Verify that when HIBP API is unavailable or slow:
// 1. Registration continues to work (fail-open)
// 2. Errors are logged but users are not blocked
// 3. System maintains acceptable performance
//
// NOTE: This test requires HIBP to be unreachable. Run with:
// - HIBP_ENABLED=false (to simulate complete failure)
// - Or network isolation of HIBP API
//
// Security Gate: S10-HIBP-003 - Fail-open behavior under HIBP API failure

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';
import { endpoint, getThinkTime } from './helpers/config.js';
import { generateRandomEmail, generateRandomPassword, generateRandomUsername } from './helpers/data.js';

// Custom metrics for fail-open analysis
const failOpenRegistrationDuration = new Trend('failopen_registration_duration', true);
const failOpenSuccessRate = new Rate('failopen_success_rate');
const failOpenRegistrationCount = new Counter('failopen_registration_count');

// Test configuration
export const options = {
  scenarios: {
    // Test registration when HIBP is unavailable
    failopen_verification: {
      executor: 'constant-vus',
      vus: 20,
      duration: '5m',
      exec: 'testFailOpenBehavior',
    },
  },

  // Performance thresholds for fail-open mode
  thresholds: {
    // Registration should still work
    'http_req_duration{endpoint:register_failopen}': [
      'p(95)<1000',   // Should be faster without HIBP API call
      'p(99)<2000',
    ],

    // Success rate should be high (>95%)
    'failopen_success_rate': [
      'rate>0.95',    // At least 95% of registrations should succeed
    ],

    // Error rate should be low
    'http_req_failed{endpoint:register_failopen}': [
      'rate<0.05',    // Less than 5% hard failures
    ],
  },

  // Test metadata
  tags: {
    test_type: 'load',
    sprint: 'sprint-10',
    feature: 'hibp-fail-open',
    security_gate: 'S10-HIBP-003',
  },
};

// Setup function
export function setup() {
  console.log('='.repeat(60));
  console.log('Sprint 10 Load Test: HIBP Fail-Open Behavior');
  console.log('='.repeat(60));
  console.log('Security Gate: S10-HIBP-003');
  console.log('Testing: System behavior when HIBP API is unavailable');
  console.log('');
  console.log('IMPORTANT: Ensure HIBP is disabled or unreachable:');
  console.log('  - Set HIBP_ENABLED=false, OR');
  console.log('  - Block network access to api.pwnedpasswords.com');
  console.log('');

  // Verify API is accessible
  const healthCheck = http.get(endpoint('/health'));
  if (healthCheck.status !== 200) {
    throw new Error('API health check failed - is the server running?');
  }
  console.log('✓ API health check passed');
  console.log('');
  console.log('Starting fail-open verification test...');
  console.log('='.repeat(60));

  return {};
}

// Test fail-open behavior
export function testFailOpenBehavior() {
  const email = generateRandomEmail();
  const username = generateRandomUsername();
  const password = generateRandomPassword(16);

  const startTime = Date.now();

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
      tags: { endpoint: 'register_failopen' },
    }
  );

  const duration = Date.now() - startTime;
  failOpenRegistrationDuration.add(duration);
  failOpenRegistrationCount.add(1);

  // In fail-open mode, registration should succeed
  const success = check(response, {
    'failopen: registration succeeds': (r) => r.status === 201 || r.status === 200,
    'failopen: user created': (r) => {
      if (r.status === 201 || r.status === 200) {
        const body = JSON.parse(r.body);
        return body.user && body.user.id;
      }
      return false;
    },
    'failopen: has access token': (r) => {
      if (r.status === 201 || r.status === 200) {
        const body = JSON.parse(r.body);
        return body.tokens && body.tokens.accessToken;
      }
      return false;
    },
    'failopen: response time < 1s': () => duration < 1000,
  });

  if (response.status === 201 || response.status === 200) {
    failOpenSuccessRate.add(true);
  } else {
    failOpenSuccessRate.add(false);
    // Log failures for investigation
    console.warn(
      `Fail-open registration failed unexpectedly: ${response.status} - ${response.body.substring(0, 200)}`
    );
  }

  sleep(getThinkTime(500, 1500) / 1000);
}

// Teardown function
export function teardown(data) {
  console.log('');
  console.log('='.repeat(60));
  console.log('Sprint 10 Fail-Open Test Complete');
  console.log('='.repeat(60));
  console.log('');
  console.log('Review the metrics above to verify:');
  console.log('1. Registration success rate > 95% (despite HIBP unavailable)');
  console.log('2. p95 latency < 1s (faster without HIBP API call)');
  console.log('3. No hard failures blocking users');
  console.log('');
  console.log('Expected behavior:');
  console.log('  - Users can register successfully');
  console.log('  - Server logs contain HIBP API failure warnings');
  console.log('  - System continues to operate normally');
  console.log('');
  console.log('Verify server logs for HIBP error messages.');
}

// Handle summary for detailed reporting
export function handleSummary(data) {
  const successRate = data.metrics.failopen_success_rate?.values?.rate || 0;
  const p95Duration = data.metrics.failopen_registration_duration?.values?.['p(95)'] || 0;
  const totalRegistrations = data.metrics.failopen_registration_count?.values?.count || 0;

  const summary = {
    'Sprint 10 - HIBP Fail-Open Load Test Summary': {
      security_gate: 'S10-HIBP-003',
      requirement: 'System continues to work when HIBP unavailable',
      results: {
        success_rate: (successRate * 100).toFixed(2) + '%',
        p95_duration_ms: p95Duration.toFixed(2),
        total_registrations: totalRegistrations,
        gate_status: successRate > 0.95 ? 'PASS ✓' : 'FAIL ✗',
      },
      notes: [
        'Check server logs for HIBP API failure warnings',
        'Ensure HIBP_ENABLED=false or HIBP API is unreachable',
        'Success indicates proper fail-open behavior',
      ],
    },
    raw_data: data,
  };

  return {
    'stdout': JSON.stringify(summary, null, 2),
    'sprint10-failopen-results.json': JSON.stringify(data, null, 2),
  };
}
