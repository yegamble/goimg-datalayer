# Security Incident Response Runbook

> Production security incident handling procedures for goimg-datalayer

## Overview

This document defines the incident response process for security events, from initial detection through post-incident review. All security incidents must follow this standardized workflow to ensure consistent handling and continuous improvement.

**Document Owner**: Security Operations Team
**Last Updated**: 2025-12-05
**Review Frequency**: Quarterly

---

## Incident Severity Classification

### Critical (P0)

**Definition**: Immediate threat to production systems, data integrity, or user privacy requiring 24/7 response.

**Examples**:
- Remote code execution (RCE) vulnerability actively exploited
- Database breach with user data exfiltration
- Authentication bypass allowing unauthorized admin access
- JWT signing key compromise
- ClamAV malware detection bypassed with confirmed malicious uploads
- Ransomware or cryptomining detected in infrastructure
- Complete service unavailability due to security event (DDoS, resource exhaustion)

**Response SLA**:
- **Detection → Acknowledgment**: 15 minutes
- **Acknowledgment → Initial Containment**: 1 hour
- **Containment → Resolution**: 24 hours
- **Communication**: Immediate escalation to leadership, stakeholder notification within 2 hours

**On-Call Requirements**: Immediate paging, war room established

---

### High (P1)

**Definition**: Significant security risk with potential for data exposure or service degradation requiring urgent attention.

**Examples**:
- Privilege escalation vulnerability allowing user→moderator elevation
- IDOR (Insecure Direct Object Reference) exposing private images
- SQL injection vulnerability (not yet exploited)
- Stored XSS affecting multiple users
- Redis/PostgreSQL credentials leaked in logs
- Session hijacking detected across multiple accounts
- Rate limiting bypass allowing API abuse
- Unauthorized access to user accounts (credential stuffing)

**Response SLA**:
- **Detection → Acknowledgment**: 30 minutes
- **Acknowledgment → Initial Containment**: 4 hours
- **Containment → Resolution**: 48 hours
- **Communication**: Security team notified within 1 hour, leadership updated every 4 hours

**On-Call Requirements**: Standard business hours response, after-hours escalation if exploitation detected

---

### Medium (P2)

**Definition**: Security issue with limited impact or low exploitability requiring timely remediation.

**Examples**:
- Reflected XSS on non-critical endpoints
- Information disclosure (version numbers, internal paths in errors)
- User enumeration vulnerability
- EXIF metadata not properly stripped from images
- Missing security headers (CSP, HSTS) on specific endpoints
- Weak password policy not enforced on legacy accounts
- Incomplete audit logging for administrative actions
- Third-party dependency with medium-severity CVE

**Response SLA**:
- **Detection → Acknowledgment**: 4 hours (business hours)
- **Acknowledgment → Initial Containment**: 24 hours
- **Containment → Resolution**: 7 days
- **Communication**: Security team notified, weekly status updates

**On-Call Requirements**: Business hours only, no paging

---

### Low (P3)

**Definition**: Security improvement or theoretical vulnerability with minimal risk requiring planned remediation.

**Examples**:
- Non-exploitable security misconfiguration
- Verbose error messages with no sensitive data
- Best practice violations (e.g., missing `X-Content-Type-Options`)
- Documentation gaps in security procedures
- Security test coverage below threshold
- Dependency with low-severity CVE and no exploit available
- Security hardening recommendations from audits

**Response SLA**:
- **Detection → Acknowledgment**: 5 business days
- **Acknowledgment → Initial Containment**: N/A
- **Containment → Resolution**: 30 days or next sprint
- **Communication**: Tracked in backlog, quarterly review

**On-Call Requirements**: None, scheduled work

---

## Incident Response Workflow

### Phase 1: Detection

**Trigger Sources**:
- Automated security monitoring alerts (Prometheus, logs)
- SIEM correlation rules
- Vulnerability scanner reports (Trivy, gosec)
- User reports via security@goimg-datalayer.example.com
- Third-party security researcher disclosure
- Anomaly detection (rate limit violations, unusual access patterns)
- ClamAV malware detections
- Failed authentication spike alerts

