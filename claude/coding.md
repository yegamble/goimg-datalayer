# Coding Standards & Tooling

## Go Style
- Follow Effective Go and Go Code Review Comments.
- Prefer descriptive names; keep functions focused (< ~50 lines when practical).
- Never place business logic in HTTP handlers; delegate to application/domain layers.
- Always wrap errors with context: `fmt.Errorf("context: %w", err)`.
- Document exported types and functions; avoid commented-out code.

## Domain Rules
- Domain layer must not import infrastructure packages.
- Entities are created via factory functions that enforce invariants.
- Value objects are immutable and validated on construction.
- Modify aggregates only through root methods; collect domain events for publishing.

## Error & Validation Patterns
- Use domain-specific errors (e.g., `ErrUserNotFound`, `ErrInvalidCredentials`).
- Validation happens at the boundary (DTOs) and in domain constructors for invariants.

## Linting & Formatting
- Run `golangci-lint` with the configured rules (errcheck, govet, staticcheck, gofmt/goimports, security linters, duplication/complexity guards, etc.).
- Keep imports ordered with `goimports` (local prefix `github.com/your-org/goimg-datalayer`).
- Prefer removing duplication or extracting constants when flagged by `gocognit`, `gocyclo`, or `goconst`.

## Pre-commit Expectations
- Install hooks: `pre-commit install` and `pre-commit install --hook-type commit-msg`.
- Hooks include formatters (`go-fmt`, `go-imports`), vet/build, `golangci-lint`, OpenAPI linting, generated-code checks, and unit tests on push.

## Make Targets (common)
- `make build` – build API and worker binaries.
- `make run` / `make run-worker` – start services locally.
- `make lint` – run golangci-lint.
- `make test`, `make test-unit`, `make test-integration`, `make test-e2e` – run suites.
- `make generate` – regenerate code from OpenAPI; ensure `git diff` is clean afterwards.
- `make migrate-up` / `make migrate-down` – run Goose migrations.
