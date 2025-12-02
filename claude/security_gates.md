# Security Gates by Sprint

> Security checkpoints and mandatory reviews for each development sprint.
> **CRITICAL**: Security gates must pass before sprint completion. Senior SecOps Engineer reviews required.

---

## Gate Process

Each sprint has **mandatory** and **recommended** security controls. Gates are blocking - sprints cannot be marked complete until all mandatory controls pass.

### Gate Workflow

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Development │────▶│  Self-Review │────▶│  Automated   │────▶│   SecOps     │
│   Complete   │     │  (Checklist) │     │   Scanning   │     │   Review     │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
                                                │                       │
                                                ▼                       ▼
                                           ┌──────────────┐     ┌──────────────┐
                                           │   PASS/FAIL  │     │   PASS/FAIL  │
                                           └──────────────┘     └──────────────┘
```

### Failure Response

- **FAIL**: Sprint blocked until remediation complete
- **CONDITIONAL PASS**: Deploy with documented risk acceptance + remediation plan
- **PASS**: Sprint approved for merge/deployment

---

## Sprint 1-2: Foundation & Domain Layer

**Focus**: Project structure, domain model, OpenAPI specification
**Security Risk**: Low (no running code yet)

### Mandatory Controls

| Control ID | Description | Pass Criteria | Verification |
|------------|-------------|---------------|--------------|
| **S1-AUTH-001** | No hardcoded secrets in codebase | Zero instances found | `git grep -iE "(password|secret|api[_-]?key|token).*=.*['\"]"` |
| **S1-ARCH-001** | Dependencies reviewed for known CVEs | Zero critical/high vulnerabilities | `trivy fs --severity HIGH,CRITICAL .` |
| **S1-ARCH-002** | .gitignore excludes sensitive files | `.env`, `*.pem`, `*.key`, `credentials.json` present | Manual review |
| **S1-CODE-001** | golangci-lint security rules enabled | `gosec`, `G101`, `G401`, `G501` rules active | Check `.golangci.yml` |
| **S1-SPEC-001** | OpenAPI spec defines security schemes | JWT auth defined for protected endpoints | Validate `api/openapi/openapi.yaml` |

### Recommended Controls

- [ ] Enable GitHub secret scanning
- [ ] Configure Dependabot for vulnerability alerts
- [ ] Document security assumptions in architecture

### SecOps Review Focus

- Architecture review for security boundaries
- OpenAPI spec review for authentication/authorization model
- Dependency choices (libraries with poor security track record)

---

## Sprint 3: Infrastructure - Identity Context

**Focus**: PostgreSQL, Redis, JWT implementation, session management
**Security Risk**: HIGH (authentication foundation)

### Mandatory Controls

| Control ID | Description | Pass Criteria | Verification |
|------------|-------------|---------------|--------------|
| **S3-AUTH-001** | JWT algorithm is RS256 (asymmetric) | RS256 explicitly configured, no HS256 | Code review: `internal/infrastructure/security/jwt.go` |
| **S3-AUTH-002** | JWT private key is 4096-bit RSA minimum | Key size verified | `openssl rsa -in jwt_key.pem -text -noout \| grep "Private-Key"` |
| **S3-AUTH-003** | Refresh tokens stored hashed (SHA-256) | Raw tokens never stored | Code review: session repository |
| **S3-AUTH-004** | Token rotation detects replay attacks | Test case demonstrates detection | Unit test: `TestTokenReplayDetection` |
| **S3-AUTH-005** | Access token TTL = 15 minutes max | Configuration enforced | Check config validation |
| **S3-AUTH-006** | Refresh token TTL = 7 days max | Configuration enforced | Check config validation |
| **S3-CRYPTO-001** | Password hashing uses Argon2id | Argon2id with params: t=2, m=65536, p=4, keyLen=32 | Code review: password service |
| **S3-CRYPTO-002** | Password comparison is constant-time | `subtle.ConstantTimeCompare()` used | Code review |
| **S3-DB-001** | Database connections use TLS/SSL | `sslmode=require` or `sslmode=verify-full` | Config review + connection test |
| **S3-DB-002** | PostgreSQL password not hardcoded | Environment variable or secrets manager | Config review |
| **S3-DB-003** | Prepared statements used for all queries | No string concatenation in SQL | Code review: all repository files |
| **S3-REDIS-001** | Redis requires authentication | `requirepass` configured | Docker Compose + config review |
| **S3-REDIS-002** | Redis connections use TLS (prod) | TLS enabled for production config | Config review |

### Security Tests Required

```go
// Required test cases (must exist and pass)
- TestJWT_RejectsHS256Algorithm
- TestJWT_RejectsExpiredToken
- TestJWT_ValidatesAudience
- TestJWT_ValidatesIssuer
- TestRefreshToken_DetectsReplay
- TestRefreshToken_RotatesOnUse
- TestPassword_UsesArgon2id
- TestPassword_RejectsWeakPasswords (< 12 chars)
- TestSession_RegeneratesOnLogin
- TestUserRepository_SQLInjectionPrevention
```

### Recommended Controls

- [ ] Implement JWT key rotation mechanism
- [ ] Add brute force protection on login (rate limiting)
- [ ] Configure session timeout for inactive users
- [ ] Add concurrent session limits per user

### SecOps Review Focus

- JWT implementation security (algorithm, claims, validation)
- Token storage and lifecycle management
- Password hashing parameters and side-channel resistance
- Database connection security (TLS, credential management)
- Refresh token rotation and replay attack prevention

### Known Risks & Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| JWT key compromise | Critical | RS256 with key rotation, monitor for suspicious token patterns |
| Database credential leak | Critical | Use secrets manager (Vault/AWS Secrets Manager), rotate regularly |
| Timing attacks on password comparison | High | Use `subtle.ConstantTimeCompare()` |
| Session fixation | Medium | Regenerate session ID on login |

---

## Sprint 4: Application & HTTP - Identity Context

**Focus**: Auth endpoints, middleware, rate limiting, security headers
**Security Risk**: HIGH (public-facing authentication)

### Mandatory Controls

| Control ID | Description | Pass Criteria | Verification |
|------------|-------------|---------------|--------------|
| **S4-HTTP-001** | Security headers applied to all responses | All headers present on test request | Integration test + header check |
| **S4-HTTP-002** | CORS allows only approved origins | Wildcard (*) not used in production | Config review |
| **S4-RATE-001** | Login endpoint rate limited (5/min per IP) | Test demonstrates blocking | Integration test: `TestRateLimit_LoginEndpoint` |
| **S4-RATE-002** | Global rate limiting enabled (100/min per IP) | Test demonstrates blocking | Integration test |
| **S4-AUTH-007** | Account enumeration prevented | Identical responses for valid/invalid email | Test: `TestLogin_PreventAccountEnumeration` |
| **S4-AUTH-008** | Account lockout after 5 failed attempts | Test demonstrates lockout | Test: `TestLogin_AccountLockout` |
| **S4-VAL-001** | Request size limits enforced | Max 10MB request body | Middleware test |
| **S4-VAL-002** | Email validation prevents injection | RFC 5322 compliance, length limits | Unit test |
| **S4-VAL-003** | Username validation blocks malicious input | Alphanumeric + limited symbols only | Unit test |
| **S4-LOG-001** | Passwords never logged | grep confirms no password in logs | Log audit: `grep -ri password logs/` |
| **S4-LOG-002** | Authentication events logged | Login success/failure logged with user ID | Log verification |
| **S4-ERR-001** | Error responses don't leak stack traces | RFC 7807 Problem Details format used | Integration test |
| **S4-ERR-002** | Database errors don't leak to client | Generic "internal error" returned | Error handler review |

### Required Security Headers

```http
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'; frame-ancestors 'none'
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: geolocation=(), microphone=(), camera=()
Strict-Transport-Security: max-age=31536000; includeSubDomains (production)
```

### Security Tests Required

```go
// Authentication bypass attempts
- TestAuth_RejectsEmptyToken
- TestAuth_RejectsMalformedToken
- TestAuth_RejectsTokenWithInvalidSignature
- TestAuth_RejectsExpiredAccessToken
- TestAuth_RejectsRevokedToken

