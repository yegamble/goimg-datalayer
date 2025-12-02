# Claude Agent Guide

> **goimg-datalayer**: Go backend for an image gallery (Flickr/Chevereto-style). Upload, moderation, user management.

This repository uses a foldered guide so Claude agents can stay within scope and keep context size low. **Load only what you need** for the area you are working in.

## Quick Start

```bash
# Setup
docker-compose -f docker/docker-compose.yml up -d
make migrate-up
make generate

# Run
make run              # API server
make run-worker       # Background jobs

# Validate before commit
make lint && make test && make validate-openapi
```

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.23+ |
| Database | PostgreSQL 16+, Redis 7+ |
| Migrations | Goose |
| Image Processing | bimg (libvips) |
| Security | ClamAV, JWT, OAuth2 |
| Object Storage | Local/S3/DO Spaces/B2 |
| Decentralized Storage | IPFS (Kubo) + Pinata/Infura |
| API Spec | OpenAPI 3.1 |
| Observability | zerolog, Prometheus, OpenTelemetry |

## Navigation

| Topic | File | When to Load |
| --- | --- | --- |
| **Sprint Plan** | `claude/sprint_plan.md` | Planning work, understanding roadmap |
| **MVP Features** | `claude/mvp_features.md` | Feature requirements, API specs |
| **Agent Workflow** | `claude/agent_workflow.md` | Multi-agent coordination, task assignments, quality gates |
| Architecture & DDD | `claude/architecture.md` | Domain modeling, bounded contexts |
| Coding standards | `claude/coding.md` | Writing/reviewing Go code |
| API & security | `claude/api_security.md` | HTTP handlers, auth, endpoints |
| **Security Gates** | `claude/security_gates.md` | Sprint security reviews, gate approvals |
| **Security Testing** | `claude/security_testing.md` | Security test requirements, tools, SAST/DAST |
| **Test Strategy** | `claude/test_strategy.md` | Designing test suites, comprehensive patterns |
| Testing & CI quick ref | `claude/testing_ci.md` | Quick test patterns, CI troubleshooting |
| IPFS & P2P storage | `claude/ipfs_storage.md` | Implementing IPFS, pinning, decentralized storage |
| Notifications & email | `claude/notifications.md` | User follows, email (SMTP), notification preferences |
| Agent checklist | `claude/agent_checklist.md` | Before committing changes |
| Scoped guide placement | `claude/placement.md` | Adding folder-local guides |

**Scoped guides**: Check for `CLAUDE.md` files in the directory you're working in. They contain context-specific rules.

## Core Rules

1. **DDD layering**: Domain logic must not import infrastructure packages
2. **OpenAPI is truth**: All HTTP changes must match `api/openapi/` spec
3. **No business logic in handlers**: Handlers delegate to application layer
4. **Wrap errors**: Always use `fmt.Errorf("context: %w", err)`
5. **Test coverage**: Minimum 80% overall; 90% for domain layer
6. **E2E tests required**: Every new API feature MUST have Newman/Postman E2E tests for regression testing

## E2E Testing Requirements

**Newman/Postman is mandatory** for all API endpoints:

- **Location**: `tests/e2e/postman/goimg-api.postman_collection.json`
- **Environment**: `tests/e2e/postman/ci.postman_environment.json`
- **CI Integration**: E2E tests run automatically in GitHub Actions after build

### When Adding New Features

1. Add Postman requests for all new endpoints in the collection
2. Include test scripts that validate:
   - Response status codes
   - Response body structure (JSON schema)
   - Business logic (e.g., created resources exist)
   - Error handling (4xx/5xx responses follow RFC 7807)
3. Update CI environment variables if needed
4. Run `make test-e2e` locally before committing

### E2E Test Categories

| Category | Purpose |
|----------|---------|
| Happy path | Verify normal operation |
| Error handling | Verify RFC 7807 error responses |
| Authentication | Verify auth flows and token handling |
| Authorization | Verify RBAC and ownership checks |
| Regression | Catch breaking changes |

## Project Structure (Key Paths)

```
internal/
├── domain/           # Entities, value objects, aggregates, repo interfaces
├── application/      # Commands, queries, application services
├── infrastructure/
│   ├── persistence/  # Postgres, Redis repositories
│   ├── storage/      # Local, S3, IPFS providers (see CLAUDE.md inside)
│   ├── security/     # JWT, OAuth, ClamAV
│   └── messaging/    # Event publishing
└── interfaces/http/  # Handlers, middleware, DTOs
api/openapi/          # OpenAPI 3.1 spec (source of truth)
tests/                # Unit, integration, e2e, contract tests
docker/               # Docker Compose with IPFS, Postgres, Redis, MinIO
```

## Before Every Commit

```bash
go fmt ./... && go vet ./... && golangci-lint run
go test -race ./...
make validate-openapi
make test-e2e  # Run Newman E2E tests (requires API server running)
```

See `claude/agent_checklist.md` for the full verification checklist.
