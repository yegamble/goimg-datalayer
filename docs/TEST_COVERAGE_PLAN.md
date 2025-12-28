# Test Coverage Improvement Plan

**Project**: goimg-datalayer
**Current Coverage**: 40.3%
**Target Coverage**: 80%
**Gap**: 39.7 percentage points
**Document Version**: 1.0
**Last Updated**: 2025-12-28

---

## Executive Summary

### Current State

The goimg-datalayer project has strong test coverage in domain and application layers (90%+) but critical gaps in infrastructure and HTTP layers where security vulnerabilities and runtime failures are most likely.

**Coverage by Layer:**

| Layer | Current Coverage | Target Coverage | Status |
|-------|-----------------|-----------------|--------|
| Domain | 91-100% | 90% | ✅ Excellent |
| Application | 92-94% | 85% | ✅ Good |
| Infrastructure | 7.9-75.9% | 70% | ❌ Critical Gaps |
| HTTP/Handlers | 0-20% | 75% | ❌ Critical Gaps |
| **Overall** | **40.3%** | **80%** | **❌ Needs Work** |

### Critical Risk Areas

1. **HTTP Handlers (0% coverage)** - All API endpoints except `/health` have zero tests
   - Security Risk: Unvalidated auth flows, RBAC bypass potential
   - Business Risk: User registration, image upload, album management untested

2. **Auth Middleware (0% coverage)** - Security-critical code completely untested
   - JWT parsing and validation
   - Token blacklist checking
   - Role-based access control

3. **Redis Session Store (0% coverage)** - Session management untested
   - Create, revoke, and session listing operations
   - Concurrent session handling

4. **S3 Storage Provider (0-17.9% coverage)** - Core image storage operations untested
   - Put/Get operations
   - Presigned URL generation
   - Error handling for network failures

### Recommended Timeline

**Phase 1 (Critical - Week 1-2)**: HTTP handlers and auth middleware
**Phase 2 (High - Week 3-4)**: Redis, S3, and repository integration tests
**Phase 3 (Medium - Week 5-6)**: Security tests (ClamAV, token services), E2E expansion
**Quick Wins (Ongoing)**: Utility functions, DTOs, helpers (can be done in parallel)

**Estimated Total Effort**: 6 weeks (1 developer full-time) or 3 weeks (2 developers)

---

## Coverage Gap Matrix

### Priority Legend
- **P0 (Critical)**: Security-critical or user-facing code with 0% coverage
- **P1 (High)**: Infrastructure code with <50% coverage
- **P2 (Medium)**: Code with 50-70% coverage or less critical paths
- **P3 (Low)**: Nice-to-have improvements for 80%+ coverage

| Component | File/Package | Current % | Target % | Gap | Priority | Risk |
|-----------|--------------|-----------|----------|-----|----------|------|
| **HTTP Handlers** |
| Auth Handler | `handlers/auth_handler.go` | 0% | 75% | 75% | P0 | High |
| Image Handler | `handlers/image_handler.go` | 0% | 75% | 75% | P0 | High |
| Album Handler | `handlers/album_handler.go` | 0% | 75% | 75% | P0 | High |
| User Handler | `handlers/user_handler.go` | 0% | 75% | 75% | P0 | Medium |
| Social Handler | `handlers/social_handler.go` | 0% | 75% | 75% | P0 | Medium |
| Explore Handler | `handlers/explore_handler.go` | 0% | 75% | 75% | P0 | Low |
| **Middleware** |
| JWT Auth | `middleware/auth.go` | 0% | 80% | 80% | P0 | Critical |
| RBAC | `middleware/rbac.go` | 0% | 80% | 80% | P0 | Critical |
| Rate Limit | `middleware/rate_limit.go` | 0% | 70% | 70% | P1 | Medium |
| **Redis Infrastructure** |
| Session Store | `redis/session_store.go` | 0% | 75% | 75% | P0 | High |
| Cache | `redis/cache.go` | 0% | 70% | 70% | P1 | Medium |
| **Storage Providers** |
| S3 Storage | `storage/s3/s3.go` | 0-17.9% | 70% | 52-70% | P1 | High |
| Local Storage | `storage/local/local.go` | ~30% | 70% | 40% | P1 | Low |
| IPFS Storage | `storage/ipfs/ipfs.go` | ~20% | 70% | 50% | P2 | Low |
| **Security** |
| ClamAV Scanner | `security/clamav/scanner.go` | 0% | 75% | 75% | P1 | High |
| Token Blacklist | `jwt/token_blacklist.go` | ~60% | 80% | 20% | P2 | Medium |
| Refresh Token Service | `jwt/refresh_token_service.go` | ~70% | 85% | 15% | P2 | Medium |
| **Repositories** |
| Image Repository | `postgres/image_repository.go` | ~40% | 75% | 35% | P1 | High |
| Album Repository | `postgres/album_repository.go` | ~40% | 75% | 35% | P1 | Medium |
| User Repository | `postgres/user_repository.go` | ~50% | 75% | 25% | P2 | Medium |
| **Utilities** |
| Storage Keys | `storage/keys.go` | 0% | 80% | 80% | P3 | Low |
| Context Helpers | `shared/context.go` | 0% | 70% | 70% | P3 | Low |
| DTOs | `dto/*.go` | ~20% | 60% | 40% | P3 | Low |

---

## Phase 1: Critical Fixes (P0) - Weeks 1-2

**Goal**: Achieve 60% overall coverage by testing security-critical HTTP and auth components.

### 1.1 HTTP Handler Tests (Auth Handler)

**File**: `internal/interfaces/http/handlers/auth_handler_test.go`

**Test Cases Required**: 15 tests

#### Registration Endpoint Tests
```go
func TestAuthHandler_Register_Success(t *testing.T)
func TestAuthHandler_Register_InvalidEmail(t *testing.T)
func TestAuthHandler_Register_WeakPassword(t *testing.T)
func TestAuthHandler_Register_DuplicateEmail(t *testing.T)
func TestAuthHandler_Register_DuplicateUsername(t *testing.T)
```

