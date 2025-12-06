# Security Policy

This document outlines the security policy for goimg-datalayer, including how to report vulnerabilities, our response process, and security best practices for contributors.

## Supported Versions

We provide security updates for the following versions:

| Version | Supported          | Status |
| ------- | ------------------ | ------ |
| 1.x.x   | :white_check_mark: | Active development (MVP) |
| < 1.0   | :x:                | Pre-release, no support |

Once the project reaches stable release, we will maintain security updates for the current major version and the previous major version for 6 months after a new major version is released.

## Security Philosophy

goimg-datalayer follows a defense-in-depth security approach with multiple layers of protection:

- **Authentication**: RS256 JWT with 4096-bit keys, refresh token rotation, replay detection
- **Authorization**: Role-based access control (RBAC) with ownership validation
- **Input Validation**: 7-step image validation pipeline including ClamAV malware scanning
- **Rate Limiting**: Multiple tiers (5/min login, 100/min global, 50/hour uploads)
- **Secure Headers**: CSP, HSTS, X-Frame-Options, and other security headers
- **Data Protection**: Argon2id password hashing, encrypted database connections
- **Audit Logging**: Comprehensive structured logging for security events

**Security Rating** (as of Sprint 8): **B+** with zero critical or high-severity vulnerabilities.

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please report it responsibly.

### How to Report

**DO NOT** open a public GitHub issue for security vulnerabilities. Instead:

1. **Email**: Send details to **security@goimg-datalayer.example.com** (replace with actual contact)
2. **Subject Line**: Use "SECURITY: [Brief Description]"
3. **Encrypted Communication**: For sensitive disclosures, request our PGP key

### What to Include

Help us understand and reproduce the issue by providing:

- **Description**: Clear explanation of the vulnerability
- **Impact**: Potential security impact and affected components
- **Steps to Reproduce**: Detailed steps to reproduce the issue
- **Proof of Concept**: Code, screenshots, or exploit demonstrating the issue
- **Environment**: Affected versions, configurations, and dependencies
- **Suggested Fix**: If you have ideas for remediation (optional but appreciated)

### What NOT to Include

- Do not include actual exploit code that could be weaponized
- Do not test vulnerabilities on production systems without permission
- Do not share the vulnerability publicly before we have had a chance to address it

## Response Process

### Our Commitment

When you report a vulnerability, we commit to:

1. **Acknowledge receipt** within **48 hours** (business days)
2. **Initial assessment** within **5 business days**
3. **Regular updates** at least every **7 days** until resolution
4. **Coordinated disclosure** working with you on timing

### Response Timeline

| Severity | Initial Response | Target Fix | Target Release |
|----------|-----------------|-----------|----------------|
| Critical | 24 hours | 7 days | 14 days |
| High | 48 hours | 14 days | 30 days |
| Medium | 5 days | 30 days | Next release |
| Low | 7 days | 60 days | Next release |

### Severity Classification

We use the following criteria to classify vulnerabilities:

**Critical**: Remote code execution, authentication bypass, privilege escalation to admin, data breach of all users
- Example: SQL injection allowing database dump, JWT secret exposure

**High**: Privilege escalation, data breach of specific users, stored XSS, IDOR allowing unauthorized access
- Example: IDOR allowing access to other users' private images, session hijacking

**Medium**: Information disclosure, denial of service, rate limit bypass, reflected XSS
- Example: User enumeration, account lockout bypass, EXIF data leakage

**Low**: Security misconfigurations, non-exploitable issues, theoretical vulnerabilities
- Example: Missing security headers on non-critical endpoints, verbose error messages

## Security Updates

### Notification Channels

We announce security updates through:

- **GitHub Security Advisories**: Primary channel for CVE assignments
- **Release Notes**: All releases include security fix details (after disclosure)
- **Security Mailing List**: Subscribe at security-announce@goimg-datalayer.example.com

### Update Process

1. Security patches are released as soon as they are validated
2. Critical and high-severity issues receive out-of-band releases
3. We backport critical fixes to supported versions
4. All security fixes include CVE identifiers when applicable

## Bug Bounty Program

**Status**: Not currently available

We do not currently offer a bug bounty program. However, we deeply appreciate responsible disclosure and will:

- Publicly acknowledge your contribution (if you wish)
- Credit you in release notes and security advisories
- Provide a detailed response about the issue and fix

We may establish a bug bounty program post-MVP based on project adoption.

## Security Best Practices for Contributors

If you contribute to goimg-datalayer, please follow these security guidelines:

### Code Security

