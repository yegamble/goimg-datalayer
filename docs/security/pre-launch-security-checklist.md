# Pre-Launch Security Checklist

**Project:** goimg-datalayer
**Version:** 1.0
**Last Updated:** 2025-12-05

This checklist must be completed and approved before production deployment. All items marked as **[LAUNCH BLOCKER]** must be resolved. High priority items should be resolved before launch or have documented compensating controls.

---

## Legend

- ✅ **Completed** - Verified and documented
- ⚠️ **In Progress** - Work underway
- ❌ **Not Started** - Requires attention
- ⏭️ **Not Applicable** - Not relevant for this deployment
- 🔒 **LAUNCH BLOCKER** - Must be resolved before production

---

## 1. Authentication & Session Management

### 1.1 JWT Token Security

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | JWT uses asymmetric algorithm (RS256, not HS256) | Verify in `/internal/infrastructure/security/jwt/jwt_service.go` |
| [ ] | 🔒 | Private key is >= 4096-bit RSA | Check key generation process |
| [ ] | 🔒 | Private key stored in secrets manager (not in code/env vars) | AWS Secrets Manager, Vault, etc. |
| [ ] | 🔒 | Public key distributed securely | Document key distribution method |
| [ ] | High | Access token TTL <= 15 minutes | Default: 15 minutes |
| [ ] | High | Refresh token TTL <= 7 days | Default: 7 days |
| [ ] | 🔒 | Token signature validation cannot be bypassed | Test with algorithm confusion attack |
| [ ] | 🔒 | No sensitive data in JWT payload | Review claims structure |
| [ ] | High | Token blacklist functional (Redis) | Test logout flow |
| [ ] | High | Token expiration strictly enforced | Test with expired tokens |

### 1.2 Session Management

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | New session ID generated on login | Prevents session fixation |
| [ ] | 🔒 | Session invalidated on logout | Blacklist check working |
| [ ] | High | Session ID is cryptographically random | UUID v4 |
| [ ] | High | Multi-device session tracking implemented | Redis session store |
| [ ] | Medium | IP/User-Agent anomaly detection active | Log suspicious changes |
| [ ] | Medium | Session timeout configured | Aligned with refresh token TTL |

### 1.3 Password Security

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | Passwords hashed with bcrypt (cost >= 12) | Check `internal/domain/identity/password.go` |
| [ ] | 🔒 | Password complexity requirements enforced | Min 8 chars, mix of types |
| [ ] | High | Account lockout after 5 failed attempts | Test brute force protection |
| [ ] | High | Lockout duration: 15 minutes | Configurable, documented |
| [ ] | High | Password reset flow secure | Token expiration, single-use |
| [ ] | Medium | No password hints or security questions | Modern best practice |

### 1.4 Account Enumeration Prevention

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | Generic error messages for login failures | "Invalid email or password" |
| [ ] | High | Constant-time password comparison | No timing attacks |
| [ ] | High | Registration confirms email without revealing existence | "Check your email" for all attempts |
| [ ] | Medium | Rate limiting on registration endpoint | Prevent enumeration via signup |

---

## 2. Authorization & Access Control

### 2.1 RBAC (Role-Based Access Control)

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | User roles properly defined (user, moderator, admin) | Document in `/docs/security/rbac.md` |
| [ ] | 🔒 | Role elevation requires admin action (no self-promotion) | Test parameter tampering |
| [ ] | 🔒 | Admin endpoints protected by role middleware | `RequireRole("admin")` |
| [ ] | High | Moderator permissions documented | Clear permission boundaries |
| [ ] | High | Role changes logged for audit | Security event logging |

### 2.2 Resource Ownership Validation

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | IDOR tests pass for images | User A cannot access User B's private images |
| [ ] | 🔒 | IDOR tests pass for albums | Cross-user album access prevented |
| [ ] | 🔒 | IDOR tests pass for comments | Only owner/moderator can delete |
| [ ] | 🔒 | Image deletion requires ownership | Or admin role |
| [ ] | 🔒 | Image update requires ownership | Metadata modification restricted |
| [ ] | High | Public images accessible to all | Visibility rules enforced |
| [ ] | High | Private images only to owner | Even with direct URL |
| [ ] | High | Unlisted images accessible via direct link | Documented behavior |

### 2.3 Function-Level Access Control

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | All protected endpoints require JWT auth | No missing authentication |
| [ ] | 🔒 | Admin endpoints inaccessible to regular users | Test with user token |
| [ ] | High | User profile modification restricted to owner | Or admin |
| [ ] | High | Bulk operations protected | Delete multiple, etc. |