#### Login Endpoint Tests
```go
func TestAuthHandler_Login_Success(t *testing.T)
func TestAuthHandler_Login_InvalidCredentials(t *testing.T)
func TestAuthHandler_Login_SuspendedUser(t *testing.T)
func TestAuthHandler_Login_MissingFields(t *testing.T)
```

#### Logout Endpoint Tests
```go
func TestAuthHandler_Logout_Success(t *testing.T)
func TestAuthHandler_Logout_InvalidToken(t *testing.T)
func TestAuthHandler_Logout_AlreadyLoggedOut(t *testing.T)
```

#### Token Refresh Tests
```go
func TestAuthHandler_RefreshToken_Success(t *testing.T)
func TestAuthHandler_RefreshToken_ExpiredToken(t *testing.T)
func TestAuthHandler_RefreshToken_RevokedToken(t *testing.T)
```

**Example Implementation**:

```go
package handlers_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/rs/zerolog"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
    "github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
)

func TestAuthHandler_Register_Success(t *testing.T) {
    // Arrange
    mockRegisterHandler := new(MockRegisterUserHandler)
    logger := zerolog.Nop()

    handler := handlers.NewAuthHandler(mockRegisterHandler, nil, nil, nil, &logger)

    reqBody := map[string]string{
        "email":    "test@example.com",
        "username": "testuser",
        "password": "SecurePass123!",
    }

    body, _ := json.Marshal(reqBody)
    req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()

    // Mock the command handler
    mockRegisterHandler.On("Handle", mock.Anything, mock.MatchedBy(func(cmd commands.RegisterUserCommand) bool {
        return cmd.Email == "test@example.com" && cmd.Username == "testuser"
    })).Return(createMockAuthResponse(), nil)

    // Act
    handler.Register(rec, req)

    // Assert
    assert.Equal(t, http.StatusCreated, rec.Code)

    var response map[string]interface{}
    err := json.NewDecoder(rec.Body).Decode(&response)
    require.NoError(t, err)

    assert.Contains(t, response, "user")
    assert.Contains(t, response, "tokens")

    mockRegisterHandler.AssertExpectations(t)
}

func TestAuthHandler_Register_WeakPassword(t *testing.T) {
    handler := handlers.NewAuthHandler(nil, nil, nil, nil, &zerolog.Nop())

    reqBody := map[string]string{
        "email":    "test@example.com",
        "username": "testuser",
        "password": "weak", // Too short, no special chars
    }

    body, _ := json.Marshal(reqBody)
    req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()

    handler.Register(rec, req)

    assert.Equal(t, http.StatusBadRequest, rec.Code)

    var problem map[string]interface{}
    json.NewDecoder(rec.Body).Decode(&problem)

    assert.Equal(t, "Validation failed", problem["title"])
    assert.Contains(t, problem, "errors")
}
```

**Estimated Effort**: 2-3 days

---

### 1.2 HTTP Handler Tests (Image Handler)

**File**: `internal/interfaces/http/handlers/image_handler_test.go`

**Test Cases Required**: 20 tests

#### Upload Tests
```go
func TestImageHandler_Upload_Success(t *testing.T)
func TestImageHandler_Upload_UnsupportedFormat(t *testing.T)
func TestImageHandler_Upload_FileTooLarge(t *testing.T)
func TestImageHandler_Upload_MalwareDetected(t *testing.T)
func TestImageHandler_Upload_Unauthorized(t *testing.T)
```

#### Retrieval Tests
```go
func TestImageHandler_Get_Success(t *testing.T)
func TestImageHandler_Get_NotFound(t *testing.T)
func TestImageHandler_Get_PrivateImageUnauthorized(t *testing.T)
func TestImageHandler_Get_PrivateImageOwnerCanAccess(t *testing.T)
```

#### Update Tests
```go
func TestImageHandler_Update_Success(t *testing.T)
func TestImageHandler_Update_NotOwner(t *testing.T)
func TestImageHandler_Update_InvalidVisibility(t *testing.T)
```

#### Delete Tests
```go
func TestImageHandler_Delete_Success(t *testing.T)
func TestImageHandler_Delete_NotOwner(t *testing.T)
func TestImageHandler_Delete_NotFound(t *testing.T)
```

#### List Tests
```go
func TestImageHandler_List_PublicImages(t *testing.T)
func TestImageHandler_List_UserImages(t *testing.T)
func TestImageHandler_List_WithPagination(t *testing.T)
func TestImageHandler_List_WithFilters(t *testing.T)
func TestImageHandler_List_WithSorting(t *testing.T)
```

**Key Testing Patterns**:

```go
func TestImageHandler_Upload_Success(t *testing.T) {
    // Create multipart form data
    var buf bytes.Buffer
    writer := multipart.NewWriter(&buf)

    // Add file
    part, _ := writer.CreateFormFile("file", "test.jpg")
    part.Write(testImageBytes)

    // Add metadata
    writer.WriteField("title", "Test Image")
    writer.WriteField("description", "A test image")
    writer.WriteField("tags", "test,image")
    writer.Close()

    req := httptest.NewRequest(http.MethodPost, "/api/v1/images", &buf)
    req.Header.Set("Content-Type", writer.FormDataContentType())
    req.Header.Set("Authorization", "Bearer "+testAccessToken)

    // Add user context (simulating JWT middleware)
    ctx := context.WithValue(req.Context(), middleware.UserIDKey, testUserID)
    req = req.WithContext(ctx)

    rec := httptest.NewRecorder()

    // Mock upload command
    mockUploadHandler.On("Handle", mock.Anything, mock.Anything).
        Return(createMockImage(), nil)

    handler.Upload(rec, req)

    assert.Equal(t, http.StatusCreated, rec.Code)
}
```

**Estimated Effort**: 3-4 days

---

### 1.3 Auth Middleware Tests

**File**: `internal/interfaces/http/middleware/auth_test.go`

**Test Cases Required**: 15 tests