**Immediate Actions**:
1. Create incident ticket in tracking system (prefix: `SEC-YYYY-NNNN`)
2. Assign severity classification (P0-P3)
3. Page on-call security engineer if P0/P1
4. Begin incident timeline documentation (all actions timestamped)
5. Preserve evidence (logs, network captures, affected system snapshots)

**Documentation Requirements**:
```yaml
Incident ID: SEC-2025-0042
Severity: P1
Detected At: 2025-12-05T14:32:18Z
Detected By: Prometheus alert (auth_failures_high)
Initial Assessment: Credential stuffing attack targeting /api/v1/auth/login
Affected Systems: Authentication service, PostgreSQL user table
Estimated Impact: 247 user accounts, no confirmed breaches
```

---

### Phase 2: Triage

**Triage Lead Responsibilities**:
- Validate incident severity classification
- Confirm incident is security-related (vs. availability/performance)
- Identify affected systems, users, and data
- Assess blast radius and potential data exposure
- Determine if incident is ongoing or historical
- Assign incident commander if P0/P1

**Triage Checklist**:
```markdown
- [ ] Severity classification confirmed by senior security engineer
- [ ] Affected systems identified and documented
- [ ] User impact assessed (number of users, data types)
- [ ] Attack vector confirmed (external, insider, supply chain)
- [ ] Indicators of compromise (IOCs) documented
- [ ] Evidence preservation completed (logs, snapshots, network captures)
- [ ] Legal/compliance notification requirement assessed
- [ ] Stakeholder communication plan drafted
```

**Triage Questions**:
1. Is this an active attack or historical finding?
2. What data was accessed/modified/exfiltrated?
3. What is the attack vector and root cause?
4. Are there additional compromised systems?
5. What is the business impact (financial, reputational, legal)?
6. Do we have sufficient logging to investigate fully?

**Escalation Criteria** (elevate to next severity):
- Evidence of data exfiltration
- Multiple attack vectors simultaneously
- Confirmed exploitation of vulnerability
- Regulatory notification requirement (GDPR breach)
- Media/public disclosure imminent

---

### Phase 3: Containment

**Objective**: Stop the incident from spreading while preserving evidence for investigation.

#### Immediate Containment (P0/P1)

**Network-Level Actions**:
```bash
# Block malicious IP addresses (example)
iptables -A INPUT -s 203.0.113.42 -j DROP

# Rate limit endpoint under attack
# Update rate limiter config (Redis)
redis-cli SET "goimg:ratelimit:/api/v1/auth/login" "1/minute"

# Isolate compromised container
docker network disconnect goimg-network compromised-container-id
```

**Application-Level Actions**:
```bash
# Revoke all JWT tokens for compromised user
# Execute in production environment
curl -X POST https://api.goimg.example.com/admin/v1/users/{user_id}/revoke-all \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Blacklist compromised JWT family
redis-cli SADD "goimg:blacklist:families" "family-uuid-123"

# Force password reset for affected users
psql -h $DB_HOST -U $DB_USER -d goimg <<SQL
UPDATE users
SET status = 'password_reset_required',
    updated_at = NOW()
WHERE id IN (SELECT user_id FROM security_incidents WHERE incident_id = 'SEC-2025-0042');
SQL

# Disable compromised API keys
UPDATE api_keys
SET status = 'revoked', revoked_at = NOW(), revoked_reason = 'SEC-2025-0042'
WHERE key_hash IN ('sha256-hash-1', 'sha256-hash-2');
```

