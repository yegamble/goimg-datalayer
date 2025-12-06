// browse-flow.js
// Load test for image browsing flow
// Tests: List recent images → View image details → Load variants → Fetch comments

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { endpoint, getThinkTime, buildQueryParams } from './helpers/config.js';
import { setupAuthenticatedUsers, getRandomUser, getAuthHeaders } from './helpers/auth.js';

// Custom metrics
const browseFlowDuration = new Trend('browse_flow_duration', true);
const imagesViewedCount = new Counter('images_viewed_count');
const variantsLoadedCount = new Counter('variants_loaded_count');
const commentsViewedCount = new Counter('comments_viewed_count');

// Test configuration
export const options = {
  // Staged ramp-up with 100 concurrent VUs over 10 minutes
  stages: [
    { duration: '2m', target: 25 },  // Ramp 0 → 25% (25 VUs)
    { duration: '3m', target: 100 }, // Ramp 25% → 100% (100 VUs)
    { duration: '10m', target: 100 }, // Hold at 100% for test duration
    { duration: '1m', target: 0 },   // Ramp down
  ],

  // Performance thresholds
  thresholds: {
    'http_req_duration{endpoint:list_images}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:get_image}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:get_variant}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:list_comments}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:explore_recent}': ['p(95)<200', 'p(99)<500'],

    // Error rate
    'http_req_failed': ['rate<0.01'], // Less than 1% errors

    // Flow duration
    'browse_flow_duration': ['p(95)<5000'], // Entire flow under 5s at p95
  },

  tags: {
    test_type: 'load',
    flow: 'browsing',
  },
};

// Main test function
export default function (data) {
  const flowStart = Date.now();

  // Some users are authenticated, some are anonymous (70/30 split)
  const isAuthenticated = Math.random() < 0.7;
  const user = isAuthenticated ? getRandomUser(data.users) : null;
  const headers = user ? getAuthHeaders(user.accessToken) : { 'Content-Type': 'application/json' };

  // Step 1: Browse recent public images
  const exploreParams = buildQueryParams({
    page: 1,
    per_page: 20,
  });

  const exploreResponse = http.get(
    endpoint(`/explore/recent${exploreParams}`),
    {
      headers: headers,
      tags: { endpoint: 'explore_recent' },
    }
  );

  const exploreSuccess = check(exploreResponse, {
    'explore recent successful': (r) => r.status === 200,
    'images returned': (r) => {
      if (r.status === 200) {
        const body = JSON.parse(r.body);
        return body.items && Array.isArray(body.items) && body.items.length > 0;
      }
      return false;
    },
  });

  if (!exploreSuccess) {
    console.error('Failed to fetch recent images');
    sleep(5);
    return;
  }

  const images = JSON.parse(exploreResponse.body).items;

  // Think time (user scrolling through images)
  sleep(getThinkTime(2000, 5000) / 1000);

  // Step 2: View 3-5 random image details
  const numImagesToView = Math.floor(Math.random() * 3) + 3; // 3-5 images
  const imagesToView = images.slice(0, Math.min(numImagesToView, images.length));

  for (const image of imagesToView) {
    // Get image details
    const imageResponse = http.get(
      endpoint(`/images/${image.id}`),
      {
        headers: headers,
        tags: { endpoint: 'get_image' },
      }
    );

    check(imageResponse, {
      'image details loaded': (r) => r.status === 200,
      'image has variants': (r) => {
        if (r.status === 200) {
          const body = JSON.parse(r.body);
          return body.variants !== undefined;
        }
        return false;
      },
    });

    if (imageResponse.status === 200) {
      imagesViewedCount.add(1);

      // Step 3: Load image variants (simulate viewing different sizes)
      const variantsToLoad = ['thumbnail', 'medium'];
      for (const size of variantsToLoad) {
        const variantResponse = http.get(
          endpoint(`/images/${image.id}/variants/${size}`),
          {
            headers: headers,
            tags: { endpoint: 'get_variant' },
          }
        );

        if (check(variantResponse, {
          'variant loaded': (r) => r.status === 200,
        })) {
          variantsLoadedCount.add(1);
        }
      }

      // Step 4: Load comments (if available)
      const commentsParams = buildQueryParams({
        page: 1,
        per_page: 10,
      });

      const commentsResponse = http.get(
        endpoint(`/images/${image.id}/comments${commentsParams}`),
        {
          headers: headers,
          tags: { endpoint: 'list_comments' },
        }
      );

      if (check(commentsResponse, {
        'comments loaded': (r) => r.status === 200,
      })) {
        commentsViewedCount.add(1);
      }
    }

    // Think time between viewing images
    sleep(getThinkTime(2000, 5000) / 1000);
  }

  // Record total flow duration
  const flowDuration = Date.now() - flowStart;
  browseFlowDuration.add(flowDuration);

  // Think time before next iteration
  sleep(getThinkTime(1000, 3000) / 1000);
}

// Setup function - create authenticated users for the test
export function setup() {
  console.log('Starting browse-flow load test');
  console.log('Configuration: 100 VUs, 10 minutes duration');

  // Verify API is accessible
  const healthCheck = http.get(endpoint('/health'));
  if (healthCheck.status !== 200) {
    throw new Error('API health check failed - is the server running?');
  }

  // Create pool of authenticated users (30% of max VUs)
  const users = setupAuthenticatedUsers(30);

  return { users };
}

export function teardown(data) {
  console.log('Browse-flow load test completed');
  console.log(`Total users created: ${data.users.length}`);
}
