# Sprint 11: Two-Factor Authentication - Test Strategy

> Comprehensive test plan for TOTP-based 2FA implementation with 90%+ coverage target.
> **Sprint Goal**: Secure, reliable 2FA with backup codes and session elevation.

---

## Executive Summary

This document defines the testing strategy for Sprint 11's Two-Factor Authentication (TOTP) implementation. Building on Sprint 10's successful 90.9% coverage achievement, we target 90%+ coverage across all layers with emphasis on cryptographic security, timing resistance, and rate limiting validation.

### Test Coverage Targets

| Component | Target | Priority | Rationale |
|-----------|--------|----------|-----------|
| TOTP Domain Logic | 95% | P0 | Cryptographic correctness is critical |
| Backup Code Service | 95% | P0 | Security-critical password recovery |
| Application Handlers | 90% | P0 | User-facing 2FA workflows |
| Infrastructure (Redis) | 80% | P1 | Rate limiting and state storage |
| HTTP Handlers | 85% | P1 | API contract compliance |
| E2E Coverage | 100% | P0 | All 2FA endpoints must be tested |

### Test Distribution (Test Pyramid)

```
                    ┌─────────────────────┐
                    │   E2E Tests         │  ~15 tests (10%)
                    │   Newman/Postman    │  All 2FA flows
                    ├─────────────────────┤
                    │ Integration Tests   │  ~30 tests (20%)
                    │ Redis + DB          │  Rate limiting
                    ├─────────────────────┤
                    │   Unit Tests        │  ~100 tests (70%)
                    │   Domain + App      │  TOTP generation
                    └─────────────────────┘
```

---

## 1. Unit Tests (Target: 90-95% Coverage)

### 1.1 TOTP Generation and Verification

**File**: `internal/domain/identity/totp_test.go`

#### Test Cases

##### 1.1.1 Secret Generation

```go
func TestTOTP_GenerateSecret_ReturnsValidBase32(t *testing.T)
func TestTOTP_GenerateSecret_ProducesUniqueSecrets(t *testing.T)
func TestTOTP_GenerateSecret_CorrectSecretLength(t *testing.T)
```

**Coverage**:
- Secret is valid Base32 (RFC 4648)
- Secret length is 160 bits (recommended by RFC 6238)
- Sequential calls produce unique secrets (entropy test)
- Secrets are URL-safe for QR code generation

##### 1.1.2 Code Generation

```go
func TestTOTP_GenerateCode_ReturnsValidSixDigits(t *testing.T)
func TestTOTP_GenerateCode_MatchesReferenceImplementation(t *testing.T)
func TestTOTP_GenerateCode_SameSecretSameTimeSameCode(t *testing.T)
func TestTOTP_GenerateCode_DifferentTimeDifferentCode(t *testing.T)
```

**Coverage**:
- Generates 6-digit code (000000-999999)
- Matches RFC 6238 test vectors
- Deterministic: same secret + time → same code
- Time-based: different time windows → different codes

##### 1.1.3 Code Verification

```go
func TestTOTP_VerifyCode_AcceptsValidCode(t *testing.T)
func TestTOTP_VerifyCode_RejectsInvalidCode(t *testing.T)
func TestTOTP_VerifyCode_AcceptsWithinTimeWindow(t *testing.T)
func TestTOTP_VerifyCode_RejectsOutsideTimeWindow(t *testing.T)
func TestTOTP_VerifyCode_TimeWindowSkewTolerance(t *testing.T)
```

**Coverage**:
- Valid code within current time window succeeds
- Invalid code fails
- Codes within ±1 time step accepted (skew tolerance)
- Codes outside ±1 time step rejected
- Clock drift handling (configurable skew)

**Table-Driven Test Example**:

```go
func TestTOTP_VerifyCode_TimeWindows(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name       string
        code       string
        timeOffset time.Duration
        wantValid  bool
    }{
        {
            name:       "current time window",
            code:       generateCodeForTime(time.Now()),
            timeOffset: 0,
            wantValid:  true,
        },
        {
            name:       "previous time window (within skew)",
            code:       generateCodeForTime(time.Now().Add(-30 * time.Second)),
            timeOffset: -30 * time.Second,
            wantValid:  true,
        },
        {
            name:       "next time window (within skew)",
            code:       generateCodeForTime(time.Now().Add(30 * time.Second)),
            timeOffset: 30 * time.Second,
            wantValid:  true,
        },
        {
            name:       "2 windows ago (outside skew)",
            code:       generateCodeForTime(time.Now().Add(-60 * time.Second)),
            timeOffset: -60 * time.Second,
            wantValid:  false,
        },
        {
            name:       "2 windows ahead (outside skew)",
            code:       generateCodeForTime(time.Now().Add(60 * time.Second)),
            timeOffset: 60 * time.Second,
            wantValid:  false,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            secret := generateTestSecret()
            result := verifyTOTPCode(secret, tt.code, time.Now().Add(tt.timeOffset))

            assert.Equal(t, tt.wantValid, result,
                "code validity mismatch for time offset %v", tt.timeOffset)
        })
    }
}
```

##### 1.1.4 QR Code Generation

```go
func TestTOTP_GenerateQRCode_ValidFormat(t *testing.T)
func TestTOTP_GenerateQRCode_CorrectURI(t *testing.T)
func TestTOTP_GenerateQRCode_CanBeScannedByAuthenticator(t *testing.T)
```

**Coverage**:
- QR code contains valid otpauth:// URI
- URI format: `otpauth://totp/{Issuer}:{AccountName}?secret={Secret}&issuer={Issuer}`
- QR code is scannable (validate image format)

---

### 1.2 Backup Code Generation and Validation

**File**: `internal/domain/identity/backup_codes_test.go`

#### Test Cases

##### 1.2.1 Code Generation

```go
func TestBackupCodes_Generate_ReturnsCorrectCount(t *testing.T)
func TestBackupCodes_Generate_UniqueAcrossSet(t *testing.T)
func TestBackupCodes_Generate_CorrectFormat(t *testing.T)
func TestBackupCodes_Generate_SufficientEntropy(t *testing.T)
```