**Infrastructure Actions**:
```bash
# Rotate compromised secrets immediately
# See secret_rotation.md for detailed procedures

# Database credentials
/home/user/goimg-datalayer/scripts/rotate_db_password.sh

# Redis password
/home/user/goimg-datalayer/scripts/rotate_redis_password.sh

# JWT signing keys (emergency rotation)
/home/user/goimg-datalayer/scripts/emergency_jwt_rotation.sh

# Isolate compromised container/VM
docker stop compromised-container-id
# OR
virsh suspend compromised-vm-name

# Take forensic snapshot before modification
docker commit compromised-container-id forensic-snapshot-$(date +%Y%m%d-%H%M%S)
```

#### System Hardening (Ongoing Containment)

```markdown
- [ ] Enable additional logging for affected components
- [ ] Deploy temporary rate limiting/IP blocklisting
- [ ] Enable step-up authentication for sensitive operations
- [ ] Restrict database access to read-only if data integrity questioned
- [ ] Deploy hot-patch if available (with rollback plan)
- [ ] Increase monitoring sensitivity (lower alert thresholds)
```

**Containment Documentation**:
```yaml
Containment Actions:
  - timestamp: 2025-12-05T14:45:00Z
    action: Blocked 42 IP addresses via iptables
    performed_by: security-engineer@example.com

  - timestamp: 2025-12-05T14:47:30Z
    action: Revoked JWT tokens for 247 affected users
    performed_by: security-engineer@example.com

  - timestamp: 2025-12-05T15:02:15Z
    action: Rotated PostgreSQL password (emergency)
    performed_by: devops-engineer@example.com
```

---

### Phase 4: Eradication

**Objective**: Remove the root cause and all traces of the attacker's presence.

#### Root Cause Analysis

**Investigation Steps**:
1. **Timeline Reconstruction**:
   - Analyze logs from 24 hours before first IOC
   - Identify initial compromise vector
   - Map lateral movement and persistence mechanisms

2. **Vulnerability Assessment**:
   ```bash
   # Run comprehensive security scan
   make security-scan

   # Check for additional vulnerabilities
   gosec -exclude-generated ./...
   trivy image goimg-datalayer:latest
   govulncheck ./...

   # Review recent code changes
   git log --since="2025-11-01" --grep="auth\|security\|jwt" --oneline
   ```

3. **Persistence Check**:
   ```bash
   # Search for backdoors in codebase
   grep -r "eval\|exec\|system\|shell_exec" internal/

   # Check for unauthorized user accounts
   psql -h $DB_HOST -U $DB_USER -d goimg <<SQL
   SELECT id, email, username, role, created_at
   FROM users
   WHERE created_at > '2025-12-01'
   AND email NOT LIKE '%@goimg.example.com'
   ORDER BY created_at DESC;
   SQL

   # Look for suspicious cron jobs / background tasks
   docker exec goimg-worker crontab -l
   redis-cli KEYS "goimg:tasks:*" | xargs redis-cli GET
   ```

#### Remediation Actions

**Code Fixes**:
```bash
# Create security patch branch
git checkout -b security-patch/SEC-2025-0042

# Fix vulnerability (example: SQL injection)
# Edit affected file with proper parameterization
vim internal/infrastructure/persistence/postgres/image_repository.go

# Add security regression test
vim tests/security/sql_injection_test.go

# Run full test suite
make test

# Security-specific tests
make test-security

# Commit with issue reference
git commit -m "fix(security): patch SQL injection in image search (SEC-2025-0042)"
```

**Deployment**:
```bash
# Build security patch
make build

# Deploy to staging for validation
make deploy-staging

# Run security validation
make test-e2e-security

# Deploy to production (blue-green)
make deploy-production-bg

# Monitor for 2 hours before finalizing
# If stable, finalize deployment
make finalize-deployment
```

**Infrastructure Cleanup**:
```bash
# Remove attacker persistence mechanisms
docker exec goimg-api rm -f /tmp/suspicious-script.sh

# Rebuild compromised containers from known-good images
docker-compose -f docker/docker-compose.yml pull
docker-compose -f docker/docker-compose.yml up -d --force-recreate

# Verify file integrity
sha256sum /usr/local/bin/goimg-api
# Compare against known-good hash from CI artifact
```