// Account security
- TestRegister_RejectsWeakPassword
- TestRegister_RejectsSQLInjectionAttempt
- TestLogin_PreventAccountEnumeration
- TestLogin_AccountLockoutAfter5Failures
- TestLogin_RateLimitEnforced

// Authorization
- TestMiddleware_RequiresAuthentication
- TestMiddleware_ValidatesJWTClaims
- TestMiddleware_RejectsExpiredToken

// Input validation
- TestValidation_EmailFormat
- TestValidation_UsernameFormat
- TestValidation_PasswordComplexity
```

### Recommended Controls

- [ ] Add CAPTCHA after 3 failed login attempts
- [ ] Implement email verification for new accounts
- [ ] Add IP-based anomaly detection
- [ ] Monitor for credential stuffing patterns

### SecOps Review Focus

- Authentication flow security (registration, login, logout)
- Middleware implementation (authentication, authorization, logging)
- Rate limiting effectiveness under load
- Error handling security (no information disclosure)
- Account lockout and brute force protection
- Security header configuration
- Request validation and sanitization

### Known Risks & Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| Brute force attacks | High | Rate limiting + account lockout + monitoring |
| Account enumeration | Medium | Generic error messages, timing attack mitigation |
| Session hijacking | Medium | HTTPS only, secure cookies, session timeout |
| CSRF attacks | Medium | SameSite cookies, CSRF tokens (if using cookies) |

---

## Sprint 5: Domain & Infrastructure - Gallery Context

**Focus**: Image processing, storage providers, ClamAV integration
**Security Risk**: CRITICAL (file upload vulnerabilities)

### Mandatory Controls

| Control ID | Description | Pass Criteria | Verification |
|------------|-------------|---------------|--------------|
| **S5-UPLOAD-001** | File size limit enforced before processing | 10MB max, enforced at middleware | Test: upload 11MB file, expect 413 |
| **S5-UPLOAD-002** | MIME type validated by content, not extension | `http.DetectContentType()` used | Code review + test with fake extension |
| **S5-UPLOAD-003** | Image dimensions validated | Max 8192x8192 pixels | Test: upload 9000x9000 image |
| **S5-UPLOAD-004** | Pixel count limit enforced | Max 100M pixels | Test: upload 10000x10000 image |
| **S5-UPLOAD-005** | ClamAV scans all uploads | Malware detection test passes | Test: upload EICAR test file |
| **S5-UPLOAD-006** | ClamAV signatures up-to-date | Signatures < 24 hours old | `freshclam` in CI, health check |
| **S5-UPLOAD-007** | Images re-encoded through libvips | Prevents polyglot files | Test: upload polyglot JPEG/HTML |
| **S5-UPLOAD-008** | EXIF metadata stripped | No GPS, camera data in output | Test: upload image with EXIF, verify removal |
| **S5-UPLOAD-009** | Filename sanitized | No path separators, special chars | Test: upload "../../../etc/passwd.jpg" |
| **S5-UPLOAD-010** | Upload rate limiting (50/hour per user) | Test demonstrates blocking | Integration test |
| **S5-STORAGE-001** | S3 buckets block public access by default | Bucket policy review | AWS policy check |
| **S5-STORAGE-002** | Storage keys are non-guessable | UUIDs or cryptographic hashes | Code review |
| **S5-STORAGE-003** | Path traversal prevention on file operations | `filepath.Clean()` + base dir validation | Code review + path traversal test |
| **S5-PROC-001** | libvips prevents decompression bombs | Memory limits configured | Config review |

### Image Validation Pipeline

```
Upload → Size Check → MIME Sniff → Decode Validation → ClamAV Scan → Re-encode → EXIF Strip → Store
         (10MB)       (Content)     (libvips)          (Malware)      (libvips)   (Privacy)
