# Security Testing Guide

> Comprehensive security testing requirements, tools, and procedures for goimg-datalayer.
> **Load this guide** when implementing security tests or validating security controls.

---

## Testing Strategy Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Security Test Pyramid                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│                    ┌──────────────────────┐                        │
│                    │  Manual Pentest      │  < 5% - Quarterly      │
│                    │  External Audits     │                        │
│                    └──────────────────────┘                        │
│                                                                     │
│              ┌────────────────────────────────┐                    │
│              │    Dynamic Security Tests      │  10% - CI/Daily    │
│              │    (OWASP ZAP, API fuzzing)    │                    │
│              └────────────────────────────────┘                    │
│                                                                     │
│        ┌──────────────────────────────────────────────┐            │
│        │       Static Security Tests (SAST)           │  20%       │
│        │  (gosec, trivy, nancy, gitleaks)             │  CI/commit │
│        └──────────────────────────────────────────────┘            │
│                                                                     │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │          Unit Security Tests (Go test)                     │    │
│  │  (Auth bypass, injection, validation, crypto)              │    │
│  │                                                            │ 65% │
│  │                                                            │ CI  │
│  └────────────────────────────────────────────────────────────┘    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 1. Unit Security Tests

Location: `tests/security/unit/` and within package tests

### 1.1 Authentication Tests

**File**: `tests/security/unit/auth_test.go`

```go
package security_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// JWT Security Tests
func TestJWT_RejectsHS256Algorithm(t *testing.T) {
    t.Parallel()

    // Attempt to create token with HS256
    token := createHS256Token(t, validClaims)

    // Should be rejected
    _, err := jwtService.Validate(token)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "algorithm not allowed")
}

func TestJWT_RejectsExpiredToken(t *testing.T) {
    t.Parallel()

    // Create expired token (issued 20 minutes ago)
    token := createExpiredToken(t, -20*time.Minute)

    _, err := jwtService.Validate(token)
    require.ErrorIs(t, err, security.ErrTokenExpired)
}

func TestJWT_ValidatesAudience(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name        string
        audience    string
        expectError bool
    }{
        {"valid audience", "goimg-api", false},
        {"wrong audience", "other-api", true},
        {"empty audience", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            token := createTokenWithAudience(t, tt.audience)
            _, err := jwtService.Validate(token)

            if tt.expectError {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}

func TestJWT_RejectsTokenWithInvalidSignature(t *testing.T) {
    t.Parallel()

    token := createValidToken(t)
    tamperedToken := token[:len(token)-5] + "AAAAA" // Corrupt signature

    _, err := jwtService.Validate(tamperedToken)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "signature")
}

// Refresh Token Tests
func TestRefreshToken_DetectsReplay(t *testing.T) {
    t.Parallel()

    // Create session with refresh token
    session, refreshToken := createSession(t)

    // Use refresh token (should succeed)
    newAccessToken1, err := authService.RefreshAccessToken(ctx, refreshToken)
    require.NoError(t, err)
    assert.NotEmpty(t, newAccessToken1)

    // Try to reuse same refresh token (should fail - replay attack)
    _, err = authService.RefreshAccessToken(ctx, refreshToken)
    require.ErrorIs(t, err, security.ErrTokenReplayDetected)
}

func TestRefreshToken_RotatesOnUse(t *testing.T) {
    t.Parallel()

    session, oldRefreshToken := createSession(t)

    // Refresh should return new refresh token
    response, err := authService.RefreshAccessToken(ctx, oldRefreshToken)
    require.NoError(t, err)

    assert.NotEqual(t, oldRefreshToken, response.RefreshToken)
    assert.NotEmpty(t, response.AccessToken)

    // Old refresh token should be invalidated
    _, err = authService.RefreshAccessToken(ctx, oldRefreshToken)
    require.Error(t, err)
}

// Password Security Tests
func TestPassword_UsesArgon2id(t *testing.T) {
    t.Parallel()

    password := "SecurePassword123!"
    hash, err := passwordService.Hash(password)
    require.NoError(t, err)

    // Verify hash format is Argon2id
    assert.Contains(t, hash, "$argon2id$")
}

func TestPassword_RejectsWeakPasswords(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        password string
        wantErr  error
    }{
        {"too short", "Pass1!", identity.ErrPasswordTooWeak},
        {"no uppercase", "password123!", identity.ErrPasswordTooWeak},
        {"no lowercase", "PASSWORD123!", identity.ErrPasswordTooWeak},
        {"no digit", "Password!", identity.ErrPasswordTooWeak},
        {"common password", "Password123!", identity.ErrPasswordCommon},
        {"valid password", "MySecureP@ssw0rd!", nil},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := identity.NewPassword(tt.password)

            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
            } else {
                require.NoError(t, err)
            }
        })
    }
}

func TestPassword_UsesConstantTimeComparison(t *testing.T) {
    t.Parallel()

    password := "SecurePassword123!"
    hash, _ := passwordService.Hash(password)

    // Timing attack test (simplified)
    // Real test would use statistical analysis
    iterations := 1000
    var correctTimings []time.Duration
    var incorrectTimings []time.Duration

    for i := 0; i < iterations; i++ {
        start := time.Now()
        _ = passwordService.Verify(password, hash)
        correctTimings = append(correctTimings, time.Since(start))

        start = time.Now()
        _ = passwordService.Verify("WrongPassword", hash)
        incorrectTimings = append(incorrectTimings, time.Since(start))
    }

    // Timing should be similar (within 20% variance)
    avgCorrect := average(correctTimings)
    avgIncorrect := average(incorrectTimings)

    variance := math.Abs(float64(avgCorrect-avgIncorrect)) / float64(avgCorrect)
    assert.Less(t, variance, 0.2, "Timing variance suggests timing attack vulnerability")
}

// Session Security Tests
func TestSession_RegeneratesOnLogin(t *testing.T) {
    t.Parallel()

    // Create initial session
    oldSessionID := createAnonymousSession(t)

    // Login
    response, err := authService.Login(ctx, LoginRequest{
        Email:     "user@example.com",
        Password:  "Password123!",
        SessionID: oldSessionID,
    })
    require.NoError(t, err)

    // New session ID should be different
    assert.NotEqual(t, oldSessionID, response.SessionID)

    // Old session should be invalidated
    _, err = sessionRepo.FindByID(ctx, oldSessionID)
    require.ErrorIs(t, err, security.ErrSessionNotFound)
}
```

