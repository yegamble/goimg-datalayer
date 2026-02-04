# Sprint 23: Test Coverage Improvement & Regression Prevention

> **Priority**: P0 CRITICAL
> **Duration**: 2 weeks
> **Goal**: Improve test coverage from ~65% to 80%+ and establish regression prevention practices
> **Status**: 🚧 IN PROGRESS

---

## Sprint Objectives

1. **Increase test coverage to 80%+ overall** (from ~65%)
2. **Close critical coverage gaps** in application layer and HTTP handlers
3. **Establish regression prevention** with local testing practices
4. **Update documentation** to reflect current state
5. **Ensure CI passes** on all workflow branches

---

## Coverage Gap Analysis

### Current State (as of 2026-01-15)

| Layer | Current | Target | Gap |
|-------|---------|--------|-----|
| **Overall** | ~65% | 80% | 15% |
| **Domain** | 90%+ | 90% | Met |
| **Application** | ~40% | 85% | 45% |
| **HTTP Handlers** | ~11% | 75% | 64% |
| **Infrastructure** | ~30% | 70% | 40% |

### Critical Gaps (Priority Order)

#### P0: Security-Critical (Week 1)

| Package | Files | Tests | Priority | Status |
|---------|-------|-------|----------|--------|
| `application/moderation` | 16 | 7 (new) | P0 | ✅ COMPLETE |
| `handlers/auth_handler` | 1 | 4 (new) | P0 | ✅ COMPLETE |
| `handlers/oauth_handler` | 1 | 3 (new) | P0 | ✅ COMPLETE |
| `handlers/twofa_handler` | 1 | 4 (new) | P0 | ✅ COMPLETE |

#### P1: Core Functionality (Week 1-2)

| Package | Files | Tests | Priority | Est. Effort |
|---------|-------|-------|----------|-------------|
| `application/community` | 29 | 1 | P1 | 3 days |
| `application/notification` | 4 | 0 | P1 | 0.5 day |
| `application/activity` | 2 | 0 | P1 | 0.5 day |
| `handlers/user_handler` | 1 | 0 | P1 | 0.5 day |
| `handlers/album_handler` | 1 | 0 | P1 | 0.5 day |

#### P2: Infrastructure (Week 2)

| Package | Files | Tests | Priority | Est. Effort |
|---------|-------|-------|----------|-------------|
| `postgres/*_repository` | 24 | 6 | P2 | 3 days |
| `infrastructure/email` | 4 | 0 | P2 | 0.5 day |
| `security/clamav` | 1 | 0 | P2 | 0.5 day |
| `jobs/tasks` | 3 | 0 | P2 | 0.5 day |

#### P3: Domain Completion (Week 2)

| Package | Files | Tests | Priority | Est. Effort |
|---------|-------|-------|----------|-------------|
| `domain/notification` | 6 | 0 | P3 | 0.5 day |

---

## Implementation Plan

### Week 1: Security & Critical Paths

#### Day 1-2: Moderation Application Layer Tests

**Status**: ✅ COMPLETE

**Files created:**
- `internal/application/moderation/commands/ban_user_test.go`
- `internal/application/moderation/commands/unban_user_test.go`
- `internal/application/moderation/commands/create_report_test.go`
- `internal/application/moderation/commands/resolve_report_test.go`
- `internal/application/moderation/commands/dismiss_report_test.go`
- `internal/application/moderation/commands/scan_image_nsfw_test.go`
- `internal/application/moderation/commands/start_review_test.go`
- `internal/application/moderation/queries/get_nsfw_scan_test.go`
- `internal/application/moderation/queries/list_active_bans_test.go`
- `internal/application/moderation/queries/list_nsfw_flagged_test.go`
- `internal/application/moderation/queries/list_nsfw_scans_by_image_test.go`

**Test scenarios for each:**
- Happy path with valid inputs
- Authorization checks (admin-only)
- Invalid input validation
- Not found errors
- Concurrent access handling

#### Day 3: Auth Handler Tests

**Status**: ✅ COMPLETE

**Files created:**
- `internal/interfaces/http/handlers/auth_handler_test.go`

