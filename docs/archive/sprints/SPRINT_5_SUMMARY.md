# Sprint 5 Quick Reference - Remaining Tasks

## Agent Assignments & Task Matrix

```
TASK                              AGENT(S)                          DURATION  PARALLEL
─────────────────────────────────────────────────────────────────────────────────────
1. Image Processor (bimg)         senior-go-architect (lead)        2-3 days  SLOT A
                                  + image-gallery-expert
                                  + backend-test-architect (review)

2. ImageRepository Impl.          Direct implementation              2 days    SLOT B-1
                                  + backend-test-architect (testing)
                                  + senior-go-architect (review)

3. AlbumRepository Impl.          Direct implementation              1.5 days  SLOT B-2
                                  + backend-test-architect (testing)
                                  + senior-go-architect (review)

4. Storage Integration Tests      backend-test-architect (lead)      2 days    SLOT C-1
                                  + cicd-guardian (support)
                                  + senior-go-architect (review)

5. Security Test Suite            senior-secops-engineer (design)    2 days    SLOT C-2
                                  + backend-test-architect (impl)
```

## Execution Timeline

```
DAY 1-2:    Parallel Start
            • Image Processor (A): Implementation phase
            • Repositories (B): Implementation phase
            • Storage Tests (C): Setup & baseline tests
            • Security Tests (C): Test design & fixtures

DAY 2-3:    Integration Testing
            • Image Processor (A): Integration tests
            • Repositories (B): Integration tests with processor
            • Storage/Security: Expand test coverage

DAY 3-4:    Code Reviews (All Parallel)
            • Image Processor (A): Security + code review
            • Repositories (B): Code review + coverage validation
            • Storage Tests (C): Review + performance baseline
            • Security Tests (C): Security review + threat validation

DAY 4-5:    Final Validation & Merge
            • All PR reviews complete
            • Coverage thresholds verified
            • Security gates passed
            • Sprint 5 merge ready
```

## Critical Dependencies

```
Image Processor (A)
    └─→ ImageRepository integration tests (B-1)
    └─→ Security Test Suite (C-2)

ImageRepository (B-1) + AlbumRepository (B-2)
    ↓
    Sprint 6 Application Layer (blocking dependency)
```

## Success Criteria by Task

### 1. Image Processor (bimg)
- [ ] 4 variants generated: thumbnail, small, medium, large
- [ ] WebP output at quality 82-88%
- [ ] EXIF metadata fully stripped
- [ ] 10MB image processed < 30 seconds
- [ ] Worker pool limits: max 32 concurrent
- [ ] Memory limits: 256MB cache
- [ ] 90%+ coverage on processor module
- [ ] Performance tested with multiple image formats

### 2. ImageRepository
- [ ] Full interface implemented in postgres
- [ ] Image + variants atomic transaction
- [ ] Pagination working (offset/cursor)
- [ ] Owner filtering correct
- [ ] Cascade delete on image delete
- [ ] 85%+ coverage
- [ ] No race conditions (go test -race)
- [ ] Concurrent access safe

### 3. AlbumRepository
- [ ] Full interface implemented in postgres
- [ ] Album-image associations working
- [ ] Ordering/position maintained
- [ ] Owner verification enforced
- [ ] Images within album paginated
- [ ] 85%+ coverage
- [ ] No race conditions
- [ ] Delete cascade correct

### 4. Storage Integration Tests
- [ ] Local FS provider: 85%+ coverage
- [ ] S3/MinIO provider: 85%+ coverage
- [ ] 10+ error scenarios tested
- [ ] Performance baseline: Local < 500ms, S3 < 5s for 10MB
- [ ] Concurrent operations verified
- [ ] Testcontainer lifecycle correct
- [ ] Tests deterministic (no flakiness)

