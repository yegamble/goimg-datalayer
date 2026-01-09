# Sprint 5 Mid-Sprint Coordination Plan

## Executive Summary

**Sprint Goal**: Complete Domain & Infrastructure for Gallery Context (Image Processing, Storage Integration, ClamAV)

**Current Status**: Pre-sprint checkpoints approved. Core infrastructure (storage, ClamAV, validators) complete. 5 critical tasks remain with significant parallelization opportunities.

**Timeline**: Remaining 5-7 days to complete remaining deliverables

---

## Remaining Deliverables Overview

| Task | Complexity | Dependencies | Estimated Days | Parallel Slot |
|------|-----------|--------------|-----------------|---------------|
| Image Processor with bimg | High | Storage providers (done) | 2-3 | A |
| ImageRepository Implementation | High | Image domain (done), Image Processor | 2 | B |
| AlbumRepository Implementation | Medium | Album domain (done) | 1.5 | B |
| Storage Integration Tests | High | Storage providers (done) | 2 | C |
| Security Test Suite | High | Image Processor (for integration) | 2 | C |

**Key Insight**: All 5 tasks can execute in parallel with minimal blocking. Critical path: Image Processor → ImageRepository integration tests

---

## Detailed Agent Assignments & Task Breakdown

### TASK 1: Image Processor with bimg (Parallel Slot A)

**Primary Agent**: `senior-go-architect`
**Secondary Agents**: `image-gallery-expert` (validation), `backend-test-architect` (testing)
**Priority**: P0 (critical path blocker)
**Duration**: 2-3 days
**Dependencies**: Storage providers (complete), Image domain entity (complete)

#### Task Description

Implement comprehensive image processing pipeline using bimg (libvips bindings) with:
- Variant generation: thumbnail (160px), small (320px), medium (800px), large (1600px)
- WebP output format with quality 82-88%
- EXIF metadata stripping
- Memory limits (256MB cache)
- Worker pool pattern (max 32 concurrent processors)

#### Deliverables

1. **Image Processor Service** (`internal/infrastructure/imaging/processor.go`)
   - `NewImageProcessor(maxConcurrent int, maxMemory int64) (*Processor, error)`
   - `Process(ctx context.Context, img []byte) (*ProcessedImage, error)`
   - `ProcessVariant(ctx context.Context, variant VariantType, img []byte) ([]byte, error)`
   - Worker pool implementation with graceful shutdown
   - Memory limit enforcement (panic recovery on overflow)

2. **Variant Definition** (`internal/infrastructure/imaging/variants.go`)
   ```go
   type VariantType string
   const (
       VariantThumbnail VariantType = "thumbnail"  // 160px max
       VariantSmall     VariantType = "small"      // 320px max
       VariantMedium    VariantType = "medium"     // 800px max
       VariantLarge     VariantType = "large"      // 1600px max
   )
   ```

3. **EXIF Stripper** (`internal/infrastructure/imaging/exif.go`)
   - Remove all metadata before re-encoding
   - Validate EXIF removal in tests

#### Acceptance Criteria

- [ ] bimg integration working with Docker libvips
- [ ] All 4 variants generated correctly from test images
- [ ] WebP quality set to 82-88% and validated
- [ ] EXIF metadata completely stripped (verified with image inspection)
- [ ] Worker pool prevents > 32 concurrent processes
- [ ] Memory limits enforced (no OOM on edge cases)
- [ ] 10MB image processed in < 30 seconds
- [ ] Unit tests: 90% coverage on processor module
- [ ] Integration tests with sample images (JPEG, PNG, GIF, WebP)

#### Review Requirements

- Code Review: `senior-go-architect` (performance, error handling)
- Security Review: `senior-secops-engineer` (file handling, DoS prevention)
- Test Review: `backend-test-architect` (coverage, edge cases)

#### Key Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| libvips memory leaks | High | Memory profiling in tests, explicit cleanup, goroutine limits |
| EXIF removal incomplete | High | Use image inspection tools in tests, verify re-encoding prevents polyglots |
| WebP quality variance | Medium | Quality testing suite, visual validation of variants |
| Docker container resource issues | Medium | Set resource limits in docker-compose, test with 256MB limit |

---

### TASK 2: ImageRepository Implementation (Parallel Slot B, Task 1)

**Primary Agent**: Direct implementation (domain expertise required)
**Secondary Agents**: `backend-test-architect` (test design), `senior-go-architect` (review)
**Priority**: P0
**Duration**: 2 days
**Dependencies**: Image domain entity (complete), Image Processor (blocks integration tests)

#### Task Description

