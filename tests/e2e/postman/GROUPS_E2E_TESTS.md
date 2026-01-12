# Groups E2E Tests - Sprint 20

## Overview

Comprehensive E2E test suite for the Groups/Communities API endpoints, covering all 14 API endpoints with 27 test scenarios and 105+ assertions.

## Test Organization

The Groups tests are organized into 5 categories:

### 1. Group Management (7 tests)
Tests for creating, retrieving, updating, and deleting groups.

- **Create Public Group - Success** (201)
  - Generates unique group name and slug
  - Validates group creation response structure
  - Stores group ID and slug for subsequent tests
  - Verifies owner is current user
  - Confirms initial member count is 1

- **Create Private Group - Success** (201)
  - Creates private group
  - Validates group type is private
  - Stores private group ID for testing access controls

- **Create Group - Invalid Data (400)**
  - Tests validation with too-short name (< 3 chars)
  - Tests validation with invalid group type
  - Validates RFC 7807 problem detail response

- **Get Group by ID - Success** (200)
  - Retrieves group using UUID
  - Validates response structure matches OpenAPI spec
  - Tests public endpoint (no auth required)

- **Get Group by Slug - Success** (200)
  - Retrieves group using URL-friendly slug
  - Validates slug-based lookup
  - Tests public endpoint (no auth required)

- **Update Group - Admin Only Success** (200)
  - Updates group description and settings
  - Validates admin/owner can update
  - Confirms changes are persisted

- **Delete Group - Owner Only Success** (204)
  - Creates temporary group for deletion
  - Validates owner can delete
  - Confirms 204 No Content response

### 2. Group Membership (8 tests)
Tests for joining, leaving, and listing group memberships.

- **Join Public Group - Instant Join** (201)
  - Creates second test user for membership tests
  - Joins public group with instant active status
  - Validates membership response structure
  - Stores membership ID for cleanup

- **Join Invite-Only Group - Request Created** (201)
  - Creates invite-only group
  - Validates join request creates "requested" status
  - Tests approval workflow requirement

- **Join Private Group - Forbidden (403)**
  - Attempts to join private group without invitation
  - Validates RFC 7807 forbidden response
  - Tests access control for private groups

- **Join Group - Already Member (409)**
  - Attempts duplicate join
  - Validates RFC 7807 conflict response
  - Tests idempotency protection

- **Leave Group - Success** (204)
  - Member leaves group successfully
  - Validates 204 No Content response
  - Tests membership removal

- **Leave Group - Owner Cannot Leave (400)**
  - Owner attempts to leave own group
  - Validates RFC 7807 bad request response
  - Tests business rule: owner must transfer or delete

- **List Group Members - With Pagination** (200)
  - Lists members with pagination parameters
  - Validates response structure
  - Tests member data completeness
  - Public endpoint (no auth required)

- **List User's Groups - Success** (200)
  - Lists all groups user belongs to
  - Validates pagination metadata
  - Confirms user has at least one group (created earlier)
  - Protected endpoint (JWT required)

### 3. Member Management (6 tests)
Tests for managing member roles, removing members, and banning users.

- **Promote Member to Admin - Owner Only** (200)
  - Re-joins member to group for promotion
  - Owner promotes member to admin role
  - Validates role update in response

- **Demote Admin to Member - Success** (200)
  - Demotes admin back to member
  - Validates role hierarchy enforcement

- **Update Role - Non-Owner Cannot Promote (403)**
  - Member attempts to promote another member
  - Validates RFC 7807 forbidden response
  - Tests permission enforcement

- **Remove Member - Admin+ Can Remove** (204)
  - Admin/owner removes member from group
  - Validates 204 No Content response
  - Tests member removal authority

- **Remove Member - Cannot Remove Owner (400/403)**
  - Attempts to remove group owner
  - Validates protection of owner role
  - Tests business rule enforcement

- **Ban Member - Admin+ Can Ban** (200)
  - Re-joins member for ban test
  - Admin/owner bans member with reason
  - Validates membership status changed to "banned"

### 4. Search & Discovery (4 tests)
Tests for listing, searching, and filtering groups.

- **List Public Groups - With Pagination** (200)
  - Lists public and invite-only groups
  - Validates pagination metadata structure
  - Tests default sorting and limits
  - Public endpoint (no auth required)

