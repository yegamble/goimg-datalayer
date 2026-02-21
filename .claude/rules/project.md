# Project: goimg-datalayer

**Last Updated:** 2026-02-21

## Overview

Go backend for a Flickr/Chevereto-style image gallery platform. Handles image upload, processing, moderation, user management, social features, and decentralized storage (IPFS).

**Phase:** Phase 3 - Advanced Features | **Sprint:** 24 (Audit & Fixes)
**Test Coverage:** ~65% overall (target: 80%) | **Domain layer:** 90% target

## Technology Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.25+ (toolchain go1.25.5 pinned) |
| Router | go-chi/chi v5 |
| Database | PostgreSQL 16 + sqlx/lib pq |
| Cache/Sessions | Redis 7 + go-redis/v9 |
| Migrations | pressly/goose v3 |
| Auth | golang-jwt/jwt v5, OAuth2 |
| Image Processing | bimg (libvips) |
| Job Queue | hibiken/asynq |
| Security | ClamAV, HIBP password checking, NSFW scanning |
| Object Storage | Local / S3 / DO Spaces / B2 / IPFS (Kubo + Pinata/Infura) |
| API Spec | OpenAPI 3.0.3 (oapi-codegen generated types) |
| Observability | rs/zerolog, Prometheus, OpenTelemetry |
| Testing | testify, testcontainers-go (Postgres + Redis), Newman/Postman |
| 2FA | pquerna/otp (TOTP) + boombuler/barcode (QR codes) |

## Architecture: DDD with CQRS-Lite

**4 layers — strict import rules:**

```
domain/          ← No external deps. Entities, value objects, repo interfaces.
application/     ← Imports domain only. Commands (writes) + Queries (reads).
infrastructure/  ← Implements domain interfaces. DB, Redis, S3, IPFS, security.
interfaces/http/ ← Chi handlers, middleware, DTOs. No business logic.
```

## Directory Structure

```
cmd/
├── api/         # API server main
├── worker/      # Background job worker (asynq)
└── migrate/     # DB migration runner
internal/
├── domain/      # gallery, identity, moderation, community, activity, notification, shared
├── application/ # commands/, queries/, services/, dto/ per bounded context
├── infrastructure/
│   ├── persistence/postgres/   # sqlx repositories
│   ├── persistence/redis/      # Redis session/cache
│   ├── storage/                # local, s3, ipfs, orchestrator, processor, validator
│   ├── security/               # jwt/, clamav/, nsfw/
│   ├── email/                  # SMTP notifications
│   └── jobs/                   # asynq workers
└── interfaces/http/
    ├── handlers/    # One handler file per domain area
    ├── middleware/  # auth, rate_limit, cors, security_headers, logging, recovery
    ├── dto/         # Request/response structs
    └── generated/   # oapi-codegen output
api/openapi/openapi.yaml  # Source of truth for all HTTP endpoints
migrations/               # Goose SQL migrations (numbered sequential)
tests/
├── unit/        # Unit tests (mocks, no I/O)
├── integration/ # testcontainers-go (real Postgres + Redis)
├── e2e/postman/ # Newman/Postman collection (290 tests, 20 categories)
└── security/    # Security-specific tests
```

## Key Commands

```bash
# Setup
make install-hooks        # MANDATORY: install git pre-commit hooks
docker compose -f docker/docker-compose.yml up -d
make migrate-up           # Apply migrations
make generate             # Run oapi-codegen

# Run
make run                  # API server
make run-worker           # Background worker

# Validate (MANDATORY before push)
make agent-check          # Full pre-push check (Go changes)
make agent-check-quick    # Quick check (non-Go changes only)
make pre-commit           # go fmt + go vet + golangci-lint

# Test
make test                 # All tests with race detector
make test-unit            # Unit tests only
make test-integration     # Integration tests (requires Docker)
make test-domain          # Domain tests (90% threshold)
make test-e2e             # Newman/Postman smoke E2E (requires API server)
make test-e2e-full        # Full E2E suite (290 tests)
make test-e2e-folder FOLDER=Auth  # Specific category

# Other
make validate-openapi     # Validate OpenAPI spec
make migrate-create NAME=migration_name
make lint                 # golangci-lint
```

## Error Handling Conventions

- **All HTTP errors:** RFC 7807 Problem Details via `middleware.WriteError(w, r, status, title, detail)`
- **Error wrapping:** Always `fmt.Errorf("context: %w", err)`
- **Domain errors:** Sentinel vars in `errors.go` per bounded context
- **Handler errors:** Map domain errors to HTTP status via switch/errors.Is

## Critical Rules (Beyond CLAUDE.md)

- **OpenAPI is truth:** All HTTP changes MUST match `api/openapi/openapi.yaml`
- **Domain purity:** `internal/domain/` never imports infrastructure packages
- **No logic in handlers:** Handlers: parse → validate → delegate → respond
- **Goose migrations:** Sequential numbered SQL files; never edit existing
- **Rate limiting before auth** in middleware stack (prevents brute-force)
- **E2E mandatory:** Every new API feature needs Postman tests

## Open Critical Issues (as of 2026-02-06)

1. Event publishing errors silently ignored (HIGH) — `application/gallery/commands/upload_image.go:170`
2. Cache stampede on HIBP cache miss (HIGH) — `infrastructure/security/password_cache.go:36`
3. Activity feed unbounded LATERAL JOIN (HIGH) — `infrastructure/persistence/postgres/activity_repository.go:38`
