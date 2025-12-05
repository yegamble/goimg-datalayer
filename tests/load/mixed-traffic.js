// mixed-traffic.js
// Mixed traffic load test with realistic traffic distribution
// Simulates realistic usage patterns with multiple concurrent flows

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import exec from 'k6/execution';
import { endpoint, getThinkTime, buildQueryParams } from './helpers/config.js';
import { setupAuthenticatedUsers, getRandomUser, getAuthHeaders, registerUser, login } from './helpers/auth.js';
import {
  generateImageTitle,
  generateImageDescription,
  generateTags,
  generateVisibility,
  generateTestImageData,
  createImageFormData,
  generateComment,
} from './helpers/data.js';

// Custom metrics for different user behaviors
const browsingSessionsCount = new Counter('browsing_sessions');
const uploadSessionsCount = new Counter('upload_sessions');
const socialSessionsCount = new Counter('social_sessions');
const authSessionsCount = new Counter('auth_sessions');

// Test configuration
export const options = {
  // Realistic traffic simulation with 150 total VUs over 15 minutes
  stages: [
    { duration: '2m', target: 38 },   // Ramp 0 → 25% (~38 VUs)
    { duration: '3m', target: 150 },  // Ramp 25% → 100% (150 VUs)
    { duration: '15m', target: 150 }, // Hold at 100% for test duration
    { duration: '1m', target: 0 },    // Ramp down
  ],

  // Performance thresholds
  thresholds: {
    // Standard endpoints
    'http_req_duration{type:read}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{type:write}': ['p(95)<500', 'p(99)<1000'],
    'http_req_duration{type:upload}': ['p(95)<5000', 'p(99)<10000'],

    // Overall error rate
    'http_req_failed': ['rate<0.01'], // Less than 1% errors

    // Specific endpoint groups
    'http_req_duration{endpoint:explore_recent}': ['p(95)<200'],
    'http_req_duration{endpoint:upload}': ['p(95)<5000'],
    'http_req_duration{endpoint:like_image}': ['p(95)<200'],
    'http_req_duration{endpoint:add_comment}': ['p(95)<200'],
  },

  tags: {
    test_type: 'load',
    flow: 'mixed',
  },
};

// Traffic distribution percentages (must sum to 100)
const TRAFFIC_DISTRIBUTION = {
  browsing: 60,    // 60% of users are browsing
  social: 25,      // 25% are performing social actions
  uploading: 10,   // 10% are uploading content
  auth: 5,         // 5% are registering/logging in
};

// Determine user scenario based on VU ID and iteration
function selectScenario() {
  const random = Math.random() * 100;

  if (random < TRAFFIC_DISTRIBUTION.browsing) {
    return 'browsing';
  } else if (random < TRAFFIC_DISTRIBUTION.browsing + TRAFFIC_DISTRIBUTION.social) {
    return 'social';
  } else if (random < TRAFFIC_DISTRIBUTION.browsing + TRAFFIC_DISTRIBUTION.social + TRAFFIC_DISTRIBUTION.uploading) {
    return 'uploading';
  } else {
    return 'auth';
  }
}

// Browsing scenario
function browsingScenario(user) {
  browsingSessionsCount.add(1);

  const isAuthenticated = user !== null;
  const headers = isAuthenticated ? getAuthHeaders(user.accessToken) : { 'Content-Type': 'application/json' };

  // Browse recent images
  const exploreParams = buildQueryParams({ page: 1, per_page: 20 });
  const exploreResponse = http.get(
    endpoint(`/explore/recent${exploreParams}`),
    {
      headers: headers,
      tags: { endpoint: 'explore_recent', type: 'read' },
    }
  );

  if (!check(exploreResponse, {
    'explore successful': (r) => r.status === 200,
  })) {
    return;
  }

  const images = JSON.parse(exploreResponse.body).items || [];
  sleep(getThinkTime(2000, 5000) / 1000);

  // View 2-4 image details
  const numToView = Math.min(Math.floor(Math.random() * 3) + 2, images.length);
  for (let i = 0; i < numToView; i++) {
    const image = images[i];

    http.get(
      endpoint(`/images/${image.id}`),
      {
        headers: headers,
        tags: { endpoint: 'get_image', type: 'read' },
      }
    );

    // Load medium variant
    http.get(
      endpoint(`/images/${image.id}/variants/medium`),
      {
        headers: headers,
        tags: { endpoint: 'get_variant', type: 'read' },
      }
    );

    sleep(getThinkTime(2000, 4000) / 1000);
  }
}