### 1.2 Authorization Tests

**File**: `tests/security/unit/authz_test.go`

```go
// RBAC Tests
func TestRBAC_UserCannotAccessAdminEndpoint(t *testing.T) {
    t.Parallel()

    // Create user with 'user' role
    user := createTestUser(t, identity.RoleUser)

    // Attempt to access admin endpoint
    err := authzService.CheckPermission(ctx, user.ID, security.PermAdminPanel)

    require.ErrorIs(t, err, security.ErrInsufficientPermissions)
}

func TestRBAC_ModeratorCannotGrantAdminRole(t *testing.T) {
    t.Parallel()

    moderator := createTestUser(t, identity.RoleModerator)
    targetUser := createTestUser(t, identity.RoleUser)

    err := userService.UpdateRole(ctx, UpdateRoleCommand{
        ActorID:      moderator.ID,
        TargetUserID: targetUser.ID,
        NewRole:      identity.RoleAdmin,
    })

    require.ErrorIs(t, err, security.ErrInsufficientPermissions)
}

func TestRBAC_UserCannotEscalateOwnRole(t *testing.T) {
    t.Parallel()

    user := createTestUser(t, identity.RoleUser)

    err := userService.UpdateRole(ctx, UpdateRoleCommand{
        ActorID:      user.ID,
        TargetUserID: user.ID,
        NewRole:      identity.RoleAdmin,
    })

    require.ErrorIs(t, err, security.ErrInsufficientPermissions)
}

// IDOR Tests
func TestImage_GetPrivateByNonOwner_Returns403(t *testing.T) {
    t.Parallel()

    owner := createTestUser(t, identity.RoleUser)
    otherUser := createTestUser(t, identity.RoleUser)

    // Owner creates private image
    image := createPrivateImage(t, owner.ID)

    // Other user tries to access
    _, err := imageService.GetImage(ctx, GetImageQuery{
        ImageID: image.ID,
        ActorID: otherUser.ID,
    })

    require.ErrorIs(t, err, gallery.ErrInsufficientPermissions)
}

func TestImage_UpdateByNonOwner_Returns403(t *testing.T) {
    t.Parallel()

    owner := createTestUser(t, identity.RoleUser)
    otherUser := createTestUser(t, identity.RoleUser)
    image := createPublicImage(t, owner.ID)

    err := imageService.UpdateImage(ctx, UpdateImageCommand{
        ImageID: image.ID,
        ActorID: otherUser.ID,
        Title:   "Hacked Title",
    })

    require.ErrorIs(t, err, gallery.ErrInsufficientPermissions)
}
```

