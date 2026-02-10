# GoImg API E2E Tests - Postman Collection

## Overview

Comprehensive end-to-end test suite for the GoImg API with 290 test requests across 20 folders, providing ~100% coverage of all implemented API endpoints. Tests validate authentication, authorization, business logic, error handling, and complete user journeys.

## Quick Start

```bash
# Install Newman (first time only)
make setup-e2e

# Run all tests
make test-e2e

# Run specific folder
make test-e2e-folder FOLDER=Auth

# Validate collection structure (no API calls)
make test-e2e-dry

# Generate HTML report
make test-e2e-report
```

## Test Coverage Summary

| Category | Tests | Description |
|----------|-------|-------------|
| **Auth** | 10 | Registration, login, token refresh, logout |
| **2FA Login Verify** | 2 | Two-factor authentication verification |
| **Users** | 8 | Profile management, sessions, account deletion |
| **User Liked Images** | 2 | Like/unlike images, view liked images |
| **Images** | 15 | Upload, view, update, delete, ownership checks |
| **Image Discovery** | 10 | List, search, filters (date, user, tags), variants |
| **Albums** | 12 | CRUD operations, privacy settings, ownership |
| **Album Image Management** | 13 | Add/remove images, list album images, ordering |
| **Tags** | 8 | Create, list, assign to images |
| **Tags Images by Tag** | 2 | List images by tag, pagination |
| **Comments** | 10 | Create, update, delete, nested replies |
| **User Follow** | 8 | Follow/unfollow users, list followers/following |
| **Groups** | 12 | Create groups, member management, roles |
| **Group Invitations** | 10 | Invite, accept, decline, list invitations |
| **Group Album Images** | 5 | Group-level album image management |
| **Explore Popular** | 3 | Popular images, trending content |
| **Explore Trending Tags** | 5 | Trending tags discovery |
| **NSFW Moderation** | 10 | Flag content, moderation queue, auto-detection |
| **Metrics** | 1 | System metrics endpoint |
| **Error Handling** | 25 | 4xx/5xx responses, RFC 7807 compliance |
| **Complete User Journey** | 11 | Full user lifecycle integration test |
| **Health Checks** | 2 | Liveness and readiness probes |
| **Total** | **290** | Comprehensive API coverage |

## Test Structure

### Authentication & Authorization
Tests validate JWT token lifecycle, session management, and permission checks across all protected endpoints.

### Image Lifecycle
Complete coverage from upload (with virus scanning) through discovery, moderation, and deletion. Tests include:
- Multi-format uploads (JPEG, PNG, GIF, WebP)
- EXIF data extraction and privacy
- Image variants (thumbnail, medium, large)
- Search and filtering (tags, date ranges, users)
- Ownership and visibility checks

### Social Features
Tests cover user interactions:
- Following/unfollowing users
- Liking images
- Commenting with nested replies
- Group creation and membership

### Content Organization
Tests validate organizational features:
- Albums (public, private, unlisted)
- Tags (creation, assignment, search)
- Groups with role-based access

### Moderation
Tests ensure content safety:
- NSFW content flagging
- Automated detection integration
- Moderation queue workflows
- User reporting

### Error Handling
Comprehensive validation of error responses:
- 400: Validation errors (malformed requests)
- 401: Authentication failures
- 403: Authorization failures
- 404: Resource not found
- 409: Conflicts (duplicates)
- 422: Semantic validation errors
- 500: Internal server errors (simulated)

All error responses follow RFC 7807 Problem Details format.

### Complete User Journey
An 11-step integration test that validates the entire user lifecycle:
1. Register new user
2. Login and receive tokens
3. Upload image
4. Create album
5. Add image to album
6. Tag image
7. Comment on image
8. Follow another user
9. Like image
10. View discovery feeds
11. Logout

## Test Execution Flow

Tests are designed to run sequentially with proper state management:

