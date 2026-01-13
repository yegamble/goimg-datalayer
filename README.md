# goimg-datalayer

Go backend for an image gallery web application (Flickr/Chevereto-style). Provides secure image hosting, user management, and content moderation.

## Status

**Current Phase**: **Phase 3 - Advanced Features** | **Sprint 20 COMPLETE** ✅

**Sprint 20 (COMPLETE)**: Groups/Communities
- ✅ Domain layer complete (community bounded context, 97.5% test coverage)
- ✅ Database migration complete (00017_create_groups.sql)
- ✅ Infrastructure layer complete (PostgreSQL repositories)
- ✅ Application layer complete (commands/queries)
- ✅ HTTP handlers and OpenAPI spec complete
- ✅ Security Gate S20: 9/10 controls passed
- ✅ Group Albums (P1) - COMPLETE (7 endpoints)
- ✅ Contract tests and performance testing COMPLETE

**Sprint 19 Complete** ✅: Trending Tags + Featured Picks
- ✅ TagRepository interface and PostgreSQL implementation
- ✅ Popular Tags API (`GET /api/v1/tags/popular`)
- ✅ Trending Tags API (`GET /api/v1/tags/trending`)
- ✅ Tag Search API (`GET /api/v1/tags/search`)
- ✅ HTTP Handlers (TagHandler) wired in router
- ✅ OpenAPI spec for tag endpoints
- ✅ E2E Tests for tag endpoints (6 Newman tests)
- ✅ Featured Picks database migration with scheduling
- ✅ FeaturedPick domain entity and repository
- ✅ Feature/Unfeature commands and ListFeatured query
- ✅ OpenAPI spec for Featured Picks (`/explore/featured`, `/moderation/featured`)
- ✅ FeaturedHandler HTTP handler wired in router
- ✅ ExploreHandler featured endpoint (`GET /api/v1/explore/featured`)
- ✅ Admin Featured management (`POST/DELETE /api/v1/moderation/featured`)
- ✅ E2E Tests for Featured Picks (6 Newman tests)
- ✅ Contract tests for Featured Picks endpoints

**Sprint 18 Complete** ✅: Test Coverage Improvement
- ✅ Overall test coverage improved from 35.9% to ~65%
- ✅ 7 of 9 priority packages now exceed targets
- ✅ Key improvements: redis (93.8%), domain/identity (95.1%), jwt (87.8%)
- ✅ storage/orchestrator (95.1%), identity/queries (95.8%)

**Phase 2 Complete** ✅: All Sprints 10-15 delivered:
- ✅ Sprint 10: Security Enhancements (timing attack mitigation, HIBP password check)
- ✅ Sprint 11: Two-Factor Authentication (TOTP, backup codes, encrypted secrets)
- ✅ Sprint 12: OAuth & Social Features (Google/GitHub OAuth, follows, activity feeds)
- ✅ Sprint 13: IPFS Storage Integration (decentralized storage with dual-mode orchestration)
- ✅ Sprint 14: Content Moderation + Guest Uploads (abuse reports, admin queue, user bans)
- ✅ Sprint 15: AI NSFW Detection + Advanced Search (SightEngine/ModerateContent, filters)

**Sprint 16 Complete** ✅: oEmbed + Social Media Cards
- ✅ oEmbed 1.0 endpoint (`GET /api/v1/oembed`) with JSON/XML support
- ✅ Open Graph meta tags for Facebook/LinkedIn
- ✅ Twitter Card support for rich previews
- ✅ Image preview page (`GET /images/{id}/preview`)
- ✅ E2E tests (8 Newman tests)

**Sprint 17 Complete** ✅: Nested Albums + Custom Variants
- ✅ Nested albums with parent-child hierarchy
- ✅ Album breadcrumb navigation (`GET /albums/{id}/breadcrumb`)
- ✅ Album children listing (`GET /albums/{id}/children`)
- ✅ VariantConfig CRUD API for custom variant presets
- ✅ Custom variant generation (`POST /api/v1/images/{id}/variants`)
- ✅ Support for fit, fill, and crop modes
- ✅ Output formats: jpeg, png, webp, avif
- ✅ E2E tests (17 tests for nested albums, variant configs, custom variants)

**Phase 1 Complete** ✅: All Sprints 1-9 delivered (MVP LAUNCHED):

- **Sprint 1-2: Foundation & Domain Layer** (4 weeks) ✅
  - Project setup with DDD architecture
  - OpenAPI 3.0 specification (5,460+ lines)
  - Complete domain layer (Identity, Gallery, Moderation, Shared contexts)
  - Domain layer test coverage: 91-100% (exceeds 90% requirement)
  - CI/CD pipeline (GitHub Actions with linting, testing, security scanning)
  - Newman/Postman E2E test infrastructure

