# Coding Standards & Tooling

> Load this guide when writing or reviewing Go code.

## Go Style Rules

1. Follow [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
2. Descriptive names - no single-letter variables except loop indices
3. Functions < 50 lines preferred; single responsibility
4. Document all exported types and functions
5. No commented-out code in commits

## Error Handling

### Always Wrap Errors

```go
// CORRECT: Wrap with context
func (r *PostgresUserRepository) FindByID(ctx context.Context, id UserID) (*User, error) {
    var user userModel
    err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", id.String())
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fmt.Errorf("user %s: %w", id, ErrUserNotFound)
        }
        return nil, fmt.Errorf("find user by id %s: %w", id, err)
    }
    return user.toDomain()
}

// WRONG: Lost context, swallowed error
func (r *PostgresUserRepository) FindByID(ctx context.Context, id UserID) (*User, error) {
    var user userModel
    r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", id.String())
    return user.toDomain() // Error ignored!
}
```

### Domain Errors

```go
// internal/domain/identity/errors.go
var (
    ErrUserNotFound       = errors.New("user not found")
    ErrUserAlreadyExists  = errors.New("user already exists")
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrEmailRequired      = errors.New("email is required")
    ErrEmailInvalid       = errors.New("email format is invalid")
    ErrPasswordTooWeak    = errors.New("password does not meet requirements")
    ErrAccountSuspended   = errors.New("account is suspended")
)
```

## Domain Rules

| Rule | Correct | Wrong |
|------|---------|-------|
| Domain imports | `stdlib`, `errors` | `database/sql`, `github.com/redis/...` |
| Entity creation | Factory function | Direct struct literal |
| Value objects | Immutable, validated | Mutable, unvalidated |
| Aggregate changes | Through root methods | Direct field access |
| Business logic | Domain/Application | HTTP handlers |

## Linting

Run before every commit:

```bash
golangci-lint run ./...
```

### Key Linters Enabled

| Linter | Purpose |
|--------|---------|
| `errcheck` | Unchecked errors |
| `govet` | Suspicious constructs |
| `staticcheck` | Static analysis |
| `gosec` | Security issues |
| `gocyclo` | Cyclomatic complexity (max 15) |
| `gocognit` | Cognitive complexity (max 20) |
| `dupl` | Code duplication (threshold 100) |
| `bodyclose` | HTTP response body close |
| `sqlclosecheck` | SQL rows close |
| `contextcheck` | Context propagation |

### Import Ordering

```go
import (
    // stdlib
    "context"
    "errors"
    "fmt"

    // external
    "github.com/lib/pq"

    // internal - local prefix
    "github.com/your-org/goimg-datalayer/internal/domain/identity"
)
```

## Pre-commit Hooks

Install once:
```bash
pre-commit install
pre-commit install --hook-type commit-msg
```

Hooks run:
- `go-fmt`, `go-imports`, `go-vet`, `go-build`
- `golangci-lint`
- OpenAPI validation
- Generated code check
- Unit tests on push

## Make Targets

| Command | Purpose |
|---------|---------|
| `make build` | Build API and worker binaries |
| `make run` | Start API server |
| `make run-worker` | Start background worker |
| `make lint` | Run golangci-lint |
| `make test` | Full test suite |
| `make test-unit` | Unit tests with race detection |
| `make test-integration` | Integration tests |
| `make generate` | Regenerate from OpenAPI |
| `make migrate-up` | Run migrations |
| `make validate-openapi` | Lint spec, check generated code |

## Code Patterns

### Handler Pattern (Correct)

```go
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request
    var req requests.CreateUserRequest
    if err := DecodeJSON(r, &req); err != nil {
        RespondProblem(w, r, ProblemBadRequest(err.Error()))
        return
    }

    // 2. Validate DTO
    if err := Validate(req); err != nil {
        RespondProblem(w, r, ProblemValidation(err))
        return
    }

    // 3. Delegate to application layer
    user, err := h.createUser.Handle(r.Context(), commands.CreateUserCommand{
        Email:    req.Email,
        Username: req.Username,
        Password: req.Password,
    })

    // 4. Map errors to HTTP responses
    if err != nil {
        switch {
        case errors.Is(err, identity.ErrUserAlreadyExists):
            RespondProblem(w, r, ProblemConflict("user already exists"))
        case errors.Is(err, identity.ErrEmailInvalid):
            RespondProblem(w, r, ProblemBadRequest("invalid email"))
        default:
            RespondProblem(w, r, ProblemInternalError())
        }
        return
    }

    // 5. Map to response DTO
    RespondJSON(w, http.StatusCreated, responses.UserResponse{
        ID:       user.ID().String(),
        Email:    user.Email().String(),
        Username: user.Username().String(),
    })
}
```

### Command Handler Pattern

```go
type CreateUserHandler struct {
    users     identity.UserRepository
    publisher EventPublisher
}

func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) (*identity.User, error) {
    // 1. Create value objects
    email, err := identity.NewEmail(cmd.Email)
    if err != nil {
        return nil, err
    }

    // 2. Check business rules
    existing, _ := h.users.FindByEmail(ctx, email)
    if existing != nil {
        return nil, identity.ErrUserAlreadyExists
    }

    // 3. Create aggregate
    user, err := identity.NewUser(email, username, passwordHash)
    if err != nil {
        return nil, err
    }

    // 4. Persist
    if err := h.users.Save(ctx, user); err != nil {
        return nil, fmt.Errorf("save user: %w", err)
    }

    // 5. Publish events
    for _, event := range user.Events() {
        h.publisher.Publish(ctx, event)
    }

    return user, nil
}
```