---

### Phase 5: Recovery

**Objective**: Restore services to normal operation with verified fixes in place.

#### Pre-Recovery Checklist

```markdown
- [ ] Vulnerability patched and tested in staging
- [ ] All attacker access revoked (accounts, tokens, keys)
- [ ] Persistence mechanisms removed
- [ ] Security controls validated (auth, authz, rate limiting)
- [ ] Monitoring confirmed operational
- [ ] Rollback plan documented and tested
- [ ] Stakeholder communication prepared
```

#### Recovery Steps

**Graduated Rollout**:
```bash
# 1. Deploy to canary (5% traffic)
kubectl set image deployment/goimg-api goimg-api=goimg-api:v1.2.3-sec-patch
kubectl scale deployment/goimg-api-canary --replicas=1

# Monitor for 30 minutes
# Check error rates, auth success rate, latency
watch -n 10 'curl -s http://prometheus:9090/api/v1/query?query=http_request_errors_total'

# 2. Expand to 25% traffic
kubectl scale deployment/goimg-api-canary --replicas=3

# Monitor for 1 hour

# 3. Full rollout
kubectl set image deployment/goimg-api goimg-api=goimg-api:v1.2.3-sec-patch
kubectl rollout status deployment/goimg-api

# 4. Verify all instances updated
kubectl get pods -l app=goimg-api -o jsonpath='{.items[*].spec.containers[0].image}'
```

**Service Restoration**:
```bash
# Re-enable disabled features
redis-cli DEL "goimg:feature_flags:uploads_disabled"

# Restore normal rate limits
redis-cli SET "goimg:ratelimit:/api/v1/auth/login" "5/minute"

# Remove emergency firewall rules
iptables -D INPUT -s 203.0.113.0/24 -j DROP

# Re-enable external integrations if paused
# (OAuth providers, IPFS gateways, etc.)
```

**User Communication**:
```markdown
Subject: Security Incident Resolution - Action Required

Dear goimg Users,

On December 5, 2025, we identified and resolved a security incident
affecting user authentication. Out of an abundance of caution, we have:

- Patched the vulnerability
- Invalidated all existing sessions
- Required password resets for potentially affected accounts

**Action Required**:
If you receive a password reset notification, please reset your password
immediately using a strong, unique password.

**What Happened**:
[Brief, non-technical description]

**What We Did**:
- Immediately contained the incident
- Patched the vulnerability within 4 hours
- Rotated all security credentials
- Enhanced monitoring

**What You Should Do**:
1. Reset your password if prompted
2. Review recent account activity
3. Enable two-factor authentication (recommended)
4. Report any suspicious activity to security@goimg.example.com

We apologize for any inconvenience and remain committed to protecting your data.

Thank you,
goimg Security Team
```

---

### Phase 6: Post-Incident Review

**Timeline**: Conduct within 5 business days of incident resolution.

#### Post-Mortem Meeting

**Attendees**:
- Incident Commander
- All responders (security, engineering, devops)
- Engineering leadership
- Product management (if user-facing impact)

**Agenda** (90 minutes):
1. **Incident Timeline Review** (15 min)
   - Detection through resolution with timestamps
   - Decisions made and rationale

2. **What Went Well** (15 min)
   - Effective detection mechanisms
   - Successful containment strategies
   - Good communication and coordination

3. **What Went Wrong** (30 min)
   - Detection delays
   - Containment challenges
   - Communication breakdowns
   - Missing tools/documentation

4. **Root Cause Analysis** (15 min)
   - Technical root cause
   - Process gaps
   - Contributing factors

5. **Action Items** (15 min)
   - Immediate fixes (1 week)
   - Short-term improvements (1 month)
   - Long-term hardening (1 quarter)

#### Post-Mortem Report Template