- **Sprint 3: Infrastructure - Identity Context** (2 weeks) ✅
  - Database migrations (users, sessions tables)
  - PostgreSQL connection pool and repositories (UserRepository, SessionRepository)
  - Redis client and session store
  - JWT service with RS256 signing (4096-bit keys enforced)
  - Refresh token rotation with replay detection
  - Token blacklist in Redis
  - Integration tests with testcontainers (PostgreSQL, Redis)

- **Sprint 4: Application & HTTP - Identity Context** (2 weeks) ✅
  - Application layer commands and queries (91.4% and 92.9% test coverage)
  - HTTP middleware (9 components): request_id, logging, recovery, security_headers, cors, rate_limit, auth, error_handler, context
  - HTTP handlers: auth_handler, user_handler, router, helpers, dto
  - RFC 7807 Problem Details error format
  - Redis-backed rate limiting (5/100/300 req/min)
  - 30+ E2E tests covering complete auth flow

- **Sprint 5: Domain & Infrastructure - Gallery Context** (2 weeks) ✅
  - Storage infrastructure: Local and S3 providers with comprehensive interface abstraction
  - Security pipeline: ClamAV malware scanning, 7-step image validation (size/MIME/dimensions/pixels/malware/EXIF/re-encode)
  - Image processing: bimg/libvips integration with 4 variant generation (thumbnail/small/medium/large)
  - Repositories: ImageRepository (764 lines), AlbumRepository (334 lines) with PostgreSQL integration
  - Database migration 00003: Gallery tables (images, image_variants, albums, album_images, tags, image_tags)
  - Test coverage: 78.9% local storage, 97.1% validator, repository integration tests
  - Security fix: SanitizeFilename consolidation (path traversal protection)

- **Sprint 6: Application & HTTP - Gallery Context** (2 weeks) ✅
  - Application layer: 24 command/query handlers for images, albums, search, social features
  - HTTP handlers: ImageHandler (6 endpoints), AlbumHandler (8 endpoints), SocialHandler (6 endpoints)
  - Asynq background job infrastructure for async image processing
  - Repositories: LikeRepository, CommentRepository, AlbumImageRepository
  - Database migration 00004: Social tables (likes, comments)
  - Ownership middleware with IDOR prevention (verified by security gate)
  - Upload rate limiting (50/hour), HTML sanitization for comments
  - Security gate S6: APPROVED (comprehensive defense-in-depth controls)

- **Sprint 7: Moderation & Social Features** - **DEFERRED TO PHASE 2** 🔄
  - Core social features (likes, comments) already implemented in Sprint 6
  - Advanced moderation features moved to post-MVP Phase 2
  - Rationale: Accelerate MVP launch; basic moderation via direct database access

- **Sprint 8: Integration, Testing & Security Hardening** (2 weeks) ✅
  - **Test Coverage Achievements** (EXCEEDED targets):
    - Gallery commands: 32.8% → **93.4%** (target: 85%, +60.6pp)
    - Gallery queries: 49.5% → **94.2%** (target: 85%, +44.7pp)
    - Domain layer: **91-100%** (target: 90%)
    - Identity application: **91-93%** (target: 85%)
  - **Security Audit**: Rating **B+** (0 critical/high vulnerabilities)
  - **E2E Tests**: 62 total test requests, 60% endpoint coverage across 9 feature areas
  - **CI/CD Hardening**: Go 1.25 pinned, Trivy exit codes fixed, Gitleaks v8.23.0 pinned
  - **Performance Optimization**: N+1 query elimination (97% reduction), database indexes (migration 00005)
  - **Security Configurations**: .gitleaks.toml, .trivyignore
  - **Test Files Added**: 13 new test files, 130+ comprehensive test functions

- **Sprint 9: MVP Polish & Launch Prep** (2 weeks) ✅ **COMPLETE - GO FOR LAUNCH**
  - **Progress**: 100% complete (22 of 22 tasks)
  - **Security Gate S9**: 100% complete (10 of 10 controls passed)
  - **Penetration Test**: A- Rating (Excellent)
  - **Audit Logging**: Grade A (SOC 2, GDPR, CCPA compliant)
  - **Launch Decision**: ✅ **GO FOR LAUNCH** - All criteria met
  - **Key Deliverables**:
    - API documentation (2,694 lines with code examples)
    - Deployment guide with Docker, Kubernetes, and cloud configurations
    - Security runbook and incident response plan
    - Prometheus metrics, Grafana dashboards, health checks
    - CDN configuration, SSL/TLS setup, secret management
    - Backup/restore procedures (RTO: 18m 42s, 37.7% below target)

See [claude/sprint_plan.md](claude/sprint_plan.md) for the complete roadmap.

## Recent Achievements

### Phase 2 Complete (2026-01-09) ✅

All Phase 2 sprints (10-15) have been successfully delivered. The platform now includes:

