# Sprint 11: Two-Factor Authentication Security Specification

> **Document Type**: Security Requirements & Threat Analysis
> **Sprint**: Sprint 11 - Two-Factor Authentication (TOTP)
> **Status**: DRAFT - Awaiting Implementation
> **Owner**: Senior Security Operations Engineer
> **Last Updated**: 2026-01-06

---

## Executive Summary

This document defines the comprehensive security requirements for implementing TOTP-based two-factor authentication (2FA) in the goimg-datalayer backend. The implementation must meet OWASP MFA guidelines, prevent common 2FA attack vectors, and integrate seamlessly with existing JWT authentication and session management.

**Scope**: TOTP setup, verification, backup codes, device management, account recovery

**Risk Level**: **CRITICAL** - Authentication is the primary security boundary

**Compliance**: OWASP ASVS 4.0 (V2.8 Multi-Factor Authentication), RFC 6238 (TOTP)

---

## Table of Contents

1. [Security Gate S11 Controls (10 Mandatory)](#security-gate-s11-controls)
2. [Threat Model & Attack Vectors](#threat-model--attack-vectors)
3. [Best Practices & Operational Security](#best-practices--operational-security)
4. [OWASP Compliance Requirements](#owasp-compliance-requirements)
5. [Implementation Checklist](#implementation-checklist)
6. [Test Requirements](#test-requirements)

---

## Security Gate S11 Controls

All 10 controls must PASS before Sprint 11 can be marked complete.

### Mandatory Controls

| Control ID | Description | Pass Criteria | Verification Method |
|------------|-------------|---------------|---------------------|
| **S11-2FA-001** | TOTP secrets encrypted at rest | AES-256-GCM with per-user key derivation | Code review + DB inspection |
| **S11-2FA-002** | Backup codes hashed with Argon2id | Same parameters as password hashing (t=2, m=65536, p=4) | Code review + hash comparison test |
| **S11-2FA-003** | Rate limiting on 2FA verification | 5 attempts/min per user, 20 attempts/min per IP | Integration test demonstrating blocking |
| **S11-2FA-004** | Session elevation after 2FA completion | JWT claims include `2fa_verified` timestamp | Token inspection test |
| **S11-2FA-005** | Audit logging for all 2FA events | Setup, verify, disable, backup code use logged | Audit log verification |
| **S11-2FA-006** | Time-based code validation window | 30-second window ± 1 step (RFC 6238 compliant) | Unit test with time skew |
| **S11-2FA-007** | TOTP secret minimum entropy | 160 bits (20 bytes) cryptographically random | Entropy test + source verification |
| **S11-2FA-008** | Backup code generation security | 8 codes, 12 chars each, crypto/rand source | Code review + collision test |
| **S11-2FA-009** | Password required for 2FA disable | Re-authentication within 5 minutes | Integration test |
| **S11-2FA-010** | TOTP replay attack prevention | Code can only be used once per time window | Replay attack test |

---

## Threat Model & Attack Vectors

### T1: TOTP Secret Theft

**Threat**: Attacker obtains TOTP secret from database or QR code screenshot

**Attack Scenarios**:
- Database breach exposing encryption keys
- QR code intercepted during setup
- Man-in-the-middle attack during setup
- Social engineering user into sharing QR code

**Mitigations**:
- **S11-2FA-001**: Encrypt TOTP secrets at rest with AES-256-GCM
- **S11-2FA-007**: Use cryptographically random secrets (crypto/rand)
- Use per-user encryption keys derived from master key + user ID
- TLS required for all 2FA setup endpoints
- Display secret only once during setup, never retrievable
- **S11-2FA-005**: Audit log secret generation and viewing

**Residual Risk**: LOW (with encryption and TLS)

---

### T2: Replay Attacks on TOTP Codes

**Threat**: Attacker captures valid TOTP code and reuses it before expiration

**Attack Scenarios**:
- Network interception of TOTP code submission
- Shoulder surfing or keylogging
- Session hijacking after partial authentication

**Mitigations**:
- **S11-2FA-010**: Implement single-use code verification (track used codes in Redis)
- **S11-2FA-006**: 30-second time window reduces attack window
- **S11-2FA-004**: Session elevation prevents reuse across sessions
- TLS prevents network interception
- Store used codes in Redis with TTL = time_step * 2 (60 seconds)

**Implementation**:
```go
// Redis key pattern
goimg:totp:used:{user_id}:{code}  TTL: 60 seconds

// Verification flow
1. Validate TOTP code against secret (time-based)
2. Check Redis: EXISTS goimg:totp:used:{user_id}:{code}
   - If exists: return ErrCodeAlreadyUsed
3. If valid and not used: SET goimg:totp:used:{user_id}:{code} 1 EX 60
4. Mark session as 2FA verified
```

**Residual Risk**: LOW (30-60 second exposure window)

---

### T3: Brute Force on Backup Codes

**Threat**: Attacker attempts to guess backup codes to bypass TOTP

**Attack Scenarios**:
- Automated brute force on backup code endpoint
- Credential stuffing with leaked backup codes
- Insufficient entropy in backup code generation

**Mitigations**:
- **S11-2FA-002**: Hash backup codes with Argon2id (same as passwords)
- **S11-2FA-008**: 12 characters per code (62^12 = ~3.2e21 combinations)
- **S11-2FA-003**: Rate limiting (5 attempts/min per user)
- Account lockout after 10 failed backup code attempts
- Each backup code can only be used once (mark as used in database)
- **S11-2FA-005**: Audit log all backup code attempts

**Code Characteristics**:
- Length: 12 characters
- Charset: Alphanumeric (a-zA-Z0-9) = 62 characters
- Entropy: log2(62^12) = ~71.4 bits
- Source: crypto/rand
- Format: XXXX-XXXX-XXXX (grouped for readability)

**Residual Risk**: VERY LOW (with rate limiting and hashing)

---

### T4: Social Engineering for 2FA Bypass

**Threat**: Attacker tricks support/admin into disabling 2FA or uses account recovery to bypass

**Attack Scenarios**:
- Phishing attack claiming "I lost my 2FA device"
- Social engineering support to disable 2FA
- Account takeover via password reset bypassing 2FA
- Insider threat (admin disabling user's 2FA)

**Mitigations**:
- **S11-2FA-009**: Password re-authentication required to disable 2FA
- Backup codes provide self-service recovery (no admin bypass needed)
- **S11-2FA-005**: Audit log with IP/user-agent for 2FA disable events
- Email notification when 2FA is disabled
- Admin actions to disable 2FA require elevated authentication
- No "bypass 2FA" feature for admins (intentional design decision)
- Account recovery email includes warning about 2FA implications

**Password Reset Handling**:
- Password reset does NOT disable 2FA
- User must still have TOTP device or backup codes after password reset
- If user loses both password AND 2FA device, manual verification required (out of band)

**Residual Risk**: MEDIUM (social engineering always possible, mitigated by logging)

---

### T5: Time Synchronization Attacks

**Threat**: Attacker exploits time skew between server and user device

**Attack Scenarios**:
- Server clock drift causes valid codes to be rejected
- User device clock is significantly wrong
- Timezone confusion during DST changes

**Mitigations**:
- **S11-2FA-006**: Accept codes from ±1 time step (30 seconds before/after)
- Use NTP on server for accurate time synchronization
- Document timezone-independent implementation (UTC only)
- Provide clear error messages for time-based failures
- Monitor server time drift with health checks

**Implementation**:
```go
// Accept codes from:
// - Current time step
// - 1 step in the past (grace period)
// - 1 step in the future (clock skew tolerance)
// Total acceptance window: 90 seconds

func VerifyTOTP(secret string, code string, timestamp time.Time) bool {
    for i := -1; i <= 1; i++ {
        t := timestamp.Add(time.Duration(i) * 30 * time.Second)
        if generateCode(secret, t) == code {
            return true
        }
    }
    return false
}
```

**Residual Risk**: LOW (acceptable time window)

---

### T6: QR Code Phishing

**Threat**: Attacker tricks user into scanning malicious QR code

**Attack Scenarios**:
- Phishing email with fake "enable 2FA" link showing attacker's TOTP secret
- Malicious website displaying QR code for attacker-controlled account
- Social engineering to replace legitimate QR code

**Mitigations**:
- Display account email/username alongside QR code
- Include issuer name in TOTP URI: `otpauth://totp/goimg:{email}?secret=...&issuer=goimg`
- Require active session to view QR code (authenticated endpoint only)
- Show QR code only once, never re-displayable
- Email notification when 2FA setup is initiated
- Clear UI indicating this is official goimg 2FA setup

**TOTP URI Format** (RFC 6238):
```
otpauth://totp/goimg:{user_email}?secret={base32_secret}&issuer=goimg&algorithm=SHA1&digits=6&period=30
```

**Residual Risk**: LOW (user awareness required)

---

### T7: Account Lockout DoS

**Threat**: Attacker intentionally triggers account lockout via failed 2FA attempts

**Attack Scenarios**:
- Attacker knows victim's username/email
- Submits invalid 2FA codes to trigger lockout
- Victim cannot access their account

**Mitigations**:
- **S11-2FA-003**: Rate limiting per IP (20 attempts/min) in addition to per-user
- Account lockout threshold: 10 failed attempts (not 5, to reduce false positives)
- Lockout duration: 15 minutes (auto-unlock, not permanent)
- Email notification on lockout
- Backup codes remain valid during lockout (alternate recovery path)
- Audit logging of lockout events for abuse detection

**Lockout Policy**:
- Threshold: 10 consecutive failed 2FA attempts
- Duration: 15 minutes
- Scope: User account (not IP-based)
- Recovery: Auto-unlock after 15 min OR backup code use
- Notification: Email sent on lockout

**Residual Risk**: MEDIUM (availability impact, but time-limited)

---

### T8: Insecure Storage of Backup Codes

**Threat**: User stores backup codes insecurely (plaintext file, screenshot, notes app)

**Attack Scenarios**:
- User saves backup codes in cloud notes app
- Screenshot of backup codes leaked in device backup
- Plaintext file on compromised device

**Mitigations**:
- Clear warnings during backup code generation
- Recommend secure storage (password manager, encrypted file)
- Single-use backup codes (can't be reused after compromise)
- **S11-2FA-002**: Hash codes server-side (leaked DB doesn't expose codes)
- Ability to regenerate backup codes (invalidates old ones)

**UI Recommendations**:
- Display warning: "Store these codes securely. Do not screenshot or save in plaintext."
- Suggest: "Save in password manager or write on paper in secure location"
- Checkbox: "I have saved these codes securely" before proceeding
- Regeneration option: "Generate new codes" (invalidates all previous codes)

**Residual Risk**: HIGH (user behavior outside our control, mitigated by hashing)

---

### T9: Session Fixation with Partial 2FA

**Threat**: Attacker hijacks session between password login and 2FA completion

**Attack Scenarios**:
- User logs in with password, receives session token
- Attacker steals session token before 2FA submission
- Attacker uses token to submit their own 2FA code

**Mitigations**:
- **S11-2FA-004**: Partial session tokens lack full privileges
- Separate session state: `authenticated` vs `2fa_verified`
- Endpoints requiring 2FA check `2fa_verified` claim in JWT
- Session regeneration after successful 2FA verification
- Short TTL (5 minutes) for partial authentication sessions

**Session Flow**:
```
1. POST /auth/login (password) → Access Token (2fa_required: true)
   Claims: {uid: "123", role: "user", 2fa_required: true}

2. POST /auth/2fa/verify (TOTP code) → New Access Token (2fa_verified: timestamp)
   Claims: {uid: "123", role: "user", 2fa_verified: 1641234567}

3. Protected endpoints check 2fa_verified claim
```

**JWT Claims**:
- `2fa_required` (bool): User has 2FA enabled but not yet verified this session
- `2fa_verified` (timestamp): Unix timestamp when 2FA was completed this session
- Middleware checks: `2fa_verified` is present and < 24 hours old

**Residual Risk**: LOW (partial sessions have limited access)

---

### T10: Device/Location Tracking Bypass

**Threat**: Attacker uses stolen credentials + 2FA from new device without detection

**Attack Scenarios**:
- Attacker has password + TOTP device access
- Logs in from unusual location/device
- No notification sent to user

**Mitigations**:
- Device fingerprinting (user-agent, IP, headers)
- Email notification for 2FA login from new device
- User device management UI (view/revoke trusted devices)
- Unusual login detection (new country, new ASN, new device)
- **S11-2FA-005**: Audit log includes IP, user-agent, geolocation metadata

**Detection Heuristics**:
- First login from IP subnet
- First login from country
- First login from user-agent family
- Login outside normal time patterns

**User Notifications**:
- Email: "New device login detected: {device}, {location}, {time}"
- In-app: List of recent 2FA logins with device/location
- Action: "Revoke device" button

**Residual Risk**: MEDIUM (notification-based, not preventative)

---

## Best Practices & Operational Security

### 1. Two-Factor Authentication During Password Reset

**Scenario**: User forgets password but has 2FA enabled

**Recommendation**: **Password reset does NOT disable 2FA**

**Flow**:
1. User requests password reset → Email sent with reset token
2. User clicks link, sets new password
3. User logs in with new password → 2FA prompt appears
4. User must still provide TOTP code or backup code

**Rationale**: Disabling 2FA during password reset would create a bypass vulnerability. If an attacker compromises email, they should not be able to bypass 2FA.

**Edge Case**: User forgets password AND loses 2FA device
- **Solution**: Manual verification required (support ticket)
- Require: Photo ID, account ownership proof, video verification
- Process documented in security runbook
- High-friction by design (prevents social engineering)

---

### 2. Account Recovery Scenarios

#### Scenario A: User Loses TOTP Device (Has Backup Codes)
**Flow**:
1. User logs in with password → 2FA prompt
2. User clicks "Use backup code instead"
3. User enters one backup code → Access granted
4. User can disable 2FA or setup new TOTP device

**Recommendation**: Require password re-authentication before allowing 2FA setup change

---

#### Scenario B: User Loses TOTP Device AND Backup Codes
**Flow**:
1. User contacts support with "lost 2FA access"
2. Support initiates verification process (NOT immediate bypass)
3. Verification requirements:
   - Government-issued photo ID
   - Ownership proof (recent uploads, purchase history, etc.)
   - Video call for identity verification
   - Email confirmation from registered address
4. Support manually disables 2FA with approval from security team
5. **Audit log** records manual 2FA disable with justification
6. User notified via email that 2FA was disabled

**SLA**: 48-72 hours (high-friction by design)

**Recommendation**: Document this process in `/docs/security/incident_response.md`

---

#### Scenario C: Account Compromise Detected (User Reports)
**Flow**:
1. User reports: "I didn't enable 2FA but it's now active"
2. Immediate account suspension (prevent attacker access)
3. Investigation: Check audit logs for 2FA setup event
   - IP address
   - User-agent
   - Timestamp
   - Session details
4. If suspicious: Reset password, disable 2FA, require re-verification
5. If legitimate (user forgot): Standard recovery process

**Recommendation**: Implement "report suspicious activity" button in user dashboard

---

### 3. Admin Bypass Requirements

**Policy**: **NO admin bypass for 2FA is permitted**

**Rationale**: Admin accounts are high-value targets. Allowing admins to bypass their own 2FA creates a critical vulnerability.

**Exceptions**:
- Emergency access procedure:
  - Requires 2-of-3 admin approval
  - Time-limited (24 hours)
  - Full audit trail
  - Automatic security review triggered

**Admin 2FA Enforcement**:
- All admin accounts MUST have 2FA enabled (no opt-out)
- Admin actions (user ban, role change, 2FA disable for others) require fresh 2FA verification (<5 minutes)
- Separate admin session elevation

**Recommendation**: Implement `RequireRecentTwoFactor(maxAge time.Duration)` middleware for sensitive admin actions

---

### 4. Logging and Alerting

#### Events to Log (Audit Trail)

| Event | Log Level | Fields | Alerting |
|-------|-----------|--------|----------|
| 2FA setup initiated | INFO | user_id, ip, user_agent | Email notification |
| 2FA setup completed | INFO | user_id, ip, user_agent, device | Email notification |
| 2FA verification success | INFO | user_id, ip, user_agent | None (normal) |
| 2FA verification failure | WARN | user_id, ip, user_agent, code_prefix | Alert after 5 failures |
| Backup code used | INFO | user_id, ip, code_id (not value) | Email notification |
| Backup codes regenerated | WARN | user_id, ip, user_agent | Email notification |
| 2FA disabled | WARN | user_id, ip, user_agent, reason | Email notification |
| Account lockout (failed 2FA) | ERROR | user_id, ip, failure_count | Email + Slack alert |
| Manual 2FA disable (support) | CRITICAL | user_id, admin_id, ticket_id, justification | Email + Security review |

#### Metrics to Track (Prometheus)

```go
// Counter: Total 2FA verification attempts
goimg_2fa_verifications_total{result="success|failure", method="totp|backup_code"}

// Counter: 2FA setup events
goimg_2fa_setups_total{result="completed|aborted"}

// Counter: Account lockouts due to 2FA failures
goimg_2fa_lockouts_total

// Histogram: 2FA verification latency
goimg_2fa_verification_duration_seconds

// Gauge: Users with 2FA enabled
goimg_2fa_enabled_users
```

#### Alerting Rules (Grafana/Prometheus)

```yaml
# High 2FA failure rate for single user
- alert: HighTwoFactorFailureRate
  expr: rate(goimg_2fa_verifications_total{result="failure"}[5m]) > 0.5
  annotations:
    summary: "High 2FA failure rate detected for user {{ $labels.user_id }}"

# Unusual spike in 2FA lockouts
- alert: TwoFactorLockoutSpike
  expr: increase(goimg_2fa_lockouts_total[1h]) > 10
  annotations:
    summary: "Unusual spike in 2FA lockouts (possible attack)"

# Manual 2FA disable events (always alert)
- alert: ManualTwoFactorDisable
  expr: increase(goimg_2fa_manual_disable_total[5m]) > 0
  annotations:
    severity: critical
    summary: "Manual 2FA disable event detected"
```

---

### 5. User Experience Best Practices

#### 2FA Setup Flow
1. User navigates to "Security Settings"
2. Clicks "Enable Two-Factor Authentication"
3. Re-authenticate with password (within 5 minutes)
4. Display QR code with TOTP secret
5. User scans QR code with authenticator app
6. User enters first TOTP code to verify setup
7. Display backup codes (8 codes, one-time display)
8. User confirms they've saved backup codes
9. 2FA enabled → Email notification sent

**UI Considerations**:
- Clear instructions: "Download Google Authenticator, Authy, or 1Password"
- Manual entry option: "Can't scan QR code? Enter this code manually: ABCD-EFGH-..."
- Issuer name displayed: "goimg (user@example.com)"
- Warning: "Don't share this QR code or secret with anyone"

---

#### 2FA Login Flow
1. User enters email + password → Success
2. System checks: Does user have 2FA enabled?
   - If NO: Issue full access token, login complete
   - If YES: Issue partial access token, redirect to 2FA prompt
3. User sees 2FA code input field
4. User enters 6-digit code from authenticator app
5. System validates code (with replay prevention)
6. Issue new full access token with `2fa_verified` claim
7. Login complete

**UI Considerations**:
- Show "Use backup code instead" link
- Show "Trusted this device" checkbox (optional, extends session)
- Error messages: Generic "Invalid code" (no hints)
- Rate limiting feedback: "Too many attempts. Try again in X minutes."

---

#### Backup Code Usage
1. User clicks "Use backup code instead"
2. Input field changes to 12-character format
3. User enters one backup code
4. System validates code (hashed comparison)
5. Mark code as used in database (single-use)
6. Issue full access token
7. Show warning: "You have X backup codes remaining. Generate new ones?"

**UI Considerations**:
- Format: `XXXX-XXXX-XXXX` (auto-format with hyphens)
- Show remaining backup code count after use
- Prominent "Generate new backup codes" button when count < 3

---

## OWASP Compliance Requirements

### OWASP ASVS 4.0 - Multi-Factor Authentication (V2.8)

| Requirement | ASVS ID | Compliance Status | Implementation |
|-------------|---------|-------------------|----------------|
| Verify that MFA is required for high-value transactions and sensitive operations | 2.8.1 | ✅ COMPLIANT | Admin actions require recent 2FA |
| Verify that MFA uses time-based OTP (TOTP) or similar cryptographic authentication | 2.8.2 | ✅ COMPLIANT | RFC 6238 TOTP implementation |
| Verify that MFA enrollment requires verification before activation | 2.8.3 | ✅ COMPLIANT | User must enter TOTP code during setup |
| Verify that MFA secrets are generated using a CSPRNG | 2.8.4 | ✅ COMPLIANT | crypto/rand for TOTP secrets |
| Verify that MFA secrets are at least 128 bits | 2.8.5 | ✅ COMPLIANT | 160-bit secrets (S11-2FA-007) |
| Verify that MFA backup codes are hashed | 2.8.6 | ✅ COMPLIANT | Argon2id hashing (S11-2FA-002) |
| Verify that MFA is enforced for administrative actions | 2.8.7 | ✅ COMPLIANT | Admin middleware requires 2FA |

### OWASP Top 10 2021 Mapping

| OWASP Risk | Relevance to 2FA | Mitigation |
|------------|------------------|------------|
| A01 - Broken Access Control | 2FA adds second factor to authentication | S11-2FA-004 (Session elevation) |
| A02 - Cryptographic Failures | TOTP secret storage, backup code hashing | S11-2FA-001, S11-2FA-002 |
| A04 - Insecure Design | Account recovery bypass, social engineering | Password required to disable 2FA |
| A07 - Identification and Authentication Failures | Primary risk area for 2FA | All 10 controls address this |

### RFC 6238 (TOTP) Compliance

| RFC 6238 Requirement | Implementation |
|----------------------|----------------|
| Time-based one-time password algorithm | ✅ Using RFC 6238 TOTP algorithm |
| HMAC-SHA1 for code generation | ✅ SHA1 (standard for compatibility) |
| 30-second time step | ✅ Configurable, default 30s |
| 6-digit codes | ✅ Standard 6-digit format |
| Time synchronization tolerance | ✅ ±1 step (90-second window) |

**Note**: SHA1 is used for TOTP only (not for password hashing). This is standard practice per RFC 6238 and maximizes compatibility with authenticator apps.

---

## Implementation Checklist

### Database Schema

```sql
-- User TOTP secrets (encrypted)
CREATE TABLE user_totp_secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    secret_encrypted BYTEA NOT NULL,          -- AES-256-GCM encrypted
    encryption_nonce BYTEA NOT NULL,          -- GCM nonce (12 bytes)
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Backup codes (hashed with Argon2id)
CREATE TABLE user_backup_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(255) NOT NULL,          -- Argon2id hash
    used BOOLEAN NOT NULL DEFAULT FALSE,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    INDEX idx_user_backup_codes (user_id, used)
);

-- Device tracking for unusual login detection
CREATE TABLE user_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_fingerprint VARCHAR(255) NOT NULL, -- Hash of user-agent + IP subnet
    ip_address INET NOT NULL,
    user_agent TEXT,
    trusted BOOLEAN NOT NULL DEFAULT FALSE,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, device_fingerprint)
);

-- 2FA audit events
CREATE TABLE totp_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    event_type VARCHAR(50) NOT NULL,          -- setup, verify_success, verify_failure, disable, etc.
    ip_address INET,
    user_agent TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    INDEX idx_totp_audit_user (user_id, created_at DESC),
    INDEX idx_totp_audit_event (event_type, created_at DESC)
);
```

### API Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/auth/2fa/setup` | Initiate 2FA setup, return QR code | Yes + recent password |
| POST | `/auth/2fa/verify-setup` | Verify TOTP code to enable 2FA | Yes |
| POST | `/auth/2fa/verify` | Verify TOTP code during login | Partial auth token |
| DELETE | `/auth/2fa` | Disable 2FA | Yes + password |
| GET | `/auth/2fa/backup-codes` | Retrieve backup codes (one-time) | Yes + recent password |
| POST | `/auth/2fa/backup-codes/regenerate` | Generate new backup codes | Yes + password |
| POST | `/auth/2fa/backup-code/verify` | Verify backup code during login | Partial auth token |
| GET | `/auth/2fa/devices` | List trusted devices | Yes |
| DELETE | `/auth/2fa/devices/{id}` | Revoke device | Yes |

### Configuration

```yaml
# config/security.yaml
totp:
  issuer: "goimg"
  algorithm: "SHA1"           # RFC 6238 standard
  digits: 6
  period: 30                  # seconds
  secret_length: 20           # bytes (160 bits)

backup_codes:
  count: 8
  length: 12                  # characters
  charset: "alphanumeric"     # a-zA-Z0-9

rate_limiting:
  totp_verification:
    per_user: 5               # attempts per minute
    per_ip: 20                # attempts per minute
  backup_code:
    per_user: 3               # attempts per minute

lockout:
  threshold: 10               # failed attempts
  duration: 900               # seconds (15 minutes)

session:
  partial_auth_ttl: 300       # seconds (5 minutes)
  2fa_verification_max_age: 86400  # seconds (24 hours)
```

### Go Packages Required

```go
require (
    github.com/pquerna/otp v1.4.0              // TOTP generation/validation
    github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e  // QR code generation
)
```

---

## Test Requirements

### Unit Tests (Required)

```go
// TOTP secret generation
- TestGenerateTOTPSecret_Entropy              // Verify 160-bit entropy
- TestGenerateTOTPSecret_Uniqueness           // No collisions in 10k generations
- TestEncryptTOTPSecret_AES256GCM             // Verify encryption algorithm
- TestDecryptTOTPSecret_Correctness           // Encrypt/decrypt round-trip

// TOTP verification
- TestVerifyTOTP_ValidCode                    // Current time step
- TestVerifyTOTP_PastTimeStep                 // -1 step (grace period)
- TestVerifyTOTP_FutureTimeStep               // +1 step (clock skew)
- TestVerifyTOTP_ExpiredCode                  // -2 steps (expired)
- TestVerifyTOTP_InvalidCode                  // Wrong code
- TestVerifyTOTP_ReplayPrevention             // Same code twice

// Backup code generation
- TestGenerateBackupCodes_Count               // Verify 8 codes generated
- TestGenerateBackupCodes_Length              // Verify 12 characters each
- TestGenerateBackupCodes_Entropy             // Verify cryptographic randomness
- TestGenerateBackupCodes_Uniqueness          // No duplicates
- TestHashBackupCode_Argon2id                 // Verify Argon2id hashing
- TestVerifyBackupCode_Valid                  // Correct code
- TestVerifyBackupCode_Invalid                // Wrong code
- TestVerifyBackupCode_AlreadyUsed            // Single-use enforcement

// Rate limiting
- TestRateLimit_TOTPVerification              // 5 attempts/min per user
- TestRateLimit_BackupCode                    // 3 attempts/min per user
- TestRateLimit_IPBased                       // 20 attempts/min per IP

// Session elevation
- TestSessionElevation_2FARequired            // Partial token issued
- TestSessionElevation_2FAVerified            // Full token after TOTP
- TestSessionElevation_ClaimPresent           // 2fa_verified in JWT
```

### Integration Tests (Required)

```go
// Full 2FA setup flow
- TestTwoFactorSetup_E2E                      // Setup → QR → Verify → Enabled
- TestTwoFactorSetup_RequiresPassword         // Password auth check
- TestTwoFactorSetup_EmailNotification        // Email sent on enable

// Full 2FA login flow
- TestTwoFactorLogin_Success                  // Password → TOTP → Access
- TestTwoFactorLogin_InvalidTOTP              // Rejection on wrong code
- TestTwoFactorLogin_BackupCode               // Alternate login method
- TestTwoFactorLogin_AccountLockout           // 10 failures → locked

// Backup code flow
- TestBackupCode_Generation                   // Generate 8 codes
- TestBackupCode_SingleUse                    // Mark as used
- TestBackupCode_Regeneration                 // Invalidate old codes

// Disable 2FA
- TestDisableTwoFactor_RequiresPassword       // Password re-auth
- TestDisableTwoFactor_EmailNotification      // Email sent
- TestDisableTwoFactor_AuditLog               // Event logged

// Unusual login detection
- TestUnusualLogin_NewDevice                  // Email notification
- TestUnusualLogin_NewCountry                 // Detection heuristic
```

### Security Tests (Required)

```go
// Attack simulations
- TestAttack_TOTPReplay                       // Replay attack blocked
- TestAttack_BackupCodeBruteForce             // Rate limiting enforced
- TestAttack_TimingAttack                     // Constant-time comparison
- TestAttack_SQLInjection_TOTPEndpoints       // Parameterized queries
- TestAttack_SessionFixation                  // Session regeneration

// Cryptography validation
- TestCrypto_TOTPSecretEntropy                // Statistical entropy test
- TestCrypto_BackupCodeEntropy                // Chi-square test
- TestCrypto_EncryptionKeyDerivation          // Per-user keys
- TestCrypto_NonceUniqueness                  // No nonce reuse

// Audit logging
- TestAudit_AllEventsLogged                   // All 2FA events captured
- TestAudit_NoSensitiveData                   // Secrets not logged
- TestAudit_ImmutableLogs                     // Insert-only audit table
```

### Performance Tests (Required)

```go
// Latency benchmarks
- BenchmarkTOTPVerification                   // <50ms target
- BenchmarkBackupCodeVerification             // <200ms (Argon2id)
- BenchmarkQRCodeGeneration                   // <100ms
- BenchmarkEncryption_TOTPSecret              // <10ms

// Load tests
- TestLoad_ConcurrentTOTPVerifications        // 100 req/s
- TestLoad_DatabaseLockout                    // Concurrent updates
```

---

## Acceptance Criteria

Sprint 11 is complete when:

1. ✅ All 10 security gate controls (S11-2FA-001 to S11-2FA-010) PASS
2. ✅ All unit tests pass with >85% coverage
3. ✅ All integration tests pass
4. ✅ All security tests pass (attack simulations)
5. ✅ OpenAPI spec updated with 2FA endpoints
6. ✅ E2E tests (Newman/Postman) for 2FA flows
7. ✅ Documentation complete:
   - User guide: "How to enable 2FA"
   - Admin guide: "2FA support procedures"
   - API documentation
8. ✅ Security review by senior-secops-engineer
9. ✅ Penetration testing: 2FA bypass attempts
10. ✅ Audit logging verified in production-like environment

---

## References

- **RFC 6238**: TOTP: Time-Based One-Time Password Algorithm
  https://datatracker.ietf.org/doc/html/rfc6238

- **OWASP ASVS 4.0**: Application Security Verification Standard (V2.8 MFA)
  https://owasp.org/www-project-application-security-verification-standard/

- **NIST SP 800-63B**: Digital Identity Guidelines (Authenticator Assurance)
  https://pages.nist.gov/800-63-3/sp800-63b.html

- **Google Authenticator PAM**: Reference implementation
  https://github.com/google/google-authenticator-libpam

- **OWASP MFA Cheat Sheet**:
  https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html

---

## Appendix A: TOTP Secret Encryption Implementation

```go
package security

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"
    "fmt"
    "io"

    "golang.org/x/crypto/pbkdf2"
)

// EncryptTOTPSecret encrypts a TOTP secret using AES-256-GCM
// Uses per-user key derivation from master key
func EncryptTOTPSecret(secret []byte, userID string, masterKey []byte) (encrypted, nonce []byte, err error) {
    // Derive per-user encryption key using PBKDF2
    salt := sha256.Sum256([]byte(userID))
    key := pbkdf2.Key(masterKey, salt[:], 100000, 32, sha256.New)

    // Create AES cipher
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
    }

    // Create GCM mode
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
    }

    // Generate random nonce (12 bytes for GCM)
    nonce = make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
    }

    // Encrypt secret
    encrypted = gcm.Seal(nil, nonce, secret, nil)

    return encrypted, nonce, nil
}

// DecryptTOTPSecret decrypts a TOTP secret
func DecryptTOTPSecret(encrypted, nonce []byte, userID string, masterKey []byte) ([]byte, error) {
    // Derive same per-user key
    salt := sha256.Sum256([]byte(userID))
    key := pbkdf2.Key(masterKey, salt[:], 100000, 32, sha256.New)

    // Create AES cipher
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher: %w", err)
    }

    // Create GCM mode
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }

    // Decrypt
    secret, err := gcm.Open(nil, nonce, encrypted, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt: %w", err)
    }

    return secret, nil
}
```

**Security Notes**:
- Master key must be stored in secrets manager (not in code)
- Per-user key derivation prevents cross-user attacks if one key leaks
- GCM provides both confidentiality and authenticity
- Nonce MUST be unique for each encryption operation
- PBKDF2 iteration count (100,000) balances security and performance

---

## Appendix B: TOTP Replay Prevention Implementation

```go
package security

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

type TOTPVerifier struct {
    redis  *redis.Client
    secret SecretStore
}

// VerifyTOTP verifies a TOTP code and prevents replay attacks
func (v *TOTPVerifier) VerifyTOTP(ctx context.Context, userID, code string) error {
    // 1. Get user's TOTP secret
    secret, err := v.secret.GetTOTPSecret(ctx, userID)
    if err != nil {
        return fmt.Errorf("failed to get secret: %w", err)
    }

    // 2. Validate TOTP code (with time skew tolerance)
    valid := v.validateTOTPWithSkew(secret, code, time.Now())
    if !valid {
        return ErrInvalidTOTPCode
    }

    // 3. Check for replay (code already used)
    replayKey := fmt.Sprintf("goimg:totp:used:%s:%s", userID, code)
    exists, err := v.redis.Exists(ctx, replayKey).Result()
    if err != nil {
        // Log error but don't fail (fail-open for Redis issues)
        logger.Error().Err(err).Msg("redis check failed, allowing verification")
    } else if exists > 0 {
        return ErrTOTPCodeAlreadyUsed
    }

    // 4. Mark code as used (60-second TTL = 2 time steps)
    err = v.redis.Set(ctx, replayKey, "1", 60*time.Second).Err()
    if err != nil {
        // Log error but don't fail (already validated)
        logger.Error().Err(err).Msg("failed to mark code as used")
    }

    return nil
}

// validateTOTPWithSkew checks TOTP code with ±1 time step tolerance
func (v *TOTPVerifier) validateTOTPWithSkew(secret, code string, timestamp time.Time) bool {
    for offset := -1; offset <= 1; offset++ {
        t := timestamp.Add(time.Duration(offset) * 30 * time.Second)
        expectedCode := generateTOTP(secret, t)

        // Constant-time comparison to prevent timing attacks
        if subtle.ConstantTimeCompare([]byte(code), []byte(expectedCode)) == 1 {
            return true
        }
    }
    return false
}
```

**Security Notes**:
- Redis failure fails open (allows verification) to prevent availability issues
- 60-second TTL covers current + past + future time steps
- Constant-time comparison prevents timing attacks on code validation
- Replay prevention is defense-in-depth (TOTP codes already time-limited)

---

## Document Control

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-06 | Senior SecOps Engineer | Initial security specification |

**Status**: DRAFT - Awaiting implementation kickoff

**Review Required**: senior-go-architect, backend-test-architect

**Next Steps**: Implementation sprint kickoff → Security gate tracking