**Test scenarios:**
- Register success/validation errors
- Login success/invalid credentials
- Refresh token flow
- Logout
- Rate limiting verification
- RFC 7807 error format

#### Day 4: OAuth & 2FA Handler Tests

**Status**: ✅ COMPLETE

**Files created:**
- `internal/interfaces/http/handlers/oauth_handler_test.go`
- `internal/interfaces/http/handlers/twofa_handler_test.go`

**Test scenarios:**
- OAuth initiate flow
- OAuth callback handling
- CSRF state validation
- 2FA setup/verify/disable
- Backup code regeneration

#### Day 5: Community Commands Tests (Part 1)

**Files to create:**
- `internal/application/community/commands/join_group_test.go`
- `internal/application/community/commands/leave_group_test.go`
- `internal/application/community/commands/invite_to_group_test.go`
- `internal/application/community/commands/accept_invitation_test.go`

### Week 2: Core & Infrastructure

#### Day 6-7: Community Completion

**Files to create:**
- Remaining community command tests
- `internal/application/community/queries/*_test.go` (all 10 queries)

#### Day 8: Notification & Activity Tests

**Files to create:**
- `internal/application/notification/commands/mark_as_read_test.go`
- `internal/application/notification/queries/get_notifications_test.go`
- `internal/application/notification/queries/get_unread_count_test.go`
- `internal/application/activity/queries/get_activity_feed_test.go`

#### Day 9: Core Handler Tests

**Files to create:**
- `internal/interfaces/http/handlers/user_handler_test.go`
- `internal/interfaces/http/handlers/album_handler_test.go`
- `internal/interfaces/http/handlers/moderation_handler_test.go`

#### Day 10: Infrastructure Tests

**Files to create:**
- `internal/infrastructure/email/smtp_test.go`
- `internal/infrastructure/email/rate_limiter_test.go`
- `internal/infrastructure/security/clamav/scanner_test.go`
- `internal/infrastructure/jobs/tasks/image_process_test.go`
- `internal/infrastructure/jobs/tasks/image_scan_test.go`

---

## Regression Prevention Practices

### Pre-Commit Checklist (MANDATORY)

```bash
# ALWAYS run before every commit
make pre-commit       # Runs go fmt, go vet, golangci-lint
make test             # Run full test suite
make validate-openapi # Validate API spec (if changed)
```

### Local Testing Strategy (Save CI Minutes)

**Before pushing ANY changes:**

1. **Run unit tests locally:**
   ```bash
   make test-unit
   ```

2. **Run domain tests with threshold:**
   ```bash
   make test-domain  # Enforces 90% coverage
   ```

3. **Run integration tests if DB changes:**
   ```bash
   docker-compose -f docker/docker-compose.yml up -d
   make test-integration
   ```

4. **Run E2E tests if API changes:**
   ```bash
   make run &  # Start server in background
   make test-e2e
   ```

### CI Workflow Verification

Test these locally before pushing to save GitHub Actions minutes:

| Workflow | Local Command | When to Run |
|----------|---------------|-------------|
| Lint | `make lint` | Every change |
| Unit Tests | `make test-unit` | Every change |
| Domain Tests | `make test-domain` | Domain changes |
| Integration | `make test-integration` | DB/repo changes |
| E2E | `make test-e2e` | API changes |
| OpenAPI | `make validate-openapi` | API spec changes |

---

## Go Version Requirements

**Current:** Go 1.25 with toolchain go1.25.5 (latest stable as of 2026-01-15)

**Important:**
- Go 1.25.5 released 2025-12-02 with security fixes
- Go 1.26 RC1 released 2025-12-16, expected stable Feb 2026
- **DO NOT downgrade Go version** - CI and local must use 1.25.x

**Verification:**
```bash
make check-go-version  # Must show >= 1.25
go version             # Should show go1.25.x
```

---

## Documentation Updates Required

### Sprint 23 Documentation Tasks

1. **README.md Roadmap** - Update Phase 3 progress
2. **NEXT_STEPS.md** - Update current sprint status
3. **Test Strategy** - Add regression prevention section
4. **API Documentation** - Verify OpenAPI is current