**Sprint 15 - AI Content Moderation**:
- Multi-provider NSFW detection (SightEngine + ModerateContent fallback)
- Advanced search filters (date, size, dimensions, NSFW status)
- Content flagging workflow for moderators

**Sprint 14 - Content Moderation & Guest Uploads**:
- Abuse reporting system with rate limiting
- Admin moderation queue (10 endpoints)
- User ban system (temporary & permanent)
- Guest upload sessions with claim-to-account flow

**Sprint 13 - IPFS Decentralized Storage**:
- Pin/unpin images to IPFS network
- Dual-storage orchestrator (primary + IPFS fallback)
- Gateway URL generation for content-addressed retrieval

**Sprint 12 - OAuth & Social Features**:
- OAuth authentication (Google, GitHub) with CSRF protection
- Follow/unfollow users with activity feeds
- Email notifications via SMTP with rate limiting

### Sprint 9 Complete - GO FOR LAUNCH ✅

**Security Gate S9: 100% COMPLETE** - All 10 launch-blocking controls passed

**Launch Decision: GO FOR LAUNCH** 🚀
- Mandatory Criteria: 8/8 passed
- Important Criteria: 6/6 passed
- Security Rating: A-
- Weighted Score: 97/100 (+14% above 85/100 threshold)
- Confidence Level: 95%

**Documentation (Complete)**:
- API documentation: 2,694 lines with code examples (curl, JavaScript, Python)
- Deployment guide: Docker, Kubernetes, and cloud configurations
- Environment configuration guide: 873 lines covering all settings
- Security runbook: Vulnerability disclosure, incident response, secret rotation
- Data retention policy: GDPR/CCPA compliant

**Monitoring & Observability (Complete)**:
- Prometheus metrics across HTTP, database, image processing, security, and business metrics
- Grafana dashboards: Application Overview, Gallery Metrics, Security Events, Infrastructure Health
- Health check endpoints: `/health` (liveness), `/health/ready` (readiness)
- Security event alerting: 8 Grafana alert rules with response runbook
- Error tracking: Sentry + GlitchTip self-hosted option

**Deployment & Operations (Complete)**:
- Production Docker Compose with resource limits, health checks, network segmentation
- CDN configuration guide: Cloudflare, AWS CloudFront, BunnyCDN
- SSL/TLS setup: Let's Encrypt with auto-renewal
- Database backup strategy: Encrypted backups with GPG, S3 upload, rotation policy
- Backup/restore tested: RTO 18m 42s (37.7% below 30m target)

**Security (Complete)**:
- Penetration testing: A- Rating (Excellent - Launch Ready)
- Audit logging: Grade A (SOC 2, GDPR, CCPA compliant)
- Incident response plan: Tabletop exercise completed

**Testing & Compliance (Complete)**:
- Contract tests: 100% OpenAPI compliance (33 MVP endpoints)
- Test coverage: 91-100% domain, 91-94% application
- E2E coverage: 60% (62 test requests)
- Rate limiting validation: All tiers verified

### Sprint 8 Highlights (Testing & Security Hardening - Completed 2025-12-05)

**Test Coverage Excellence**:
- Gallery application layer coverage increased from 32-49% to **93-94%** (60+ percentage point improvement)
- All test coverage targets exceeded across domain, application, and infrastructure layers
- 13 new comprehensive test files with 130+ test functions
- 19 E2E tests for social features (likes, comments) with full validation

**Security Posture**: **B+ Rating**
- Zero critical or high-severity vulnerabilities
- Comprehensive security controls: ClamAV scanning, IDOR prevention, HTML sanitization, rate limiting
- CI/CD pipeline hardened: Go 1.25 pinned, Trivy/Gitleaks configured with suppressions
- Security gate approved with "excellent security posture" assessment

**Performance Optimization**:
- N+1 query elimination: 97% reduction (51 queries → 2 queries)
- Database performance indexes added (migration 00005)
- Batch loading for image variants
- Documented optimization strategies and benchmarks

**Production Readiness**:
- E2E test coverage: 60% of implemented endpoints (62 test requests)
- All Sprint 1-6 features fully tested and verified
- CI/CD pipeline stable with all security scans passing
- Ready for Sprint 9 launch preparation

## MVP Features

Based on [Flickr/Chevereto competitive analysis](claude/mvp_features.md):

### Implemented Features ✅

**User Management**
- ✅ Email/password registration with secure password policy (Argon2id)
- ✅ JWT authentication (15min access + 7-day refresh tokens)
- ✅ User profiles with image galleries
- ✅ Role-Based Access Control (Admin, Moderator, User)
- ✅ Session management with token rotation and replay detection

