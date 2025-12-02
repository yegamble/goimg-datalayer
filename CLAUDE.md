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
| Language | Go 1.22+ |
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
| Architecture & DDD | `claude/architecture.md` | Domain modeling, bounded contexts |
| Coding standards | `claude/coding.md` | Writing/reviewing Go code |
| API & security | `claude/api_security.md` | HTTP handlers, auth, endpoints |
| Testing & CI | `claude/testing_ci.md` | Writing tests, CI issues |
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
```

See `claude/agent_checklist.md` for the full verification checklist.
