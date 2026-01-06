# Sprint 4 Kickoff: Application & HTTP - Identity Context

**Sprint**: Sprint 4
**Duration**: 2 weeks (Days 1-14)
**Start Date**: 2025-12-03
**End Date**: 2025-12-17
**Sprint Goal**: Deliver fully functional authentication and user management API with production-grade security

---

## Executive Summary

Sprint 4 builds on the infrastructure foundation from Sprint 3 (database, Redis, JWT service) to deliver user-facing authentication and user management capabilities. This sprint implements the application layer (commands/queries) and HTTP layer (handlers, middleware) for the Identity Context.

### Success Criteria

**Primary Objectives**:
1. Users can register, login, refresh tokens, and logout via REST API
2. JWT authentication middleware protects all secured endpoints
3. Account security controls prevent abuse (rate limiting, lockout, enumeration)
4. All auth flows have Newman/Postman E2E tests
5. Security gate S4 passes with zero critical findings

**Metrics**:
- 100% of planned endpoints implemented and tested
- Application layer test coverage >= 85%
- HTTP layer test coverage >= 75%
- Rate limiting verified under load (5 req/min login, 100 req/min global)
- Zero auth-related security vulnerabilities
- E2E test suite covers 100% of auth flows

### Dependencies Confirmed

**Sprint 3 Deliverables (All Complete)**:
- [x] Database migrations (users, sessions tables)
- [x] PostgreSQL repositories (UserRepository, SessionRepository)
- [x] Redis client and session store
- [x] JWT service with RS256 signing
- [x] Refresh token rotation with replay detection
- [x] Token blacklist in Redis
- [x] Integration tests with testcontainers

**External Dependencies**:
- PostgreSQL 16+ running (via docker-compose)
- Redis 7+ running (via docker-compose)
- OpenAPI spec defines auth endpoints (verified: `/home/user/goimg-datalayer/api/openapi/openapi.yaml`)

---

## Sprint Goals

### Goal 1: Application Layer - Auth Use Cases
Implement CQRS command/query handlers for authentication flows following DDD patterns.

**Deliverables**:
- RegisterUserCommand + handler
- LoginCommand + handler
- RefreshTokenCommand + handler
- LogoutCommand + handler
- GetUserQuery + handler
- UpdateUserCommand + handler

**Acceptance Criteria**:
- All handlers delegate to domain layer (no business logic in application layer)
- Handlers orchestrate repository calls, JWT service, and domain events
- Application layer test coverage >= 85%
- Unit tests with mocks for all dependencies

### Goal 2: HTTP Layer - Auth Endpoints
Implement RESTful authentication API with security best practices.

**Deliverables**:
- POST /api/v1/auth/register - User registration
- POST /api/v1/auth/login - User login with JWT issuance
- POST /api/v1/auth/refresh - Token refresh with rotation
- POST /api/v1/auth/logout - Session termination
- GET /api/v1/users/{id} - User profile retrieval
- PUT /api/v1/users/{id} - User profile update
- DELETE /api/v1/users/{id} - Account deletion

**Acceptance Criteria**:
- All endpoints match OpenAPI specification exactly
- RFC 7807 Problem Details for all error responses
- Request validation with go-playground/validator
- HTTP layer test coverage >= 75%

### Goal 3: Security Middleware
Implement production-grade security controls for the API.

**Deliverables**:
- JWT authentication middleware (Bearer token validation)
- Request ID middleware (correlation IDs for tracing)
- Structured logging middleware (zerolog with context)
- Security headers middleware (X-Frame-Options, CSP, etc.)
- Rate limiting middleware (Redis-backed sliding window)
- CORS configuration (environment-specific origins)
- Error mapping to RFC 7807

**Acceptance Criteria**:
- All security headers present on responses
- Rate limits enforced (login: 5/min, global: 100/min)
- JWT validation rejects expired, malformed, revoked tokens
- Audit logging captures all auth events (success/failure)
- No sensitive data in logs (passwords, tokens)

### Goal 4: E2E Test Coverage
Comprehensive Newman/Postman test suite for regression testing.

**Deliverables**:
- Postman collection updated with auth endpoints
- Test scripts validate response structure and business logic
- Happy path tests for all flows
- Error handling tests (4xx/5xx scenarios)
- Authentication flow tests (login → access protected resource → refresh → logout)

**Acceptance Criteria**:
- `make test-e2e` passes locally
- CI Newman job passes
- 100% endpoint coverage in Postman collection
- Test scripts validate JSON schema and status codes

---

## Agent Task Assignments

### Lead Agent: senior-go-architect

**Responsibilities**:
- Overall sprint coordination and technical leadership
- Architecture review for application and HTTP layers
- Code review for all implementations
- Performance optimization and Go idioms validation

**Assigned Tasks**:
1. **Pre-Sprint Architecture Review** (Pre-Sprint checkpoint)
   - Priority: P0
   - Complexity: M
   - Review CQRS command/query handler patterns
   - Validate middleware architecture and execution order
   - Approve error mapping strategy to RFC 7807
   - Review dependency injection approach