### 1.3 Input Validation Tests

**File**: `tests/security/unit/injection_test.go`

```go
// SQL Injection Tests
func TestUserRepository_SQLInjectionPrevention(t *testing.T) {
    t.Parallel()

    suite := setupTestSuite(t)
    repo := postgres.NewUserRepository(suite.DB)

    // Attempt SQL injection via email
    maliciousEmail := "admin'--"

    _, err := repo.FindByEmail(ctx, maliciousEmail)

    // Should return "not found", not execute injection
    require.ErrorIs(t, err, identity.ErrUserNotFound)

    // Verify no users were deleted
    count, _ := repo.Count(ctx)
    assert.Greater(t, count, 0)
}

func TestSearch_PreventsSQLInjection(t *testing.T) {
    t.Parallel()

    tests := []string{
        "'; DROP TABLE images--",
        "1' OR '1'='1",
        "admin'--",
        "1; DELETE FROM users WHERE 1=1--",
    }

    for _, maliciousQuery := range tests {
        t.Run(maliciousQuery, func(t *testing.T) {
            results, err := searchService.Search(ctx, SearchQuery{
                Query: maliciousQuery,
            })

            // Should not error, should sanitize
            require.NoError(t, err)

            // Should return empty or safe results
            assert.NotNil(t, results)
        })
    }
}

// XSS Prevention Tests
func TestComment_RejectsXSSPayload(t *testing.T) {
    t.Parallel()

    xssPayloads := []string{
        "<script>alert('XSS')</script>",
        "<img src=x onerror=alert('XSS')>",
        "javascript:alert('XSS')",
        "<iframe src='javascript:alert(\"XSS\")'></iframe>",
    }

    for _, payload := range xssPayloads {
        t.Run(payload, func(t *testing.T) {
            comment, err := gallery.NewComment(userID, imageID, payload)

            if err == nil {
                // If allowed, must be sanitized
                assert.NotContains(t, comment.Content(), "<script")
                assert.NotContains(t, comment.Content(), "javascript:")
                assert.NotContains(t, comment.Content(), "onerror")
            }
        })
    }
}

// Path Traversal Tests
func TestStorage_PreventPathTraversal(t *testing.T) {
    t.Parallel()

    maliciousPaths := []string{
        "../../../etc/passwd",
        "..\\..\\..\\windows\\system32\\config\\sam",
        "....//....//....//etc/passwd",
        "/etc/passwd",
        "C:\\Windows\\System32\\config\\sam",
    }

    for _, path := range maliciousPaths {
        t.Run(path, func(t *testing.T) {
            err := storageService.Put(ctx, path, []byte("data"))

            // Should reject or sanitize
            require.Error(t, err, "Path traversal should be prevented")
        })
    }
}
```

### 1.4 File Upload Security Tests

**File**: `tests/security/unit/upload_test.go`