---

## 3. Input Validation & Injection Prevention

### 3.1 SQL Injection

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | All SQL queries use parameterized statements | No string concatenation |
| [ ] | 🔒 | Search parameters validated | `?q=` in image search |
| [ ] | 🔒 | Filter parameters validated | User listing filters |
| [ ] | 🔒 | Sorting parameters whitelisted | Prevent ORDER BY injection |
| [ ] | High | SQL injection tests pass | Run `/tests/security/injection-tests.sh` |
| [ ] | High | Database errors not exposed to users | Generic error messages |

### 3.2 XSS (Cross-Site Scripting)

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | Image metadata sanitized (title, description) | HTML escaped or stripped |
| [ ] | 🔒 | User input in responses properly escaped | JSON encoding |
| [ ] | High | XSS payloads in error messages escaped | Reflected XSS prevention |
| [ ] | High | Comment text sanitized | bluemonday or similar |
| [ ] | High | Tag names sanitized | No script tags in tags |
| [ ] | Medium | CSP header configured | Content-Security-Policy |

### 3.3 Command Injection

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | Filenames sanitized before processing | No shell metacharacters |
| [ ] | 🔒 | No user input passed to system commands | Use libraries, not exec |
| [ ] | High | Path traversal prevented | `../` sequences blocked |
| [ ] | High | Null byte injection prevented | Filename validation |

### 3.4 Other Injection Vectors

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | CRLF injection prevented | Header injection blocked |
| [ ] | Medium | Template injection tested | No server-side template evaluation |
| [ ] | Medium | LDAP injection (if applicable) | Not applicable if no LDAP |

---

## 4. File Upload Security

### 4.1 Malware Scanning

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | ClamAV daemon running and accessible | Docker container healthy |
| [ ] | 🔒 | EICAR test file rejected | Run `TC-UPLOAD-001` test |
| [ ] | 🔒 | Malware scan integrated in upload flow | Before storage |
| [ ] | High | ClamAV signatures up-to-date | freshclam running |
| [ ] | High | Scan failures handled gracefully | Reject upload on scan error |
| [ ] | High | Malware detection logged | Security event |

### 4.2 File Type Validation

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | MIME type validation enforced | Not just file extension |
| [ ] | 🔒 | Magic number (file signature) validation | Detect polyglot files |
| [ ] | 🔒 | Allowed formats whitelisted | JPEG, PNG, GIF, WebP only |
| [ ] | 🔒 | SVG files rejected or sanitized | XSS vector |
| [ ] | High | Polyglot files re-encoded | bimg/libvips processing |
| [ ] | High | Double extensions rejected | `.jpg.php` blocked |
| [ ] | High | Content-type mismatch detected | PHP file with image MIME type |

### 4.3 File Size & Resource Limits

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | Maximum file size enforced | 10MB default |
| [ ] | 🔒 | Oversized files rejected with 413 | No memory exhaustion |
| [ ] | High | Upload rate limiting configured | Per user, per hour |
| [ ] | High | Total storage quota per user | Prevent storage abuse |
| [ ] | Medium | Concurrent upload limits | Prevent resource exhaustion |

### 4.4 Metadata & EXIF

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | EXIF data stripped during processing | Privacy protection |
| [ ] | High | Malicious EXIF payloads handled | XSS in EXIF comments |
| [ ] | Medium | GPS coordinates optionally preserved | User consent required |

---

## 5. Security Headers & Configuration

### 5.1 HTTP Security Headers

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | Strict-Transport-Security (HSTS) enabled | Production only, max-age=31536000 |
| [ ] | 🔒 | Content-Security-Policy configured | Restrict script sources |
| [ ] | High | X-Content-Type-Options: nosniff | Prevent MIME sniffing |
| [ ] | High | X-Frame-Options: DENY | Clickjacking prevention |
| [ ] | High | X-XSS-Protection: 1; mode=block | Legacy browsers |
| [ ] | High | Referrer-Policy configured | strict-origin-when-cross-origin |
| [ ] | Medium | Permissions-Policy configured | Disable dangerous features |

### 5.2 CORS Configuration

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | CORS origins whitelisted | No wildcard in production |
| [ ] | High | Credentials properly handled | Access-Control-Allow-Credentials |
| [ ] | High | Preflight requests handled | OPTIONS method |
| [ ] | Medium | Unnecessary headers not exposed | Access-Control-Expose-Headers |

