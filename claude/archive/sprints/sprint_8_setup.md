# Sprint 8: Integration, Testing & Security Hardening - Detailed Plan

> **Status**: SETUP COMPLETE - Ready for execution
> **Duration**: 2 weeks
> **Sprint Goal**: Achieve comprehensive test coverage, security hardening, and performance baseline for MVP launch readiness

---

## Executive Summary

### Current State Baseline

**Test Coverage** (as of Sprint 8 start):
- ✅ **Domain Layer**: 94.1% average (91.4% gallery, 96.5% identity, 100% moderation, 97.5% shared)
  - **Status**: EXCEEDS 90% requirement - excellent foundation
  - **Test Files**: 31 test files with comprehensive table-driven tests
- ⚠️ **Application Layer**: 17 test files, but incomplete coverage (build failures)
  - **Status**: NEEDS WORK - target 85%+ coverage
  - **Gaps**: Commands and queries have incomplete mocks/tests
- ⚠️ **Infrastructure Layer**: Mixed (78.9% local storage, 96.3% validator, 38.3% JWT, 7.9% Redis)
  - **Status**: NEEDS WORK - target 70%+ coverage
  - **Gaps**: Missing integration tests with testcontainers
- ❌ **HTTP Handlers**: 1 test file, build failures
  - **Status**: CRITICAL GAP - target 75%+ coverage
  - **Gaps**: No comprehensive handler tests
- ✅ **Integration Tests**: Infrastructure exists with testcontainers
  - **Status**: PARTIAL - needs expansion
  - **Files**: user_repository_test.go, session_repository_test.go, gallery_repository_test.go
- ✅ **E2E Tests**: Postman collection (2,133 lines, 30+ requests)
  - **Status**: GOOD FOUNDATION - needs expansion for gallery/moderation
  - **Infrastructure**: Newman CI integration complete

**CI/CD Security** (already in place):
- ✅ gosec - Go security scanner (SARIF output to GitHub Security tab)
- ✅ trivy - Vulnerability scanner (filesystem + config)
- ✅ govulncheck - Official Go vulnerability checker
- ✅ gitleaks - Secret detection (full history scan)
- ✅ CodeQL - Advanced semantic analysis
- ✅ SBOM generation (Syft + Grype)
- ✅ Dependency review (PRs only)

**Performance Baseline**: NOT YET ESTABLISHED

---

## Sprint 8 Deliverables (Detailed)

### 1. Testing Excellence (60% of sprint)

#### 1.1 Application Layer Tests (Priority: P0)
**Owner**: backend-test-architect
**Target Coverage**: 85%+

**Tasks**:
- [ ] **Commands Testing** (6-8 hours)
  - RegisterUserCommand (mock UserRepository, validate business rules)
  - LoginCommand (mock auth service, test rate limiting logic)
  - RefreshTokenCommand (test token rotation)
  - UploadImageCommand (mock storage provider, ClamAV service)
  - CreateAlbumCommand (test ownership validation)
  - AddCommentCommand (test sanitization)

- [ ] **Queries Testing** (4-6 hours)
  - GetUserQuery (mock repository)
  - GetImageQuery (test visibility rules)
  - ListImagesQuery (test pagination, filters)
  - SearchImagesQuery (test query building)
  - GetUserSessionsQuery (test filtering)

- [ ] **Integration with Domain** (2-3 hours)
  - Verify application services correctly call domain aggregates
  - Test error propagation from domain to application layer
  - Validate domain events are emitted and handled

**Acceptance Criteria**:
- All command handlers have 85%+ coverage
- All query handlers have 85%+ coverage
- Mocks properly validate interactions
- Error scenarios comprehensively tested
- No business logic in tests (test behavior, not implementation)

---

#### 1.2 HTTP Handler Tests (Priority: P0)
**Owner**: backend-test-architect (primary), test-strategist (E2E scenarios)
**Target Coverage**: 75%+

**Tasks**:
- [ ] **Auth Handlers** (6-8 hours)
  - POST /auth/register (201, 400, 409 scenarios)
  - POST /auth/login (200, 401, 429 rate limit)
  - POST /auth/refresh (200, 401 expired)
  - POST /auth/logout (204, 401)
  - Test middleware integration (auth, rate limit, error handler)

- [ ] **User Handlers** (4-6 hours)
  - GET /users/{id} (200, 404, 403 ownership)
  - PUT /users/{id} (200, 400 validation, 403 ownership)
  - DELETE /users/{id} (204, 403 ownership)
  - GET /users/{id}/sessions (200, 403)
  - Test IDOR prevention