```

### Security Tests Required

```go
// File upload attacks
- TestUpload_RejectsOversizedFile (11MB)
- TestUpload_RejectsMalware (EICAR test file)
- TestUpload_RejectsInvalidMIME
- TestUpload_RejectsCorruptedImage
- TestUpload_RejectsPolyglotFile (JPEG+HTML)
- TestUpload_SanitizesFilename
- TestUpload_PreventPathTraversal
- TestUpload_RateLimitEnforced

// Image processing
- TestProcessor_StripEXIFMetadata
- TestProcessor_ReEncodesImage
- TestProcessor_EnforcesDimensionLimits
- TestProcessor_EnforcesPixelCountLimit

// Storage security
- TestStorage_GeneratesNonGuessableKeys
- TestStorage_PreventsPathTraversal
- TestStorage_RespectsVisibilitySettings
```

### Recommended Controls

- [ ] Implement image content analysis (NSFW detection)
- [ ] Add image hash deduplication
- [ ] Monitor ClamAV scan failures/timeouts
- [ ] Implement progressive upload for large files

### SecOps Review Focus

- Image validation pipeline completeness
- ClamAV integration and signature management
- File upload attack surface (polyglot, bombs, malware)
- Storage security (access control, key randomness)
- EXIF stripping and privacy protection
- libvips configuration and memory limits
- Rate limiting on upload endpoints

### Known Risks & Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| Malware upload | Critical | ClamAV scanning + re-encoding through libvips |
| Polyglot files (JPEG+JS) | High | Re-encode all images, content type validation |
| Decompression bombs | High | Memory limits in libvips, pixel count limits |
| Path traversal | High | Sanitize filenames, validate paths, use UUIDs |
| EXIF data privacy leak | Medium | Strip all EXIF metadata |
| Storage DoS (upload spam) | Medium | Rate limiting + disk quotas |

---

## Sprint 6: Application & HTTP - Gallery Context

**Focus**: Gallery endpoints, albums, search, social features
**Security Risk**: HIGH (authorization, IDOR vulnerabilities)

### Mandatory Controls

| Control ID | Description | Pass Criteria | Verification |
|------------|-------------|---------------|--------------|
| **S6-AUTHZ-001** | Ownership verified on image read | User cannot read others' private images | Test: `TestImage_CannotAccessOthersPrivate` |
| **S6-AUTHZ-002** | Ownership verified on image update | User cannot update others' images | Test: `TestImage_CannotUpdateOthers` |
| **S6-AUTHZ-003** | Ownership verified on image delete | User cannot delete others' images | Test: `TestImage_CannotDeleteOthers` |
| **S6-AUTHZ-004** | Album ownership verified | User cannot modify others' albums | Test: `TestAlbum_OwnershipEnforced` |
| **S6-IDOR-001** | Image ID authorization prevents IDOR | Numeric ID enumeration impossible | Test: iterate IDs, verify 403/404 |
| **S6-IDOR-002** | Album ID authorization prevents IDOR | Numeric ID enumeration impossible | Test: iterate IDs, verify 403/404 |
| **S6-VAL-004** | Comment content sanitized | HTML/script tags stripped or encoded | Test: submit XSS payload in comment |
| **S6-VAL-005** | Search query sanitized for SQL injection | Parameterized queries only | Test: search for "'; DROP TABLE--" |
| **S6-SEARCH-001** | Search results respect visibility | Private images not in public search | Test: search for private image |
| **S6-RATE-003** | Comment spam prevention (rate limiting) | 10 comments/min per user | Integration test |

### Security Tests Required

```go
// Authorization (IDOR prevention)
- TestImage_GetPrivateByNonOwner_Returns403
- TestImage_UpdateByNonOwner_Returns403
- TestImage_DeleteByNonOwner_Returns403
- TestAlbum_AddImageByNonOwner_Returns403
- TestImage_EnumerationPrevention