2. **Application Layer Implementation Guidance** (Days 1-7)
   - Priority: P0
   - Complexity: L
   - Guide implementation of command/query handlers
   - Ensure proper separation of concerns
   - Review orchestration logic in handlers

3. **HTTP Handler Code Review** (Days 8-12)
   - Priority: P0
   - Complexity: M
   - Review all HTTP handlers for proper delegation
   - Verify no business logic in handlers
   - Check error mapping completeness

4. **Pre-Merge Quality Gate** (Day 13-14)
   - Priority: P0
   - Complexity: S
   - Final code review approval
   - Verify Go idioms and best practices
   - Confirm DDD layering principles

---

### Critical Agent: senior-secops-engineer

**Responsibilities**:
- Security controls implementation review
- Security gate S4 approval
- Authentication flow security validation
- Rate limiting and account lockout verification

**Assigned Tasks**:
1. **Security Architecture Review** (Pre-Sprint checkpoint)
   - Priority: P0
   - Complexity: M
   - Review middleware security (rate limiting, CORS, headers)
   - Validate JWT authentication middleware design
   - Review account lockout and enumeration prevention strategy

2. **Account Security Controls Implementation** (Days 3-7)
   - Priority: P0
   - Complexity: L
   - Implement account lockout after 5 failed attempts
   - Implement account enumeration prevention (generic error messages)
   - Add audit logging for authentication events
   - Verify no sensitive data in logs

3. **Security Middleware Review** (Days 8-10)
   - Priority: P0
   - Complexity: M
   - Review authentication middleware implementation
   - Validate rate limiting under load (k6 or vegeta)
   - Check security headers configuration
   - Verify CORS policy

4. **Security Gate S4 Review** (Day 12-14)
   - Priority: P0
   - Complexity: L
   - Execute all S4 security gate checks (see `/home/user/goimg-datalayer/claude/security_gates.md`)
   - Validate controls S4-HTTP-001 through S4-ERR-002
   - Verify all required security tests pass
   - Sign off on security gate or document remediation items

---

### Critical Agent: backend-test-architect

**Responsibilities**:
- Application layer test strategy and implementation
- Test coverage validation (85% application, 75% handlers)
- Integration test design with testcontainers
- Mock creation for dependencies

**Assigned Tasks**:
1. **Application Layer Test Strategy** (Pre-Sprint checkpoint)
   - Priority: P0
   - Complexity: M
   - Design test strategy for command/query handlers
   - Plan mocking approach for repositories and JWT service
   - Define test fixtures and table-driven test patterns

2. **Application Layer Unit Tests** (Days 2-8)
   - Priority: P0
   - Complexity: L
   - Implement unit tests for all command handlers
   - Implement unit tests for all query handlers
   - Achieve 85%+ coverage for application layer
   - Use table-driven tests with t.Parallel()

3. **HTTP Layer Integration Tests** (Days 9-12)
   - Priority: P0
   - Complexity: L
   - Implement integration tests for auth handlers
   - Implement integration tests for user handlers
   - Use testcontainers for PostgreSQL and Redis
   - Achieve 75%+ coverage for HTTP layer

4. **Pre-Merge Coverage Review** (Day 13-14)
   - Priority: P0
   - Complexity: S
   - Validate application layer coverage >= 85%
   - Validate HTTP layer coverage >= 75%
   - Verify race detector clean (`go test -race ./internal/application/...`)
   - Review test quality (not just coverage)

---

### Supporting Agent: cicd-guardian

**Responsibilities**:
- CI/CD pipeline health monitoring
- Newman E2E test integration in CI
- Docker configuration updates if needed
- Pipeline performance optimization

**Assigned Tasks**:
1. **CI Pipeline Verification** (Pre-Sprint checkpoint)
   - Priority: P1
   - Complexity: S
   - Verify CI pipeline ready for Sprint 4 deliverables
   - Check Docker Compose services healthy
   - Validate Newman/Postman CI integration

2. **CI Failure Response** (Continuous, Days 1-14)
   - Priority: P0
   - Complexity: Varies
   - Monitor CI pipeline for failures
   - Diagnose infrastructure vs code issues
   - Coordinate with agents for code-related fixes

3. **E2E Test CI Integration** (Days 10-12)
   - Priority: P0
   - Complexity: M
   - Verify Newman E2E tests run in CI after build
   - Ensure test reports uploaded as artifacts
   - Validate E2E tests block merge on failure

4. **Pre-Merge Pipeline Validation** (Day 13-14)
   - Priority: P0
   - Complexity: S
   - Confirm all CI jobs green
   - Verify Newman E2E tests passing
   - Check security scans (gosec, trivy) passing

---

### Supporting Agent: test-strategist

**Responsibilities**:
- E2E test design and implementation
- Postman collection creation and maintenance
- API contract testing (OpenAPI compliance)
- Edge case and boundary condition identification

