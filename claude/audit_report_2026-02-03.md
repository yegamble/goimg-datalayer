# Comprehensive Codebase Audit Report

> **Audit Date**: 2026-02-03
> **Branch**: main (commit 18d1916)
> **Auditor**: Claude Code (Opus 4.5)
> **Status**: Complete - 540 Go files analyzed

---

## Executive Summary

| Agent | Status | Key Finding |
|-------|--------|-------------|
| Code Consistency | Complete | 20 issues (5 critical) |
| Unfinished Code | Complete | Stub NSFW repository in production |
| Security | Complete | Grade A- (95/100) |
| Documentation | Complete | 91% complete, 15 action items |
| Test Suite | Complete | 72/100 quality score |
| Performance | Complete | 2 critical, 4 high severity issues |

**Overall Assessment**: The codebase demonstrates solid architecture and security practices, but has critical issues that require immediate attention before the next production deployment.

---

## Critical Issues (Must Fix Before Deployment)

### 1. InMemoryPasswordCache - Race Condition & Memory Leak

**Severity**: CRITICAL
**File**: `internal/infrastructure/security/password_cache.go:110-141`

**Issue**: The `InMemoryPasswordCache` struct uses an unbounded map without synchronization:

```go
type InMemoryPasswordCache struct {
    data map[string]bool  // No mutex, no TTL, unbounded
}

func (c *InMemoryPasswordCache) Set(...) error {
    key := prefix + ":" + suffix
    c.data[key] = pwned  // RACE CONDITION: concurrent writes unsynchronized
    return nil
}
```

**Problems**:
- No `sync.Mutex` protecting concurrent access
- No TTL support (entries never evicted)
- Will consume unbounded memory if used in production
- Not thread-safe for concurrent requests

**Impact**:
- Memory leak in production (grows monotonically)
- Data races on concurrent password checks
- Memory exhaustion under load

**Fix**:
- Add `sync.RWMutex` protection
- Implement LRU eviction with max size limit
- Mark as test-only or replace with Redis-only implementation

---

### 2. Stub NSFW Repository in Production

**Severity**: CRITICAL
**File**: `cmd/api/main.go:873-890`

**Issue**: `noOpNSFWScanRepository` returns `nil, nil` for all operations with `//nolint:nilnil` suppressions:

```go
func (r *noOpNSFWScanRepository) FindByID(ctx context.Context, id moderation.NSFWScanID) (*moderation.NSFWScan, error) {
    return nil, nil  //nolint:nilnil
}

func (r *noOpNSFWScanRepository) FindByImageID(ctx context.Context, imageID gallery.ImageID) (*moderation.NSFWScan, error) {
    return nil, nil  //nolint:nilnil
}
```

**Impact**: NSFW scan functionality is completely non-functional in production.

**Fix**:
- Implement real PostgreSQL repository
- Or disable NSFW feature via feature flag until implementation complete

---

### 3. Silently Ignored Event Publishing Errors

**Severity**: CRITICAL
**File**: `internal/application/gallery/commands/upload_image.go:170-177`

**Issue**: Domain events fail to publish but errors are only logged, not returned:

```go
if err := h.eventPublisher.Publish(ctx, event); err != nil {
    h.logger.Error(...)
    // No action taken - error silently ignored
}
```

**Also affects**:
- `internal/application/gallery/commands/update_image.go:167-174`
- Similar patterns in other command handlers

**Impact**:
- Lost business events
- Eventual consistency violations
- Silent failures in production

**Fix**:
- Implement retry mechanism with exponential backoff
- Or implement transactional outbox pattern
- At minimum, return error to caller

---

### 4. Cache Stampede Vulnerability

**Severity**: CRITICAL
**File**: `internal/infrastructure/security/password_cache.go:36-107`

**Issue**: `RedisPasswordCache.Get()` doesn't handle concurrent calls correctly:

```go
func (c *RedisPasswordCache) Get(ctx context.Context, prefix, suffix string) (bool, bool) {
    key := c.buildKey(prefix, suffix)
    val, err := c.client.Get(ctx, key).Result()
    // Returns cache miss on any Redis error
    return false, false
}
```

