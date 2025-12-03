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

**Status**: Sprint 1-3 COMPLETE. Sprint 4 (Application & HTTP - Identity Context) in progress.

**What Exists** (Completed in Sprint 1-3):
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
- Newman/Postman E2E test infrastructure (collection + CI environment)
- Pre-commit hooks for code quality
- Makefile with all development targets
- Docker Compose with 6 services (PostgreSQL, Redis, ClamAV, IPFS, MinIO, networking)
- Architecture documentation (DDD patterns, coding standards, API security)
- CLAUDE.md guides for each layer
- Placeholder cmd directories (api, worker, migrate)
- Database migrations (users, sessions tables) with Goose
- PostgreSQL connection pool and repositories (UserRepository, SessionRepository)
- Redis client and session store
- JWT service with RS256 signing (4096-bit keys enforced)
- Refresh token rotation with replay detection
- Token blacklist in Redis
- Integration tests with testcontainers (PostgreSQL, Redis)

**What's Missing** (Sprint 4+):
- Application layer commands and queries (Sprint 4+)
- HTTP handlers and middleware (Sprint 4+)
- Image processing and storage providers (Sprint 5+)
- ClamAV integration (Sprint 5+)

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

**Duration**: 2 weeks
**Focus**: Auth use cases, HTTP handlers, middleware

### Agent Assignments
- **Lead**: senior-go-architect
- **Critical**: senior-secops-engineer, backend-test-architect
- **Supporting**: cicd-guardian, test-strategist

### Agent Checkpoints

#### Pre-Sprint
- [ ] senior-go-architect: Review CQRS command/query handler patterns
- [ ] senior-secops-engineer: Review middleware security (rate limiting, CORS, headers)
- [ ] test-strategist: Plan E2E test scenarios for authentication flows

#### Mid-Sprint (Day 7)
- [ ] senior-secops-engineer: Review authentication middleware and account lockout logic
- [ ] senior-go-architect: Review error mapping to RFC 7807 Problem Details
- [ ] backend-test-architect: Application layer coverage >= 85%

#### Pre-Merge
- [ ] senior-go-architect: Code review approval (handler patterns, no business logic in HTTP layer)
- [ ] senior-secops-engineer: Security checklist verified (account enumeration, lockout, session regeneration)
- [ ] backend-test-architect: Coverage thresholds met, race detector clean
- [ ] test-strategist: Newman/Postman E2E tests passing (auth flow coverage 100%)
- [ ] test-strategist: Postman collection updated with all new endpoints

### Quality Gates

**Automated**:
- Rate limiting tests passing (login: 5/min, global: 100/min)
- Security headers middleware verified
- Auth E2E tests with Newman passing
- `go test -race ./internal/application/...` clean

**Manual**:
- Account enumeration prevention verified
- Generic error messages for failed auth
- Audit logging captures all auth events
- No sensitive data in logs (passwords, tokens)

### Deliverables

#### Application Layer
- [ ] `RegisterUserCommand` + handler
- [ ] `LoginCommand` + handler
- [ ] `RefreshTokenCommand` + handler
- [ ] `LogoutCommand` + handler
- [ ] `GetUserQuery` + handler
- [ ] `UpdateUserCommand` + handler

#### HTTP Layer
- [ ] Auth handlers (`/auth/*`)
- [ ] User handlers (`/users/*`)
- [ ] JWT authentication middleware
- [ ] Request ID middleware
- [ ] Structured logging middleware (zerolog)
- [ ] Security headers middleware
- [ ] CORS configuration
- [ ] Error mapping to RFC 7807 Problem Details

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
- [ ] Account enumeration prevention (generic error messages)
- [ ] Account lockout after 5 failed attempts
- [ ] Session ID regeneration on login
- [ ] Audit logging for auth events
- [ ] No sensitive data in logs

---

## Sprint 5: Domain & Infrastructure - Gallery Context