```go
// File Upload Attack Tests
func TestUpload_RejectsOversizedFile(t *testing.T) {
    t.Parallel()

    // 11MB file (exceeds 10MB limit)
    largeFile := make([]byte, 11*1024*1024)

    _, err := uploadService.ValidateUpload(ctx, largeFile)

    require.ErrorIs(t, err, gallery.ErrFileTooLarge)
}

func TestUpload_RejectsMalware(t *testing.T) {
    t.Parallel()

    // EICAR test file (standard malware test signature)
    eicarFile := []byte(`X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*`)

    _, err := uploadService.ProcessUpload(ctx, ProcessUploadCommand{
        Data:     eicarFile,
        Filename: "test.jpg",
        UserID:   userID,
    })

    require.ErrorIs(t, err, gallery.ErrMalwareDetected)
}

func TestUpload_RejectsPolyglotFile(t *testing.T) {
    t.Parallel()

    // Polyglot JPEG/HTML file
    polyglot := loadTestFile(t, "fixtures/polyglot_jpeg_html.jpg")

    result, err := uploadService.ProcessUpload(ctx, ProcessUploadCommand{
        Data:     polyglot,
        Filename: "innocent.jpg",
        UserID:   userID,
    })

    // Should succeed but be re-encoded
    require.NoError(t, err)

    // Re-encoded file should not contain HTML
    processedData := storageService.Get(ctx, result.StorageKey)
    assert.NotContains(t, string(processedData), "<script")
    assert.NotContains(t, string(processedData), "<html")
}

func TestUpload_SanitizesFilename(t *testing.T) {
    t.Parallel()

    tests := []struct {
        input    string
        contains []string
        notContains []string
    }{
        {
            input:       "../../../etc/passwd.jpg",
            notContains: []string{"..", "/"},
        },
        {
            input:       "normal<script>.jpg",
            notContains: []string{"<", ">"},
        },
        {
            input:       "file|with|pipes.jpg",
            notContains: []string{"|"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            sanitized := sanitizeFilename(tt.input)

            for _, forbidden := range tt.notContains {
                assert.NotContains(t, sanitized, forbidden)
            }
        })
    }
}

func TestProcessor_StripEXIFMetadata(t *testing.T) {
    t.Parallel()

    // Image with GPS coordinates
    imageWithEXIF := loadTestFile(t, "fixtures/image_with_gps.jpg")

    processed, err := imageProcessor.Process(imageWithEXIF)
    require.NoError(t, err)

    // Verify EXIF stripped
    metadata, err := extractEXIF(processed)
    require.NoError(t, err)

    assert.Empty(t, metadata.GPS)
    assert.Empty(t, metadata.CameraModel)
    assert.Empty(t, metadata.Copyright)
}
```

---

## 2. Static Security Analysis (SAST)

### 2.1 gosec - Go Security Checker

**Installation**:
```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

**Configuration**: `.gosecrc.yaml`

```yaml
{
  "severity": "medium",
  "confidence": "medium",
  "exclude": [
    "G104",  # Unhandled errors (covered by errcheck)
  ],
  "include": [
    "G101",  # Hardcoded credentials
    "G102",  # Bind to all interfaces
    "G103",  # Unsafe code usage
    "G104",  # Unhandled errors
    "G201",  # SQL injection
    "G202",  # SQL string concatenation
    "G203",  # HTML template without escaping
    "G204",  # Command injection
    "G301",  # Poor file permissions
    "G302",  # Poor file permissions
    "G303",  # Predictable temp file
    "G304",  # File path from user input
    "G305",  # Path traversal
    "G401",  # Weak crypto (MD5, SHA1)
    "G402",  # TLS bad version
    "G403",  # Weak RSA key
    "G404",  # Weak random source
    "G501",  # Blacklisted import (crypto/md5)
    "G502",  # Blacklisted import (crypto/sha1)
    "G503",  # Blacklisted import (crypto/des)
    "G504",  # Blacklisted import (net/http/cgi)
  ]
}
```

**Usage**:
```bash
# Scan entire codebase
gosec ./...

# Scan with JSON output for CI
gosec -fmt=json -out=gosec-report.json ./...

# Scan with SARIF output for GitHub Code Scanning
gosec -fmt=sarif -out=gosec-report.sarif ./...

# Fail on high severity findings
gosec -severity=high -confidence=medium ./...
```

**CI Integration** (`.github/workflows/security.yml`):
```yaml
- name: Run gosec
  run: |
    gosec -fmt=sarif -out=gosec-report.sarif ./...

- name: Upload SARIF
  uses: github/codeql-action/upload-sarif@v2
  with:
    sarif_file: gosec-report.sarif
