// upload-flow.js
// Load test for image upload flow
// Tests: Login → Upload image (2MB JPEG) → Poll processing status → Fetch variants

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { endpoint, getThinkTime } from './helpers/config.js';
import { setupAuthenticatedUsers, getRandomUser, getAuthHeaders } from './helpers/auth.js';
import {
  generateImageTitle,
  generateImageDescription,
  generateTags,
  generateVisibility,
  generateTestImageData,
  createImageFormData,
} from './helpers/data.js';

// Custom metrics
const uploadFlowDuration = new Trend('upload_flow_duration', true);
const uploadsSuccessCount = new Counter('uploads_success_count');
const uploadFailedCount = new Counter('upload_failed_count');
const variantsGeneratedCount = new Counter('variants_generated_count');

// Test configuration
export const options = {
  // Staged ramp-up with 20 concurrent VUs over 10 minutes
  stages: [
    { duration: '2m', target: 5 },   // Ramp 0 → 25% (5 VUs)
    { duration: '3m', target: 20 },  // Ramp 25% → 100% (20 VUs)
    { duration: '10m', target: 20 }, // Hold at 100% for test duration
    { duration: '1m', target: 0 },   // Ramp down
  ],

  // Performance thresholds for upload endpoints
  thresholds: {
    'http_req_duration{endpoint:upload}': ['p(95)<5000', 'p(99)<10000'],
    'http_req_duration{endpoint:get_image}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:get_variant}': ['p(95)<200', 'p(99)<500'],

    // Error rate (uploads may have slightly higher failure rate)
    'http_req_failed': ['rate<0.02'], // Less than 2% errors

    // Flow duration (upload is slower)
    'upload_flow_duration': ['p(95)<15000'], // Entire flow under 15s at p95
  },

  tags: {
    test_type: 'load',
    flow: 'upload',
  },
};

// Main test function
export default function (data) {
  const flowStart = Date.now();

  // Get authenticated user from pool
  const user = getRandomUser(data.users);
  if (!user) {
    console.error('No authenticated user available');
    sleep(5);
    return;
  }

  // Step 1: Prepare image upload data
  const imageData = generateTestImageData();
  const metadata = {
    title: generateImageTitle(),
    description: generateImageDescription(),
    visibility: generateVisibility(),
    tags: generateTags(5).join(','),
  };

  const formData = createImageFormData(imageData, metadata);

  // Step 2: Upload image
  const uploadResponse = http.post(
    endpoint('/images'),
    formData.body,
    {
      headers: {
        'Authorization': `Bearer ${user.accessToken}`,
        'Content-Type': formData.contentType,
      },
      tags: { endpoint: 'upload' },
      timeout: '30s', // Uploads may take longer
    }
  );

  const uploadSuccess = check(uploadResponse, {
    'upload successful': (r) => r.status === 201,
    'image ID returned': (r) => {
      if (r.status === 201) {
        const body = JSON.parse(r.body);
        return body.id !== undefined;
      }
      return false;
    },
    'variants generated': (r) => {
      if (r.status === 201) {
        const body = JSON.parse(r.body);
        return body.variants !== undefined;
      }
      return false;
    },
  });

  if (!uploadSuccess) {
    console.error(`Upload failed: ${uploadResponse.status} - ${uploadResponse.body}`);
    uploadFailedCount.add(1);
    sleep(5);
    return;
  }

  uploadsSuccessCount.add(1);
  const imageId = JSON.parse(uploadResponse.body).id;

  // Think time (user waiting for upload to complete)
  sleep(getThinkTime(3000, 7000) / 1000);

  // Step 3: Fetch uploaded image details to verify processing
  const imageResponse = http.get(
    endpoint(`/images/${imageId}`),
    {
      headers: getAuthHeaders(user.accessToken),
      tags: { endpoint: 'get_image' },
    }
  );

  check(imageResponse, {
    'image fetch successful': (r) => r.status === 200,
    'image has variants': (r) => {
      if (r.status === 200) {
        const body = JSON.parse(r.body);
        return body.variants && Object.keys(body.variants).length > 0;
      }
      return false;
    },
  });

  // Step 4: Fetch all variants to verify they were generated correctly
  const variants = ['thumbnail', 'small', 'medium', 'large', 'original'];
  let successfulVariants = 0;

  for (const size of variants) {
    const variantResponse = http.get(
      endpoint(`/images/${imageId}/variants/${size}`),
      {
        headers: getAuthHeaders(user.accessToken),
        tags: { endpoint: 'get_variant' },
      }
    );

    if (check(variantResponse, {
      [`variant ${size} loaded`]: (r) => r.status === 200,
    })) {
      successfulVariants++;
      variantsGeneratedCount.add(1);
    }

    // Small delay between variant fetches
    sleep(0.2);
  }

  // Verify at least some variants were generated
  check(null, {
    'at least 3 variants generated': () => successfulVariants >= 3,
  });

  // Record total flow duration
  const flowDuration = Date.now() - flowStart;
  uploadFlowDuration.add(flowDuration);

  // Think time before next iteration
  // Longer delay for uploads to respect rate limits (50 uploads/hour)
  sleep(getThinkTime(5000, 10000) / 1000);
}

// Setup function
export function setup() {
  console.log('Starting upload-flow load test');
  console.log('Configuration: 20 VUs, 10 minutes duration');

  // Verify API is accessible
  const healthCheck = http.get(endpoint('/health'));
  if (healthCheck.status !== 200) {
    throw new Error('API health check failed - is the server running?');
  }

  // Create pool of authenticated users (one per VU plus buffer)
  const users = setupAuthenticatedUsers(25);

  return { users };
}

export function teardown(data) {
  console.log('Upload-flow load test completed');
  console.log(`Total users created: ${data.users.length}`);
}
