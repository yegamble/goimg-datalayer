// Authentication helpers for k6 load tests
// Provides reusable functions for user registration, login, and token management

import http from 'k6/http';
import { check, sleep } from 'k6';
import { endpoint } from './config.js';
import { generateUsername, generateEmail, generatePassword } from './data.js';

// Register a new user
export function registerUser(username = null, email = null, password = null) {
  const user = {
    username: username || generateUsername(),
    email: email || generateEmail(),
    password: password || generatePassword(),
  };

  const response = http.post(
    endpoint('/auth/register'),
    JSON.stringify(user),
    {
      headers: { 'Content-Type': 'application/json' },
    }
  );

  const success = check(response, {
    'registration successful': (r) => r.status === 201,
    'user ID returned': (r) => {
      if (r.status === 201) {
        const body = JSON.parse(r.body);
        return body.id !== undefined;
      }
      return false;
    },
  });

  if (success) {
    const body = JSON.parse(response.body);
    return {
      id: body.id,
      username: user.username,
      email: user.email,
      password: user.password,
    };
  }

  // Log error if registration failed
  console.error(`Registration failed: ${response.status} - ${response.body}`);
  return null;
}

// Login with email and password
export function login(email, password) {
  const response = http.post(
    endpoint('/auth/login'),
    JSON.stringify({ email, password }),
    {
      headers: { 'Content-Type': 'application/json' },
    }
  );

  const success = check(response, {
    'login successful': (r) => r.status === 200,
    'access token returned': (r) => {
      if (r.status === 200) {
        const body = JSON.parse(r.body);
        return body.access_token !== undefined && body.refresh_token !== undefined;
      }
      return false;
    },
  });

  if (success) {
    const body = JSON.parse(response.body);
    return {
      accessToken: body.access_token,
      refreshToken: body.refresh_token,
      expiresIn: body.expires_in,
    };
  }

  console.error(`Login failed: ${response.status} - ${response.body}`);
  return null;
}

// Refresh access token using refresh token
export function refreshToken(refreshToken) {
  const response = http.post(
    endpoint('/auth/refresh'),
    JSON.stringify({ refresh_token: refreshToken }),
    {
      headers: { 'Content-Type': 'application/json' },
    }
  );

  const success = check(response, {
    'token refresh successful': (r) => r.status === 200,
    'new tokens returned': (r) => {
      if (r.status === 200) {
        const body = JSON.parse(r.body);
        return body.access_token !== undefined && body.refresh_token !== undefined;
      }
      return false;
    },
  });

  if (success) {
    const body = JSON.parse(response.body);
    return {
      accessToken: body.access_token,
      refreshToken: body.refresh_token,
      expiresIn: body.expires_in,
    };
  }

  return null;
}

// Logout (invalidate refresh token)
export function logout(accessToken) {
  const response = http.post(
    endpoint('/auth/logout'),
    null,
    {
      headers: {
        'Authorization': `Bearer ${accessToken}`,
      },
    }
  );

  check(response, {
    'logout successful': (r) => r.status === 204,
  });

  return response.status === 204;
}

// Get authorization headers with Bearer token
export function getAuthHeaders(accessToken) {
  return {
    'Authorization': `Bearer ${accessToken}`,
    'Content-Type': 'application/json',
  };
}

// Create a fully authenticated user (register + login)
export function createAuthenticatedUser() {
  const user = registerUser();
  if (!user) {
    return null;
  }

  // Small delay to avoid rate limiting
  sleep(0.1);

  const tokens = login(user.email, user.password);
  if (!tokens) {
    return null;
  }

  return {
    id: user.id,
    username: user.username,
    email: user.email,
    password: user.password,
    accessToken: tokens.accessToken,
    refreshToken: tokens.refreshToken,
    expiresIn: tokens.expiresIn,
  };
}

// Setup function to create multiple authenticated users for load testing
export function setupAuthenticatedUsers(count = 10) {
  console.log(`Setting up ${count} authenticated users...`);
  const users = [];

  for (let i = 0; i < count; i++) {
    const user = createAuthenticatedUser();
    if (user) {
      users.push(user);

      // Progress indicator
      if ((i + 1) % 10 === 0) {
        console.log(`Created ${i + 1}/${count} users`);
      }
    }

    // Throttle to avoid overwhelming the server during setup
    sleep(0.2);
  }

  console.log(`Setup complete: ${users.length} authenticated users created`);
  return users;
}

// Get a random user from the pool
export function getRandomUser(users) {
  if (!users || users.length === 0) {
    return null;
  }
  return users[Math.floor(Math.random() * users.length)];
}

// Verify token is still valid by making a test request
export function verifyToken(accessToken, userId) {
  const response = http.get(
    endpoint(`/users/${userId}`),
    {
      headers: getAuthHeaders(accessToken),
    }
  );

  return response.status === 200;
}