**Assigned Tasks**:
1. **E2E Test Strategy Design** (Pre-Sprint checkpoint)
   - Priority: P0
   - Complexity: M
   - Design E2E test scenarios for authentication flows
   - Plan Postman collection structure
   - Identify edge cases and error scenarios

2. **Postman Collection Implementation** (Days 5-10)
   - Priority: P0
   - Complexity: L
   - Create Postman requests for all auth endpoints
   - Add test scripts for response validation
   - Implement authentication flow tests (login → access → refresh → logout)
   - Add error handling tests (4xx/5xx)

3. **E2E Test Validation** (Days 11-13)
   - Priority: P0
   - Complexity: M
   - Verify `make test-e2e` passes locally
   - Validate test scripts check JSON schema and status codes
   - Test authentication flow end-to-end
   - Verify error scenarios covered

4. **Pre-Merge E2E Approval** (Day 13-14)
   - Priority: P0
   - Complexity: S
   - Confirm 100% endpoint coverage in Postman
   - Verify Newman E2E tests passing in CI
   - Validate test quality (not just happy paths)

---

## Task Breakdown with Priorities

### P0 Tasks (Must Complete for MVP)

#### Application Layer Commands (Days 1-5)

**Task 1.1: RegisterUserCommand + Handler**
- Complexity: M (Medium)
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/application/identity/commands/register_user.go`
- Acceptance Criteria:
  - Validates email, username, password via domain layer
  - Checks for existing user (email, username)
  - Hashes password with Argon2id
  - Saves user via UserRepository
  - Returns user ID on success
  - Returns domain errors on failure
  - Unit tests with mocks (coverage >= 90%)

**Task 1.2: LoginCommand + Handler**
- Complexity: L (Large)
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/application/identity/commands/login.go`
- Acceptance Criteria:
  - Retrieves user by email
  - Verifies password (constant-time comparison)
  - Implements account lockout (5 failed attempts)
  - Implements session ID regeneration on login
  - Issues access token (15 min TTL) and refresh token (7 day TTL)
  - Saves session via SessionRepository
  - Logs authentication event (audit)
  - Returns tokens on success
  - Returns generic error on failure (no enumeration)
  - Unit tests with mocks (coverage >= 90%)

**Task 1.3: RefreshTokenCommand + Handler**
- Complexity: M
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/application/identity/commands/refresh_token.go`
- Acceptance Criteria:
  - Validates refresh token (not expired, not blacklisted)
  - Detects replay attacks (token reuse)
  - Implements token rotation (invalidate old, issue new)
  - Issues new access and refresh tokens
  - Updates session in SessionRepository
  - Logs token refresh event
  - Returns new tokens on success
  - Unit tests with mocks (coverage >= 90%)

**Task 1.4: LogoutCommand + Handler**
- Complexity: S (Small)
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/application/identity/commands/logout.go`
- Acceptance Criteria:
  - Invalidates session in SessionRepository
  - Adds access token to blacklist (Redis)
  - Logs logout event
  - Returns success
  - Unit tests with mocks (coverage >= 90%)

**Task 1.5: GetUserQuery + Handler**
- Complexity: S
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/application/identity/queries/get_user.go`
- Acceptance Criteria:
  - Retrieves user by ID from UserRepository
  - Returns user DTO (no password hash)
  - Handles not found error
  - Unit tests with mocks (coverage >= 90%)

**Task 1.6: UpdateUserCommand + Handler**
- Complexity: M
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/application/identity/commands/update_user.go`
- Acceptance Criteria:
  - Validates ownership (user can only update own profile)
  - Allows updating: display_name, bio
  - Does NOT allow updating: email, username, role (special flows)
  - Saves via UserRepository
  - Logs update event
  - Returns updated user DTO
  - Unit tests with mocks (coverage >= 90%)

---

#### HTTP Layer Handlers (Days 6-10)