### 5.3 TLS/SSL Configuration

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | TLS 1.2 minimum (prefer TLS 1.3) | Disable TLS 1.0, 1.1 |
| [ ] | 🔒 | Valid SSL certificate installed | Not self-signed |
| [ ] | 🔒 | Certificate expiration monitoring | Alerts before expiry |
| [ ] | High | Strong cipher suites only | No weak ciphers (RC4, 3DES) |
| [ ] | High | HTTPS redirect for HTTP requests | Force secure connection |
| [ ] | Medium | OCSP stapling enabled | Performance + privacy |

---

## 6. Dependency & Code Security

### 6.1 Dependency Management

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | Go version is supported (not EOL) | Currently: Go 1.24 |
| [ ] | 🔒 | No Critical CVEs in dependencies | Run `govulncheck ./...` |
| [ ] | High | Dependency versions documented | `go.mod` committed |
| [ ] | High | Regular dependency updates scheduled | Monthly review |
| [ ] | Medium | Unused dependencies removed | Clean `go.mod` |
| [ ] | Medium | Dependabot or Renovate configured | Automated updates |

### 6.2 Static Code Analysis

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | gosec scan passes | No critical findings |
| [ ] | High | golangci-lint configured | In CI/CD pipeline |
| [ ] | High | No hardcoded secrets in code | Run `git secrets` |
| [ ] | Medium | Code coverage >= 80% | Security-critical paths tested |
| [ ] | Medium | SAST tools in CI/CD | Automated security scanning |

### 6.3 Secret Management

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | No secrets in environment variables | Use secrets manager |
| [ ] | 🔒 | JWT keys in secrets manager | Not in Docker image |
| [ ] | 🔒 | Database credentials in secrets manager | Not hardcoded |
| [ ] | High | AWS credentials managed via IAM roles | No access keys |
| [ ] | High | Redis password configured | requirepass in production |
| [ ] | High | Secrets rotation policy defined | Quarterly rotation |

---

## 7. Logging & Monitoring

### 7.1 Security Logging

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | Failed login attempts logged | Include IP, timestamp |
| [ ] | 🔒 | Account lockouts logged | Security incident |
| [ ] | 🔒 | Token revocations logged | Logout, security events |
| [ ] | High | Successful logins logged | Audit trail |
| [ ] | High | Authorization failures logged | Permission denied events |
| [ ] | High | Malware detection logged | Include virus name |
| [ ] | Medium | Admin actions logged | User management, role changes |
| [ ] | Medium | File uploads logged | User, file size, type |

### 7.2 Sensitive Data Protection in Logs

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | Passwords never logged | Not even hashed |
| [ ] | 🔒 | JWT tokens not logged | Only token ID (jti) |
| [ ] | 🔒 | Email addresses hashed in logs | SHA-256 for privacy |
| [ ] | High | Credit card numbers not logged | If payment implemented |
| [ ] | High | PII redacted in logs | GDPR compliance |

### 7.3 Monitoring & Alerting

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | Failed login rate alerts | > 10/min from single IP |
| [ ] | High | Account lockout alerts | Unusual spike |
| [ ] | High | Malware detection alerts | Immediate escalation |
| [ ] | High | Anomaly detection configured | IP/location changes |
| [ ] | Medium | Token replay detection alerts | Refresh token reuse |
| [ ] | Medium | Rate limit exceeded alerts | Potential DoS |

---

## 8. Rate Limiting & DoS Prevention

### 8.1 Authentication Rate Limits

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | Login endpoint rate limited | 5 attempts/min per IP |
| [ ] | High | Registration rate limited | Prevent fake accounts |
| [ ] | High | Password reset rate limited | 3 requests/hour per email |
| [ ] | High | Token refresh rate limited | Prevent abuse |

### 8.2 API Rate Limits

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | Global rate limit configured | 100 req/min per IP |
| [ ] | High | Authenticated rate limit higher | 300 req/min per user |
| [ ] | High | Upload rate limit strict | 50 uploads/hour per user |
| [ ] | Medium | Rate limit headers included | X-RateLimit-* headers |
| [ ] | Medium | 429 Too Many Requests returned | RFC compliant |

### 8.3 Resource Protection

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | Request timeout configured | 30 seconds |
| [ ] | High | Maximum request body size | 50MB |
| [ ] | High | Connection limits configured | Prevent exhaustion |
| [ ] | Medium | Slowloris protection | Reverse proxy configuration |

---

## 9. Data Protection & Privacy