- [ ] **Image Handlers** (8-10 hours)
  - POST /images (201 multipart upload, 400, 413 too large, 415 unsupported)
  - GET /images/{id} (200, 404, 403 visibility)
  - PUT /images/{id} (200, 403 ownership)
  - DELETE /images/{id} (204, 403)
  - GET /images (200 list with pagination)
  - Test file upload edge cases

- [ ] **Album Handlers** (4-6 hours)
  - POST /albums (201, 400)
  - GET /albums/{id} (200, 404, 403)
  - PUT /albums/{id} (200, 403)
  - DELETE /albums/{id} (204, 403)
  - POST /albums/{id}/images (201, 404 album/image)

- [ ] **Middleware Tests** (4-6 hours)
  - Auth middleware (401 missing token, 401 invalid)
  - Rate limit middleware (429 after threshold)
  - CORS middleware (preflight, headers)
  - Security headers middleware (all required headers present)
  - Error handler middleware (RFC 7807 format)

**Acceptance Criteria**:
- All handlers return correct HTTP status codes
- All error responses follow RFC 7807 Problem Details format
- All authenticated endpoints reject missing/invalid tokens
- All ownership checks prevent IDOR
- httptest.ResponseRecorder used for isolation
- No database/Redis dependencies (mocked services)

---

#### 1.3 Integration Tests with Testcontainers (Priority: P0)
**Owner**: backend-test-architect
**Target**: Comprehensive repository + service integration

**Tasks**:
- [ ] **PostgreSQL Repository Tests** (8-10 hours)
  - UserRepository (CRUD, FindByEmail, ExistsByUsername, pagination)
  - SessionRepository (CRUD, FindActiveByUserID, DeleteExpired)
  - ImageRepository (CRUD, FindByOwner, Search, UpdateStatus)
  - AlbumRepository (CRUD, AddImage, RemoveImage, ListImages)
  - CommentRepository (CRUD, FindByImage, pagination)
  - LikeRepository (AddLike, RemoveLike, CountByImage, HasUserLiked)
  - ReportRepository (CRUD, FindPending, UpdateStatus)
  - Test transaction rollback scenarios
  - Test concurrent access (race conditions)

- [ ] **Redis Service Tests** (4-6 hours)
  - SessionStore (Set, Get, Delete, Exists)
  - TokenBlacklist (Add, IsBlacklisted, Cleanup)
  - RateLimiter (Allow, Increment, Reset)
  - Test TTL/expiration
  - Test key collisions

- [ ] **ClamAV Integration Tests** (2-3 hours)
  - Scan clean file (should pass)
  - Scan EICAR test file (should detect malware)
  - Test timeout scenarios
  - Test daemon unavailable graceful degradation

**Acceptance Criteria**:
- All integration tests use testcontainers (PostgreSQL 16, Redis 7)
- Tests clean up database between runs (TRUNCATE tables)
- Tests are tagged with `//go:build integration`
- Tests can run in parallel where safe
- All tests pass with `-race` flag
- Integration tests excluded from unit test runs (`go test -short`)

---

#### 1.4 E2E Tests with Newman/Postman (Priority: P1)
**Owner**: test-strategist
**Target**: 100% API endpoint coverage

**Tasks**:
- [ ] **Expand Auth Flow Tests** (2-3 hours)
  - Add MFA scenarios (if implemented)
  - Add password reset flow
  - Add OAuth flows (if implemented)
  - Test session expiration/refresh

- [ ] **Gallery E2E Tests** (6-8 hours)
  - Upload image workflow (register → login → upload → verify variants)
  - Album management (create → add images → list → delete)
  - Image visibility (private → public → unlisted)
  - Search functionality (by title, tags, owner)
  - Like/unlike workflow
  - Comment CRUD workflow