**Image Management**
- ✅ Image upload (single & bulk via API)
- ✅ Supported formats: JPEG, PNG, GIF, WebP
- ✅ Auto-generated variants: thumbnail (150px), small (320px), medium (800px), large (1600px), original
- ✅ ClamAV malware scanning on all uploads
- ✅ EXIF metadata extraction and optional stripping
- ✅ 7-step validation pipeline (size, MIME, dimensions, pixels, malware, EXIF, re-encode)
- ✅ Image CRUD operations (create, read, update, delete)
- ✅ Image listing with filters (owner, album, visibility, tags)
- ✅ Full-text search (title, description)

**Organization**
- ✅ Albums (single-level): create, read, update, delete
- ✅ Album image management (add/remove images)
- ✅ User-defined tags (stored in database)
- ✅ Privacy settings (public, private, unlisted) per image and album

**Social Features**
- ✅ Likes/favorites on images (with idempotency)
- ✅ Comments on images with HTML sanitization
- ✅ Like/comment counts and listings
- ✅ Public gallery/explore page (recent images, search)
- ✅ Ownership validation and IDOR prevention

**Storage Options**
- ✅ Local filesystem (development)
- ✅ S3-compatible storage (AWS S3, DigitalOcean Spaces, Backblaze B2, MinIO)
- ✅ Async background job processing (Asynq/Redis)

**API & Security**
- ✅ RESTful API with OpenAPI 3.0 spec (5,460+ lines)
- ✅ Rate limiting: 5 login/min, 100 global/min, 300 authenticated/min, 50 uploads/hour
- ✅ RFC 7807 Problem Details error responses
- ✅ Security headers middleware (CSP, HSTS, X-Frame-Options, etc.)
- ✅ CORS configuration
- ✅ Request ID correlation
- ✅ Structured logging (zerolog)

### Deferred to Phase 2 🔄

**Content Moderation** ✅ COMPLETE (Sprint 14)
- ✅ Abuse reporting system API - Sprint 14 COMPLETE
- ✅ Admin moderation queue - Sprint 14 COMPLETE
- ✅ Content flags (Safe/NSFW) API - Sprint 14 COMPLETE
- ✅ User bans (temporary & permanent) API - Sprint 14 COMPLETE

**Advanced Features**
- ✅ OAuth providers (Google, GitHub) - Sprint 12 COMPLETE
- ✅ Follow/unfollow users - Sprint 12 COMPLETE
- ✅ Email notifications (SMTP) - Sprint 12 COMPLETE
- ✅ Activity feeds - Sprint 12 COMPLETE
- ✅ IPFS decentralized storage integration - Sprint 13 COMPLETE
- ✅ MFA/TOTP support - Sprint 11 COMPLETE
- ✅ Guest uploads - Sprint 14 COMPLETE
- ✅ Content moderation suite - Sprint 14 COMPLETE
- 🔄 Advanced tag endpoints (popular tags, tag search, tag-based listing) - Future
- ✅ AI NSFW detection - Sprint 15 COMPLETE
- ✅ Advanced search filters - Sprint 15 COMPLETE
- 🔄 Watermarking - Phase 3

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.24+ (CI uses 1.24.x, stable: 1.24.11, 1.25.5 also available) |
| Database | PostgreSQL 16+ |
| Cache/Sessions | Redis 7+ |
| Migrations | Goose |
| Image Processing | bimg (libvips) |
| Security | ClamAV, JWT, OAuth2 |
| Storage | Local FS / S3 / DO Spaces / Backblaze B2 |
| Decentralized Storage | IPFS (Kubo) with remote pinning support |
| API Docs | OpenAPI 3.0 |
| Observability | zerolog, Prometheus, OpenTelemetry |

## Quick Start

### Prerequisites

```bash
go >= 1.24    # Latest stable: 1.25.5 (Go 1.26 expected Feb 2026)
docker >= 24.0
docker-compose >= 2.20
libvips >= 8.14  # Required for image processing (Sprint 5+)
```

### Setup

```bash
# Clone
git clone https://github.com/your-org/goimg-datalayer.git
cd goimg-datalayer

# Start dependencies
docker-compose -f docker/docker-compose.yml up -d

# Run migrations
make migrate-up

# Generate code from OpenAPI
make generate

# Start the API
make run
```

### Development

```bash
# Install pre-commit hooks
pre-commit install
pre-commit install --hook-type commit-msg

# Run tests
make test

# Lint
make lint

# Validate API contract
make validate-openapi
```

## Project Structure

```
goimg-datalayer/
├── api/openapi/          # OpenAPI 3.0 specification
├── cmd/
│   ├── api/              # HTTP server
│   ├── worker/           # Background jobs
│   └── migrate/          # Migration CLI
├── internal/
│   ├── domain/           # DDD domain models
│   ├── application/      # Use cases, commands, queries
│   ├── infrastructure/   # DB, storage, external services
│   └── interfaces/http/  # Handlers, middleware, DTOs
├── tests/                # Integration & E2E tests
├── docker/               # Docker configurations
└── claude/               # AI agent guides
```