**Coverage**:
- Generates 10 backup codes per set
- All codes in set are unique
- Each code is 8 characters (uppercase alphanumeric, excluding ambiguous chars)
- Uses crypto/rand for secure generation
- No similar-looking characters (0/O, 1/I/l)

**Entropy Test**:

```go
func TestBackupCodes_Generate_SufficientEntropy(t *testing.T) {
    t.Parallel()

    const iterations = 1000
    codes := make(map[string]bool)

    for i := 0; i < iterations; i++ {
        set := GenerateBackupCodes()
        for _, code := range set {
            codes[code] = true
        }
    }

    // With 8-char codes from 32-char alphabet: 32^8 = ~1.2e12 possibilities
    // Expect very few collisions over 10,000 codes
    assert.Greater(t, len(codes), iterations*9,
        "insufficient entropy: too many collisions")
}
```

##### 1.2.2 Code Hashing

```go
func TestBackupCodes_Hash_UsesBcrypt(t *testing.T)
func TestBackupCodes_Hash_DifferentHashesForSameCode(t *testing.T)
func TestBackupCodes_Hash_VerifyWithBcrypt(t *testing.T)
```

**Coverage**:
- Hashes use bcrypt with cost 12
- Multiple hashes of same code produce different hashes (salt)
- `bcrypt.CompareHashAndPassword` can verify hashes

##### 1.2.3 Code Validation

```go
func TestBackupCodes_Verify_AcceptsValidCode(t *testing.T)
func TestBackupCodes_Verify_RejectsInvalidCode(t *testing.T)
func TestBackupCodes_Verify_RejectsUsedCode(t *testing.T)
func TestBackupCodes_Verify_CaseInsensitiveComparison(t *testing.T)
```

**Coverage**:
- Valid code matches hash
- Invalid code fails verification
- Used code (marked as consumed) rejected
- Case-insensitive comparison ("ABCD1234" == "abcd1234")

---

### 1.3 Domain Entity Tests

**File**: `internal/domain/identity/user_2fa_test.go`

#### Test Cases

##### 1.3.1 User 2FA State

```go
func TestUser_Enable2FA_SuccessfullyEnables(t *testing.T)
func TestUser_Enable2FA_AlreadyEnabledFails(t *testing.T)
func TestUser_Disable2FA_SuccessfullyDisables(t *testing.T)
func TestUser_Disable2FA_AlreadyDisabledFails(t *testing.T)
func TestUser_Has2FAEnabled_ReturnsCorrectState(t *testing.T)
```

**Coverage**:
- Enable 2FA sets flag and emits event
- Enabling when already enabled returns error
- Disable 2FA clears flag and emits event
- Disabling when already disabled returns error
- Status query returns correct boolean

##### 1.3.2 Session Elevation

```go
func TestSession_Require2FA_SetsElevationFlag(t *testing.T)
func TestSession_IsElevated_ReturnsFalseBeforeVerification(t *testing.T)
func TestSession_IsElevated_ReturnsTrueAfterVerification(t *testing.T)
func TestSession_ElevationExpiry_ExpiresAfter15Minutes(t *testing.T)
```

**Coverage**:
- Session created for 2FA-enabled user requires elevation
- Session not elevated until 2FA verified
- Session elevated after successful 2FA verification
- Elevation expires after 15 minutes (re-prompt for sensitive operations)

---

### 1.4 Application Layer Tests

**File**: `internal/application/identity/commands/setup_2fa_test.go`

#### Test Cases

##### 1.4.1 2FA Setup Command

```go
func TestSetup2FAHandler_Handle_GeneratesSecretAndQR(t *testing.T)
func TestSetup2FAHandler_Handle_ReturnsBackupCodes(t *testing.T)
func TestSetup2FAHandler_Handle_RequiresPasswordConfirmation(t *testing.T)
func TestSetup2FAHandler_Handle_AlreadyEnabledFails(t *testing.T)
```

**Coverage**:
- Generates TOTP secret
- Generates QR code
- Generates 10 backup codes
- Requires current password for authorization
- Fails if 2FA already enabled

##### 1.4.2 2FA Verification Command

```go
func TestVerify2FASetupHandler_Handle_ValidCodeEnables2FA(t *testing.T)
func TestVerify2FASetupHandler_Handle_InvalidCodeFails(t *testing.T)
func TestVerify2FASetupHandler_Handle_CodeReuseBlocked(t *testing.T)
func TestVerify2FASetupHandler_Handle_RateLimited(t *testing.T)
```

**Coverage**:
- Valid TOTP code enables 2FA and persists secret
- Invalid code returns error (does not enable)
- Same code cannot be used twice (replay protection)
- Rate limiting enforced (5 attempts per 5 minutes)

##### 1.4.3 2FA Login Verification Command

```go
func TestVerify2FALoginHandler_Handle_ValidCodeElevatesSession(t *testing.T)
func TestVerify2FALoginHandler_Handle_BackupCodeWorks(t *testing.T)
func TestVerify2FALoginHandler_Handle_BackupCodeMarkedUsed(t *testing.T)
func TestVerify2FALoginHandler_Handle_RateLimited(t *testing.T)
```

**Coverage**:
- Valid TOTP code elevates session
- Backup code works as alternative
- Backup code marked as used after verification
- Rate limiting enforced (5 attempts per 5 minutes)

##### 1.4.4 2FA Disable Command

```go
func TestDisable2FAHandler_Handle_RequiresPasswordAndCode(t *testing.T)
func TestDisable2FAHandler_Handle_RevokesSecret(t *testing.T)
func TestDisable2FAHandler_Handle_InvalidatesBackupCodes(t *testing.T)
func TestDisable2FAHandler_Handle_EmitsAuditEvent(t *testing.T)
```

**Coverage**:
- Requires current password + valid 2FA code
- Deletes TOTP secret from database
- Marks all backup codes as invalid
- Emits User2FADisabled event for audit

---

## 2. Integration Tests (Target: 80% Coverage)

### 2.1 Database Tests

**File**: `tests/integration/totp_repository_test.go`

#### Test Cases

##### 2.1.1 TOTP Secret Persistence