- **Search Groups by Name - Success** (200)
  - Full-text search for groups
  - Validates search query matching
  - Tests result relevance
  - Public endpoint (no auth required)

- **List Groups - Private Not in Results** (200)
  - Validates private groups excluded from public listings
  - Tests access control for discovery
  - Confirms only public/invite-only groups visible

- **Filter Groups by Type - Success** (200)
  - Filters groups by type parameter
  - Validates all results match filter
  - Tests query parameter handling

### 5. Error Handling (2 tests)
Tests for RFC 7807 error response compliance.

- **Get Group - Not Found (404)**
  - Requests non-existent group UUID
  - Validates RFC 7807 problem detail structure
  - Tests X-Request-ID header presence
  - Confirms error title and status code

- **Create Group - Unauthorized (401)**
  - Attempts creation without JWT token
  - Validates RFC 7807 unauthorized response
  - Tests WWW-Authenticate header presence
  - Confirms authentication enforcement

## API Endpoints Coverage

All 14 Sprint 20 Groups endpoints are covered:

### Public Endpoints (no auth)
1. `GET /api/v1/groups` - List public groups
2. `GET /api/v1/groups/search?q=` - Search groups
3. `GET /api/v1/groups/{groupID}` - Get group by ID
4. `GET /api/v1/groups/by-slug/{slug}` - Get group by slug
5. `GET /api/v1/groups/{groupID}/members` - List members

### Protected Endpoints (JWT required)
6. `POST /api/v1/groups` - Create group
7. `PUT /api/v1/groups/{groupID}` - Update group (admin+)
8. `DELETE /api/v1/groups/{groupID}` - Delete group (owner only)
9. `POST /api/v1/groups/{groupID}/join` - Join group
10. `DELETE /api/v1/groups/{groupID}/leave` - Leave group
11. `PUT /api/v1/groups/{groupID}/members/{userID}/role` - Update role (admin+)
12. `DELETE /api/v1/groups/{groupID}/members/{userID}` - Remove member (admin+)
13. `POST /api/v1/groups/{groupID}/members/{userID}/ban` - Ban member (admin+)
14. `GET /api/v1/me/groups` - User's groups

## Test Metrics

- **Total Test Requests**: 27
- **Total Assertions**: 105+
- **Average Assertions per Test**: ~4
- **API Endpoints Covered**: 14/14 (100%)
- **HTTP Methods Tested**: GET, POST, PUT, DELETE
- **Status Codes Tested**: 200, 201, 204, 400, 401, 403, 404, 409
- **RFC 7807 Error Tests**: 5 tests
- **Authentication Tests**: Protected vs public endpoints
- **Authorization Tests**: Owner, admin, member role hierarchy
- **Business Logic Tests**: Group types, join flows, owner protections

## Collection Variables

The following environment variables are used by the Groups tests:

### Group Variables
- `testGroupId` - Generic group ID for tests
- `testGroupName` - Dynamically generated group name
- `testGroupSlug` - Dynamically generated URL slug
- `testPublicGroupId` - Public group ID (main test group)
- `testPublicGroupSlug` - Public group slug
- `testPrivateGroupId` - Private group ID (for access control tests)
- `testPrivateGroupName` - Private group name
- `testPrivateGroupSlug` - Private group slug
- `testInviteOnlyGroupId` - Invite-only group ID
- `testDeleteGroupId` - Temporary group for deletion test

### Membership Variables
- `testMemberId` - Second user ID for membership tests
- `memberAccessToken` - JWT token for second user
- `testMembershipId` - Membership record ID