## Architecture

This project follows **Domain-Driven Design (DDD)** with **Clean Architecture**:

- **Domain Layer**: Entities, Value Objects, Aggregates, Repository interfaces
- **Application Layer**: Commands, Queries, Application Services
- **Infrastructure Layer**: PostgreSQL, Redis, S3, ClamAV implementations
- **Presentation Layer**: HTTP handlers, middleware, DTOs

See [claude/architecture.md](claude/architecture.md) for detailed patterns.

## API Documentation

The OpenAPI specification is the **single source of truth** for the HTTP API:

```
api/openapi/
├── openapi.yaml      # Main spec
├── schemas/          # Reusable schemas
└── paths/            # Endpoint definitions
```

Generate server code:
```bash
make generate
```

## Testing

```bash
make test              # Full suite
make test-unit         # Unit tests
make test-integration  # Integration tests (requires DB)
make test-e2e          # End-to-end tests (Newman/Postman)
```

### Test Coverage (Last Updated: 2026-01-12)

**Overall Coverage: ~65%** (improved from 35.9% in Sprint 18)

| Layer | Package | Coverage | Sprint 18 Change |
|-------|---------|----------|------------------|
| **Domain** | activity | 88.9% | — |
| | gallery | 72.6% | — |
| | identity | **95.1%** | ↑ from 32.9% |
| | moderation | 100.0% | — |
| | shared | 78.0% | — |
| **Application** | gallery/commands | 67.7% | — |
| | gallery/queries | 75.4% | — |
| | identity/commands | 37.7% | ↑ from 34.9% |
| | identity/queries | **95.8%** | ↑ from 62.5% |
| | identity (service) | 90.9% | — |
| **Infrastructure** | jobs/asynq | 75.9% | — |
| | persistence/redis | **93.8%** | ↑ from 7.9% |
| | secrets | 68.0% | — |
| | security/jwt | **87.8%** | ↑ from 36.4% |
| | security/nsfw | **77.2%** | ↑ from 32.3% |
| | storage/ipfs | 83.5% | — |
| | storage/local | 75.9% | — |
| | storage/orchestrator | **95.1%** | ↑ from 55.6% |
| | storage/s3 | 36.8% | ↑ from 17.9% |
| | storage/validator | 96.3% | — |
| **HTTP** | middleware | **65.4%** | ↑ from 36.0% |

**Sprint 18 Summary**: 7 of 9 priority packages now exceed targets. 2 packages limited by architecture (concrete types vs interfaces).

**E2E Test Coverage**:
- 70+ total test requests across 10 feature areas
- 65% endpoint coverage (implemented features)
- Comprehensive coverage: Auth, Users, Images, Albums, Social, Explore, oEmbed, IPFS, Moderation
- Auth flow fully covered (register, login, refresh, logout, 2FA, OAuth)
- RFC 7807 error response validation

**Test Files**: 150+ comprehensive test functions across 20+ test files

## Configuration

Environment variables or config file:

### Core Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | - |
| `REDIS_URL` | Redis connection string | - |
| `JWT_SECRET` | JWT signing secret | - |
| `STORAGE_PROVIDER` | `local`, `s3`, `spaces`, `b2`, `minio` | `local` |
| `CLAMAV_HOST` | ClamAV daemon address | `localhost:3310` |

### Storage Provider Settings

| Variable | Description | Required For |
|----------|-------------|--------------|
| `LOCAL_STORAGE_PATH` | Local filesystem directory | local |
| `S3_ENDPOINT` | S3 API endpoint | s3, spaces, b2, minio |
| `S3_BUCKET` | Bucket name | s3, spaces, b2, minio |
| `S3_ACCESS_KEY` | Access key ID | s3, spaces, b2, minio |
| `S3_SECRET_KEY` | Secret access key | s3, spaces, b2, minio |
| `S3_REGION` | AWS region | s3, spaces |
| `S3_USE_PATH_STYLE` | Use path-style URLs | minio, b2 |

### IPFS Settings

IPFS can be enabled **alongside** any primary storage provider for decentralized backup and content-addressed retrieval.

| Variable | Description | Default |
|----------|-------------|---------|
| `IPFS_ENABLED` | Enable IPFS storage | `false` |
| `IPFS_API_ENDPOINT` | IPFS node HTTP API | `http://localhost:5001` |
| `IPFS_GATEWAY_ENDPOINT` | Public gateway for URLs | `https://ipfs.io` |
| `IPFS_PIN_BY_DEFAULT` | Auto-pin uploaded content | `true` |
| `IPFS_ASYNC_UPLOAD` | Non-blocking IPFS uploads | `true` |
| `IPFS_REQUIRE_PIN` | Fail upload if pinning fails | `false` |

#### Remote Pinning Services (Optional)

For production reliability, configure additional pinning services:

