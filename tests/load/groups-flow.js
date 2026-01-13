// groups-flow.js
// Load test for Sprint 20 Groups/Communities feature
// Tests: List groups → Get group details → List members → Search groups → List albums
// Target: Group list query < 200ms at p95 for 1000 groups

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { endpoint, getThinkTime, buildQueryParams } from './helpers/config.js';
import { setupAuthenticatedUsers, getRandomUser, getAuthHeaders } from './helpers/auth.js';

// Custom metrics for Sprint 20
const groupsFlowDuration = new Trend('groups_flow_duration', true);
const groupsViewedCount = new Counter('groups_viewed_count');
const groupsSearchedCount = new Counter('groups_searched_count');
const membersListedCount = new Counter('members_listed_count');
const albumsListedCount = new Counter('albums_listed_count');

// List groups response time (critical metric for Sprint 20)
const listGroupsP95 = new Trend('list_groups_p95', true);

// Test configuration
export const options = {
  // Staged ramp-up with 100 concurrent VUs over 5 minutes
  stages: [
    { duration: '1m', target: 25 },   // Ramp 0 → 25% (25 VUs)
    { duration: '2m', target: 100 },  // Ramp 25% → 100% (100 VUs)
    { duration: '5m', target: 100 },  // Hold at 100% for test duration
    { duration: '1m', target: 0 },    // Ramp down
  ],

  // Performance thresholds - Sprint 20 requirements
  thresholds: {
    // CRITICAL: Group list query < 200ms at p95 (Sprint 20 requirement)
    'http_req_duration{endpoint:list_groups}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:get_group}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:list_members}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:search_groups}': ['p(95)<300', 'p(99)<600'],
    'http_req_duration{endpoint:list_albums}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:get_album}': ['p(95)<200', 'p(99)<500'],

    // Error rate
    'http_req_failed': ['rate<0.01'], // Less than 1% errors

    // Flow duration
    'groups_flow_duration': ['p(95)<3000'], // Entire flow under 3s at p95
  },

  tags: {
    test_type: 'load',
    flow: 'groups',
    sprint: '20',
  },
};

// Main test function
export default function (data) {
  const flowStart = Date.now();

  // Most users are authenticated for groups (90/10 split)
  const isAuthenticated = Math.random() < 0.9;
  const user = isAuthenticated ? getRandomUser(data.users) : null;
  const headers = user ? getAuthHeaders(user.accessToken) : { 'Content-Type': 'application/json' };

  // Step 1: List public groups (CRITICAL - must be < 200ms)
  const listParams = buildQueryParams({
    page: 1,
    per_page: 20,
    group_type: 'public',
  });

  const listStart = Date.now();
  const listResponse = http.get(
    endpoint(`/groups${listParams}`),
    {
      headers: headers,
      tags: { endpoint: 'list_groups' },
    }
  );
  listGroupsP95.add(Date.now() - listStart);

  const listSuccess = check(listResponse, {
    'list groups successful': (r) => r.status === 200,
    'groups returned': (r) => {
      if (r.status === 200) {
        const body = JSON.parse(r.body);
        return body.groups && Array.isArray(body.groups);
      }
      return false;
    },
    'list groups < 200ms': (r) => r.timings.duration < 200,
  });

  if (!listSuccess) {
    console.error('Failed to list groups');
    sleep(2);
    return;
  }

  const groups = JSON.parse(listResponse.body).groups;

  // Think time (user browsing groups)
  sleep(getThinkTime(1000, 3000) / 1000);

  // Step 2: View 2-3 random group details
  const numGroupsToView = Math.min(Math.floor(Math.random() * 2) + 2, groups.length);
  const groupsToView = groups.slice(0, numGroupsToView);

  for (const group of groupsToView) {
    // Get group details
    const groupResponse = http.get(
      endpoint(`/groups/${group.id}`),
      {
        headers: headers,
        tags: { endpoint: 'get_group' },
      }
    );

    check(groupResponse, {
      'group details loaded': (r) => r.status === 200,
      'group has member_count': (r) => {
        if (r.status === 200) {
          const body = JSON.parse(r.body);
          return body.member_count !== undefined;
        }
        return false;
      },
    });

    if (groupResponse.status === 200) {
      groupsViewedCount.add(1);

      // Step 3: List group members
      const membersParams = buildQueryParams({
        page: 1,
        per_page: 20,
      });

      const membersResponse = http.get(
        endpoint(`/groups/${group.id}/members${membersParams}`),
        {
          headers: headers,
          tags: { endpoint: 'list_members' },
        }
      );

      if (check(membersResponse, {
        'members listed': (r) => r.status === 200,
      })) {
        membersListedCount.add(1);
      }

      // Step 4: List group albums (Sprint 20 feature)
      const albumsParams = buildQueryParams({
        page: 1,
        per_page: 10,
      });

      const albumsResponse = http.get(
        endpoint(`/groups/${group.id}/albums${albumsParams}`),
        {
          headers: headers,
          tags: { endpoint: 'list_albums' },
        }
      );

      if (check(albumsResponse, {
        'albums listed': (r) => r.status === 200,
      })) {
        albumsListedCount.add(1);

        // If albums exist, view one
        if (albumsResponse.status === 200) {
          const albumsBody = JSON.parse(albumsResponse.body);
          if (albumsBody.albums && albumsBody.albums.length > 0) {
            const album = albumsBody.albums[0];
            const albumResponse = http.get(
              endpoint(`/groups/${group.id}/albums/${album.id}`),
              {
                headers: headers,
                tags: { endpoint: 'get_album' },
              }
            );

            check(albumResponse, {
              'album details loaded': (r) => r.status === 200,
            });
          }
        }
      }
    }

    // Think time between viewing groups
    sleep(getThinkTime(1000, 3000) / 1000);
  }

  // Step 5: Search groups
  const searchTerms = ['photography', 'nature', 'travel', 'art', 'test'];
  const searchTerm = searchTerms[Math.floor(Math.random() * searchTerms.length)];

  const searchParams = buildQueryParams({
    q: searchTerm,
    page: 1,
    per_page: 20,
  });

  const searchResponse = http.get(
    endpoint(`/groups/search${searchParams}`),
    {
      headers: headers,
      tags: { endpoint: 'search_groups' },
    }
  );

  if (check(searchResponse, {
    'search successful': (r) => r.status === 200,
    'search < 300ms': (r) => r.timings.duration < 300,
  })) {
    groupsSearchedCount.add(1);
  }

  // Record total flow duration
  const flowDuration = Date.now() - flowStart;
  groupsFlowDuration.add(flowDuration);

  // Think time before next iteration
  sleep(getThinkTime(1000, 2000) / 1000);
}

// Setup function - create authenticated users and verify API
export function setup() {
  console.log('Starting groups-flow load test (Sprint 20)');
  console.log('Configuration: 100 VUs, 5 minutes duration');
  console.log('Target: Group list query < 200ms at p95');

  // Verify API is accessible
  const healthCheck = http.get(endpoint('/health'));
  if (healthCheck.status !== 200) {
    throw new Error('API health check failed - is the server running?');
  }

  // Create pool of authenticated users (50% of max VUs)
  const users = setupAuthenticatedUsers(50);

  console.log(`Created ${users.length} authenticated users for testing`);

  return { users };
}

export function teardown(data) {
  console.log('Groups-flow load test completed (Sprint 20)');
  console.log(`Total users created: ${data.users.length}`);
  console.log('Check thresholds for Sprint 20 performance requirements');
}
