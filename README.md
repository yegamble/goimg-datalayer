# goimg-datalayer

Go backend for an image gallery web application (Flickr/Chevereto-style). Provides secure image hosting, user management, and content moderation.

## Status

**Current Phase**: Sprint 4 - Application & HTTP Layer (Identity Context)

**Completed**:
- Sprint 1-2: Foundation & Domain Layer (4 weeks)
  - Project setup with DDD architecture
  - OpenAPI 3.1 specification (2,300+ lines)
  - Complete domain layer (Identity, Gallery, Moderation, Shared contexts)
  - Domain layer test coverage: 95% (exceeds 90% requirement)
  - CI/CD pipeline (GitHub Actions with linting, testing, security scanning)
  - Newman/Postman E2E test infrastructure

- Sprint 3: Infrastructure - Identity Context (2 weeks)
  - Database migrations (users, sessions tables)
  - PostgreSQL connection pool and repositories (UserRepository, SessionRepository)
  - Redis client and session store
  - JWT service with RS256 signing (4096-bit keys enforced)
  - Refresh token rotation with replay detection
  - Token blacklist in Redis
  - Integration tests with testcontainers (PostgreSQL, Redis)

**In Progress**:
- Sprint 4: Application & HTTP - Identity Context (Auth use cases, HTTP handlers, middleware)

See [claude/sprint_plan.md](claude/sprint_plan.md) for the complete 8-9 sprint roadmap.

## MVP Features

Based on [Flickr/Chevereto competitive analysis](claude/mvp_features.md):

**User Management**
- Email/password registration with secure password policy (Argon2id)
- JWT authentication (15min access + 7-day refresh tokens)
- User profiles with image galleries
- Role-Based Access Control (Admin, Moderator, User)

**Image Management**
- Drag-drop upload (single & bulk)
- Supported formats: JPEG, PNG, GIF, WebP
- Auto-generated variants: thumbnail, small, medium, large, original
- ClamAV malware scanning on all uploads
- EXIF metadata extraction and optional stripping

**Organization**
- Albums (single-level)
- User-defined tags
- Basic search (tags, titles)
- Privacy settings (public, private, unlisted)

**Social Features**
- Likes/favorites on images
- Comments
- Public gallery/explore page

**Content Moderation**
- Abuse reporting system
- Admin moderation queue
- Content flags (Safe/NSFW)
- User bans (temporary & permanent)

**Storage Options**
- Local filesystem (development)
- S3-compatible (AWS, DigitalOcean Spaces, Backblaze B2)
- **IPFS support** (Phase 2) for decentralized, content-addressed storage

**API**
- RESTful API with OpenAPI 3.1 spec
- Rate limiting (100 req/min global, 300 authenticated)
- RFC 7807 Problem Details error responses

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.24+ |
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
go >= 1.24
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
make test-e2e          # End-to-end tests
```

Coverage targets:
- Overall: 80%
- Domain: 90%
- Application: 85%
- Infrastructure: 70%

## Configuration

Environment variables or config file:

### Core Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | - |
| `REDIS_URL` | Redis connection string | - |
| `JWT_SECRET` | JWT signing secret | - |
| `STORAGE_PROVIDER` | `local`, `s3`, `spaces`, `b2` | `local` |
| `CLAMAV_HOST` | ClamAV daemon address | `localhost:3310` |

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
| 1-2 | Foundation & Domain Layer | 4 weeks | **COMPLETE** |
| 3 | Infrastructure - Identity (DB, Redis, JWT) | 2 weeks | **COMPLETE** |
| 4 | Application & HTTP - Auth | 2 weeks | In Progress |
| 5 | Domain & Infrastructure - Gallery | 2 weeks | Planned |
| 6 | Application & HTTP - Gallery | 2 weeks | Planned |
| 7 | Moderation & Social Features | 2 weeks | Planned |
| 8 | Integration, Testing & Security | 2 weeks | Planned |
| 9 | MVP Polish & Launch | 2 weeks | Planned |

**Phase 2** (post-MVP): OAuth providers, user follows, activity feeds, email notifications, IPFS storage

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
