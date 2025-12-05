// social-flow.js
// Load test for social interactions flow
// Tests: Browse images → Like image → Comment → Follow user → View feed

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { endpoint, getThinkTime, buildQueryParams } from './helpers/config.js';
import { setupAuthenticatedUsers, getRandomUser, getAuthHeaders } from './helpers/auth.js';
import { generateComment } from './helpers/data.js';

// Custom metrics
const socialFlowDuration = new Trend('social_flow_duration', true);
const likesCount = new Counter('likes_count');
const commentsCount = new Counter('comments_count');
const unlikesCount = new Counter('unlikes_count');

// Test configuration
export const options = {
  // Staged ramp-up with 75 concurrent VUs over 10 minutes
  stages: [
    { duration: '2m', target: 19 },  // Ramp 0 → 25% (~19 VUs)
    { duration: '3m', target: 75 },  // Ramp 25% → 100% (75 VUs)
    { duration: '10m', target: 75 }, // Hold at 100% for test duration
    { duration: '1m', target: 0 },   // Ramp down
  ],

  // Performance thresholds
  thresholds: {
    'http_req_duration{endpoint:explore_recent}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:like_image}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:unlike_image}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:add_comment}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:list_comments}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:user_likes}': ['p(95)<200', 'p(99)<500'],

    // Error rate
    'http_req_failed': ['rate<0.01'], // Less than 1% errors

    // Flow duration
    'social_flow_duration': ['p(95)<8000'], // Entire flow under 8s at p95
  },

  tags: {
    test_type: 'load',
    flow: 'social',
  },
};

// Main test function
export default function (data) {
  const flowStart = Date.now();

  // Get authenticated user
  const user = getRandomUser(data.users);
  if (!user) {
    console.error('No authenticated user available');
    sleep(5);
    return;
  }

  const headers = getAuthHeaders(user.accessToken);

  // Step 1: Browse recent images
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

  // Think time (user scrolling)
  sleep(getThinkTime(1000, 4000) / 1000);

  // Step 2: Like 2-4 random images
  const numImagesToLike = Math.floor(Math.random() * 3) + 2; // 2-4 images
  const imagesToLike = images.slice(0, Math.min(numImagesToLike, images.length));
  const likedImages = [];

  for (const image of imagesToLike) {
    const likeResponse = http.post(
      endpoint(`/images/${image.id}/like`),
      null,
      {
        headers: headers,
        tags: { endpoint: 'like_image' },
      }
    );

    if (check(likeResponse, {
      'like successful': (r) => r.status === 200,
      'like count updated': (r) => {
        if (r.status === 200) {
          const body = JSON.parse(r.body);
          return body.liked === true && body.like_count !== undefined;
        }
        return false;
      },
    })) {
      likesCount.add(1);
      likedImages.push(image.id);
    }

    // Think time between likes
    sleep(getThinkTime(1000, 4000) / 1000);
  }

  // Step 3: Comment on 1-2 images
  const numImagesToComment = Math.floor(Math.random() * 2) + 1; // 1-2 images
  const imagesToComment = images.slice(0, Math.min(numImagesToComment, images.length));

  for (const image of imagesToComment) {
    const commentBody = {
      content: generateComment(),
    };

    const commentResponse = http.post(
      endpoint(`/images/${image.id}/comments`),
      JSON.stringify(commentBody),
      {
        headers: headers,
        tags: { endpoint: 'add_comment' },
      }
    );

    if (check(commentResponse, {
      'comment added': (r) => r.status === 201,
      'comment has ID': (r) => {
        if (r.status === 201) {
          const body = JSON.parse(r.body);
          return body.id !== undefined;
        }
        return false;
      },
    })) {
      commentsCount.add(1);
    }

    // Think time after commenting
    sleep(getThinkTime(2000, 5000) / 1000);
  }

  // Step 4: View own liked images
  const userLikesParams = buildQueryParams({
    page: 1,
    per_page: 20,
  });

  const userLikesResponse = http.get(
    endpoint(`/users/${user.id}/likes${userLikesParams}`),
    {
      headers: headers,
      tags: { endpoint: 'user_likes' },
    }
  );

  check(userLikesResponse, {
    'user likes fetched': (r) => r.status === 200,
  });

  // Think time (user viewing their likes)
  sleep(getThinkTime(2000, 4000) / 1000);

  // Step 5: Unlike one previously liked image (simulate changing mind)
  if (likedImages.length > 0) {
    const imageToUnlike = likedImages[Math.floor(Math.random() * likedImages.length)];

    const unlikeResponse = http.del(
      endpoint(`/images/${imageToUnlike}/like`),
      null,
      {
        headers: headers,
        tags: { endpoint: 'unlike_image' },
      }
    );

    if (check(unlikeResponse, {
      'unlike successful': (r) => r.status === 200,
      'like status updated': (r) => {
        if (r.status === 200) {
          const body = JSON.parse(r.body);
          return body.liked === false;
        }
        return false;
      },
    })) {
      unlikesCount.add(1);
    }
  }

  // Record total flow duration
  const flowDuration = Date.now() - flowStart;
  socialFlowDuration.add(flowDuration);

  // Think time before next iteration
  sleep(getThinkTime(1000, 4000) / 1000);
}

// Setup function
export function setup() {
  console.log('Starting social-flow load test');
  console.log('Configuration: 75 VUs, 10 minutes duration');

  // Verify API is accessible
  const healthCheck = http.get(endpoint('/health'));
  if (healthCheck.status !== 200) {
    throw new Error('API health check failed - is the server running?');
  }

  // Create pool of authenticated users (one per VU plus buffer)
  const users = setupAuthenticatedUsers(80);

  return { users };
}

export function teardown(data) {
  console.log('Social-flow load test completed');
  console.log(`Total users created: ${data.users.length}`);
}