// Input validation
- TestComment_RejectsXSSPayload
- TestSearch_PreventsSQLInjection
- TestTag_SanitizesInput

// Visibility enforcement
- TestGallery_PrivateImagesNotInPublicFeed
- TestSearch_ExcludesPrivateImages
- TestUser_CannotListOthersPrivateImages

// Rate limiting
- TestComment_RateLimitEnforced
- TestLike_RateLimitEnforced
```

### Recommended Controls

- [ ] Add content moderation queue for new images
- [ ] Implement CAPTCHA on comment submission
- [ ] Add user blocking feature
- [ ] Monitor for spam patterns

### SecOps Review Focus

- Authorization enforcement at every endpoint
- IDOR vulnerability testing (vertical and horizontal)
- Input sanitization on user-generated content
- Search functionality SQL injection prevention
- Visibility rules enforcement
- Rate limiting on social features

### Known Risks & Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| IDOR (Insecure Direct Object Reference) | High | Ownership checks at handler + application layer |
| XSS in comments | High | Input sanitization + output encoding |
| SQL injection in search | Critical | Parameterized queries only |
| Comment spam | Medium | Rate limiting + CAPTCHA |
| Visibility bypass | High | Enforce visibility at query level |

---

## Sprint 7: Moderation & Social Features

**Focus**: Content moderation, reporting, admin tools, RBAC
**Security Risk**: HIGH (privilege escalation, audit logging)

### Mandatory Controls

| Control ID | Description | Pass Criteria | Verification |
|------------|-------------|---------------|--------------|
| **S7-RBAC-001** | Role-based permissions enforced at handler | Middleware blocks unauthorized roles | Test: user accesses admin endpoint |
| **S7-RBAC-002** | Role-based permissions enforced at application | Double-check in command handlers | Code review |
| **S7-RBAC-003** | No privilege escalation paths | Users cannot grant themselves admin | Test: user calls role update API |
| **S7-RBAC-004** | Admin actions require re-authentication | Sensitive actions need password re-entry | Test: admin ban without recent auth |
| **S7-AUDIT-001** | All moderation actions logged | Ban, delete, approve logged with actor | Audit log verification |
| **S7-AUDIT-002** | Audit logs are immutable | Insert-only, no update/delete | Database constraint review |
| **S7-AUDIT-003** | Audit logs include IP and user agent | Context captured for investigations | Log schema review |
| **S7-MOD-001** | Report abuse prevention | Rate limit on report submission | Test: submit 100 reports rapidly |
| **S7-MOD-002** | Moderators cannot moderate own content | Self-moderation blocked | Test: mod approves own image |

### Security Tests Required

```go
// RBAC and privilege escalation
- TestRBAC_UserCannotAccessAdminEndpoint
- TestRBAC_ModeratorCannotGrantAdminRole
- TestRBAC_UserCannotEscalateOwnRole
- TestRBAC_AdminCanManageRoles