- [ ] **Moderation E2E Tests** (4-6 hours)
  - Report workflow (user reports → moderator reviews → action taken)
  - Ban user workflow (admin bans → verify user can't login)
  - Admin panel access (test RBAC)

- [ ] **Error Scenario Tests** (2-3 hours)
  - 400 Bad Request (invalid JSON, validation errors)
  - 401 Unauthorized (missing/expired token)
  - 403 Forbidden (RBAC, ownership)
  - 404 Not Found (non-existent resources)
  - 409 Conflict (duplicate email/username)
  - 413 Payload Too Large (oversized upload)
  - 415 Unsupported Media Type (wrong file type)
  - 429 Too Many Requests (rate limit)
  - 500 Internal Server Error (simulate failures)

- [ ] **Newman CI Integration** (2 hours)
  - Update ci.postman_environment.json with all required variables
  - Add pre-request scripts for auth token management
  - Add test scripts for all responses (status, schema, business rules)
  - Verify Newman job runs successfully in GitHub Actions

**Acceptance Criteria**:
- Postman collection covers 100% of OpenAPI endpoints
- All requests have test scripts validating:
  - HTTP status codes
  - Response body structure (JSON schema)
  - Business logic assertions (e.g., created resource exists)
  - RFC 7807 error format
- Collection uses variables for environment-specific config
- Pre-request scripts handle token refresh automatically
- Newman reports uploaded as CI artifacts

---

#### 1.5 Contract Tests (OpenAPI Compliance) (Priority: P1)
**Owner**: test-strategist
**Target**: 100% OpenAPI spec alignment

**Tasks**:
- [ ] **OpenAPI Validation Framework** (4-6 hours)
  - Set up kin-openapi for runtime validation
  - Create contract test harness
  - Test all request schemas match spec
  - Test all response schemas match spec
  - Test all parameter validations match spec
  - Test security schemes match spec

- [ ] **Generate Test Cases from OpenAPI** (3-4 hours)
  - Use oapi-codegen to generate test stubs
  - Validate generated code matches implementation
  - Test for spec drift (CI check)

**Acceptance Criteria**:
- All API requests/responses validated against OpenAPI spec
- Contract tests fail if API diverges from spec
- CI job enforces OpenAPI drift detection
- OpenAPI spec is 100% accurate (no manual edits to generated code)

---

#### 1.6 Security Test Suite (OWASP Top 10) (Priority: P0)
**Owner**: senior-secops-engineer (lead), backend-test-architect (implementation)
**Target**: Comprehensive security test coverage

**Tasks**:
- [ ] **A01: Broken Access Control** (6-8 hours)
  - Test IDOR on all endpoints (user tries to access other user's resources)
  - Test vertical privilege escalation (user → moderator → admin)
  - Test horizontal privilege escalation (user A → user B)
  - Test missing function-level access control
  - Test forced browsing to admin endpoints

- [ ] **A02: Cryptographic Failures** (3-4 hours)
  - Verify Argon2id password hashing
  - Verify JWT RS256 signing
  - Verify refresh tokens stored hashed
  - Verify sensitive data not logged
  - Verify TLS configuration (if applicable)

- [ ] **A03: Injection** (6-8 hours)
  - SQL injection payloads in all inputs (queries, filters, search)
  - Command injection in file processing
  - XSS payloads in comments, titles, descriptions
  - Path traversal in file uploads/downloads
  - LDAP injection (if applicable)

- [ ] **A04: Insecure Design** (4-6 hours)
  - Test account lockout after failed logins
  - Test rate limiting under load
  - Test password policy enforcement
  - Test session timeout
  - Test replay attack prevention (refresh token rotation)

- [ ] **A05: Security Misconfiguration** (3-4 hours)
  - Verify security headers (CSP, HSTS, X-Frame-Options, etc.)
  - Verify no stack traces in production responses
  - Verify no default credentials
  - Verify CORS configuration restrictive
  - Verify error messages don't leak sensitive info

- [ ] **A07: Identification and Authentication Failures** (6-8 hours)
  - Test account enumeration prevention (generic error messages)
  - Test token replay detection
  - Test credential stuffing protection
  - Test weak password rejection
  - Test session fixation prevention

- [ ] **A08: Software and Data Integrity Failures** (4-6 hours)
  - Test file upload MIME type validation (magic bytes, not extension)
  - Test polyglot file detection and re-encoding
  - Test EXIF metadata stripping
  - Test malware scanning (EICAR test file)

- [ ] **A09: Security Logging and Monitoring Failures** (3-4 hours)
  - Verify all auth events logged
  - Verify all authz failures logged
  - Verify no sensitive data in logs (passwords, tokens)
  - Test audit log integrity

- [ ] **A10: Server-Side Request Forgery (SSRF)** (2-3 hours)
  - Test URL validation if fetching images from URL
  - Test localhost/internal IP blocking

**Deliverables**:
- `tests/security/owasp/` directory with test files for each category
- Security test suite runs in CI on every commit
- Security test failures block merge to main
- Security test report uploaded as CI artifact

---

#### 1.7 Load Testing Setup (Priority: P1)
**Owner**: test-strategist
**Target**: Establish performance baseline

**Tasks**:
- [ ] **Select Load Testing Tool** (1-2 hours)
  - Evaluate k6 vs vegeta
  - Decision: k6 (JavaScript-based, Grafana integration)

- [ ] **Create Load Test Scenarios** (6-8 hours)
  - Authentication flow (register → login → refresh)
  - Image upload (10MB files, 50 concurrent users)
  - Image listing/search (paginated queries)
  - Album operations (CRUD)
  - Mixed workload (realistic user behavior)

- [ ] **Establish Performance Baselines** (4-6 hours)
  - P50, P95, P99 latencies for each endpoint
  - Throughput (requests/second)
  - Error rate under load
  - Resource utilization (CPU, memory, connections)

- [ ] **CI Integration** (2-3 hours)
  - Add load test job to CI (runs nightly or on-demand)
  - Store performance metrics over time
  - Alert on performance regressions (>20% latency increase)

**Acceptance Criteria**:
- Load tests can simulate 100 concurrent users
- Load tests validate response correctness (not just speed)
- Performance baselines documented for all critical endpoints
- Load test results tracked over time (detect regressions)

**Performance Targets** (initial baselines):
- API response time: P95 < 200ms (excluding uploads)
- Image upload: < 30 seconds for 10MB
- Database queries: P95 < 50ms
- Availability: 99.9% under normal load

---

### 2. Security Hardening (25% of sprint)

#### 2.1 Security Scanning in CI (Already Complete)
**Owner**: cicd-guardian
**Status**: ✅ COMPLETE (verified in .github/workflows/security.yml)

**Existing Coverage**:
- ✅ gosec - SARIF output to GitHub Security tab
- ✅ trivy - filesystem + config scan
- ✅ govulncheck - Go vulnerability database
- ✅ gitleaks - secret detection
- ✅ CodeQL - semantic analysis
- ✅ SBOM generation (Syft + Grype)

**Validation Tasks** (no implementation needed):
- [x] Verify all scans run on every commit
- [x] Verify SARIF results upload to GitHub Security tab
- [x] Verify scan failures block merge to main
- [x] Verify weekly scheduled scan runs

---

#### 2.2 Dependency Vulnerability Check (Priority: P1)
**Owner**: senior-secops-engineer
**Tasks**:
- [ ] **Manual Vulnerability Audit** (3-4 hours)
  - Review govulncheck results
  - Review trivy dependency scan results
  - Categorize vulnerabilities (critical, high, medium, low)
  - Create remediation plan for high/critical
  - Document accepted risks for medium/low (with justification)

- [ ] **Dependency Update Strategy** (2-3 hours)
  - Identify outdated dependencies
  - Test updates in isolated branch
  - Verify no breaking changes
  - Update go.mod and go.sum
  - Run full test suite after updates

**Acceptance Criteria**:
- Zero critical vulnerabilities
- Zero high vulnerabilities (or documented mitigation)
- All dependencies < 1 year old
- Vulnerability report documented in `docs/security/vulnerability-audit.md`

---

#### 2.3 Penetration Testing (Manual) (Priority: P1)
**Owner**: senior-secops-engineer
**Tasks**:
- [ ] **Authentication Penetration Tests** (4-6 hours)
  - Brute force attack (verify rate limiting)
  - Credential stuffing (verify account lockout)
  - Token theft/replay (verify token rotation)
  - Session fixation (verify regeneration on login)
  - OAuth vulnerabilities (if implemented)

- [ ] **Authorization Penetration Tests** (4-6 hours)
  - IDOR (test all endpoints with different user IDs)
  - Privilege escalation (user → moderator → admin)
  - Missing access controls (unauthenticated access)
  - RBAC bypass attempts

- [ ] **Input Validation Penetration Tests** (4-6 hours)
  - SQL injection (all input fields)
  - XSS (stored, reflected, DOM-based)
  - Command injection (file processing)
  - Path traversal (file uploads/downloads)

- [ ] **File Upload Penetration Tests** (4-6 hours)
  - Malware upload (EICAR, actual malware samples)
  - Polyglot files (JPEG+ZIP, JPEG+HTML)
  - Oversized files (pixel flood, decompression bomb)
  - MIME type bypass (rename .exe to .jpg)
  - Double extension bypass (image.jpg.exe)

- [ ] **Business Logic Penetration Tests** (3-4 hours)
  - Race conditions (concurrent likes, comments)
  - Integer overflow (large file sizes, IDs)
  - State machine bypass (skip image processing)
  - Payment bypass (if applicable)

**Deliverables**:
- Penetration test report (`docs/security/pentest-report.md`)
- List of vulnerabilities found (CVE-style)
- Remediation recommendations
- Retesting after fixes

---

#### 2.4 Rate Limiting Validation Under Load (Priority: P2)
**Owner**: test-strategist (test design), senior-secops-engineer (validation)
**Tasks**:
- [ ] **Login Rate Limit Test** (2-3 hours)
  - Send 10 requests in < 1 minute (expect 429 after 5th)
  - Verify Retry-After header
  - Verify rate limit resets after window

- [ ] **Global Rate Limit Test** (2-3 hours)
  - Send 150 requests/minute from same IP (expect 429 after 100th)
  - Verify different IPs have separate limits

- [ ] **Authenticated Rate Limit Test** (2-3 hours)
  - Send 350 requests/minute with valid token (expect 429 after 300th)
  - Verify rate limit is per user, not per IP

- [ ] **Upload Rate Limit Test** (2-3 hours)
  - Upload 60 images in 1 hour (expect 429 after 50th)
  - Verify rate limit persists across sessions

**Acceptance Criteria**:
- All rate limits enforced under load
- Rate limit bypass attempts detected
- Rate limit counters accurate (no off-by-one errors)
- Rate limit resets work correctly

---

#### 2.5 Token Revocation Verification (Priority: P2)
**Owner**: senior-secops-engineer
**Tasks**:
- [ ] **Logout Revocation Test** (1-2 hours)
  - Login → get token → logout → verify token blacklisted
  - Attempt to use blacklisted token (expect 401)

- [ ] **Refresh Token Revocation Test** (1-2 hours)
  - Refresh token → verify old token invalidated
  - Attempt token replay (expect 401 + family revocation)

- [ ] **User Deletion Revocation Test** (1-2 hours)
  - Delete user → verify all tokens revoked
  - Attempt to use token after deletion (expect 401)

- [ ] **Ban Revocation Test** (1-2 hours)
  - Ban user → verify all tokens revoked
  - Banned user attempts login (expect 403)

**Acceptance Criteria**:
- All revocation scenarios tested
- Revoked tokens cannot be used
- Token blacklist cleanup works (expired tokens removed)
- No token reuse possible

---

#### 2.6 Audit Log Review (Priority: P2)
**Owner**: senior-secops-engineer
**Tasks**:
- [ ] **Audit Log Completeness** (2-3 hours)
  - Verify all auth events logged (login, logout, refresh, failed attempts)
  - Verify all authz failures logged (403 responses)
  - Verify all moderation actions logged (ban, report resolution)
  - Verify all sensitive operations logged (role change, data export)

- [ ] **Audit Log Format** (1-2 hours)
  - Verify structured logging (JSON format)
  - Verify required fields (timestamp, user ID, action, IP, user-agent)
  - Verify no sensitive data logged (passwords, tokens)

- [ ] **Audit Log Integrity** (1-2 hours)
  - Verify logs cannot be tampered with (append-only)
  - Verify log retention policy
  - Verify log export capability

**Acceptance Criteria**:
- All security-relevant events logged
- Logs are machine-readable (JSON)
- Logs contain sufficient context for forensics
- No sensitive data in logs
- Audit logs pass compliance review

---

### 3. Performance Optimization (15% of sprint)

#### 3.1 Database Query Optimization (Priority: P1)
**Owner**: senior-go-architect
**Tasks**:
- [ ] **Query Analysis** (2-3 hours)
  - Enable PostgreSQL query logging
  - Identify slow queries (> 50ms)
  - Run EXPLAIN ANALYZE on slow queries
  - Identify missing indexes

- [ ] **Index Analysis and Tuning** (4-6 hours)
  - Add indexes for frequently queried columns:
    - `users.email` (already unique index)
    - `users.username` (already unique index)
    - `sessions.user_id, sessions.expires_at` (composite)
    - `images.owner_id, images.created_at` (composite for pagination)
    - `images.status, images.visibility` (composite for filtering)
    - `album_images.album_id, album_images.added_at` (composite)
    - `comments.image_id, comments.created_at` (composite)
    - `reports.status, reports.created_at` (composite)
  - Verify index usage with EXPLAIN
  - Remove unused indexes (increase write overhead)

- [ ] **Query Optimization** (4-6 hours)
  - Optimize N+1 queries (use JOINs or batching)
  - Optimize search queries (full-text search indexes)
  - Optimize pagination queries (cursor-based for large datasets)
  - Add partial indexes for common filters (status = 'active')

- [ ] **Benchmark Improvements** (2-3 hours)
  - Re-run load tests after optimizations
  - Verify P95 latency improvements (target: 30%+ reduction)
  - Document query performance before/after

**Acceptance Criteria**:
- All queries < 50ms P95
- No N+1 queries
- Index hit rate > 95%
- Query optimization documented in ADR

---

#### 3.2 Connection Pool Tuning (Priority: P2)
**Owner**: senior-go-architect
**Tasks**:
- [ ] **PostgreSQL Connection Pool** (2-3 hours)
  - Benchmark different pool sizes (10, 25, 50, 100)
  - Monitor connection wait times
  - Set MaxOpenConns based on CPU cores and workload
  - Set MaxIdleConns to reduce connection churn
  - Set ConnMaxLifetime to handle load balancer timeouts

- [ ] **Redis Connection Pool** (1-2 hours)
  - Benchmark different pool sizes
  - Set PoolSize based on concurrency requirements
  - Set MinIdleConns for warm connections
  - Set PoolTimeout for failfast behavior

**Acceptance Criteria**:
- Connection pool sizes documented
- No connection exhaustion under load
- Connection wait times < 10ms P95

---

#### 3.3 Cache Strategy Implementation (Priority: P2)
**Owner**: senior-go-architect
**Tasks**:
- [ ] **Identify Cacheable Data** (2-3 hours)
  - User profiles (infrequently changed)
  - Image metadata (read-heavy)
  - Album listings (read-heavy)
  - Public image counts (read-heavy)

- [ ] **Implement Cache Layer** (6-8 hours)
  - Add cache middleware (Redis-backed)
  - Add cache-aside pattern for reads
  - Add cache invalidation on writes
  - Add cache warming for popular data
  - Add cache TTL strategy (short for user data, long for static)

- [ ] **Cache Hit Rate Monitoring** (2-3 hours)
  - Add cache hit/miss metrics
  - Monitor cache eviction rate
  - Tune cache size based on hit rate
  - Target: 80%+ cache hit rate for reads

**Acceptance Criteria**:
- Cache hit rate > 80% for cacheable endpoints
- Cache invalidation works correctly
- No stale data served
- Cache improves P95 latency by 50%+ for cached endpoints

---

#### 3.4 Response Time Benchmarks (Priority: P1)
**Owner**: test-strategist
**Tasks**:
- [ ] **Establish Baseline** (3-4 hours)
  - Benchmark all API endpoints (no load)
  - Record P50, P95, P99 latencies
  - Document resource utilization (CPU, memory, DB connections)

- [ ] **Load Testing** (4-6 hours)
  - Run load tests at 10, 50, 100, 200 concurrent users
  - Identify breaking points (where latency spikes)
  - Measure throughput (requests/second)
  - Monitor error rates

- [ ] **Performance Report** (2-3 hours)
  - Document all benchmarks
  - Create performance dashboard (Grafana)
  - Set performance alerts (>20% regression)

**Acceptance Criteria**:
- All endpoints benchmarked
- Performance baselines documented
- Performance dashboard created
- Performance alerts configured

---

## Agent Assignments Summary

| Agent | Primary Responsibility | Estimated Hours | Deliverables |
|-------|------------------------|-----------------|--------------|
| **backend-test-architect** (Lead) | Unit & integration tests | 50-60 hours | Application tests (85%+), handler tests (75%+), integration tests |
| **test-strategist** | E2E, contract, load tests | 30-40 hours | Expanded Postman collection, contract tests, k6 load tests, benchmarks |
| **senior-secops-engineer** | Security tests & pentest | 35-45 hours | OWASP test suite, pentest report, vulnerability audit, rate limit validation |
| **senior-go-architect** | Performance optimization | 20-25 hours | Query optimization, connection pool tuning, cache strategy |
| **cicd-guardian** | CI/CD validation | 5-10 hours | Verify security scans, add load test job, performance tracking |
| **scrum-master** | Coordination & reporting | 10-15 hours | Sprint tracking, quality gate validation, final report |

---

## Work Breakdown Structure (WBS)

### Week 1: Testing Foundation

**Days 1-2** (Mon-Tue):
- [ ] backend-test-architect: Application layer tests setup
- [ ] test-strategist: E2E test scenario design
- [ ] senior-secops-engineer: OWASP A01-A03 test implementation
- [ ] senior-go-architect: Query analysis and indexing

**Days 3-5** (Wed-Fri):
- [ ] backend-test-architect: HTTP handler tests (auth, user, image)
- [ ] test-strategist: Expand Postman collection (gallery, moderation)
- [ ] senior-secops-engineer: OWASP A04-A08 test implementation, pentest prep
- [ ] senior-go-architect: Query optimization, connection pool tuning

### Week 2: Integration & Validation

**Days 6-7** (Mon-Tue):
- [ ] backend-test-architect: Integration tests with testcontainers
- [ ] test-strategist: Contract tests, k6 load test setup
- [ ] senior-secops-engineer: Manual penetration testing
- [ ] senior-go-architect: Cache strategy implementation

**Days 8-9** (Wed-Thu):
- [ ] backend-test-architect: Coverage gap closure, test refinement
- [ ] test-strategist: Load testing, performance baselines
- [ ] senior-secops-engineer: Security validation, audit log review
- [ ] senior-go-architect: Performance optimization validation

**Day 10** (Fri):
- [ ] All agents: Final quality gate review
- [ ] scrum-master: Sprint report generation
- [ ] All agents: Retrospective participation

---

## Quality Gates

### Pre-Sprint Planning Gate ✅
- [x] Previous sprint retrospective actions completed
- [x] Sprint 8 backlog refined and estimated
- [x] Dependencies from Sprint 7 resolved
- [x] Team capacity calculated
- [x] Sprint goal aligns with MVP roadmap

### Mid-Sprint Checkpoint (Day 5)
- [ ] Application layer tests >= 70% coverage (on track for 85%)
- [ ] HTTP handler tests >= 50% coverage (on track for 75%)
- [ ] OWASP A01-A03 tests complete
- [ ] No critical blockers unresolved > 24 hours
- [ ] Sprint burndown within 10% of ideal

### Pre-Merge Quality Gate (End of Sprint)
**Automated**:
- [ ] Overall coverage >= 80%
- [ ] Domain coverage >= 90% (already met)
- [ ] Application coverage >= 85%
- [ ] Handler coverage >= 75%
- [ ] All tests pass with `-race` flag
- [ ] gosec scan: zero critical/high findings
- [ ] trivy scan: zero critical vulnerabilities
- [ ] gitleaks scan: zero secrets detected
- [ ] OpenAPI validation passes
- [ ] Newman E2E tests pass (100% endpoints covered)

**Manual**:
- [ ] Penetration test complete (report signed off)
- [ ] Security test suite passing (OWASP Top 10 coverage)
- [ ] Performance benchmarks established
- [ ] Load tests passing at 100 concurrent users
- [ ] Rate limiting validated under load
- [ ] Token revocation verified
- [ ] Audit log completeness verified
- [ ] Agent checklist verified by scrum-master

---

## Sprint Ceremonies

### Sprint Planning (Day 1, 2 hours)
**Attendees**: All agents
**Agenda**:
1. Review Sprint 8 goals and deliverables
2. Clarify acceptance criteria for each task
3. Confirm agent assignments and capacity
4. Identify dependencies and risks
5. Commit to sprint scope

### Daily Standup (Async, 15 minutes)
**Format**:
```markdown
### [Agent Name] - [Date]
**Yesterday**: Completed application command tests (RegisterUser, Login)
**Today**: Implementing query handler tests (GetUser, ListImages)
**Blockers**: None
**Progress**: Application layer coverage now at 60% (target 85%)
```

### Mid-Sprint Checkpoint (Day 5, 1 hour)
**Attendees**: All agents
**Agenda**:
1. Review progress toward sprint goal
2. Coverage metrics review
3. Address blockers
4. Adjust assignments if needed

### Sprint Review (Day 10, 1.5 hours)
**Attendees**: All agents + stakeholders
**Agenda**:
1. Demo test coverage improvements
2. Demo security test suite
3. Demo performance benchmarks
4. Review quality gates status
5. Acceptance criteria verification

### Sprint Retrospective (Day 10, 1 hour)
**Attendees**: All agents
**Format**: Start/Stop/Continue
**Agenda**:
1. What went well?
2. What didn't go well?
3. What should we change?
4. Action items for Sprint 9

---

## Risk Register

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Testcontainer setup complexity | High | Medium | Allocate extra time for backend-test-architect, create reusable test helpers |
| Performance optimization uncovers architectural issues | High | Low | Involve senior-go-architect early, document tradeoffs |
| Penetration testing finds critical vulnerabilities | Critical | Medium | Budget time for remediation, defer non-critical to Sprint 9 |
| Coverage targets not met | High | Medium | Daily tracking, adjust scope if needed (defer load testing) |
| E2E tests flaky | Medium | Medium | Invest in test stability, add retries, isolate state |
| Load testing infrastructure unavailable | Medium | Low | Use local k6, defer CI integration to Sprint 9 |

---

## Success Metrics

### Testing Metrics
- **Overall coverage**: >= 80% (CI enforced)
- **Domain coverage**: >= 90% (already met: 94.1%)
- **Application coverage**: >= 85%
- **Handler coverage**: >= 75%
- **Integration test count**: >= 30 tests
- **E2E test count**: >= 50 requests (100% endpoint coverage)
- **Security test count**: >= 100 tests (OWASP Top 10)
- **Test execution time**: < 5 minutes (unit + integration)

### Security Metrics
- **Critical vulnerabilities**: 0
- **High vulnerabilities**: 0 (or documented mitigation)
- **gosec findings**: 0 high-severity
- **trivy findings**: 0 critical
- **Secret leaks**: 0
- **Pentest findings**: All remediated or documented

### Performance Metrics
- **API response time**: P95 < 200ms (excluding uploads)
- **Image upload**: < 30 seconds for 10MB
- **Database queries**: P95 < 50ms
- **Cache hit rate**: > 80% for cached endpoints
- **Throughput**: > 100 req/sec sustained
- **Error rate**: < 0.1% under load

---

## Deliverables Checklist

### Testing
- [ ] Application layer tests (85%+ coverage)
- [ ] HTTP handler tests (75%+ coverage)
- [ ] Integration tests with testcontainers (30+ tests)
- [ ] E2E tests with Newman (50+ requests, 100% endpoint coverage)
- [ ] Contract tests (OpenAPI compliance)
- [ ] Security tests (OWASP Top 10, 100+ tests)
- [ ] Load testing setup (k6 scripts)
- [ ] Test documentation updated

### Security
- [ ] Penetration test report
- [ ] Vulnerability audit report
- [ ] Security test suite (OWASP Top 10)
- [ ] Rate limiting validation report
- [ ] Token revocation verification
- [ ] Audit log completeness report
- [ ] Security runbook updated

### Performance
- [ ] Query optimization (ADR documented)
- [ ] Index analysis and tuning
- [ ] Connection pool tuning (configurations documented)
- [ ] Cache strategy implemented
- [ ] Performance baseline report (all endpoints benchmarked)
- [ ] Load test results (100 concurrent users)
- [ ] Performance dashboard (Grafana)

### Documentation
- [ ] Sprint 8 report
- [ ] Test strategy updated
- [ ] Security testing guide updated
- [ ] Performance tuning guide
- [ ] Agent retrospective notes

---

## Notes for Sprint 9 (MVP Polish & Launch Prep)

**Deferred from Sprint 8** (if time runs short):
- Load testing CI integration (can be manual for Sprint 8)
- Advanced cache warming strategies
- Performance dashboard polish
- Additional E2E test scenarios (edge cases)

**Hand-off to Sprint 9**:
- Security findings remediation (if any high-priority items remain)
- Performance optimization round 2 (based on Sprint 8 findings)
- Production deployment configuration
- Monitoring and alerting setup
- Final security audit
- Launch readiness checklist

---

## Agent Communication Channels

- **Blocker escalation**: Report in daily standup or Slack immediately
- **Code reviews**: GitHub PR comments + review request
- **Design decisions**: Document in ADR (Architecture Decision Record)
- **Sprint progress**: Update JIRA/Linear/GitHub Issues daily
- **Documentation**: Update relevant CLAUDE.md files

---

## Appendix: Useful Commands

### Run Unit Tests
```bash
go test -short -race -cover ./...
```

### Run Integration Tests
```bash
go test -tags=integration -race -cover ./tests/integration/...
```

### Run Security Tests
```bash
go test -race -cover ./tests/security/...
```

### Run E2E Tests
```bash
newman run tests/e2e/postman/goimg-api.postman_collection.json \
  --environment tests/e2e/postman/ci.postman_environment.json
```

### Check Coverage
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Security Scans
```bash
gosec ./...
trivy fs .
govulncheck ./...
gitleaks detect --verbose
```

### Load Testing
```bash
k6 run tests/load/image_upload.js
```

---

**Sprint 8 Setup Document Version**: 1.0
**Created**: 2025-12-04
**Author**: scrum-master agent
**Status**: READY FOR EXECUTION