### 9.1 Data Encryption

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | Data encrypted in transit (TLS) | All communication |
| [ ] | High | Database encryption at rest | If storing PII |
| [ ] | High | S3 bucket encryption enabled | If using S3 |
| [ ] | Medium | Redis encryption in transit | TLS connection |

### 9.2 GDPR Compliance

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | Privacy policy published | User consent documented |
| [ ] | High | Data retention policy defined | How long data is kept |
| [ ] | High | User data export implemented | GDPR right to access |
| [ ] | High | User data deletion implemented | GDPR right to erasure |
| [ ] | Medium | Cookie consent implemented | If using cookies |
| [ ] | Medium | DPO contact published | Data protection officer |

### 9.3 PCI-DSS (if handling payments)

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | ⏭️ | Use payment processor (Stripe, etc.) | No card data stored |
| [ ] | ⏭️ | PCI SAQ-A completed | If using external processor |
| [ ] | ⏭️ | Cardholder data never logged | Critical requirement |

---

## 10. Incident Response & Business Continuity

### 10.1 Incident Response Plan

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | Incident response plan documented | `/docs/security/incident-response.md` |
| [ ] | High | Security contact published | security@example.com |
| [ ] | High | Escalation procedures defined | Who to notify, when |
| [ ] | Medium | Post-incident review process | Learn from incidents |

### 10.2 Backup & Recovery

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | Database backups automated | Daily minimum |
| [ ] | High | Backup restoration tested | Quarterly test |
| [ ] | High | Point-in-time recovery possible | Transaction logs |
| [ ] | Medium | Disaster recovery plan | RPO/RTO defined |

---

## 11. Third-Party Integrations

### 11.1 ClamAV (Malware Scanning)

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | ClamAV signatures auto-update | freshclam running |
| [ ] | High | ClamAV version supported | Not EOL |
| [ ] | High | Network isolation configured | No internet access needed |
| [ ] | Medium | Resource limits set | Memory, CPU caps |

### 11.2 AWS S3 (Object Storage)

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | Bucket not publicly accessible | Private by default |
| [ ] | High | IAM roles used (not access keys) | Credential-less access |
| [ ] | High | Versioning enabled | Prevent accidental deletion |
| [ ] | Medium | Lifecycle policies configured | Cost optimization |

### 11.3 Redis (Cache/Session Store)

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | requirepass configured | Authentication required |
| [ ] | High | Network isolation | Not internet-accessible |
| [ ] | High | Persistence configured | AOF or RDB |
| [ ] | Medium | Memory limits set | maxmemory policy |

---

## 12. Penetration Testing & Security Validation

### 12.1 Automated Security Tests

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | 🔒 | Authentication tests pass | `/tests/security/auth-security-tests.sh` |
| [ ] | 🔒 | Injection tests pass | `/tests/security/injection-tests.sh` |
| [ ] | 🔒 | Upload security tests pass | `/tests/security/upload-security-tests.sh` |
| [ ] | 🔒 | Access control tests pass | `/tests/security/access-control-tests.sh` |
| [ ] | High | No Critical findings | CVSS 9.0-10.0 |
| [ ] | High | No High findings unmitigated | CVSS 7.0-8.9 |

### 12.2 Manual Penetration Testing

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | Penetration test completed | External or internal team |
| [ ] | High | All Critical findings resolved | Or compensating controls |
| [ ] | High | High findings documented | Risk acceptance if not fixed |
| [ ] | Medium | Retest completed | Verify fixes effective |

### 12.3 Security Documentation

| Status | Priority | Item | Notes |
|--------|----------|------|-------|
| [ ] | High | security.txt published | /.well-known/security.txt |
| [ ] | High | Vulnerability disclosure policy | How to report bugs |
| [ ] | Medium | Security architecture diagram | Document design |
| [ ] | Medium | Threat model documented | STRIDE or similar |

---

## Approval & Sign-off

### Pre-Launch Review

| Role | Name | Signature | Date |
|------|------|-----------|------|
| **Security Engineer** | | | |
| **Engineering Lead** | | | |
| **Product Owner** | | | |
| **DevOps Lead** | | | |

### Post-Launch Review (30 days)

| Role | Name | Signature | Date |
|------|------|-----------|------|
| **Security Engineer** | | | |
| **Engineering Lead** | | | |

---

## Continuous Improvement

This checklist should be reviewed and updated:
- **Monthly**: Review findings and update priorities
- **Quarterly**: Full security audit and dependency updates
- **Annually**: Penetration test and threat model review
- **After incidents**: Update based on lessons learned

**Version History:**

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | 2025-12-05 | Initial checklist | Senior SecOps Engineer |