Implement PostgreSQL repository for Image aggregate with proper transaction handling for image + variants as atomic unit.

#### Deliverables

1. **Repository Interface** (exists: `internal/domain/gallery/repository.go`)
   - Already defined in domain layer
   - Implement in `internal/infrastructure/persistence/postgres/image_repository.go`

2. **ImageRepository Implementation**
   ```go
   type PostgresImageRepository struct {
       db *sqlx.DB
       tx *sqlx.Tx  // Optional for transaction
   }

   func (r *PostgresImageRepository) Save(ctx context.Context, img *Image) error
   func (r *PostgresImageRepository) FindByID(ctx context.Context, id ImageID) (*Image, error)
   func (r *PostgresImageRepository) FindByOwner(ctx context.Context, ownerID UserID, pagination Pagination) ([]*Image, error)
   func (r *PostgresImageRepository) Delete(ctx context.Context, id ImageID) error
   ```

3. **Transaction Handling**
   - Save image + all variants as single transaction
   - Rollback on variant save failure
   - Proper error mapping to domain errors

#### Acceptance Criteria

- [ ] All ImageRepository interface methods implemented
- [ ] Image + variants saved as atomic transaction
- [ ] Proper error mapping (ErrImageNotFound, ErrOwnerMismatch, etc.)
- [ ] Pagination working correctly
- [ ] Concurrent access handled safely (connection pooling)
- [ ] Integration tests with PostgreSQL testcontainer passing
- [ ] 85%+ test coverage
- [ ] No race conditions (go test -race)

#### Integration Test Requirements

- [ ] Save and retrieve image with variants
- [ ] Multiple variants per image
- [ ] Pagination with cursor/offset
- [ ] Owner filtering
- [ ] Delete cascade to variants
- [ ] Concurrent saves (race detection)

#### Review Requirements

- Code Review: `senior-go-architect` (query patterns, error handling, transaction semantics)
- Test Review: `backend-test-architect` (integration test comprehensiveness, testcontainer setup)

---

### TASK 3: AlbumRepository Implementation (Parallel Slot B, Task 2)

**Primary Agent**: Direct implementation
**Secondary Agents**: `backend-test-architect` (test design), `senior-go-architect` (review)
**Priority**: P0
**Duration**: 1.5 days
**Dependencies**: Album domain entity (complete), optional: ImageRepository (for join tests)

#### Task Description

Implement PostgreSQL repository for Album entity with album-image association and ordering.

#### Deliverables

1. **AlbumRepository Implementation** (`internal/infrastructure/persistence/postgres/album_repository.go`)
   ```go
   func (r *PostgresAlbumRepository) Save(ctx context.Context, album *Album) error
   func (r *PostgresAlbumRepository) FindByID(ctx context.Context, id AlbumID) (*Album, error)
   func (r *PostgresAlbumRepository) FindByOwner(ctx context.Context, ownerID UserID, pagination Pagination) ([]*Album, error)
   func (r *PostgresAlbumRepository) AddImage(ctx context.Context, albumID AlbumID, imageID ImageID, position int) error
   func (r *PostgresAlbumRepository) RemoveImage(ctx context.Context, albumID AlbumID, imageID ImageID) error
   func (r *PostgresAlbumRepository) GetImages(ctx context.Context, albumID AlbumID, pagination Pagination) ([]*Image, error)
   func (r *PostgresAlbumRepository) Delete(ctx context.Context, id AlbumID) error
   ```

2. **Album-Image Association**
   - Bidirectional querying (album's images, image's albums)
   - Position/ordering support
   - Cascade delete behavior

#### Acceptance Criteria

- [ ] All AlbumRepository interface methods implemented
- [ ] Album-image associations created correctly
- [ ] Position/ordering maintained
- [ ] Owner verification (can only modify own albums)
- [ ] Integration tests with PostgreSQL testcontainer
- [ ] 85%+ test coverage
- [ ] Pagination working for images within album
- [ ] No race conditions (go test -race)

#### Integration Test Requirements

