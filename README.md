# goimg-datalayer

Go backend for an image gallery web application (Flickr/Chevereto-style). Provides secure image hosting, user management, and content moderation.

## Status

**Current Phase**: Sprint 9 - MVP Polish & Launch Prep (IN PROGRESS - Started 2025-12-05)

**Completed Sprints**:

- **Sprint 1-2: Foundation & Domain Layer** (4 weeks) ✅
  - Project setup with DDD architecture
  - OpenAPI 3.1 specification (2,341 lines)
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
  - **E2E Tests**: 60% endpoint coverage, 38 total test requests, 19 social features tests
  - **CI/CD Hardening**: Go 1.25 pinned, Trivy exit codes fixed, Gitleaks v8.23.0 pinned
  - **Performance Optimization**: N+1 query elimination (97% reduction), database indexes (migration 00005)
  - **Security Configurations**: .gitleaks.toml, .trivyignore
  - **Test Files Added**: 13 new test files, 130+ comprehensive test functions

**Current Sprint Focus (Sprint 9 - In Progress)**:
- MVP Polish & Launch Prep
  - Production monitoring and observability (Prometheus, Grafana)
  - Deployment documentation and runbooks
  - Load testing and performance benchmarks
  - Contract testing (OpenAPI compliance validation)
  - Launch readiness checklist

See [claude/sprint_plan.md](claude/sprint_plan.md) for the complete roadmap.

## Recent Achievements

### Sprint 8 Highlights (Testing & Security Hardening)

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
- E2E test coverage: 60% of implemented endpoints (38 test requests)
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
- ✅ RESTful API with OpenAPI 3.1 spec (2,341 lines)
- ✅ Rate limiting: 5 login/min, 100 global/min, 300 authenticated/min, 50 uploads/hour
- ✅ RFC 7807 Problem Details error responses
- ✅ Security headers middleware (CSP, HSTS, X-Frame-Options, etc.)
- ✅ CORS configuration
- ✅ Request ID correlation
- ✅ Structured logging (zerolog)

### Deferred to Phase 2 🔄

**Content Moderation** (Basic moderation available via database access)
- 🔄 Abuse reporting system API
- 🔄 Admin moderation queue UI
- 🔄 Content flags (Safe/NSFW) API
- 🔄 User bans (temporary & permanent) API

**Advanced Features**
- 🔄 OAuth providers (Google, GitHub)
- 🔄 Email verification and notifications (SMTP)
- 🔄 Follow/unfollow users
- 🔄 Activity feeds
- 🔄 IPFS decentralized storage integration
- 🔄 Advanced tag endpoints (popular tags, tag search, tag-based listing)
- 🔄 MFA/TOTP support
- 🔄 Guest uploads
- 🔄 Watermarking

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.25+ |
| Database | PostgreSQL 16+ |
| Cache/Sessions | Redis 7+ |
| Migrations | Goose |
| Image Processing | bimg (libvips) |
| Security | ClamAV, JWT, OAuth2 |
| Storage | Local FS / S3 / DO Spaces / Backblaze B2 |
| Decentralized Storage | IPFS (Kubo) with remote pinning support |
| API Docs | OpenAPI 3.1 |
| Observability | zerolog, Prometheus, OpenTelemetry |

## Quick Start

### Prerequisites

```bash
go >= 1.25
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
├── api/openapi/          # OpenAPI 3.1 specification
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

### Test Coverage Achievements (Sprint 8)

**Achieved Coverage** (exceeded all targets):

| Layer | Target | Actual | Status |
|-------|--------|--------|--------|
| Domain | 90% | **91-100%** | ✅ **EXCEEDED** |
| Application - Gallery Commands | 85% | **93.4%** | ✅ **EXCEEDED** |
| Application - Gallery Queries | 85% | **94.2%** | ✅ **EXCEEDED** |
| Application - Identity | 85% | **91-93%** | ✅ **EXCEEDED** |
| Infrastructure - Storage | 70% | **78-97%** | ✅ **EXCEEDED** |
| Overall Project | 80% | In Progress | 🔄 Sprint 9 |

**E2E Test Coverage**:
- 38 total test requests across 9 feature areas
- 60% endpoint coverage (implemented features)
- 19 comprehensive social features tests (likes, comments)
- Auth flow fully covered (register, login, refresh, logout)
- RFC 7807 error response validation

**Test Files**: 130+ comprehensive test functions across 13 test files added in Sprint 8

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

| Sprint | Focus | Duration | Status |
|--------|-------|----------|--------|
| 1-2 | Foundation & Domain Layer | 4 weeks | **COMPLETE** ✅ |
| 3 | Infrastructure - Identity (DB, Redis, JWT) | 2 weeks | **COMPLETE** ✅ |
| 4 | Application & HTTP - Auth | 2 weeks | **COMPLETE** ✅ |
| 5 | Domain & Infrastructure - Gallery | 2 weeks | **COMPLETE** ✅ |
| 6 | Application & HTTP - Gallery | 2 weeks | **COMPLETE** ✅ |
| 7 | Moderation & Social Features | 2 weeks | **DEFERRED** 🔄 |
| 8 | Integration, Testing & Security Hardening | 2 weeks | **COMPLETE** ✅ |
| 9 | MVP Polish & Launch Prep | 2 weeks | **IN PROGRESS** 🚀 (Started: 2025-12-05) |

**Sprint 7 Note**: Core social features (likes, comments) were completed in Sprint 6. Advanced moderation features (abuse reporting API, moderation queue, user ban API) deferred to Phase 2. Basic moderation available via direct database access.

**Phase 2** (post-MVP):
- Advanced moderation (reporting, admin queue, ban system)
- OAuth providers (Google, GitHub)
- User follows and activity feeds
- Email notifications (SMTP)
- IPFS decentralized storage
- Advanced tag features
- MFA/TOTP support

See [sprint_plan.md](claude/sprint_plan.md) for detailed breakdown.

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
| [architecture.md](claude/architecture.md) | DDD patterns and structure |
| [coding.md](claude/coding.md) | Go standards and tooling |
| [api_security.md](claude/api_security.md) | HTTP and security |
| [testing_ci.md](claude/testing_ci.md) | Testing and CI/CD |
| [ipfs_storage.md](claude/ipfs_storage.md) | IPFS integration and P2P storage |
| [notifications.md](claude/notifications.md) | Email and notification system |
| [agent_checklist.md](claude/agent_checklist.md) | Pre-commit checklist |

## License

[MIT](LICENSE)