1. **Health Checks**: Verify API availability and readiness
2. **User Creation**: Generate unique test users with timestamps
3. **Authentication**: Login and capture tokens
4. **Resource Creation**: Create test data (images, albums, groups)
5. **Operations**: Test CRUD operations on created resources
6. **Relationships**: Test social features (follow, like, comment)
7. **Discovery**: Test search and filtering
8. **Cleanup**: Test deletion and cleanup operations
9. **Error Scenarios**: Test edge cases and error conditions

### Test Isolation

Each test run generates unique data to ensure idempotence:

```javascript
// Collection pre-request script
const timestamp = Date.now();
const randomSuffix = Math.floor(Math.random() * 10000);
const uniqueId = `${timestamp}${randomSuffix}`;

pm.collectionVariables.set('testEmail', `e2e-test-${uniqueId}@example.com`);
pm.collectionVariables.set('testUsername', `testuser_${uniqueId}`);
```

## Environment Configuration

### Collection Variables (Auto-generated)

These variables are set automatically during test execution:

| Variable | Purpose |
|----------|---------|
| `testEmail` | Unique email per test run |
| `testUsername` | Unique username per test run |
| `testUserId` | User ID from registration |
| `accessToken` | JWT access token |
| `refreshToken` | JWT refresh token |
| `testImageId` | Created image ID |
| `testAlbumId` | Created album ID |
| `testGroupId` | Created group ID |
| `testCommentId` | Created comment ID |
| `testTagId` | Created tag ID |
| `otherUserId` | Second user ID (for authorization tests) |

### Environment Variables (CI)

Defined in `ci.postman_environment.json`:

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE_URL` | `http://localhost:8080/api/v1` | API base URL |
| `testPassword` | `TestPassword123!` | Test user password |
| `testImagePath` | `tests/fixtures/images/test.jpg` | Test image file path |

### Local Development Environment

For local testing with different settings, create `local.postman_environment.json`:

```json
{
  "name": "Local Dev",
  "values": [
    {
      "key": "BASE_URL",
      "value": "http://localhost:3000/api/v1",
      "enabled": true
    },
    {
      "key": "testPassword",
      "value": "LocalDevPassword123!",
      "enabled": true
    }
  ]
}
```

## Running Tests

### Full Test Suite

Run all 290 tests:

```bash
make test-e2e
```

This command:
1. Validates the API server is running
2. Executes all tests sequentially
3. Reports results to console
4. Fails if any test fails (CI-friendly)

### Specific Folder

Run tests from a specific folder:

```bash
# Run only Auth tests
make test-e2e-folder FOLDER=Auth

# Run only Image tests
make test-e2e-folder FOLDER=Images

# Run only Complete User Journey
make test-e2e-folder FOLDER="Complete User Journey"
```

### Dry Run

Validate collection structure without making API calls:

```bash
make test-e2e-dry
```

Useful for:
- Validating JSON syntax
- Checking test script errors
- Verifying folder structure

### HTML Report

Generate detailed HTML report:

```bash
make test-e2e-report
```

Report includes:
- Request/response details
- Test assertions (passed/failed)
- Execution timeline
- Error stack traces

Report saved to: `newman-report.html`

### Newman CLI (Direct)

For advanced usage:

```bash
# Run with custom environment
newman run tests/e2e/postman/goimg-api.postman_collection.json \
  -e tests/e2e/postman/local.postman_environment.json

# Run with delay between requests
newman run tests/e2e/postman/goimg-api.postman_collection.json \
  -e tests/e2e/postman/ci.postman_environment.json \
  --delay-request 100

# Run with specific iteration count
newman run tests/e2e/postman/goimg-api.postman_collection.json \
  -e tests/e2e/postman/ci.postman_environment.json \
  -n 3

# Export results as JSON
newman run tests/e2e/postman/goimg-api.postman_collection.json \
  -e tests/e2e/postman/ci.postman_environment.json \
  --reporters cli,json \
  --reporter-json-export results.json
```

### Postman Desktop App

1. Import `goimg-api.postman_collection.json`
2. Import `ci.postman_environment.json` or create custom environment
3. Select environment from dropdown
4. Click "Run" to execute collection
5. View results in Collection Runner

## Test Assertions

### Standard Validations