// Moderation security
- TestModeration_RequiresModeratorRole
- TestModeration_CannotModerateOwnContent
- TestReport_RateLimitEnforced
- TestBan_RequiresAdminRole
- TestBan_RequiresRecentAuthentication

// Audit logging
- TestAudit_LogsUserBan
- TestAudit_LogsImageDeletion
- TestAudit_LogsRoleChange
- TestAudit_LogsModeratorActions
```

### Recommended Controls

- [ ] Implement audit log alerting for suspicious patterns
- [ ] Add moderator performance dashboard
- [ ] Implement appeal process for bans
- [ ] Add two-person rule for critical actions

### SecOps Review Focus

- RBAC implementation (role assignment, permission checks)
- Privilege escalation prevention
- Audit logging completeness and immutability
- Moderator action security
- Admin panel security
- Report abuse mechanisms

### Known Risks & Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| Privilege escalation | Critical | Multi-layer permission checks, audit logging |
| Admin account compromise | Critical | Re-authentication for sensitive actions, MFA (future) |
| Audit log tampering | High | Immutable logs, separate storage |
| Moderator abuse | Medium | Audit logging, peer review of actions |
| Report spam | Medium | Rate limiting, anti-abuse detection |

---

## Sprint 8: Integration, Testing & Security Hardening

**Focus**: Comprehensive security testing, penetration testing, vulnerability remediation
**Security Risk**: CRITICAL (final security validation before launch)

### Mandatory Controls

| Control ID | Description | Pass Criteria | Verification |
|------------|-------------|---------------|--------------|
| **S8-SCAN-001** | gosec static analysis passes | Zero high-severity findings | `gosec -severity high ./...` |
| **S8-SCAN-002** | trivy container scan passes | Zero critical/high vulnerabilities | `trivy image goimg:latest` |
| **S8-SCAN-003** | nancy dependency scan passes | Zero critical/high vulnerabilities | `nancy sleuth` |
| **S8-SCAN-004** | OWASP ZAP dynamic scan passes | Zero high-risk findings | ZAP baseline scan |
| **S8-TEST-001** | All security test suites pass | 100% pass rate | CI security test job |
| **S8-TEST-002** | Penetration testing complete | No critical findings unresolved | Pentest report |
| **S8-LOAD-001** | Rate limiting validated under load | Holds under 10x normal traffic | Load test with k6 |
| **S8-SECRETS-001** | No secrets in container images | Trivy secret scan passes | `trivy fs --security-checks secret .` |
| **S8-SECRETS-002** | No secrets in git history | gitleaks scan passes | `gitleaks detect --verbose` |

### Security Test Suites

#### 1. Authentication Security Tests

```bash
# Test suite location: tests/security/auth_test.go
go test -v ./tests/security -run TestAuth
```

Tests:
- Token replay attacks
- Token forgery attempts
- Session fixation
- Brute force protection
- Account lockout
- Concurrent session handling

#### 2. Authorization Security Tests

```bash
# Test suite location: tests/security/authz_test.go
go test -v ./tests/security -run TestAuthz
```

Tests:
- Vertical privilege escalation (user → admin)
- Horizontal privilege escalation (user A → user B)
- IDOR on all resource types
- Missing function-level access control

#### 3. Input Validation Security Tests

```bash
# Test suite location: tests/security/injection_test.go
go test -v ./tests/security -run TestInjection
```

Tests:
- SQL injection on all endpoints
- NoSQL injection (if applicable)
- Command injection
- Path traversal
- XML/JSON injection
- LDAP injection (if applicable)

#### 4. File Upload Security Tests

```bash
# Test suite location: tests/security/upload_test.go
go test -v ./tests/security -run TestUpload
```

Tests:
- Malware upload (EICAR)
- Polyglot file upload (JPEG+JS)
- Oversized file upload
- Pixel flood attack
- MIME type bypass
- Path traversal via filename

#### 5. API Security Tests

```bash
# Test suite location: tests/security/api_test.go
go test -v ./tests/security -run TestAPI
```

Tests:
- Rate limiting enforcement
- CORS policy validation
- Security headers presence
- Request size limits
- API parameter tampering

### Penetration Testing Checklist

**Manual testing required:**

1. **Authentication**
   - [ ] Password reset flow (token expiry, replay)
   - [ ] Multi-factor authentication bypass (if implemented)
   - [ ] OAuth flow vulnerabilities (if implemented)

2. **Session Management**
   - [ ] Session fixation
   - [ ] Session hijacking
   - [ ] Logout functionality
   - [ ] Concurrent session handling

3. **Authorization**
   - [ ] Vertical privilege escalation (all roles)
   - [ ] Horizontal privilege escalation (all resources)
   - [ ] Forced browsing
   - [ ] Missing function-level access control

4. **Input Validation**
   - [ ] SQL injection (all parameters)
   - [ ] XSS (reflected, stored, DOM-based)
   - [ ] Command injection
   - [ ] XML external entities (XXE)
   - [ ] Server-side request forgery (SSRF)

5. **Business Logic**
   - [ ] Rate limit bypass techniques
   - [ ] Payment logic flaws (if applicable)
   - [ ] Workflow circumvention

6. **Cryptography**
   - [ ] Weak algorithms in use
   - [ ] Hardcoded keys/secrets
   - [ ] Insecure randomness

### Recommended Controls

- [ ] Implement Web Application Firewall (WAF)
- [ ] Add DDoS protection (Cloudflare, AWS Shield)
- [ ] Implement security monitoring (SIEM)
- [ ] Add breach detection monitoring

### SecOps Review Focus

- Review all security scan results
- Validate penetration test findings
- Verify remediation of identified vulnerabilities
- Assess residual risk for known issues
- Review security monitoring and alerting
- Validate incident response procedures

### Known Risks & Acceptance Criteria

| Finding | Severity | Accepted? | Remediation Plan |
|---------|----------|-----------|------------------|
| (Findings from pentest) | - | - | - |

---

## Sprint 9: MVP Polish & Launch Prep

**Focus**: Production hardening, monitoring, documentation, launch readiness
**Security Risk**: MEDIUM (operational security)

### Mandatory Controls

| Control ID | Description | Pass Criteria | Verification |
|------------|-------------|---------------|--------------|
| **S9-PROD-001** | Secrets manager configured (not env vars) | AWS Secrets Manager/Vault in use | Config review |
| **S9-PROD-002** | TLS/SSL certificates valid | Certs from trusted CA, not expired | Certificate check |
| **S9-PROD-003** | Database backups encrypted | Encryption at rest enabled | Backup config review |
| **S9-PROD-004** | Backup restoration tested | Restore completes successfully | Test restore procedure |
| **S9-MON-001** | Security event alerting configured | Alerts on auth failures, privilege escalation | Test alerts |
| **S9-MON-002** | Error tracking configured | Sentry/equivalent capturing errors | Test error submission |
| **S9-MON-003** | Audit log monitoring active | Anomaly detection configured | Review dashboard |
| **S9-DOC-001** | SECURITY.md created | Vulnerability disclosure policy | File exists |
| **S9-DOC-002** | Security runbook complete | Incident response procedures documented | Review runbook |
| **S9-COMP-001** | Data retention policy documented | GDPR/CCPA compliance addressed | Policy review |

### Pre-Launch Security Checklist

```markdown
## Infrastructure
- [ ] Production database uses SSL/TLS
- [ ] Redis uses authentication and TLS
- [ ] S3 buckets block public access
- [ ] IAM roles follow least privilege
- [ ] Network security groups restrict access
- [ ] No default credentials in use