---

## Success Criteria

### Coverage Targets

| Metric | Target | Verification |
|--------|--------|--------------|
| Overall | 80%+ | `make test-coverage` |
| Domain | 90%+ | `make test-domain` |
| Application | 85%+ | Coverage report |
| Handlers | 75%+ | Coverage report |

### Quality Gates

- [ ] All CI workflows pass on feature branch
- [ ] All CI workflows pass on main after merge
- [ ] Zero lint errors (`make lint`)
- [ ] OpenAPI spec validates (`make validate-openapi`)
- [ ] E2E tests pass (`make test-e2e`)
- [ ] No Go version downgrades

### Deliverables

- [ ] 30+ new test files created
- [ ] 200+ new test cases
- [ ] Coverage increased to 80%+
- [ ] Documentation updated
- [ ] Regression prevention practices documented

---

## Test File Inventory (To Be Created)

### Application Layer (24 files)

```
internal/application/moderation/commands/
├── ban_user_test.go (Complete)
├── unban_user_test.go (Complete)
├── create_report_test.go (Complete)
├── resolve_report_test.go (Complete)
├── dismiss_report_test.go (Complete)
├── scan_image_nsfw_test.go (Complete)
└── start_review_test.go (Complete)

internal/application/moderation/queries/
├── get_report_test.go (Complete)
├── get_nsfw_scan_test.go (Complete)
├── get_user_ban_status_test.go (Complete)
├── list_pending_reports_test.go (Complete)
├── list_active_bans_test.go (Complete)
├── list_nsfw_flagged_test.go (Complete)
└── list_nsfw_scans_by_image_test.go (Complete)

internal/application/community/commands/
├── join_group_test.go
├── leave_group_test.go
├── invite_to_group_test.go
├── accept_invitation_test.go
├── decline_invitation_test.go
├── update_group_test.go
├── delete_group_test.go
├── ban_member_test.go
├── remove_member_test.go
├── update_member_role_test.go
├── share_image_to_group_test.go
├── approve_group_image_test.go
├── reject_group_image_test.go
├── create_group_album_test.go
├── update_group_album_test.go
├── delete_group_album_test.go
├── add_image_to_album_test.go
└── remove_image_from_album_test.go

internal/application/community/queries/
├── get_group_test.go
├── get_group_by_slug_test.go
├── get_group_album_test.go
├── list_public_groups_test.go
├── search_groups_test.go
├── list_group_members_test.go
├── list_user_groups_test.go
├── list_group_albums_test.go
├── list_approved_group_images_test.go
└── list_pending_group_images_test.go

internal/application/notification/
├── commands/mark_as_read_test.go
└── queries/
    ├── get_notifications_test.go
    └── get_unread_count_test.go

internal/application/activity/queries/
└── get_activity_feed_test.go
```

### HTTP Handlers (12 files)

```
internal/interfaces/http/handlers/
├── auth_handler_test.go (Complete)
├── oauth_handler_test.go (Complete)
├── twofa_handler_test.go (Complete)
├── user_handler_test.go
├── album_handler_test.go
├── moderation_handler_test.go
├── group_handler_test.go
├── group_album_handler_test.go
├── group_image_handler_test.go
├── notification_handler_test.go
├── activity_handler_test.go
└── tag_handler_test.go
```

### Infrastructure (8 files)

```
internal/infrastructure/
├── email/
│   ├── smtp_test.go
│   └── rate_limiter_test.go
├── security/clamav/
│   └── scanner_test.go
├── jobs/tasks/
│   ├── image_process_test.go
│   └── image_scan_test.go
└── persistence/postgres/
    ├── user_repository_test.go
    ├── image_repository_test.go
    └── album_repository_test.go
```

### Domain (2 files)

```
internal/domain/notification/
├── notification_test.go
└── notification_id_test.go
```

---

## Notes

- Use table-driven tests (`[]struct{...}`) for comprehensive coverage
- Mock external dependencies (DB, Redis, SMTP)
- Use `testify/assert` and `testify/require` for assertions
- Follow existing test patterns in the codebase
- Run `make pre-commit` before every commit