```go
func TestJWTAuth_ValidToken(t *testing.T)
func TestJWTAuth_MissingAuthHeader(t *testing.T)
func TestJWTAuth_MalformedAuthHeader(t *testing.T)
func TestJWTAuth_InvalidTokenSignature(t *testing.T)
func TestJWTAuth_ExpiredToken(t *testing.T)
func TestJWTAuth_BlacklistedToken(t *testing.T)
func TestJWTAuth_MissingClaims(t *testing.T)
func TestJWTAuth_InvalidTokenFormat(t *testing.T)

func TestRequireRole_AdminAccess(t *testing.T)
func TestRequireRole_UserAccessDenied(t *testing.T)
func TestRequireRole_ModeratorAccess(t *testing.T)
func TestRequireRole_NoRoleInContext(t *testing.T)

func TestRequireAnyRole_MultipleRoles(t *testing.T)
func TestRequireAnyRole_NoMatchingRole(t *testing.T)
func TestRequireAnyRole_EmptyRoleList(t *testing.T)
```

**Example Implementation**:

```go
func TestJWTAuth_ValidToken(t *testing.T) {
    // Arrange
    mockJWTService := new(MockJWTService)
    mockBlacklist := new(MockTokenBlacklist)

    middleware := NewJWTAuthMiddleware(mockJWTService, mockBlacklist)

    validToken := "valid.jwt.token"

    mockJWTService.On("ValidateToken", validToken).
        Return(&jwt.Claims{
            Subject: "user-123",
            Custom: map[string]interface{}{
                "role": "user",
                "email": "test@example.com",
            },
        }, nil)

    mockBlacklist.On("IsBlacklisted", mock.Anything, validToken).
        Return(false, nil)

    // Create test handler
    nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := middleware.GetUserID(r.Context())
        assert.Equal(t, "user-123", userID)
        w.WriteHeader(http.StatusOK)
    })

    req := httptest.NewRequest(http.MethodGet, "/test", nil)
    req.Header.Set("Authorization", "Bearer "+validToken)
    rec := httptest.NewRecorder()

    // Act
    middleware.JWTAuth()(nextHandler).ServeHTTP(rec, req)

    // Assert
    assert.Equal(t, http.StatusOK, rec.Code)
    mockJWTService.AssertExpectations(t)
    mockBlacklist.AssertExpectations(t)
}

func TestJWTAuth_BlacklistedToken(t *testing.T) {
    mockJWTService := new(MockJWTService)
    mockBlacklist := new(MockTokenBlacklist)

    middleware := NewJWTAuthMiddleware(mockJWTService, mockBlacklist)

    blacklistedToken := "blacklisted.jwt.token"

    mockJWTService.On("ValidateToken", blacklistedToken).
        Return(&jwt.Claims{Subject: "user-123"}, nil)

    mockBlacklist.On("IsBlacklisted", mock.Anything, blacklistedToken).
        Return(true, nil)

    nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        t.Fatal("Should not reach here")
    })

    req := httptest.NewRequest(http.MethodGet, "/test", nil)
    req.Header.Set("Authorization", "Bearer "+blacklistedToken)
    rec := httptest.NewRecorder()

    middleware.JWTAuth()(nextHandler).ServeHTTP(rec, req)

    assert.Equal(t, http.StatusUnauthorized, rec.Code)

    var problem map[string]interface{}
    json.NewDecoder(rec.Body).Decode(&problem)
    assert.Contains(t, problem["detail"], "blacklisted")
}

func TestRequireRole_UserAccessDenied(t *testing.T) {
    middleware := NewRBACMiddleware()

    nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        t.Fatal("Should not reach here")
    })

    req := httptest.NewRequest(http.MethodGet, "/admin", nil)
    ctx := context.WithValue(req.Context(), middleware.UserRoleKey, "user")
    req = req.WithContext(ctx)
    rec := httptest.NewRecorder()

    middleware.RequireRole("admin")(nextHandler).ServeHTTP(rec, req)

    assert.Equal(t, http.StatusForbidden, rec.Code)
}
```

**Estimated Effort**: 2 days

---

### 1.4 RBAC Middleware Tests

**File**: `internal/interfaces/http/middleware/rbac_test.go`

**Test Cases Required**: 10 tests

```go
func TestRequireOwnership_OwnerCanAccess(t *testing.T)
func TestRequireOwnership_NonOwnerDenied(t *testing.T)
func TestRequireOwnership_AdminCanAccessAnyResource(t *testing.T)
func TestRequireOwnership_ModeratorCanAccessAnyResource(t *testing.T)
func TestRequireOwnership_MissingResourceID(t *testing.T)
func TestRequireOwnership_ResourceNotFound(t *testing.T)
func TestRequireOwnership_InvalidResourceID(t *testing.T)
```

**Estimated Effort**: 1-2 days

---

### Phase 1 Summary

**Total Test Files**: 4
**Total Test Cases**: ~60
**Estimated Effort**: 8-11 days
**Expected Coverage Gain**: +20-25% (total: 60-65%)

---

## Phase 2: Infrastructure (P1) - Weeks 3-4

**Goal**: Achieve 70% overall coverage by testing Redis, S3, and repository integrations.

### 2.1 Redis Session Store Tests

**File**: `internal/infrastructure/persistence/redis/session_store_test.go` (existing - needs expansion)

**Additional Test Cases Required**: 10 tests

Current coverage shows basic CRUD is tested. Need to add:

```go
func TestSessionStore_ConcurrentCreation(t *testing.T)
func TestSessionStore_SessionExpiration(t *testing.T)
func TestSessionStore_GetUserSessions_Pagination(t *testing.T)
func TestSessionStore_RevokeAll_ConcurrentSessions(t *testing.T)
func TestSessionStore_MaxSessionsPerUser(t *testing.T)
func TestSessionStore_SessionRefresh(t *testing.T)
func TestSessionStore_InvalidSessionData(t *testing.T)
func TestSessionStore_RedisConnectionFailure(t *testing.T)
func TestSessionStore_TransactionRollback(t *testing.T)
func TestSessionStore_BulkOperations(t *testing.T)
```

