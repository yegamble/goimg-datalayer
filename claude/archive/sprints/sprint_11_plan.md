# Sprint 11: Two-Factor Authentication (TOTP)

> **Status**: IN PROGRESS
> **Start Date**: 2026-01-06
> **Duration**: 2 weeks
> **Sprint Goal**: Implement TOTP-based two-factor authentication with backup codes and unusual login detection

---

## Overview

Sprint 11 adds two-factor authentication (2FA) to enhance account security. Users can enable TOTP-based 2FA using authenticator apps (Google Authenticator, Authy, etc.) with backup codes for recovery.

## Key Features

| Feature | Priority | Description |
|---------|----------|-------------|
| TOTP Setup | P0 | Generate secrets, QR codes for authenticator apps |
| TOTP Verification | P0 | Verify 6-digit codes during login |
| Backup Codes | P0 | 10 one-time recovery codes |
| 2FA Management | P1 | Enable/disable 2FA, regenerate backup codes |
| Unusual Login Detection | P1 | Detect logins from new devices/locations |
| Login Notifications | P2 | Email notifications for new device logins |

---

## Technical Implementation

### Library Selection

**Recommended**: `github.com/pquerna/otp` v1.4.0

- RFC 6238 (TOTP) and RFC 4226 (HOTP) compliant
- Google Authenticator compatible
- Built-in QR code generation
- No known CVEs
- 2.8k+ GitHub stars, actively maintained

### Database Migrations

**Migration 00006: Two-Factor Authentication Tables**

```sql
-- user_totp_secrets: Encrypted TOTP secrets
CREATE TABLE user_totp_secrets (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    encrypted_secret BYTEA NOT NULL,  -- AES-256-GCM (60 bytes)
    issuer VARCHAR(100) NOT NULL DEFAULT 'goimg',
    account_name VARCHAR(255) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- user_backup_codes: Hashed backup codes (Argon2id)
CREATE TABLE user_backup_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(255) NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- user_devices: Device tracking for unusual login detection
CREATE TABLE user_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fingerprint_hash VARCHAR(64) NOT NULL,
    ip_address INET NOT NULL,
    user_agent TEXT NOT NULL,
    trusted BOOLEAN NOT NULL DEFAULT false,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/2fa/setup` | Initialize 2FA (returns QR code + backup codes) |
| POST | `/auth/2fa/verify` | Verify TOTP code (complete setup or login) |
| DELETE | `/auth/2fa` | Disable 2FA (requires password) |
| POST | `/auth/2fa/backup-codes` | Regenerate backup codes |
| GET | `/users/me/2fa/status` | Get 2FA status |

### Application Layer Commands/Queries

**Commands:**
- `Setup2FACommand` - Generate TOTP secret and backup codes
- `Verify2FACommand` - Verify TOTP code during setup
- `Verify2FALoginCommand` - Verify TOTP/backup code during login
- `Disable2FACommand` - Disable 2FA (requires password)
- `RegenerateBackupCodesCommand` - Generate new backup codes

**Queries:**
- `Get2FAStatusQuery` - Check if 2FA is enabled
- `ListTrustedDevicesQuery` - List known devices

### Session Flow

```
1. POST /auth/login (email + password)
   → If 2FA enabled: Return partial_session_token (5 min TTL)
   → If no 2FA: Return access + refresh tokens

2. POST /auth/2fa/verify (partial_session + totp_code)
   → If valid: Return access + refresh tokens (full session)
   → If invalid: Return 401

3. Protected endpoints require full session
```

---

## Security Requirements (Gate S11)

| Control ID | Requirement | Status |
|------------|-------------|--------|
| S11-2FA-001 | TOTP secrets encrypted at rest (AES-256-GCM) | Pending |
| S11-2FA-002 | Backup codes hashed (Argon2id) | Pending |
| S11-2FA-003 | Rate limiting on 2FA verification (5/min) | Pending |
| S11-2FA-004 | Session elevation after 2FA completion | Pending |
| S11-2FA-005 | Audit logging for all 2FA events | Pending |
| S11-2FA-006 | Time window validation (±1 step) | Pending |
| S11-2FA-007 | TOTP secret minimum entropy (160 bits) | Pending |
| S11-2FA-008 | Secure backup code generation (crypto/rand) | Pending |
| S11-2FA-009 | Password required to disable 2FA | Pending |
| S11-2FA-010 | TOTP replay attack prevention | Pending |