### Existing Variables Used
- `accessToken` - Primary user JWT token
- `testUserId` - Primary user ID
- `testPassword` - Standard test password
- `baseUrl` - API base URL (http://localhost:8080/api/v1)

## Test Patterns

### Pre-request Scripts
- **Unique Data Generation**: Timestamp + random suffix for idempotent tests
- **Dynamic User Creation**: Creates second user for membership tests
- **Authentication Setup**: Registers and logs in test users
- **Resource Preparation**: Creates temporary resources (groups) for specific tests

### Test Assertions
- **Status Code Validation**: Every test validates HTTP status
- **Content-Type Headers**: Validates JSON and RFC 7807 problem+json
- **Response Structure**: Validates all required fields per OpenAPI spec
- **Business Logic**: Tests domain rules (owner protections, role hierarchy)
- **Performance**: Response time assertions (< 2000ms)
- **Security Headers**: X-Request-ID, WWW-Authenticate when appropriate
- **Data Persistence**: Stores IDs for chaining and cleanup

### RFC 7807 Error Validation
All error tests validate:
- Correct Content-Type: `application/problem+json; charset=utf-8`
- Required fields: `type`, `title`, `status`, `detail`
- Status code consistency between header and body
- Descriptive error titles and details

## Running the Tests

### Prerequisites
1. API server running on http://localhost:8080
2. Database migrated with latest schema
3. Auth and Users endpoints functional (tests depend on user creation)

### Execution

```bash
# Run all tests including Groups
newman run tests/e2e/postman/goimg-api.postman_collection.json \
  -e tests/e2e/postman/ci.postman_environment.json

# Run only Groups tests
newman run tests/e2e/postman/goimg-api.postman_collection.json \
  -e tests/e2e/postman/ci.postman_environment.json \
  --folder "Groups"

# Run specific subcategory
newman run tests/e2e/postman/goimg-api.postman_collection.json \
  -e tests/e2e/postman/ci.postman_environment.json \
  --folder "1. Group Management"

# Run with verbose output
newman run tests/e2e/postman/goimg-api.postman_collection.json \
  -e tests/e2e/postman/ci.postman_environment.json \
  --folder "Groups" \
  --reporters cli,json \
  --reporter-json-export results.json
```

### CI/CD Integration

The Groups tests are integrated into the existing CI pipeline:
- Tests run automatically after build in GitHub Actions
- Must pass before merge to main branch
- Failures block deployment
- Test results are archived for debugging

## Test Dependencies

Tests should run in sequence within each category:

1. **Group Management**: Create groups before other operations
2. **Group Membership**: Requires existing groups and users
3. **Member Management**: Requires members in groups
4. **Search & Discovery**: Requires groups to exist
5. **Error Handling**: Independent, can run anytime

## Coverage Analysis

### Happy Path Coverage
- Create, read, update, delete groups
- Join and leave groups
- Promote, demote, remove, and ban members
- Search and filter groups
- List members and user's groups

### Edge Cases Covered
- Invalid input validation (too short names, invalid types)
- Duplicate operations (already a member)
- Permission violations (non-owner trying to promote)
- Business rule enforcement (owner cannot leave)
- Access controls (private groups not discoverable)
- Group type behaviors (instant join vs approval required)

### Security Testing
- Authentication enforcement (401 tests)
- Authorization by role (403 tests)
- Resource ownership validation
- Token-based access control
- Public vs protected endpoint separation

## Maintenance Notes

### Adding New Tests
1. Follow the existing structure and naming conventions
2. Use pre-request scripts for dynamic data generation
3. Include RFC 7807 validation for error tests
4. Store resource IDs in collection variables
5. Add new variables to environment file if needed
6. Document new tests in this file

### Updating Tests
- Keep tests aligned with OpenAPI spec changes
- Update assertions when response schemas change
- Maintain test independence and idempotency
- Preserve error handling test coverage

### Debugging Failures
- Check console output for specific assertion failures
- Verify API server is running and healthy
- Confirm database migrations are up to date
- Check collection variable state between tests
- Review pre-request script execution logs

## Related Documentation

- OpenAPI Spec: `/home/user/goimg-datalayer/api/openapi/openapi.yaml` (lines 4670-5156)
- Sprint 20 Plan: `/home/user/goimg-datalayer/claude/sprint_20_plan.md`
- Test Strategy: `/home/user/goimg-datalayer/claude/test_strategy.md`
- Groups Domain: `/home/user/goimg-datalayer/internal/domain/group/`
- Groups Handlers: `/home/user/goimg-datalayer/internal/interfaces/http/handlers/group_handlers.go`

## Success Criteria

All 27 tests passing indicates:
- All 14 Groups API endpoints are functional
- Authentication and authorization working correctly
- Group types (public, private, invite-only) behave as specified
- Role hierarchy (owner > admin > member) is enforced
- Business rules (owner protections, join flows) are implemented
- RFC 7807 error responses are compliant
- Performance targets are met (< 2s response times)
- Regression protection is in place for Sprint 20 features