```

### 2.2 trivy - Vulnerability Scanner

**Installation**:
```bash
# macOS
brew install aquasecurity/trivy/trivy

# Linux
wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | sudo apt-key add -
echo "deb https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | sudo tee -a /etc/apt/sources.list.d/trivy.list
sudo apt-get update && sudo apt-get install trivy
```

**Usage**:
```bash
# Scan filesystem for vulnerabilities
trivy fs --severity HIGH,CRITICAL .

# Scan for secrets in codebase
trivy fs --security-checks secret .

# Scan Go dependencies
trivy fs --scanners vuln go.mod

# Scan container image
trivy image goimg:latest

# Generate SBOM
trivy fs --format cyclonedx --output sbom.json .
```

**CI Integration**:
```yaml
- name: Run Trivy vulnerability scanner
  uses: aquasecurity/trivy-action@master
  with:
    scan-type: 'fs'
    scan-ref: '.'
    severity: 'HIGH,CRITICAL'
    exit-code: '1'
```

### 2.3 nancy - Dependency Vulnerability Scanner

**Installation**:
```bash
go install github.com/sonatype-nexus-community/nancy@latest
```

**Usage**:
```bash
# Scan dependencies
go list -json -m all | nancy sleuth

# CI-friendly output
go list -json -m all | nancy sleuth --quiet
```

**CI Integration**:
```yaml
- name: Nancy dependency check
  run: |
    go list -json -m all | nancy sleuth
```

### 2.4 gitleaks - Secret Detection

**Installation**:
```bash
# macOS
brew install gitleaks

# Linux
wget https://github.com/zricethezav/gitleaks/releases/download/v8.18.0/gitleaks_8.18.0_linux_x64.tar.gz
tar -xzf gitleaks_8.18.0_linux_x64.tar.gz
sudo mv gitleaks /usr/local/bin/
```

**Configuration**: `.gitleaks.toml`

```toml
title = "goimg gitleaks config"

[[rules]]
id = "generic-api-key"
description = "Generic API Key"
regex = '''(?i)(api[_-]?key|apikey)['"]?\s*[:=]\s*['"]?[a-zA-Z0-9]{20,}'''
tags = ["key", "API"]

[[rules]]
id = "aws-access-key"
description = "AWS Access Key"
regex = '''AKIA[0-9A-Z]{16}'''
tags = ["key", "AWS"]

[[rules]]
id = "private-key"
description = "Private Key"
regex = '''-----BEGIN (RSA|DSA|EC|OPENSSH) PRIVATE KEY-----'''
tags = ["key", "private"]

[[rules]]
id = "jwt-token"
description = "JWT Token"
regex = '''eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9._-]{10,}'''
tags = ["key", "JWT"]

[allowlist]
paths = [
  '''tests/fixtures/''',  # Test fixtures
  '''\.md$''',             # Documentation
]
```

**Usage**:
```bash
# Scan current state
gitleaks detect --verbose

# Scan entire git history
gitleaks detect --verbose --source . --log-opts="--all"

# Scan for secrets in pre-commit hook
gitleaks protect --verbose --staged
```

**Pre-commit Hook** (`.git/hooks/pre-commit`):
```bash
#!/bin/bash
gitleaks protect --verbose --staged
```

---

## 3. Dynamic Security Testing (DAST)

### 3.1 OWASP ZAP - Baseline Scan

**Installation**:
```bash
docker pull owasp/zap2docker-stable
```

**Usage**:
```bash
# Baseline scan (passive)
docker run -v $(pwd):/zap/wrk:rw -t owasp/zap2docker-stable zap-baseline.py \
  -t http://localhost:8080 \
  -r zap-report.html

# Full scan (active + passive)
docker run -v $(pwd):/zap/wrk:rw -t owasp/zap2docker-stable zap-full-scan.py \
  -t http://localhost:8080 \
  -r zap-report.html

# API scan with OpenAPI spec
docker run -v $(pwd):/zap/wrk:rw -t owasp/zap2docker-stable zap-api-scan.py \
  -t http://localhost:8080 \
  -f openapi \
  -r zap-report.html \
  /zap/wrk/api/openapi/openapi.yaml
