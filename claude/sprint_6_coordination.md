# Sprint 6 Coordination Plan

> **Sprint 6**: Application & HTTP - Gallery Context
> **Duration**: 2 weeks (2025-12-04 to 2025-12-17)
> **Status**: IN PROGRESS
> **Lead**: senior-go-architect

---

## Executive Summary

Sprint 6 delivers the core gallery functionality including image upload, albums, tags, search, and social features (likes/comments). This sprint has **HIGH** security risk due to IDOR vulnerabilities and authorization requirements. Critical focus areas: ownership validation, background job processing, and input sanitization.

**Success Criteria**:
- All 10 application layer commands/queries implemented with 85%+ coverage
- HTTP layer complete with ownership middleware
- Upload rate limiting functional (50/hour)
- Background job processing operational
- Search with full-text PostgreSQL working
- Security gate S6 PASS (IDOR prevention, ownership validation)

---

## Table of Contents

1. [Sprint Goals](#sprint-goals)
2. [Agent Assignments](#agent-assignments)
3. [Task Breakdown](#task-breakdown)
4. [Dependency Map](#dependency-map)
5. [Execution Timeline](#execution-timeline)
6. [Work Packages](#work-packages)
7. [Checkpoint Schedule](#checkpoint-schedule)
8. [Risk Register](#risk-register)
9. [Quality Gates](#quality-gates)
10. [Communication Plan](#communication-plan)

---

## Sprint Goals

### Primary Objective
Deliver end-to-end gallery functionality from upload through social interactions, with secure authorization and async processing.

### Key Results
1. **Upload Flow**: Users can upload images with async processing (ClamAV + variant generation)
2. **Organization**: Users can create albums and add images with tags
3. **Discovery**: Users can search images by title, description, and tags
4. **Social**: Users can like and comment on images
5. **Security**: All mutations validate ownership, no IDOR vulnerabilities
6. **Performance**: Background jobs process uploads within 30 seconds

### Non-Goals (Deferred)
- IPFS storage integration (Phase 2)
- Advanced search filters (Phase 2)
- Nested albums (Phase 2)
- Comment editing (Phase 2)

---

## Agent Assignments

### Lead Agent
**senior-go-architect**
- Overall sprint coordination
- Architecture decisions (asynq integration, pagination strategy)
- Code review for all application and HTTP layers
- Performance validation

### Critical Agents
**image-gallery-expert**
- Upload flow UX design review
- Background job pipeline design
- Feature completeness validation
- Competitive feature parity check (Flickr/Chevereto)

**backend-test-architect**
- Test strategy for async job processing
- Integration tests for repositories
- Coverage validation (85%+ application layer)
- Performance tests for search queries

**senior-secops-engineer**
- IDOR prevention verification
- Ownership validation at all endpoints
- Input sanitization review (comments, search)
- Security gate S6 approval

### Supporting Agents
**test-strategist**
- Postman/Newman E2E test suite
- Upload flow E2E tests (multipart form-data)
- Search and pagination E2E tests
- Social features E2E tests

**cicd-guardian** (on-demand)
- Background job infrastructure in CI
- Testcontainers for asynq (if needed)
- Pipeline optimization for image processing tests

---

## Task Breakdown

### Phase 1: Foundation (Days 1-3)
**Goal**: Database schema, background job infrastructure, middleware

#### Tasks
1. **Database Migration: Social Tables** (Priority: P0)
   - Owner: Direct implementation
   - Deliverable: `/home/user/goimg-datalayer/migrations/00004_create_social_tables.sql`
   - Acceptance Criteria:
     - [ ] Likes table with composite PK (user_id, image_id)
     - [ ] Comments table with UUID PK
     - [ ] Indexes on foreign keys
     - [ ] Migration up/down tested
   - Dependencies: None
   - Estimated: 2 hours

2. **Asynq Background Job Infrastructure** (Priority: P0)
   - Owner: senior-go-architect (design) → Direct implementation
   - Deliverable: `/home/user/goimg-datalayer/internal/infrastructure/jobs/`
   - Acceptance Criteria:
     - [ ] Asynq client configured with Redis
     - [ ] Job queue definitions (image:process, image:scan)
     - [ ] Worker setup in cmd/worker
     - [ ] Integration tests with testcontainers
   - Dependencies: None
   - Estimated: 6 hours

3. **Ownership Validation Middleware** (Priority: P0)
   - Owner: Direct implementation → senior-secops-engineer (review)
   - Deliverable: `/home/user/goimg-datalayer/internal/interfaces/http/middleware/ownership.go`
   - Acceptance Criteria:
     - [ ] Extracts resource ID from request (image_id, album_id)
     - [ ] Queries repository to verify ownership
     - [ ] Returns 403 Forbidden if not owner
     - [ ] Caches ownership checks (Redis)
     - [ ] Unit tests with mocked repositories
   - Dependencies: None
   - Estimated: 4 hours

4. **Upload Rate Limiting Middleware** (Priority: P0)
   - Owner: Direct implementation
   - Deliverable: Enhance existing rate_limit.go
   - Acceptance Criteria:
     - [ ] Separate rate limiter for uploads (50/hour per user)
     - [ ] Uses Redis for distributed limiting
     - [ ] Returns 429 with Retry-After header
     - [ ] Integration test validates enforcement
   - Dependencies: None
   - Estimated: 2 hours

### Phase 2: Application Layer - Image Management (Days 3-6)
**Goal**: Core image CRUD commands and queries

#### Tasks
5. **UploadImageCommand + Handler** (Priority: P0)
   - Owner: Direct implementation → image-gallery-expert (review)
   - Deliverable: `/home/user/goimg-datalayer/internal/application/commands/upload_image.go`
   - Acceptance Criteria:
     - [ ] Accepts multipart file, metadata (title, description, visibility, tags)
     - [ ] Validates file size, MIME type (reuse Sprint 5 validator)
     - [ ] Creates Image aggregate with status=processing
     - [ ] Stores original file via storage provider
     - [ ] Enqueues background job for processing (ClamAV + variants)
     - [ ] Returns 201 with image ID immediately (async processing)
     - [ ] Unit tests with mocked storage + job queue
     - [ ] Integration test end-to-end
   - Dependencies: Task 2 (asynq)
   - Estimated: 8 hours

6. **UpdateImageCommand + Handler** (Priority: P1)
   - Owner: Direct implementation
   - Deliverable: `/home/user/goimg-datalayer/internal/application/commands/update_image.go`
   - Acceptance Criteria:
     - [ ] Allows updating title, description, visibility, tags
     - [ ] Validates ownership (command level + handler)
     - [ ] Updates Image aggregate
     - [ ] Publishes ImageUpdated event
     - [ ] Unit tests with ownership validation
   - Dependencies: None
   - Estimated: 4 hours

7. **DeleteImageCommand + Handler** (Priority: P1)
   - Owner: Direct implementation
   - Deliverable: `/home/user/goimg-datalayer/internal/application/commands/delete_image.go`
   - Acceptance Criteria:
     - [ ] Validates ownership
     - [ ] Soft-deletes Image aggregate (deleted_at timestamp)
     - [ ] Enqueues background job to delete files from storage
     - [ ] Publishes ImageDeleted event
     - [ ] Unit tests with ownership validation
   - Dependencies: Task 2 (asynq)
   - Estimated: 4 hours

8. **GetImageQuery + Handler** (Priority: P0)
   - Owner: Direct implementation
   - Deliverable: `/home/user/goimg-datalayer/internal/application/queries/get_image.go`
   - Acceptance Criteria:
     - [ ] Validates visibility (public/unlisted/private)
     - [ ] Returns full image details with variants
     - [ ] Includes owner info (username, display_name)
     - [ ] Includes computed counts (likes, comments, views)
     - [ ] Unit tests with visibility enforcement
   - Dependencies: None
   - Estimated: 3 hours

9. **ListImagesQuery + Handler** (Priority: P0)
   - Owner: Direct implementation → senior-go-architect (pagination review)
   - Deliverable: `/home/user/goimg-datalayer/internal/application/queries/list_images.go`
   - Acceptance Criteria:
     - [ ] Supports filters: owner_id, album_id, visibility, tags
     - [ ] Supports sorting: created_at, view_count, like_count
     - [ ] Offset-based pagination (page, per_page)
     - [ ] Returns pagination metadata (total, total_pages)
     - [ ] Respects visibility rules (private only for owner)
     - [ ] Unit tests with various filter combinations
     - [ ] Performance test with 10k images
   - Dependencies: None
   - Estimated: 6 hours

### Phase 3: Application Layer - Albums & Search (Days 6-8)
**Goal**: Album management and search functionality

#### Tasks
10. **CreateAlbumCommand + Handler** (Priority: P1)
    - Owner: Direct implementation
    - Deliverable: `/home/user/goimg-datalayer/internal/application/commands/create_album.go`
    - Acceptance Criteria:
      - [ ] Accepts title, description, visibility
      - [ ] Creates Album aggregate
      - [ ] Sets owner_id from authenticated user
      - [ ] Publishes AlbumCreated event
      - [ ] Unit tests
    - Dependencies: None
    - Estimated: 3 hours

11. **AddImageToAlbumCommand + Handler** (Priority: P1)
    - Owner: Direct implementation
    - Deliverable: `/home/user/goimg-datalayer/internal/application/commands/add_image_to_album.go`
    - Acceptance Criteria:
      - [ ] Validates album ownership
      - [ ] Validates image ownership OR image is public
      - [ ] Creates album_images association
      - [ ] Supports bulk add (max 100 images per request)
      - [ ] Unit tests with ownership scenarios
    - Dependencies: None
    - Estimated: 4 hours

12. **SearchImagesQuery + Handler** (Priority: P1)
    - Owner: Direct implementation → backend-test-architect (SQL review)
    - Deliverable: `/home/user/goimg-datalayer/internal/application/queries/search_images.go`
    - Acceptance Criteria:
      - [ ] PostgreSQL full-text search on title + description
      - [ ] Tag-based filtering (AND logic for multiple tags)
      - [ ] Respects visibility (public search excludes private)
      - [ ] Pagination support
      - [ ] Parameterized queries (SQL injection prevention)
      - [ ] Unit tests with SQL injection attempts
      - [ ] Performance test with 10k images
    - Dependencies: None
    - Estimated: 6 hours

### Phase 4: Application Layer - Social Features (Days 8-10)
**Goal**: Likes and comments

#### Tasks
13. **LikeImageCommand + Handler** (Priority: P1)
    - Owner: Direct implementation
    - Deliverable: `/home/user/goimg-datalayer/internal/application/commands/like_image.go`
    - Acceptance Criteria:
      - [ ] Idempotent (liking twice = same as once)
      - [ ] Creates likes record (user_id, image_id)
      - [ ] Increments like_count atomically (UPDATE images SET like_count = like_count + 1)
      - [ ] Publishes ImageLiked event
      - [ ] Unit tests with idempotency check
    - Dependencies: Task 1 (migration)
    - Estimated: 3 hours

14. **UnlikeImageCommand + Handler** (Priority: P1)
    - Owner: Direct implementation
    - Deliverable: Same file as Task 13
    - Acceptance Criteria:
      - [ ] Idempotent (unliking when not liked = no-op)
      - [ ] Deletes likes record
      - [ ] Decrements like_count atomically
      - [ ] Publishes ImageUnliked event
      - [ ] Unit tests
    - Dependencies: Task 1 (migration)
    - Estimated: 2 hours

15. **AddCommentCommand + Handler** (Priority: P1)
    - Owner: Direct implementation → senior-secops-engineer (sanitization review)
    - Deliverable: `/home/user/goimg-datalayer/internal/application/commands/add_comment.go`
    - Acceptance Criteria:
      - [ ] Accepts image_id, content (1-1000 chars)
      - [ ] Sanitizes content (HTML/script tag removal)
      - [ ] Creates Comment aggregate
      - [ ] Publishes CommentAdded event
      - [ ] Rate limiting (10 comments/min per user)
      - [ ] Unit tests with XSS payload attempts
    - Dependencies: Task 1 (migration)
    - Estimated: 4 hours

16. **DeleteCommentCommand + Handler** (Priority: P1)
    - Owner: Direct implementation
    - Deliverable: `/home/user/goimg-datalayer/internal/application/commands/delete_comment.go`
    - Acceptance Criteria:
      - [ ] Validates ownership OR image owner OR moderator role
      - [ ] Soft-deletes Comment aggregate
      - [ ] Publishes CommentDeleted event
      - [ ] Unit tests with authorization scenarios
    - Dependencies: Task 1 (migration)
    - Estimated: 3 hours

### Phase 5: HTTP Layer (Days 10-12)
**Goal**: REST endpoints with authorization

#### Tasks
17. **Image Handlers** (Priority: P0)
    - Owner: Direct implementation → scrum-master (review)
    - Deliverable: `/home/user/goimg-datalayer/internal/interfaces/http/handlers/image_handler.go`
    - Acceptance Criteria:
      - [ ] POST /api/v1/images (upload, multipart/form-data)
      - [ ] GET /api/v1/images (list with filters)
      - [ ] GET /api/v1/images/{id} (get single)
      - [ ] PUT /api/v1/images/{id} (update, ownership middleware)
      - [ ] DELETE /api/v1/images/{id} (delete, ownership middleware)
      - [ ] All endpoints map domain errors to RFC 7807
      - [ ] DTOs for request/response
      - [ ] OpenAPI spec updated
    - Dependencies: Tasks 5-9
    - Estimated: 6 hours

18. **Album Handlers** (Priority: P1)
    - Owner: Direct implementation
    - Deliverable: `/home/user/goimg-datalayer/internal/interfaces/http/handlers/album_handler.go`
    - Acceptance Criteria:
      - [ ] POST /api/v1/albums (create)
      - [ ] GET /api/v1/albums (list user's albums)
      - [ ] GET /api/v1/albums/{id} (get with images)
      - [ ] PUT /api/v1/albums/{id} (update, ownership middleware)
      - [ ] DELETE /api/v1/albums/{id} (delete, ownership middleware)
      - [ ] POST /api/v1/albums/{id}/images (add images)
      - [ ] DELETE /api/v1/albums/{id}/images/{image_id} (remove image)
      - [ ] OpenAPI spec updated
    - Dependencies: Tasks 10-11
    - Estimated: 6 hours

19. **Search Handler** (Priority: P1)
    - Owner: Direct implementation
    - Deliverable: `/home/user/goimg-datalayer/internal/interfaces/http/handlers/search_handler.go`
    - Acceptance Criteria:
      - [ ] GET /api/v1/search/images?q={query}&tags={tags}
      - [ ] Delegates to SearchImagesQuery
      - [ ] Returns paginated results
      - [ ] OpenAPI spec updated
    - Dependencies: Task 12
    - Estimated: 2 hours

20. **Social Interaction Handlers** (Priority: P1)
    - Owner: Direct implementation
    - Deliverable: `/home/user/goimg-datalayer/internal/interfaces/http/handlers/social_handler.go`
    - Acceptance Criteria:
      - [ ] POST /api/v1/images/{id}/like (like)
      - [ ] DELETE /api/v1/images/{id}/like (unlike)
      - [ ] GET /api/v1/images/{id}/likes (list users who liked)
      - [ ] POST /api/v1/images/{id}/comments (add comment)
      - [ ] GET /api/v1/images/{id}/comments (list comments)
      - [ ] DELETE /api/v1/comments/{id} (delete comment)
      - [ ] OpenAPI spec updated
    - Dependencies: Tasks 13-16
    - Estimated: 5 hours

### Phase 6: Testing & Validation (Days 12-14)
**Goal**: Comprehensive test coverage and E2E validation

#### Tasks
21. **Unit Test Coverage Validation** (Priority: P0)
    - Owner: backend-test-architect
    - Deliverable: Coverage report >= 85% for application layer
    - Acceptance Criteria:
      - [ ] All commands have unit tests
      - [ ] All queries have unit tests
      - [ ] Ownership validation tested
      - [ ] Error cases covered
      - [ ] go test -race ./... passes
    - Dependencies: Tasks 5-20
    - Estimated: 6 hours

22. **Integration Tests for Background Jobs** (Priority: P0)
    - Owner: backend-test-architect
    - Deliverable: `/home/user/goimg-datalayer/tests/integration/jobs_test.go`
    - Acceptance Criteria:
      - [ ] Test image:process job (ClamAV + variants)
      - [ ] Test image:scan job (malware detection)
      - [ ] Test job retry on failure
      - [ ] Test job timeout handling
      - [ ] Uses testcontainers for Redis
    - Dependencies: Task 2, Task 5
    - Estimated: 6 hours

23. **E2E Tests with Newman/Postman** (Priority: P0)
    - Owner: test-strategist
    - Deliverable: Updated `/home/user/goimg-datalayer/tests/e2e/postman/goimg-api.postman_collection.json`
    - Acceptance Criteria:
      - [ ] Upload flow (POST /images with multipart form)
      - [ ] Image CRUD operations
      - [ ] Album CRUD operations
      - [ ] Search functionality
      - [ ] Like/unlike operations
      - [ ] Comment CRUD operations
      - [ ] Authorization tests (403 on others' private images)
      - [ ] Error scenarios (400, 413, 415, 422, 429)
      - [ ] make test-e2e passes in CI
    - Dependencies: Tasks 17-20
    - Estimated: 8 hours

24. **Security Test Suite** (Priority: P0)
    - Owner: senior-secops-engineer
    - Deliverable: `/home/user/goimg-datalayer/tests/security/gallery_authz_test.go`
    - Acceptance Criteria:
      - [ ] IDOR tests for images (user A cannot access user B's private images)
      - [ ] IDOR tests for albums
      - [ ] Ownership bypass attempts
      - [ ] SQL injection in search
      - [ ] XSS in comments
      - [ ] Rate limit enforcement tests
      - [ ] All S6 security gates pass
    - Dependencies: Tasks 17-20
    - Estimated: 6 hours

25. **Performance Testing** (Priority: P1)
    - Owner: backend-test-architect
    - Deliverable: Performance benchmarks
    - Acceptance Criteria:
      - [ ] Search query < 200ms P95 (10k images)
      - [ ] List images < 100ms P95 (pagination)
      - [ ] Background job processing < 30s (10MB image)
      - [ ] Upload endpoint < 500ms (excluding processing)
    - Dependencies: Tasks 5, 9, 12
    - Estimated: 4 hours

---

## Dependency Map

```
┌────────────────────────────────────────────────────────────────┐
│                       PHASE 1: FOUNDATION                       │
│  [1] Social Tables Migration                                    │
│  [2] Asynq Background Jobs   ──┐                                │
│  [3] Ownership Middleware       │                                │
│  [4] Upload Rate Limiting       │                                │
└────────────────────────────────┼────────────────────────────────┘
                                 │
    ┌────────────────────────────┼────────────────────────────────┐
    │          PHASE 2: IMAGE MANAGEMENT                           │
    │  [5] UploadImageCommand ◄──┘                                 │
    │  [6] UpdateImageCommand                                      │
    │  [7] DeleteImageCommand ◄──┐                                 │
    │  [8] GetImageQuery          │                                │
    │  [9] ListImagesQuery        │                                │
    └────────────────────────────┼────────────────────────────────┘
                                 │
    ┌────────────────────────────┼────────────────────────────────┐
    │         PHASE 3: ALBUMS & SEARCH                             │
    │  [10] CreateAlbumCommand    │                                │
    │  [11] AddImageToAlbumCommand│                                │
    │  [12] SearchImagesQuery     │                                │
    └────────────────────────────┼────────────────────────────────┘
                                 │
    ┌────────────────────────────┼────────────────────────────────┐
    │           PHASE 4: SOCIAL FEATURES                           │
    │  [13] LikeImageCommand      │                                │
    │  [14] UnlikeImageCommand    │                                │
    │  [15] AddCommentCommand     │                                │
    │  [16] DeleteCommentCommand  │                                │
    └────────────────────────────┼────────────────────────────────┘
                                 │
    ┌────────────────────────────┼────────────────────────────────┐
    │              PHASE 5: HTTP LAYER                             │
    │  [17] Image Handlers ◄──────┘                                │
    │  [18] Album Handlers                                         │
    │  [19] Search Handler                                         │
    │  [20] Social Handlers                                        │
    └────────────────────────────┼────────────────────────────────┘
                                 │
    ┌────────────────────────────▼────────────────────────────────┐
    │            PHASE 6: TESTING & VALIDATION                     │
    │  [21] Unit Test Coverage                                     │
    │  [22] Integration Tests (Jobs)                               │
    │  [23] E2E Tests (Newman)                                     │
    │  [24] Security Tests                                         │
    │  [25] Performance Tests                                      │
    └─────────────────────────────────────────────────────────────┘
```

**Critical Path**: Tasks 1, 2, 5, 17, 23 (Migration → Jobs → Upload → Handlers → E2E)

---

## Execution Timeline

### Week 1 (Days 1-7)

| Day | Phase | Tasks | Owner | Deliverable |
|-----|-------|-------|-------|-------------|
| **Mon 12/4** | Foundation | 1, 2 | Direct + senior-go-architect | Migration + asynq setup |
| **Tue 12/5** | Foundation | 3, 4 | Direct | Middleware complete |
| **Wed 12/6** | Image Mgmt | 5, 6 | Direct + image-gallery-expert | Upload + Update commands |
| **Thu 12/7** | Image Mgmt | 7, 8, 9 | Direct + senior-go-architect | Delete + Query commands |
| **Fri 12/8** | Albums | 10, 11 | Direct | Album commands |
| **Sat-Sun** | - | - | - | Buffer/catch-up |

### Week 2 (Days 8-14)

| Day | Phase | Tasks | Owner | Deliverable |
|-----|-------|-------|-------|-------------|
| **Mon 12/9** | Search + Social | 12, 13, 14 | Direct + backend-test-architect | Search + Like commands |
| **Tue 12/10** | Social + HTTP | 15, 16, 17 | Direct + senior-secops-engineer | Comment + Image handlers |
| **Wed 12/11** | HTTP | 18, 19, 20 | Direct | Album + Social handlers |
| **Thu 12/12** | Testing | 21, 22 | backend-test-architect | Unit + Integration tests |
| **Fri 12/13** | Testing | 23, 24 | test-strategist + senior-secops-engineer | E2E + Security tests |
| **Mon 12/16** | Validation | 25 + Final Review | All agents | Performance + Gate approval |
| **Tue 12/17** | Retrospective | Sprint completion | scrum-master | Sprint 6 marked COMPLETE |

---

## Work Packages

### Work Package 1: senior-go-architect

**Role**: Lead, Architecture, Code Review
**Commitment**: 20 hours over 2 weeks

#### Deliverables
1. **Asynq Integration Design** (Day 1)
   - Research asynq patterns (task prioritization, retry policies)
   - Design job queue architecture (queues, workers, monitoring)
   - Document in `/home/user/goimg-datalayer/docs/architecture/background-jobs.md`
   - Review implementation in Task 2

2. **Pagination Strategy Review** (Day 4)
   - Review ListImagesQuery pagination implementation
   - Validate SQL query performance (EXPLAIN ANALYZE)
   - Recommend cursor-based pagination if offset shows issues at scale
   - Approve Task 9

3. **Code Review - Application Layer** (Days 5-10)
   - Review all commands (Tasks 5-7, 10-11, 13-16)
   - Review all queries (Tasks 8-9, 12)
   - Validate DDD patterns (no business logic in handlers)
   - Approve or request changes

4. **Code Review - HTTP Layer** (Days 11-12)
   - Review all handlers (Tasks 17-20)
   - Validate error mapping to RFC 7807
   - Ensure DTOs properly isolate domain from HTTP
   - Approve or request changes

5. **Performance Validation** (Day 13)
   - Review Task 25 performance benchmarks
   - Identify bottlenecks if benchmarks fail
   - Recommend optimizations (indexes, caching)

#### Checkpoints
- **Pre-Sprint** (Day 1): Review asynq integration approach
- **Mid-Sprint** (Day 7): Review ownership middleware implementation
- **Pre-Merge** (Day 14): Code review approval

---

### Work Package 2: image-gallery-expert

**Role**: Feature Validation, UX Review
**Commitment**: 10 hours over 2 weeks

#### Deliverables
1. **Upload Flow UX Review** (Day 1)
   - Review upload endpoint design (multipart, immediate response)
   - Validate async processing UX (status polling endpoint?)
   - Compare with Flickr/Chevereto upload flows
   - Document recommendations

2. **Background Job Pipeline Design** (Day 2)
   - Review job queue architecture from senior-go-architect
   - Validate job priorities (scan before process? parallel?)
   - Recommend retry policies for transient failures
   - Document in architecture doc

3. **Feature Completeness Review** (Day 12)
   - Validate all MVP features from `/home/user/goimg-datalayer/claude/mvp_features.md`
   - Check parity with Flickr/Chevereto baseline
   - Identify missing features (document as Phase 2)
   - Sign off on feature completeness

4. **E2E User Flow Validation** (Day 13)
   - Test upload → album creation → tag → search → like → comment flow
   - Validate visibility settings work correctly
   - Identify UX issues or gaps
   - Document in Sprint 6 review

#### Checkpoints
- **Pre-Sprint** (Day 1): Upload flow UX and background job design review
- **Mid-Sprint** (Day 7): Verify search functionality and pagination
- **Pre-Merge** (Day 12): Feature completeness verified

---

### Work Package 3: backend-test-architect

**Role**: Testing Strategy, Coverage Validation
**Commitment**: 24 hours over 2 weeks

#### Deliverables
1. **Test Strategy for Async Processing** (Day 1)
   - Design integration test approach for background jobs
   - Evaluate testcontainers for asynq (Redis + worker)
   - Document in `/home/user/goimg-datalayer/claude/test_strategy.md`
   - Review with senior-go-architect

2. **Unit Test Implementation** (Days 5-10)
   - Write unit tests for commands as they're implemented
   - Focus on error cases, ownership validation
   - Achieve 85%+ coverage for application layer
   - Deliverable: Task 21 completion

3. **Integration Tests for Background Jobs** (Days 8-9)
   - Implement Task 22 (jobs integration tests)
   - Test image:process job end-to-end
   - Test job retry and failure scenarios
   - Validate with testcontainers

4. **SQL Review for Search** (Day 6)
   - Review Task 12 (SearchImagesQuery) SQL
   - Run EXPLAIN ANALYZE for query plans
   - Validate parameterized queries (SQL injection prevention)
   - Recommend indexes if needed

5. **Performance Testing** (Days 12-13)
   - Implement Task 25 (performance benchmarks)
   - Load test with 10k images in database
   - Identify bottlenecks (missing indexes, N+1 queries)
   - Document results

6. **Coverage Validation** (Day 14)
   - Run coverage report: `go test -coverprofile=coverage.out ./...`
   - Validate overall 80%, application 85%, domain 90%
   - Identify gaps and request additional tests
   - Sign off on coverage gate

#### Checkpoints
- **Pre-Sprint** (Day 1): Test strategy for async processing
- **Mid-Sprint** (Day 7): Application layer coverage >= 85%
- **Pre-Merge** (Day 14): Coverage thresholds met, race detector clean

---

### Work Package 4: senior-secops-engineer

**Role**: Security Validation, IDOR Prevention
**Commitment**: 16 hours over 2 weeks

#### Deliverables
1. **Ownership Middleware Review** (Day 3)
   - Review Task 3 implementation
   - Validate authorization logic (owner check)
   - Check for race conditions (TOCTOU)
   - Recommend caching strategy (Redis)

2. **Input Sanitization Review** (Day 10)
   - Review Task 15 (AddCommentCommand) sanitization
   - Validate HTML/script tag removal
   - Test with OWASP XSS payloads
   - Approve or request changes

3. **Search Query SQL Injection Review** (Day 6)
   - Review Task 12 (SearchImagesQuery) SQL
   - Validate parameterized queries only
   - Test with SQL injection payloads (' OR '1'='1, '; DROP TABLE--)
   - Approve or request changes

4. **Security Test Suite Implementation** (Days 12-13)
   - Implement Task 24 (security tests)
   - IDOR tests (user A accessing user B's resources)
   - Authorization bypass attempts
   - SQL injection tests
   - XSS tests in comments
   - Rate limit enforcement tests

5. **Security Gate S6 Review** (Day 14)
   - Execute all S6 mandatory controls (see Security Gates section)
   - Validate ownership verification on all mutations
   - Verify IDOR prevention mechanisms
   - Review input sanitization on comments
   - Review rate limiting on uploads
   - Sign off: PASS / CONDITIONAL PASS / FAIL

#### Checkpoints
- **Pre-Sprint** (Day 1): Review ownership middleware design
- **Mid-Sprint** (Day 7): Review ownership/permission middleware implementation
- **Pre-Merge** (Day 14): IDOR prevention verified, ownership checks validated

---

### Work Package 5: test-strategist

**Role**: E2E Testing, API Contract Testing
**Commitment**: 12 hours over 2 weeks

#### Deliverables
1. **E2E Test Planning** (Day 1)
   - Identify all API endpoints for Sprint 6
   - Design test scenarios (happy path, error cases)
   - Plan multipart form-data upload tests
   - Document in test plan

2. **Postman Collection Update** (Days 11-13)
   - Implement Task 23 (E2E tests with Newman)
   - Add requests for all new endpoints:
     - POST /api/v1/images (multipart)
     - GET /api/v1/images (with filters)
     - PUT/DELETE /api/v1/images/{id}
     - POST /api/v1/albums
     - POST /api/v1/albums/{id}/images
     - GET /api/v1/search/images
     - POST /api/v1/images/{id}/like
     - POST /api/v1/images/{id}/comments
   - Add test scripts for:
     - Status code validation
     - Response body schema validation
     - Authorization tests (403 on private images)
     - Error format validation (RFC 7807)
   - Validate `make test-e2e` passes locally and in CI

3. **E2E Flow Testing** (Day 13)
   - Test complete user flow:
     1. Register/Login
     2. Upload image (multipart)
     3. Create album
     4. Add image to album
     5. Search for image
     6. Like image
     7. Comment on image
   - Validate all responses follow OpenAPI spec
   - Document any discrepancies

4. **OpenAPI Compliance Validation** (Day 14)
   - Validate all new endpoints match OpenAPI spec
   - Run `make validate-openapi`
   - Report any spec drift
   - Ensure CI passes

#### Checkpoints
- **Mid-Sprint** (Day 7): E2E test suite planning complete
- **Pre-Merge** (Day 14): Newman/Postman E2E tests passing (30+ test requests)

---

### Work Package 6: scrum-master (You)

**Role**: Coordination, Quality Validation, Sprint Management
**Commitment**: Continuous

#### Deliverables
1. **Sprint Kickoff** (Day 1)
   - Facilitate pre-sprint checkpoint
   - Confirm agent commitments
   - Clarify task assignments
   - Document in sprint notes

2. **Daily Progress Tracking** (Days 1-14)
   - Monitor task completion status
   - Identify blockers early
   - Coordinate agent handoffs
   - Update sprint burndown

3. **Mid-Sprint Checkpoint** (Day 7)
   - Facilitate checkpoint meeting
   - Review progress (should be 50% complete)
   - Adjust assignments if needed
   - Escalate risks

4. **Work Quality Verification** (Days 5-14)
   - Review completed tasks against acceptance criteria
   - Validate agent checklist items completed
   - Ensure tests pass before marking tasks complete
   - Request remediation if quality gaps exist

5. **Pre-Merge Quality Gate** (Day 14)
   - Execute all quality gates (see section below)
   - Coordinate final agent approvals
   - Validate security gate S6 PASS
   - Approve merge or block with issues

6. **Sprint Retrospective** (Day 15)
   - Facilitate retrospective (Start/Stop/Continue)
   - Document lessons learned
   - Create improvement backlog
   - Update sprint_plan.md with Sprint 6 COMPLETE

---

## Checkpoint Schedule

### Pre-Sprint Checkpoint (Day 1, 9:00 AM)

**Attendees**: scrum-master, senior-go-architect, image-gallery-expert, backend-test-architect

**Duration**: 90 minutes

**Agenda**:
1. **Sprint Goal Review** (15 min)
   - Confirm sprint goals and success criteria
   - Review MVP feature alignment

2. **Technical Approach** (30 min)
   - senior-go-architect presents asynq integration design
   - image-gallery-expert reviews upload flow UX
   - Discussion and alignment

3. **Risk Review** (20 min)
   - Review risk register (see section below)
   - Assign mitigation owners
   - Identify early warning signals

4. **Agent Commitments** (15 min)
   - Confirm each agent's work package
   - Validate time availability (PTO, conflicts)
   - Adjust if needed

5. **Dependencies** (10 min)
   - Review dependency map
   - Identify critical path
   - Plan parallel work where possible

**Output**: Sprint kickoff summary with agent commitments

---

### Mid-Sprint Checkpoint (Day 7, 2:00 PM)

**Attendees**: scrum-master, senior-go-architect, backend-test-architect, senior-secops-engineer

**Duration**: 45 minutes

**Agenda**:
1. **Burndown Review** (10 min)
   - Review completed tasks (target: 50%)
   - Identify slippage
   - Adjust timeline if needed

2. **Coverage Trajectory** (10 min)
   - backend-test-architect reports coverage status
   - Target: 70%+ at mid-sprint
   - Identify gaps

3. **Blocker Resolution** (15 min)
   - Review active blockers
   - Assign resolution owners
   - Set resolution deadlines

4. **Quality Check** (10 min)
   - Review any failing tests
   - Review linting issues
   - Plan remediation

**Output**: Burndown status, risk report, corrective actions

---

### Pre-Merge Quality Gate (Day 14, 10:00 AM)

**Attendees**: All agents with Pre-Merge checklist items

**Duration**: 90 minutes

**Agenda**:
1. **Automated Quality Gates** (20 min)
   - Run all automated checks (see Quality Gates section)
   - Address any failures
   - Confirm green build

2. **Agent Approvals** (40 min)
   - senior-go-architect: Code review approval
   - backend-test-architect: Coverage thresholds met (85%+ application, 80%+ overall)
   - senior-secops-engineer: Security gate S6 PASS
   - test-strategist: E2E tests passing
   - image-gallery-expert: Feature completeness verified

3. **OpenAPI Spec Alignment** (10 min)
   - Validate `make validate-openapi` passes
   - Review any spec updates
   - Confirm no breaking changes

4. **Agent Checklist Compliance** (10 min)
   - Review `/home/user/goimg-datalayer/claude/agent_checklist.md`
   - Confirm all items checked
   - Document any exceptions

5. **Merge Decision** (10 min)
   - scrum-master: Approve merge or block with issues
   - If blocked: Create remediation plan with owners and deadlines
   - If approved: Merge to main and mark Sprint 6 COMPLETE

**Output**: Merge approval or list of blockers

---

### Sprint Retrospective (Day 15, 2:00 PM)

**Attendees**: scrum-master (facilitator), all active agents

**Duration**: 60 minutes

**Format**: Start/Stop/Continue

**Agenda**:
1. **Metrics Review** (10 min)
   - Velocity: planned vs actual
   - Coverage: achieved vs target
   - Defects: count and severity
   - Test success rate

2. **Start (New Practices)** (15 min)
   - What should we start doing?
   - Examples: Earlier security reviews, more pairing

3. **Stop (Eliminate Practices)** (15 min)
   - What should we stop doing?
   - Examples: Late testing, unclear task descriptions

4. **Continue (Effective Practices)** (10 min)
   - What worked well?
   - Examples: Clear work packages, daily check-ins

5. **Action Items** (10 min)
   - Assign owners to improvement items
   - Set due dates (typically Sprint 7)
   - Track in improvement backlog

**Output**: Retrospective notes with improvement actions

---

## Risk Register

### Risk 1: Background Job Processing Complexity
**Category**: Technical
**Severity**: High
**Probability**: Medium

**Description**: Asynq integration may have unexpected complexity (queue management, job retries, monitoring, worker scaling).

**Impact**:
- Sprint 6 upload flow delayed
- Tasks 5, 7, 22 blocked
- 3-4 day delay potential

**Mitigation**:
- **Owner**: senior-go-architect
- **Actions**:
  1. Research asynq best practices on Day 1 (2 hours)
  2. Create spike/prototype before Task 2 implementation
  3. Use testcontainers for integration tests
  4. Have fallback: synchronous processing if asynq fails

**Early Warning Signals**:
- Task 2 taking > 8 hours
- Integration tests failing on Day 3
- Worker not processing jobs

**Escalation**: If blocked > 1 day, escalate to scrum-master for scope adjustment

---

### Risk 2: IDOR Vulnerabilities in Gallery Endpoints
**Category**: Security
**Severity**: Critical
**Probability**: Medium

**Description**: Authorization checks may be missed on some endpoints, allowing users to access/modify others' resources.

**Impact**:
- Security gate S6 FAIL
- Sprint 6 blocked until remediation
- Potential 2-3 day delay

**Mitigation**:
- **Owner**: senior-secops-engineer
- **Actions**:
  1. Review Task 3 (ownership middleware) on Day 3
  2. Validate authorization in Task 24 (security tests)
  3. Double-check authorization at both handler and application layer
  4. IDOR testing on all endpoints before merge

**Early Warning Signals**:
- Ownership middleware tests failing
- Security tests revealing authorization bypasses
- Code review identifies missing checks

**Escalation**: Security gate FAIL escalates immediately to senior-secops-engineer and scrum-master

---

### Risk 3: Search Query SQL Injection
**Category**: Security
**Severity**: Critical
**Probability**: Low

**Description**: Search functionality may use string concatenation instead of parameterized queries, allowing SQL injection.

**Impact**:
- Security gate S6 FAIL
- Critical vulnerability in production
- Sprint blocked until fix

**Mitigation**:
- **Owner**: backend-test-architect + senior-secops-engineer
- **Actions**:
  1. Code review Task 12 SQL queries on Day 6
  2. Validate parameterized queries only (no fmt.Sprintf)
  3. SQL injection tests in Task 24
  4. Use sqlx named parameters for readability

**Early Warning Signals**:
- Code review identifies string concatenation
- Security tests detect SQL injection vulnerability

**Escalation**: If found, immediately fix before any other work continues

---

### Risk 4: Rate Limiting Ineffective Under Load
**Category**: Performance
**Severity**: Medium
**Probability**: Medium

**Description**: Upload rate limiting (50/hour) may not work correctly under concurrent requests or distributed instances.

**Impact**:
- Upload spam not prevented
- DoS vulnerability
- May require refactoring rate limiter

**Mitigation**:
- **Owner**: backend-test-architect
- **Actions**:
  1. Test rate limiting with concurrent requests (Task 25)
  2. Validate Redis-based limiting works across instances
  3. Use sliding window algorithm (not fixed window)
  4. Test with > 50 rapid requests

**Early Warning Signals**:
- Rate limit tests failing
- Race conditions in rate limiter
- Redis connection issues

**Escalation**: If ineffective, may defer to Sprint 7 with documented risk acceptance

---

### Risk 5: Ownership Middleware Performance
**Category**: Performance
**Severity**: Medium
**Probability**: Low

**Description**: Ownership middleware may add latency if it queries database on every request without caching.

**Impact**:
- P95 latency > 200ms on protected endpoints
- User experience degradation
- May require caching refactor

**Mitigation**:
- **Owner**: senior-go-architect
- **Actions**:
  1. Design with Redis caching from start (Task 3)
  2. Cache ownership checks for 5 minutes
  3. Performance test protected endpoints (Task 25)
  4. Validate P95 < 200ms

**Early Warning Signals**:
- Performance tests show high latency
- Database query count high on list endpoints
- N+1 query pattern detected

**Escalation**: If performance fails, implement caching before merge

---

### Risk 6: Pagination Performance at Scale
**Category**: Performance
**Severity**: Medium
**Probability**: Medium

**Description**: Offset-based pagination may have poor performance with large datasets (> 10k images).

**Impact**:
- Slow list/search endpoints
- May need to refactor to cursor-based pagination

**Mitigation**:
- **Owner**: senior-go-architect + backend-test-architect
- **Actions**:
  1. Test ListImagesQuery with 10k images (Task 25)
  2. Run EXPLAIN ANALYZE on pagination queries
  3. Add indexes on sorting columns (created_at, view_count, like_count)
  4. Consider cursor-based pagination if offset is slow

**Early Warning Signals**:
- Pagination queries > 200ms with 10k rows
- Full table scan in EXPLAIN output
- Linear time complexity with offset

**Escalation**: If slow, defer cursor-based pagination to Sprint 7 with documentation

---

### Risk 7: Test Coverage Below Target
**Category**: Quality
**Severity**: Medium
**Probability**: Low

**Description**: Application layer coverage may fall below 85% target due to time constraints.

**Impact**:
- Quality gate FAIL
- May miss critical bugs
- Sprint 6 blocked until tests written

**Mitigation**:
- **Owner**: backend-test-architect
- **Actions**:
  1. Write tests concurrently with implementation (not after)
  2. Check coverage daily starting Day 5
  3. Identify coverage gaps mid-sprint (Day 7)
  4. Prioritize tests for critical paths (upload, ownership)

**Early Warning Signals**:
- Coverage < 70% at mid-sprint
- Complex commands have no tests
- Error paths not tested

**Escalation**: If coverage low on Day 12, extend sprint 1-2 days for test writing

---

### Risk 8: Newman E2E Tests Flaky in CI
**Category**: Testing
**Severity**: Low
**Probability**: Medium

**Description**: E2E tests may be flaky due to timing issues, background job delays, or test data conflicts.

**Impact**:
- CI failures blocking merge
- Delays in identifying real issues
- Developer frustration

**Mitigation**:
- **Owner**: test-strategist + cicd-guardian
- **Actions**:
  1. Use unique test data per run (UUIDs in test fixtures)
  2. Add wait conditions for async operations (poll job status)
  3. Clean up test data between runs
  4. Retry flaky tests (max 2 retries) in CI

**Early Warning Signals**:
- E2E tests pass locally but fail in CI
- Intermittent failures (not consistent)
- Race conditions in test execution

**Escalation**: If flakiness > 10%, investigate root cause before merge

---

## Quality Gates

### Automated Quality Gates

Run these checks before Pre-Merge checkpoint:

```bash
# 1. Code Quality
go fmt ./...                          # Format code
go vet ./...                          # Static analysis
golangci-lint run                     # Linting (zero errors)

# 2. Testing
go test -race ./...                   # Unit tests with race detector
go test -coverprofile=coverage.out ./... # Coverage report
go tool cover -func=coverage.out      # Validate thresholds

# 3. API Contract
make validate-openapi                 # OpenAPI spec validation
make generate                         # Ensure codegen produces no diff

# 4. E2E Testing
make test-e2e                         # Newman/Postman tests

# 5. Security Scanning
gosec -severity high ./...            # Static security scan
trivy fs --severity HIGH,CRITICAL .   # Vulnerability scan
```

**Pass Criteria**: All commands exit with code 0

---

### Manual Quality Gates

#### 1. Code Review Checklist (senior-go-architect)

- [ ] No business logic in handlers (delegates to application layer)
- [ ] Domain layer has no infrastructure imports
- [ ] Errors wrapped with context (`fmt.Errorf("context: %w", err)`)
- [ ] Command/Query handlers follow CQRS patterns
- [ ] DTOs properly isolate HTTP from domain
- [ ] Repository methods use parameterized queries
- [ ] No commented-out code or unresolved TODOs

#### 2. Test Coverage Checklist (backend-test-architect)

- [ ] Overall coverage >= 80%
- [ ] Application layer coverage >= 85%
- [ ] Domain layer coverage >= 90%
- [ ] All commands have unit tests
- [ ] All queries have unit tests
- [ ] Error cases tested
- [ ] Race detector clean (`go test -race`)

#### 3. Security Gate S6 (senior-secops-engineer)

Execute all mandatory controls from `/home/user/goimg-datalayer/claude/security_gates.md`:

**Authorization Controls**:
- [ ] **S6-AUTHZ-001**: Ownership verified on image read (private images)
- [ ] **S6-AUTHZ-002**: Ownership verified on image update
- [ ] **S6-AUTHZ-003**: Ownership verified on image delete
- [ ] **S6-AUTHZ-004**: Album ownership verified

**IDOR Prevention**:
- [ ] **S6-IDOR-001**: Image ID authorization prevents IDOR (test: iterate IDs, verify 403/404)
- [ ] **S6-IDOR-002**: Album ID authorization prevents IDOR

**Input Validation**:
- [ ] **S6-VAL-004**: Comment content sanitized (HTML/script tags stripped)
- [ ] **S6-VAL-005**: Search query sanitized for SQL injection (parameterized queries)

**Search & Visibility**:
- [ ] **S6-SEARCH-001**: Search results respect visibility (private images excluded)

**Rate Limiting**:
- [ ] **S6-RATE-003**: Comment spam prevention (10 comments/min per user)

**Security Tests**:
- [ ] All tests in `/home/user/goimg-datalayer/tests/security/gallery_authz_test.go` pass
- [ ] IDOR tests pass (user A cannot access user B's private resources)
- [ ] SQL injection tests pass (search query)
- [ ] XSS tests pass (comment sanitization)

**Sign-Off**: ☐ PASS / ☐ CONDITIONAL PASS / ☐ FAIL

---

#### 4. E2E Test Checklist (test-strategist)

- [ ] Postman collection updated with all Sprint 6 endpoints
- [ ] Upload flow tested (multipart form-data)
- [ ] Authorization tests pass (403 on private images)
- [ ] Error scenarios covered (400, 413, 415, 422, 429)
- [ ] RFC 7807 error format validated
- [ ] `make test-e2e` passes locally
- [ ] Newman tests pass in CI

#### 5. Feature Completeness Checklist (image-gallery-expert)

- [ ] Upload flow functional (async processing)
- [ ] Albums functional (create, add images)
- [ ] Tags functional (add to images, search by tag)
- [ ] Search functional (title, description, tags)
- [ ] Likes functional (like, unlike, count)
- [ ] Comments functional (add, delete, list)
- [ ] Visibility settings work (public, private, unlisted)
- [ ] Rate limiting functional (50 uploads/hour)

#### 6. Performance Checklist (backend-test-architect)

- [ ] Upload endpoint < 500ms P95 (excluding processing)
- [ ] List images < 100ms P95 (10k images)
- [ ] Search query < 200ms P95 (10k images)
- [ ] Background job processing < 30s (10MB image)
- [ ] No N+1 query patterns detected

---

### Quality Gate Decision Matrix

| Gate | Status | Action |
|------|--------|--------|
| All automated gates PASS | ✅ | Proceed to manual gates |
| Any automated gate FAIL | ❌ | Fix before proceeding |
| Code review APPROVED | ✅ | Proceed |
| Code review CHANGES REQUESTED | 🔶 | Address feedback, re-review |
| Test coverage >= thresholds | ✅ | Proceed |
| Test coverage < thresholds | ❌ | Write additional tests |
| Security gate S6 PASS | ✅ | Proceed |
| Security gate S6 CONDITIONAL PASS | 🔶 | Document risk acceptance |
| Security gate S6 FAIL | ❌ | Block merge, remediate |
| E2E tests PASS | ✅ | Proceed |
| E2E tests FAIL | ❌ | Fix tests or code |
| Feature completeness verified | ✅ | Proceed |
| Feature gaps identified | 🔶 | Document for Phase 2 |
| Performance benchmarks met | ✅ | Proceed to merge |
| Performance below target | 🔶 | Optimize or accept with risk |

**Merge Approval**: All gates must be ✅ or 🔶 (with documented risk acceptance)

---

## Communication Plan

### Daily Stand-up (Async)

**Format**: Written update in project channel

**Template**:
```markdown
### [Agent Name] - [Date]

**Yesterday**:
- Completed: Task X (description)
- Progress: Task Y (X% complete)

**Today**:
- Plan: Task Z (description)

**Blockers**:
- [Blocker description] [requires: agent/resource]
```

**Timing**: Post by 10:00 AM daily

---

### Sprint Summary (Weekly)

**Owner**: scrum-master

**Format**: Written report

**Template**: See Communication Protocol section in agent_workflow.md

**Distribution**: All agents

**Timing**: End of Week 1 (Day 7), End of Week 2 (Day 14)

---

### Blocker Escalation Protocol

**Level 1 - Agent Self-Resolution** (0-4 hours):
- Agent attempts independent resolution
- Consult documentation, research solutions

**Level 2 - Peer Agent Assistance** (4-24 hours):
- Scrum master assigns supporting agent
- Collaborative problem-solving session

**Level 3 - Scrum Master Escalation** (24-48 hours):
- Scrum master re-prioritizes work
- Task reassignment or scope adjustment

**Level 4 - Stakeholder Escalation** (48+ hours):
- Critical path impact
- Requires external resources or decisions

---

### Agent Contact Matrix

| Need | Primary Agent | Backup Agent |
|------|---------------|--------------|
| Architecture decisions | senior-go-architect | scrum-master |
| Test strategy | backend-test-architect | test-strategist |
| Security review | senior-secops-engineer | scrum-master |
| E2E testing | test-strategist | backend-test-architect |
| Feature validation | image-gallery-expert | scrum-master |
| Blocker resolution | scrum-master | senior-go-architect |

---

## Appendix A: Task Checklist

Use this checklist to track task completion:

```markdown
### Phase 1: Foundation
- [ ] Task 1: Social Tables Migration
- [ ] Task 2: Asynq Background Jobs
- [ ] Task 3: Ownership Middleware
- [ ] Task 4: Upload Rate Limiting

### Phase 2: Image Management
- [ ] Task 5: UploadImageCommand
- [ ] Task 6: UpdateImageCommand
- [ ] Task 7: DeleteImageCommand
- [ ] Task 8: GetImageQuery
- [ ] Task 9: ListImagesQuery

### Phase 3: Albums & Search
- [ ] Task 10: CreateAlbumCommand
- [ ] Task 11: AddImageToAlbumCommand
- [ ] Task 12: SearchImagesQuery

### Phase 4: Social Features
- [ ] Task 13: LikeImageCommand
- [ ] Task 14: UnlikeImageCommand
- [ ] Task 15: AddCommentCommand
- [ ] Task 16: DeleteCommentCommand

### Phase 5: HTTP Layer
- [ ] Task 17: Image Handlers
- [ ] Task 18: Album Handlers
- [ ] Task 19: Search Handler
- [ ] Task 20: Social Handlers

### Phase 6: Testing & Validation
- [ ] Task 21: Unit Test Coverage
- [ ] Task 22: Integration Tests (Jobs)
- [ ] Task 23: E2E Tests (Newman)
- [ ] Task 24: Security Tests
- [ ] Task 25: Performance Tests
```

---

## Appendix B: Agent Checkpoint Template

Use this template for agent checkpoint reviews:

```markdown
## [Agent Name] Checkpoint - Sprint 6

**Date**: YYYY-MM-DD
**Phase**: Pre-Sprint / Mid-Sprint / Pre-Merge

### Review Items
- [ ] Item 1
- [ ] Item 2

### Findings
| ID | Severity | Description | Recommendation |
|----|----------|-------------|----------------|
| F1 | High | [Description] | [Action] |

### Status
☐ APPROVED
☐ APPROVED WITH RECOMMENDATIONS
☐ CHANGES REQUIRED

**Signature**: ___________________
**Date**: ___________________
```

---

## Version History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | 2025-12-04 | Initial Sprint 6 coordination plan | scrum-master |

---

## Next Steps

1. **Today (2025-12-04)**:
   - Facilitate pre-sprint checkpoint (9:00 AM)
   - Confirm agent commitments
   - Begin Task 1 (migration) and Task 2 (asynq setup)

2. **Week 1 Focus**:
   - Complete Phase 1-3 (Foundation, Image Management, Albums)
   - Mid-sprint checkpoint Day 7

3. **Week 2 Focus**:
   - Complete Phase 4-6 (Social, HTTP, Testing)
   - Pre-merge quality gate Day 14
   - Sprint retrospective Day 15

**Let's build an amazing gallery feature! 🚀**