- [ ] Create album and add images
- [ ] Retrieve images in album with correct ordering
- [ ] Move images between positions
- [ ] Remove image from album (doesn't delete image)
- [ ] Delete album (cascade rules)
- [ ] Pagination within album

#### Review Requirements

- Code Review: `senior-go-architect` (join query patterns, ownership checks)
- Test Review: `backend-test-architect` (association testing, cascade behavior)

---

### TASK 4: Storage Integration Tests (Parallel Slot C, Task 1)

**Primary Agent**: `backend-test-architect`
**Secondary Agents**: `cicd-guardian` (container management), `senior-go-architect` (review)
**Priority**: P0
**Duration**: 2 days
**Dependencies**: Storage providers complete (local, S3)

#### Task Description

Comprehensive integration tests for storage providers with real containers (local filesystem, MinIO S3).

#### Test Coverage

1. **Local Filesystem Provider Tests**
   - Create/read/delete operations
   - Directory structure validation
   - Concurrent access
   - Permission errors
   - Disk space errors (mocked)
   - Cleanup on delete

2. **S3-Compatible Provider Tests**
   - MinIO container setup and teardown
   - Create/read/delete operations
   - Bucket isolation
   - Concurrent operations
   - Connection timeout handling
   - Credentials validation
   - Region handling (if applicable)

3. **Error Handling Tests**
   - Provider unavailable
   - Permission denied
   - File not found
   - Corrupted data
   - Timeout recovery

4. **Performance Benchmarks**
   - Local: 10MB write < 500ms
   - S3: 10MB write < 5s (includes network)
   - Concurrent operations scaling

#### Deliverables

1. **Storage Test Suite** (`internal/infrastructure/storage/provider_test.go`)
   - Testcontainer-based integration tests
   - Benchmark tests
   - Error scenario coverage

#### Acceptance Criteria

- [ ] All storage provider methods tested
- [ ] Local FS provider: 85%+ coverage
- [ ] S3 provider: 85%+ coverage
- [ ] Error scenarios tested (10+ edge cases)
- [ ] Performance benchmarks document baseline
- [ ] MinIO container lifecycle managed correctly
- [ ] Concurrent access safety verified
- [ ] Tests deterministic (no flakiness)

#### Testcontainer Setup

```go
// Use testcontainers-go for MinIO
container, err := testcontainers.GenericContainer(ctx,
    testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image: "minio/minio:latest",
            Env: map[string]string{
                "MINIO_ROOT_USER": "minioadmin",
                "MINIO_ROOT_PASSWORD": "minioadmin",
            },
            ExposedPorts: []string{"9000/tcp"},
        },
        Started: true,
    })
```

#### Review Requirements

- Code Review: `senior-go-architect` (test patterns, container management)
- Test Review: `backend-test-architect` (coverage, reliability)
- Infrastructure Review: `cicd-guardian` (container orchestration, resource limits)

---

### TASK 5: Security Test Suite (Parallel Slot C, Task 2)

**Primary Agent**: `senior-secops-engineer` (design) → `backend-test-architect` (implementation)
**Secondary Agents**: `image-gallery-expert` (business logic validation)
**Priority**: P0
**Duration**: 2 days
**Dependencies**: Image Processor (complete), ClamAV integration (complete), Validator pipeline (complete)

#### Task Description

Implement comprehensive security test suite for image upload pipeline covering malware detection, polyglot prevention, and dimension limits.

#### Test Categories

1. **Malware Detection Tests**
   - EICAR test file detection (standard test malware)
   - ClamAV response validation
   - Signature update verification
   - False positive handling

2. **Polyglot File Prevention Tests**
   - JPEG+ZIP polyglot attempt (should fail)
   - PNG+executable polyglot attempt (should fail)
   - Re-encoded WebP validation (prevents original polyglot binary)
   - Magic number verification after re-encoding

3. **Dimension & Pixel Limit Tests**
   - Maximum dimensions: 8192x8192
   - Maximum pixels: 100M (e.g., 10000x10000)
   - Edge cases: 8192x8192 exactly (100M pixels exactly)
   - Oversized dimension rejection
   - Oversized pixel count rejection

4. **File Validation Tests**
   - Max file size: 10MB
   - MIME type sniffing (not extension)
   - Corrupted file handling
   - Truncated file handling
   - Empty file rejection

5. **Path Traversal Prevention Tests**
   - Storage paths use random UUIDs (no user input)
   - Validate no symlink attacks possible
   - Filename sanitization

6. **Rate Limiting Tests**
   - Upload rate limit: 50/hour per user
   - Concurrent upload limits
   - Rate limit headers present

#### Deliverables

1. **Security Test Suite** (`tests/security/image_security_test.go`)
   - Malware detection test fixtures
   - Polyglot test file generation
   - Dimension validation tests
   - Rate limiting validation

2. **Test Fixtures**
   - `tests/fixtures/eicar.txt` (EICAR test file)
   - `tests/fixtures/polyglot-jpeg-zip.bin` (polyglot test)
   - Various edge case images

#### Acceptance Criteria

- [ ] EICAR detection test passing (verified with ClamAV)
- [ ] Polyglot prevention verified (3+ test cases)
- [ ] Dimension limits enforced (8192x8192 max)
- [ ] Pixel limits enforced (100M max)
- [ ] File size limits enforced (10MB max)
- [ ] MIME type validation working (magic byte check)
- [ ] Path traversal prevention verified
- [ ] Rate limiting working under load
- [ ] 90%+ coverage for security validation code
- [ ] All tests document expected behavior

#### Test Execution

```bash
# Run security tests
go test -v -timeout 5m ./tests/security/...

# With coverage
go test -v -cover -coverprofile=security.out ./tests/security/...
```

#### Review Requirements

- Security Review: `senior-secops-engineer` (threat model validation, test adequacy)
- Code Review: `backend-test-architect` (test implementation, coverage)
- Domain Review: `image-gallery-expert` (business rule validation)

#### Security Checklist (Pre-Merge)

- [ ] ClamAV malware detection verified
- [ ] Image re-encoding prevents polyglot exploits
- [ ] Dimension/pixel limits enforced
- [ ] EXIF metadata fully stripped
- [ ] File size limits enforced
- [ ] MIME type validation uses content sniffing
- [ ] Rate limiting tested under concurrent load
- [ ] No hardcoded paths in storage operations

---

## Parallel Execution Schedule

```
TIMELINE (assuming start of remaining tasks as Day 1)

Week:   Day 1-2             Day 2-3             Day 3-4            Day 4-5
        (Slot A)            (Slot B)            (Slot C)           (Finalization)

SLOT A: Image Processor     [████████] ──→ Code Review
        [████████████████]                    Integration Tests
                                              [████████████]

SLOT B: ImageRepository     ────────→ [████████████████] Code Review ──→ Final
        AlbumRepository     [██████████████████] Testing
                            Integration Tests
                            [████████████████]

SLOT C: Storage Tests       ────────→ [████████████████] Code Review
        Security Tests      [████████████████] Implementation
                            [████████████████]

Key Points:
• Image Processor completes first (blocks ImageRepository integration tests)
• Repositories and Storage Tests can start day 1-2 (limited blocking)
• Security Tests start after Image Processor ready
• All code reviews run in parallel on day 3-4
• Final integration/merge validation day 4-5
```

---

## Status Report Template

Use this format for daily/mid-day updates:

```markdown
## Sprint 5 Status - [Date]

**Overall Sprint Progress**: [X]% complete

### Completed Since Last Update
- [x] Task 1: [agent name] - Specific completion
- [x] Task 2: [agent name] - Specific completion

### In Progress
- [ ] Task 3: [agent name] - [Y]% complete (est. [timeframe])
- [ ] Task 4: [agent name] - [Z]% complete (est. [timeframe])

### Blockers
- Blocker 1: [description] [status: OPEN/RESOLVED]
  - Required: [agent/resource]
  - Impact: [task blocked]
  - Mitigation: [plan]

### Risks
- Risk 1: [description] [probability: HIGH/MEDIUM/LOW]
  - Impact: [deliverable at risk]
  - Mitigation: [preventive action]

### Metrics
- Test coverage: [X]% (target: 80%)
- Domain coverage: [Y]% (target: 90%)
- Code review readiness: [Z]% of PR reviews completed

### Next 24 Hours
- [ ] Action 1 [owner: agent-name] [timeframe]
- [ ] Action 2 [owner: agent-name] [timeframe]

### Quality Gates Status
- [ ] CI pipeline: GREEN / YELLOW / RED
- [ ] Code quality: [status]
- [ ] Test coverage: [status]
- [ ] Security validation: [status]
```

---

## Quality Gate Checklist (Pre-Merge for Each Task)

### Code Quality Gates

```bash
# All tasks must pass these checks before merge
go fmt ./...                    # No formatting issues
go vet ./...                    # No vet warnings
golangci-lint run              # Linting clean
go test -race ./...            # No race conditions
go test -v -cover ./...        # Coverage check
make validate-openapi          # API spec consistency
```

### Coverage Requirements (Per Task)

| Task | Overall | Domain | Infrastructure | Test Coverage |
|------|---------|--------|-----------------|----------------|
| Image Processor | 85% | N/A | 90% | Unit + Integration |
| ImageRepository | 85% | N/A | 90% | Unit + Integration |
| AlbumRepository | 85% | N/A | 90% | Unit + Integration |
| Storage Tests | 85% | N/A | 90% | Integration only |
| Security Tests | N/A | N/A | N/A | Security coverage |

### Security Gate (Pre-Merge)

- [ ] No hardcoded credentials
- [ ] No commented code
- [ ] Input validation in place
- [ ] Error messages don't leak internals
- [ ] Dependency vulnerabilities checked (`nancy sleuth`)
- [ ] `gosec ./...` clean
- [ ] OWASP checks passed (per task)

### Documentation Requirements

- [ ] README/guide updated (if needed)
- [ ] Code comments on complex logic
- [ ] Function documentation (godoc format)
- [ ] Integration points documented
- [ ] Architecture decisions recorded (if applicable)

---

## Risk Management

### Critical Path Risk: Image Processor Delays

**Scenario**: Image Processor not ready by day 2
**Impact**: ImageRepository integration tests blocked (high priority)
**Mitigation**:
1. Pair programming: senior-go-architect + additional resource
2. Reduce variant count to 2 (thumbnail, large) for MVP
3. Defer WebP optimization to Sprint 6
4. Use mocked processor for initial repository tests

### Resource Risk: Test Coverage Gaps

**Scenario**: Coverage below 80% overall
**Impact**: CI/CD gate fails, merge blocked
**Mitigation**:
1. backend-test-architect does daily coverage review (day 2-3)
2. Identify gaps early and prioritize
3. Add focused test cases for missed paths
4. Adjust scope if needed (move items to Sprint 6)

### Integration Risk: Container Setup Failure

**Scenario**: Testcontainers or MinIO setup fails
**Impact**: Storage and Security tests cannot execute
**Mitigation**:
1. cicd-guardian validates Docker setup day 1
2. Pre-test container images available locally
3. Fallback to Docker Compose manual setup if needed
4. Document troubleshooting guide

---

## Daily Standup Questions

**For each agent daily:**

1. "What did you complete since last standup?"
2. "What are you working on today?"
3. "What blockers or risks have emerged?"
4. "Is your task on track for completion by [target date]?"
5. "Do you need help from another agent?"

**For scrum-master:**

1. "Are all 5 remaining tasks progressing?"
2. "Any cross-agent dependencies blocking work?"
3. "Coverage trajectory healthy?"
4. "Any quality gate concerns emerging?"
5. "Sprint goal still achievable?"

---

## Success Criteria (Sprint 5 Complete)

All of these must be true to mark Sprint 5 complete:

1. **Image Processor**
   - [ ] bimg integration working
   - [ ] 4 variants generated correctly
   - [ ] EXIF stripping verified
   - [ ] Performance targets met
   - [ ] Code review approved

2. **ImageRepository**
   - [ ] Full interface implemented
   - [ ] Integration tests passing
   - [ ] 85%+ coverage
   - [ ] Code review approved

3. **AlbumRepository**
   - [ ] Full interface implemented
   - [ ] Integration tests passing
   - [ ] 85%+ coverage
   - [ ] Code review approved

4. **Storage Integration Tests**
   - [ ] Local FS provider: 85%+ coverage
   - [ ] S3 provider: 85%+ coverage
   - [ ] Error scenarios tested
   - [ ] Performance baseline established

5. **Security Test Suite**
   - [ ] Malware detection verified
   - [ ] Polyglot prevention verified
   - [ ] All dimension/pixel limits tested
   - [ ] Rate limiting validated

6. **Overall Quality**
   - [ ] Overall test coverage >= 80%
   - [ ] Domain coverage >= 90%
   - [ ] CI/CD pipeline green
   - [ ] Security gates passed
   - [ ] No critical/high vulnerabilities
   - [ ] Sprint retrospective completed

---

## Next Sprint Preparation (Sprint 6)

**Dependencies from Sprint 5** (must complete):
- Image Processor (blocks upload handler)
- ImageRepository (blocks upload command)
- AlbumRepository (blocks album commands)

**Sprint 6 Blockers if Sprint 5 Incomplete**:
- Application layer commands depend on repositories
- HTTP handlers depend on all infrastructure
- E2E tests require processors + validators

**Recommended Sprint 5 Handoff**:
- Architecture decisions documented (libvips, variant strategy)
- Integration points clear (how handlers call repositories)
- Performance baselines established
- Known limitations documented (e.g., no concurrent uploads > 32)

---

## Contact & Escalation

**Scrum Master** (Overall Coordination):
- Task: Sprint 5 Coordination & Progress Tracking
- Escalation: Sprint blockers, resource conflicts, scope changes

**Agent Leads by Task**:
1. Image Processor: `senior-go-architect`
2. Repositories: Code team (direct implementation)
3. Storage Tests: `backend-test-architect`
4. Security Tests: `senior-secops-engineer` (design)

**Quality Gates**:
- Pre-merge reviews: Lead agent for each area
- Security gate: `senior-secops-engineer`
- Test coverage: `backend-test-architect`

---

## Version History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | 2025-12-04 | Initial Sprint 5 coordination plan | scrum-master |