**Estimated Effort**: 2 days

---

### 2.2 S3 Storage Provider Tests

**File**: `internal/infrastructure/storage/s3/s3_test.go` (needs expansion)

**Test Cases Required**: 20 tests

```go
// Basic Operations
func TestS3Storage_Put_Success(t *testing.T)
func TestS3Storage_Put_InvalidKey(t *testing.T)
func TestS3Storage_Put_NetworkError(t *testing.T)
func TestS3Storage_Put_BucketNotFound(t *testing.T)
func TestS3Storage_Put_InsufficientPermissions(t *testing.T)

func TestS3Storage_Get_Success(t *testing.T)
func TestS3Storage_Get_NotFound(t *testing.T)
func TestS3Storage_Get_CorruptedObject(t *testing.T)

func TestS3Storage_Delete_Success(t *testing.T)
func TestS3Storage_Delete_NotFound(t *testing.T)
func TestS3Storage_Delete_MultipleObjects(t *testing.T)

func TestS3Storage_Exists_True(t *testing.T)
func TestS3Storage_Exists_False(t *testing.T)

// Presigned URLs
func TestS3Storage_PresignedURL_Success(t *testing.T)
func TestS3Storage_PresignedURL_ExpiredURL(t *testing.T)
func TestS3Storage_PresignedURL_InvalidDuration(t *testing.T)

// Metadata and Stats
func TestS3Storage_Stat_Success(t *testing.T)
func TestS3Storage_Stat_NotFound(t *testing.T)
func TestS3Storage_Stat_MetadataRetrieval(t *testing.T)

// Error Handling
func TestS3Storage_RetryLogic(t *testing.T)
```

**Testing Strategy**: Use MinIO testcontainer or LocalStack

```go
func setupMinIOTestContainer(t *testing.T) *s3.Client {
    ctx := context.Background()

    minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "minio/minio:latest",
            ExposedPorts: []string{"9000/tcp"},
            Env: map[string]string{
                "MINIO_ROOT_USER":     "minioadmin",
                "MINIO_ROOT_PASSWORD": "minioadmin",
            },
            Cmd: []string{"server", "/data"},
            WaitingFor: wait.ForHTTP("/minio/health/live").WithPort("9000/tcp"),
        },
        Started: true,
    })
    require.NoError(t, err)

    t.Cleanup(func() {
        minioContainer.Terminate(ctx)
    })

    endpoint, _ := minioContainer.Endpoint(ctx, "")

    // Create S3 client pointing to MinIO
    cfg := aws.NewConfig().
        WithRegion("us-east-1").
        WithEndpoint("http://" + endpoint).
        WithCredentials(credentials.NewStaticCredentials("minioadmin", "minioadmin", "")).
        WithS3ForcePathStyle(true)

    return s3.New(session.Must(session.NewSession(cfg)))
}
```

**Estimated Effort**: 3 days

---

### 2.3 Repository Integration Tests

**Files**:
- `internal/infrastructure/persistence/postgres/image_repository_test.go`
- `internal/infrastructure/persistence/postgres/album_repository_test.go`
- `internal/infrastructure/persistence/postgres/user_repository_test.go` (expand existing)

**Test Cases per Repository**: 15-20 tests

**Image Repository Tests**:
```go
func TestImageRepository_Save_NewImage(t *testing.T)
func TestImageRepository_Save_UpdateExisting(t *testing.T)
func TestImageRepository_Save_OptimisticLockConflict(t *testing.T)
func TestImageRepository_FindByID_Success(t *testing.T)
func TestImageRepository_FindByID_NotFound(t *testing.T)
func TestImageRepository_FindByUserID_Success(t *testing.T)
func TestImageRepository_FindByUserID_WithPagination(t *testing.T)
func TestImageRepository_Search_ByTags(t *testing.T)
func TestImageRepository_Search_ByTitle(t *testing.T)
func TestImageRepository_Search_ByVisibility(t *testing.T)
func TestImageRepository_Delete_Success(t *testing.T)
func TestImageRepository_Delete_NotFound(t *testing.T)
func TestImageRepository_Transaction_Commit(t *testing.T)
func TestImageRepository_Transaction_Rollback(t *testing.T)
func TestImageRepository_BulkOperations(t *testing.T)
```

**Testing Pattern with testcontainers**:

```go
func setupTestDB(t *testing.T) *sqlx.DB {
    ctx := context.Background()

    postgresContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        postgres.WithInitScripts("../migrations/00001_init.sql"),
    )
    require.NoError(t, err)

    t.Cleanup(func() {
        postgresContainer.Terminate(ctx)
    })

    connStr, _ := postgresContainer.ConnectionString(ctx)
    db, err := sqlx.Connect("postgres", connStr)
    require.NoError(t, err)

    return db
}

func TestImageRepository_Save_NewImage(t *testing.T) {
    db := setupTestDB(t)
    repo := postgres.NewImageRepository(db)

    image := createTestImage(t)

    err := repo.Save(context.Background(), image)
    require.NoError(t, err)

    // Verify persistence
    retrieved, err := repo.FindByID(context.Background(), image.ID())
    require.NoError(t, err)
    assert.Equal(t, image.Title(), retrieved.Title())
}
```

**Estimated Effort**: 4-5 days (all repositories)

---

### Phase 2 Summary

**Total Test Files**: 4
**Total Test Cases**: ~65
**Estimated Effort**: 9-10 days
**Expected Coverage Gain**: +10-15% (total: 70-75%)

---

## Phase 3: Security & Integration (P2) - Weeks 5-6

**Goal**: Achieve 80%+ overall coverage with security tests and expanded E2E coverage.

### 3.1 ClamAV Scanner Tests

**File**: `internal/infrastructure/security/clamav/scanner_test.go`

**Test Cases Required**: 12 tests