```

**CI Integration** (run on staging environment):
```yaml
- name: OWASP ZAP Baseline Scan
  run: |
    docker run -v $(pwd):/zap/wrk:rw -t owasp/zap2docker-stable \
      zap-baseline.py -t ${{ secrets.STAGING_URL }} -r zap-report.html

- name: Upload ZAP Report
  uses: actions/upload-artifact@v3
  with:
    name: zap-report
    path: zap-report.html
```

### 3.2 API Fuzzing

**Tool**: `go-fuzz` or `ffuf`

**Installation**:
```bash
go install github.com/dvyukov/go-fuzz/go-fuzz@latest
go install github.com/dvyukov/go-fuzz/go-fuzz-build@latest

# Or use ffuf for HTTP fuzzing
go install github.com/ffuf/ffuf@latest
```

**Example Fuzzing Target**:

```go
// fuzz_test.go
package gallery

import "testing"

func FuzzImageTitleValidation(f *testing.F) {
    // Seed corpus
    f.Add("Valid Title")
    f.Add("")
    f.Add("A")
    f.Add(strings.Repeat("A", 1000))

    f.Fuzz(func(t *testing.T, title string) {
        // Should never panic
        _, err := NewImageTitle(title)

        // If accepted, must be safe
        if err == nil {
            assert.NotContains(t, title, "<script")
            assert.LessOrEqual(t, len(title), 255)
        }
    })
}
```

**HTTP Fuzzing with ffuf**:
```bash
# Fuzz image upload endpoint
ffuf -w payloads.txt -u http://localhost:8080/api/v1/images \
  -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -d "@FUZZ" \
  -mc all \
  -fc 500
```

---

## 4. Integration Security Tests

Location: `tests/security/integration/`

### 4.1 Rate Limiting Tests

```go
func TestRateLimit_LoginEndpoint(t *testing.T) {
    suite := setupIntegrationSuite(t)

    // Attempt 6 login requests (limit is 5/min)
    for i := 0; i < 6; i++ {
        resp := suite.PostJSON("/api/v1/auth/login", map[string]string{
            "email":    "test@example.com",
            "password": "WrongPassword",
        })

        if i < 5 {
            assert.Equal(t, 401, resp.StatusCode, "Attempt %d should be allowed", i+1)
        } else {
            assert.Equal(t, 429, resp.StatusCode, "Attempt %d should be rate limited", i+1)

            // Check Retry-After header
            assert.NotEmpty(t, resp.Header.Get("Retry-After"))
        }
    }
}
```

### 4.2 End-to-End Security Scenarios

```go
func TestE2E_PrivilegeEscalation_UserToAdmin(t *testing.T) {
    suite := setupIntegrationSuite(t)

    // Create regular user
    user := suite.CreateUser("user@example.com", "User123!", identity.RoleUser)
    userToken := suite.LoginUser(user.Email, "User123!")

    // Attempt 1: Direct role update via API
    resp := suite.PutJSON("/api/v1/users/"+user.ID.String()+"/role",
        map[string]string{"role": "admin"},
        suite.WithAuth(userToken))
    assert.Equal(t, 403, resp.StatusCode, "Should not allow self role update")

    // Attempt 2: Modify JWT token
    tamperedToken := suite.TamperJWT(userToken, map[string]interface{}{
        "role": "admin",
    })
    resp = suite.GetJSON("/api/v1/admin/users", suite.WithAuth(tamperedToken))
    assert.Equal(t, 401, resp.StatusCode, "Should reject tampered token")

    // Verify user still has 'user' role
    user, _ = suite.GetUser(user.ID)
    assert.Equal(t, identity.RoleUser, user.Role)
}
```

---

## 5. Security Test Automation in CI

### 5.1 GitHub Actions Workflow

**File**: `.github/workflows/security.yml`

```yaml
name: Security Scanning

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]
  schedule:
    # Run weekly on Sundays at 2am
    - cron: '0 2 * * 0'

