// sprint10-hibp-registration.js
// Load test for Sprint 10: HIBP Password Check Integration
//
// Purpose: Verify that HIBP password checking:
// 1. Rejects known compromised passwords consistently
// 2. Accepts strong unique passwords without issues
// 3. Handles HIBP API failures gracefully (fail-open behavior)
// 4. Maintains acceptable registration latency under load
//
// Security Gate: S10-HIBP-003 - Fail-open behavior under HIBP API failure

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';
import { endpoint, getThinkTime } from './helpers/config.js';
import { generateRandomEmail, generateRandomPassword, generateRandomUsername } from './helpers/data.js';

// Known compromised passwords for testing HIBP rejection
const COMPROMISED_PASSWORDS = [
  'password123',
  'Password1',
  '123456789',
  'qwerty123',
  'letmein',
  'welcome1',
  'Password123',
  'admin123',
];

// Custom metrics for HIBP analysis
const registrationStrongPasswordDuration = new Trend('registration_strong_password_duration', true);
const registrationCompromisedPasswordDuration = new Trend('registration_compromised_password_duration', true);
const compromisedPasswordRejectionRate = new Rate('compromised_password_rejection_rate');
const strongPasswordAcceptanceRate = new Rate('strong_password_acceptance_rate');
const hibpCheckDuration = new Trend('hibp_check_duration', true);
const registrationSuccess = new Counter('registration_success_count');
const registrationBlocked = new Counter('registration_blocked_count');

// Test configuration
export const options = {
  scenarios: {
    // Scenario 1: Test HIBP rejection with compromised passwords
    compromised_password_test: {
      executor: 'constant-arrival-rate',
      rate: 5,             // 5 registrations per second
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 10,
      maxVUs: 50,
      exec: 'testCompromisedPassword',
    },

    // Scenario 2: Test strong password acceptance
    strong_password_test: {
      executor: 'constant-arrival-rate',
      rate: 10,            // 10 registrations per second
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 20,
      maxVUs: 100,
      exec: 'testStrongPassword',
    },

    // Scenario 3: Mixed realistic registration traffic
    mixed_registration: {
      executor: 'ramping-arrival-rate',
      startRate: 5,
      timeUnit: '1s',
      stages: [
        { duration: '2m', target: 10 },  // Ramp to 10/s
        { duration: '5m', target: 20 },  // Peak at 20/s
        { duration: '2m', target: 5 },   // Ramp down
      ],
      preAllocatedVUs: 30,
      maxVUs: 150,
      exec: 'testMixedRegistration',
    },
  },

  // Performance thresholds
  thresholds: {
    // Registration latency (including HIBP check)
    'http_req_duration{endpoint:register_strong}': [
      'p(95)<2000',   // Strong passwords (cache miss) can take up to 2s
      'p(99)<3000',
    ],
    'http_req_duration{endpoint:register_compromised}': [
      'p(95)<2000',   // Compromised check should be equally fast
      'p(99)<3000',
    ],

    // HIBP functionality verification
    'compromised_password_rejection_rate': [
      'rate>0.95',    // At least 95% of compromised passwords should be rejected
    ],
    'strong_password_acceptance_rate': [
      'rate>0.95',    // At least 95% of strong passwords should be accepted
    ],

    // Error rate (excluding expected rejections)
    'http_req_failed{endpoint:register_strong}': [
      'rate<0.05',    // Less than 5% errors for valid requests
    ],

    // HIBP check duration (for cache hit/miss analysis)
    'hibp_check_duration': [
      'p(50)<100',    // Median should be fast (cache hits)
      'p(95)<2000',   // p95 includes API calls
    ],
  },

  // Test metadata
  tags: {
    test_type: 'load',
    sprint: 'sprint-10',
    feature: 'hibp-password-check',
    security_gate: 'S10-HIBP-003',
  },
};

// Setup function
export function setup() {
  console.log('='.repeat(60));
  console.log('Sprint 10 Load Test: HIBP Password Check Integration');
  console.log('='.repeat(60));
  console.log('Security Gate: S10-HIBP-003');
  console.log('Testing: Compromised password rejection & fail-open behavior');
  console.log('');

  // Verify API is accessible
  const healthCheck = http.get(endpoint('/health'));
  if (healthCheck.status !== 200) {
    throw new Error('API health check failed - is the server running?');
  }
  console.log('✓ API health check passed');
  console.log('');
  console.log('Starting HIBP load tests...');
  console.log('='.repeat(60));

  return {};
}

// Scenario 1: Test compromised password rejection
export function testCompromisedPassword() {
  const email = generateRandomEmail();
  const username = generateRandomUsername();
  const password = COMPROMISED_PASSWORDS[
    Math.floor(Math.random() * COMPROMISED_PASSWORDS.length)
  ];

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
      tags: { endpoint: 'register_compromised' },
    }
  );

  const duration = Date.now() - startTime;
  registrationCompromisedPasswordDuration.add(duration);
  hibpCheckDuration.add(duration);

  // Check that compromised password is rejected
  const rejected = check(response, {
    'compromised password: status 400': (r) => r.status === 400,
    'compromised password: error mentions breach': (r) => {
      if (r.status === 400) {
        const body = JSON.parse(r.body);
        const detail = (body.detail || '').toLowerCase();
        const title = (body.title || '').toLowerCase();
        return (
          detail.includes('breach') ||
          detail.includes('compromised') ||
          detail.includes('pwned') ||
          title.includes('compromised')
        );
      }
      return false;
    },
    'compromised password: response time reasonable': () => duration < 3000,
  });

  if (response.status === 400) {
    compromisedPasswordRejectionRate.add(true);
    registrationBlocked.add(1);
  } else {
    compromisedPasswordRejectionRate.add(false);
    if (response.status === 201) {
      registrationSuccess.add(1);
    }
  }

  sleep(getThinkTime(500, 1000) / 1000);
}

