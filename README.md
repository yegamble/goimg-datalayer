# goimg-datalayer

Go backend for an image gallery web application (Flickr/Chevereto-style). Provides secure image hosting, user management, and content moderation.

## Status

**Current Phase**: **Phase 3 - Advanced Features** | **Sprint 23 IN PROGRESS** 🚧

| Phase | Status | Sprints | Highlights |
|-------|--------|---------|------------|
| Phase 1 (MVP) | ✅ Complete | 1-9 | Core gallery, auth, storage, security |
| Phase 2 (Advanced) | ✅ Complete | 10-15 | 2FA, OAuth, IPFS, moderation, AI NSFW |
| Phase 3 (In Progress) | 🚧 Active | 16-23 | Social cards, groups, test coverage |

**Sprint 23 (Current)**: Test Coverage & Regression Prevention
- Target: Improve coverage from ~65% to 80%+
- See [claude/sprint_23_test_coverage_plan.md](claude/sprint_23_test_coverage_plan.md)

**Latest Audit**: [2026-02-03](claude/audit_report_2026-02-03.md) - 5 critical issues identified

For detailed sprint history, see [claude/sprint_plan.md](claude/sprint_plan.md) and [claude/NEXT_STEPS.md](claude/NEXT_STEPS.md).

## Quick Start

### Prerequisites

- **Go 1.25+** (required - toolchain go1.25.5 pinned)
- Docker and Docker Compose
- libvips (for image processing): `apt-get install libvips-dev` or `brew install vips`

### Setup

```bash
# Clone and setup
git clone <repo>
cd goimg-datalayer
make install-hooks    # Install pre-commit hooks (MANDATORY)

# Start dependencies
docker-compose -f docker/docker-compose.yml up -d

# Run migrations
make migrate-up

# Start the server
make run              # API server on :8080
make run-worker       # Background jobs (separate terminal)
```

### Development Commands

```bash
# Run before every commit
make pre-commit       # Format, vet, lint

# Testing
make test             # All tests
make test-unit        # Unit tests only
make test-integration # Integration tests
make test-e2e         # E2E tests (Newman)

# API validation
make validate-openapi # Validate OpenAPI spec
```

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.25+ (toolchain go1.25.5 pinned) |
| Database | PostgreSQL 16+, Redis 7+ |
| Migrations | Goose |
| Image Processing | bimg (libvips) |
| Security | ClamAV, JWT (RS256), OAuth2, 2FA (TOTP) |
| Object Storage | Local/S3/DO Spaces/B2 + IPFS |
| API Spec | OpenAPI 3.0.3 |
| Observability | zerolog, Prometheus, OpenTelemetry |

## Project Structure

```
internal/
├── domain/           # DDD domain models, value objects, repository interfaces
├── application/      # Use cases, commands, queries
├── infrastructure/   # DB, storage, external services
└── interfaces/http/  # Handlers, middleware, DTOs
api/openapi/          # OpenAPI 3.0 spec (source of truth)
tests/                # Integration & E2E tests
docker/               # Docker Compose configurations
claude/               # AI agent guides and documentation
```

## Key Features

**User Management**
- Email/password and OAuth (Google, GitHub) authentication
- Two-factor authentication (TOTP) with backup codes
- JWT tokens with refresh rotation
- Role-based access control (Admin, Moderator, User)

**Image Management**
- Multi-format support: JPEG, PNG, GIF, WebP
- Auto-generated variants (thumbnail, small, medium, large)
- ClamAV malware scanning, EXIF extraction
- IPFS decentralized storage option

**Organization**
- Albums with nested hierarchy
- User-defined tags
- Privacy settings (public, private, unlisted)
- Groups/Communities with shared albums

**Social & Discovery**
- Likes, comments, follows, activity feeds
- Trending tags, featured picks
- oEmbed, Open Graph, Twitter Cards

**Content Moderation**
- AI NSFW detection (SightEngine/ModerateContent)
- Abuse reporting, admin queue, user bans
- Guest uploads with claim-to-account

## Configuration

Essential environment variables:

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_URL` | Redis connection string |
| `JWT_PRIVATE_KEY_PATH` | Path to RS256 private key |
| `STORAGE_PROVIDER` | `local`, `s3`, `spaces`, `b2`, `minio` |

See [docs/deployment/environment_variables.md](docs/deployment/environment_variables.md) for complete configuration.

## Documentation

| Topic | File |
|-------|------|
| Current Status | [claude/NEXT_STEPS.md](claude/NEXT_STEPS.md) |
| Sprint Roadmap | [claude/sprint_plan.md](claude/sprint_plan.md) |
| API Reference | [docs/api/README.md](docs/api/README.md) |
| Deployment | [docs/deployment/README.md](docs/deployment/README.md) |
| Architecture | [claude/architecture.md](claude/architecture.md) |
| Security | [SECURITY.md](SECURITY.md) |

## Test Coverage

**Current**: ~65% overall (Sprint 23 targeting 80%+)

| Layer | Coverage | Target |
|-------|----------|--------|
| Domain | 90%+ | 90% |
| Application | ~40% | 85% |
| Infrastructure | ~30% | 70% |

See [claude/test_strategy.md](claude/test_strategy.md) for testing patterns.

## E2E Testing

**290 Postman E2E tests** across 20 categories provide comprehensive API validation.

**Coverage**: ~100% endpoint coverage with tests for happy paths, error handling, authentication, authorization, and regression detection.

### Running E2E Tests

```bash
# Setup (one-time, creates test environment)
make setup-e2e

# Run all E2E tests
make test-e2e              # Full suite (290 tests)

# Run specific category
make test-e2e-folder FOLDER=Auth
make test-e2e-folder FOLDER=Images
make test-e2e-folder FOLDER=Albums

# Dry run (validate collection structure)
make test-e2e-dry

# Generate detailed HTML report
make test-e2e-report

# Run full CI pipeline locally
make ci-local              # Lint, test, E2E, validate OpenAPI
```

**Docker Compose Compatibility**: E2E tests support both standalone (`docker-compose`) and plugin (`docker compose`) commands.

**Location**: `tests/e2e/postman/goimg-api.postman_collection.json`

See [tests/e2e/README.md](tests/e2e/README.md) for collection structure and test categories.

## Contributing

1. Follow [claude/coding.md](claude/coding.md) for coding standards
2. Run `make pre-commit` before committing
3. Ensure tests pass: `make test`
4. Validate API changes: `make validate-openapi`
5. See [claude/agent_checklist.md](claude/agent_checklist.md) for full checklist

## License

[MIT](LICENSE)