jobs:
  security-unit-tests:
    name: Security Unit Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'

      - name: Run security tests
        run: |
          go test -v -run TestSecurity ./tests/security/unit/...
          go test -v -run TestAuth ./...
          go test -v -run TestInjection ./...

  static-analysis:
    name: Static Analysis (SAST)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Run gosec
        uses: securego/gosec@master
        with:
          args: '-fmt sarif -out gosec-results.sarif ./...'

      - name: Upload gosec SARIF
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: gosec-results.sarif

      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          severity: 'HIGH,CRITICAL'
          format: 'sarif'
          output: 'trivy-results.sarif'

      - name: Upload Trivy SARIF
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: trivy-results.sarif

  secret-scanning:
    name: Secret Detection
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
        with:
          fetch-depth: 0  # Full history for gitleaks

      - name: Run gitleaks
        uses: gitleaks/gitleaks-action@v2
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  dependency-check:
    name: Dependency Vulnerabilities
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'

      - name: Install nancy
        run: go install github.com/sonatype-nexus-community/nancy@latest

      - name: Run nancy
        run: go list -json -m all | nancy sleuth

  container-scanning:
    name: Container Image Scan
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Build Docker image
        run: docker build -t goimg:test .

      - name: Run Trivy on container
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: 'goimg:test'
          format: 'sarif'
          output: 'trivy-container.sarif'
          severity: 'HIGH,CRITICAL'

      - name: Upload container scan results
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: trivy-container.sarif
```

### 5.2 Pre-commit Hooks

**File**: `.pre-commit-config.yaml`

```yaml
repos:
  - repo: https://github.com/gitleaks/gitleaks
    rev: v8.18.0
    hooks:
      - id: gitleaks
        name: Detect secrets

  - repo: local
    hooks:
      - id: gosec
        name: Go security check
        entry: gosec
        args: ['-quiet', './...']
        language: system
        pass_filenames: false

      - id: security-tests
        name: Run security tests
        entry: bash -c 'go test -short -run TestSecurity ./tests/security/unit/...'
        language: system
        pass_filenames: false
```

**Installation**:
```bash
pip install pre-commit
pre-commit install
```

---

## 6. Security Testing Checklist

### Pre-Sprint Security Test Plan

**For each sprint, create a security test plan**:

```markdown
## Sprint X Security Test Plan

### Scope
- [ ] Features being tested: [list]
- [ ] Attack surface changes: [describe]
- [ ] New endpoints: [list]
- [ ] New dependencies: [list]

### Unit Security Tests
- [ ] Authentication tests written
- [ ] Authorization tests written
- [ ] Input validation tests written
- [ ] All tests passing

### Static Analysis
- [ ] gosec scan passing (zero high-severity)
- [ ] trivy scan passing (zero critical/high)
- [ ] gitleaks scan passing (zero secrets)
- [ ] nancy scan passing (zero high-severity)

### Integration Tests
- [ ] E2E security scenarios passing
- [ ] Rate limiting validated
- [ ] CORS configuration tested
- [ ] Security headers verified

### Manual Testing
- [ ] Authentication flow tested
- [ ] Authorization boundaries tested
- [ ] Input fuzzing performed
- [ ] Error handling verified

### Pass/Fail Criteria
- All automated tests MUST pass
- Zero critical/high vulnerabilities
- All manual tests documented
- Security review sign-off obtained
```

### Security Testing Coverage Matrix

| Feature | Unit Tests | SAST | DAST | Pentest | Status |
|---------|------------|------|------|---------|--------|
| Authentication | ✓ | ✓ | ✓ | ✓ | PASS |
| Authorization | ✓ | ✓ | ✓ | ✓ | PASS |
| Image Upload | ✓ | ✓ | ✓ | Pending | IN PROGRESS |
| Search | ✓ | ✓ | - | - | PARTIAL |
| Moderation | - | ✓ | - | - | TODO |

---

## 7. Vulnerability Response Workflow

### Discovery → Remediation Process

```
1. DISCOVERY
   - Automated scan finding
   - Security test failure
   - External report (bug bounty)
   - Manual code review

2. TRIAGE (within 4 hours)
   - Assess severity (CVSS score)
   - Determine exploitability
   - Check for compensating controls
   - Assign owner

3. RISK ASSESSMENT (within 24 hours)
   - Impact analysis
   - Exposure analysis
   - Business risk
   - Prioritization

