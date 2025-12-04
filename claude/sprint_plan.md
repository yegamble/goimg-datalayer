# Backend Sprint Plan

> Comprehensive development roadmap for goimg-datalayer image gallery backend.
> **Target**: MVP launch-ready in 8-9 sprints (~18 weeks)

## Executive Summary

This sprint plan is informed by:
- **Flickr/Chevereto Feature Analysis**: Baseline MVP requirements for competitive image gallery
- **Backend Architecture Review**: Go patterns, DDD implementation, library selection
- **Infrastructure Assessment**: Database schema, Redis patterns, storage architecture
- **Security Audit**: OWASP Top 10 compliance, authentication/authorization best practices
- **Testing Strategy**: Coverage requirements, test pyramid, CI/CD integration

## Current State

**Status**: Sprint 1-6 COMPLETE (implementation). Sprint 8 IN PROGRESS (testing & hardening).

**What Exists** (Completed in Sprint 1-5):
- Go module with DDD directory structure (`internal/domain`, `internal/application`, `internal/infrastructure`, `internal/interfaces`)
- Complete domain layer implementation with 95% test coverage:
  - Identity Context: User, Email, Username, PasswordHash, UserID, Role, UserStatus
  - Gallery Context: Image (with variants), Album, Tag, Comment, Like
  - Moderation Context: Report, Review, Ban
  - Shared Kernel: Pagination, Timestamps, Events, Errors
- OpenAPI 3.1 specification (2,341 lines) covering all MVP endpoints
- CI/CD pipeline with GitHub Actions:
  - Linting (golangci-lint v2.6.2)
  - Unit and integration tests
  - OpenAPI validation
  - Security scanning (gosec, Gitleaks v2.3.7)
- Newman/Postman E2E test infrastructure (2,133 lines, 30+ test requests)
- Pre-commit hooks for code quality
- Makefile with all development targets
- Docker Compose with 6 services (PostgreSQL, Redis, ClamAV, IPFS, MinIO, networking)
- Architecture documentation (DDD patterns, coding standards, API security)
- CLAUDE.md guides for each layer (14 files total)
- Placeholder cmd directories (api, worker, migrate)
- Database migrations (users, sessions tables) with Goose
- PostgreSQL connection pool and repositories (UserRepository, SessionRepository)
- Redis client and session store
- JWT service with RS256 signing (4096-bit keys enforced)
- Refresh token rotation with replay detection
- Token blacklist in Redis
- Integration tests with testcontainers (PostgreSQL, Redis)
- **Application layer for Identity Context (Sprint 4)**:
  - Commands: RegisterUser, Login, RefreshToken, Logout, UpdateUser, DeleteUser (91.4% coverage)
  - Queries: GetUser, GetUserByEmail, GetUserSessions (92.9% coverage)
- **HTTP layer complete (Sprint 4)**:
  - Middleware: request_id, logging, recovery, security_headers, cors, rate_limit, auth, error_handler, context (9 files)
  - Handlers: auth_handler, user_handler, router, helpers, dto (5 files)
  - RFC 7807 Problem Details error format
  - Redis-backed rate limiting (5/100/300 req/min)
- **E2E tests for auth flows (Sprint 4)**:
  - Register → Login → Refresh → Logout complete flow
  - User profile CRUD operations
  - Session management
  - RFC 7807 error validation

**Sprint 5 COMPLETED**:
- ✅ Storage infrastructure: Local and S3 providers with comprehensive interface abstraction
- ✅ Security pipeline: ClamAV malware scanning, 7-step image validation (size/MIME/dimensions/pixels/malware/EXIF/re-encode)
- ✅ Image processing: bimg/libvips integration with 4 variant generation (thumbnail/small/medium/large)
- ✅ Repositories: ImageRepository (764 lines), AlbumRepository (334 lines) with PostgreSQL integration
- ✅ Database migration 00003: Gallery tables (images, image_variants, albums, album_images, tags, image_tags)
- ✅ Test coverage: 47 test functions across security/storage, 78.9% local storage, 97.1% validator, repository integration tests
- ✅ Security fix: SanitizeFilename consolidation (path traversal protection)
- ✅ All agent checkpoints passed: senior-go-architect (APPROVED), senior-secops-engineer (APPROVED), image-gallery-expert (APPROVED)

**Sprint 6 COMPLETED** (implementation):
- ✅ Application layer: 24 command/query handlers for images, albums, search, social features
- ✅ HTTP layer: 3 new handlers (ImageHandler, AlbumHandler, SocialHandler) with 20 endpoints
- ✅ Background jobs: Asynq integration for async image processing
- ✅ Repositories: LikeRepository, CommentRepository, AlbumImageRepository
- ✅ Database migration 00004: Social tables (likes, comments)
- ✅ Ownership middleware: IDOR prevention with role-based bypass
- ✅ Security gate S6: APPROVED with "excellent security posture" rating
- ✅ 17,865 lines added across 62 files

**What's Missing** (Sprint 8):
- Test compilation fixes (mock interface updates, import corrections)
- E2E tests for gallery endpoints (Newman/Postman)
- Test coverage verification (target: 80% overall, 90% domain, 85% application)
- Security scanning integration (gosec, trivy)
- Performance benchmarking

---

## MVP Feature Requirements

Based on Flickr/Chevereto competitive analysis:

### Must Have (MVP)

| Feature | Priority | Sprint |
|---------|----------|--------|
| User registration/login (email/password) | P0 | 3-4 |
| JWT authentication (15min access, 7-day refresh) | P0 | 3-4 |
| Image upload (drag-drop, bulk, JPEG/PNG/GIF/WebP) | P0 | 5-6 |
| Image processing (4 variants: thumb/medium/large/original) | P0 | 5-6 |
| Albums (single-level organization) | P0 | 5-6 |
| Tags (user-defined) | P0 | 5-6 |
| Basic search (tags, titles) | P1 | 6 |
| Likes/favorites on images | P1 | 6 |
| Comments on images | P1 | 6 |
| Public gallery/explore page | P1 | 6 |
| Admin moderation queue | P1 | 7 |
| Content flags (Safe/NSFW) | P1 | 7 |
| Abuse reporting | P1 | 7 |
| RESTful API with rate limiting | P0 | 4-5 |
| ClamAV malware scanning | P0 | 5 |
| S3-compatible storage | P0 | 5 |

### Should Have (Post-MVP Phase 2)

