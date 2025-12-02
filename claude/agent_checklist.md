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

### Security Review (CRITICAL)

**Authentication & Authorization:**
- [ ] JWT tokens use RS256 (asymmetric) algorithm only
- [ ] Access tokens expire within 15 minutes
- [ ] Refresh tokens are hashed (SHA-256) before storage
- [ ] Token rotation implemented with replay detection
- [ ] Session invalidation on logout clears Redis state
- [ ] Password hashing uses Argon2id (not bcrypt)
  - Time: 2, Memory: 64MB, Threads: 4, KeyLen: 32
- [ ] RBAC permissions verified at handler and application layer
- [ ] No privilege escalation paths exist
- [ ] Admin actions require authentication re-verification

**Input Validation & Injection Prevention:**
- [ ] All user inputs validated at boundaries (handlers/commands)
- [ ] SQL queries use parameterized statements (no string concatenation)
- [ ] Search queries sanitized for SQL injection
- [ ] Path traversal prevention on file operations
  - Use `filepath.Clean()` and validate against base directory
- [ ] Command injection prevention (avoid `os/exec` with user input)
- [ ] JSON/XML parsers have size limits configured
- [ ] No reflected user input in responses without encoding

**Image Security:**
- [ ] File size limit enforced (10MB max) before processing
- [ ] MIME type validated via content sniffing, not extension
- [ ] Image dimensions validated (max 8192x8192)
- [ ] Pixel count limit enforced (max 100M pixels)
- [ ] ClamAV malware scanning on all uploads
  - Verify signatures are up-to-date
- [ ] Images re-encoded through libvips (prevents polyglot files)
- [ ] EXIF metadata stripped before storage
- [ ] Upload rate limiting enabled (50/hour per user)
- [ ] Filename sanitization (no path separators)

**Data Protection:**
- [ ] No hardcoded secrets, API keys, or credentials
- [ ] Passwords never logged (even hashed)
- [ ] PII not logged (email, IP addresses redacted in logs)
- [ ] Database connections use TLS/SSL
- [ ] Error messages don't leak implementation details
- [ ] Stack traces not exposed to clients
- [ ] Sensitive data encrypted at rest (if applicable)
- [ ] Redis connections authenticated with password

**Session & Token Security:**
- [ ] Session IDs are cryptographically random (UUID v4)
- [ ] Session fixation prevention (regenerate on login)
- [ ] Concurrent session limits enforced
- [ ] Token blacklist checked on revocation
- [ ] JWT "kid" header validated to prevent key confusion
- [ ] Token audience ("aud") claim validated

**HTTP Security:**
- [ ] Security headers applied to all responses:
  - X-Content-Type-Options: nosniff
  - X-Frame-Options: DENY
  - X-XSS-Protection: 1; mode=block
  - Content-Security-Policy: default-src 'self'
  - Strict-Transport-Security (production only)
- [ ] CORS configuration restricts origins (not "*" in production)
- [ ] Rate limiting configured per endpoint sensitivity
- [ ] Request size limits enforced (prevent DoS)
- [ ] Timeout configured on all HTTP clients

**Authorization Checks (IDOR Prevention):**
- [ ] Resource ownership verified before read/write/delete
- [ ] User cannot access other users' private resources
- [ ] Album/image visibility rules enforced
- [ ] Moderator actions logged to audit trail
- [ ] Admin panel endpoints require admin role

**Audit & Logging:**
- [ ] Authentication events logged (login, logout, failure)
- [ ] Authorization failures logged
- [ ] Moderation actions logged with actor ID
- [ ] File upload events logged
- [ ] No sensitive data in logs (passwords, tokens)
- [ ] Request IDs present for tracing

**Third-Party Dependencies:**
- [ ] No known vulnerabilities in dependencies
  - Run: `go list -json -m all | nancy sleuth`
  - Run: `trivy fs --security-checks vuln .`
- [ ] Dependencies pinned to specific versions
- [ ] Minimal attack surface (fewest dependencies necessary)

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
