# GoImg API E2E Tests - Postman Collection

## Overview

Comprehensive end-to-end test suite for the GoImg API, with a focus on Sprint 4 authentication and user management endpoints.

## Test Structure

### 1. Health Check
- **Liveness probe**: Validates basic API availability
- **Readiness probe**: Confirms all services (Postgres, Redis) are operational

### 2. Auth Tests (Sprint 4)

#### Registration Flow
- **Register - Success** (201): Creates new user with unique email/username
- **Register - Email Already Exists** (409): Validates duplicate email detection
- **Register - Username Already Exists** (409): Validates duplicate username detection
- **Register - Weak Password** (400/422): Validates password strength requirements

#### Login Flow
- **Login - Success** (200): Returns access token, refresh token, and user data
- **Login - Wrong Password** (401): Generic error message (security best practice)
- **Login - Unknown User** (401): Generic error message (security best practice)

#### Token Management
- **Access Protected Resource - Valid Token** (200): Verifies JWT authentication
- **Access Protected Resource - No Token** (401): Validates auth middleware
- **Refresh Token - Success** (200): Issues new token pair
- **Refresh Token - Invalid Token** (401): Rejects invalid/expired tokens

#### Logout Flow
- **Logout** (204): Blacklists current token
- **Access After Logout** (401): Confirms token blacklist enforcement

### 3. Users Tests (Sprint 4)

#### Profile Management
- **Get User by ID - Success** (200): Returns public profile (no email)
- **Get User by ID - Not Found** (404): RFC 7807 error response
- **Update Own Profile** (200): Allows users to modify their profile
- **Update Another User Profile - Forbidden** (403): Enforces ownership checks

#### Session Management
- **Get User Sessions** (200): Returns array of active sessions with metadata

#### Account Deletion
- **Delete Account - Wrong Password** (401): Requires password confirmation
- **Delete Account - Success** (204): Permanently removes user

### 4. E2E Flows

#### Complete Auth Journey
An 8-step integration test that validates the entire user lifecycle:
1. **Register** new user with unique credentials
2. **Login** and receive tokens
3. **Access Profile** with access token
4. **Update Profile** information
5. **Refresh Token** to get new token pair
6. **Verify Sessions** are tracked
7. **Logout** and blacklist token
8. **Verify Token Blacklisted** (401 on access attempt)

### 5. Images, Albums, Explore (Future Sprints)
Placeholder tests for future feature development.

### 6. Error Handling
- **401 - Unauthorized**: Missing or invalid auth
- **404 - Not Found**: Resource doesn't exist
- **422 - Validation Error**: Invalid input data

All error responses follow **RFC 7807 Problem Details** format.

## Key Features

### Idempotent Tests
Each test run generates unique test data using timestamps and random suffixes:
```javascript
const timestamp = Date.now();
const randomSuffix = Math.floor(Math.random() * 10000);
const uniqueId = `${timestamp}${randomSuffix}`;
pm.collectionVariables.set('testEmail', `e2e-test-${uniqueId}@example.com`);
```

### Security Validation
- Generic error messages for authentication failures (prevent user enumeration)
- Password never exposed in responses
- JWT token validation and blacklisting
- Authorization checks (users can't modify other users' data)

### Comprehensive Assertions
Each test validates:
- **Status code**: Expected HTTP response code
- **Response structure**: JSON schema validation
- **Business logic**: Correct data returned/modified
- **Headers**: X-Request-ID present, Content-Type correct
- **Security**: No sensitive data exposed (passwords, private fields)
- **RFC 7807 compliance**: Error responses follow standard format

### Test Dependencies
Tests are designed to run sequentially with proper state management:
- Tokens stored in collection variables after login
- User IDs captured after registration
- Pre-request scripts handle re-authentication when needed
- Separate variable sets for E2E flow isolation (`e2eEmail`, `e2eAccessToken`, etc.)

## Running Tests

### Local Execution
```bash
# With Newman CLI
newman run tests/e2e/postman/goimg-api.postman_collection.json \
  -e tests/e2e/postman/ci.postman_environment.json

# With Postman desktop
# 1. Import goimg-api.postman_collection.json
# 2. Import ci.postman_environment.json
# 3. Select CI environment
# 4. Run collection
```

### CI/CD Integration
```yaml
# .github/workflows/e2e-tests.yml
- name: Run E2E Tests
  run: |
    newman run tests/e2e/postman/goimg-api.postman_collection.json \
      -e tests/e2e/postman/ci.postman_environment.json \
      --reporters cli,json \
      --reporter-json-export newman-results.json
```

### Make Target
```bash
make test-e2e
```

## Environment Variables

### Collection Variables (Auto-generated)
- `testEmail`: Unique email per test run
- `testUsername`: Unique username per test run
- `testUserId`: User ID from registration
- `accessToken`: JWT access token
- `refreshToken`: JWT refresh token
- `loggedOutToken`: Token after logout (for blacklist test)
- `otherUserId`: Second user ID (for authorization tests)
- `testPassword`: Fixed secure password (defined in collection)