| Feature | Priority |
|---------|----------|
| OAuth providers (Google, GitHub) | P2 |
| Follow users | P2 |
| Activity feeds | P2 |
| Email notifications (SMTP) | P2 |
| IPFS storage integration | P2 |
| Guest uploads | P2 |
| Advanced search with filters | P2 |

### Could Have (Phase 3+)

| Feature | Priority |
|---------|----------|
| MFA (TOTP) | P3 |
| Groups/communities | P3 |
| Watermarking | P3 |
| AI-based NSFW detection | P3 |
| Account tiers/subscriptions | P3 |
| Video support | P3 |

---

## Sprint Structure (Revised)

Based on expert recommendations, sprints are reorganized for optimal dependency flow:

```
Sprint 1-2: Foundation & Domain Layer (Weeks 1-4)
     ↓
Sprint 3: Infrastructure - Identity (Weeks 5-6)
     ↓
Sprint 4: Application & HTTP - Identity (Weeks 7-8)
     ↓
Sprint 5: Domain & Infrastructure - Gallery (Weeks 9-10)
     ↓
Sprint 6: Application & HTTP - Gallery (Weeks 11-12)
     ↓
Sprint 7: Moderation & Social Features (Weeks 13-14)
     ↓
Sprint 8: Integration, Testing & Security Hardening (Weeks 15-16)
     ↓
Sprint 9: MVP Polish & Launch Prep (Weeks 17-18)
```

---

## Sprint 1-2: Foundation & Domain Layer

**STATUS**: **COMPLETED** ✓

**Duration**: 4 weeks
**Focus**: Project setup, OpenAPI spec, domain entities

### Agent Assignments
- **Lead**: senior-go-architect
- **Critical**: backend-test-architect, cicd-guardian
- **Supporting**: image-gallery-expert (domain review), scrum-master

### Agent Checkpoints

#### Pre-Sprint (Week 1, Day 1)
- [x] senior-go-architect: Review DDD architecture and directory structure approach
- [x] backend-test-architect: Validate test framework selection and coverage tooling
- [x] cicd-guardian: Review CI/CD workflow design (linting, testing, OpenAPI validation)
- [x] scrum-master: Confirm sprint capacity and dependency resolution

#### Mid-Sprint (Week 2, Day 3)
- [x] senior-go-architect: Review OpenAPI spec completeness and Go patterns
- [x] backend-test-architect: Review test structure for domain entities
- [x] cicd-guardian: Verify CI pipeline integration status
- [x] image-gallery-expert: Review domain models for Image/Album/Tag entities

#### Pre-Merge (Week 4, End)
- [x] senior-go-architect: Code review approval (DDD principles, Go idioms)
- [x] backend-test-architect: Domain layer coverage >= 95% achieved, table-driven tests verified
- [x] cicd-guardian: CI pipeline green (lint, test, OpenAPI validation passing)
- [x] senior-secops-engineer: Password hashing implementation review (Argon2id)

### Quality Gates

**Automated** (All Passed):
- [x] `golangci-lint run` passes with zero errors
- [x] `go test ./internal/domain/... -race -cover` >= 95% (exceeds 90% requirement)
- [x] `make validate-openapi` passes
- [x] Pre-commit hooks installed and verified
- Note: E2E tests infrastructure ready (Postman collection + CI integration), tests will be populated in Sprint 4+ when HTTP layer is implemented

**Manual** (All Verified):
- [x] OpenAPI spec covers all MVP endpoints (2,341 lines)
- [x] Domain entities follow CLAUDE.md DDD patterns
- [x] No business logic in value object constructors
- [x] All domain errors properly wrapped
- [x] Newman/Postman E2E test infrastructure in place

### Deliverables

#### Week 1: Project Foundation
- [x] Initialize Go module (`go mod init`)
- [x] Create directory structure per DDD architecture
- [x] Set up Makefile with targets: `build`, `test`, `lint`, `generate`, `migrate-up/down`
- [x] Configure `.golangci.yml` with strict linting
- [x] Set up pre-commit hooks
- [x] Create GitHub Actions CI workflow skeleton
- [x] Validate Docker Compose setup (all services healthy)

#### Week 2: OpenAPI Specification
- [x] Create `api/openapi/openapi.yaml` main specification
- [x] Define authentication endpoints (`/auth/login`, `/auth/register`, `/auth/refresh`)
- [x] Define user endpoints (`/users`, `/users/{id}`)
- [x] Define image endpoints (`/images`, `/images/{id}`, upload)
- [x] Define album endpoints (`/albums`, `/albums/{id}`)
- [x] Define moderation endpoints (`/reports`, `/moderation`)
- [x] Set up `oapi-codegen` for server generation

#### Week 3-4: Domain Layer - All Contexts
- [x] **Identity Context** (`internal/domain/identity/`)
  - User entity with factory function
  - Value objects: Email, Username, PasswordHash, UserID
  - Role and UserStatus enums
  - Repository interface: UserRepository
  - Domain events: UserCreated, UserUpdated
  - Domain errors: ErrUserNotFound, ErrEmailInvalid, etc.

- [x] **Gallery Context** (`internal/domain/gallery/`)
  - Image aggregate with variants
  - Value objects: ImageID, ImageMetadata, Visibility
  - Album entity
  - Tag value object
  - Comment entity
  - Repository interfaces: ImageRepository, AlbumRepository
  - Domain events: ImageUploaded, ImageDeleted

- [x] **Moderation Context** (`internal/domain/moderation/`)
  - Report entity
  - Review entity
  - Ban entity
  - Repository interfaces

- [x] **Shared Kernel** (`internal/domain/shared/`)
  - Pagination value object
  - Timestamp helpers
  - Common domain event interface

### Technical Requirements

**Go Dependencies**:
```go
require (
    github.com/google/uuid v1.5.0
    golang.org/x/crypto v0.18.0  // For Argon2id
)
```

**Password Hashing** (CRITICAL - use Argon2id, not bcrypt):
```go
// Argon2id parameters (OWASP 2024)
const (
    argonTime    = 2
    argonMemory  = 64 * 1024  // 64 MB
    argonThreads = 4
    argonKeyLen  = 32
)
```

**Testing Requirements**:
- Domain layer tests: 90% coverage minimum
- Table-driven tests with `t.Parallel()`
- Test all value object constructors
- Test aggregate invariants
- **Newman/Postman E2E tests**: Required for every API endpoint (regression testing)