---

## Test Strategy

### Coverage Targets
- **Overall**: 90%+ (matching Sprint 10)
- **TOTP Core**: 95% (cryptographic correctness)
- **Application Handlers**: 90%
- **Infrastructure**: 80%
- **E2E**: 100% endpoint coverage

### Test Categories

**Unit Tests:**
- TOTP generation/verification (time window ±1 step)
- Backup code generation/hashing
- Domain entity tests (User 2FA state)

**Integration Tests:**
- Database persistence (encrypted secrets)
- Redis rate limiting (5 attempts/min)
- Full 2FA flow tests

**E2E Tests (Newman/Postman):**
- Setup flow: `/auth/2fa/setup` → `/auth/2fa/verify`
- Login with 2FA
- Backup code usage
- Error scenarios (invalid code, rate limited)

**Security Tests:**
- Brute force resistance
- Timing attack resistance
- Session elevation verification

---

## Implementation Timeline

### Week 1 (Days 1-7)

| Day | Task | Owner |
|-----|------|-------|
| 1-2 | Domain layer: Value objects (TOTPSecret, BackupCode, DeviceFingerprint) | senior-go-architect |
| 2-3 | Domain layer: User aggregate modifications | senior-go-architect |
| 3-4 | Infrastructure: SecretEncryptor (AES-256-GCM) | senior-secops-engineer |
| 4-5 | Infrastructure: TOTPService (pquerna/otp wrapper) | senior-go-architect |
| 5-6 | Database migration 00006 | cicd-guardian |
| 6-7 | Unit tests (90%+ coverage) | backend-test-architect |

### Week 2 (Days 8-14)

| Day | Task | Owner |
|-----|------|-------|
| 8-9 | Application layer: Setup2FA, Verify2FA commands | senior-go-architect |
| 9-10 | Application layer: Disable2FA, Get2FAStatus | senior-go-architect |
| 10-11 | HTTP handlers: /auth/2fa/* endpoints | senior-go-architect |
| 11-12 | Modify LoginHandler for partial sessions | senior-go-architect |
| 12-13 | E2E tests (Newman/Postman) | test-strategist |
| 13-14 | Security Gate S11 review | senior-secops-engineer |

---

## Agent Assignments

| Agent | Responsibilities |
|-------|------------------|
| **senior-go-architect** | Architecture, library selection, code implementation |
| **senior-secops-engineer** | Security requirements, encryption, audit logging |
| **backend-test-architect** | Test strategy, coverage targets |
| **cicd-guardian** | Migration, CI/CD integration |
| **test-strategist** | E2E tests, Postman collection |
| **scrum-master** | Sprint coordination, progress tracking |

---

## Key Architecture Decisions

### 1. TOTP Secret Encryption
- **Algorithm**: AES-256-GCM
- **Key**: Master key from environment (`TOTP_MASTER_KEY`)
- **Format**: nonce (12 bytes) + ciphertext + tag (16 bytes)

### 2. Backup Code Security
- **Hashing**: Argon2id (same as passwords)
- **Format**: 8-character base32 (e.g., `ABCD1234`)
- **Count**: 10 codes per user

### 3. Session Handling
- **Partial session**: JWT with `type: "partial"`, 5-min TTL
- **Full session**: JWT with `2fa_verified: timestamp`
- **Elevation required**: Partial sessions cannot access protected resources

### 4. Recovery Flow
1. **Backup codes** (primary) - Use instead of TOTP
2. **Email recovery** (fallback) - Time-limited link
3. **Admin reset** (last resort) - Requires audit trail

---

## Dependencies

```go
require (
    github.com/pquerna/otp v1.4.0  // TOTP generation/validation
)
```

---

## Success Criteria

- [ ] All 10 Security Gate S11 controls pass
- [ ] Test coverage 90%+
- [ ] E2E tests passing (setup, login, backup, disable flows)
- [ ] No critical/high vulnerabilities
- [ ] Performance: 2FA verify <150ms p95
- [ ] Documentation complete

---

## References

- [RFC 6238 - TOTP](https://datatracker.ietf.org/doc/html/rfc6238)
- [OWASP MFA Guidelines](https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html)
- [pquerna/otp Library](https://github.com/pquerna/otp)
