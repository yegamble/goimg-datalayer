---
name: new-endpoint
version: 1.0.0
description: >
  Use this skill when adding a new API endpoint to goimg-datalayer. Triggers when:
  - User asks to "add endpoint", "implement API", "create route", or "add handler"
  - A new feature requires a new HTTP endpoint (not modifying an existing one)
  - User references an OpenAPI path that doesn't have a handler yet
  Covers the full pipeline: OpenAPI spec → oapi-codegen → domain → application →
  infrastructure → HTTP handler → Postman E2E test.
---

# Skill: New API Endpoint

## Prerequisites

Before starting, ensure:
- [ ] `make install-hooks` has been run
- [ ] Docker services running (`docker compose -f docker/docker-compose.yml up -d`)

## Step-by-Step Workflow

### Step 1 — OpenAPI Spec First (source of truth)

Edit `api/openapi/openapi.yaml`. Add:
- Path + HTTP method under `paths:`
- Request body schema under `components/schemas/`
- Response schemas (success + error)
- Security requirements (`bearerAuth` for protected routes)

Validate: `make validate-openapi`

### Step 2 — Regenerate Types

```bash
make generate   # runs oapi-codegen → updates internal/interfaces/http/generated/
```

### Step 3 — Domain Layer (if new entity needed)

Only if this endpoint introduces a new domain concept. Follow `bounded-context-scaffold.md`.

If modifying an existing entity, add methods to the aggregate root.

### Step 4 — Application Layer

**For write operations** — create command in `internal/application/{context}/commands/`:
```go
type CreateFooCommand struct { ... }
type CreateFooHandler struct { repo foo.FooRepository; ... }
func (h *CreateFooHandler) Handle(ctx context.Context, cmd CreateFooCommand) (*foo.Foo, error) { ... }
```

**For read operations** — create query in `internal/application/{context}/queries/`:
```go
type GetFooQuery struct { ... }
type GetFooHandler struct { repo foo.FooRepository }
func (h *GetFooHandler) Handle(ctx context.Context, q GetFooQuery) (*foo.Foo, error) { ... }
```

Write tests with mocked repositories (85% coverage target).

### Step 5 — Infrastructure (if new DB queries needed)

Add methods to the relevant `internal/infrastructure/persistence/postgres/*_repository.go`.

If new columns/tables are needed: create a migration first (see `db-migration` skill).

### Step 6 — HTTP Handler

Add method to existing handler or create `internal/interfaces/http/handlers/{context}_handler.go`:

```go
func (h *FooHandler) CreateFoo(w http.ResponseWriter, r *http.Request) {
    // 1. Parse auth context (GetUserFromContext)
    // 2. Decode + validate request body
    // 3. Delegate to application command/query handler
    // 4. Map domain result to DTO
    // 5. middleware.WriteJSON(w, http.StatusCreated, dto)
    // All errors: middleware.WriteError(w, r, status, title, detail)
}
```

### Step 7 — Register Route

In `internal/interfaces/http/handlers/router.go`, add route to appropriate auth group.

### Step 8 — Wire Dependencies

In `cmd/api/main.go`, inject new command/query handler into the HTTP handler constructor.

### Step 9 — Handler Tests

`internal/interfaces/http/handlers/{context}_handler_test.go`:
- Use `httptest.NewRecorder()` + `httptest.NewRequest()`
- Mock application layer handlers
- Test: happy path, auth failure, validation error, not found, domain errors
- 75% coverage target

### Step 10 — Postman E2E Test

Add to `tests/e2e/postman/goimg-api.postman_collection.json`:
- Happy path request with assertions on status + body schema
- Error cases (missing auth, invalid input)
- Run: `make test-e2e-dry` (validates JSON structure)

### Step 11 — Pre-Push Checks (MANDATORY)

```bash
make agent-check       # Full validation
make test              # All tests with race detector
make validate-openapi  # OpenAPI spec valid
```

## Common Pitfalls

- **Don't skip OpenAPI first** — oapi-codegen generates types from the spec; writing handler before spec creates drift
- **Don't put business logic in handler** — handler only parses/validates/delegates/responds
- **Match error types** — use `errors.Is(err, domain.ErrFooNotFound)` to map to 404
- **Use RFC 7807** — all errors via `middleware.WriteError`, never raw `http.Error`
- **Rate limit sensitive endpoints** — login, register, password reset get per-IP limits

## Reference Files

- OpenAPI spec: `api/openapi/openapi.yaml`
- Router: `internal/interfaces/http/handlers/router.go`
- Existing handler example: `internal/interfaces/http/handlers/image_handler.go`
- Middleware errors: `internal/interfaces/http/middleware/error_handler.go`
- DI wiring: `cmd/api/main.go`
- Postman collection: `tests/e2e/postman/goimg-api.postman_collection.json`