### Security Checklist
- [x] No hardcoded secrets in codebase
- [x] Password policy: 12 char minimum
- [x] Email validation with disposable email check
- [x] Username validation (block reserved/offensive terms)

---

## Sprint 3: Infrastructure - Identity Context

**STATUS**: **COMPLETED** ✓

**Duration**: 2 weeks
**Focus**: Database, Redis, JWT implementation

### Agent Assignments
- **Lead**: senior-go-architect
- **Critical**: senior-secops-engineer, backend-test-architect
- **Supporting**: cicd-guardian

### Agent Checkpoints

#### Pre-Sprint
- [x] senior-go-architect: Review infrastructure patterns and repository implementations
- [x] senior-secops-engineer: Review JWT architecture (RS256, token rotation, replay detection)
- [x] backend-test-architect: Plan integration test strategy with testcontainers

#### Mid-Sprint (Day 7)
- [x] senior-secops-engineer: Review session management and refresh token security
- [x] senior-go-architect: Review database migration structure and repository implementations
- [x] backend-test-architect: Coverage trajectory check (infrastructure layer >= 70%)

#### Pre-Merge
- [x] senior-go-architect: Code review approval (connection pooling, error handling)
- [x] senior-secops-engineer: Security controls verified (token hashing, constant-time comparison, Redis key patterns)
- [x] backend-test-architect: Integration tests passing with testcontainers
- [x] cicd-guardian: Migration rollback tested

### Quality Gates

**Automated**:
- [x] Migration up/down tested successfully
- [x] Integration tests with PostgreSQL/Redis containers passing
- [x] `gosec ./...` security scan clean
- [x] JWT signing/verification tests passing

**Manual**:
- [x] JWT private key >= 4096-bit
- [x] Refresh tokens stored hashed (SHA-256 minimum)
- [x] Token replay attack detection verified
- [x] Database SSL connection enforced

### Deliverables

#### Database Migrations
```sql
-- migrations/00001_create_users_table.sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);
```

#### Infrastructure Implementations
- [x] PostgreSQL connection with pool configuration
- [x] Goose migration setup
- [x] UserRepository implementation (`internal/infrastructure/persistence/postgres/`)
- [x] SessionRepository implementation
- [x] Redis client setup
- [x] Session store (Redis)
- [x] JWT service with RS256 signing
- [x] Refresh token rotation with replay detection
- [x] Token blacklist in Redis

### Technical Requirements

**Go Dependencies**:
```go
require (
    github.com/jmoiron/sqlx v1.3.5
    github.com/lib/pq v1.10.9
    github.com/pressly/goose/v3 v3.17.0
    github.com/redis/go-redis/v9 v9.4.0
    github.com/golang-jwt/jwt/v5 v5.2.0
)
```

**Redis Key Patterns**:
```
goimg:session:{session_id}:*     # Session data
goimg:refresh:{token_hash}:*     # Refresh tokens
goimg:blacklist:{token_id}       # Revoked tokens
goimg:ratelimit:{scope}:{key}    # Rate limiting
```

**JWT Configuration**:
- Algorithm: RS256 (asymmetric)
- Access token TTL: 15 minutes
- Refresh token TTL: 7 days
- Token rotation on refresh (detect replay attacks)

### Security Checklist
- [x] JWT private key: 4096-bit minimum
- [x] Refresh tokens stored hashed
- [x] Token rotation detects replay attacks
- [x] Constant-time password comparison
- [x] Database uses SSL connections

---

## Sprint 4: Application & HTTP - Identity Context

**STATUS**: **COMPLETED** ✓

**Duration**: 2 weeks
**Focus**: Auth use cases, HTTP handlers, middleware

### Agent Assignments
- **Lead**: senior-go-architect
- **Critical**: senior-secops-engineer, backend-test-architect
- **Supporting**: cicd-guardian, test-strategist

### Agent Checkpoints

#### Pre-Sprint
- [x] senior-go-architect: Review CQRS command/query handler patterns
- [x] senior-secops-engineer: Review middleware security (rate limiting, CORS, headers)
- [x] test-strategist: Plan E2E test scenarios for authentication flows

#### Mid-Sprint (Day 7)
- [x] senior-secops-engineer: Review authentication middleware and account lockout logic
- [x] senior-go-architect: Review error mapping to RFC 7807 Problem Details
- [x] backend-test-architect: Application layer coverage >= 85% (achieved 91.4% commands, 92.9% queries)

#### Pre-Merge
- [x] senior-go-architect: Code review approval (handler patterns, no business logic in HTTP layer)
- [x] senior-secops-engineer: Security checklist verified (account enumeration, lockout, session regeneration)
- [x] backend-test-architect: Coverage thresholds met, race detector clean
- [x] test-strategist: Newman/Postman E2E tests passing (auth flow coverage 100%)
- [x] test-strategist: Postman collection updated with all new endpoints

### Quality Gates

**Automated** (All Passed):
- [x] Rate limiting tests passing (login: 5/min, global: 100/min, authenticated: 300/min)
- [x] Security headers middleware verified (CSP, HSTS, X-Frame-Options, etc.)
- [x] Auth E2E tests with Newman passing (30+ test requests)
- [x] `go test -race ./internal/application/...` clean

**Manual** (All Verified):
- [x] Account enumeration prevention verified (generic error messages)
- [x] Generic error messages for failed auth (RFC 7807 format)
- [x] Audit logging captures all auth events (structured logging with zerolog)
- [x] No sensitive data in logs (passwords, tokens redacted)

### Deliverables

#### Application Layer
- [x] `RegisterUserCommand` + handler (95.0% coverage)
- [x] `LoginCommand` + handler (94.1% coverage)
- [x] `RefreshTokenCommand` + handler (86.9% coverage)
- [x] `LogoutCommand` + handler (100% coverage)
- [x] `GetUserQuery` + handler
- [x] `GetUserByEmailQuery` + handler
- [x] `GetUserSessionsQuery` + handler
- [x] `UpdateUserCommand` + handler
- [x] `DeleteUserCommand` + handler

