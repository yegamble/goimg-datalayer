# Adding a New Bounded Context

End-to-end checklist for adding a new bounded context (e.g., `billing`, `analytics`).

## Layer Order (strict — implement in this sequence)

### 1. Domain Layer (`internal/domain/{context}/`)

```
{context}/
├── errors.go          # var ErrFooNotFound = errors.New("foo not found")
├── events.go          # Domain event structs (e.g., FooCreated)
├── repository.go      # Interface: FooRepository
├── {entity}_id.go     # UUID value object (NewFooID, ParseFooID)
├── {value_object}.go  # Immutable VOs with validation in constructor
└── {entity}.go        # Aggregate root via NewFoo() factory
```

**Rules:**
- Only import `stdlib` + other domain packages
- Entities created via `NewFoo()` factory — no raw struct literals
- Value objects validate in constructor, return error on invalid input
- Repository interface here; implementation goes in infrastructure

### 2. Application Layer (`internal/application/{context}/`)

```
{context}/
├── commands/
│   └── create_foo.go    # CreateFooCommand + CreateFooHandler
├── queries/
│   └── get_foo.go       # GetFooQuery + GetFooHandler
└── dto/                 # Application DTOs (if needed)
```

**Rules:**
- Only import domain packages (never infrastructure or http)
- Commands: state-changing, publish events after save
- Queries: read-only, no side effects
- Convert primitives → value objects before calling domain
- Wrap all errors: `fmt.Errorf("context: %w", err)`
- Coverage target: 85%

### 3. Infrastructure Layer (`internal/infrastructure/persistence/postgres/`)

```
foo_repository.go       # PostgresUserRepository implementing FooRepository
```

**Pattern:**
```go
type fooModel struct { ID string `db:"id"`; ... }
func (m *fooModel) toDomain() (*foo.Foo, error) { ... }
func fromDomain(f *foo.Foo) *fooModel { ... }

func (r *PostgresFooRepository) FindByID(ctx context.Context, id foo.FooID) (*foo.Foo, error) {
    var model fooModel
    err := sqlx.GetContext(ctx, r.executor(), &model, `SELECT * FROM foos WHERE id=$1`, id.String())
    if errors.Is(err, sql.ErrNoRows) {
        return nil, foo.ErrFooNotFound
    }
    if err != nil { return nil, fmt.Errorf("find foo: %w", err) }
    return model.toDomain()
}
```

### 4. Database Migration (`migrations/`)

```
00022_create_foos.sql   # Next sequential number
```

```sql
-- +goose Up
CREATE TABLE foos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- fields...
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS foos;
```

Run: `make migrate-up`

### 5. HTTP Handler (`internal/interfaces/http/handlers/foo_handler.go`)

```go
type FooHandler struct {
    createFoo *commands.CreateFooHandler
    getFoo    *queries.GetFooHandler
    logger    zerolog.Logger
}
// Parse → validate → delegate → map to DTO → respond
// All errors use middleware.WriteError(w, r, status, title, detail)
```

### 6. Router Registration (`internal/interfaces/http/handlers/router.go`)

Add routes in the appropriate auth group.

### 7. OpenAPI Spec (`api/openapi/openapi.yaml`)

Add paths, schemas, and responses. Run `make validate-openapi`.

### 8. Dependency Wiring (`cmd/api/main.go`)

Wire repo → command/query handler → HTTP handler → router.

### 9. Tests

| Layer | Target | Pattern |
|-------|--------|---------|
| Domain | 90% | Table-driven, no mocks |
| Application | 85% | Mock repos via testify/mock |
| Infrastructure | 70% | testcontainers-go (real Postgres) |
| HTTP | 75% | httptest, mock application handlers |

### 10. Postman E2E

Add requests to `tests/e2e/postman/goimg-api.postman_collection.json` covering happy path, auth, validation errors.

## Quick Verification

```bash
make lint && make test && make validate-openapi && make agent-check
```
