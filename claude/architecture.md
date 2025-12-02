# Architecture & Domain Model

## Project Overview
- Backend for an image gallery with upload, moderation, and user management.
- Go 1.22+, PostgreSQL 16+, Redis 7+, bimg/libvips, Goose migrations.

## Clean Architecture & DDD
```
Presentation (HTTP handlers, middleware, DTOs)
Application (commands, queries, application services)
Domain (entities, value objects, aggregates, domain services/events)
Infrastructure (persistence, storage, external services)
```

### Bounded Contexts
- **Identity & Access**: user, role, permission, session, OAuth.
- **Gallery**: image, album, tag, comment, visibility.
- **Moderation**: report, review, ban, appeal.
- **Shared Kernel**: pagination, timestamps, shared value objects.

### Aggregate Guidance
- Validate invariants in constructors/factories.
- Aggregates own their invariants and collect domain events; expose getters and methods only through the root.
- Repository interfaces live in the domain layer; implementations live in infrastructure.

### Project Structure (abridged)
```
goimg-datalayer/
├── api/openapi/           # OpenAPI 3.1 spec & schemas
├── cmd/                   # Entrypoints: api, worker, migrate
├── internal/
│   ├── domain/            # DDD domain models and interfaces
│   ├── application/       # Commands, queries, services
│   ├── infrastructure/    # Postgres, Redis, storage, security
│   └── interfaces/http/   # HTTP server, routes, middleware, DTOs
├── tests/                 # unit, integration, e2e, contract tests
├── scripts/, docker/, pkg/, Makefile
└── claude/                # Topic-specific guides for agents
```