## Application
- [ ] All dependencies up to date
- [ ] No known vulnerabilities (trivy scan)
- [ ] Rate limiting enabled
- [ ] Security headers configured
- [ ] CORS properly configured
- [ ] Secrets externalized

## Monitoring
- [ ] Security event logging enabled
- [ ] Alerts configured for:
  - Failed authentication attempts
  - Privilege escalation attempts
  - Rate limit violations
  - Error rate spikes
  - Malware detections
- [ ] On-call rotation established
- [ ] Incident response plan tested

## Compliance
- [ ] Privacy policy published
- [ ] Terms of service published
- [ ] Cookie consent (if EU traffic)
- [ ] GDPR data subject rights implemented
- [ ] Vulnerability disclosure policy published

## Documentation
- [ ] Security runbook complete
- [ ] Deployment guide includes security steps
- [ ] Disaster recovery procedures documented
- [ ] Audit log retention policy defined
```

### Recommended Controls

- [ ] Third-party security audit (external pentest)
- [ ] Bug bounty program setup
- [ ] Security awareness training for team
- [ ] Quarterly security review schedule

### SecOps Review Focus

- Production security configuration
- Secrets management and key rotation
- Monitoring and alerting effectiveness
- Incident response readiness
- Backup and disaster recovery procedures
- Compliance with applicable regulations

### Launch Approval Criteria

| Criteria | Status | Evidence |
|----------|--------|----------|
| All critical vulnerabilities resolved | ☐ | Pentest report, scan results |
| Security monitoring operational | ☐ | Alert test results |
| Incident response plan tested | ☐ | Tabletop exercise report |
| Compliance requirements met | ☐ | Legal review |
| Backups tested and working | ☐ | Restore test log |

---

## Continuous Security (Post-Launch)

### Ongoing Requirements

| Activity | Frequency | Owner |
|----------|-----------|-------|
| Dependency vulnerability scanning | Daily (CI) | DevOps |
| Container image scanning | Daily (CI) | DevOps |
| Security log review | Daily | SecOps |
| Penetration testing | Quarterly | External auditor |
| Security training | Quarterly | All engineers |
| Incident response drill | Bi-annually | SecOps |
| Access review | Quarterly | SecOps |
| Certificate renewal | Before expiry | DevOps |

### Vulnerability Response Workflow

```
Discovery → Triage → Risk Assessment → Remediation → Validation → Documentation
   (0h)      (4h)         (24h)           (varies)       (varies)      (7d)