| Variable | Description |
|----------|-------------|
| `IPFS_PINATA_ENABLED` | Enable Pinata pinning |
| `IPFS_PINATA_JWT` | Pinata API JWT token |
| `IPFS_INFURA_ENABLED` | Enable Infura pinning |
| `IPFS_INFURA_PROJECT_ID` | Infura project ID |
| `IPFS_INFURA_PROJECT_SECRET` | Infura project secret |

See [claude/ipfs_storage.md](claude/ipfs_storage.md) for detailed IPFS integration documentation.

## Roadmap

### Phase 1: MVP (COMPLETE ✅)

| Sprint | Focus | Duration | Status |
|--------|-------|----------|--------|
| 1-2 | Foundation & Domain Layer | 4 weeks | **COMPLETE** ✅ |
| 3 | Infrastructure - Identity (DB, Redis, JWT) | 2 weeks | **COMPLETE** ✅ |
| 4 | Application & HTTP - Auth | 2 weeks | **COMPLETE** ✅ |
| 5 | Domain & Infrastructure - Gallery | 2 weeks | **COMPLETE** ✅ |
| 6 | Application & HTTP - Gallery | 2 weeks | **COMPLETE** ✅ |
| 7 | Moderation & Social Features | 2 weeks | **DEFERRED** 🔄 |
| 8 | Integration, Testing & Security Hardening | 2 weeks | **COMPLETE** ✅ |
| 9 | MVP Polish & Launch Prep | 2 weeks | **COMPLETE** ✅ GO FOR LAUNCH |

### Phase 2: Advanced Features (COMPLETE ✅)

| Sprint | Focus | Status | Key Deliverables |
|--------|-------|--------|------------------|
| **10** | Security Enhancements | ✅ **COMPLETE** | Random login delay, HIBP password check, Prometheus metrics |
| **11** | Two-Factor Authentication | ✅ **COMPLETE** | TOTP setup/verify, backup codes, rate limiting, E2E tests |
| **12** | OAuth & Social Features | ✅ **COMPLETE** | OAuth, Follow/unfollow, Activity feeds, Email notifications, Session elevation |
| **13** | IPFS Storage | ✅ **COMPLETE** | Decentralized storage, IPFS client, orchestrator, 3 API endpoints, E2E tests |
| **14** | Content Moderation + Guest Uploads | ✅ **COMPLETE** | Abuse reporting, admin queue, user bans, guest uploads, Security Gate S14 |
| **15** | AI NSFW + Advanced Search | ✅ **COMPLETE** | AI content moderation (SightEngine/ModerateContent), advanced search filters |

#### Sprint 10: Security Enhancements ✅ COMPLETE
- ✅ Random login delay (100-300ms) - Timing attack mitigation
- ✅ HIBP password check - Compromised password rejection
- ✅ Prometheus metrics for security monitoring
- ✅ Security Gate S10 passed (10/10 controls)

#### Sprint 11: Two-Factor Authentication ✅ COMPLETE
- ✅ TOTP implementation with QR code setup (RFC 6238 compliant)
- ✅ Backup codes (10 one-time codes, Argon2id hashed)
- ✅ Secrets encrypted at rest (AES-256-GCM)
- ✅ HTTP endpoints implemented (5 endpoints)
- ✅ OpenAPI spec updated
- ✅ Rate limiting (5 attempts/min) on 2FA verification
- ✅ E2E tests (13 Newman tests)
- ✅ Security Gate S11 passed (4/5 controls, 1 deferred)

#### Sprint 12: OAuth & Social Features ✅ COMPLETE
- ✅ Google OAuth integration (provider + callback)
- ✅ GitHub OAuth integration (provider + callback)
- ✅ OAuth HTTP endpoints (5 endpoints)
- ✅ OAuth E2E tests (9 Newman tests)
- ✅ Security controls (CSRF state, token encryption, callback validation)
- ✅ Follow/unfollow users (domain, application, infrastructure, HTTP layers)
- ✅ Follow/unfollow router wiring (4 endpoints)
- ✅ Follow E2E tests (18 Newman tests)
- ✅ Activity feeds (GET /api/v1/feed - timeline from followed users)
- ✅ Email notifications (SMTP with rate limiting)
- ✅ Session elevation after 2FA (S11-2FA-004 security control)
- ✅ Notification system (internal + email with preferences)

#### Sprint 13: IPFS Storage ✅ COMPLETE
- ✅ IPFS client for Kubo node integration (~400 lines)
- ✅ Storage orchestrator (dual-storage: primary + IPFS)
- ✅ Database migration for IPFS fields (CID, pin status)
- ✅ IPFSMetadata domain value object
- ✅ Three storage modes: `primary_only`, `dual_sync`, `dual_async`
- ✅ Application layer commands (`PinImageToIPFS`, `UnpinImageFromIPFS`) and queries (`GetImageIPFSStatus`)
- ✅ HTTP endpoints (3 endpoints: POST/GET/DELETE `/images/{id}/ipfs`)
- ✅ OpenAPI spec updated with IPFS schemas
- ✅ Unit tests (27 test scenarios) and E2E tests (8 Newman tests)

