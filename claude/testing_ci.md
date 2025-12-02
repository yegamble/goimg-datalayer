# Testing Strategy & CI/CD

## Test Pyramid
- Unit (~60–70%): domain logic, value objects, handlers without external deps.
- Integration (~20–25%): repositories, external services via Testcontainers/real services.
- E2E (~10–15%): Newman/Postman against a running API.

## Commands
- `make test` – full suite.
- `make test-unit` – race-enabled short tests.
- `make test-integration` – integration suite (requires Postgres/Redis, migrations).
- `make test-e2e` – Newman collection after starting services.
- `make validate-openapi` – lint spec and regenerate code; diff must stay clean.

## Coverage
- Minimum overall coverage 80%; domain 90%, application 85%, infrastructure 70%, handlers 75%.

## CI/CD Expectations
- GitHub Actions run lint, unit, integration, contract validation, e2e (where applicable), and coverage checks.
- Security workflows include `gosec` and `trivy` scans; dependency review on PRs.
- Ensure migrations, OpenAPI updates, and Postman collections accompany breaking changes.

## Observability (runtime checks)
- Use zerolog for structured logs; include request IDs and operation names.
- Expose Prometheus metrics for HTTP, uploads, storage, and moderation queues.
- Emit OpenTelemetry traces with meaningful attributes (IDs, content types, durations).