```go
func TestClamAVScanner_Scan_CleanFile(t *testing.T)
func TestClamAVScanner_Scan_MalwareDetected(t *testing.T)
func TestClamAVScanner_Scan_SuspiciousFile(t *testing.T)
func TestClamAVScanner_ScanReader_CleanStream(t *testing.T)
func TestClamAVScanner_ScanReader_MalwareInStream(t *testing.T)
func TestClamAVScanner_ScanReader_LargeFile(t *testing.T)
func TestClamAVScanner_Ping_Success(t *testing.T)
func TestClamAVScanner_Ping_DaemonNotRunning(t *testing.T)
func TestClamAVScanner_Version_Success(t *testing.T)
func TestClamAVScanner_ConnectionTimeout(t *testing.T)
func TestClamAVScanner_SocketConnectionError(t *testing.T)
func TestClamAVScanner_InvalidResponse(t *testing.T)
```

**Testing Strategy**: Use ClamAV testcontainer

```go
func setupClamAVContainer(t *testing.T) string {
    ctx := context.Background()

    clamavContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "clamav/clamav:latest",
            ExposedPorts: []string{"3310/tcp"},
            WaitingFor:   wait.ForLog("clamd: started").WithStartupTimeout(120 * time.Second),
        },
        Started: true,
    })
    require.NoError(t, err)

    t.Cleanup(func() {
        clamavContainer.Terminate(ctx)
    })

    host, _ := clamavContainer.Host(ctx)
    port, _ := clamavContainer.MappedPort(ctx, "3310")

    return fmt.Sprintf("%s:%s", host, port.Port())
}

func TestClamAVScanner_Scan_MalwareDetected(t *testing.T) {
    clamavAddr := setupClamAVContainer(t)
    scanner := clamav.NewScanner(clamavAddr)

    // EICAR test file (standard malware test string)
    eicar := []byte("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*")

    result, err := scanner.Scan(context.Background(), eicar)
    require.NoError(t, err)

    assert.False(t, result.Clean)
    assert.Contains(t, result.VirusName, "EICAR")
}
```

**Estimated Effort**: 2 days

---

### 3.2 Token Services Tests

**Files**:
- `internal/infrastructure/security/jwt/token_blacklist_test.go` (expand existing ~60% coverage)
- `internal/infrastructure/security/jwt/refresh_token_service_test.go` (expand existing ~70% coverage)

**Additional Test Cases**: 15 tests total

**Token Blacklist Tests** (add 8 tests):
```go
func TestTokenBlacklist_ConcurrentBlacklisting(t *testing.T)
func TestTokenBlacklist_BulkBlacklist(t *testing.T)
func TestTokenBlacklist_ExpirationCleanup(t *testing.T)
func TestTokenBlacklist_RedisFailover(t *testing.T)
func TestTokenBlacklist_PersistenceVerification(t *testing.T)
func TestTokenBlacklist_DuplicateBlacklisting(t *testing.T)
func TestTokenBlacklist_BlacklistExpiry(t *testing.T)
func TestTokenBlacklist_InvalidTokenFormat(t *testing.T)
```

**Refresh Token Service Tests** (add 7 tests):
```go
func TestRefreshTokenService_TokenRotation(t *testing.T)
func TestRefreshTokenService_FamilyRevocation(t *testing.T)
func TestRefreshTokenService_ReuseDetection(t *testing.T)
func TestRefreshTokenService_MaxTokensPerUser(t *testing.T)
func TestRefreshTokenService_ExpiredTokenCleanup(t *testing.T)
func TestRefreshTokenService_ConcurrentRefresh(t *testing.T)
func TestRefreshTokenService_InvalidParentToken(t *testing.T)
```

**Estimated Effort**: 2 days

---

### 3.3 E2E Test Expansion

**File**: `tests/e2e/postman/goimg-api.postman_collection.json`

