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

**Status**: Greenfield project with comprehensive documentation, no Go code implemented.

**What Exists**:
- Docker Compose with 6 services (PostgreSQL, Redis, ClamAV, IPFS, MinIO, networking)
- Architecture documentation (DDD patterns, coding standards, API security)
- CLAUDE.md guides for each layer

**What's Missing**:
- All Go implementation code
- Database migrations
- OpenAPI specification
- Tests
- CI/CD workflows

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

**Duration**: 4 weeks
**Focus**: Project setup, OpenAPI spec, domain entities

### Deliverables

#### Week 1: Project Foundation
- [ ] Initialize Go module (`go mod init`)
- [ ] Create directory structure per DDD architecture
- [ ] Set up Makefile with targets: `build`, `test`, `lint`, `generate`, `migrate-up/down`
- [ ] Configure `.golangci.yml` with strict linting
- [ ] Set up pre-commit hooks
- [ ] Create GitHub Actions CI workflow skeleton
- [ ] Validate Docker Compose setup (all services healthy)

#### Week 2: OpenAPI Specification
- [ ] Create `api/openapi/openapi.yaml` main specification
- [ ] Define authentication endpoints (`/auth/login`, `/auth/register`, `/auth/refresh`)
- [ ] Define user endpoints (`/users`, `/users/{id}`)
- [ ] Define image endpoints (`/images`, `/images/{id}`, upload)
- [ ] Define album endpoints (`/albums`, `/albums/{id}`)
- [ ] Define moderation endpoints (`/reports`, `/moderation`)
- [ ] Set up `oapi-codegen` for server generation

#### Week 3-4: Domain Layer - All Contexts
- [ ] **Identity Context** (`internal/domain/identity/`)
  - User entity with factory function
  - Value objects: Email, Username, PasswordHash, UserID
  - Role and UserStatus enums
  - Repository interface: UserRepository
  - Domain events: UserCreated, UserUpdated
  - Domain errors: ErrUserNotFound, ErrEmailInvalid, etc.

- [ ] **Gallery Context** (`internal/domain/gallery/`)
  - Image aggregate with variants
  - Value objects: ImageID, ImageMetadata, Visibility
  - Album entity
  - Tag value object
  - Comment entity
  - Repository interfaces: ImageRepository, AlbumRepository
  - Domain events: ImageUploaded, ImageDeleted

- [ ] **Moderation Context** (`internal/domain/moderation/`)
  - Report entity
  - Review entity
  - Ban entity
  - Repository interfaces

- [ ] **Shared Kernel** (`internal/domain/shared/`)
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

### Security Checklist
- [ ] No hardcoded secrets in codebase
- [ ] Password policy: 12 char minimum
- [ ] Email validation with disposable email check
- [ ] Username validation (block reserved/offensive terms)

---

## Sprint 3: Infrastructure - Identity Context

**Duration**: 2 weeks
**Focus**: Database, Redis, JWT implementation

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
- [ ] PostgreSQL connection with pool configuration
- [ ] Goose migration setup
- [ ] UserRepository implementation (`internal/infrastructure/persistence/postgres/`)
- [ ] SessionRepository implementation
- [ ] Redis client setup
- [ ] Session store (Redis)
- [ ] JWT service with RS256 signing
- [ ] Refresh token rotation with replay detection
- [ ] Token blacklist in Redis

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
- [ ] JWT private key: 4096-bit minimum
- [ ] Refresh tokens stored hashed
- [ ] Token rotation detects replay attacks
- [ ] Constant-time password comparison
- [ ] Database uses SSL connections

---

## Sprint 4: Application & HTTP - Identity Context

**Duration**: 2 weeks
**Focus**: Auth use cases, HTTP handlers, middleware

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