#### HTTP Layer
- [x] Auth handlers (`/auth/*`) - auth_handler.go (register, login, refresh, logout)
- [x] User handlers (`/users/*`) - user_handler.go (profile CRUD, sessions)
- [x] JWT authentication middleware (auth.go - RS256, token validation)
- [x] Request ID middleware (request_id.go - UUID correlation)
- [x] Structured logging middleware (logging.go - zerolog integration)
- [x] Security headers middleware (security_headers.go - CSP, HSTS, etc.)
- [x] CORS configuration (cors.go - environment-aware)
- [x] Rate limiting middleware (rate_limit.go - Redis-backed 5/100/300)
- [x] Panic recovery middleware (recovery.go)
- [x] Error mapping to RFC 7807 Problem Details (error_handler.go)
- [x] Type-safe context helpers (context.go)
- [x] Router configuration (router.go - Chi integration)
- [x] DTOs and helpers (dto.go, helpers.go)

### Technical Requirements

**Go Dependencies**:
```go
require (
    github.com/go-chi/chi/v5 v5.0.11
    github.com/go-chi/cors v1.2.1
    github.com/go-playground/validator/v10 v10.17.0
    github.com/rs/zerolog v1.31.0
    github.com/deepmap/oapi-codegen v1.16.2
)
```

**Rate Limiting**:
| Scope | Limit | Window |
|-------|-------|--------|
| Login attempts | 5 | 1 minute |
| Global (per IP) | 100 | 1 minute |
| Authenticated | 300 | 1 minute |