#### Sprint 14: Content Moderation + Guest Uploads ✅ COMPLETE
**Content Moderation** (COMPLETE ✅):
- ✅ Abuse reporting system API (`POST /api/v1/reports`) with rate limiting (10/hour)
- ✅ Admin moderation queue (`GET /api/v1/moderation/reports`, review, resolve, dismiss)
- ✅ User ban system (`POST/DELETE/GET /api/v1/users/{id}/ban`)
- ✅ List active bans (`GET /api/v1/moderation/bans`)
- ✅ Database migrations (reports, bans, reviews tables)
- ✅ E2E tests (17 Newman tests)

**Guest Uploads** (COMPLETE ✅):
- ✅ Guest session creation (`POST /api/v1/auth/guest`) with IP rate limiting (10/hour)
- ✅ Guest user type in domain layer
- ✅ Database migration for guest user support
- ✅ Upload claim to registered account (`POST /api/v1/guest/images/{id}/claim`)
- ✅ Guest E2E tests (5 Newman tests)

**Security Gate S14**: ✅ APPROVED
- Full security review: `claude/SECURITY_GATE_S14_REPORT.md`

#### Sprint 15: AI NSFW + Advanced Search ✅ COMPLETE
- ✅ AI NSFW detection (SightEngine/ModerateContent API integration)
- ✅ Advanced search filters (date range, file size, dimensions, NSFW status)
- ✅ Content flagging based on AI analysis
- ✅ Multi-provider orchestration with fallback support
- ✅ NSFW scan management endpoints (4 endpoints)
- ✅ Database migration for NSFW scan tables

### Phase 3: Advanced Features (5/7 Sprints Complete)

Phase 3 focuses on discoverability, social sharing, and platform scalability. Sprints 16-20 are complete; Sprints 21-22 remain in backlog.

| Sprint | Focus | Duration | Priority | Status |
|--------|-------|----------|----------|--------|
| **16** | oEmbed + Social Media Cards | 1 week | P1 | ✅ **COMPLETE** |
| **17** | Nested Albums + Custom Variants | 2 weeks | P2 | ✅ **COMPLETE** |
| **18** | Test Coverage Improvement | 2 weeks | P1 | ✅ **COMPLETE** |
| **19** | Trending Tags + Featured Picks | 1 week | P2 | ✅ **COMPLETE** |
| **20** | Groups/Communities | 2 weeks | P3 | ✅ **COMPLETE** (9/10 security controls) |
| **21** | Video Support | 3 weeks | P3 | 📋 Backlog |
| **22** | Account Tiers/Subscriptions | 2 weeks | P3 | 📋 Backlog |

#### Sprint 16: oEmbed + Social Media Cards ✅ COMPLETE

- ✅ oEmbed 1.0 endpoint (`GET /api/v1/oembed`) - JSON/XML support
- ✅ OpenAPI spec for oEmbed and preview endpoints
- ✅ Router wiring for oEmbed and preview handlers
- ✅ Open Graph meta tags for Facebook/LinkedIn
- ✅ Twitter Card support for rich previews
- ✅ Image preview page (`GET /images/{id}/preview`) with social meta tags
- ✅ E2E tests (8 Newman tests)

#### Sprint 17: Nested Albums + Custom Variants ✅ COMPLETE

- ✅ Nested albums with parent_id field for hierarchical organization
- ✅ Breadcrumb navigation (`GET /albums/{id}/breadcrumb`)
- ✅ Children listing (`GET /albums/{id}/children`)
- ✅ VariantConfig domain entity and PostgreSQL repository
- ✅ VariantConfig CRUD API (6 endpoints at `/variant-configs`)
- ✅ Custom variant generation (`POST /api/v1/images/{id}/variants`)
- ✅ Crop modes: fit, fill, crop
- ✅ Output formats: jpeg, png, webp, avif
- ✅ E2E tests (17 Newman tests)

#### Sprint 18: Test Coverage Improvement ✅ COMPLETE

- ✅ Overall coverage improved from 35.9% to ~65%
- ✅ 7 of 9 priority packages now exceed targets
- ✅ Key improvements: redis (93.8%), domain/identity (95.1%), jwt (87.8%)
- ✅ storage/orchestrator (95.1%), identity/queries (95.8%)
- ⚠️ 2 packages limited by architecture (concrete types vs interfaces)

#### Sprint 19: Trending Tags + Featured Picks ✅ COMPLETE