// Social interaction scenario
function socialScenario(user) {
  if (!user) {
    // Social actions require authentication, skip if no user
    return;
  }

  socialSessionsCount.add(1);
  const headers = getAuthHeaders(user.accessToken);

  // Get images to interact with
  const exploreParams = buildQueryParams({ page: 1, per_page: 10 });
  const exploreResponse = http.get(
    endpoint(`/explore/recent${exploreParams}`),
    { headers: headers, tags: { endpoint: 'explore_recent', type: 'read' } }
  );

  if (!check(exploreResponse, {
    'explore successful': (r) => r.status === 200,
  })) {
    return;
  }

  const images = JSON.parse(exploreResponse.body).items || [];
  if (images.length === 0) return;

  sleep(getThinkTime(1000, 3000) / 1000);

  // Like 1-3 images
  const numToLike = Math.min(Math.floor(Math.random() * 3) + 1, images.length);
  for (let i = 0; i < numToLike; i++) {
    http.post(
      endpoint(`/images/${images[i].id}/like`),
      null,
      {
        headers: headers,
        tags: { endpoint: 'like_image', type: 'write' },
      }
    );

    sleep(getThinkTime(1000, 3000) / 1000);
  }

  // Comment on 1 image
  if (images.length > 0) {
    const imageToComment = images[Math.floor(Math.random() * images.length)];
    const commentBody = { content: generateComment() };

    http.post(
      endpoint(`/images/${imageToComment.id}/comments`),
      JSON.stringify(commentBody),
      {
        headers: headers,
        tags: { endpoint: 'add_comment', type: 'write' },
      }
    );

    sleep(getThinkTime(2000, 4000) / 1000);
  }
}

// Upload scenario
function uploadScenario(user) {
  if (!user) {
    // Uploads require authentication
    return;
  }

  uploadSessionsCount.add(1);

  const imageData = generateTestImageData();
  const metadata = {
    title: generateImageTitle(),
    description: generateImageDescription(),
    visibility: generateVisibility(),
    tags: generateTags(5).join(','),
  };

  const formData = createImageFormData(imageData, metadata);

  const uploadResponse = http.post(
    endpoint('/images'),
    formData.body,
    {
      headers: {
        'Authorization': `Bearer ${user.accessToken}`,
        'Content-Type': formData.contentType,
      },
      tags: { endpoint: 'upload', type: 'upload' },
      timeout: '30s',
    }
  );

  check(uploadResponse, {
    'upload successful': (r) => r.status === 201,
  });

  sleep(getThinkTime(5000, 10000) / 1000);
}

// Authentication scenario
function authScenario() {
  authSessionsCount.add(1);

  // Register new user
  const user = registerUser();
  if (!user) {
    return;
  }

  sleep(getThinkTime(1000, 2000) / 1000);

  // Login
  const tokens = login(user.email, user.password);
  if (!tokens) {
    return;
  }

  sleep(getThinkTime(2000, 4000) / 1000);

  // Fetch profile
  http.get(
    endpoint(`/users/${user.id}`),
    {
      headers: getAuthHeaders(tokens.accessToken),
      tags: { endpoint: 'get_user', type: 'read' },
    }
  );
}

// Main test function
export default function (data) {
  const scenario = selectScenario();

  // Get user if needed (some scenarios work without authentication)
  let user = null;
  if (scenario !== 'auth' && (scenario === 'uploading' || scenario === 'social' || Math.random() < 0.7)) {
    user = getRandomUser(data.users);
  }

  // Execute selected scenario
  switch (scenario) {
    case 'browsing':
      browsingScenario(user);
      break;
    case 'social':
      socialScenario(user);
      break;
    case 'uploading':
      uploadScenario(user);
      break;
    case 'auth':
      authScenario();
      break;
  }

  // Think time before next iteration
  sleep(getThinkTime(1000, 5000) / 1000);
}

// Setup function
export function setup() {
  console.log('Starting mixed-traffic load test');
  console.log('Configuration: 150 VUs, 15 minutes duration');
  console.log('Traffic distribution:');
  console.log(`  - Browsing: ${TRAFFIC_DISTRIBUTION.browsing}%`);
  console.log(`  - Social: ${TRAFFIC_DISTRIBUTION.social}%`);
  console.log(`  - Uploading: ${TRAFFIC_DISTRIBUTION.uploading}%`);
  console.log(`  - Auth: ${TRAFFIC_DISTRIBUTION.auth}%`);

  // Verify API is accessible
  const healthCheck = http.get(endpoint('/health'));
  if (healthCheck.status !== 200) {
    throw new Error('API health check failed - is the server running?');
  }

  // Create pool of authenticated users (50% of max VUs)
  const users = setupAuthenticatedUsers(75);

  return { users };
}

export function teardown(data) {
  console.log('Mixed-traffic load test completed');
  console.log(`Total users created: ${data.users.length}`);
}