4. REMEDIATION (varies by severity)
   Critical: 24 hours
   High: 7 days
   Medium: 30 days
   Low: 90 days

5. VALIDATION
   - Fix verified in code
   - Security test added
   - Regression test passed
   - Scan confirms resolution

6. DOCUMENTATION
   - Post-mortem (critical/high)
   - Lessons learned
   - Process improvements
```

### Vulnerability Ticket Template

```markdown
## Vulnerability: [Title]

**Severity**: Critical / High / Medium / Low
**CVSS Score**: X.X
**CWE**: CWE-XXX
**Discovered**: YYYY-MM-DD
**Discoverer**: [Name/Tool]

### Description
[Detailed description of vulnerability]

### Location
- File: `path/to/file.go`
- Line: XXX
- Function: `FunctionName()`

### Impact
- Confidentiality: High / Medium / Low
- Integrity: High / Medium / Low
- Availability: High / Medium / Low

### Exploitability
- [ ] Publicly known exploit
- [ ] Easy to exploit
- [ ] Requires authentication
- [ ] Requires special access
- [ ] Requires user interaction

### Reproduction Steps
1. Step 1
2. Step 2
3. Expected: [malicious outcome]

### Remediation
**Recommendation**: [Specific fix]
**Alternative**: [If applicable]

### Timeline
- Discovered: YYYY-MM-DD
- Triaged: YYYY-MM-DD
- Fix deployed: YYYY-MM-DD (target)
- Validated: YYYY-MM-DD

### Compensating Controls
- [List any temporary mitigations]
```

---

## 8. Security Testing Tools Summary

| Tool | Type | Purpose | CI Integration | Frequency |
|------|------|---------|----------------|-----------|
| `go test` | Unit | Security test cases | ✓ | Every commit |
| `gosec` | SAST | Code vulnerability scan | ✓ | Every commit |
| `trivy` | SAST | Dependency & container scan | ✓ | Every commit |
| `nancy` | SAST | Go dependency vulnerabilities | ✓ | Daily |
| `gitleaks` | Secret Detection | Detect hardcoded secrets | ✓ | Every commit |
| OWASP ZAP | DAST | Dynamic API scanning | ✓ | Nightly (staging) |
| `ffuf` | Fuzzing | HTTP endpoint fuzzing | Manual | Weekly |
| Burp Suite | Manual | Penetration testing | Manual | Quarterly |
| Nmap | Recon | Network scanning | Manual | Monthly |

---

## 9. Security Testing Best Practices

### DO:
- Write security tests alongside feature code
- Automate everything possible
- Test negative cases (rejection scenarios)
- Use test fixtures for malicious payloads
- Run security scans in CI before merge
- Document security test coverage
- Review scan results regularly
- Treat security test failures as P0 bugs

### DON'T:
- Skip security tests to meet deadlines
- Disable security checks without review
- Ignore low-severity findings indefinitely
- Test only happy paths
- Use production data in security tests
- Commit sensitive test credentials
- Run destructive tests against production

---

## 10. Additional Resources

### OWASP Testing Guide
- https://owasp.org/www-project-web-security-testing-guide/

### Go Security Best Practices
- https://github.com/OWASP/Go-SCP

### CVE Databases
- https://nvd.nist.gov/
- https://cve.mitre.org/

### Security Checklists
- OWASP ASVS: https://owasp.org/www-project-application-security-verification-standard/
- SANS Top 25: https://www.sans.org/top25-software-errors/

---

## Appendix: Sample Test Payloads

### SQL Injection Payloads
```
' OR '1'='1
'; DROP TABLE users--
1' UNION SELECT NULL--
admin'--
' OR 1=1--
```

### XSS Payloads
```
<script>alert('XSS')</script>
<img src=x onerror=alert('XSS')>
javascript:alert('XSS')
<svg onload=alert('XSS')>
```

### Path Traversal Payloads
```
../../../etc/passwd
....//....//....//etc/passwd
..%2F..%2F..%2Fetc%2Fpasswd
```

### Command Injection Payloads
```
; ls -la
| cat /etc/passwd
`whoami`
$(id)
```