### 5. Security Test Suite
- [ ] EICAR malware detection working
- [ ] 3+ polyglot prevention tests passing
- [ ] Dimension limits: max 8192x8192
- [ ] Pixel limits: max 100M pixels
- [ ] File size limit: max 10MB
- [ ] MIME type validation via magic bytes
- [ ] Path traversal prevention verified
- [ ] Rate limiting under concurrent load
- [ ] 90%+ coverage on validation code

## Quality Gates (All Tasks)

**Before Merge - Every Task Must Pass**:
```bash
go fmt ./...                    # Format check
go vet ./...                    # Vet analysis
golangci-lint run              # Linting
go test -race ./...            # Race detection
go test -v -cover ./...        # Coverage check
make validate-openapi          # OpenAPI validation
```

**Coverage Targets**:
- Overall: >= 80%
- Domain/Infrastructure: >= 90% (Image Processor)
- Application: >= 85% (Repositories)
- Tests: Deterministic, no flakiness

**Security Checklist**:
- [ ] No hardcoded secrets/credentials
- [ ] No commented code
- [ ] Input validation in place
- [ ] Error messages don't leak internals
- [ ] Dependencies vulnerability-free (nancy)
- [ ] gosec clean
- [ ] OWASP threats addressed

## Daily Standup Checklist

**Each agent answers**:
1. What did you complete since last standup?
2. What are you working on today?
3. What blockers/risks emerged?
4. On track for [target date]? YES/NO/AT_RISK
5. Need help from another agent? YES/NO

**Scrum Master tracks**:
1. All 5 tasks progressing on schedule?
2. Cross-agent blockers resolved?
3. Coverage trajectory healthy?
4. Any quality gate concerns?
5. Sprint goal still achievable?

## Risk Watch List

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Image Processor delay (libvips, memory) | CRITICAL | Pair programming, reduce scope to 2 variants |
| Coverage gaps emerge | HIGH | Daily review day 2-3, focus on security/validation |
| Testcontainer setup fails | HIGH | cicd-guardian validates day 1, fallback to manual |
| Race conditions in repos | MEDIUM | `go test -race` mandatory, code review focused |
| Storage tests flakiness | MEDIUM | Timeout tuning, network isolation verification |

## File Paths to Watch

**Image Processor**:
- `/home/user/goimg-datalayer/internal/infrastructure/imaging/processor.go`
- `/home/user/goimg-datalayer/internal/infrastructure/imaging/variants.go`

**Repositories**:
- `/home/user/goimg-datalayer/internal/infrastructure/persistence/postgres/image_repository.go`
- `/home/user/goimg-datalayer/internal/infrastructure/persistence/postgres/album_repository.go`

**Tests**:
- `/home/user/goimg-datalayer/internal/infrastructure/storage/provider_test.go`
- `/home/user/goimg-datalayer/tests/security/image_security_test.go`

## Handoff to Sprint 6

**Critical Outputs for Next Sprint**:
- Image Processor fully functional & documented
- ImageRepository interface complete & tested
- AlbumRepository interface complete & tested
- Integration test baselines established
- Security test suite green
- Performance benchmarks documented

**Known Limitations to Document**:
- Max 32 concurrent image processing jobs
- WebP quality range 82-88% (not configurable)
- Memory limit 256MB per processor
- S3 latency ~2-3s per operation

**Architecture Decisions Made**:
- libvips (bimg) for image processing
- PostgreSQL transactions for image + variants atomicity
- Worker pool pattern for processor concurrency
- Testcontainers for storage integration tests
- Security-first variant generation (re-encode prevents polyglots)

---

## Reference Documents

- Full plan: `/home/user/goimg-datalayer/SPRINT_5_COORDINATION.md`
- Sprint plan: `/home/user/goimg-datalayer/claude/sprint_plan.md`
- Agent workflow: `/home/user/goimg-datalayer/claude/agent_workflow.md`
- Test strategy: `/home/user/goimg-datalayer/claude/test_strategy.md`
- Agent checklist: `/home/user/goimg-datalayer/claude/agent_checklist.md`