**Duration**: 2 weeks
**Focus**: Image processing, storage providers, ClamAV

### Agent Assignments
- **Lead**: senior-go-architect
- **Critical**: image-gallery-expert, senior-secops-engineer
- **Supporting**: backend-test-architect

### Agent Checkpoints

#### Pre-Sprint
- [ ] image-gallery-expert: Review image processing pipeline (bimg, variants, EXIF stripping)
- [ ] senior-secops-engineer: Review image validation pipeline and ClamAV integration
- [ ] senior-go-architect: Review storage provider interface design

#### Mid-Sprint (Day 7)
- [ ] image-gallery-expert: Verify variant generation quality and performance
- [ ] senior-secops-engineer: Review malware scanning integration and polyglot prevention
- [ ] backend-test-architect: Coverage check for gallery domain/infrastructure

#### Pre-Merge
- [ ] senior-go-architect: Code review approval (storage abstraction, error handling)
- [ ] image-gallery-expert: Image quality validation (variant sizes, formats, EXIF removal)
- [ ] senior-secops-engineer: Security validation pipeline verified (6-step process)
- [ ] backend-test-architect: Integration tests with ClamAV container passing

### Quality Gates

**Automated**:
- Image processing tests with sample images (JPEG, PNG, GIF, WebP)
- ClamAV malware detection test with EICAR file
- Storage provider tests (local, S3-compatible)
- Performance: 10MB image processed in < 30 seconds

**Manual**:
- EXIF metadata fully stripped
- Re-encoding prevents polyglot files
- S3 bucket policies reviewed (block public access)
- libvips memory usage within limits

### Deliverables

#### Database Migrations
```sql
-- migrations/00002_create_gallery_tables.sql
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
- [ ] Storage interface definition
- [ ] Local filesystem storage provider
- [ ] S3-compatible storage provider (AWS SDK v2)
- [ ] ClamAV client integration
- [ ] Image processor with libvips (bimg)
- [ ] Image validator (size, MIME, pixels, malware)
- [ ] EXIF metadata stripper
- [ ] Image repository implementation
- [ ] Album repository implementation

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
- [ ] ClamAV signatures up to date
- [ ] Image re-encoding prevents polyglot exploits
- [ ] S3 buckets block public access by default
- [ ] Upload rate limiting (50/hour)

---

## Sprint 6: Application & HTTP - Gallery Context

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

#### Application Layer
- [ ] `UploadImageCommand` + handler (with processing pipeline)
- [ ] `DeleteImageCommand` + handler
- [ ] `UpdateImageCommand` + handler
- [ ] `GetImageQuery` + handler
- [ ] `ListImagesQuery` + handler (with pagination, filters)
- [ ] `CreateAlbumCommand` + handler
- [ ] `AddImageToAlbumCommand` + handler
- [ ] `SearchImagesQuery` + handler
- [ ] `LikeImageCommand` + handler
- [ ] `AddCommentCommand` + handler

#### HTTP Layer
- [ ] Image handlers (`/images/*`)
- [ ] Album handlers (`/albums/*`)
- [ ] Upload handler (multipart)
- [ ] Search handler
- [ ] Ownership/permission middleware
- [ ] Upload rate limiting middleware

#### Database Migrations
```sql
-- migrations/00003_create_social_tables.sql
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

### Security Checklist
- [ ] Ownership validation on all mutations
- [ ] IDOR prevention (verify user owns resource)
- [ ] Input sanitization on comments
- [ ] Rate limiting on uploads

---

## Sprint 7: Moderation & Social Features

**Duration**: 2 weeks
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

**Duration**: 2 weeks
**Focus**: Comprehensive testing, security hardening, performance

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

#### Testing
- [ ] Unit tests for all domain entities (90%+ coverage)
- [ ] Unit tests for application handlers (85%+ coverage)
- [ ] Integration tests with testcontainers (PostgreSQL, Redis)
- [ ] E2E tests with Newman/Postman
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