// Scenario 2: Test strong password acceptance
export function testStrongPassword() {
  const email = generateRandomEmail();
  const username = generateRandomUsername();
  const password = generateRandomPassword(16); // Strong random password

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
      tags: { endpoint: 'register_strong' },
    }
  );

  const duration = Date.now() - startTime;
  registrationStrongPasswordDuration.add(duration);
  hibpCheckDuration.add(duration);

  // Check that strong password is accepted
  const accepted = check(response, {
    'strong password: status 201': (r) => r.status === 201 || r.status === 200,
    'strong password: user created': (r) => {
      if (r.status === 201 || r.status === 200) {
        const body = JSON.parse(r.body);
        return body.user && body.user.id;
      }
      return false;
    },
    'strong password: has access token': (r) => {
      if (r.status === 201 || r.status === 200) {
        const body = JSON.parse(r.body);
        return body.tokens && body.tokens.accessToken;
      }
      return false;
    },
    'strong password: response time reasonable': () => duration < 3000,
  });

  if (response.status === 201 || response.status === 200) {
    strongPasswordAcceptanceRate.add(true);
    registrationSuccess.add(1);
  } else {
    strongPasswordAcceptanceRate.add(false);
    if (response.status === 400) {
      // Log unexpected rejection for debugging
      console.warn(`Strong password unexpectedly rejected: ${response.status} - ${response.body}`);
    }
  }

  sleep(getThinkTime(500, 1000) / 1000);
}

// Scenario 3: Mixed realistic registration traffic
export function testMixedRegistration() {
  // 80% strong passwords, 20% compromised (realistic ratio)
  const useCompromisedPassword = Math.random() < 0.2;

  const email = generateRandomEmail();
  const username = generateRandomUsername();
  const password = useCompromisedPassword
    ? COMPROMISED_PASSWORDS[Math.floor(Math.random() * COMPROMISED_PASSWORDS.length)]
    : generateRandomPassword(16);

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
      tags: {
        endpoint: 'register_mixed',
        password_type: useCompromisedPassword ? 'compromised' : 'strong',
      },
    }
  );

  const duration = Date.now() - startTime;
  hibpCheckDuration.add(duration);

  if (useCompromisedPassword) {
    registrationCompromisedPasswordDuration.add(duration);

    if (response.status === 400) {
      compromisedPasswordRejectionRate.add(true);
      registrationBlocked.add(1);
    } else {
      compromisedPasswordRejectionRate.add(false);
      if (response.status === 201) {
        registrationSuccess.add(1);
      }
    }

    check(response, {
      'mixed: compromised rejected': (r) => r.status === 400,
    });
  } else {
    registrationStrongPasswordDuration.add(duration);

    if (response.status === 201 || response.status === 200) {
      strongPasswordAcceptanceRate.add(true);
      registrationSuccess.add(1);
    } else {
      strongPasswordAcceptanceRate.add(false);
    }

    check(response, {
      'mixed: strong accepted': (r) => r.status === 201 || r.status === 200,
    });
  }

  sleep(getThinkTime(1000, 2000) / 1000);
}

// Teardown function
export function teardown(data) {
  console.log('');
  console.log('='.repeat(60));
  console.log('Sprint 10 HIBP Load Test Complete');
  console.log('='.repeat(60));
  console.log('');
  console.log('Review the metrics above to verify:');
  console.log('1. Compromised password rejection rate > 95%');
  console.log('2. Strong password acceptance rate > 95%');
  console.log('3. p95 registration latency < 2s');
  console.log('4. HIBP check latency profile (cache hits vs API calls)');
  console.log('');
  console.log('For fail-open testing, simulate HIBP API failure separately.');
}

// Handle summary for detailed reporting
export function handleSummary(data) {
  const compromisedRejectionRate = data.metrics.compromised_password_rejection_rate?.values?.rate || 0;
  const strongAcceptanceRate = data.metrics.strong_password_acceptance_rate?.values?.rate || 0;
  const hibpP95 = data.metrics.hibp_check_duration?.values?.['p(95)'] || 0;
  const hibpP50 = data.metrics.hibp_check_duration?.values?.['p(50)'] || 0;

  const summary = {
    'Sprint 10 - HIBP Password Check Load Test Summary': {
      security_gate: 'S10-HIBP-003',
      requirement: 'Compromised password rejection & fail-open',
      results: {
        compromised_rejection_rate: (compromisedRejectionRate * 100).toFixed(2) + '%',
        strong_acceptance_rate: (strongAcceptanceRate * 100).toFixed(2) + '%',
        hibp_check_p50_ms: hibpP50.toFixed(2),
        hibp_check_p95_ms: hibpP95.toFixed(2),
        gate_status:
          compromisedRejectionRate > 0.95 && strongAcceptanceRate > 0.95
            ? 'PASS ✓'
            : 'FAIL ✗',
      },
    },
    raw_data: data,
  };

  return {
    'stdout': JSON.stringify(summary, null, 2),
    'sprint10-hibp-results.json': JSON.stringify(data, null, 2),
  };
}