**Problems**:
- No cache stampede protection (concurrent misses → parallel API calls to HIBP)
- Failing open silently hides Redis connectivity issues
- No monitoring of how many HIBP API calls occur after cache miss

**Impact**:
- HIBP API rate limiting violations under load
- API request amplification (N concurrent requests → N HIBP calls)

**Fix**:
- Implement `singleflight` for request coalescing
- Add probabilistic early expiration (refresh before TTL expires)
- Add metrics: `cache_miss_count`, `api_call_count`

---

### 5. Activity Feed Unbounded LATERAL JOIN

**Severity**: CRITICAL
**File**: `internal/infrastructure/persistence/postgres/activity_repository.go:38-51`

**Issue**: The feed query uses `CROSS JOIN LATERAL` with unbounded inner LIMIT:

```sql
SELECT a.id, a.actor_id, a.activity_type, ...
FROM user_follows uf
CROSS JOIN LATERAL (
    SELECT *
    FROM activities
    WHERE actor_id = uf.followed_id
    ORDER BY created_at DESC
    LIMIT $4  -- Inner limit = limit + offset (can be large)
) a
WHERE uf.follower_id = $1
ORDER BY a.created_at DESC
LIMIT $2 OFFSET $3
```

At line 213:
```go
pagination.Limit()+pagination.Offset(), // $4: Inner limit (must be limit + offset)
```

**Impact**:
- Memory explosion for deep pagination (offset 10000+ × followed users)
- CPU spikes on large LATERAL join
- Slow user feed generation at scale

**Fix**:
- Cap inner LIMIT: `min(limit + offset, 1000)`
- Implement cursor-based pagination instead of offset
- Add index: `CREATE INDEX idx_activities_actor_created ON activities(actor_id, created_at DESC)`

---

## High Priority Issues

### 6. Incomplete Mock Implementations (Tests)

**Severity**: HIGH
**Files**:
- `internal/application/identity/commands/delete_user_test.go:19-48`
- `internal/application/identity/queries/get_user_sessions_test.go:17-46`

**Issue**: Mock objects use `panic("not implemented")` for required interface methods:

```go
// Line 24-36 in delete_user_test.go
func (m *MockSessionStoreForDelete) Create() error { panic("not implemented") }
func (m *MockSessionStoreForDelete) Get() error    { panic("not implemented") }
func (m *MockSessionStoreForDelete) Revoke() error { panic("not implemented") }
```

**Impact**: Tests won't catch interface changes; runtime panics in untested paths.

**Fix**: Implement all methods or use testify/mock properly.

---

### 7. Flaky Tests with time.Sleep (18 files)

**Severity**: HIGH
**Key Files**:
- `tests/unit/jwt_service_test.go:209` - `time.Sleep(2 * time.Second)` for token expiration
- `tests/integration/session_store_test.go:203` - `time.Sleep(2 * time.Second)` for Redis TTL
- `tests/integration/token_blacklist_test.go:113,191` - Multiple sleeps
- `internal/infrastructure/persistence/redis/session_store_test.go:514` - `time.Sleep(2500ms)`

**Impact**: Tests fail intermittently in CI.

**Fix**: Replace with mock clock or polling with timeout.

---

### 8. Missing Community Application Layer Tests

**Severity**: HIGH
**Location**: `internal/application/community/`

**Coverage**: 3 tests for 33 files = **9%**

**Missing Tests**:
- All group query handlers (10+ handlers)
- Group album operations
- RBAC tests
- Group settings tests

**Fix**: Add comprehensive test suite for community domain.

---

### 9. Activity Feed Count Query Missing Index

**Severity**: HIGH
**File**: `internal/infrastructure/persistence/postgres/activity_repository.go:53-58`

**Issue**: Count query executes full table scan:

```sql
SELECT COUNT(*)
FROM activities a
INNER JOIN user_follows uf ON a.actor_id = uf.followed_id
WHERE uf.follower_id = $1
```

**Fix**: Create indexes:
```sql
CREATE INDEX idx_user_follows_follower ON user_follows(follower_id);
CREATE INDEX idx_activities_actor_created ON activities(actor_id DESC, created_at DESC);
```

