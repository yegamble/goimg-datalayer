# Sprint 8: Quick Reference Guide

> **For Agents**: Quick lookup for tasks, commands, and acceptance criteria during Sprint 8 execution.

---

## Sprint Goals (TL;DR)

1. ✅ **Testing**: Achieve 80%+ overall coverage (85% app, 75% handlers)
2. ✅ **Security**: Zero critical vulnerabilities, complete OWASP Top 10 tests, pentest
3. ✅ **Performance**: Establish baselines, optimize queries, implement caching

---

## Agent Quick Assignments

| Agent | Primary Focus | Key Deliverable |
|-------|---------------|-----------------|
| **backend-test-architect** | Unit + integration tests | 85%+ app coverage, 75%+ handler coverage |
| **test-strategist** | E2E + load tests | 100% endpoint coverage, performance baseline |
| **senior-secops-engineer** | Security tests + pentest | OWASP tests, pentest report, 0 criticals |
| **senior-go-architect** | Performance | Query optimization, cache strategy |
| **cicd-guardian** | CI/CD validation | Security scans verified |
| **scrum-master** | Coordination | Sprint tracking, quality gates |

---

## Daily Checklist (For All Agents)

- [ ] Update daily standup in Slack/tracking doc
- [ ] Commit code with tests
- [ ] Run `go test -race ./...` locally before push
- [ ] Update coverage metrics in tracking doc
- [ ] Report blockers immediately (don't wait 24h)

---

## Essential Commands

### Testing
```bash
# Unit tests (fast, no external deps)
go test -short -race -cover ./...

# Integration tests (with testcontainers)
go test -tags=integration -race -cover ./tests/integration/...

# Security tests
go test -race -cover ./tests/security/...

# E2E tests
newman run tests/e2e/postman/goimg-api.postman_collection.json \
  --environment tests/e2e/postman/ci.postman_environment.json

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

### Security Scanning
```bash
# Go security checker
gosec ./...

# Vulnerability scanner
trivy fs --severity HIGH,CRITICAL .

# Go vulnerability database
govulncheck ./...

# Secret detection
gitleaks detect --verbose
```

### Performance
```bash
# Database query analysis
PGOPTIONS="-c log_statement=all" go test ./...

# Load testing
k6 run tests/load/image_upload.js

# Benchmarks
go test -bench=. -benchmem ./...
```

---

## Coverage Targets (Quick Ref)

| Layer | Target | Command |
|-------|--------|---------|
| **Overall** | 80% | `go test -cover ./...` |
| **Domain** | 90% | `go test -cover ./internal/domain/...` (already 94.1%) |
| **Application** | 85% | `go test -cover ./internal/application/...` |
| **Handlers** | 75% | `go test -cover ./internal/interfaces/http/handlers/...` |
| **Infrastructure** | 70% | `go test -cover ./internal/infrastructure/...` |

---

## Acceptance Criteria (Quick Ref)

### Application Layer Tests
- [ ] All command handlers tested with mocks
- [ ] All query handlers tested with mocks
- [ ] Error propagation tested
- [ ] Domain event emission tested
- [ ] 85%+ coverage

### HTTP Handler Tests
- [ ] All endpoints return correct status codes
- [ ] All errors follow RFC 7807 format
- [ ] Auth middleware tested (401 scenarios)
- [ ] IDOR prevention tested
- [ ] Rate limiting tested (429 scenarios)
- [ ] 75%+ coverage

### Integration Tests
- [ ] All repositories tested with testcontainers
- [ ] CRUD operations tested
- [ ] Transaction rollback tested
- [ ] Concurrent access tested (race detector)
- [ ] Tests tagged with `//go:build integration`

### E2E Tests
- [ ] 100% endpoint coverage in Postman collection
- [ ] All requests have test scripts (status, schema, business rules)
- [ ] Error scenarios covered (4xx, 5xx)
- [ ] Newman runs successfully in CI

### Security Tests
- [ ] OWASP Top 10 test coverage (A01-A10)
- [ ] All tests passing
- [ ] Pentest report complete
- [ ] Zero critical/high vulnerabilities

### Performance
- [ ] All endpoints benchmarked (P50, P95, P99)
- [ ] Query optimization documented
- [ ] Cache strategy implemented
- [ ] Load tests passing (100 concurrent users)

---

## OWASP Top 10 Quick Ref

1. **A01: Broken Access Control** - Test IDOR, privilege escalation
2. **A02: Cryptographic Failures** - Verify Argon2id, RS256
3. **A03: Injection** - SQL, XSS, command, path traversal
4. **A04: Insecure Design** - Account lockout, rate limiting
5. **A05: Security Misconfiguration** - Security headers, error messages
6. **A07: Auth Failures** - Account enumeration, token replay
7. **A08: Data Integrity** - File upload validation, MIME types
8. **A09: Logging Failures** - Audit logs, no sensitive data
9. **A10: SSRF** - URL validation

---

## Test Structure Examples

### Application Layer Test Template
```go
func TestCommandHandler_Success(t *testing.T) {
    t.Parallel()

    mockRepo := new(MockRepository)
    handler := NewHandler(mockRepo)

    mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

    result, err := handler.Handle(context.Background(), command)

    require.NoError(t, err)
    assert.NotNil(t, result)
    mockRepo.AssertExpectations(t)
}
```

### HTTP Handler Test Template
```go
func TestHandler_Success(t *testing.T) {
    t.Parallel()

    mockService := new(MockService)
    handler := NewHandler(mockService)

    mockService.On("Execute", mock.Anything).Return(result, nil)

    req := httptest.NewRequest(http.MethodPost, "/endpoint", body)
    rec := httptest.NewRecorder()

    handler.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
    mockService.AssertExpectations(t)
}
```

### Integration Test Template
```go
func TestRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    suite := testhelpers.SetupTestSuite(t)
    repo := postgres.NewRepository(suite.DB)

    // Test logic here

    suite.CleanDatabase(t) // Cleanup
}
```

---

## Quality Gate Checklist (Pre-Merge)

**Automated** (must pass CI):
- [ ] `go test -race ./...` passes
- [ ] `golangci-lint run` passes
- [ ] `gosec ./...` zero critical/high
- [ ] `trivy fs .` zero critical
- [ ] `gitleaks detect` zero secrets
- [ ] Coverage >= 80% overall
- [ ] Domain coverage >= 90%
- [ ] Application coverage >= 85%
- [ ] Handler coverage >= 75%
- [ ] OpenAPI validation passes
- [ ] Newman E2E tests pass

**Manual** (scrum-master verification):
- [ ] Pentest report complete
- [ ] Security test suite passing
- [ ] Performance benchmarks established
- [ ] Load tests passing
- [ ] Rate limiting validated
- [ ] Token revocation verified
- [ ] Audit log completeness verified
- [ ] Agent checklist complete

---

## Blocker Escalation Path

1. **Self-resolve** (0-4 hours) - Try to fix independently
2. **Team help** (4-24 hours) - Ask in standup or Slack
3. **Scrum master** (24-48 hours) - Escalate to scrum-master
4. **Stakeholder** (48+ hours) - Critical path impact

---

## Common Pitfalls (Avoid These)

❌ **Don't**:
- Write integration tests without testcontainers
- Mock domain entities (test real domain logic)
- Skip error scenarios
- Test implementation details (test behavior)
- Commit code without running tests locally
- Ignore flaky tests (fix or disable)
- Copy-paste tests without understanding
- Use sleep() in tests (use channels/waitgroups)

✅ **Do**:
- Use table-driven tests
- Run tests with `-race` flag
- Use `t.Parallel()` for independent tests
- Use `t.Helper()` for test helpers
- Clean up resources (defer cleanup)
- Test edge cases
- Write descriptive test names
- Document complex test scenarios

---

## File Locations (Quick Ref)

```
tests/
├── unit/              # Pure unit tests (no external deps)
├── integration/       # Testcontainer-based tests
│   ├── containers/    # Testcontainer setup
│   └── fixtures/      # Test data
├── security/          # OWASP security tests
│   ├── owasp/         # OWASP Top 10 tests
│   └── fixtures/      # Malware samples, test files
├── e2e/               # End-to-end tests
│   └── postman/       # Newman/Postman collections
└── load/              # Load testing scripts (k6)

internal/
├── domain/            # Domain tests (90%+ coverage)
├── application/       # Application tests (85%+ coverage)
│   ├── commands/      # Command handler tests
│   └── queries/       # Query handler tests
├── infrastructure/    # Infrastructure tests (70%+ coverage)
└── interfaces/http/   # HTTP handler tests (75%+ coverage)
```

---

## Performance Targets (Quick Ref)

| Endpoint Type | Target P95 | Target P99 |
|---------------|------------|------------|
| Simple GET | < 50ms | < 100ms |
| Complex Query | < 200ms | < 500ms |
| POST/PUT | < 100ms | < 200ms |
| Image Upload | < 30s | < 60s |
| Search | < 200ms | < 500ms |

| Resource | Target |
|----------|--------|
| Cache Hit Rate | > 80% |
| DB Query Time | P95 < 50ms |
| Throughput | > 100 req/sec |
| Error Rate | < 0.1% |

---

## Key Documentation Files

- **Sprint Plan**: `/home/user/goimg-datalayer/claude/sprint_plan.md`
- **Sprint 8 Setup**: `/home/user/goimg-datalayer/claude/sprint_8_setup.md`
- **Sprint 8 Tracking**: `/home/user/goimg-datalayer/claude/sprint_8_tracking.md`
- **Test Strategy**: `/home/user/goimg-datalayer/claude/test_strategy.md`
- **Security Testing**: `/home/user/goimg-datalayer/claude/security_testing.md`
- **Agent Workflow**: `/home/user/goimg-datalayer/claude/agent_workflow.md`
- **Agent Checklist**: `/home/user/goimg-datalayer/claude/agent_checklist.md`

---

## Sprint 8 Success Definition

**Sprint is successful if**:
- All coverage targets met (80%/85%/75%)
- Zero critical vulnerabilities
- Penetration test complete
- Performance baselines established
- All quality gates passing
- Agent retrospective completed
- Sprint 9 handoff documented

---

**Quick Reference Version**: 1.0
**Last Updated**: 2025-12-04
**Purpose**: Fast lookup during Sprint 8 execution