```

**SLAs by Severity:**

| Severity | Response Time | Remediation Time |
|----------|---------------|------------------|
| Critical | 2 hours | 24 hours |
| High | 4 hours | 7 days |
| Medium | 1 day | 30 days |
| Low | 3 days | 90 days |

### Security Metrics

Track and report monthly:

- Mean time to detect (MTTD) security incidents
- Mean time to respond (MTTR) to incidents
- Number of vulnerabilities by severity
- Percentage of systems with up-to-date patches
- Failed authentication attempts
- Rate limit violations
- Malware detections

---

## Security Gate Sign-Off Template

Use this template for each sprint gate review:

```markdown
## Sprint X Security Gate Review

**Sprint**: Sprint X - [Name]
**Reviewer**: [SecOps Engineer Name]
**Date**: YYYY-MM-DD

### Mandatory Controls Review

| Control ID | Status | Evidence | Notes |
|------------|--------|----------|-------|
| SX-XXX-001 | ☐ PASS / ☐ FAIL | [Link/file] | |
| SX-XXX-002 | ☐ PASS / ☐ FAIL | [Link/file] | |

### Security Tests Review

- [ ] All required security tests implemented
- [ ] All security tests passing
- [ ] Code coverage meets requirements

### Findings

| ID | Severity | Description | Remediation | Status |
|----|----------|-------------|-------------|--------|
| F1 | Critical | [Description] | [Plan] | Open/Closed |

### Risk Assessment

**Residual Risks:**
- [List any accepted risks with justification]

### Recommendation

☐ **APPROVE** - All controls passed, no critical findings
☐ **CONDITIONAL APPROVE** - Minor findings, remediation plan acceptable
☐ **REJECT** - Critical findings must be resolved

**Signature**: ___________________
**Date**: ___________________
```

---

## References

- OWASP Top 10 2021: https://owasp.org/Top10/
- OWASP ASVS 4.0: https://owasp.org/www-project-application-security-verification-standard/
- CWE Top 25: https://cwe.mitre.org/top25/
- NIST Cybersecurity Framework: https://www.nist.gov/cyberframework