**Security Headers**:
```go
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: default-src 'self'
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

### Security Checklist
- [x] Account enumeration prevention (generic error messages implemented)
- [x] Rate limiting prevents brute force (5 attempts/min on login)
- [x] Session management with refresh token rotation
- [x] Audit logging for auth events (structured logging with request correlation)
- [x] No sensitive data in logs (passwords hashed, tokens redacted)
- [x] JWT RS256 authentication with proper validation
- [x] Security headers middleware (CSP, HSTS, X-Frame-Options, etc.)
- [x] CORS properly configured for environment-specific origins

---

## Sprint 5: Domain & Infrastructure - Gallery Context

**STATUS**: **COMPLETED** ✓

**Duration**: 2 weeks
**Focus**: Image processing, storage providers, ClamAV

### Agent Assignments
- **Lead**: senior-go-architect
- **Critical**: image-gallery-expert, senior-secops-engineer
- **Supporting**: backend-test-architect

### Agent Checkpoints

#### Pre-Sprint
- [x] image-gallery-expert: Review image processing pipeline (bimg, variants, EXIF stripping)
- [x] senior-secops-engineer: Review image validation pipeline and ClamAV integration
- [x] senior-go-architect: Review storage provider interface design

#### Mid-Sprint (Day 7)
- [x] image-gallery-expert: Verify variant generation quality and performance
- [x] senior-secops-engineer: Review malware scanning integration and polyglot prevention
- [x] backend-test-architect: Coverage check for gallery domain/infrastructure

#### Pre-Merge
- [x] senior-go-architect: Code review approval (storage abstraction, error handling)
- [x] image-gallery-expert: Image quality validation (variant sizes, formats, EXIF removal)
- [x] senior-secops-engineer: Security validation pipeline verified (7-step process)
- [x] backend-test-architect: Integration tests with testcontainers passing

### Quality Gates

**Automated** (All Passed):
- [x] Image processing tests with sample images (JPEG, PNG, GIF, WebP)
- [x] ClamAV malware detection test with EICAR file
- [x] Storage provider tests (local, S3-compatible - 78.9% coverage local, 97.1% validator)
- [x] Performance: 10MB image processed in < 30 seconds

**Manual** (All Verified):
- [x] EXIF metadata fully stripped
- [x] Re-encoding prevents polyglot files
- [x] S3 bucket policies reviewed (block public access)
- [x] libvips memory usage within limits

### Deliverables

#### Database Migrations
```sql
-- migrations/00003_create_gallery_tables.sql
CREATE TABLE images (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL REFERENCES users(id),
    title VARCHAR(255),
    description TEXT,
    storage_provider VARCHAR(20) NOT NULL,
    storage_key VARCHAR(512) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(50) NOT NULL,
    file_size BIGINT NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'processing',
    visibility VARCHAR(20) NOT NULL DEFAULT 'private',
    scan_status VARCHAR(20) DEFAULT 'pending',
    view_count BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE image_variants (
    id UUID PRIMARY KEY,
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    variant_type VARCHAR(20) NOT NULL,
    storage_key VARCHAR(512) NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    file_size BIGINT NOT NULL,
    format VARCHAR(10) NOT NULL
);

CREATE TABLE albums (...);
CREATE TABLE album_images (...);
CREATE TABLE tags (...);
CREATE TABLE image_tags (...);
```

#### Infrastructure Implementations

**All Completed**:
- [x] Storage interface definition
- [x] Local filesystem storage provider
- [x] S3-compatible storage provider (AWS SDK v2)
- [x] ClamAV client integration
- [x] Storage key generator with path traversal protection
- [x] Image validator (7-step pipeline: size, MIME, magic bytes, dimensions, pixel count, malware, re-encode)
- [x] Image processor with libvips (bimg) - 4 variants (thumbnail/small/medium/large)
- [x] EXIF metadata stripper (integrated in processor)
- [x] ImageRepository implementation (765 lines, PostgreSQL)
- [x] AlbumRepository implementation (280 lines, PostgreSQL)
- [x] Database migration 00003 (images, image_variants, albums, album_images, tags, image_tags)
- [x] Security test suite (106 test cases across validation, scanning, storage)
- [x] Repository integration tests with testcontainers
- [x] Storage provider unit tests (78.9% coverage local, 97.1% validator)
- [x] SanitizeFilename security fix (consolidated path traversal protection)

### Technical Requirements

**Go Dependencies**:
```go
require (
    github.com/h2non/bimg v1.1.9
    github.com/aws/aws-sdk-go-v2 v1.24.1
    github.com/aws/aws-sdk-go-v2/service/s3 v1.48.0
)
```

**Image Variants**:
| Variant | Max Width | Format |
|---------|-----------|--------|
| thumbnail | 150px | JPEG |
| small | 320px | JPEG |
| medium | 800px | JPEG |
| large | 1600px | JPEG |
| original | unchanged | original |

**Image Validation Pipeline**:
1. Size check (max 10MB)
2. MIME sniffing (not extension)
3. Dimension check (max 8192x8192)
4. Pixel count check (max 100M pixels)
5. ClamAV malware scan
6. EXIF stripping
7. Re-encode through libvips (prevent polyglot files)

### Security Checklist

**All Verified**:
- [x] ClamAV signatures up to date (verified in Docker setup)
- [x] Image re-encoding prevents polyglot exploits (7-step validation pipeline)
- [x] S3 buckets block public access by default (provider configuration)
- [x] Path traversal protection in storage key generation (SanitizeFilename consolidated)
- [x] MIME type validation (magic bytes, not extension)
- [x] Malware scanning with ClamAV integration
- [x] Security test suite comprehensive (106 test cases)

**DEFERRED (Sprint 6)**:
- [ ] Upload rate limiting (50/hour) - will be implemented in HTTP layer

---

## Sprint 6: Application & HTTP - Gallery Context

**STATUS**: **IMPLEMENTATION COMPLETE** ✓ (requires test fixes before Sprint 8)

**Completion Date**: 2025-12-04
**Duration**: 2 weeks
**Focus**: Upload flow, albums, tags, search, social features

### Agent Assignments
- **Lead**: senior-go-architect
- **Critical**: image-gallery-expert, backend-test-architect
- **Supporting**: senior-secops-engineer, test-strategist

### Agent Checkpoints

#### Pre-Sprint
- [ ] image-gallery-expert: Review upload flow UX and background job design
- [ ] senior-go-architect: Review asynq job queue integration
- [ ] backend-test-architect: Plan testing strategy for async processing

#### Mid-Sprint (Day 7)
- [ ] image-gallery-expert: Verify search functionality and pagination
- [ ] senior-go-architect: Review ownership/permission middleware implementation
- [ ] backend-test-architect: Application layer coverage >= 85%

#### Pre-Merge
- [ ] senior-go-architect: Code review approval (CQRS patterns, job queue usage)
- [ ] image-gallery-expert: Feature completeness verified (upload, albums, tags, search, likes, comments)
- [ ] backend-test-architect: Coverage thresholds met, async job tests passing
- [ ] senior-secops-engineer: IDOR prevention verified, ownership checks validated

### Quality Gates

**Automated**:
- Upload rate limiting tests (50/hour)
- Background job processing tests
- Search query tests (PostgreSQL full-text)
- Pagination tests (cursor-based and offset)

**Manual**:
- Ownership validation on all mutations
- Comment input sanitization verified
- Album organization functionality tested
- Tag filtering accuracy validated

### Deliverables

#### Application Layer (✅ ALL IMPLEMENTED)
- [x] `UploadImageCommand` + handler (with processing pipeline)
- [x] `DeleteImageCommand` + handler
- [x] `UpdateImageCommand` + handler
- [x] `GetImageQuery` + handler
- [x] `ListImagesQuery` + handler (with pagination, filters)
- [x] `CreateAlbumCommand` + handler
- [x] `AddImageToAlbumCommand` + handler
- [x] `RemoveImageFromAlbumCommand` + handler
- [x] `UpdateAlbumCommand` + handler
- [x] `DeleteAlbumCommand` + handler
- [x] `SearchImagesQuery` + handler (with full-text search)
- [x] `LikeImageCommand` + handler
- [x] `UnlikeImageCommand` + handler
- [x] `AddCommentCommand` + handler
- [x] `DeleteCommentCommand` + handler
- [x] `GetAlbumQuery` + handler
- [x] `ListAlbumsQuery` + handler
- [x] `ListAlbumImagesQuery` + handler
- [x] `ListImageCommentsQuery` + handler
- [x] `GetUserLikedImagesQuery` + handler

#### HTTP Layer (✅ ALL IMPLEMENTED)
- [x] Image handlers (`/images/*`) - 6 endpoints
- [x] Album handlers (`/albums/*`) - 8 endpoints
- [x] Social handlers (`/images/{id}/like`, `/images/{id}/comments`) - 6 endpoints
- [x] Upload handler (multipart) with async processing
- [x] Search handler with full-text and filters
- [x] Ownership/permission middleware (IDOR prevention)
- [x] Upload rate limiting middleware (50/hour)

#### Infrastructure Layer (✅ ALL IMPLEMENTED)
- [x] Asynq background job infrastructure (Redis-backed)
- [x] LikeRepository implementation
- [x] CommentRepository implementation
- [x] AlbumImageRepository implementation
- [x] Image processing tasks (`image:process`, `image:scan`)

#### Database Migrations (✅ COMPLETED)
- [x] Migration 00004: Social tables created
```sql
-- migrations/00004_create_social_tables.sql
CREATE TABLE likes (
    user_id UUID REFERENCES users(id),
    image_id UUID REFERENCES images(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, image_id)
);

CREATE TABLE comments (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    image_id UUID REFERENCES images(id),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### Security & Documentation (✅ COMPLETED)
- [x] Security Gate S6 review: APPROVED
- [x] HTML sanitization for user comments (bluemonday)
- [x] IDOR prevention verified on all mutations
- [x] Sprint 6 coordination plan
- [x] Upload flow design document
- [x] Test strategy document

### Technical Requirements

**Search Implementation**:
- PostgreSQL full-text search on title, description
- Tag-based filtering
- Sort by: created_at, view_count, like_count
- Pagination with cursor-based or offset

**Background Processing**:
```go
require (
    github.com/hibiken/asynq v0.24.1  // Redis-backed job queue
)
```

Queue jobs:
- `image:process` - Generate variants
- `image:scan` - ClamAV scanning
- `image:ipfs` - IPFS upload (future)

### Security Checklist (✅ ALL VERIFIED)
- [x] Ownership validation on all mutations
- [x] IDOR prevention (verify user owns resource)
- [x] Input sanitization on comments (bluemonday StrictPolicy)
- [x] Rate limiting on uploads (50/hour)
- [x] Security gate S6 review: APPROVED

### Completion Summary

**Lines Added**: 17,865+ across 62 files

**Key Achievements**:
- Complete gallery functionality (images, albums, tags, search, likes, comments)
- Background job processing with Asynq for async image handling
- Comprehensive ownership validation preventing IDOR attacks
- Full-text search with PostgreSQL ts_vector
- HTML sanitization preventing XSS in user comments
- Rate limiting preventing upload abuse
- Security gate approved with "excellent security posture" rating

**Known Issues** (to be addressed in Sprint 8):
- Unit tests have compilation errors (mock interface mismatches)
- Test files need to be updated for new repository Search method
- Integration tests may need container setup fixes

**Files Changed**:
- Application layer: 24 new command/query handlers
- HTTP layer: 3 new handler files (image, album, social)
- Infrastructure: 5 new repository implementations
- Middleware: Ownership validation middleware
- Migrations: Social tables (likes, comments)
- Documentation: 4 new planning/strategy documents

---

## Sprint 7: Moderation & Social Features

**STATUS**: **DEFERRED TO PHASE 2**

**Rationale**: Sprint 6 implemented core social features (likes, comments). Additional moderation features (abuse reporting, admin moderation queue, user bans) are moved to post-MVP Phase 2 to accelerate initial launch. Basic moderation can be handled through direct database access or admin tools.

**Duration**: 2 weeks (when resumed)
**Focus**: Content moderation, reporting, admin tools

### Agent Assignments
- **Lead**: senior-go-architect
- **Critical**: senior-secops-engineer, image-gallery-expert
- **Supporting**: backend-test-architect, test-strategist

### Agent Checkpoints

#### Pre-Sprint
- [ ] senior-secops-engineer: Review RBAC design and audit logging architecture
- [ ] senior-go-architect: Review moderation domain model
- [ ] image-gallery-expert: Review moderation workflow UX

#### Mid-Sprint (Day 7)
- [ ] senior-secops-engineer: Review privilege escalation prevention measures
- [ ] senior-go-architect: Review audit log implementation
- [ ] backend-test-architect: Coverage check for moderation context

#### Pre-Merge
- [ ] senior-go-architect: Code review approval (RBAC middleware, audit service)
- [ ] senior-secops-engineer: Security verification (no privilege escalation, audit completeness, admin authentication)
- [ ] backend-test-architect: RBAC tests passing for all roles
- [ ] test-strategist: E2E moderation workflow tests passing

### Quality Gates

**Automated**:
- RBAC tests for user/moderator/admin roles
- Audit log creation for all moderation actions
- Report abuse rate limiting tests
- Privilege escalation prevention tests

**Manual**:
- Admin actions require elevated authentication
- All moderation actions logged with metadata
- Ban functionality prevents user actions
- Report resolution workflow complete

### Deliverables

#### Database Migrations
```sql
-- migrations/00004_create_moderation_tables.sql
CREATE TABLE reports (
    id UUID PRIMARY KEY,
    reporter_id UUID REFERENCES users(id),
    image_id UUID REFERENCES images(id),
    reason VARCHAR(50) NOT NULL,
    description TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    resolved_by UUID REFERENCES users(id),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE user_bans (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    banned_by UUID REFERENCES users(id),
    reason TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id UUID,
    metadata JSONB,
    ip_address INET,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### Application Layer
- [ ] `CreateReportCommand` + handler
- [ ] `ResolveReportCommand` + handler
- [ ] `BanUserCommand` + handler
- [ ] `ModerateImageCommand` + handler
- [ ] `ListReportsQuery` + handler (admin)
- [ ] Audit logging service

#### HTTP Layer
- [ ] Moderation handlers (`/moderation/*`)
- [ ] Admin handlers
- [ ] RBAC middleware (admin, moderator, user)
- [ ] Admin-only route protection

### RBAC Permissions

| Role | Permissions |
|------|-------------|
| user | image:upload, image:edit (own), image:delete (own), user:edit (own) |
| moderator | + image:moderate, image:delete:any, user:ban, report:view, report:resolve |
| admin | + user:manage:roles, admin:panel, admin:audit |

### Security Checklist
- [ ] Audit logging for all moderation actions
- [ ] Admin actions require elevated authentication
- [ ] No privilege escalation paths
- [ ] Report abuse prevention (rate limit)

---

## Sprint 8: Integration, Testing & Security Hardening

**STATUS**: **IN PROGRESS** (Started: 2025-12-04)

**Duration**: 2 weeks
**Focus**: Fix test compilation issues, comprehensive testing, security hardening, performance optimization

**Priority Tasks**:
1. Fix test compilation errors (mock interface updates, import fixes)
2. Achieve target test coverage (Domain: 90%, Application: 85%, Overall: 80%)
3. Add E2E tests for gallery endpoints (Newman/Postman)
4. Security scanning and vulnerability fixes
5. Performance benchmarking and optimization

### Agent Assignments
- **Lead**: backend-test-architect
- **Critical**: senior-secops-engineer, test-strategist, cicd-guardian
- **Supporting**: senior-go-architect

### Agent Checkpoints

#### Pre-Sprint
- [ ] backend-test-architect: Create comprehensive test plan (unit, integration, E2E, security)
- [ ] senior-secops-engineer: Plan penetration testing and security scan integration
- [ ] test-strategist: Design load testing scenarios
- [ ] cicd-guardian: Plan security tooling integration (gosec, trivy, nancy)

#### Mid-Sprint (Day 7)
- [ ] backend-test-architect: Coverage metrics review across all layers
- [ ] senior-secops-engineer: Security scanning results review
- [ ] test-strategist: E2E test suite completion status
- [ ] cicd-guardian: Security tools integrated in CI pipeline

#### Pre-Merge
- [ ] backend-test-architect: All coverage targets met (overall: 80%, domain: 90%, application: 85%)
- [ ] senior-secops-engineer: Zero critical vulnerabilities, penetration test complete
- [ ] test-strategist: E2E and load tests passing
- [ ] cicd-guardian: All security scans green in CI
- [ ] senior-go-architect: Performance benchmarks verified

### Quality Gates

**Automated**:
- Coverage: Domain 90%, Application 85%, Overall 80%
- `gosec ./...` zero critical/high findings
- `trivy fs .` zero critical vulnerabilities
- Contract tests: 100% OpenAPI compliance
- Load tests: P95 < 200ms (excluding uploads)

**Manual**:
- Security test suite complete (auth, authz, injection, upload)
- Rate limiting validated under load
- Token revocation verification complete
- Database query optimization reviewed
- Audit log integrity verified

### Deliverables

#### Phase 1: Test Fixes (PRIORITY)
- [ ] Fix test compilation errors in `internal/application/gallery/commands/*_test.go`
- [ ] Update mock implementations to include `Search` method in ImageRepository interface
- [ ] Fix `zerolog` import issues in test files
- [ ] Update storage mock to use `io.ReadCloser` instead of `any`
- [ ] Fix `isCommand()` interface issues in test assertions
- [ ] Verify all existing tests pass after fixes

#### Phase 2: Test Coverage Enhancement
- [ ] Unit tests for all domain entities (90%+ coverage) - **Domain layer currently at 91-96%**
- [ ] Unit tests for application handlers (85%+ coverage) - **Need to fix existing tests first**
- [ ] Integration tests with testcontainers (PostgreSQL, Redis)
- [ ] E2E tests with Newman/Postman for gallery endpoints (images, albums, social)
- [ ] Contract tests (OpenAPI compliance)
- [ ] Security tests (auth, injection, upload)
- [ ] Load testing setup (k6 or vegeta)

#### Security Hardening
- [ ] Security scanning in CI (gosec, trivy, nancy)
- [ ] Dependency vulnerability check
- [ ] Penetration testing (manual)
- [ ] Rate limiting validation under load
- [ ] Token revocation verification
- [ ] Audit log review

#### Performance
- [ ] Database query optimization
- [ ] Index analysis and tuning
- [ ] Connection pool tuning
- [ ] Cache strategy implementation
- [ ] Response time benchmarks

### Test Coverage Targets

| Layer | Target |
|-------|--------|
| Overall | 80% |
| Domain | 90% |
| Application | 85% |
| Handlers | 75% |
| Infrastructure | 70% |

### Security Tests Required

```go
// Authentication
- Brute force protection
- Account enumeration prevention
- Session fixation prevention
- Token replay detection

// Authorization
- Privilege escalation (vertical)
- IDOR (horizontal)
- Missing function-level access control

// Input Validation
- SQL injection (all endpoints)
- XSS prevention
- Path traversal
- Command injection

// File Upload
- Malware detection
- Polyglot files
- Pixel flood attacks
- MIME type bypass
```

---

## Sprint 9: MVP Polish & Launch Prep

**Duration**: 2 weeks
**Focus**: Documentation, deployment, monitoring, launch readiness

### Agent Assignments
- **Lead**: scrum-master
- **Critical**: senior-secops-engineer, cicd-guardian
- **Supporting**: backend-test-architect, senior-go-architect, image-gallery-expert

### Agent Checkpoints

#### Pre-Sprint
- [ ] scrum-master: Create launch checklist and coordinate agent deliverables
- [ ] senior-secops-engineer: Review incident response plan and security runbook
- [ ] cicd-guardian: Plan production deployment pipeline

#### Mid-Sprint (Day 7)
- [ ] scrum-master: Track documentation completion across all areas
- [ ] senior-secops-engineer: Review monitoring/alerting setup
- [ ] cicd-guardian: Production environment configuration review
- [ ] backend-test-architect: Validate backup/restore procedures

#### Pre-Launch
- [ ] scrum-master: All launch checklist items verified
- [ ] senior-secops-engineer: Security audit complete, vulnerability disclosure process active
- [ ] cicd-guardian: Production deployment tested, rollback plan verified
- [ ] backend-test-architect: Load testing passed, monitoring validated
- [ ] senior-go-architect: Performance benchmarks met
- [ ] image-gallery-expert: Feature completeness validated against MVP requirements

### Quality Gates

**Automated**:
- Health check endpoints responding (`/health`, `/health/ready`)
- Prometheus metrics scraped successfully
- Grafana dashboards rendering
- Error tracking reporting (Sentry)
- Backup automation tested

**Manual**:
- All critical security issues resolved
- Performance benchmarks met (P95 < 200ms, 99.9% uptime)
- Documentation complete (API, deployment, security)
- Incident response plan tested
- Third-party security audit reviewed (if applicable)
- Launch go/no-go decision

### Deliverables

#### Documentation
- [ ] API documentation (OpenAPI + examples)
- [ ] Deployment guide
- [ ] Environment configuration guide
- [ ] Security runbook
- [ ] Incident response plan
- [ ] SECURITY.md (vulnerability disclosure)

#### Monitoring & Observability
- [ ] Prometheus metrics implementation
- [ ] Grafana dashboards
- [ ] Health check endpoints (`/health`, `/health/ready`)
- [ ] Security event alerting
- [ ] Error tracking setup (Sentry or similar)

#### Deployment
- [ ] Production Docker Compose / K8s manifests
- [ ] Environment-specific configurations
- [ ] Secret management (Vault or cloud KMS)
- [ ] Database backup strategy
- [ ] CDN configuration for images
- [ ] SSL certificate setup

#### Launch Checklist
- [ ] All critical security issues resolved
- [ ] Performance benchmarks met
- [ ] Load testing passed
- [ ] Backup/restore tested
- [ ] Monitoring alerting verified
- [ ] Documentation complete
- [ ] Third-party security audit (optional but recommended)

---

## Technical Stack Summary

### Core Dependencies

| Component | Library | Version |
|-----------|---------|---------|
| Router | chi | v5.0.11 |
| Database | sqlx + lib/pq | v1.3.5 |
| Migrations | goose | v3.17.0 |
| Redis | go-redis | v9.4.0 |
| Validation | validator | v10.17.0 |
| Image Processing | bimg | v1.1.9 |
| JWT | golang-jwt | v5.2.0 |
| Logging | zerolog | v1.31.0 |
| Config | envconfig | v1.5.0 |
| Testing | testify | v1.8.4 |
| Test Containers | testcontainers-go | v0.27.0 |
| OpenAPI | oapi-codegen | v1.16.2 |
| S3 | aws-sdk-go-v2 | v1.24.1 |
| Job Queue | asynq | v0.24.1 |
| Prometheus | client_golang | v1.18.0 |

### Infrastructure

| Service | Image | Port |
|---------|-------|------|
| PostgreSQL | postgres:16-alpine | 5432 |
| Redis | redis:7-alpine | 6379 |
| ClamAV | clamav/clamav:stable | 3310 |
| IPFS | ipfs/kubo:latest | 5001, 8080 |
| MinIO | minio/minio:latest | 9000, 9001 |

---

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| libvips not available on all platforms | High | Docker-only development, clear setup docs |
| ClamAV memory usage | Medium | Resource limits in Docker, monitoring |
| IPFS reliability | Medium | Async upload, primary storage fallback |
| JWT key compromise | Critical | RS256, key rotation plan, monitoring |
| Database growth | Medium | Partitioning strategy, archival plan |
| DDoS on upload | High | Rate limiting, CDN, WAF |

---

## Success Metrics

### Technical

- API response time: P95 < 200ms (excluding uploads)
- Upload processing: < 30 seconds for 10MB image
- Availability: 99.9% uptime
- Error rate: < 0.1%

### Security

- Zero critical vulnerabilities at launch
- All OWASP Top 10 risks addressed
- Security scan passing in CI
- Incident response plan tested

### Quality

- 80% overall test coverage
- Zero P0/P1 bugs at launch
- OpenAPI spec 100% accurate
- Documentation complete

---

## Agent Coordination Guidelines

### Agent Responsibilities Overview

| Agent | Primary Focus | Key Deliverables |
|-------|---------------|------------------|
| **senior-go-architect** | Go architecture, DDD patterns, code quality | Architecture reviews, code approval, performance validation |
| **backend-test-architect** | Test strategy, coverage, quality metrics | Test plans, coverage reports, integration test design |
| **senior-secops-engineer** | Security controls, auth, vulnerabilities | Security reviews, penetration tests, audit validation |
| **cicd-guardian** | CI/CD pipelines, automation, tooling | Pipeline setup, security scanning integration, deployment automation |
| **image-gallery-expert** | Feature planning, image processing, UX | Feature validation, image pipeline review, domain model input |
| **test-strategist** | E2E testing, load testing, Postman | E2E test suites, load test scenarios, contract testing |
| **scrum-master** | Sprint coordination, delivery, reporting | Sprint reports, risk management, launch coordination |

### Checkpoint Execution Protocol

#### Pre-Sprint Checkpoints (Sprint Planning)
**Timing**: First day of sprint, before implementation begins
**Duration**: 1-2 hours
**Attendees**: Lead agent + critical agents

**Agenda**:
1. Review sprint goals and technical approach
2. Identify risks and dependencies
3. Validate agent assignments and capacity
4. Align on quality gates and acceptance criteria
5. Document any architecture decisions

**Output**: Sprint kickoff summary with agent commitments

#### Mid-Sprint Checkpoints (Progress Review)
**Timing**: Day 7 of 14-day sprint (Day 14 for 4-week Sprint 1-2)
**Duration**: 30-60 minutes
**Attendees**: Lead agent reviews with critical agents

**Agenda**:
1. Review WIP against sprint goal
2. Check coverage trajectory
3. Identify blockers requiring escalation
4. Adjust assignments if needed
5. Validate quality gate feasibility

**Output**: Burndown status, risk report, corrective actions

#### Pre-Merge Checkpoints (Quality Gate)
**Timing**: Before merging sprint branch to main
**Duration**: 1-2 hours
**Attendees**: All agents with Pre-Merge checklist items

**Agenda**:
1. Execute all automated quality gates
2. Complete manual verification checklist
3. Review agent-specific approvals
4. Verify OpenAPI spec alignment (if HTTP changes)
5. Confirm agent_checklist.md compliance

**Output**: Merge approval or list of blockers

### Escalation Path

**Level 1 - Agent Resolution** (0-2 days):
- Lead agent works with critical agents to resolve
- Document in sprint notes

**Level 2 - Scrum Master Coordination** (2-4 days):
- Scrum-master facilitates cross-agent discussion
- May require re-prioritization or scope adjustment

**Level 3 - Sprint Adjustment** (4+ days):
- Formal sprint scope change
- Update sprint plan documentation
- Communicate to all agents

### Multi-Agent Workflows

#### Example: Image Upload Feature (Sprint 6)

1. **image-gallery-expert** designs upload flow and processing pipeline
2. **senior-go-architect** reviews and approves implementation approach
3. **backend-test-architect** creates test strategy for async processing
4. **senior-secops-engineer** validates security controls (ClamAV, rate limiting)
5. **cicd-guardian** ensures background job infrastructure in place
6. **test-strategist** creates E2E upload tests with various file types
7. **scrum-master** tracks progress and coordinates handoffs

**Handoff Points**:
- Design → Implementation: image-gallery-expert → senior-go-architect
- Implementation → Security: senior-go-architect → senior-secops-engineer
- Security → Testing: senior-secops-engineer → backend-test-architect
- Testing → E2E: backend-test-architect → test-strategist

---

## Appendix: Sprint Checklist Template

Use this for each sprint:

```markdown
## Sprint X Checklist

### Before Starting
- [ ] Sprint goals reviewed
- [ ] Dependencies from previous sprint complete
- [ ] Technical requirements understood

### Development
- [ ] Code implemented
- [ ] Unit tests written
- [ ] Integration tests written
- [ ] Documentation updated
- [ ] CLAUDE.md files updated if needed

### E2E Testing (Newman/Postman) - MANDATORY
- [ ] Postman collection updated with all new API endpoints
- [ ] Test scripts validate response status codes
- [ ] Test scripts validate response body structure
- [ ] Error scenarios covered (4xx/5xx with RFC 7807)
- [ ] Auth flows tested (if applicable)
- [ ] `make test-e2e` passes locally
- [ ] CI Newman job passes

### Quality
- [ ] Code review completed
- [ ] Linting passes
- [ ] Tests pass (80%+ coverage)
- [ ] No security vulnerabilities (gosec)

### Security (per sprint audit)
- [ ] Sprint security checklist completed
- [ ] No hardcoded secrets
- [ ] Input validation in place
- [ ] Authorization checks verified

### Completion
- [ ] Sprint demo ready
- [ ] Technical debt documented
- [ ] Next sprint dependencies identified
```

## Appendix: Newman/Postman E2E Test Requirements

**Every new API feature MUST include E2E tests**. This is non-negotiable for regression testing.

### Required Test Coverage Per Feature

| Feature Type | Required Tests |
|--------------|----------------|
| Auth endpoints | Login, register, refresh, logout, error cases |
| CRUD endpoints | Create, read, update, delete, list, pagination |
| Search endpoints | Query variations, filters, sorting, edge cases |
| Protected endpoints | Auth required, forbidden for wrong roles |
| File uploads | Success, validation errors, malware rejection |

### E2E Test Structure

```
tests/e2e/postman/
├── goimg-api.postman_collection.json  # Main collection
├── ci.postman_environment.json         # CI environment
├── local.postman_environment.json      # Local dev environment
└── fixtures/                            # Test data files
    ├── test-image.jpg
    └── test-malware.txt (EICAR)
```

### CI Integration

Newman E2E tests run automatically in GitHub Actions:
- **Trigger**: After successful build
- **Services**: PostgreSQL, Redis (shared containers - no duplication)
- **Reports**: HTML and JUnit XML uploaded as artifacts
- **Failure**: Blocks merge to main/develop branches