### Environment Variables (CI)
Defined in `ci.postman_environment.json`:
- `BASE_URL`: API base URL (default: `http://localhost:8080/api/v1`)
- `testImagePath`: Path to test image fixture
- `testImageId`: Image ID for CRUD tests
- `testAlbumId`: Album ID for CRUD tests

## Test Coverage

### Auth Endpoints
- ✅ POST `/auth/register` - Success (201)
- ✅ POST `/auth/register` - Duplicate email (409)
- ✅ POST `/auth/register` - Duplicate username (409)
- ✅ POST `/auth/register` - Weak password (400/422)
- ✅ POST `/auth/login` - Success (200)
- ✅ POST `/auth/login` - Wrong password (401)
- ✅ POST `/auth/login` - Unknown user (401)
- ✅ POST `/auth/refresh` - Success (200)
- ✅ POST `/auth/refresh` - Invalid token (401)
- ✅ POST `/auth/logout` - Success (204)

### User Endpoints
- ✅ GET `/users/{id}` - Success (200)
- ✅ GET `/users/{id}` - Not found (404)
- ✅ PUT `/users/{id}` - Success (200)
- ✅ PUT `/users/{id}` - Forbidden (403)
- ✅ DELETE `/users/{id}` - Wrong password (401)
- ✅ DELETE `/users/{id}` - Success (204)
- ✅ GET `/users/{id}/sessions` - Success (200)

### Protected Resource Access
- ✅ GET `/users/me` - With valid token (200)
- ✅ GET `/users/me` - Without token (401)
- ✅ GET `/users/me` - After logout (401)

## Edge Cases Covered

### Authentication
- ❌ Duplicate email registration
- ❌ Duplicate username registration
- ❌ Weak password (fails validation)
- ❌ Login with wrong password
- ❌ Login with non-existent user
- ❌ Access with expired/invalid token
- ❌ Access with blacklisted token (post-logout)
- ❌ Refresh with invalid token

### Authorization
- ❌ Update another user's profile
- ❌ Delete account without password confirmation
- ❌ Delete account with wrong password

### Data Validation
- ✅ Email format validation
- ✅ Username format validation
- ✅ Password strength requirements
- ✅ Required fields validation

### Security
- ✅ Generic auth error messages (no user enumeration)
- ✅ Password not exposed in responses
- ✅ Email hidden in public profiles
- ✅ Token blacklisting after logout
- ✅ WWW-Authenticate header on 401 responses

## Best Practices

### Test Isolation
- Each test generates unique data
- Tests don't depend on hardcoded IDs
- State is explicitly passed via collection variables
- E2E flow uses separate variable namespace

### Error Validation
All error tests verify:
1. Correct HTTP status code
2. RFC 7807 Problem Details structure (`type`, `title`, `status`, `detail`)
3. Appropriate error message content
4. X-Request-ID header presence

### Response Validation
All success tests verify:
1. Correct HTTP status code
2. Expected JSON structure
3. Data integrity (values match requests)
4. Required headers
5. No sensitive data leakage
6. Response time (where appropriate)

### Security Testing
- Token-based auth validation
- Authorization (ownership) checks
- Input validation and sanitization
- Error message information disclosure prevention

## Troubleshooting

### Test Failures

**Symptom**: 401 errors on authenticated requests
- **Cause**: Token expired or not set
- **Fix**: Check token is stored after login, verify JWT TTL

**Symptom**: 409 errors on registration
- **Cause**: Test data collision (rare but possible)
- **Fix**: Re-run tests; unique ID generation should resolve

**Symptom**: Pre-request script failures
- **Cause**: Base URL incorrect or API not running
- **Fix**: Verify `BASE_URL` in environment, start API server

**Symptom**: Tests pass individually but fail when run as collection
- **Cause**: Test execution order or state management issue
- **Fix**: Check collection variables are properly set/cleared

## Next Steps

### Future Enhancements
1. **Rate limiting tests**: Validate API throttling
2. **Email verification flow**: Test email confirmation endpoints
3. **Password reset flow**: Test forgot password functionality
4. **OAuth2 integration**: Test social login (Google, GitHub)
5. **2FA tests**: Multi-factor authentication validation
6. **Admin role tests**: RBAC enforcement for privileged operations
7. **Image upload tests**: File handling and virus scanning
8. **Pagination tests**: Large dataset handling

### Performance Testing
- Convert to k6 for load testing
- Establish baseline response times
- Identify bottlenecks under concurrent load

## References

- [Postman Collection Format v2.1](https://schema.getpostman.com/json/collection/v2.1.0/docs/)
- [Newman CLI Documentation](https://learning.postman.com/docs/running-collections/using-newman-cli/command-line-integration-with-newman/)
- [RFC 7807: Problem Details for HTTP APIs](https://datatracker.ietf.org/doc/html/rfc7807)
- [JWT Best Practices](https://datatracker.ietf.org/doc/html/rfc8725)

## Maintenance

This test suite should be updated whenever:
- New API endpoints are added
- Endpoint behavior changes
- New error conditions are introduced
- Authentication/authorization logic changes
- Response schemas are modified

**Owned by**: Test Strategist
**Last updated**: 2025-12-03
**Version**: 1.0.0 (Sprint 4)