---

### 10. Untyped errors.New() Without Context

**Severity**: HIGH
**Files**:
- `internal/application/community/commands/add_image_to_album.go`
- `internal/application/community/commands/remove_image_from_album.go`
- `internal/application/community/commands/create_group.go`
- `internal/application/identity/errors.go`

**Issue**: Returns `errors.New("message")` instead of `fmt.Errorf("context: %w", err)`.

**Fix**: Use `fmt.Errorf()` consistently with context wrapping.

---

### 11. Duplicate Helper Functions

**Severity**: HIGH
**File**: `internal/domain/identity/device_fingerprint.go:158-172`

**Issue**: Two functions doing similar work:

```go
func contains(s, substr string) bool { ... }
func containsSubstring(s, substr string) bool { ... }
```

**Fix**: Consolidate into one function.

---

### 12. Missing Global Request Body Size Limits

**Severity**: HIGH
**Location**: HTTP middleware layer

**Issue**: Only image uploads have multipart form size limits (50MB). Non-multipart endpoints lack explicit `http.MaxBytesReader` protection.

**Fix**: Add middleware:
```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB max for JSON
```

---

## Medium Priority Issues

### 13. Batch Loader No Safety Limits

**File**: `internal/infrastructure/persistence/postgres/batch_loader.go:20-71`

**Issue**: No limit on `imageIDs` length in batch operations.

**Fix**: Add hard limit: `if len(imageIDs) > 1000 { return error }`

---

### 14. FindExpiredGuests Unbounded Slice

**File**: `internal/infrastructure/persistence/postgres/user_repository.go:328-343`

**Issue**: Caller-controlled LIMIT without validation, unbounded slice allocation.

**Fix**: Enforce maximum: `const maxBatchSize = 10000`

---

### 15. Image Search Complex Dynamic Query

**File**: `internal/infrastructure/persistence/postgres/image_repository.go:578-696`

**Issue**: LATERAL subquery for NSFW scans executed per image.

**Fix**: Add index: `CREATE INDEX idx_nsfw_scans_image_status ON nsfw_scans(image_id, status, scanned_at DESC)`

---

### 16. Tag Management N+1 Risk

**File**: `internal/infrastructure/persistence/postgres/image_repository.go:950-968`

**Issue**: `FindByID` triggers 3 separate queries (image, variants, tags).

**Fix**: Audit handlers for loops calling `FindByID`; use `FindByIDs` with batch loading.

---

### 17. Validate X-Forwarded-For When TrustProxy Enabled

**Files**: `internal/interfaces/http/middleware/logging.go`, `rate_limit.go`

**Issue**: When `TrustProxy=true`, trusts X-Forwarded-For without validation.

**Fix**: Add allowlist check before trusting proxy headers.

---

### 18. Missing Sprint 22 Plan Document

**Location**: `claude/` directory

**Issue**: Account Tiers/Subscriptions mentioned in phase_3 but no detailed spec.

**Fix**: Create `claude/sprint_22_plan.md` with requirements.

---

### 19. Bloated NEXT_STEPS.md

**File**: `claude/NEXT_STEPS.md` (31.8 KB)

**Issue**: File mixes historical context with current status.

**Fix**: Split into `NEXT_STEPS.md` (current only) + `archive/sprint_history.md`.

---

## Security Assessment

### Grade: A- (95/100)

**Strengths**:
- RS256 JWT with 4096-bit RSA keys (OWASP 2024 compliant)
- Short-lived access tokens (15 minutes)
- Token blacklist via Redis
- Comprehensive rate limiting (6 different limiters)
- Full security headers (CSP, HSTS, X-Frame-Options, etc.)
- All queries use parameterized statements (no SQL injection)
- RFC 7807 error responses (no internal details exposed)

**Minor Gaps**:
- Missing global request body size limits
- X-Forwarded-For validation when TrustProxy enabled
- Cookie security headers (if sessions used)

---

## Test Suite Assessment

### Score: 72/100

**Metrics**:
- Total test files: 206
- Skipped tests: 127
- Tests using time.Sleep: 18

**Coverage by Layer**:

