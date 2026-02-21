# User Story Gap Analysis & Integration Testing Plan

Created: 2026-02-21
Status: PENDING
Approved: Yes
Iterations: 0
Worktree: No

> **Status Lifecycle:** PENDING → COMPLETE → VERIFIED
> **Iterations:** Tracks implement→verify cycles (incremented by verify phase)

## Summary

**Goal:** Identify missing features against Flickr-style image gallery user stories, fill critical gaps (password reset, email verification), and add comprehensive integration tests for IPFS, S3/MinIO, and SMTP using Docker testcontainers.

**Architecture:** Extends existing DDD architecture (domain → application → infrastructure → HTTP). Integration tests use testcontainers-go with real Docker containers (ipfs/kubo, minio/minio, mailpit/mailpit), following the established Postgres/Redis container pattern.

**Tech Stack:** Go 1.25+, testcontainers-go, ipfs/kubo:v0.31.0, minio/minio, mailpit/mailpit, testify

## Scope

### In Scope

- Gap analysis: Flickr-style user stories vs current implementation
- IPFS integration tests with real Kubo Docker container
- IPFS orchestrator integration tests (dual-sync mode with primary + IPFS)
- S3 integration test consolidation (shared MinIO testcontainer helper)
- SMTP integration tests with Mailpit Docker container
- Password reset flow implementation
- Email verification flow implementation
- Postman E2E test updates for new auth endpoints

### Out of Scope

- Frontend/UI implementation
- Video support
- AI-based NSFW detection beyond current implementation
- CDN integration
- GraphQL endpoint
- Implementing `dual_async` IPFS writes (current code is a placeholder — documented as known gap)

## Prerequisites

- Docker running (for testcontainers)
- Go 1.25+ installed
- `make install-hooks` completed

## Runtime Environment

- **Start dependencies:** `docker-compose -f docker/docker-compose.yml up -d`
- **Run migrations:** `make migrate-up`
- **Start API:** `make run` (listens on port 8080 by default)
- **Start worker:** `make run-worker` (background job processing)
- **Health check:** `curl http://localhost:8080/health`
- **Run E2E tests:** `make test-e2e`
- **Run integration tests:** `go test -tags integration ./tests/integration/...`

## Context for Implementer

- **Patterns to follow:** Testcontainer pattern in `tests/integration/containers/postgres.go` and `tests/integration/containers/redis.go`
- **Docker availability:** `tests/integration/containers/docker_availability.go` — uses `isDockerUnavailablePanic()` / `isDockerUnavailable()` / `skipDockerUnavailable()` pattern (NOT `testing.Short()` at the container-helper level)
- **Port allocation:** Testcontainers use automatic random port allocation (expose container port, retrieve mapped host port dynamically). Never use fixed host port bindings — they conflict with docker-compose services already binding to those ports.
- **IPFS client:** `internal/infrastructure/storage/ipfs/client.go` — uses Kubo HTTP API directly via net/http
- **IPFS orchestrator:** `internal/infrastructure/storage/orchestrator/orchestrator.go` — dual-storage coordinator. **Critical:** `dual_async` mode currently does NOT write to IPFS (lines 89-99 fall through to primary-only). Only `dual_sync` mode writes to both. This is a known code gap.
- **IPFS Docker:** Already defined in `docker/docker-compose.yml` as `ipfs/kubo:v0.31.0`
- **Existing IPFS tests:** `internal/infrastructure/storage/ipfs/client_test.go` — 872 lines of unit tests using httptest mock server. No real IPFS integration tests exist.
- **S3 client:** `internal/infrastructure/storage/s3/s3.go` — uses aws-sdk-go-v2, supports AWS S3, DO Spaces, Backblaze B2, MinIO
- **Existing S3 integration tests:** `internal/infrastructure/storage/s3/s3_integration_test.go` — inline MinIO testcontainer setup (not using shared containers/ pattern). Tests exist but the container helper is not reusable.
- **SMTP sender:** `internal/infrastructure/email/smtp.go` — full SMTP implementation with TLS, rate limiting, MIME multipart. No integration tests exist (no SMTP testcontainer).
- **Email sender interface:** The identity application layer currently has NO email sender interface. Command handlers (register, login, etc.) do not accept an email sender dependency. An `EmailSender` interface must be created in the application layer for Tasks 7 and 8 to avoid violating DDD layering rules.
- **User domain entity:** `internal/domain/identity/user.go` — `ReconstructUser()` and `ReconstructUserWith2FA()` have fixed parameter lists. Adding `email_verified` field requires updating both reconstruction functions and all repository scan mappings.
- **Key files to read first:**
  - `tests/integration/containers/postgres.go` — testcontainer helper pattern
  - `tests/integration/containers/docker_availability.go` — Docker skip pattern
  - `internal/infrastructure/storage/ipfs/client.go` — IPFS client API
  - `internal/infrastructure/storage/orchestrator/orchestrator.go` — dual-storage modes (read Put/PutBytes carefully)
  - `internal/infrastructure/storage/s3/s3_integration_test.go` — existing inline MinIO setup
  - `internal/infrastructure/email/smtp.go` — SMTP sender implementation
  - `internal/domain/identity/user.go` — User entity reconstruction functions

