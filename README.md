# goimg-datalayer

Go backend for an image gallery web application (Flickr/Chevereto-style). Provides secure image hosting, user management, and content moderation.

## Features

- Image upload, processing, and multi-provider storage
- **IPFS support** for decentralized, content-addressed storage
- Dual-storage mode: primary storage (S3/Local) + IPFS pinning
- User authentication (JWT + OAuth2)
- Role-Based Access Control (Admin, Moderator, User)
- Content moderation workflows
- Rate limiting and malware scanning (ClamAV)
- OpenAPI 3.1 specification

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.22+ |
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
go >= 1.22
docker >= 24.0
docker-compose >= 2.20
libvips >= 8.14
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
| [architecture.md](claude/architecture.md) | DDD patterns and structure |
| [coding.md](claude/coding.md) | Go standards and tooling |
| [api_security.md](claude/api_security.md) | HTTP and security |
| [testing_ci.md](claude/testing_ci.md) | Testing and CI/CD |
| [ipfs_storage.md](claude/ipfs_storage.md) | IPFS integration and P2P storage |
| [agent_checklist.md](claude/agent_checklist.md) | Pre-commit checklist |

## License

[MIT](LICENSE)