| Layer | Test Files | Source Files | Coverage |
|-------|-----------|--------------|----------|
| Domain | ~70 | 105 | 67% |
| Application | ~100 | 131 | 76% |
| Infrastructure | ~20 | 73 | 27% |
| HTTP/Interfaces | ~16 | 41 | 39% |

**Critical Gaps**:
- Community application: 9%
- Activity application: 0%
- Notification application: 25%

---

## Documentation Assessment

### Completeness: 91%

**Fully Documented**:
- Sprints 1-20 (complete)
- Architecture and DDD layering
- API security practices
- Testing and CI workflows

**Missing/Outdated**:
- Sprint 22 plan (Account Tiers)
- Sprint 21 (Video Support) - DRAFT only
- NEXT_STEPS.md needs splitting

---

## Recommended Action Plan

### Week 1 (Critical - Before Next Deployment)

| # | Task | File | Effort |
|---|------|------|--------|
| 1 | Fix InMemoryPasswordCache race condition | password_cache.go | 2h |
| 2 | Implement NSFW repository or disable feature | main.go | 4h |
| 3 | Add cache stampede protection | password_cache.go | 2h |
| 4 | Fix top 5 flaky time.Sleep tests | Various test files | 3h |
| 5 | Cap activity feed inner LIMIT | activity_repository.go | 1h |

### Week 2 (High Priority)

| # | Task | File | Effort |
|---|------|------|--------|
| 1 | Add database indexes for activity feed | migrations/ | 1h |
| 2 | Complete mock implementations | delete_user_test.go | 1h |
| 3 | Add Community application layer tests | application/community/ | 6h |
| 4 | Add global request body size limits | middleware/ | 1h |
| 5 | Fix errors.New() → fmt.Errorf() | Multiple files | 2h |

### Week 3 (Medium Priority)

| # | Task | File | Effort |
|---|------|------|--------|
| 1 | Add safety limits to batch operations | batch_loader.go | 2h |
| 2 | Remove duplicate helper functions | device_fingerprint.go | 30m |
| 3 | Split NEXT_STEPS.md | claude/ | 1h |
| 4 | Create sprint_22_plan.md | claude/ | 2h |

---

## Files Requiring Immediate Attention

```
CRITICAL:
├── internal/infrastructure/security/password_cache.go (race condition)
├── cmd/api/main.go:873-890 (stub NSFW repo)
├── internal/application/gallery/commands/upload_image.go:170-177 (swallowed errors)
└── internal/infrastructure/persistence/postgres/activity_repository.go:38-51 (unbounded query)

HIGH:
├── internal/application/identity/commands/delete_user_test.go (incomplete mocks)
├── tests/unit/jwt_service_test.go:209 (flaky sleep)
├── tests/integration/session_store_test.go:203 (flaky sleep)
└── internal/application/community/ (missing tests)
```

---

## Appendix: Code Consistency Issues (Full List)

### Untyped Errors (Replace with fmt.Errorf)

1. `internal/application/community/commands/add_image_to_album.go` - `errors.New("image already in album")`
2. `internal/application/community/commands/remove_image_from_album.go` - `errors.New("image not in album")`
3. `internal/application/community/commands/create_group.go` - `errors.New("only group owner can perform this action")`
4. `internal/application/identity/errors.go` - Multiple sentinel errors

### Silent Error Swallowing

1. `internal/application/gallery/commands/upload_image.go:170-177` - Event publishing
2. `internal/application/gallery/commands/upload_image.go:181-187` - Job enqueueing
3. `internal/application/gallery/commands/update_image.go:167-174` - Event publishing

### Infrastructure Layer Imports in Application

1. `internal/application/gallery/commands/upload_image.go:13` - Imports `storage` package directly

### Incomplete Mock Implementations

1. `internal/application/identity/commands/delete_user_test.go:19-48` - 5 panic methods
2. `internal/application/identity/queries/get_user_sessions_test.go:17-46` - 5 panic methods

### Dead/Duplicate Code

1. `internal/domain/identity/device_fingerprint.go:141-172` - Duplicate `contains()` and `containsSubstring()`

---

**Audit Complete**

*Generated by Claude Code audit on 2026-02-03*