**Current E2E Coverage**: ~60% (based on user's description)

**Missing Test Scenarios**: 25 requests

#### Album Management (10 requests)
- Create album (authenticated)
- Get album details (public)
- Get album details (private - owner access)
- Get album details (private - unauthorized)
- Update album metadata
- Add image to album
- Remove image from album
- List album images
- Delete album
- List user albums

#### Image Search & Discovery (8 requests)
- Search images by tags
- Search images by title
- Search images by user
- Filter by visibility (public only for unauthenticated)
- Sort by date (newest first)
- Sort by popularity (likes count)
- Pagination (page 1, page 2)
- Combined filters and sorting

#### Moderation Workflows (7 requests)
- Report image (authenticated user)
- List reports (moderator)
- Review report - approve (moderator)
- Review report - reject (moderator)
- Ban user (admin)
- List banned users (admin)
- Unban user (admin)

**Example Postman Test Scripts**:

```javascript
// Album Creation Test
pm.test("Status code is 201 Created", function () {
    pm.response.to.have.status(201);
});

pm.test("Response contains album ID", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property('id');
    pm.environment.set("album_id", jsonData.id);
});

pm.test("Album has correct owner", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData.userId).to.eql(pm.environment.get("user_id"));
});

// Search Test
pm.test("Search returns results array", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property('images');
    pm.expect(jsonData.images).to.be.an('array');
});

pm.test("Pagination metadata present", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property('totalCount');
    pm.expect(jsonData).to.have.property('offset');
    pm.expect(jsonData).to.have.property('limit');
});
```

**Estimated Effort**: 2-3 days

---

### Phase 3 Summary

**Total Test Files**: 3 + Postman collection updates
**Total Test Cases**: ~52
**Estimated Effort**: 6-7 days
**Expected Coverage Gain**: +5-10% (total: 80%+)

---

## Quick Wins (Parallel Tasks)

These are high-value, low-effort tests that can be done in parallel with other phases.

### Quick Win 1: Storage Key Utilities (0% → 80%)

**File**: `internal/infrastructure/storage/keys_test.go`

**Test Cases**: 8 tests (1 hour)

```go
func TestGenerateKey_ValidFormat(t *testing.T)
func TestGenerateKey_UniqueKeys(t *testing.T)
func TestValidateKey_Valid(t *testing.T)
func TestValidateKey_Invalid(t *testing.T)
func TestParseKey_Success(t *testing.T)
func TestParseKey_MalformedKey(t *testing.T)
func TestSanitizeFilename_RemovesSpecialChars(t *testing.T)
func TestSanitizeFilename_PreservesExtension(t *testing.T)
```

---

### Quick Win 2: Context Helpers (0% → 70%)

**File**: `internal/shared/context_test.go`

**Test Cases**: 6 tests (30 minutes)

```go
func TestGetUserID_Present(t *testing.T)
func TestGetUserID_Missing(t *testing.T)
func TestGetUserRole_Present(t *testing.T)
func TestGetUserRole_Missing(t *testing.T)
func TestGetRequestID_Present(t *testing.T)
func TestGetRequestID_Generated(t *testing.T)
```

---

### Quick Win 3: DTO Mapping Functions (20% → 60%)

**Files**: Various `dto/*.go` files

**Test Cases**: 12 tests (2 hours)

```go
func TestUserDTO_FromDomain(t *testing.T)
func TestUserDTO_ToDomain(t *testing.T)
func TestImageDTO_FromDomain(t *testing.T)
func TestImageDTO_FromDomain_NilFields(t *testing.T)
func TestAlbumDTO_FromDomain(t *testing.T)
func TestPaginatedResponse_Mapping(t *testing.T)
// ... etc
```

---

### Quick Win 4: Error Response Helpers (0% → 80%)

**File**: `internal/interfaces/http/errors_test.go`

**Test Cases**: 10 tests (1 hour)

```go
func TestProblemBadRequest_Structure(t *testing.T)
func TestProblemNotFound_Structure(t *testing.T)
func TestProblemUnauthorized_Structure(t *testing.T)
func TestProblemForbidden_Structure(t *testing.T)
func TestProblemConflict_Structure(t *testing.T)
func TestProblemValidation_WithErrors(t *testing.T)
func TestProblemInternalError_NoDetailsLeaked(t *testing.T)
func TestRespondProblem_SetsCorrectHeaders(t *testing.T)
func TestRespondProblem_IncludesTraceID(t *testing.T)
func TestRespondProblem_RFC7807Compliance(t *testing.T)
```

---

### Quick Wins Summary

**Total Effort**: 5-6 hours
**Expected Coverage Gain**: +2-3%
**Can be done by junior developers or in spare time between phases**

---

## Test Templates by Layer

### HTTP Handler Template

```go
package handlers_test

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/go-chi/chi/v5"
    "github.com/rs/zerolog"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
    "github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// Mock command/query handler
type MockCommandHandler struct {
    mock.Mock
}

func (m *MockCommandHandler) Handle(ctx context.Context, cmd interface{}) (interface{}, error) {
    args := m.Called(ctx, cmd)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0), args.Error(1)
}

func TestHandler_Endpoint_Success(t *testing.T) {
    // Arrange
    mockHandler := new(MockCommandHandler)
    logger := zerolog.Nop()
    handler := handlers.NewHandler(mockHandler, &logger)

    reqBody := map[string]interface{}{
        "field1": "value1",
        "field2": "value2",
    }
    body, _ := json.Marshal(reqBody)

    req := httptest.NewRequest(http.MethodPost, "/api/v1/endpoint", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer test-token")

    // Add user context (simulating middleware)
    ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-123")
    req = req.WithContext(ctx)

    rec := httptest.NewRecorder()

    // Mock expectations
    mockHandler.On("Handle", mock.Anything, mock.Anything).
        Return(createMockResult(), nil)

    // Act
    handler.Endpoint(rec, req)

    // Assert
    assert.Equal(t, http.StatusOK, rec.Code)

    var response map[string]interface{}
    err := json.NewDecoder(rec.Body).Decode(&response)
    require.NoError(t, err)
    assert.Contains(t, response, "expectedField")

    mockHandler.AssertExpectations(t)
}

func TestHandler_Endpoint_ValidationError(t *testing.T) {
    handler := handlers.NewHandler(nil, &zerolog.Nop())

    reqBody := map[string]interface{}{
        "invalidField": "value",
    }
    body, _ := json.Marshal(reqBody)

    req := httptest.NewRequest(http.MethodPost, "/api/v1/endpoint", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()

    handler.Endpoint(rec, req)

    assert.Equal(t, http.StatusBadRequest, rec.Code)

    var problem map[string]interface{}
    json.NewDecoder(rec.Body).Decode(&problem)
    assert.Equal(t, "Validation failed", problem["title"])
}
```

---

### Middleware Test Template

```go
package middleware_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func TestMiddleware_Success(t *testing.T) {
    // Arrange
    mockDependency := new(MockDependency)
    mw := middleware.NewMiddleware(mockDependency)

    nextCalled := false
    nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        nextCalled = true
        w.WriteHeader(http.StatusOK)
    })

    req := httptest.NewRequest(http.MethodGet, "/test", nil)
    rec := httptest.NewRecorder()

    mockDependency.On("SomeMethod", mock.Anything).Return(nil)

    // Act
    mw.Handler()(nextHandler).ServeHTTP(rec, req)

    // Assert
    assert.True(t, nextCalled, "Next handler should be called")
    assert.Equal(t, http.StatusOK, rec.Code)
    mockDependency.AssertExpectations(t)
}

func TestMiddleware_BlocksRequest(t *testing.T) {
    mw := middleware.NewMiddleware(nil)

    nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        t.Fatal("Next handler should not be called")
    })

    req := httptest.NewRequest(http.MethodGet, "/test", nil)
    rec := httptest.NewRecorder()

    mw.Handler()(nextHandler).ServeHTTP(rec, req)

    assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
```

---

### Repository Integration Test Template

```go
package postgres_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go/modules/postgres"

    pgRepo "github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
    "github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func setupTestDB(t *testing.T) *sqlx.DB {
    t.Helper()

    ctx := context.Background()

    postgresContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        postgres.WithInitScripts("../../../../migrations/"),
    )
    require.NoError(t, err)

    t.Cleanup(func() {
        postgresContainer.Terminate(ctx)
    })

    connStr, _ := postgresContainer.ConnectionString(ctx)
    db, err := sqlx.Connect("postgres", connStr)
    require.NoError(t, err)

    return db
}

func TestRepository_Save_Success(t *testing.T) {
    // Arrange
    db := setupTestDB(t)
    repo := pgRepo.NewRepository(db)
    ctx := context.Background()

    entity := createTestEntity(t)

    // Act
    err := repo.Save(ctx, entity)

    // Assert
    require.NoError(t, err)

    // Verify persistence
    retrieved, err := repo.FindByID(ctx, entity.ID())
    require.NoError(t, err)
    assert.Equal(t, entity.ID(), retrieved.ID())
}

func TestRepository_Transaction_Rollback(t *testing.T) {
    db := setupTestDB(t)
    repo := pgRepo.NewRepository(db)
    ctx := context.Background()

    txRepo, err := repo.WithTx(ctx)
    require.NoError(t, err)

    entity := createTestEntity(t)
    err = txRepo.Save(ctx, entity)
    require.NoError(t, err)

    // Rollback
    err = txRepo.RollbackTx(ctx)
    require.NoError(t, err)

    // Verify entity not persisted
    _, err = repo.FindByID(ctx, entity.ID())
    assert.ErrorIs(t, err, identity.ErrNotFound)
}
```

---

### Infrastructure Service Test Template

```go
package s3_test

import (
    "bytes"
    "context"
    "io"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/yegamble/goimg-datalayer/internal/infrastructure/storage/s3"
)

func setupMinIO(t *testing.T) *s3.Storage {
    // Use MinIO testcontainer or mock
    // ...
    return storage
}

func TestS3Storage_Put_Success(t *testing.T) {
    // Arrange
    storage := setupMinIO(t)
    ctx := context.Background()

    key := "test-images/test.jpg"
    data := []byte("test image data")
    reader := bytes.NewReader(data)

    // Act
    err := storage.Put(ctx, key, reader, int64(len(data)), storage.PutOptions{
        ContentType: "image/jpeg",
    })

    // Assert
    require.NoError(t, err)

    // Verify file exists
    exists, err := storage.Exists(ctx, key)
    require.NoError(t, err)
    assert.True(t, exists)
}

func TestS3Storage_Get_NotFound(t *testing.T) {
    storage := setupMinIO(t)
    ctx := context.Background()

    _, err := storage.Get(ctx, "nonexistent-key")

    assert.Error(t, err)
    assert.ErrorIs(t, err, storage.ErrNotFound)
}
```

---

## Success Metrics

### Coverage Tracking

**Run coverage after each phase:**

```bash
# Generate coverage report
make test-coverage

# View HTML report
go tool cover -html=coverage.out

# Check coverage by package
go test -coverprofile=coverage.out ./... && \
  go tool cover -func=coverage.out | grep -E 'total|handlers|middleware|redis|s3'
```

**Expected Milestones:**

| Phase | Target Coverage | Key Packages |
|-------|----------------|--------------|
| Baseline | 40.3% | Current state |
| Phase 1 Complete | 60-65% | handlers/, middleware/ >75% |
| Phase 2 Complete | 70-75% | redis/, s3/, postgres/ >70% |
| Phase 3 Complete | 80%+ | security/, e2e/ expanded |

---

### Quality Metrics

Beyond coverage percentage, track:

1. **Test Flakiness**: <1% flaky test rate
2. **Test Execution Time**: <5 minutes for full suite
3. **Test Maintenance**: Zero skipped/ignored tests
4. **Mutation Testing Score**: 70%+ (consider using go-mutesting)

---

### CI/CD Integration

Update `.github/workflows/test.yml` to enforce coverage:

```yaml
- name: Run tests with coverage
  run: make test-coverage

- name: Check coverage threshold
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$COVERAGE < 80" | bc -l) )); then
      echo "Coverage $COVERAGE% is below 80% threshold"
      exit 1
    fi
    echo "Coverage: $COVERAGE%"

- name: Upload coverage to Codecov
  uses: codecov/codecov-action@v3
  with:
    files: ./coverage.out
    fail_ci_if_error: true
```

---

## Estimated Effort Summary

### By Phase

| Phase | Test Files | Test Cases | Days (1 Dev) | Days (2 Devs) |
|-------|-----------|------------|--------------|---------------|
| Phase 1 (P0) | 4 | ~60 | 8-11 | 4-6 |
| Phase 2 (P1) | 4 | ~65 | 9-10 | 5-6 |
| Phase 3 (P2) | 3 + E2E | ~52 | 6-7 | 3-4 |
| Quick Wins | 4 | ~36 | 0.75 | 0.5 |
| **Total** | **15** | **~213** | **24-28** | **13-16** |

**Calendar Time**:
- 1 developer: 5-6 weeks
- 2 developers: 3-4 weeks

### By Developer Type

**Senior Developer Tasks** (Phases 1-2):
- HTTP handler tests
- Middleware tests
- Repository integration tests
- ~18 days

**Mid-Level Developer Tasks** (Phase 3):
- Infrastructure service tests
- E2E test expansion
- ~6-7 days

**Junior Developer Tasks** (Quick Wins):
- Utility function tests
- DTO mapping tests
- Error response tests
- ~0.75 days

---

## Risks and Mitigation

### Risk 1: Test Environment Setup Complexity

**Risk**: Setting up testcontainers for Postgres, Redis, MinIO, ClamAV may be time-consuming.

**Mitigation**:
- Create reusable test helper package: `tests/testhelpers/containers.go`
- Document setup in `tests/README.md`
- Use Docker Compose for local test environment: `docker-compose.test.yml`

---

### Risk 2: Flaky Integration Tests

**Risk**: Network-dependent tests (S3, Redis) may be flaky.

**Mitigation**:
- Use testcontainers with proper wait strategies
- Add retry logic for network operations
- Separate integration tests with build tags: `//go:build integration`
- Run integration tests separately in CI

---

### Risk 3: Test Maintenance Burden

**Risk**: 200+ new tests require ongoing maintenance.

**Mitigation**:
- Use table-driven tests to reduce duplication
- Create test fixture factories: `testhelpers/fixtures.go`
- Document test patterns in `tests/TESTING.md`
- Enforce "test with code" policy (PR must include tests)

---

### Risk 4: Coverage Measurement Accuracy

**Risk**: Coverage % may not reflect actual code quality.

**Mitigation**:
- Review coverage reports manually
- Focus on edge cases and error paths
- Use mutation testing to validate test effectiveness
- Track test-to-code ratio (target: 1.5:1)

---

## Next Steps

### Immediate Actions (Week 1)

1. **Set up test infrastructure**
   - Configure testcontainers
   - Create test helper utilities
   - Document test setup in `tests/README.md`

2. **Start Phase 1: Auth Handler**
   - `handlers/auth_handler_test.go` (highest priority)
   - Run coverage: should reach ~45-50%

3. **Parallel: Quick Win 1 (Storage Keys)**
   - Easy win to build momentum
   - 1 hour effort

### Weekly Checkpoints

**Week 1**: Auth handler tests complete, coverage ~50%
**Week 2**: Image handler + middleware tests complete, coverage ~60%
**Week 3**: Redis + S3 tests complete, coverage ~70%
**Week 4**: Repository integration tests complete, coverage ~75%
**Week 5**: Security tests (ClamAV, tokens) complete, coverage ~78%
**Week 6**: E2E expansion complete, coverage 80%+

### Long-Term Improvements

1. **Add mutation testing**: Use go-mutesting to validate test effectiveness
2. **Performance benchmarks**: Add `_bench_test.go` files for critical paths
3. **Fuzz testing**: Add fuzzing for parser/validator functions
4. **Contract testing**: Pact tests for external API integrations
5. **Property-based testing**: Use `gopter` for complex domain logic

---

## Appendix A: Required Mock Implementations

Create these mocks in `tests/mocks/`:

```go
// mocks/commands.go
type MockRegisterUserHandler struct { mock.Mock }
type MockLoginHandler struct { mock.Mock }
type MockUploadImageHandler struct { mock.Mock }

// mocks/queries.go
type MockGetUserHandler struct { mock.Mock }
type MockListImagesHandler struct { mock.Mock }

// mocks/services.go
type MockJWTService struct { mock.Mock }
type MockTokenBlacklist struct { mock.Mock }
type MockRefreshTokenService struct { mock.Mock }
type MockClamAVScanner struct { mock.Mock }

// mocks/repositories.go
type MockUserRepository struct { mock.Mock }
type MockImageRepository struct { mock.Mock }
type MockAlbumRepository struct { mock.Mock }

// mocks/storage.go
type MockStorage struct { mock.Mock }
```

---

## Appendix B: Test Fixtures

Create test data factories in `tests/testhelpers/fixtures.go`:

```go
package testhelpers

import (
    "time"

    "github.com/google/uuid"
    "github.com/yegamble/goimg-datalayer/internal/domain/identity"
    "github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

func ValidUser() *identity.User {
    email, _ := identity.NewEmail("test@example.com")
    username, _ := identity.NewUsername("testuser")
    password, _ := identity.HashPassword("SecurePass123!")

    user, _ := identity.NewUser(email, username, password)
    return user
}

func ValidImage(userID identity.UserID) *gallery.Image {
    title, _ := gallery.NewTitle("Test Image")
    description, _ := gallery.NewDescription("A test image")

    metadata := gallery.ImageMetadata{
        Width:      1920,
        Height:     1080,
        Format:     "jpeg",
        Size:       1024000,
        Hash:       "abc123",
    }

    image, _ := gallery.NewImage(
        userID,
        title,
        description,
        "https://example.com/test.jpg",
        metadata,
    )

    return image
}

func ValidAlbum(userID identity.UserID) *gallery.Album {
    title, _ := gallery.NewTitle("Test Album")
    description, _ := gallery.NewDescription("A test album")

    album, _ := gallery.NewAlbum(userID, title, description)
    return album
}
```

---

## Appendix C: GitHub Actions Workflow

Update `.github/workflows/test.yml`:

```yaml
name: Test Coverage

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Install dependencies
        run: go mod download

      - name: Run unit tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Run integration tests
        run: go test -v -tags=integration -coverprofile=coverage-integration.out ./tests/integration/...
        env:
          POSTGRES_HOST: localhost
          POSTGRES_PORT: 5432
          REDIS_HOST: localhost
          REDIS_PORT: 6379

      - name: Merge coverage reports
        run: |
          go install github.com/wadey/gocovmerge@latest
          gocovmerge coverage.out coverage-integration.out > coverage-merged.out

      - name: Check coverage threshold
        run: |
          COVERAGE=$(go tool cover -func=coverage-merged.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Total coverage: $COVERAGE%"
          if (( $(echo "$COVERAGE < 80" | bc -l) )); then
            echo "❌ Coverage $COVERAGE% is below 80% threshold"
            exit 1
          fi
          echo "✅ Coverage threshold met: $COVERAGE%"

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage-merged.out
          fail_ci_if_error: true

      - name: Generate coverage report
        run: go tool cover -html=coverage-merged.out -o coverage.html

      - name: Upload coverage HTML
        uses: actions/upload-artifact@v3
        with:
          name: coverage-report
          path: coverage.html
```

---

## Document Maintenance

**Review Frequency**: Monthly during active development, quarterly after reaching 80% coverage

**Owner**: Test Strategist + Senior Go Architect

**Update Triggers**:
- New API endpoints added
- Infrastructure changes (new storage provider, security service)
- Coverage drops below 75%
- Test suite execution time exceeds 5 minutes

---

**End of Test Coverage Improvement Plan**