- **Never commit secrets**: Use environment variables for all credentials
- **Validate all input**: Assume all user input is malicious
- **Use parameterized queries**: Never concatenate SQL strings
- **Wrap errors**: Use `fmt.Errorf("context: %w", err)` for proper error chains
- **Test security controls**: Include test cases for auth, authz, and validation failures

### Authentication & Authorization

- **Check ownership**: Verify users own resources before mutations
- **Require authentication**: Protected endpoints must use auth middleware
- **Validate roles**: Use RBAC middleware for admin/moderator actions
- **Log security events**: Record all authentication and authorization failures

### Data Protection

- **Hash passwords**: Use Argon2id (configured in `internal/domain/identity/password.go`)
- **Strip EXIF**: Remove metadata from uploaded images
- **Sanitize HTML**: Use bluemonday for user-generated content
- **Encrypt at rest**: Use database encryption for sensitive fields

### API Security

- **Rate limit**: Apply appropriate rate limits to all endpoints
- **RFC 7807 errors**: Use ProblemDetail format for all errors
- **Security headers**: Ensure middleware adds CSP, HSTS, etc.
- **CORS configuration**: Restrict origins to trusted domains

### Dependencies

- **Pin versions**: Use exact versions in `go.mod`
- **Review updates**: Check changelogs before updating dependencies
- **Scan regularly**: CI runs Trivy and gosec on every commit
- **Monitor advisories**: Subscribe to Go security announcements

### Testing

- **Security tests**: Add tests for common vulnerabilities (SQLi, XSS, IDOR)
- **Negative tests**: Test that unauthorized actions are blocked
- **Fuzz testing**: Consider fuzzing for input validation functions
- **E2E security**: Include security scenarios in Newman/Postman tests

## Compliance & Standards

goimg-datalayer follows these security standards:

- **OWASP Top 10**: All vulnerabilities addressed (see Sprint 8 security audit)
- **CWE/SANS Top 25**: Common weaknesses mitigated in design
- **NIST Guidelines**: Password requirements follow NIST SP 800-63B
- **GDPR/CCPA**: Data retention and deletion capabilities (user privacy rights)

## Security Audit History

| Date | Type | Findings | Status |
|------|------|----------|--------|
| 2025-12-05 | Internal (Sprint 8) | 0 critical, 0 high | **B+ Rating** |
| TBD | External (Pre-Launch) | Pending | Planned for Sprint 9 |

See `docs/sprint8-security-audit.md` for detailed findings from the Sprint 8 audit.

## Incident Response

In the event of a security incident:

1. **Detection**: Security monitoring and alerting (Prometheus, logs)
2. **Containment**: Immediate mitigation steps documented in runbooks
3. **Eradication**: Root cause analysis and permanent fix
4. **Recovery**: Restore services with verified fix deployed
5. **Post-Incident**: Publish security advisory and post-mortem

See the security runbook documentation for detailed operational procedures:
- `docs/security/incident_response.md` - Incident detection, triage, containment, and recovery
- `docs/security/monitoring.md` - Security event monitoring and alerting
- `docs/security/secret_rotation.md` - Credential rotation procedures
- `docs/security/data_retention_policy.md` - Data retention, GDPR/CCPA compliance, and user privacy rights

## Security Tools

Our security toolchain includes:

| Tool | Purpose | Frequency |
|------|---------|-----------|
| **gosec** | Static analysis for Go vulnerabilities | Every commit (CI) |
| **Trivy** | Container and dependency scanning | Every commit (CI) |
| **Gitleaks** | Secret scanning in commits | Every commit (CI) |
| **ClamAV** | Malware detection in uploads | Every upload (runtime) |
| **golangci-lint** | Code quality and security linting | Every commit (CI) |
| **Nancy** | Dependency vulnerability checking | Planned (Sprint 9) |

All security scans must pass before code can be merged.

## Security Contact

For security-related inquiries:

- **Email**: security@goimg-datalayer.example.com
- **GitHub**: Open a security advisory on the [goimg-datalayer repository](https://github.com/yegamble/goimg-datalayer/security/advisories)
- **Response Time**: 48 hours for acknowledgment

For general questions about security features, use the public GitHub Discussions.

## Acknowledgments

We thank the following researchers for responsible disclosure:

- (None yet - be the first!)

Your contributions help make goimg-datalayer more secure for everyone.

## Additional Resources

- [Architecture Documentation](claude/architecture.md) - Security design patterns
- [API Security Guide](claude/api_security.md) - HTTP security controls
- [Security Testing Guide](claude/security_testing.md) - Testing security controls
- [OpenAPI Specification](api/openapi/openapi.yaml) - API contracts and validation

---

**Last Updated**: 2025-12-05 (Sprint 9)

**Next Review**: Before production launch (Sprint 9 completion)