```markdown
# Security Incident Post-Mortem: SEC-2025-0042

## Executive Summary
**Incident**: SQL injection in image search endpoint
**Severity**: P1 (High)
**Duration**: 2025-12-05 14:32 - 2025-12-05 18:45 (4h 13m)
**Impact**: Potential unauthorized access to 247 user records, no confirmed data exfiltration
**Root Cause**: Unsafe string concatenation in dynamic SQL query

## Timeline
| Time (UTC) | Event |
|------------|-------|
| 14:32 | Alert: High authentication failure rate |
| 14:38 | On-call engineer acknowledges, begins investigation |
| 14:45 | SQL injection vulnerability confirmed |
| 14:47 | Containment: Affected endpoint disabled |
| 15:02 | Database credentials rotated |
| 15:30 | Security patch developed |
| 16:15 | Patch deployed to staging, validated |
| 17:00 | Production deployment begins |
| 18:45 | Incident declared resolved |

## Impact Assessment
- **Users Affected**: 247 accounts potentially exposed
- **Data At Risk**: Email addresses, usernames (no passwords exposed)
- **Service Disruption**: Image search disabled for 2 hours
- **Financial Impact**: $0 (no ransomware, no regulatory fines)
- **Reputational Impact**: Low (proactive disclosure, rapid response)

## Root Cause
Unsafe string concatenation in `internal/infrastructure/persistence/postgres/image_repository.go`:

```go
// VULNERABLE CODE (removed)
query := "SELECT * FROM images WHERE tags LIKE '%" + userInput + "%'"
```

**Contributing Factors**:
1. Code review missed unsafe SQL construction
2. SAST tool (gosec) not configured to detect this pattern
3. No security regression test for SQL injection
4. Developer unfamiliar with parameterized query best practices

## What Went Well
- Prometheus alerting detected unusual activity within 6 minutes
- On-call engineer responded within SLA (30 min for P1)
- Containment achieved quickly (endpoint disabled)
- Security patch developed and deployed in 4 hours
- Clear communication to stakeholders throughout

## What Went Wrong
- Vulnerability introduced despite code review process
- SAST configuration gap (gosec should have caught this)
- No automated security tests for SQL injection
- Incident response runbook not fully followed (skipped step 2.4)
- Log retention insufficient for full forensic analysis (only 7 days)

## Lessons Learned
1. **Prevention**: Static analysis must be comprehensive and blocking
2. **Detection**: Need specialized security monitoring beyond availability
3. **Response**: Runbook training needed for all on-call engineers
4. **Forensics**: Extend log retention to 90 days for security events

## Action Items

### Immediate (1 week)
- [ ] #SEC-123: Enable gosec SQL injection detection rules (@security-team)
- [ ] #SEC-124: Add SQL injection regression tests to CI (@test-team)
- [ ] #SEC-125: Conduct security code review training (@engineering)
- [ ] #SEC-126: Update incident response runbook with lessons learned (@security-ops)

### Short-term (1 month)
- [ ] #SEC-127: Implement SAST as blocking CI step (@devops)
- [ ] #SEC-128: Extend security log retention to 90 days (@infrastructure)
- [ ] #SEC-129: Add ORM/query builder to prevent raw SQL (@architecture)
- [ ] #SEC-130: Quarterly security training for all engineers (@hr)

### Long-term (1 quarter)
- [ ] #SEC-131: Implement SIEM with correlation rules (@security-ops)
- [ ] #SEC-132: Automated security regression testing suite (@test-team)
- [ ] #SEC-133: Third-party penetration test (@security-leadership)
- [ ] #SEC-134: Bug bounty program pilot (@security-leadership)

## Appendix

### Evidence Preservation
- Logs: `/forensics/SEC-2025-0042/logs/` (encrypted, 7-year retention)
- Database snapshot: `s3://goimg-forensics/SEC-2025-0042/db-snapshot.sql.gz`
- Network captures: `s3://goimg-forensics/SEC-2025-0042/pcap/`
- Container image: `registry.example.com/forensics/goimg-api:SEC-2025-0042`

