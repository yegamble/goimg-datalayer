# Agent Verification Checklist

> **CRITICAL**: Complete this checklist before submitting any code changes.

## Quick Validation Commands

```bash
# Run all checks
go fmt ./... && go vet ./... && golangci-lint run
go test -race ./...
make validate-openapi
```

---

## Before Writing Code

- [ ] Read relevant bounded context in `internal/domain/`
- [ ] Check existing domain errors in `errors.go`
- [ ] Review OpenAPI spec if touching HTTP endpoints
- [ ] Identify DDD patterns needed (entity, value object, aggregate)
- [ ] Skim existing tests for patterns to follow

---

## During Implementation

### Domain Layer Rules

| DO | DON'T |
|----|-------|
| Use factory functions for entities | Create entities with struct literals |
| Make value objects immutable | Allow mutation after construction |
| Validate in constructors | Skip validation |
| Define repo interfaces in domain | Import `database/sql` in domain |
| Collect domain events | Publish events inside aggregates |

### Application Layer Rules

| DO | DON'T |
|----|-------|
| Create value objects from primitives | Pass primitives to domain |
| Handle all error cases | Ignore errors from repositories |
| Wrap errors with context | Return bare errors |
| Publish events after save | Publish before persistence |

### HTTP Handler Rules

| DO | DON'T |
|----|-------|
| Parse → Validate → Delegate → Respond | Put business logic in handlers |
| Use Problem Details for errors | Return raw error messages |
| Map domain errors to HTTP status | Expose internal error details |
| Use DTOs at boundaries | Pass domain objects to JSON |

---

## Before Committing

### Code Quality

- [ ] `go fmt ./...` passes
- [ ] `go vet ./...` passes
- [ ] `golangci-lint run` passes
- [ ] `go test -race ./...` passes
- [ ] No hardcoded secrets or credentials
- [ ] No commented-out code
- [ ] No `TODO` without issue reference

### API Contract

- [ ] `make validate-openapi` passes
- [ ] `make generate` produces no diff
- [ ] New endpoints documented in OpenAPI spec
- [ ] Error responses use RFC 7807 format

### Test Coverage

- [ ] New code has tests
- [ ] Coverage >= 80% for changed files
- [ ] Domain code coverage >= 90%
- [ ] Tests are parallelized where possible

### Security

- [ ] No SQL injection vulnerabilities
- [ ] Input validation at boundaries
- [ ] Errors don't leak internal details
- [ ] JWT/RBAC changes tested

---

## Common Mistakes to Avoid

### Architecture Violations

```go
// WRONG: Infrastructure import in domain
package identity
import "database/sql"  // ❌ Never in domain layer

// CORRECT: Only stdlib in domain
package identity
import "errors"  // ✓
```

### Error Handling

```go
// WRONG: Swallowing errors
user, _ := repo.FindByID(ctx, id)  // ❌

// WRONG: No context
return err  // ❌

// CORRECT: Wrap with context
if err != nil {
    return nil, fmt.Errorf("find user %s: %w", id, err)  // ✓
}
```

### Business Logic Placement

```go
// WRONG: Logic in handler
func (h *Handler) Create(w, r) {
    if len(req.Password) < 8 {  // ❌ Business rule in handler
        // ...
    }
}

// CORRECT: Logic in domain
func NewPassword(value string) (Password, error) {
    if len(value) < 8 {  // ✓ Domain validates
        return Password{}, ErrPasswordTooWeak
    }
}
```

### Entity Creation

```go
// WRONG: Direct construction
user := &User{
    email: "test@example.com",  // ❌ Bypasses validation
}

// CORRECT: Factory function
email, _ := NewEmail("test@example.com")
user, err := NewUser(email, username, hash)  // ✓ Validates
```

---

## Pull Request Requirements

Before submitting PR:

- [ ] All CI checks pass
- [ ] Code coverage maintained or improved
- [ ] OpenAPI spec updated (if API changed)
- [ ] Postman collection updated (if API changed)
- [ ] Database migrations included (if schema changed)
- [ ] No breaking changes (or version bumped)
- [ ] Commit messages follow convention

---

## Quick Reference

### Layer Import Rules

| Layer | Can Import | Cannot Import |
|-------|------------|---------------|
| Domain | stdlib | application, infrastructure, interfaces |
| Application | domain | infrastructure, interfaces |
| Infrastructure | domain, application | interfaces |
| Interfaces | all | - |

### Test Coverage Targets

| Layer | Target |
|-------|--------|
| Domain | 90% |
| Application | 85% |
| Infrastructure | 70% |
| Handlers | 75% |
| **Overall** | **80%** |

### HTTP Status Mapping

| Domain Error | HTTP Status |
|--------------|-------------|
| `ErrNotFound` | 404 |
| `ErrAlreadyExists` | 409 |
| `ErrInvalidCredentials` | 401 |
| `ErrInsufficientPermissions` | 403 |
| Validation errors | 400 |
| Unknown/internal | 500 |