**Task 2.1: Auth Handlers (POST /api/v1/auth/*)**
- Complexity: L
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/interfaces/http/handlers/auth_handler.go`
- Acceptance Criteria:
  - POST /auth/register: Delegates to RegisterUserCommand
  - POST /auth/login: Delegates to LoginCommand
  - POST /auth/refresh: Delegates to RefreshTokenCommand
  - POST /auth/logout: Delegates to LogoutCommand
  - All handlers parse and validate request DTOs
  - All handlers map domain errors to RFC 7807 Problem Details
  - No business logic in handlers (only orchestration)
  - Integration tests with testcontainers (coverage >= 75%)

**Task 2.2: User Handlers (GET/PUT/DELETE /api/v1/users/{id})**
- Complexity: M
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/interfaces/http/handlers/user_handler.go`
- Acceptance Criteria:
  - GET /users/{id}: Delegates to GetUserQuery
  - PUT /users/{id}: Delegates to UpdateUserCommand
  - DELETE /users/{id}: Delegates to DeleteUserCommand (future)
  - Ownership validation middleware applied
  - All handlers map domain errors to RFC 7807 Problem Details
  - Integration tests with testcontainers (coverage >= 75%)

---

#### Middleware (Days 7-11)

**Task 3.1: JWT Authentication Middleware**
- Complexity: L
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/interfaces/http/middleware/auth.go`
- Acceptance Criteria:
  - Extracts Bearer token from Authorization header
  - Validates JWT signature and claims
  - Checks token not expired
  - Checks token not blacklisted (Redis)
  - Sets authenticated user context
  - Returns 401 for invalid/missing token
  - Returns 403 for blacklisted token
  - Unit tests (coverage >= 80%)
  - Integration tests with testcontainers

**Task 3.2: Request ID Middleware**
- Complexity: S
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/interfaces/http/middleware/request_id.go`
- Acceptance Criteria:
  - Generates UUID for each request
  - Sets X-Request-ID response header
  - Adds request ID to logger context
  - Propagates request ID through context
  - Unit tests (coverage >= 80%)

**Task 3.3: Structured Logging Middleware (zerolog)**
- Complexity: M
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/interfaces/http/middleware/logging.go`
- Acceptance Criteria:
  - Logs request method, path, status, duration
  - Includes request ID in logs
  - Includes user ID if authenticated
  - Does NOT log sensitive data (passwords, tokens, auth headers)
  - Uses zerolog with structured fields
  - Unit tests (coverage >= 80%)

**Task 3.4: Security Headers Middleware**
- Complexity: S
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/interfaces/http/middleware/security_headers.go`
- Acceptance Criteria:
  - Sets X-Content-Type-Options: nosniff
  - Sets X-Frame-Options: DENY
  - Sets X-XSS-Protection: 1; mode=block
  - Sets Referrer-Policy: strict-origin-when-cross-origin
  - Sets Content-Security-Policy: default-src 'self'
  - Sets Permissions-Policy: geolocation=(), microphone=()
  - Unit tests verify all headers present (coverage >= 80%)

**Task 3.5: Rate Limiting Middleware**
- Complexity: L
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/interfaces/http/middleware/rate_limit.go`
- Acceptance Criteria:
  - Implements Redis-backed sliding window algorithm
  - Login endpoint: 5 req/min per IP
  - Global: 100 req/min per IP
  - Authenticated: 300 req/min per user
  - Returns 429 with Retry-After header
  - Returns X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset headers
  - Unit tests with Redis mock (coverage >= 80%)
  - Load tests verify limits enforced

**Task 3.6: CORS Middleware**
- Complexity: S
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/interfaces/http/middleware/cors.go`
- Acceptance Criteria:
  - Uses go-chi/cors library
  - Environment-specific allowed origins (no wildcard in production)
  - Allows methods: GET, POST, PUT, DELETE, OPTIONS
  - Allows headers: Authorization, Content-Type
  - Exposes headers: X-Request-ID, X-RateLimit-*
  - Unit tests (coverage >= 80%)

**Task 3.7: Error Mapping to RFC 7807**
- Complexity: M
- Owner: Direct implementation
- Location: `/home/user/goimg-datalayer/internal/interfaces/http/middleware/error_handler.go`
- Acceptance Criteria:
  - Maps domain errors to HTTP status codes
  - Returns RFC 7807 Problem Details JSON
  - Includes traceId (request ID) in response
  - Does NOT leak internal errors to client
  - Logs internal errors with full stack trace
  - Unit tests for all error types (coverage >= 80%)

---

#### E2E Tests (Days 8-13)

**Task 4.1: Postman Collection - Auth Endpoints**
- Complexity: M
- Owner: test-strategist
- Location: `/home/user/goimg-datalayer/tests/e2e/postman/goimg-api.postman_collection.json`
- Acceptance Criteria:
  - POST /auth/register request with test scripts
  - POST /auth/login request with test scripts
  - POST /auth/refresh request with test scripts
  - POST /auth/logout request with test scripts
  - Test scripts validate status codes (200, 201, 400, 401, 409, 429)
  - Test scripts validate JSON schema
  - Test scripts validate business logic (tokens issued, session created)

**Task 4.2: Postman Collection - User Endpoints**
- Complexity: M
- Owner: test-strategist
- Location: `/home/user/goimg-datalayer/tests/e2e/postman/goimg-api.postman_collection.json`
- Acceptance Criteria:
  - GET /users/{id} request with test scripts
  - PUT /users/{id} request with test scripts
  - DELETE /users/{id} request with test scripts (future)
  - Test scripts validate authorization (403 for non-owners)
  - Test scripts validate JSON schema

**Task 4.3: Authentication Flow E2E Test**
- Complexity: L
- Owner: test-strategist
- Location: `/home/user/goimg-datalayer/tests/e2e/postman/goimg-api.postman_collection.json`
- Acceptance Criteria:
  - Test flow: Register → Login → Access protected resource → Refresh → Logout
  - Verify tokens issued and accepted
  - Verify refresh token rotation
  - Verify logout invalidates tokens
  - Test scripts validate complete flow

**Task 4.4: Error Handling E2E Tests**
- Complexity: M
- Owner: test-strategist
- Location: `/home/user/goimg-datalayer/tests/e2e/postman/goimg-api.postman_collection.json`
- Acceptance Criteria:
  - Test 400 Bad Request (validation errors)
  - Test 401 Unauthorized (missing/invalid token)
  - Test 403 Forbidden (insufficient permissions)
  - Test 409 Conflict (user already exists)
  - Test 429 Too Many Requests (rate limiting)
  - Verify RFC 7807 Problem Details format

---

### P1 Tasks (Important but Can Slip)

**Task 5.1: Account Lockout Mechanism**
- Complexity: M
- Owner: senior-secops-engineer
- Location: `/home/user/goimg-datalayer/internal/application/identity/commands/login.go`
- Acceptance Criteria:
  - Track failed login attempts per user (Redis)
  - Lock account after 5 failed attempts
  - Lockout duration: 15 minutes
  - Clear failed attempts on successful login
  - Return generic error on lockout (no enumeration)
  - Unit tests (coverage >= 90%)

**Task 5.2: Audit Logging for Auth Events**
- Complexity: M
- Owner: senior-secops-engineer
- Location: `/home/user/goimg-datalayer/internal/application/identity/commands/*.go`
- Acceptance Criteria:
  - Log login success (user ID, IP, user agent, timestamp)
  - Log login failure (email attempted, IP, reason, timestamp)
  - Log logout (user ID, timestamp)
  - Log token refresh (user ID, session ID, timestamp)
  - Log user registration (user ID, email, timestamp)
  - Use structured logging (zerolog)
  - Do NOT log sensitive data

**Task 5.3: OpenAPI Spec Validation**
- Complexity: S
- Owner: senior-go-architect
- Location: `/home/user/goimg-datalayer/api/openapi/openapi.yaml`
- Acceptance Criteria:
  - Verify all auth endpoints defined
  - Verify security schemes defined (JWT Bearer)
  - Verify request/response schemas match implementation
  - `make validate-openapi` passes

---

### P2 Tasks (Nice to Have)

**Task 6.1: Password Strength Meter (Backend)**
- Complexity: S
- Owner: Optional
- Location: `/home/user/goimg-datalayer/internal/domain/identity/password.go`
- Acceptance Criteria:
  - Calculate password strength score (0-100)
  - Return strength in registration response
  - Do NOT block weak passwords (just inform)

**Task 6.2: Session Management Endpoint**
- Complexity: M
- Owner: Optional
- Location: `/home/user/goimg-datalayer/internal/interfaces/http/handlers/session_handler.go`
- Acceptance Criteria:
  - GET /users/{id}/sessions: List active sessions
  - DELETE /users/{id}/sessions/{session_id}: Revoke session
  - User can only manage own sessions

**Task 6.3: Email Verification Flow**
- Complexity: L
- Owner: Optional (Post-Sprint 4)
- Deferred to Sprint 5 or later
- Requires SMTP integration

---

## Checkpoints Schedule

### Pre-Sprint Checkpoint (Day 0 - December 3, 2025)

**Timing**: Before implementation begins
**Duration**: 2 hours
**Attendees**: senior-go-architect (lead), senior-secops-engineer, backend-test-architect, test-strategist

**Agenda**:
1. Review Sprint 4 goals and deliverables (15 min)
2. senior-go-architect: Review CQRS command/query handler patterns (30 min)
3. senior-secops-engineer: Review middleware security (rate limiting, CORS, headers) (30 min)
4. backend-test-architect: Present test strategy for application and HTTP layers (20 min)
5. test-strategist: Present E2E test scenarios for authentication flows (20 min)
6. Identify risks and dependencies (10 min)
7. Validate agent assignments and capacity (5 min)

**Output**:
- Pre-sprint checkpoint summary with agent commitments
- Architecture decision record (if needed)
- Risk register updated
- Agent task assignments confirmed

**Action Items**:
- [ ] senior-go-architect: Document CQRS handler pattern for team
- [ ] senior-secops-engineer: Document account lockout strategy
- [ ] backend-test-architect: Create test fixture templates
- [ ] test-strategist: Draft Postman collection structure
- [ ] scrum-master: Confirm all Pre-Sprint action items completed before Day 1

---

### Mid-Sprint Checkpoint (Day 7 - December 10, 2025)

**Timing**: Midpoint of sprint
**Duration**: 1 hour
**Attendees**: senior-go-architect (lead), senior-secops-engineer, backend-test-architect

**Agenda**:
1. Review sprint burndown (progress toward sprint goal) (10 min)
2. senior-secops-engineer: Review authentication middleware and account lockout logic (15 min)
3. senior-go-architect: Review error mapping to RFC 7807 Problem Details (15 min)
4. backend-test-architect: Application layer coverage trajectory check (expect >= 70% at this point) (15 min)
5. Identify blockers requiring escalation (5 min)

**Output**:
- Burndown status report
- Blocker resolution plan (if any)
- Adjusted assignments (if needed)

**Metrics to Review**:
- Sprint progress: Expected ~50% complete
- Application layer test coverage: Expected >= 70%
- Blocked tasks: Expected 0
- CI pipeline health: Expected green

**Red Flags**:
- Coverage < 60%
- >1 task blocked >24 hours
- CI pipeline red
- Agent bandwidth exceeded

---

### Pre-Merge Checkpoint (Day 13-14 - December 16-17, 2025)

**Timing**: Before merging sprint branch to main
**Duration**: 2 hours
**Attendees**: All agents with Pre-Merge checklist items

**Agenda**:
1. Execute all automated quality gates (30 min)
   - `go fmt ./...`
   - `go vet ./...`
   - `golangci-lint run`
   - `go test -race ./internal/application/...`
   - `go test -race ./internal/interfaces/http/...`
   - `make validate-openapi`
   - `make test-e2e`
   - `gosec ./...`

2. Complete manual verification checklist (30 min)
   - senior-go-architect: Code review approval (DDD layering, Go idioms)
   - backend-test-architect: Coverage thresholds met (application: 85%, HTTP: 75%)
   - test-strategist: Newman/Postman E2E tests passing (auth flow coverage 100%)
   - test-strategist: Postman collection updated with all new endpoints
   - senior-secops-engineer: Security checklist verified (S4-HTTP-001 through S4-ERR-002)

3. Review agent-specific approvals (30 min)
   - senior-go-architect: No business logic in HTTP handlers
   - senior-secops-engineer: Account enumeration prevention verified
   - senior-secops-engineer: Account lockout functional
   - senior-secops-engineer: Audit logging captures all auth events
   - senior-secops-engineer: No sensitive data in logs
   - backend-test-architect: Race detector clean
   - test-strategist: E2E tests cover error scenarios

4. Verify OpenAPI spec alignment (if HTTP changes) (15 min)
   - All endpoints match spec
   - All DTOs match schemas
   - Security schemes correctly applied

5. Confirm agent_checklist.md compliance (15 min)
   - Review `/home/user/goimg-datalayer/claude/agent_checklist.md`

**Output**:
- Merge approval OR list of blocking issues
- Sprint completion report
- Technical debt identified (if any)

**Pass Criteria**:
- All automated tests green
- All manual approvals received
- Zero critical security findings
- Coverage thresholds met

**Fail Response**:
- Document blocking issues
- Create remediation tasks
- Re-run Pre-Merge checkpoint after fixes

---

### Sprint Retrospective (Day 14 - December 17, 2025)

**Timing**: End of sprint
**Duration**: 1 hour
**Attendees**: All active agents from sprint

**Format**: Start/Stop/Continue

**Agenda**:
1. Sprint metrics review (10 min)
   - Velocity: [X] story points completed
   - Completion rate: [Y]%
   - Defect count: [Z]
   - Test coverage achieved
   - Sprint goal: Achieved / Partially / Not achieved

2. What went well? (15 min)
   - Celebrate wins
   - Identify effective practices

3. What didn't go well? (15 min)
   - Identify pain points
   - Surface blockers

4. What should we change? (15 min)
   - Actionable improvements with owners
   - Start/Stop/Continue format

5. Review previous retrospective action items (5 min)
   - Mark completed items
   - Carry forward incomplete items

**Output**:
- Sprint retrospective document
- Improvement backlog with owners and due dates
- Carry-over tasks to Sprint 5 (if any)

**Template**:
```markdown
## Sprint 4 Retrospective

**Date**: 2025-12-17
**Participants**: senior-go-architect, senior-secops-engineer, backend-test-architect, cicd-guardian, test-strategist, scrum-master

### Start (New practices to adopt)
- [ ] Action 1 [owner: agent-name] [due: Sprint 5]
- [ ] Action 2

### Stop (Practices to eliminate)
- [ ] Action 1 [owner: agent-name] [due: Sprint 5]
- [ ] Action 2

### Continue (Effective practices to maintain)
- Practice 1
- Practice 2

### Metrics
- Velocity: [X] points (planned: [Y])
- Completion rate: [Z]%
- Application coverage: [A]%
- HTTP coverage: [B]%
- E2E tests: [C] tests

### Key Insights
- Insight 1
- Insight 2
```

---

## Risk Assessment

### High Priority Risks

**Risk 1: Rate Limiting Implementation Complexity**
- **Description**: Redis-backed sliding window rate limiting is complex and prone to race conditions
- **Impact**: High - Rate limiting is a P0 security control
- **Probability**: Medium
- **Mitigation**:
  - Use proven library (e.g., go-redis/redis-rate)
  - Implement atomic Lua scripts in Redis
  - Load test with k6 or vegeta
  - senior-secops-engineer reviews implementation
- **Contingency**: If blocked, implement simpler fixed-window algorithm, document as technical debt

**Risk 2: Account Enumeration Prevention**
- **Description**: Difficult to prevent timing attacks on login endpoint
- **Impact**: Medium - Security vulnerability
- **Probability**: Medium
- **Mitigation**:
  - Use constant-time password comparison
  - Add random delay to login response (100-200ms)
  - Return identical errors for valid/invalid email
  - senior-secops-engineer validates implementation
- **Contingency**: Document as known limitation, implement progressive delays

**Risk 3: E2E Test Flakiness**
- **Description**: Newman tests may be flaky due to timing issues or container startup
- **Impact**: Medium - Blocks merge if tests fail
- **Probability**: Medium
- **Mitigation**:
  - Use testcontainers for consistent environment
  - Add retry logic in Postman pre-request scripts
  - Implement health check polling before tests
  - test-strategist monitors test stability
- **Contingency**: Mark flaky tests as pending, document in issue tracker

**Risk 4: Application Layer Coverage Threshold**
- **Description**: Achieving 85% coverage may require significant effort
- **Impact**: Medium - Quality gate blocks merge
- **Probability**: Low
- **Mitigation**:
  - Start with table-driven tests (high coverage per effort)
  - backend-test-architect monitors coverage daily
  - Focus on command handlers first (higher complexity)
- **Contingency**: Reduce threshold to 80% with scrum-master approval

---

### Medium Priority Risks

**Risk 5: JWT Blacklist Performance**
- **Description**: Redis blacklist lookups add latency to every authenticated request
- **Impact**: Low - May affect response times
- **Probability**: Medium
- **Mitigation**:
  - Use Redis pipelining for batch lookups
  - Set short TTL on blacklist entries (access token TTL)
  - Monitor Redis latency in CI
- **Contingency**: Optimize or remove blacklist check for non-sensitive endpoints

**Risk 6: CORS Configuration Complexity**
- **Description**: Environment-specific CORS origins may cause deployment issues
- **Impact**: Low - Frontend integration may fail
- **Probability**: Low
- **Mitigation**:
  - Document CORS configuration in deployment guide
  - Add validation on startup (fail fast if invalid)
  - Test CORS in integration tests
- **Contingency**: Allow wildcard in development/staging, strict in production

---

### Low Priority Risks

**Risk 7: OpenAPI Spec Drift**
- **Description**: Implementation may diverge from OpenAPI spec
- **Impact**: Low - API contract broken
- **Probability**: Low
- **Mitigation**:
  - Run `make validate-openapi` in CI on every commit
  - Use oapi-codegen to generate server stubs
  - senior-go-architect reviews spec alignment
- **Contingency**: Update spec to match implementation (if justifiable)

**Risk 8: Audit Logging Performance**
- **Description**: Logging every auth event may impact performance
- **Impact**: Low
- **Probability**: Low
- **Mitigation**:
  - Use asynchronous logging (zerolog default)
  - Batch log writes
  - Monitor logging overhead in load tests
- **Contingency**: Reduce log verbosity for high-frequency events

---

### Dependencies and Blockers

**External Dependencies**:
- [ ] PostgreSQL 16+ available (docker-compose)
- [ ] Redis 7+ available (docker-compose)
- [ ] OpenAPI spec defines auth endpoints (verified)
- [ ] Sprint 3 deliverables complete (verified)

**Inter-Sprint Dependencies**:
- Sprint 3 → Sprint 4: JWT service, repositories, Redis client (RESOLVED)
- Sprint 4 → Sprint 5: Auth middleware for image upload endpoints (WILL BLOCK Sprint 5 if not complete)

**Agent Dependencies**:
- test-strategist → senior-go-architect: E2E tests depend on handler implementation
- senior-secops-engineer → senior-go-architect: Security review depends on implementation complete
- backend-test-architect → senior-go-architect: Application tests depend on handler structure

---

## Success Metrics

### Sprint Completion Criteria

**Must Have (Blocking)**:
- [x] All P0 tasks completed
- [x] Application layer coverage >= 85%
- [x] HTTP layer coverage >= 75%
- [x] Race detector clean (`go test -race ./internal/application/...`)
- [x] Newman E2E tests passing (100% auth flow coverage)
- [x] Security gate S4 passed
- [x] All automated tests green
- [x] OpenAPI spec aligned with implementation

**Should Have (Non-Blocking)**:
- [ ] All P1 tasks completed
- [ ] Audit logging for all auth events
- [ ] Account lockout after 5 failed attempts
- [ ] Rate limiting validated under load

**Nice to Have**:
- [ ] P2 tasks completed
- [ ] Session management endpoints

---

### Quality Metrics

**Code Quality**:
- `golangci-lint run` passes with zero errors
- `go vet ./...` passes
- `gosec ./...` passes (zero critical/high findings)
- No code duplication >15 lines

**Test Quality**:
- Application layer: 85%+ coverage
- HTTP layer: 75%+ coverage
- Domain layer: 95%+ coverage (from Sprint 3)
- Zero flaky tests
- All tests use t.Parallel() where possible

**Security Quality**:
- All security gate S4 controls passed
- Zero account enumeration vulnerabilities
- Account lockout functional
- Rate limiting enforced
- Security headers present
- No sensitive data in logs

**API Quality**:
- 100% OpenAPI compliance
- RFC 7807 Problem Details for all errors
- Consistent response structure
- Proper HTTP status codes

---

## Definition of Done

### Per Task

- [x] Code implemented following DDD patterns
- [x] Unit tests written (table-driven, mocked dependencies)
- [x] Integration tests written (testcontainers)
- [x] Code reviewed by senior-go-architect
- [x] Security reviewed by senior-secops-engineer (if auth-related)
- [x] Documentation updated (if needed)
- [x] OpenAPI spec updated (if HTTP changes)
- [x] Agent checklist verified

### Per Sprint

- [x] All P0 tasks complete
- [x] All automated tests passing
- [x] Coverage thresholds met
- [x] Security gate S4 passed
- [x] Newman E2E tests passing
- [x] CI pipeline green
- [x] OpenAPI spec aligned
- [x] Sprint demo ready
- [x] Technical debt documented
- [x] Sprint retrospective complete

---

## Communication Protocol

### Daily Stand-up (Async)

Each active agent posts daily update by 10:00 AM:

```markdown
### [Agent Name] - [Date]

**Yesterday**:
- Completed: [task description]
- Progress: [task description] ([X]% complete)

**Today**:
- Plan: [task description]

**Blockers**:
- [Blocker description] [requires: agent/resource]
```

### Sprint Summary (Weekly)

scrum-master posts on Day 7:

```markdown
## Sprint 4 Summary - Week 1

**Sprint Goal**: Deliver fully functional authentication and user management API
**Status**: [On Track / At Risk / Off Track]

### Completed This Week
- [x] Task 1 [agent: name]
- [x] Task 2 [agent: name]

### In Progress
- [ ] Task 3 [agent: name] ([X]% complete)
- [ ] Task 4 [agent: name] ([X]% complete)

### Blocked
- [ ] Task 5 [blocker: description] [requires: resource]

### Risks
- Risk 1: [description] [impact: High/Medium/Low] [mitigation: plan]

### Metrics
- Sprint progress: [X]% complete (expected: 50%)
- Application coverage: [Y]% (target: 85%)
- HTTP coverage: [Z]% (target: 75%)
- Blockers: [N] (target: 0)
```

---

## References

**Project Documentation**:
- Sprint Plan: `/home/user/goimg-datalayer/claude/sprint_plan.md`
- MVP Features: `/home/user/goimg-datalayer/claude/mvp_features.md`
- Agent Workflow: `/home/user/goimg-datalayer/claude/agent_workflow.md`
- Architecture: `/home/user/goimg-datalayer/claude/architecture.md`
- API Security: `/home/user/goimg-datalayer/claude/api_security.md`
- Security Gates: `/home/user/goimg-datalayer/claude/security_gates.md`
- Agent Checklist: `/home/user/goimg-datalayer/claude/agent_checklist.md`

**OpenAPI Specification**:
- Main Spec: `/home/user/goimg-datalayer/api/openapi/openapi.yaml`

**Testing**:
- Postman Collection: `/home/user/goimg-datalayer/tests/e2e/postman/goimg-api.postman_collection.json`
- CI Environment: `/home/user/goimg-datalayer/tests/e2e/postman/ci.postman_environment.json`

---

## Appendix: Agent Quick Reference

### Task Assignment Quick Matrix

| Task Type | Primary Agent | Review By |
|-----------|---------------|-----------|
| Application command/query | Direct implementation | senior-go-architect |
| HTTP handler | Direct implementation | senior-go-architect |
| Security middleware | senior-secops-engineer | senior-go-architect |
| Unit tests | backend-test-architect | scrum-master |
| Integration tests | backend-test-architect | scrum-master |
| E2E tests | test-strategist | backend-test-architect |
| CI/CD issues | cicd-guardian | scrum-master |
| Security gate review | senior-secops-engineer | scrum-master |

### Escalation Path

**Level 1: Self-Resolution** (0-4 hours)
**Level 2: Peer Agent Assistance** (4-24 hours)
**Level 3: Scrum Master Coordination** (24-48 hours)
**Level 4: Stakeholder Escalation** (48+ hours)

### Key Commands

```bash
# Development
make run              # Start API server
make test             # Run all tests
make lint             # Run linters
make validate-openapi # Validate OpenAPI spec
make test-e2e         # Run Newman E2E tests

# Docker
docker-compose -f docker/docker-compose.yml up -d  # Start services

# Testing
go test -race ./internal/application/...           # Application layer tests
go test -race ./internal/interfaces/http/...       # HTTP layer tests
go test -cover ./...                               # Coverage report

# Security
gosec ./...                                        # Security scan
trivy fs --severity HIGH,CRITICAL .                # Dependency scan
```

---

**Sprint 4 Kickoff Date**: 2025-12-03
**Prepared By**: scrum-master
**Version**: 1.0
**Status**: APPROVED

---

## Sign-Off

**senior-go-architect**: _________________ Date: _______
**senior-secops-engineer**: _________________ Date: _______
**backend-test-architect**: _________________ Date: _______
**cicd-guardian**: _________________ Date: _______
**test-strategist**: _________________ Date: _______
**scrum-master**: _________________ Date: _______