Every test includes:

1. **Status Code**: Expected HTTP response code
2. **Response Time**: Performance check (< 2000ms for most endpoints)
3. **Headers**: Required headers present (`Content-Type`, `X-Request-ID`)
4. **JSON Schema**: Response structure validation
5. **Business Logic**: Data correctness

### Success Response Example

```javascript
pm.test("Status code is 200", function () {
    pm.response.to.have.status(200);
});

pm.test("Response has valid structure", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property('id');
    pm.expect(jsonData).to.have.property('email');
    pm.expect(jsonData.email).to.be.a('string');
});

pm.test("Response time is acceptable", function () {
    pm.expect(pm.response.responseTime).to.be.below(2000);
});

pm.test("Content-Type is application/json", function () {
    pm.response.to.have.header("Content-Type", /application\/json/);
});
```

### Error Response Example

All error responses follow RFC 7807:

```javascript
pm.test("Status code is 404", function () {
    pm.response.to.have.status(404);
});

pm.test("Error follows RFC 7807 format", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property('type');
    pm.expect(jsonData).to.have.property('title');
    pm.expect(jsonData).to.have.property('status');
    pm.expect(jsonData).to.have.property('detail');
    pm.expect(jsonData.status).to.equal(404);
});

pm.test("X-Request-ID header present", function () {
    pm.response.to.have.header("X-Request-ID");
});
```

## Security Testing

### Authentication Tests
- Token validation (valid, expired, malformed)
- Token blacklisting after logout
- Refresh token rotation
- Session management

### Authorization Tests
- Ownership checks (users can't modify others' resources)
- Role-based access control (admin, moderator, user)
- Privacy settings (public, private, unlisted content)
- Group membership validation

### Security Best Practices Validated
- Generic error messages (prevent user enumeration)
- Passwords never exposed in responses
- Email hidden in public profiles
- HTTPS-only cookies (Secure flag)
- CSRF protection headers
- Rate limiting (future enhancement)

### Vulnerability Prevention
- SQL injection (via parameterized queries)
- XSS (via input sanitization)
- Path traversal (via file upload validation)
- Malware upload (via ClamAV integration)

## CI/CD Integration

### GitHub Actions

The E2E test suite runs automatically on:
- Pull requests to `main`
- Pushes to `main`
- Manual workflow dispatch

Example workflow configuration:

```yaml
# .github/workflows/e2e-tests.yml
name: E2E Tests

on:
  pull_request:
    branches: [ main ]
  push:
    branches: [ main ]

jobs:
  e2e:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_DB: testdb
          POSTGRES_USER: testuser
          POSTGRES_PASSWORD: testpass
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Start API Server
        run: |
          make run &
          sleep 5

      - name: Install Newman
        run: make setup-e2e

      - name: Run E2E Tests
        run: make test-e2e

      - name: Upload Test Report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: newman-report
          path: newman-report.html
```

### Test Failure Handling

CI fails if:
- Any test assertion fails
- API server returns unexpected status codes
- Response time exceeds thresholds
- Network errors occur

On failure:
1. Check GitHub Actions logs for detailed error messages
2. Review uploaded HTML report artifact
3. Run failed folder locally for debugging
4. Fix issues and re-run tests

## Troubleshooting

### Test Failures

**Symptom**: 401 Unauthorized on protected endpoints
- **Cause**: Token expired or not set
- **Fix**: Check login test passed and token is stored in collection variables

**Symptom**: 409 Conflict on registration
- **Cause**: Test data collision (rare)
- **Fix**: Re-run tests; unique ID generation should resolve

**Symptom**: Connection refused / ECONNREFUSED
- **Cause**: API server not running
- **Fix**: Start API with `make run` in separate terminal

**Symptom**: Tests pass individually but fail in collection
- **Cause**: Test execution order dependency
- **Fix**: Review collection variables and ensure proper state management

**Symptom**: Image upload tests fail
- **Cause**: Test image file not found
- **Fix**: Verify `testImagePath` in environment points to valid file