```go
func TestTOTPRepository_SaveSecret_PersistsCorrectly(t *testing.T)
func TestTOTPRepository_GetSecret_RetrievesCorrectly(t *testing.T)
func TestTOTPRepository_UpdateSecret_ModifiesExisting(t *testing.T)
func TestTOTPRepository_DeleteSecret_RemovesFromDB(t *testing.T)
func TestTOTPRepository_SecretEncryptedAtRest(t *testing.T)
```

**Coverage**:
- Save secret to `user_totp_secrets` table
- Retrieve secret by user ID
- Update existing secret (regeneration)
- Delete secret on 2FA disable
- Secrets encrypted with AES-256-GCM

**Database Schema**:

```sql
CREATE TABLE user_totp_secrets (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    encrypted_secret BYTEA NOT NULL,  -- AES-256-GCM encrypted
    nonce BYTEA NOT NULL,              -- GCM nonce
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

##### 2.1.2 Backup Codes Persistence

```go
func TestBackupCodeRepository_SaveCodes_PersistsHashes(t *testing.T)
func TestBackupCodeRepository_VerifyCode_MatchesHash(t *testing.T)
func TestBackupCodeRepository_MarkUsed_SetsUsedFlag(t *testing.T)
func TestBackupCodeRepository_ListUnusedCodes_ReturnsCount(t *testing.T)
```

**Coverage**:
- Save 10 hashed backup codes
- Verify code matches stored hash
- Mark code as used (update `used_at`)
- Count remaining unused codes

**Database Schema**:

```sql
CREATE TABLE user_backup_codes (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(255) NOT NULL,  -- bcrypt hash
    used BOOLEAN NOT NULL DEFAULT FALSE,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backup_codes_user ON user_backup_codes(user_id) WHERE NOT used;
```

##### 2.1.3 2FA Audit Log

```go
func TestAuditLog_2FASetup_Logged(t *testing.T)
func TestAuditLog_2FAVerification_Logged(t *testing.T)
func TestAuditLog_2FADisable_Logged(t *testing.T)
func TestAuditLog_FailedAttempts_Logged(t *testing.T)
```

**Coverage**:
- Log 2FA setup initiation
- Log successful 2FA verification
- Log 2FA disable
- Log failed verification attempts (security monitoring)

---

### 2.2 Redis Tests (Rate Limiting)

**File**: `tests/integration/totp_rate_limiter_test.go`

#### Test Cases

##### 2.2.1 Verification Rate Limiting

```go
func TestRateLimiter_2FAVerification_AllowsUpTo5Attempts(t *testing.T)
func TestRateLimiter_2FAVerification_Blocks6thAttempt(t *testing.T)
func TestRateLimiter_2FAVerification_ResetsAfter5Minutes(t *testing.T)
func TestRateLimiter_2FAVerification_PerUserLimiting(t *testing.T)
```

**Coverage**:
- Allow 5 verification attempts in 5 minutes
- Block 6th attempt with "rate_limited" error
- Reset counter after 5-minute window
- Rate limit is per-user (not global)

**Redis Key Pattern**:

```
goimg:2fa:ratelimit:{user_id}:verify
TTL: 300 seconds (5 minutes)
Value: Counter (1-5)
```

##### 2.2.2 Setup Rate Limiting

```go
func TestRateLimiter_2FASetup_AllowsUpTo3Attempts(t *testing.T)
func TestRateLimiter_2FASetup_Blocks4thAttempt(t *testing.T)
func TestRateLimiter_2FASetup_ResetsAfter1Hour(t *testing.T)
```

**Coverage**:
- Allow 3 setup attempts per hour (prevent abuse)
- Block 4th attempt
- Reset after 1 hour

---

### 2.3 End-to-End 2FA Flow Tests

**File**: `tests/integration/2fa_flow_test.go`

#### Test Cases

##### 2.3.1 Complete Setup Flow

```go
func Test2FA_SetupFlow_SuccessfulEnable(t *testing.T)
```

**Steps**:
1. User calls `/auth/2fa/setup` with password
2. Receives secret, QR code URL, and 10 backup codes
3. User scans QR code with authenticator app (simulated)
4. User calls `/auth/2fa/verify-setup` with code from app
5. 2FA enabled, secret persisted, backup codes hashed

**Assertions**:
- Secret stored encrypted in database
- 10 backup codes in database (all unused)
- User.has_2fa_enabled = true
- Audit log entry created

##### 2.3.2 Login with 2FA Flow

```go
func Test2FA_LoginFlow_RequiresCode(t *testing.T)
```

**Steps**:
1. User logs in with username + password
2. Server returns `2fa_required: true` (session not elevated)
3. User calls `/auth/2fa/verify-login` with TOTP code
4. Server elevates session
5. Subsequent API calls succeed

**Assertions**:
- Initial login returns non-elevated session
- Protected endpoints return 403 before 2FA verification
- After verification, protected endpoints succeed
- Session expiry resets elevation after 15 minutes

##### 2.3.3 Backup Code Usage Flow

```go
func Test2FA_BackupCode_WorksWhenTOTPUnavailable(t *testing.T)
```

**Steps**:
1. User loses access to authenticator app
2. User logs in with username + password
3. User calls `/auth/2fa/verify-login` with backup code
4. Server verifies backup code, marks as used
5. Session elevated

**Assertions**:
- Backup code successfully verifies
- Backup code marked as `used = true` in database
- Backup code cannot be reused
- Session elevated successfully

##### 2.3.4 Disable 2FA Flow

```go
func Test2FA_DisableFlow_RevokesAccessAndSecrets(t *testing.T)
```

**Steps**:
1. User calls `/auth/2fa/disable` with password + TOTP code
2. Server verifies credentials and code
3. Server deletes TOTP secret
4. Server invalidates all backup codes
5. 2FA disabled for user

**Assertions**:
- TOTP secret deleted from database
- All backup codes marked as invalid
- User.has_2fa_enabled = false
- Audit log entry created

---

## 3. E2E Tests (Newman/Postman) - 100% Endpoint Coverage

**File**: `tests/e2e/postman/goimg-api.postman_collection.json`

### 3.1 2FA Setup Flow

#### Request: Setup 2FA

```json
{
    "name": "POST /auth/2fa/setup",
    "request": {
        "method": "POST",
        "header": [
            {"key": "Authorization", "value": "Bearer {{access_token}}"}
        ],
        "body": {
            "mode": "raw",
            "raw": "{\"password\": \"{{user_password}}\"}"
        },
        "url": "{{base_url}}/api/v1/auth/2fa/setup"
    },
    "event": [
        {
            "listen": "test",
            "script": {
                "exec": [
                    "pm.test('Status code is 200', function() {",
                    "    pm.response.to.have.status(200);",
                    "});",
                    "",
                    "pm.test('Response has secret and QR code', function() {",
                    "    const json = pm.response.json();",
                    "    pm.expect(json).to.have.property('secret');",
                    "    pm.expect(json).to.have.property('qr_code_url');",
                    "    pm.expect(json).to.have.property('backup_codes');",
                    "    pm.expect(json.backup_codes).to.be.an('array').with.length(10);",
                    "    ",
                    "    pm.collectionVariables.set('totp_secret', json.secret);",
                    "    pm.collectionVariables.set('backup_code', json.backup_codes[0]);",
                    "});",
                    "",
                    "pm.test('Secret is valid base32', function() {",
                    "    const secret = pm.response.json().secret;",
                    "    pm.expect(secret).to.match(/^[A-Z2-7]+=*$/);",
                    "});",
                    "",
                    "pm.test('Backup codes have correct format', function() {",
                    "    const codes = pm.response.json().backup_codes;",
                    "    codes.forEach(code => {",
                    "        pm.expect(code).to.match(/^[A-Z0-9]{8}$/);",
                    "    });",
                    "});"
                ]
            }
        }
    ]
}
```

#### Request: Verify 2FA Setup

```json
{
    "name": "POST /auth/2fa/verify-setup",
    "request": {
        "method": "POST",
        "header": [
            {"key": "Authorization", "value": "Bearer {{access_token}}"}
        ],
        "body": {
            "mode": "raw",
            "raw": "{\"code\": \"{{generated_totp_code}}\"}"
        },
        "url": "{{base_url}}/api/v1/auth/2fa/verify-setup"
    },
    "event": [
        {
            "listen": "test",
            "script": {
                "exec": [
                    "pm.test('Status code is 200', function() {",
                    "    pm.response.to.have.status(200);",
                    "});",
                    "",
                    "pm.test('2FA enabled confirmation', function() {",
                    "    const json = pm.response.json();",
                    "    pm.expect(json.enabled).to.be.true;",
                    "    pm.expect(json.message).to.include('successfully enabled');",
                    "});"
                ]
            }
        }
    ]
}
```

---

### 3.2 2FA Login Flow

#### Request: Login (2FA Prompt)

```json
{
    "name": "POST /auth/login (with 2FA)",
    "request": {
        "method": "POST",
        "body": {
            "mode": "raw",
            "raw": "{\"email\": \"{{user_email}}\", \"password\": \"{{user_password}}\"}"
        },
        "url": "{{base_url}}/api/v1/auth/login"
    },
    "event": [
        {
            "listen": "test",
            "script": {
                "exec": [
                    "pm.test('Status code is 200', function() {",
                    "    pm.response.to.have.status(200);",
                    "});",
                    "",
                    "pm.test('Requires 2FA verification', function() {",
                    "    const json = pm.response.json();",
                    "    pm.expect(json.requires_2fa).to.be.true;",
                    "    pm.expect(json.session_id).to.exist;",
                    "    pm.collectionVariables.set('session_id', json.session_id);",
                    "});"
                ]
            }
        }
    ]
}
```

#### Request: Verify 2FA Login

```json
{
    "name": "POST /auth/2fa/verify-login",
    "request": {
        "method": "POST",
        "body": {
            "mode": "raw",
            "raw": "{\"session_id\": \"{{session_id}}\", \"code\": \"{{generated_totp_code}}\"}"
        },
        "url": "{{base_url}}/api/v1/auth/2fa/verify-login"
    },
    "event": [
        {
            "listen": "test",
            "script": {
                "exec": [
                    "pm.test('Status code is 200', function() {",
                    "    pm.response.to.have.status(200);",
                    "});",
                    "",
                    "pm.test('Returns access and refresh tokens', function() {",
                    "    const json = pm.response.json();",
                    "    pm.expect(json.access_token).to.exist;",
                    "    pm.expect(json.refresh_token).to.exist;",
                    "    pm.collectionVariables.set('access_token', json.access_token);",
                    "});"
                ]
            }
        }
    ]
}
```

---

### 3.3 Backup Code Flow

#### Request: Use Backup Code

```json
{
    "name": "POST /auth/2fa/verify-login (backup code)",
    "request": {
        "method": "POST",
        "body": {
            "mode": "raw",
            "raw": "{\"session_id\": \"{{session_id}}\", \"backup_code\": \"{{backup_code}}\"}"
        },
        "url": "{{base_url}}/api/v1/auth/2fa/verify-login"
    },
    "event": [
        {
            "listen": "test",
            "script": {
                "exec": [
                    "pm.test('Status code is 200', function() {",
                    "    pm.response.to.have.status(200);",
                    "});",
                    "",
                    "pm.test('Backup code successfully verifies', function() {",
                    "    const json = pm.response.json();",
                    "    pm.expect(json.access_token).to.exist;",
                    "});",
                    "",
                    "pm.test('Warning about backup code usage', function() {",
                    "    const json = pm.response.json();",
                    "    pm.expect(json.warning).to.include('backup code has been used');",
                    "});"
                ]
            }
        }
    ]
}
```

#### Request: Reuse Backup Code (Should Fail)

```json
{
    "name": "POST /auth/2fa/verify-login (reuse backup code)",
    "request": {
        "method": "POST",
        "body": {
            "mode": "raw",
            "raw": "{\"session_id\": \"{{new_session_id}}\", \"backup_code\": \"{{backup_code}}\"}"
        },
        "url": "{{base_url}}/api/v1/auth/2fa/verify-login"
    },
    "event": [
        {
            "listen": "test",
            "script": {
                "exec": [
                    "pm.test('Status code is 401', function() {",
                    "    pm.response.to.have.status(401);",
                    "});",
                    "",
                    "pm.test('Error indicates code already used', function() {",
                    "    const json = pm.response.json();",
                    "    pm.expect(json.title).to.include('Invalid Code');",
                    "    pm.expect(json.detail).to.include('already been used');",
                    "});"
                ]
            }
        }
    ]
}
```

---

### 3.4 Error Scenarios

#### Request: Invalid TOTP Code

```json
{
    "name": "POST /auth/2fa/verify-login (invalid code)",
    "request": {
        "method": "POST",
        "body": {
            "mode": "raw",
            "raw": "{\"session_id\": \"{{session_id}}\", \"code\": \"000000\"}"
        },
        "url": "{{base_url}}/api/v1/auth/2fa/verify-login"
    },
    "event": [
        {
            "listen": "test",
            "script": {
                "exec": [
                    "pm.test('Status code is 401', function() {",
                    "    pm.response.to.have.status(401);",
                    "});",
                    "",
                    "pm.test('Error is RFC 7807 compliant', function() {",
                    "    const json = pm.response.json();",
                    "    pm.expect(json).to.have.property('type');",
                    "    pm.expect(json).to.have.property('title');",
                    "    pm.expect(json.title).to.include('Invalid Code');",
                    "});"
                ]
            }
        }
    ]
}
```

#### Request: Rate Limited

```json
{
    "name": "POST /auth/2fa/verify-login (rate limited)",
    "request": {
        "method": "POST",
        "body": {
            "mode": "raw",
            "raw": "{\"session_id\": \"{{session_id}}\", \"code\": \"123456\"}"
        },
        "url": "{{base_url}}/api/v1/auth/2fa/verify-login"
    },
    "event": [
        {
            "listen": "prerequest",
            "script": {
                "exec": [
                    "// Send 5 invalid attempts first",
                    "for (let i = 0; i < 5; i++) {",
                    "    pm.sendRequest({",
                    "        url: pm.environment.get('base_url') + '/api/v1/auth/2fa/verify-login',",
                    "        method: 'POST',",
                    "        body: { mode: 'raw', raw: JSON.stringify({ session_id: pm.collectionVariables.get('session_id'), code: '111111' }) }",
                    "    });",
                    "}"
                ]
            }
        },
        {
            "listen": "test",
            "script": {
                "exec": [
                    "pm.test('Status code is 429', function() {",
                    "    pm.response.to.have.status(429);",
                    "});",
                    "",
                    "pm.test('Rate limit headers present', function() {",
                    "    pm.expect(pm.response.headers.get('X-RateLimit-Limit')).to.equal('5');",
                    "    pm.expect(pm.response.headers.get('X-RateLimit-Remaining')).to.equal('0');",
                    "    pm.expect(pm.response.headers.get('Retry-After')).to.exist;",
                    "});"
                ]
            }
        }
    ]
}
```

---

### 3.5 2FA Disable Flow

#### Request: Disable 2FA

```json
{
    "name": "DELETE /auth/2fa",
    "request": {
        "method": "DELETE",
        "header": [
            {"key": "Authorization", "value": "Bearer {{access_token}}"}
        ],
        "body": {
            "mode": "raw",
            "raw": "{\"password\": \"{{user_password}}\", \"code\": \"{{generated_totp_code}}\"}"
        },
        "url": "{{base_url}}/api/v1/auth/2fa"
    },
    "event": [
        {
            "listen": "test",
            "script": {
                "exec": [
                    "pm.test('Status code is 200', function() {",
                    "    pm.response.to.have.status(200);",
                    "});",
                    "",
                    "pm.test('2FA disabled confirmation', function() {",
                    "    const json = pm.response.json();",
                    "    pm.expect(json.enabled).to.be.false;",
                    "    pm.expect(json.message).to.include('disabled successfully');",
                    "});"
                ]
            }
        }
    ]
}
```

---

## 4. Security Tests

### 4.1 Brute Force Resistance

**File**: `tests/security/2fa_brute_force_test.go`

#### Test Cases

```go
func TestSecurity_BruteForce_RateLimitPreventsGuessing(t *testing.T)
func TestSecurity_BruteForce_IPBanAfter10FailedAttempts(t *testing.T)
func TestSecurity_BruteForce_AccountLockAfter20FailedAttempts(t *testing.T)
```

**Coverage**:
- Rate limiting blocks after 5 attempts in 5 minutes
- IP temporarily banned after 10 failed attempts (24 hours)
- Account locked after 20 failed attempts (manual unlock required)

**Calculation**:
- 6-digit TOTP: 1,000,000 possibilities
- With 5 attempts per 5 minutes: 60 attempts per hour
- Brute force time: ~1,900 years (infeasible)

---

### 4.2 Timing Attack Resistance

**File**: `tests/security/2fa_timing_test.go`

#### Test Cases

```go
func TestSecurity_Timing_ConstantTimeVerification(t *testing.T)
func TestSecurity_Timing_NoLeakageOnInvalidCode(t *testing.T)
func TestSecurity_Timing_NoLeakageOnNonExistentUser(t *testing.T)
```

**Coverage**:
- Code verification uses constant-time comparison
- Invalid code response time same as valid code
- Non-existent user response time same as existing user

**Statistical Test**:

```go
func TestSecurity_Timing_NoStatisticalDifference(t *testing.T) {
    validTimings := make([]time.Duration, 100)
    invalidTimings := make([]time.Duration, 100)

    for i := 0; i < 100; i++ {
        start := time.Now()
        verifyCode(validCode)
        validTimings[i] = time.Since(start)

        start = time.Now()
        verifyCode(invalidCode)
        invalidTimings[i] = time.Since(start)
    }

    // Mann-Whitney U test: no significant difference
    pValue := mannWhitneyUTest(validTimings, invalidTimings)
    assert.Greater(t, pValue, 0.05,
        "timing difference is statistically significant (timing leak)")
}
```

---

### 4.3 Session Elevation Tests

**File**: `tests/security/session_elevation_test.go`

#### Test Cases

```go
func TestSecurity_Elevation_RequiredFor2FAUser(t *testing.T)
func TestSecurity_Elevation_NotRequiredForNon2FAUser(t *testing.T)
func TestSecurity_Elevation_ExpiresAfter15Minutes(t *testing.T)
func TestSecurity_Elevation_RequiredForSensitiveOperations(t *testing.T)
```

**Coverage**:
- Users with 2FA enabled require session elevation
- Users without 2FA don't require elevation (normal session)
- Elevated session expires after 15 minutes of inactivity
- Sensitive operations (password change, 2FA disable) require elevation

**Sensitive Operations**:
- Change password
- Change email
- Disable 2FA
- Delete account
- Export user data

---

### 4.4 TOTP Time Window Tests

**File**: `tests/security/totp_time_window_test.go`

#### Test Cases

```go
func TestSecurity_TimeWindow_AcceptsCurrent30SecondWindow(t *testing.T)
func TestSecurity_TimeWindow_AcceptsPrevious30SecondWindow(t *testing.T)
func TestSecurity_TimeWindow_AcceptsNext30SecondWindow(t *testing.T)
func TestSecurity_TimeWindow_Rejects2WindowsAgo(t *testing.T)
func TestSecurity_TimeWindow_Rejects2WindowsAhead(t *testing.T)
```

**Coverage**:
- Accept codes from current 30-second window (T)
- Accept codes from previous window (T-1) for clock skew
- Accept codes from next window (T+1) for clock skew
- Reject codes from 2+ windows ago (T-2)
- Reject codes from 2+ windows ahead (T+2)

**Time Window Configuration**:
- Window size: 30 seconds (RFC 6238 default)
- Skew tolerance: ±1 window (90 seconds total acceptance)

---

## 5. Test Coverage Targets by Component

### 5.1 Coverage Breakdown

| Component | File | Target | Lines | Branches |
|-----------|------|--------|-------|----------|
| **TOTP Core** | `internal/domain/identity/totp.go` | 95% | 150 | 40 |
| **Backup Codes** | `internal/domain/identity/backup_codes.go` | 95% | 120 | 30 |
| **User 2FA Methods** | `internal/domain/identity/user_2fa.go` | 90% | 80 | 20 |
| **Setup Handler** | `internal/application/identity/commands/setup_2fa.go` | 90% | 200 | 50 |
| **Verify Handler** | `internal/application/identity/commands/verify_2fa.go` | 90% | 180 | 45 |
| **Login Handler** | `internal/application/identity/commands/verify_2fa_login.go` | 90% | 150 | 40 |
| **Disable Handler** | `internal/application/identity/commands/disable_2fa.go` | 90% | 100 | 25 |
| **TOTP Repository** | `internal/infrastructure/persistence/totp_repository.go` | 80% | 250 | 60 |
| **Backup Code Repo** | `internal/infrastructure/persistence/backup_code_repository.go` | 80% | 200 | 50 |
| **Rate Limiter** | `internal/infrastructure/redis/2fa_rate_limiter.go` | 80% | 120 | 30 |
| **HTTP Handlers** | `internal/interfaces/http/handlers/2fa_handler.go` | 85% | 300 | 70 |
| **E2E Tests** | `tests/e2e/postman/*` | 100% | N/A | N/A |

### 5.2 Minimum Coverage Requirements

**Overall Sprint 11 Coverage**: 90%

**Critical Path Coverage** (must be 100%):
- TOTP code generation algorithm
- TOTP code verification algorithm
- Backup code verification
- Rate limiting enforcement
- Secret encryption/decryption
- Session elevation logic

**Non-Critical Coverage** (can be <90%):
- QR code generation (visual output)
- Audit logging (already tested in Sprint 9)
- HTTP error mapping (already tested in Sprint 4)

---

## 6. Test Execution Strategy

### 6.1 Test Phases

#### Phase 1: Unit Tests (Days 1-3)

**Order**:
1. TOTP core functions (generation, verification)
2. Backup code generation and validation
3. Domain entity methods (User.Enable2FA, etc.)
4. Application command handlers
5. Infrastructure repositories

**Execution**:
```bash
make test-unit
go test -v -race -cover ./internal/domain/identity/*2fa*
go test -v -race -cover ./internal/application/identity/commands/*2fa*
```

**Coverage Check**:
```bash
go test -coverprofile=coverage.out ./internal/...
go tool cover -func=coverage.out | grep "2fa"
```

#### Phase 2: Integration Tests (Days 4-5)

**Order**:
1. Database tests (secrets, backup codes)
2. Redis tests (rate limiting)
3. End-to-end flow tests

**Execution**:
```bash
make test-integration
docker-compose -f docker/docker-compose.yml up -d postgres redis
go test -v -tags=integration ./tests/integration/*2fa*
```

#### Phase 3: E2E Tests (Days 6-7)

**Order**:
1. 2FA setup flow
2. 2FA login flow
3. Backup code flow
4. Error scenarios
5. Rate limiting

**Execution**:
```bash
make test-e2e
newman run tests/e2e/postman/goimg-api.postman_collection.json \
    --folder "2FA Tests" \
    --environment tests/e2e/postman/ci.postman_environment.json \
    --reporters cli,junit
```

#### Phase 4: Security Tests (Days 8-9)

**Order**:
1. Brute force resistance
2. Timing attack resistance
3. Session elevation
4. Time window validation

**Execution**:
```bash
go test -v -tags=security ./tests/security/*2fa*
gosec -tests ./internal/.../2fa*
```

---

### 6.2 CI Pipeline Integration

**GitHub Actions Workflow** (`.github/workflows/sprint11-tests.yml`):

```yaml
name: Sprint 11 2FA Tests

on:
  push:
    branches: [sprint-11-*]
  pull_request:
    branches: [main, develop]

jobs:
  unit-tests:
    name: Unit Tests (2FA)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run 2FA Unit Tests
        run: |
          go test -v -race -coverprofile=coverage.out \
            ./internal/domain/identity/*2fa* \
            ./internal/application/identity/commands/*2fa* \
            ./internal/infrastructure/persistence/*2fa*

      - name: Check Coverage
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$COVERAGE < 90.0" | bc -l) )); then
            echo "Coverage $COVERAGE% is below 90% target"
            exit 1
          fi
          echo "Coverage: $COVERAGE% ✓"

  integration-tests:
    name: Integration Tests (2FA)
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_PASSWORD: test
          POSTGRES_DB: goimg_test
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
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run Migrations
        run: make migrate-up
        env:
          DATABASE_URL: postgres://postgres:test@localhost:5432/goimg_test?sslmode=disable

      - name: Run Integration Tests
        run: go test -v -tags=integration ./tests/integration/*2fa*
        env:
          DATABASE_URL: postgres://postgres:test@localhost:5432/goimg_test?sslmode=disable
          REDIS_URL: redis://localhost:6379

  e2e-tests:
    name: E2E Tests (2FA)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Start Services
        run: docker-compose -f docker/docker-compose.yml up -d

      - name: Wait for API
        run: |
          timeout 60 bash -c 'until curl -f http://localhost:8080/health; do sleep 2; done'

      - name: Install Newman
        run: npm install -g newman

      - name: Run 2FA E2E Tests
        run: |
          newman run tests/e2e/postman/goimg-api.postman_collection.json \
            --folder "2FA Tests" \
            --environment tests/e2e/postman/ci.postman_environment.json \
            --reporters cli,junit \
            --reporter-junit-export junit-report.xml

      - name: Upload Test Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: e2e-test-results
          path: junit-report.xml

  security-tests:
    name: Security Tests (2FA)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run Security Tests
        run: go test -v -tags=security ./tests/security/*2fa*

      - name: Run gosec
        run: |
          go install github.com/securego/gosec/v2/cmd/gosec@latest
          gosec -tests ./internal/.../2fa* ./tests/security/*2fa*
```

---

## 7. Test Data Management

### 7.1 Test Fixtures

**TOTP Test Vectors** (RFC 6238 Appendix B):

```go
// tests/fixtures/totp_vectors.go
var RFC6238TestVectors = []struct {
    Secret    string
    Timestamp int64
    Code      string
}{
    {
        Secret:    "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", // "12345678901234567890" base32
        Timestamp: 59,                                  // 1970-01-01 00:00:59 UTC
        Code:      "94287082",
    },
    {
        Secret:    "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ",
        Timestamp: 1111111109,
        Code:      "07081804",
    },
    {
        Secret:    "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ",
        Timestamp: 1234567890,
        Code:      "89005924",
    },
}
```

**Test User Fixtures**:

```go
// tests/fixtures/2fa_users.go
func Create2FAEnabledUser(t *testing.T, db *sqlx.DB) *identity.User {
    t.Helper()

    user := createTestUser(t, db)
    secret := GenerateTOTPSecret()
    backupCodes := GenerateBackupCodes()

    // Enable 2FA
    user.Enable2FA(secret, backupCodes)

    // Save to DB
    repo := postgres.NewUserRepository(db)
    err := repo.Save(context.Background(), user)
    require.NoError(t, err)

    return user
}
```

---

### 7.2 Mock Implementations

**Mock TOTP Service**:

```go
// tests/mocks/totp_service_mock.go
type MockTOTPService struct {
    mock.Mock
}

func (m *MockTOTPService) GenerateSecret() (string, error) {
    args := m.Called()
    return args.String(0), args.Error(1)
}

func (m *MockTOTPService) GenerateQRCodeURL(secret, issuer, account string) (string, error) {
    args := m.Called(secret, issuer, account)
    return args.String(0), args.Error(1)
}

func (m *MockTOTPService) VerifyCode(secret, code string, timestamp time.Time) bool {
    args := m.Called(secret, code, timestamp)
    return args.Bool(0)
}
```

**Mock Rate Limiter**:

```go
// tests/mocks/rate_limiter_mock.go
type MockRateLimiter struct {
    mock.Mock
}

func (m *MockRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
    args := m.Called(ctx, key)
    return args.Bool(0), args.Error(1)
}

func (m *MockRateLimiter) Remaining(ctx context.Context, key string) (int, error) {
    args := m.Called(ctx, key)
    return args.Int(0), args.Error(1)
}
```

---

## 8. Performance Benchmarks

### 8.1 TOTP Performance Benchmarks

**File**: `internal/domain/identity/totp_benchmark_test.go`

```go
func BenchmarkTOTP_GenerateSecret(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _ = GenerateTOTPSecret()
    }
}
// Expected: <100µs per operation

func BenchmarkTOTP_GenerateCode(b *testing.B) {
    secret := GenerateTOTPSecret()
    timestamp := time.Now()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = GenerateTOTPCode(secret, timestamp)
    }
}
// Expected: <50µs per operation

func BenchmarkTOTP_VerifyCode(b *testing.B) {
    secret := GenerateTOTPSecret()
    code := GenerateTOTPCode(secret, time.Now())

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = VerifyTOTPCode(secret, code, time.Now())
    }
}
// Expected: <100µs per operation (constant-time comparison)

func BenchmarkBackupCodes_Generate(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _ = GenerateBackupCodes()
    }
}
// Expected: <500µs per operation (10 codes with crypto/rand)

func BenchmarkBackupCodes_Verify(b *testing.B) {
    code := "ABCD1234"
    hash, _ := bcrypt.GenerateFromPassword([]byte(code), 12)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = bcrypt.CompareHashAndPassword(hash, []byte(code))
    }
}
// Expected: ~50ms per operation (bcrypt cost 12)
```

### 8.2 Performance Targets

| Operation | Target Latency | P95 | P99 |
|-----------|----------------|-----|-----|
| Generate TOTP secret | <100µs | 80µs | 120µs |
| Generate TOTP code | <50µs | 40µs | 60µs |
| Verify TOTP code | <100µs | 80µs | 120µs |
| Generate backup codes (10) | <500µs | 400µs | 600µs |
| Verify backup code (bcrypt) | <60ms | 55ms | 70ms |
| 2FA setup endpoint | <200ms | 180ms | 250ms |
| 2FA verify endpoint | <150ms | 120ms | 200ms |

---

## 9. Test Execution Checklist

### 9.1 Pre-Merge Checklist

Before merging Sprint 11 2FA implementation:

- [ ] All unit tests pass (`go test ./internal/.../2fa*`)
- [ ] Test coverage ≥90% overall
- [ ] Critical path coverage = 100%
- [ ] All integration tests pass
- [ ] All E2E tests pass (Newman)
- [ ] All security tests pass
- [ ] Benchmarks within target latencies
- [ ] No race conditions (`go test -race`)
- [ ] `gosec` scan clean (no critical/high findings)
- [ ] OpenAPI spec updated with 2FA endpoints
- [ ] Database migrations tested (up and down)
- [ ] Redis rate limiting verified
- [ ] Audit logging complete
- [ ] Documentation updated

### 9.2 Security Gate S11 Requirements

| Control ID | Requirement | Test Evidence |
|------------|-------------|---------------|
| S11-2FA-001 | TOTP secrets encrypted at rest (AES-256-GCM) | Integration test verifies encrypted storage |
| S11-2FA-002 | Backup codes hashed (bcrypt cost 12) | Unit test verifies bcrypt hashing |
| S11-2FA-003 | Rate limiting on 2FA verification (5/5min) | Integration test verifies Redis rate limiter |
| S11-2FA-004 | Session elevation after 2FA completion | Integration test verifies elevated session |
| S11-2FA-005 | Audit logging for all 2FA events | Integration test verifies audit log entries |
| S11-2FA-006 | Time window limited to ±1 step (90s total) | Unit test verifies time window boundaries |
| S11-2FA-007 | Constant-time code verification | Security test verifies no timing leak |
| S11-2FA-008 | Brute force protection | Security test verifies rate limiting |
| S11-2FA-009 | Replay protection for codes | Unit test verifies code reuse blocked |
| S11-2FA-010 | Secure random generation (crypto/rand) | Unit test verifies entropy |

**Security Gate Approval**: All 10 controls must pass before Sprint 11 completion.

---

## 10. Test Metrics and Reporting

### 10.1 Coverage Report Format

```bash
# Generate HTML coverage report
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out -o coverage.html

# Generate function-level coverage report
go tool cover -func=coverage.out > coverage.txt

# Extract 2FA-specific coverage
grep "2fa" coverage.txt
```

**Example Output**:

```
internal/domain/identity/totp.go:15:         GenerateSecret           95.0%
internal/domain/identity/totp.go:30:         GenerateCode             100.0%
internal/domain/identity/totp.go:45:         VerifyCode               92.5%
internal/domain/identity/backup_codes.go:10: GenerateBackupCodes      98.0%
internal/domain/identity/backup_codes.go:25: VerifyBackupCode         91.0%
---
Overall 2FA Coverage: 93.5%
```

### 10.2 Test Duration Report

```bash
# Run tests with verbose timing
go test -v -json ./internal/.../2fa* | go-test-report

# Expected output:
# Unit Tests:        2.3s (150 tests)
# Integration Tests: 8.7s (30 tests)
# E2E Tests:         45.2s (15 tests)
# Total:             56.2s
```

### 10.3 Flaky Test Detection

Run tests 10 times to detect flakiness:

```bash
for i in {1..10}; do
    go test -v ./internal/.../2fa* || echo "FAILED on iteration $i"
done
```

**Acceptance Criteria**: Zero failures across 10 runs.

---

## 11. Lessons Learned from Sprint 10

### 11.1 Apply Sprint 10 Best Practices

**What Worked Well**:
1. **Statistical timing analysis** - Prevented timing leak in login delay
2. **Table-driven tests** - Covered edge cases comprehensively
3. **Benchmark tests** - Validated performance targets
4. **Mock interfaces** - Enabled isolated unit testing
5. **Testcontainers** - Realistic integration testing

**Apply to Sprint 11**:
- Use statistical tests for TOTP timing verification
- Table-driven tests for time window scenarios
- Benchmark TOTP operations for performance baseline
- Mock TOTP service for handler tests
- Testcontainers for Redis rate limiting tests

### 11.2 Avoid Sprint 10 Pitfalls

**Issues Encountered**:
1. Initial test coverage below target (fixed by adding more test cases)
2. Timing tests flaky on slow CI runners (fixed by increasing tolerance)
3. Integration tests slow (optimized with parallel execution)

**Prevention in Sprint 11**:
- Write tests alongside implementation (not after)
- Use generous timing tolerances (±50ms) for integration tests
- Parallelize tests with `t.Parallel()` where safe

---

## 12. Success Criteria

Sprint 11 testing is considered successful when:

1. **Coverage Targets Met**:
   - Overall: ≥90%
   - TOTP core: ≥95%
   - Application handlers: ≥90%
   - Infrastructure: ≥80%

2. **All Tests Pass**:
   - Unit tests: 100% pass rate
   - Integration tests: 100% pass rate
   - E2E tests: 100% pass rate
   - Security tests: 100% pass rate

3. **Performance Targets Met**:
   - All benchmarks within target latencies
   - No performance regressions from Sprint 10

4. **Security Gate S11 Approved**:
   - All 10 security controls verified
   - Zero critical/high vulnerabilities

5. **Quality Metrics**:
   - Zero flaky tests (10 consecutive runs pass)
   - Zero race conditions detected
   - 100% E2E endpoint coverage

6. **Documentation Complete**:
   - All test files have descriptive comments
   - Test strategy documented (this file)
   - Coverage reports generated

---

## 13. References

### 13.1 TOTP Standards

- [RFC 6238: TOTP](https://datatracker.ietf.org/doc/html/rfc6238)
- [RFC 4226: HOTP](https://datatracker.ietf.org/doc/html/rfc4226)
- [RFC 4648: Base32 Encoding](https://datatracker.ietf.org/doc/html/rfc4648)

### 13.2 Security Best Practices

- [OWASP 2FA Guidelines](https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html)
- [NIST SP 800-63B: Authentication](https://pages.nist.gov/800-63-3/sp800-63b.html)

### 13.3 Testing Resources

- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Testify Documentation](https://pkg.go.dev/github.com/stretchr/testify)
- [Testcontainers for Go](https://golang.testcontainers.org/)

---

**Document Version**: 1.0
**Last Updated**: 2026-01-06
**Author**: Backend Test Architect
**Approvers**: Senior Go Architect, Senior SecOps Engineer
