# Agent Verification Checklist

Use this checklist before submitting changes. Load only this file when you need the checklist to keep context small.

## Before Writing Code
- Review the relevant bounded context under `internal/domain/` and its `errors.go`.
- Confirm the OpenAPI contract in `api/openapi/` for any HTTP surface changes.
- Identify DDD patterns involved (entity, value object, aggregate, domain service).
- Skim existing tests for patterns you should follow.

## During Implementation
- Keep domain code free from infrastructure imports.
- Validate invariants in factories/constructors; value objects must be immutable.
- Route all aggregate mutations through the root and collect/publish domain events.
- HTTP handlers should delegate to application commands/queries; no business logic inline.
- Wrap errors with context and prefer typed/domain errors.

## Before Committing
- `go fmt ./...`, `go vet ./...`, `golangci-lint run`.
- `go test -race ./...` or targeted suites as appropriate.
- `make validate-openapi` plus `make generate` with a clean `git diff`.
- Ensure coverage ≥ 80% overall for new/changed code.
- No hardcoded secrets; remove dead or commented-out code.

## Security & Contract
- Run `gosec ./...` and `trivy fs .` when touching security or dependencies.
- Confirm JWT/RBAC changes align with middleware and permission maps.
- Keep Problem Details responses consistent and avoid leaking raw database errors.
