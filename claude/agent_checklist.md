# Agent Verification Checklist

> **CRITICAL**: Complete this checklist before submitting any code changes.

## Known Critical Issues (2026-02-03 Audit)

Before working on these areas, be aware of documented critical issues:

| File | Issue | Action |
|------|-------|--------|
| `internal/infrastructure/security/password_cache.go:110-141` | Race condition in InMemoryPasswordCache | Add mutex or mark test-only |
| `cmd/api/main.go:869-900` | Stub NSFW repository | Implement real repository |
| `internal/application/gallery/commands/upload_image.go:170-177` | Swallowed event errors | Add retry/outbox pattern |
| `internal/infrastructure/persistence/postgres/activity_repository.go:38-51` | Unbounded LATERAL JOIN | Cap inner limit |

See `claude/audit_report_2026-02-03.md` for full details and fix recommendations.

---

## Mandatory CI Check Before Push

> **ALL CLAUDE AGENTS MUST RUN `make agent-check` BEFORE PUSHING ANY COMMITS**

```bash
# REQUIRED: Run before every push (MANDATORY - NO EXCEPTIONS)
make agent-check            # Full check (Go + non-Go changes)
make agent-check-quick      # Quick check (non-Go changes only)

# RECOMMENDED: Additional checks if Go toolchain available
make pre-commit             # go fmt + go vet + golangci-lint
```

### Agent Push Sequence (REQUIRED)

```bash
# 1. Stage changes
git add <files>

# 2. Run CI check (MANDATORY)
make agent-check

# 3. Commit (only if step 2 passes)
git commit -m "message"

# 4. Push
git push -u origin <branch>
```

If checks fail:

1. Fix ALL failures
2. Re-run `make agent-check` until it passes
3. Only push when all checks pass

**Never use `git commit --no-verify` or skip CI checks.**

## Quick Validation Commands

```bash
# MANDATORY before push
make agent-check                              # Agent CI validation
# Additional checks
make pre-commit                               # Full lint check
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

### Lint Check (MANDATORY)

> **Run `make pre-commit` before every commit. This is non-negotiable.**

- [ ] `make pre-commit` passes (runs go fmt, go vet, golangci-lint)

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

- [ ] **`make agent-check` passes** (MANDATORY - agent CI validation)
- [ ] **`make pre-commit` passes** (MANDATORY - lint check, if Go toolchain available)
- [ ] All CI checks pass
- [ ] Code coverage maintained or improved
- [ ] OpenAPI spec updated (if API changed)
- [ ] Postman collection updated (if API changed)
- [ ] Database migrations included (if schema changed)
- [ ] No breaking changes (or version bumped)
- [ ] Commit messages follow convention
- [ ] **Regression Check**: Verified no critical functions were removed or broken
- [ ] **Robustness**: Solution does not rely on missing components/hacks
- [ ] **Auto-Merge Ready**: Code quality is sufficient for auto-merge


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