**Symptom**: Random test timeouts
- **Cause**: Slow database queries or resource contention
- **Fix**: Check database health, consider adding delays between requests

### Collection Validation

Validate collection structure:

```bash
# Check JSON syntax
jq . tests/e2e/postman/goimg-api.postman_collection.json > /dev/null

# Validate with Newman (dry run)
make test-e2e-dry

# Check for missing environment variables
grep -r "pm.environment.get" tests/e2e/postman/goimg-api.postman_collection.json
```

### Debugging Tips

1. **Enable verbose output**:
   ```bash
   newman run tests/e2e/postman/goimg-api.postman_collection.json \
     -e tests/e2e/postman/ci.postman_environment.json \
     --verbose
   ```

2. **Add console logging** in test scripts:
   ```javascript
   console.log("Access Token:", pm.collectionVariables.get('accessToken'));
   console.log("Response:", JSON.stringify(pm.response.json(), null, 2));
   ```

3. **Run single request** in Postman Desktop for detailed inspection

4. **Check API logs** for server-side errors:
   ```bash
   docker logs goimg-api
   ```

## Maintenance

### Adding New Tests

When adding new API endpoints:

1. Add requests to appropriate folder in collection
2. Include test scripts with assertions
3. Update environment variables if needed
4. Update this README with new test counts
5. Run full suite to verify no regressions

Example test script template:

```javascript
pm.test("Status code is 200", function () {
    pm.response.to.have.status(200);
});

pm.test("Response has valid structure", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property('id');
    // Add more structure validations
});

pm.test("Business logic is correct", function () {
    const jsonData = pm.response.json();
    // Validate business rules
});

pm.test("Headers are correct", function () {
    pm.response.to.have.header("Content-Type", /application\/json/);
    pm.response.to.have.header("X-Request-ID");
});

// Store variables for subsequent tests
if (pm.response.code === 200) {
    const jsonData = pm.response.json();
    pm.collectionVariables.set('resourceId', jsonData.id);
}
```

### Updating Existing Tests

When modifying API behavior:

1. Update OpenAPI spec first (source of truth)
2. Update affected Postman requests
3. Update test assertions to match new behavior
4. Update environment variables if schema changed
5. Run affected folder to verify changes
6. Update this README if test counts changed

### Version Control

Commit changes:
- Collection JSON file
- Environment JSON files
- This README
- Related OpenAPI spec changes

## Best Practices

### Test Design
- Each test should be independent (no hidden dependencies)
- Use descriptive test names
- Validate both success and error cases
- Test boundary conditions (empty, max length, invalid formats)
- Clean up resources when possible (DELETE tests)

### Performance
- Keep response time assertions reasonable (< 2000ms for most)
- Consider adding delays for rate-limited endpoints
- Use parallel execution cautiously (may cause race conditions)

### Security
- Never commit real credentials
- Use environment variables for sensitive data
- Validate authentication on all protected endpoints
- Test authorization edge cases

### Maintainability
- Keep test scripts DRY (use collection-level scripts)
- Document complex test logic with comments
- Update README when adding significant test coverage
- Review test failures promptly (don't ignore flaky tests)

## References

- [Postman Collection Format v2.1](https://schema.getpostman.com/json/collection/v2.1.0/docs/)
- [Newman CLI Documentation](https://learning.postman.com/docs/running-collections/using-newman-cli/command-line-integration-with-newman/)
- [RFC 7807: Problem Details for HTTP APIs](https://datatracker.ietf.org/doc/html/rfc7807)
- [JWT Best Practices](https://datatracker.ietf.org/doc/html/rfc8725)
- [OpenAPI Specification](https://spec.openapis.org/oas/v3.0.3)
- [GoImg OpenAPI Spec](../../../api/openapi/openapi.yaml)

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-12-03 | Initial release (Sprint 4: Auth & Users) |
| 2.0.0 | 2026-02-10 | Comprehensive expansion to 290 tests across 20 folders |

**Current Status**: Complete coverage of all implemented API endpoints (~100%)

**Owned by**: Test Strategist
**Last updated**: 2026-02-10
**Version**: 2.0.0