- ✅ TagRepository interface (domain layer with FindPopular, FindTrending, SearchByPrefix)
- ✅ TagRepository PostgreSQL implementation with trending algorithm
- ✅ Popular Tags API (`GET /api/v1/tags/popular`) with period filtering
- ✅ Trending Tags API (`GET /api/v1/tags/trending`) with trend scores
- ✅ Tag Search API (`GET /api/v1/tags/search`) with prefix matching
- ✅ HTTP Handlers (TagHandler at /api/v1/tags/)
- ✅ OpenAPI spec for tag endpoints
- ✅ E2E Tests for tag endpoints (6 Newman tests)
- ✅ Featured Picks database migration with scheduling support
- ✅ FeaturedPick domain entity and repository interface
- ✅ Feature/Unfeature commands and ListFeatured query
- ✅ FeaturedHandler wired in router (admin-only)
- ✅ ExploreHandler featured endpoint (public)
- ✅ OpenAPI spec for Featured Picks endpoints
- ✅ E2E Tests for Featured Picks (6 Newman tests)
- ✅ Contract tests for Featured Picks endpoints

#### Sprint 20: Groups/Communities ✅ COMPLETE

- ✅ Community bounded context (new domain layer)
- ✅ Group, GroupMembership, GroupInvitation domain entities
- ✅ Value objects: GroupID, GroupName, GroupSlug, GroupSettings, GroupRole, GroupType
- ✅ 19 domain events for group lifecycle
- ✅ Database migration (00017_create_groups.sql) with 7 tables
- ✅ PostgreSQL repositories (Group, Membership, Image, Invitation, Activity)
- ✅ Application layer (14 commands, 8 queries)
- ✅ HTTP handlers: GroupHandler (14 endpoints), GroupImageHandler (5 endpoints)
- ✅ OpenAPI spec for groups endpoints
- ✅ E2E tests (27 Newman tests planned)
- ✅ Security Gate S20: 9/10 controls passed
  - ✅ Group privacy enforcement (S20-GROUP-001)
  - ✅ RBAC for group actions (S20-GROUP-002)
  - ✅ Authorization on all endpoints (S20-GROUP-003)
  - ✅ Privilege escalation prevention (S20-GROUP-004)
  - ✅ Moderation queue security (S20-GROUP-005)
  - ✅ Invitation token security (S20-GROUP-006)
  - ✅ Rate limiting (S20-GROUP-007)
  - ✅ Audit logging (S20-GROUP-008)
  - ✅ SQL injection prevention (S20-GROUP-009)
  - ⚠️ XSS prevention (S20-GROUP-010) - partial (frontend responsibility)
- ✅ Group Albums implementation (7 endpoints: CRUD + image management)
- ✅ E2E tests (34 Newman tests, 7 for group albums)
- ✅ Contract tests validation (OpenAPI: 85 paths, 63 schemas)
- ✅ Performance testing (`make load-test-groups`, target < 200ms p95)

**Future Sprints** (Backlog):
- **Video Support** (Sprint 21) - Video uploads with HLS/DASH streaming
- **Account Tiers** (Sprint 22) - Free, Pro, Business levels with different limits

See [claude/phase_3_sprint_plan.md](claude/phase_3_sprint_plan.md) for detailed Phase 3 planning.

## Contributing

1. Follow the coding standards in [claude/coding.md](claude/coding.md)
2. Ensure tests pass: `make test`
3. Run linter: `make lint`
4. Validate API changes: `make validate-openapi`
5. See [claude/agent_checklist.md](claude/agent_checklist.md) before submitting

## AI Agent Guides

This repository includes structured guides for AI coding assistants in the `claude/` directory:

| Guide | Purpose |
|-------|---------|
| [CLAUDE.md](CLAUDE.md) | Entry point and navigation |
| [sprint_plan.md](claude/sprint_plan.md) | **Development roadmap (8-9 sprints)** |
| [mvp_features.md](claude/mvp_features.md) | **Feature specifications and API design** |
| [agent_workflow.md](claude/agent_workflow.md) | **Multi-agent coordination and task assignments** |
| [architecture.md](claude/architecture.md) | DDD patterns and structure |
| [coding.md](claude/coding.md) | Go standards and tooling |
| [api_security.md](claude/api_security.md) | HTTP and security |
| [security_gates.md](claude/security_gates.md) | **Sprint security reviews and gate approvals** |
| [security_testing.md](claude/security_testing.md) | **Security test requirements and tools** |
| [test_strategy.md](claude/test_strategy.md) | **Comprehensive test design patterns** |
| [testing_ci.md](claude/testing_ci.md) | Testing and CI/CD quick reference |
| [ipfs_storage.md](claude/ipfs_storage.md) | IPFS integration and P2P storage |
| [notifications.md](claude/notifications.md) | Email and notification system |
| [agent_checklist.md](claude/agent_checklist.md) | Pre-commit checklist |
| [placement.md](claude/placement.md) | Adding folder-local guides |

## License

[MIT](LICENSE)