### External References
- CVE-2025-XXXXX (assigned post-disclosure)
- OWASP A03:2021 Injection
- CWE-89: SQL Injection

### Disclosure
- **Internal**: All engineering staff (2025-12-05)
- **Users**: Email notification (2025-12-06)
- **Public**: Security advisory published (2025-12-20, 15-day embargo)
```

---

## Communication Templates

### Internal Escalation (Slack/Email)

```markdown
SUBJECT: [P1 SECURITY INCIDENT] SEC-2025-0042 - SQL Injection

INCIDENT ID: SEC-2025-0042
SEVERITY: P1 (High)
STATUS: Containment in progress
INCIDENT COMMANDER: @jane.security

SUMMARY:
SQL injection vulnerability discovered in image search endpoint. Potentially
affecting 247 user accounts. No confirmed data exfiltration.

IMPACT:
- Image search endpoint disabled
- Database credentials rotated
- User sessions invalidated

NEXT STEPS:
- Security patch deploying to staging (ETA: 30 min)
- Production deployment planned for 17:00 UTC
- User notification drafted for leadership approval

WAR ROOM: #incident-sec-2025-0042
STATUS PAGE: https://status.goimg.example.com/incidents/SEC-2025-0042

Updates every 30 minutes or on significant status change.
```

### User Notification (Email)

```markdown
SUBJECT: Important Security Update - Action Required

Dear [User],

We recently identified and resolved a security vulnerability in our image
search feature. While we have no evidence your account was compromised, we
have taken the following precautionary measures:

WHAT WE DID:
- Fixed the vulnerability immediately
- Invalidated all active sessions (you will need to log in again)
- Enhanced our security monitoring

ACTION REQUIRED:
Please reset your password at your earliest convenience:
https://goimg.example.com/reset-password

RECOMMENDED (OPTIONAL):
- Enable two-factor authentication for additional security
- Review your account activity for any suspicious actions

QUESTIONS?
Contact us at security@goimg.example.com

We apologize for any inconvenience and appreciate your understanding.

goimg Security Team
```

### Public Disclosure (Security Advisory)

```markdown
# Security Advisory: SQL Injection in Image Search (CVE-2025-XXXXX)

**Published**: 2025-12-20
**Severity**: High (CVSS 7.5)
**Affected Versions**: goimg-datalayer v1.2.0 - v1.2.2
**Fixed In**: v1.2.3

## Summary
A SQL injection vulnerability in the image search endpoint could allow
authenticated attackers to access unauthorized data.

## Impact
An authenticated attacker could exploit this vulnerability to:
- Access other users' email addresses and usernames
- Enumerate database schema
- Execute arbitrary SQL queries

**No passwords or payment information were exposed.**

## Mitigation
Update to goimg-datalayer v1.2.3 or later:
```
docker pull goimg/goimg-datalayer:v1.2.3
```

## Timeline
- **2025-12-05 14:32 UTC**: Vulnerability discovered via security monitoring
- **2025-12-05 14:45 UTC**: Vulnerability confirmed, containment initiated
- **2025-12-05 18:45 UTC**: Patch deployed, incident resolved
- **2025-12-06**: User notification sent
- **2025-12-20**: Public disclosure (15-day embargo)

## Credit
Internal security team. We thank our monitoring systems and rapid response team.

## References
- CVE-2025-XXXXX
- GitHub Security Advisory: GHSA-xxxx-xxxx-xxxx
- Fix commit: https://github.com/yegamble/goimg-datalayer/commit/abc123