## Shared Design: Auth Token Storage

**Both password reset (Task 7) and email verification (Task 8) use token-based flows.** To avoid scope coupling and reduce ambiguity:

- **Separate tables:** `password_reset_tokens` and `email_verification_tokens` — each has its own table. Do NOT combine into a single `auth_tokens` table.
- **Table schema (both):** `id UUID PK DEFAULT gen_random_uuid()`, `user_id UUID FK REFERENCES users(id) ON DELETE CASCADE`, `token UUID NOT NULL UNIQUE`, `expires_at TIMESTAMPTZ NOT NULL`, `used_at TIMESTAMPTZ` (null = unused), `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- **Concurrent request policy:** On each new token request, invalidate (DELETE) all previous unused tokens for the same user_id, then insert the new token. This prevents multiple valid tokens from existing simultaneously.
- **Repository interface:** Create a `TokenRepository` interface in `internal/domain/identity/repository.go` with methods: `CreateToken(ctx, userID, tokenType, expiresAt)`, `FindValidToken(ctx, token)`, `InvalidateToken(ctx, token)`, `InvalidateAllForUser(ctx, userID, tokenType)`. Both password reset and email verification handlers use this interface.

## Email Sender Interface Design

Command handlers in the application layer must not import the infrastructure email package directly. Design:

- Create `internal/application/identity/email_sender.go` defining:
  ```go
  type EmailSender interface {
      SendPasswordResetEmail(ctx context.Context, email, token string) error
      SendVerificationEmail(ctx context.Context, email, token string) error
  }
  ```
- The existing `SMTPSender` in `internal/infrastructure/email/smtp.go` will implement this interface
- Wire in `cmd/api/main.go`: inject `SMTPSender` as the `EmailSender` implementation
- When SMTP is disabled (`config.Enabled=false`), the `SMTPSender.Send()` already returns `ErrSMTPDisabled` — handlers should log and continue (do not fail the request)

## Gap Analysis: Flickr-Style User Stories vs Implementation

### Fully Implemented (25+ features)

| User Story | Status | Evidence |
|-----------|--------|----------|
| User registration (email/password) | Done | `POST /auth/register` with HIBP, rate limiting |
| User login with lockout | Done | `POST /auth/login` with 5 attempt lockout |
| Token refresh with rotation | Done | `POST /auth/refresh` |
| User profiles (CRUD) | Done | `/users/me`, `/users/{id}` |
| OAuth (Google, GitHub) | Done | `/auth/oauth/*` routes |
| 2FA/TOTP with backup codes | Done | `/auth/2fa/*` routes |
| Image upload with processing | Done | `POST /images` + asynq variants + ClamAV |
| Image retrieval/listing/update/delete | Done | Full CRUD on `/images/*` |
| Image variants (thumb/small/medium/large) | Done | `/images/{id}/variants/{size}` |
| Albums CRUD + image management | Done | `/albums/*` routes |
| Tags (search, popular, trending) | Done | `/tags/*` routes |
| Likes/favorites | Done | `/images/{id}/like` |
| Comments (add, list, delete) | Done | `/images/{id}/comments`, `/comments/{id}` |
| Follow/unfollow users | Done | `/users/{id}/follow` + followers/following lists |
| Activity feed | Done | `/feed` |
| Notifications | Done | `/notifications/*` |
| Content moderation (reports, bans) | Done | `/reports`, `/moderation/*` |
| NSFW scanning | Done | `/moderation/nsfw/*` |
| Groups/communities | Done | `/groups/*` with invitations, albums, images |
| Explore (recent, popular, featured) | Done | `/explore/*` |
| oEmbed | Done | `/oembed` |
| Image preview (social meta) | Done | `/images/{id}/preview` |
| QR codes | Done | `/images/{id}/qr` |
| Guest uploads | Done | `/auth/guest`, `/guest/images/{id}/claim` |
| Rate limiting (global, auth, login, upload) | Done | Redis-backed sliding window |
| Security headers (CSP, HSTS, etc.) | Done | Middleware stack |
| Health checks + Prometheus metrics | Done | `/health`, `/health/ready`, `/metrics` |
| RFC 7807 error responses | Done | `middleware.WriteError` |
| IPFS pin/unpin/status API | Done | `/images/{id}/ipfs` |
| Custom variant configs | Done | `/variant-configs/*` |
| Featured picks | Done | `/moderation/featured/*` |

### Gaps Found (Missing or Incomplete)

| User Story | Gap Type | Priority | Detail |
|-----------|----------|----------|--------|
| **Password reset flow** | Missing | HIGH | No `/auth/forgot-password` or `/auth/reset-password` endpoints. Users cannot recover accounts. |
| **Email verification** | Missing | HIGH | No `/auth/verify-email` endpoint. Users registered but emails unverified. |
| **Image download (original)** | Needs verification | MEDIUM | MVP spec defines `GET /images/{id}/download` but not confirmed in router. Task 7 includes a verification step. |
| **IPFS integration tests** | Missing | HIGH | 872 lines of unit tests (httptest mock) but zero integration tests with real Kubo node. |
| **IPFS orchestrator integration tests** | Missing | HIGH | Orchestrator unit tests use mock storage. Only dual-sync mode actually writes to IPFS (dual-async is a code placeholder). |
| **S3/MinIO integration test consolidation** | Partial | MEDIUM | S3 integration tests exist but use inline container setup, not the shared `containers/` pattern. |
| **SMTP integration tests** | Missing | HIGH | Full SMTP sender exists but zero integration tests. Cannot verify email delivery works. |
| **Orchestrator dual_async mode** | Code gap | MEDIUM | `ModeDualAsync` never writes to IPFS — falls through to primary-only. Not in scope to fix but documented. |
| **Watermarking** | Missing | LOW | Phase 2 feature, not MVP-critical |
| **Bulk operations** | Partial | LOW | Upload supports multiple but no batch delete/move/visibility-change |
| **Signed URLs for private images** | Missing | MEDIUM | No presigned URL support for time-limited access |

### Test Coverage Gaps

| Area | Current | Target | Gap |
|------|---------|--------|-----|
| Overall | ~65% | 80% | -15% |
| IPFS client (integration) | 0% | 70% | No real IPFS tests |
| IPFS orchestrator (integration) | 0% | 90% | No dual-storage integration tests |
| S3/MinIO (integration) | Exists but inline | 70% | Consolidate to shared pattern |
| SMTP (integration) | 0% | 70% | No real SMTP tests |
| Password reset | 0% | 85% | Feature doesn't exist |
| Email verification | 0% | 85% | Feature doesn't exist |

## Progress Tracking

**MANDATORY: Update this checklist as tasks complete. Change `[ ]` to `[x]`.**

- [x] Task 1: Create IPFS testcontainer helper
- [x] Task 2: Write IPFS client integration tests
- [ ] Task 3: Write IPFS orchestrator integration tests
- [ ] Task 4: Create SMTP testcontainer helper + integration tests
- [ ] Task 5: Create S3/MinIO shared testcontainer helper
- [ ] Task 6: Verify image download endpoint
- [ ] Task 7: Implement password reset flow
- [ ] Task 8: Implement email verification flow
- [ ] Task 9: Update Postman E2E collection for auth endpoints

**Total Tasks:** 9 | **Completed:** 2 | **Remaining:** 7

## Implementation Tasks

### Task 1: Create IPFS Testcontainer Helper

**Objective:** Create a reusable IPFS testcontainer helper following the existing Postgres/Redis container pattern, using ipfs/kubo:v0.31.0 Docker image.

**Dependencies:** None

**Files:**

- Create: `tests/integration/containers/ipfs.go`

**Key Decisions / Notes:**

- Follow pattern from `tests/integration/containers/postgres.go:31` — `NewPostgresContainer()` returns struct with connection info + cleanup
- Use `ipfs/kubo:v0.31.0` image (matches `docker/docker-compose.yml`)
- Expose container port 5001 (API) with **automatic random host port allocation** — do NOT use fixed port mapping. Retrieve the dynamically allocated host:port after container start (same as postgres.go and redis.go). This avoids conflicts with the docker-compose IPFS service already binding to host port 5001.
- Wait strategy: Use `wait.ForLog("API server listening on")` with `WithStartupTimeout(90 * time.Second)` — NOT HTTP health check. The Kubo daemon's HTTP API endpoint `/api/v0/id` may respond before full initialization completes. Log-based wait ensures the daemon is fully ready. Add a retry wrapper around the first `NodeID()` call as secondary verification.
- Return `*ipfs.Client` pre-configured with the container's dynamically allocated API endpoint
- Handle Docker unavailability gracefully using existing `isDockerUnavailablePanic` / `isDockerUnavailable` pattern from `tests/integration/containers/docker_availability.go` — do NOT use `testing.Short()` at the container-helper level
- Use `//go:build integration` build tag

**Definition of Done:**

- [ ] `tests/integration/containers/ipfs.go` exists with `NewIPFSContainer()` function
- [ ] Returns struct with `Container`, `Client *ipfs.Client`, `APIEndpoint string`
- [ ] Uses automatic port allocation (no fixed host port binding)
- [ ] Uses log-based wait strategy (`wait.ForLog("API server listening on")`)
- [ ] Container starts and `NodeID()` returns a valid peer ID
- [ ] Cleanup via `t.Cleanup()` terminates container
- [ ] Build tag `//go:build integration` present
- [ ] Tests skip gracefully when Docker is unavailable (using `isDockerUnavailablePanic` pattern)

**Verify:**

- `go build -tags integration ./tests/integration/containers/` — compiles
- `go test -tags integration -run TestIPFSContainer -v ./tests/integration/...` — container starts and NodeID returns valid peer ID

---

### Task 2: Write IPFS Client Integration Tests

**Objective:** Write integration tests for the IPFS client against a real Kubo node, covering Add, Get, Pin, Unpin, IsPinned, Delete, Stat, NodeID operations.

**Dependencies:** Task 1

**Files:**

- Create: `tests/integration/ipfs_client_test.go`

**Key Decisions / Notes:**

- Use `//go:build integration` build tag
- Use `containers.NewIPFSContainer()` from Task 1
- Test the full Add → Get → Pin → IsPinned → Unpin → Delete lifecycle
- Verify content integrity: add bytes, get bytes, compare
- Test CID validation with real CIDs returned from Kubo
- Test error cases: Get non-existent CID, Unpin non-pinned content
- Test concurrent operations (multiple adds in parallel)
- Follow table-driven test pattern from `tests/CLAUDE.md`
- **NodeID test** verifies the container is reachable and returns a valid peer ID (confirms testcontainer started correctly)
- **Stat test** verifies content metadata (size, cumulative size) is accurate after Add — these are client-layer tests independent of the orchestrator interface

**Definition of Done:**

- [ ] Integration test file exists with `//go:build integration` tag
- [ ] Tests cover: Add, AddBytes, Get, GetBytes, Pin, Unpin, IsPinned, Delete, Stat, NodeID
- [ ] Content integrity verified (add → get → compare bytes match)
- [ ] Error cases tested (non-existent CID, invalid CID)
- [ ] All tests pass against real Kubo container
- [ ] Tests skip gracefully when Docker unavailable

**Verify:**

- `go test -tags integration -run TestIPFS -v ./tests/integration/` — all IPFS client tests pass

---

### Task 3: Write IPFS Orchestrator Integration Tests

**Objective:** Integration tests for the storage orchestrator using local storage + real IPFS container. Tests verify `primary_only` and `dual_sync` modes work correctly. `dual_async` mode is documented as NOT writing to IPFS (code placeholder).

**Dependencies:** Task 1

**Files:**

- Create: `tests/integration/orchestrator_test.go`

**Key Decisions / Notes:**

- Use local storage (temp dir via `t.TempDir()`) as primary + real IPFS container
- **Critical: `dual_async` does NOT write to IPFS.** Reading `orchestrator.go` lines 89-99: the `Put()` method only enters `putDualSync()` when mode is `ModeDualSync`. For `ModeDualAsync`, it falls through to primary-only (`o.primary.Put()`). There is no goroutine spawned for async IPFS writes. Tests must reflect actual behavior, not aspirational behavior.
- Test `ModePrimaryOnly`: verify content in primary, no IPFS interaction
- Test `ModeDualSync`: verify content in both primary and IPFS after Put. Since `putDualSync()` (line 118) silently discards the CID from `AddBytes()`, verify IPFS content by computing the expected CID deterministically (IPFS CIDs are content-addressed — same data produces same CID) and checking `IsPinned(expectedCID)` or re-adding via direct client and comparing CIDs.
- Test `ModeDualAsync`: verify it behaves identically to primary-only (content in primary, NOT in IPFS). Add a comment explaining this is a known code gap.
- Test fallback: when `FallbackEnabled=true`, add content to IPFS directly via client, then attempt Get through orchestrator with a key that looks like a CID — should fall back to IPFS.
- Follow orchestrator interface from `internal/infrastructure/storage/orchestrator/orchestrator.go`

**Definition of Done:**

- [ ] Integration test file exists with `//go:build integration` tag
- [ ] Tests cover `ModePrimaryOnly`, `ModeDualSync`, `ModeDualAsync`
- [ ] `ModeDualSync` verified: content exists in primary AND IPFS after Put (CID verified via direct client call)
- [ ] `ModeDualAsync` verified: content exists in primary only, NOT in IPFS (documents known code gap)
- [ ] Fallback read from IPFS tested when primary returns error for CID-like key
- [ ] All tests pass

**Verify:**

- `go test -tags integration -run TestOrchestrator -v ./tests/integration/` — all orchestrator tests pass

---

### Task 4: Create SMTP Testcontainer Helper + Integration Tests

**Objective:** Create a reusable SMTP testcontainer using Mailpit (modern MailHog replacement) and write integration tests for the `SMTPSender`.

**Dependencies:** None

**Files:**

- Create: `tests/integration/containers/smtp.go`
- Create: `tests/integration/smtp_test.go`

**Key Decisions / Notes:**

- Use `mailpit/mailpit:latest` Docker image — lightweight SMTP server with REST API for inspecting messages
- Expose SMTP port 1025 and HTTP API port 8025 (both with automatic random port allocation)
- Wait strategy: `wait.ForLog("smtp server listening")` with 30s timeout
- Return struct with `SMTPPort`, `APIPort`, `Container`, cleanup
- Integration tests verify:
  - `Send()` delivers email to Mailpit's SMTP server
  - Use Mailpit's REST API (`GET /api/v1/messages`) to verify the email was received
  - Verify email content: From, To, Subject, HTML body, plain text body
  - Test `SendNewFollowerEmail()` and `SendMalwareDetectedEmail()` convenience methods
  - Test rate limiting: send more than configured rate limit, verify `ErrRateLimited` returned
  - Test TLS and plain modes
- Follow existing testcontainer pattern from `tests/integration/containers/postgres.go`

**Definition of Done:**

- [ ] `tests/integration/containers/smtp.go` exists with `NewSMTPContainer()` function
- [ ] Returns struct with `Container`, `SMTPPort string`, `APIEndpoint string`
- [ ] Uses automatic port allocation
- [ ] Integration test sends email and verifies receipt via Mailpit REST API
- [ ] Tests cover: Send, SendNewFollowerEmail, SendMalwareDetectedEmail
- [ ] Rate limiting tested
- [ ] Build tag `//go:build integration` present
- [ ] Tests skip gracefully when Docker unavailable

**Verify:**

- `go build -tags integration ./tests/integration/containers/` — compiles
- `go test -tags integration -run TestSMTP -v ./tests/integration/` — all SMTP tests pass

---

### Task 5: Create S3/MinIO Shared Testcontainer Helper

**Objective:** Extract the inline MinIO testcontainer setup from `s3_integration_test.go` into a shared reusable helper in `tests/integration/containers/`, consistent with the Postgres/Redis pattern.

**Dependencies:** None

**Files:**

- Create: `tests/integration/containers/minio.go`
- Modify: `internal/infrastructure/storage/s3/s3_integration_test.go` (refactor to use shared container)

**Key Decisions / Notes:**

- The existing `s3_integration_test.go` already has a working inline `setupMinIO()` function (lines 28-78) — extract and generalize it
- Use `minio/minio:RELEASE.2024-10-13T13-34-11Z` image (matches `docker/docker-compose.yml`)
- Return struct with `Container`, `Endpoint string`, `AccessKey string`, `SecretKey string`, `BucketName string`
- Auto-create the test bucket via MinIO client or mc CLI in container
- Wait strategy: `wait.ForLog("API:")` with 60s timeout (matches existing pattern)
- Refactor `s3_integration_test.go` to use `containers.NewMinIOContainer()` instead of inline setup
- Verify existing S3 tests still pass after refactoring

**Definition of Done:**

- [ ] `tests/integration/containers/minio.go` exists with `NewMinIOContainer()` function
- [ ] Returns struct with connection details matching S3 client Config
- [ ] Uses automatic port allocation
- [ ] `s3_integration_test.go` refactored to use shared container helper
- [ ] All existing S3 integration tests still pass after refactoring
- [ ] Build tag `//go:build integration` present
- [ ] Tests skip gracefully when Docker unavailable

**Verify:**

- `go build -tags integration ./tests/integration/containers/` — compiles
- `go test -tags integration -run TestS3 -v ./internal/infrastructure/storage/s3/` — existing S3 tests pass with shared container

---

### Task 6: Verify Image Download Endpoint

**Objective:** Confirm whether `GET /images/{id}/download` exists in the router. If missing, implement it.

**Dependencies:** None

**Files:**

- Read: `internal/interfaces/http/handlers/router.go` — check for download route
- Read: `internal/interfaces/http/handlers/image_handler.go` — check for download handler
- If missing:
  - Modify: `api/openapi/openapi.yaml` (add download path)
  - Modify: `internal/interfaces/http/handlers/image_handler.go` (add DownloadImage handler)
  - Modify: `internal/interfaces/http/handlers/router.go` (add route)

**Key Decisions / Notes:**

- MVP spec defines `GET /images/{id}/download` for original image download
- The variant endpoint `/images/{id}/variants/{size}` exists but may not serve the original
- If endpoint exists: mark gap as resolved, update gap analysis table
- If endpoint is missing: implement following existing handler patterns. The handler should set `Content-Disposition: attachment` header and stream the original image from storage.

**Definition of Done:**

- [ ] Status of `GET /images/{id}/download` confirmed (exists or implemented)
- [ ] If implemented: OpenAPI spec updated, handler follows existing patterns, route added
- [ ] Gap analysis table in this plan updated with resolved status

**Verify:**

- `grep -n "download" internal/interfaces/http/handlers/router.go` — confirms route presence
- If implemented: `make validate-openapi` — spec valid

---

### Task 7: Implement Password Reset Flow

**Objective:** Add password reset endpoints (forgot password + reset password) following existing auth handler patterns and RFC 7807 error responses.

**Dependencies:** None (can start before Task 4/5 if email is logged instead of sent)

**Files:**

- Modify: `api/openapi/openapi.yaml` (add password reset paths/schemas)
- Create: `internal/application/identity/email_sender.go` (EmailSender interface)
- Create: `internal/application/identity/commands/request_password_reset.go`
- Create: `internal/application/identity/commands/reset_password.go`
- Modify: `internal/domain/identity/repository.go` (add `TokenRepository` interface)
- Create: `internal/infrastructure/persistence/postgres/token_repository.go`
- Modify: `internal/infrastructure/email/smtp.go` (add `SendPasswordResetEmail` method)
- Modify: `internal/interfaces/http/handlers/auth_handler.go` (add ForgotPassword + ResetPassword)
- Modify: `internal/interfaces/http/handlers/router.go` (add routes)
- Modify: `cmd/api/main.go` (wire TokenRepository, EmailSender, new command handlers)
- Create: `migrations/00022_create_password_reset_tokens.sql` (verify migration number at implementation time — `ls migrations/ | tail -1` first)
- Create: `internal/application/identity/commands/request_password_reset_test.go`
- Create: `internal/application/identity/commands/reset_password_test.go`

**Key Decisions / Notes:**

- `POST /auth/forgot-password` — accepts email, always returns 200 (prevents enumeration)
- `POST /auth/reset-password` — accepts token + new password
- **Token storage:** Separate `password_reset_tokens` table (see Shared Design section above)
- **Concurrent request policy:** DELETE all previous unused tokens for the user before inserting new one (prevents multiple valid tokens)
- Rate limit: 3 reset requests per hour per email (in router)
- Token is single-use — marked as used (`used_at = NOW()`) on successful reset
- Token expiry: 1 hour
- Invalidate all existing sessions on password reset
- **EmailSender interface:** Create `internal/application/identity/email_sender.go` defining `SendPasswordResetEmail(ctx, email, token) error`. The command handler accepts this as a dependency. When SMTP is disabled, `SMTPSender.Send()` returns `ErrSMTPDisabled` — the handler logs a warning and continues (the request does not fail). This allows development without SMTP configured.
- Follow command handler pattern from `internal/application/identity/commands/register_user.go`
- Anti-enumeration: same response whether email exists or not, use constant-time comparison for tokens

**Definition of Done:**

- [ ] `POST /api/v1/auth/forgot-password` returns 200 for any email
- [ ] `POST /api/v1/auth/reset-password` validates token and resets password
- [ ] Token expires after 1 hour
- [ ] Token is single-use (used_at set on consumption)
- [ ] Concurrent requests: previous unused tokens deleted before new token created
- [ ] Unit test asserts that `ResetPasswordHandler.Handle()` calls `TokenRepository.InvalidateAllForUser()` and session invalidation
- [ ] Rate limited (3/hour per email hash)
- [ ] OpenAPI spec updated and validates
- [ ] Unit tests for command handlers (85% coverage)
- [ ] No account enumeration via timing or response differences
- [ ] `EmailSender` interface created in application layer (no infrastructure imports)
- [ ] Postman E2E collection updated with test scripts for new endpoints (tracked in Task 9)

**Verify:**

- `make validate-openapi` — spec valid
- `go test -run TestPasswordReset ./internal/application/identity/commands/` — command tests pass
- `make lint && make test` — full suite passes

---

### Task 8: Implement Email Verification Flow

**Objective:** Add email verification endpoints so new users must verify their email before full account access.

**Dependencies:** Task 7 (shares TokenRepository interface and EmailSender interface)

**Files:**

- Modify: `api/openapi/openapi.yaml` (add verification paths/schemas)
- Create: `internal/application/identity/commands/send_verification_email.go`
- Create: `internal/application/identity/commands/verify_email.go`
- Modify: `internal/domain/identity/user.go` (add `EmailVerified bool`, `EmailVerifiedAt *time.Time` fields + `VerifyEmail()` method)
- Modify: `internal/domain/identity/user.go` — update `ReconstructUser()` and `ReconstructUserWith2FA()` function signatures to include `emailVerified` parameter
- Modify: `internal/infrastructure/persistence/postgres/user_repository.go` (update `userModel` struct with `email_verified` and `email_verified_at` fields, update `toDomain()` and `fromDomain()` mapping, update all SELECT queries)
- Modify: `internal/infrastructure/email/smtp.go` (add `SendVerificationEmail` method)
- Modify: `internal/interfaces/http/handlers/auth_handler.go` (add VerifyEmail + ResendVerification)
- Modify: `internal/interfaces/http/handlers/router.go` (add routes)
- Modify: `cmd/api/main.go` (wire dependencies)
- Create: `migrations/00023_add_email_verification.sql` (verify migration number at implementation time)
- Create: `internal/application/identity/commands/verify_email_test.go`
- Modify: `tests/integration/containers/postgres.go` — add `email_verification_tokens` to the table truncation list in Cleanup()

**Key Decisions / Notes:**

- `POST /auth/verify-email` — accepts verification token
- `POST /auth/resend-verification` — resends verification email (authenticated, rate limited)
- Migration adds `email_verified BOOLEAN DEFAULT FALSE` and `email_verified_at TIMESTAMPTZ` to users table, plus creates `email_verification_tokens` table
- Verification token: UUID, 24-hour expiry, uses `TokenRepository` from Task 7
- On registration: auto-send verification email (modify existing `RegisterUserHandler` to call `EmailSender.SendVerificationEmail()`)
- **Unverified user enforcement:** Unverified users can still login. Enforcement is at handler level, not middleware. Specifically: image upload (`POST /images`), comment creation (`POST /images/{id}/comments`), and group creation (`POST /groups`) handlers check `user.EmailVerified` and return 403 with detail "Email verification required" if false. Other endpoints (viewing, browsing, profile) remain accessible.
- **Cascading impact:** Adding `email_verified` to User struct changes `ReconstructUser()` and `ReconstructUserWith2FA()` signatures. All callers in `user_repository.go` must be updated. Existing users in DB get `email_verified = FALSE` from migration default — this is safe since enforcement only limits specific write operations.
- Follow existing auth handler patterns in `auth_handler.go`

**Definition of Done:**

- [ ] `POST /api/v1/auth/verify-email` verifies email with token
- [ ] `POST /api/v1/auth/resend-verification` resends (rate limited, authenticated)
- [ ] Verification token expires after 24 hours
- [ ] Token is single-use
- [ ] Migration adds `email_verified` column to users and creates `email_verification_tokens` table
- [ ] Domain entity has `VerifyEmail()` method and `EmailVerified` field
- [ ] `ReconstructUser()` and `ReconstructUserWith2FA()` updated with `emailVerified` parameter
- [ ] `user_repository.go` model and mappings updated
- [ ] Upload/comment/group-create handlers return 403 for unverified users
- [ ] OpenAPI spec updated and validates
- [ ] Unit tests for command handlers
- [ ] No account enumeration
- [ ] Postman E2E collection updated (tracked in Task 9)

**Verify:**

- `make validate-openapi` — spec valid
- `go test -run TestVerifyEmail ./internal/application/identity/commands/` — tests pass
- `make migrate-up` — migration applies cleanly
- `make lint && make test` — full suite passes

---

### Task 9: Update Postman E2E Collection for Auth Endpoints

**Objective:** Add Postman test requests for all new auth endpoints (password reset and email verification) to the E2E test collection.

**Dependencies:** Task 7, Task 8

**Files:**

- Modify: `tests/e2e/postman/goimg-api.postman_collection.json`
- Modify: `tests/e2e/postman/ci.postman_environment.json` (if new env vars needed)

**Key Decisions / Notes:**

- Add requests for: `POST /auth/forgot-password`, `POST /auth/reset-password`, `POST /auth/verify-email`, `POST /auth/resend-verification`
- Each request must have test scripts validating:
  - Response status codes (200 for forgot-password, 200 for successful reset, 400 for invalid token, 429 for rate limited)
  - Response body structure (RFC 7807 for errors)
  - Anti-enumeration: forgot-password returns 200 for non-existent email too
- Follow existing collection patterns (Auth folder structure)
- E2E tests for email verification may need to either: (a) use a test token directly via DB setup, or (b) skip the full flow if no SMTP is configured in CI. Document the approach chosen.

**Definition of Done:**

- [ ] Postman collection contains requests for all 4 new auth endpoints
- [ ] Each request has test scripts validating status codes and response body structure
- [ ] Error cases included (invalid token, expired token, rate limited)
- [ ] `make test-e2e-dry` passes (JSON structure valid)
- [ ] `make test-e2e` runs without collection errors (when API is running)

**Verify:**

- `make test-e2e-dry` — validates collection JSON structure
- `make agent-check` — full validation passes

---

## Testing Strategy

- **Unit tests:** Mock repositories for command handlers (password reset, email verification). Mock EmailSender interface.
- **Integration tests:**
  - Real IPFS Kubo container for IPFS client + orchestrator (Tasks 2, 3)
  - Real Mailpit container for SMTP sender (Task 4)
  - Real MinIO container for S3 storage (Task 5)
  - All via testcontainers-go
- **E2E tests:** Postman requests for password reset and email verification endpoints (Task 9)
- **Build tag:** All integration tests use `//go:build integration`

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| IPFS container slow to start | Medium | Low | Use log-based wait strategy (`wait.ForLog("API server listening on")`) with 90s startup timeout. Add retry wrapper on first NodeID() call as secondary verification. |
| Password reset token timing attacks | Low | High | Use constant-time comparison for tokens (`crypto/subtle.ConstantTimeCompare`), same response for all emails regardless of existence. |
| Email verification blocks users | Medium | Medium | Unverified users can still login. Only upload, comment creation, and group creation are restricted. |
| Docker unavailable in CI | Low | Medium | Tests skip gracefully using the existing `isDockerUnavailablePanic()`/`isDockerUnavailable()`/`skipDockerUnavailable()` pattern from `tests/integration/containers/docker_availability.go` — identical to postgres.go and redis.go. |
| Port conflict with docker-compose | Medium | Medium | All testcontainers use automatic random port allocation (no fixed host port bindings). Retrieve mapped port dynamically after container start. |
| Migration number conflict with parallel branches | Low | Medium | Check actual migration state (`ls migrations/ | tail -1`) at implementation time. Use sequential numbers based on what exists then, not what exists at plan time. |
| ReconstructUser() signature change cascading | Medium | Medium | Update all callers in user_repository.go when adding email_verified field. Run full test suite after to catch any missed callers. |

### Deferred Ideas

**Discovered by expert analysis — not in scope for this plan but tracked for future sprints:**

**P1 (High value, next sprint candidates):**
- EXIF/metadata extraction on upload (Flickr stores camera model, lens, GPS, ISO, etc.)
- NSFW self-declaration toggle on upload (user marks own content as sensitive)
- `GET /images/{id}/likes` — list users who liked an image (endpoint missing)
- Album image reorder (drag-and-drop ordering)
- Change password endpoint (`POST /auth/change-password` — distinct from reset)
- View count race condition — current increment is not atomic
- Report system limited to images only — should support comments/users/groups

**P2 (Medium value):**
- Comment editing (`PUT /comments/{id}` — only create/delete exist)
- Popular period filtering is a stub (hardcoded, not time-window based)
- User profile missing social counters (total followers, following, images, likes)
- User search endpoint (`GET /users?q=...`)
- Bulk operations (batch delete, move, visibility change)
- Upload with `album_id` param (currently two-step: upload then add to album)
- Unlisted visibility enforcement verification

**P3 (Lower priority):**
- Signed/presigned URLs for private image access
- CDN integration (CloudFront/Cloudflare)
- @mentions in comments with notification
- oEmbed discovery tags in image HTML previews
- Watermarking support
- Advanced search (date range, dimensions, file size filters)
- Implement actual `dual_async` IPFS writes (currently a code placeholder)

**Testing coverage gaps (from test architecture analysis):**
- Application layer at ~40% coverage (target 80%) — needs unit test push
- HTTP handlers at ~11% coverage — needs handler-level tests
- No ClamAV integration tests
- No notification service tests
- Missing integration tests for: moderation, community, follow/social repositories