## Contact
security@goimg-datalayer.example.com
```

---

## Incident Response Checklist

### Detection Phase
```markdown
- [ ] Incident ticket created (SEC-YYYY-NNNN)
- [ ] Severity classification assigned (P0/P1/P2/P3)
- [ ] On-call engineer paged (if P0/P1)
- [ ] Evidence preservation started (logs, snapshots)
- [ ] Initial timeline documentation begun
```

### Triage Phase
```markdown
- [ ] Severity confirmed by senior engineer
- [ ] Affected systems identified
- [ ] User impact assessed
- [ ] Attack vector determined
- [ ] IOCs documented
- [ ] Legal/compliance notification assessed
- [ ] Stakeholder communication plan created
```

### Containment Phase
```markdown
- [ ] Attack stopped (firewall rules, service disable)
- [ ] Lateral movement prevented (network isolation)
- [ ] Attacker access revoked (tokens, keys, passwords)
- [ ] Emergency secrets rotation completed
- [ ] Forensic snapshots taken
- [ ] Enhanced monitoring enabled
```

### Eradication Phase
```markdown
- [ ] Root cause identified
- [ ] Vulnerability patched
- [ ] Security regression test added
- [ ] All backdoors/persistence removed
- [ ] Compromised systems rebuilt
- [ ] Code review completed
- [ ] SAST/DAST scans passed
```

### Recovery Phase
```markdown
- [ ] Patch validated in staging
- [ ] Rollback plan documented
- [ ] Graduated rollout completed
- [ ] Services restored
- [ ] User communication sent
- [ ] Monitoring confirmed normal
```

### Post-Incident Phase
```markdown
- [ ] Post-mortem meeting scheduled
- [ ] Post-mortem report completed
- [ ] Action items assigned with deadlines
- [ ] Runbook updated with lessons learned
- [ ] Team training conducted
- [ ] Public disclosure prepared (if applicable)
```

---

## Appendix: Useful Commands

### Log Analysis
```bash
# Search for suspicious authentication attempts
grep "authentication_failed" /var/log/goimg-api/*.log | \
  awk '{print $1, $5}' | sort | uniq -c | sort -rn | head -20

# Find SQL errors indicating injection attempts
grep -i "sql.*error\|syntax.*error" /var/log/goimg-api/*.log

# Track IP addresses with high request rates
awk '{print $1}' /var/log/nginx/access.log | \
  sort | uniq -c | sort -rn | head -50
```

### Database Forensics
```sql
-- Find recently created admin users
SELECT id, email, username, role, created_at
FROM users
WHERE role IN ('admin', 'moderator')
AND created_at > NOW() - INTERVAL '7 days'
ORDER BY created_at DESC;

-- Audit trail for specific user
SELECT * FROM audit_log
WHERE user_id = 'suspicious-user-id'
ORDER BY timestamp DESC
LIMIT 100;

-- Detect mass data exports
SELECT user_id, COUNT(*) as download_count
FROM image_downloads
WHERE downloaded_at > NOW() - INTERVAL '1 hour'
GROUP BY user_id
HAVING COUNT(*) > 100
ORDER BY download_count DESC;
```

### Network Analysis
```bash
# Capture traffic for forensics (10 minutes)
tcpdump -i eth0 -w /forensics/capture-$(date +%Y%m%d-%H%M%S).pcap \
  -G 600 -W 1 'port 443 or port 80'

# Find connections to suspicious IPs
netstat -an | grep ESTABLISHED | grep "203.0.113"

# Monitor real-time connections
watch -n 1 'ss -tn | tail -n +2 | awk "{print \$5}" | \
  cut -d: -f1 | sort | uniq -c | sort -rn | head -20'
```

---

## Document Control

**Version History**:
- v1.0 (2025-12-05): Initial creation for Sprint 9
- Next review: 2025-03-05

**Related Documents**:
- `/home/user/goimg-datalayer/SECURITY.md` - Vulnerability disclosure policy
- `/home/user/goimg-datalayer/docs/security/monitoring.md` - Security monitoring guide
- `/home/user/goimg-datalayer/docs/security/secret_rotation.md` - Secret rotation procedures
- `/home/user/goimg-datalayer/claude/security_gates.md` - Security gate requirements

**Approval**:
- Security Operations Lead: [Pending]
- Engineering Director: [Pending]
- Legal/Compliance: [Pending]
