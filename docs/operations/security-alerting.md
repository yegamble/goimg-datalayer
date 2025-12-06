# Security Alerting Runbook

> Incident response procedures for security alerts in goimg-datalayer.

**Last Updated**: 2025-12-05
**Owner**: Security Team
**Severity Classification**: P0 (Critical) to P3 (Low)
**On-Call Rotation**: PagerDuty integration enabled for P0/P1 alerts

---

## Table of Contents

1. [Alert Overview](#alert-overview)
2. [Escalation Matrix](#escalation-matrix)
3. [Authentication Failures](#authentication-failures)
4. [Rate Limit Violations](#rate-limit-violations)
5. [Privilege Escalation](#privilege-escalation)
6. [Malware Detection](#malware-detection)
7. [Brute Force Attack](#brute-force-attack)
8. [Account Enumeration](#account-enumeration)
9. [Account Lockout](#account-lockout)
10. [Alert Silencing](#alert-silencing)
11. [Incident Classification](#incident-classification)

---

## Alert Overview

### Alert Severity Levels

| Severity | Response Time | Notification Channels | Example Alerts |
|----------|---------------|----------------------|----------------|
| **Critical (P0)** | Immediate (15 min) | PagerDuty + Slack + Email | Malware Detection |
| **Critical (P1)** | 30 minutes | PagerDuty + Slack + Email | Privilege Escalation |
| **Warning** | 2 hours | Slack + Email | Authentication Failures |
| **Info** | Best effort | Email only | Pattern detection |

### Security Gate Compliance

All alerts implement security gate **S9-MON-001**: All security events must generate alerts.

**Metric Naming Convention**:
- `goimg_security_auth_failures_total` - Authentication failures
- `goimg_security_rate_limit_exceeded_total` - Rate limit violations
- `goimg_security_authorization_denied_total` - Authorization failures
- `goimg_security_malware_detected_total` - Malware detections

---

## Escalation Matrix

### Escalation Path

1. **First Responder**: On-call engineer (PagerDuty)
2. **Escalation Level 1**: Security team lead (after 30 min)
3. **Escalation Level 2**: Engineering manager + CISO (after 1 hour)
4. **Escalation Level 3**: CTO + Legal (data breach confirmed)

### Contact Information

| Role | Contact Method | Response SLA |
|------|----------------|--------------|
| On-Call Engineer | PagerDuty | 15 minutes |
| Security Team | security@goimg.dev | 30 minutes |
| Engineering Manager | eng-manager@goimg.dev | 1 hour |
| CISO | ciso@goimg.dev | 2 hours |

---

## Authentication Failures

**Alert**: `High Authentication Failure Rate`
**Threshold**: >10 failures/minute
**Severity**: Warning
**Priority**: P2

### Symptoms

- Spike in `goimg_security_auth_failures_total` metric
- Multiple failed login attempts from one or more IPs
- Potential brute-force or credential stuffing attack

### Triage Steps

1. **Identify the scope**:
   ```bash
   # Query Prometheus for failure rate by IP
   rate(goimg_security_auth_failures_total{reason="invalid_credentials"}[5m]) by (ip)
   ```

2. **Check Grafana dashboard**:
   - Navigate to [Security Events Dashboard](http://localhost:3000/d/goimg-security-events/security-events)
   - Review "Authentication Failures" panel
   - Check "Top Failed Auth IPs" table

3. **Review application logs**:
   ```bash
   # Check logs for authentication failures
   docker logs goimg-api | grep "event=login_failure"

   # Filter by IP address
   docker logs goimg-api | grep "event=login_failure" | grep "ip=192.168.1.100"
   ```

### Investigation

1. **Determine attack type**:
   - **Targeted**: Failures concentrated on specific accounts → Credential stuffing
   - **Distributed**: Failures across many accounts → Brute force or enumeration
   - **Single IP**: Automated attack from one source
   - **Many IPs**: Distributed attack or botnet

2. **Check account lockout status**:
   ```bash
   # Query Redis for locked accounts
   docker exec goimg-redis redis-cli --scan --pattern "goimg:lockout:*"

   # Get specific account lockout details
   docker exec goimg-redis redis-cli GET "goimg:lockout:<user_id>"
   ```

3. **Review rate limit effectiveness**:
   ```bash
   # Check if rate limiting is working
   docker logs goimg-api | grep "event=rate_limit_exceeded"
   ```

### Remediation

#### Immediate Actions (Within 15 minutes)

1. **Verify account lockout is active**:
   - Check Redis for lockout keys: `goimg:lockout:*`
   - Verify lockout policy: 5 attempts → 15 minute lockout
   - Ensure email notifications are sent to affected users

2. **Identify attacking IPs**:
   ```bash
   # Get top attacking IPs from last hour
   docker logs goimg-api --since 1h | grep "event=login_failure" | \
     awk '{print $6}' | sort | uniq -c | sort -rn | head -20
   ```

3. **Implement IP blocking** (if attack is severe):
   ```bash
   # Add to firewall blocklist (example with iptables)
   sudo iptables -A INPUT -s 192.168.1.100 -j DROP

   # Or add to nginx rate limit zone
   # Edit /etc/nginx/conf.d/rate-limit.conf
   deny 192.168.1.100;
   ```

#### Short-term Actions (Within 1 hour)

1. **Notify affected users**:
   - Send email alerts to users with failed login attempts
   - Recommend password changes if credentials may be compromised
   - Provide instructions for account recovery

2. **Adjust rate limits** (if needed):
   ```go
   // Update rate limit configuration
   // File: internal/interfaces/http/middleware/rate_limit.go

   // Reduce login attempts during attack
   LoginRateLimit: 3 requests per 5 minutes (instead of 5)
   ```

3. **Enable additional logging**:
   - Increase log verbosity for authentication events
   - Capture additional context (User-Agent, Referer, etc.)

#### Long-term Actions (Within 24 hours)

1. **Analyze attack patterns**:
   - Export logs to SIEM for analysis
   - Identify compromised credentials
   - Determine attack source and methodology

2. **Strengthen defenses**:
   - Implement CAPTCHA for login after 2 failures
   - Add device fingerprinting
   - Require 2FA for high-value accounts

3. **Update WAF rules**:
   - Block known attack patterns
   - Add IP reputation checks
   - Implement geo-blocking if appropriate

### False Positive Scenarios

- **Legitimate user forgot password**: Single account, few attempts
- **API client misconfiguration**: Automated retries with wrong credentials
- **Clock skew**: JWT token validation failures due to time drift

**Resolution**: Whitelist known IPs, fix client configuration, sync clocks.

---

## Rate Limit Violations

**Alert**: `High Rate Limit Violation Rate`
**Threshold**: >100 violations/minute globally
**Severity**: Warning
**Priority**: P2

### Symptoms

- Spike in `goimg_security_rate_limit_exceeded_total` metric
- Many 429 (Too Many Requests) responses
- Potential API abuse or DDoS attack

### Triage Steps

1. **Identify affected endpoints**:
   ```bash
   # Query Prometheus for violations by endpoint
   rate(goimg_security_rate_limit_exceeded_total[5m]) by (endpoint, ip)
   ```

2. **Check Grafana dashboard**:
   - Navigate to [Security Events Dashboard](http://localhost:3000/d/goimg-security-events/security-events)
   - Review "Rate Limit Violations" panel
   - Check which endpoints are being hit

3. **Review application logs**:
   ```bash
   # Check logs for rate limit hits
   docker logs goimg-api | grep "event=rate_limit_exceeded"

   # Group by endpoint
   docker logs goimg-api | grep "event=rate_limit_exceeded" | \
     awk '{print $8}' | sort | uniq -c | sort -rn
   ```

### Investigation

1. **Determine if legitimate traffic**:
   - **Legitimate spike**: Product launch, marketing campaign, viral content
   - **Malicious abuse**: Scraping, credential stuffing, resource exhaustion
   - **Misconfigured client**: Automated retry loops, missing backoff

2. **Check traffic source**:
   ```bash
   # Get top IPs hitting rate limits
   docker logs goimg-api --since 30m | grep "event=rate_limit_exceeded" | \
     awk '{print $6}' | sort | uniq -c | sort -rn | head -20
   ```

3. **Analyze request patterns**:
   ```bash
   # Check User-Agent distribution
   docker logs goimg-api | grep "event=rate_limit_exceeded" | \
     grep -oP 'user_agent=\K[^"]*' | sort | uniq -c | sort -rn
   ```

### Remediation

#### Immediate Actions (Within 15 minutes)

1. **Verify rate limits are appropriate**:
   - Global: 100 req/min per IP
   - Authenticated: 300 req/min per user
   - Login: 5 req/min per IP

2. **Identify top offenders**:
   ```bash
   # Get IPs with most rate limit hits
   docker logs goimg-api --since 1h | grep "event=rate_limit_exceeded" | \
     awk '{print $6}' | sort | uniq -c | sort -rn | head -10
   ```

3. **Implement temporary blocking** (if severe):
   ```bash
   # Block abusive IPs at firewall level
   sudo iptables -A INPUT -s 203.0.113.10 -j DROP
   ```

#### Short-term Actions (Within 1 hour)

1. **Contact legitimate high-volume users**:
   - Identify API keys in logs
   - Reach out to explain rate limits
   - Provide guidance on retry backoff strategies

2. **Adjust rate limits** (if spike is legitimate):
   ```go
   // Temporarily increase limits during traffic spike
   // File: internal/interfaces/http/middleware/rate_limit.go

   GlobalRateLimit: 200 requests per minute (temporary)
   ```

3. **Enable adaptive rate limiting**:
   - Implement per-user rate limit tiers
   - Provide higher limits for verified accounts
   - Add burst allowance for short spikes

#### Long-term Actions (Within 24 hours)

1. **Implement tiered rate limits**:
   - Free tier: 100 req/min
   - Verified tier: 500 req/min
   - Premium tier: 2000 req/min

2. **Add retry guidance**:
   - Include `Retry-After` header in 429 responses
   - Document backoff strategies in API docs
   - Provide client SDKs with built-in retry logic

3. **Monitor and tune**:
   - Analyze P95/P99 API usage patterns
   - Adjust limits based on real usage
   - Implement circuit breakers for backend protection

### False Positive Scenarios

- **Traffic spike from legitimate event**: Product launch, viral post
- **Misconfigured automation**: CI/CD, monitoring tools, scrapers
- **Mobile app release**: Thousands of users upgrading simultaneously

**Resolution**: Temporarily increase limits, coordinate with engineering team.

---

## Privilege Escalation

**Alert**: `Privilege Escalation Attempt Detected`
**Threshold**: ANY occurrence
**Severity**: Critical
**Priority**: P1

### Symptoms

- ANY increase in `goimg_security_authorization_denied_total` metric
- User attempting to access resources above their permission level
- Potential account compromise or malicious insider

### Triage Steps

1. **Identify the user and resource**:
   ```bash
   # Query Prometheus for authorization denials
   goimg_security_authorization_denied_total{} by (user_id, resource, required_permission)
   ```

2. **Check Grafana dashboard**:
   - Navigate to [Security Events Dashboard](http://localhost:3000/d/goimg-security-events/security-events)
   - Review "Authorization Failures" panel
   - Check user_id, resource, and required_permission labels

3. **Review application logs**:
   ```bash
   # Check logs for authorization failures
   docker logs goimg-api | grep "event=permission_denied"

   # Filter by user ID
   docker logs goimg-api | grep "event=permission_denied" | grep "user_id=550e8400"
   ```

### Investigation

1. **Determine user context**:
   ```bash
   # Get user details from database
   docker exec goimg-postgres psql -U goimg -c \
     "SELECT id, email, username, role, status FROM users WHERE id = '550e8400-e29b-41d3-a456-426614174000';"
   ```

2. **Review recent user activity**:
   ```bash
   # Check last 24 hours of activity
   docker logs goimg-api --since 24h | grep "user_id=550e8400"

   # Check authentication history
   docker logs goimg-api --since 24h | grep "event=login_success" | grep "user_id=550e8400"
   ```

3. **Check for session hijacking**:
   ```bash
   # Compare IP addresses across sessions
   docker logs goimg-api | grep "user_id=550e8400" | \
     grep -oP 'ip=\K[^"]*' | sort -u

   # Check for unusual User-Agent changes
   docker logs goimg-api | grep "user_id=550e8400" | \
     grep -oP 'user_agent=\K[^"]*' | sort -u
   ```

### Remediation

#### Immediate Actions (Within 15 minutes)

1. **Lock the user account**:
   ```bash
   # Add account to lockout list in Redis
   docker exec goimg-redis redis-cli SET \
     "goimg:lockout:550e8400-e29b-41d3-a456-426614174000" \
     "$(date -d '+1 hour' +%s)" EX 3600
   ```

2. **Revoke all active sessions**:
   ```bash
   # Blacklist all user tokens
   docker exec goimg-redis redis-cli SCAN 0 MATCH "goimg:session:*:550e8400*" | \
     xargs docker exec goimg-redis redis-cli DEL
   ```

3. **Page security team lead**:
   - This is a P1 alert - PagerDuty notification sent automatically
   - Manual escalation: Call security team lead immediately

#### Short-term Actions (Within 30 minutes)

1. **Investigate account compromise**:
   - Check if user's password was recently changed
   - Review 2FA status and recent challenges
   - Check for suspicious login locations

2. **Analyze access attempt**:
   ```bash
   # Get full context of denied request
   docker logs goimg-api | grep "event=permission_denied" | \
     grep "user_id=550e8400" | tail -20
   ```

3. **Determine intent**:
   - **Accidental**: User clicked wrong button, UI bug
   - **Malicious**: Deliberate attempt to access admin functions
   - **Compromised**: Account taken over by attacker

#### Long-term Actions (Within 2 hours)

1. **If legitimate user error**:
   - Document the confusion point
   - Improve UI/UX to prevent future mistakes
   - Unlock account after verification
   - Send email explaining the alert

2. **If account compromised**:
   - Force password reset
   - Invalidate all API keys
   - Review recent actions for data exfiltration
   - Notify user of compromise
   - Implement additional 2FA requirements

3. **If malicious insider**:
   - Escalate to CISO and legal team
   - Preserve audit logs
   - Begin incident response process
   - Consider law enforcement involvement

### False Positive Scenarios

- **UI bug**: Button visible to user but permission check fails
- **Stale permissions cache**: User role changed but cache not updated
- **Development/testing**: Engineer testing RBAC on production (BAD!)

**Resolution**: Fix UI, clear cache, restrict production testing.

---

## Malware Detection

**Alert**: `Malware Detected`
**Threshold**: ANY occurrence
**Severity**: Critical
**Priority**: P0

### Symptoms

- ANY increase in `goimg_security_malware_detected_total` metric
- ClamAV detected malicious content in uploaded file
- Immediate threat to infrastructure and users

### Triage Steps

1. **Identify the malware**:
   ```bash
   # Query Prometheus for malware detections
   goimg_security_malware_detected_total{} by (file_type, threat_name, user_id)
   ```

2. **Check ClamAV logs**:
   ```bash
   # Review ClamAV scan results
   docker logs goimg-clamav | grep "FOUND"

   # Get threat details
   docker logs goimg-clamav | grep -A 5 "FOUND"
   ```

3. **Review application logs**:
   ```bash
   # Check logs for malware detection events
   docker logs goimg-api | grep "event=malware_detected"
   ```

### Investigation

1. **Identify the file and user**:
   ```bash
   # Get file details from logs
   docker logs goimg-api | grep "event=malware_detected" | tail -1

   # Extract user_id and file_id
   docker logs goimg-api | grep "event=malware_detected" | \
     grep -oP 'user_id=\K[^"]*'
   ```

2. **Get user context**:
   ```bash
   # Query user details
   docker exec goimg-postgres psql -U goimg -c \
     "SELECT id, email, username, created_at FROM users WHERE id = '<user_id>';"

   # Check user's upload history
   docker exec goimg-postgres psql -U goimg -c \
     "SELECT COUNT(*) as total_uploads FROM images WHERE user_id = '<user_id>';"
   ```

3. **Check for additional infected files**:
   ```bash
   # Scan all files from this user
   docker exec goimg-api sh -c "clamscan --recursive /uploads/<user_id>/"
   ```

### Remediation

#### Immediate Actions (Within 10 minutes)

1. **Quarantine the file**:
   ```bash
   # Move infected file to quarantine directory
   mkdir -p /quarantine
   mv /uploads/<path-to-file> /quarantine/<timestamp>-<file_id>

   # Update database to mark file as quarantined
   docker exec goimg-postgres psql -U goimg -c \
     "UPDATE images SET status = 'quarantined', quarantined_at = NOW() WHERE id = '<file_id>';"
   ```

2. **Block the user account immediately**:
   ```bash
   # Suspend user account
   docker exec goimg-postgres psql -U goimg -c \
     "UPDATE users SET status = 'suspended', suspended_at = NOW(), suspension_reason = 'Malware upload detected' WHERE id = '<user_id>';"
   ```

3. **Revoke all active sessions**:
   ```bash
   # Blacklist all tokens for this user
   docker exec goimg-redis redis-cli SCAN 0 MATCH "goimg:session:*:<user_id>*" | \
     xargs docker exec goimg-redis redis-cli DEL
   ```

4. **Page security team immediately**:
   - This is a P0 alert - PagerDuty notification sent automatically
   - Manual page: Call on-call security engineer NOW

#### Short-term Actions (Within 30 minutes)

1. **Investigate user account**:
   ```bash
   # Check account creation details
   docker logs goimg-api | grep "event=user_created" | grep "<user_id>"

   # Review all login activity
   docker logs goimg-api | grep "event=login_success" | grep "user_id=<user_id>"

   # Get IP addresses used
   docker logs goimg-api | grep "user_id=<user_id>" | \
     grep -oP 'ip=\K[^"]*' | sort -u
   ```

2. **Check for additional malicious uploads**:
   ```bash
   # Scan all uploads from this user
   docker exec goimg-api sh -c "clamscan --recursive --infected /uploads/ | grep '<user_id>'"

   # Check database for user's files
   docker exec goimg-postgres psql -U goimg -c \
     "SELECT id, filename, created_at FROM images WHERE user_id = '<user_id>' ORDER BY created_at DESC;"
   ```

3. **Analyze malware type**:
   - **EICAR test file**: Harmless test file (likely developer testing)
   - **Trojan/Backdoor**: Serious threat - investigate for C2 communication
   - **Ransomware**: Critical threat - check for encryption attempts
   - **PUP/Adware**: Low risk but policy violation

4. **Determine attack vector**:
   - **Compromised account**: Legitimate user account taken over
   - **Malicious user**: Deliberate upload of malware
   - **Innocent user**: User unknowingly uploaded infected file

#### Long-term Actions (Within 2 hours)

1. **If innocent user with infected file**:
   - Notify user of infection
   - Provide guidance on cleaning their device
   - Scan all their uploads
   - Unlock account after verification and re-upload clean files

2. **If compromised account**:
   - Force password reset
   - Enable mandatory 2FA
   - Review account recovery options
   - Check for data exfiltration
   - Restore account after security verification

3. **If malicious user**:
   - Permanently ban account and email
   - Report to authorities if appropriate
   - Block IP address ranges
   - Review other accounts from same IP/device

4. **System-wide actions**:
   - Update ClamAV signatures: `docker exec goimg-clamav freshclam`
   - Scan all recent uploads (last 24 hours)
   - Review file upload validation logic
   - Consider additional file scanning (YARA rules, sandboxing)

### Malware Threat Classifications

| Threat Type | Severity | Action | Escalation |
|-------------|----------|--------|------------|
| EICAR test | Low | Log, quarantine | None |
| PUP/Adware | Low | Quarantine, notify user | None |
| Trojan/Backdoor | Critical | Quarantine, ban user, scan all uploads | CISO |
| Ransomware | Critical | Immediate lockdown, investigate C2 | CISO + Legal |
| APT malware | Critical | Full incident response | CISO + FBI |

### False Positive Scenarios

- **EICAR test file**: Developers testing malware detection
- **Benign file misidentified**: False positive from ClamAV
- **Encrypted archive**: Password-protected zip flagged as suspicious

**Resolution**: Verify with VirusTotal, manual analysis, update ClamAV signatures.

---

## Brute Force Attack

**Alert**: `Potential Brute Force Attack`
**Threshold**: >50 failed login attempts from single IP in 10 minutes
**Severity**: High
**Priority**: P2

### Symptoms

- Pattern of many failed authentication attempts from single IP
- Spike in `goimg_security_auth_failures_total{reason="invalid_credentials"}` metric
- Potential credential guessing attack

### Triage Steps

1. **Identify the attacking IP**:
   ```bash
   # Query Prometheus for top IPs with auth failures
   topk(10, sum by (ip) (increase(goimg_security_auth_failures_total{reason="invalid_credentials"}[10m])))
   ```

2. **Check attack scope**:
   ```bash
   # Get total attempts from this IP
   docker logs goimg-api --since 30m | grep "event=login_failure" | \
     grep "ip=203.0.113.10" | wc -l

   # Check if multiple accounts targeted
   docker logs goimg-api --since 30m | grep "event=login_failure" | \
     grep "ip=203.0.113.10" | grep -oP 'email_hash=\K[^"]*' | sort -u | wc -l
   ```

### Investigation

1. **Determine attack characteristics**:
   - **Targeted**: Same email attempted repeatedly (credential guessing)
   - **Spray**: Different emails attempted (password spraying)
   - **Speed**: Request rate (slow: evasion, fast: automated)

2. **Check IP reputation**:
   ```bash
   # Check AbuseIPDB
   curl "https://api.abuseipdb.com/api/v2/check?ipAddress=203.0.113.10" \
     -H "Key: YOUR_API_KEY"

   # Check VirusTotal
   curl "https://www.virustotal.com/api/v3/ip_addresses/203.0.113.10" \
     -H "x-apikey: YOUR_API_KEY"
   ```

### Remediation

#### Immediate Actions (Within 15 minutes)

1. **Block the IP address**:
   ```bash
   # Add to firewall blocklist
   sudo iptables -A INPUT -s 203.0.113.10 -j DROP

   # Or add to nginx rate limit deny list
   echo "deny 203.0.113.10;" >> /etc/nginx/conf.d/blocklist.conf
   sudo nginx -s reload
   ```

2. **Verify account lockout is working**:
   ```bash
   # Check for locked accounts
   docker exec goimg-redis redis-cli --scan --pattern "goimg:lockout:*" | wc -l
   ```

3. **Notify affected users** (if targeted attack):
   - Send email alert about suspicious activity
   - Recommend password change
   - Enable 2FA

#### Short-term Actions (Within 1 hour)

1. **Add to permanent blocklist**:
   ```bash
   # Add to application-level blocklist in Redis
   docker exec goimg-redis redis-cli SADD "goimg:blocklist:ips" "203.0.113.10"
   ```

2. **Monitor for distributed attack**:
   ```bash
   # Check if attack is coming from multiple IPs
   docker logs goimg-api --since 1h | grep "event=login_failure" | \
     awk '{print $6}' | sort | uniq -c | sort -rn | head -20
   ```

3. **Adjust rate limits temporarily**:
   ```go
   // Reduce login rate limit during attack
   LoginRateLimit: 2 requests per 10 minutes
   ```

#### Long-term Actions (Within 24 hours)

1. **Implement CAPTCHA**:
   - Add reCAPTCHA to login form after 2 failures
   - Use hCaptcha for privacy-focused alternative

2. **Add IP reputation checking**:
   - Integrate with AbuseIPDB, IPQualityScore
   - Block known VPN/proxy IPs (if appropriate)
   - Implement risk scoring

3. **Monitor and adjust**:
   - Review effectiveness of new controls
   - Analyze attack patterns for future prevention
   - Update WAF rules

---

## Account Enumeration

**Alert**: `Potential Account Enumeration`
**Threshold**: >200 authentication attempts from single IP in 1 hour
**Severity**: Warning
**Priority**: P3

### Symptoms

- High volume of authentication attempts across many different emails
- Attacker trying to identify valid email addresses
- Lower priority than brute force (information gathering phase)

### Triage Steps

1. **Identify enumeration pattern**:
   ```bash
   # Get unique emails attempted from this IP
   docker logs goimg-api --since 1h | grep "event=login_failure" | \
     grep "ip=203.0.113.10" | grep -oP 'email_hash=\K[^"]*' | sort -u | wc -l
   ```

2. **Check if responses reveal account existence**:
   ```bash
   # Verify we're using generic error messages
   docker logs goimg-api | grep "event=login_failure" | \
     grep -oP 'response=\K[^"]*' | sort -u

   # Should ONLY see: "Invalid email or password"
   ```

### Investigation

1. **Determine data source**:
   - **Dictionary attack**: Common email patterns (john@, admin@, etc.)
   - **Leaked database**: Emails from data breach
   - **Social engineering**: Emails scraped from social media

2. **Check timing patterns**:
   ```bash
   # Plot attempts over time to see if automated
   docker logs goimg-api --since 2h | grep "event=login_failure" | \
     grep "ip=203.0.113.10" | awk '{print $1}' | uniq -c
   ```

### Remediation

#### Immediate Actions (Within 2 hours)

1. **Block the IP address**:
   ```bash
   sudo iptables -A INPUT -s 203.0.113.10 -j DROP
   ```

2. **Verify constant-time responses**:
   - Ensure authentication failures have same timing regardless of email validity
   - Check that we hash password even for non-existent accounts

3. **Enable CAPTCHA** (if not already):
   - Add CAPTCHA to login form after 1 failure for this IP range

#### Short-term Actions (Within 24 hours)

1. **Review error messages**:
   ```go
   // Verify generic error messages are used
   // BAD: "Email not found" vs "Password incorrect"
   // GOOD: "Invalid email or password" for both cases
   ```

2. **Implement honeypot accounts**:
   - Create fake accounts with common names
   - Alert on any login attempts to honeypot accounts
   - Automatic IP ban on honeypot hit

3. **Add rate limiting by email hash**:
   ```go
   // Limit attempts per email (even if email doesn't exist)
   RateLimitByEmailHash: 5 attempts per hour
   ```

#### Long-term Actions (Within 1 week)

1. **Implement HIBP integration**:
   - Check passwords against Have I Been Pwned database
   - Reject compromised passwords at registration
   - Force password resets for compromised accounts

2. **Add behavioral analysis**:
   - Track login patterns (time of day, location, device)
   - Alert on anomalous login attempts
   - Implement risk-based authentication

3. **Educate users**:
   - Security awareness training
   - Password manager recommendations
   - 2FA adoption campaign

---

## Account Lockout

**Alert**: `Account Lockout Triggered`
**Threshold**: ANY occurrence
**Severity**: High
**Priority**: P2

### Symptoms

- ANY increase in `goimg_security_auth_failures_total{reason="account_locked"}` metric
- User accounts automatically locked after exceeding failed login threshold
- Indicates ongoing brute-force attack or compromised credentials

### Triage Steps

1. **Identify locked accounts**:
   ```bash
   # Query Prometheus for lockout events
   increase(goimg_security_auth_failures_total{reason="account_locked"}[5m])

   # Get locked accounts from Redis
   docker exec goimg-redis redis-cli --scan --pattern "goimg:lockout:*"
   ```

2. **Check Grafana dashboard**:
   - Navigate to [Security Events Dashboard](http://localhost:3000/d/goimg-security-events/security-events)
   - Review "Authentication Failures" panel
   - Filter by reason="account_locked"

3. **Review application logs**:
   ```bash
   # Check logs for lockout events
   docker logs goimg-api | grep "event=account_lockout"

   # Get user IDs and source IPs
   docker logs goimg-api | grep "event=account_lockout" | \
     grep -oP 'user_id=\K[^"]*|ip=\K[^"]*'
   ```

### Investigation

1. **Determine attack scope**:
   ```bash
   # Count number of locked accounts
   docker exec goimg-redis redis-cli --scan --pattern "goimg:lockout:*" | wc -l

   # Get lockout details for specific account
   docker exec goimg-redis redis-cli GET "goimg:lockout:<user_id>"
   docker exec goimg-redis redis-cli GET "goimg:failed_attempts:<user_id>"
   ```

2. **Identify attack source**:
   ```bash
   # Get IPs that triggered lockouts
   docker logs goimg-api --since 1h | grep "event=account_lockout" | \
     grep -oP 'ip=\K[^"]*' | sort | uniq -c | sort -rn

   # Check if single IP or distributed attack
   docker logs goimg-api --since 1h | grep "event=login_failure" | \
     grep -oP 'ip=\K[^"]*' | sort -u | wc -l
   ```

3. **Check affected users**:
   ```bash
   # Get user details for locked accounts
   docker exec goimg-postgres psql -U goimg -c \
     "SELECT id, email, username, created_at FROM users WHERE id IN (
       SELECT substring(key FROM 16) FROM redis_keys WHERE key LIKE 'goimg:lockout:%'
     );"
   ```

### Remediation

#### Immediate Actions (Within 15 minutes)

1. **Verify lockout mechanism is working**:
   ```bash
   # Check lockout policy settings
   # Expected: 5 failed attempts → 15 minute lockout

   # Verify Redis TTL on lockout keys
   docker exec goimg-redis redis-cli --scan --pattern "goimg:lockout:*" | \
     xargs -I {} docker exec goimg-redis redis-cli TTL {}
   ```

2. **Identify attack pattern**:
   - **Targeted**: Few specific accounts locked → Credential stuffing
   - **Mass lockout**: Many accounts locked → Distributed brute force or DoS
   - **High-value accounts**: Admin/moderator accounts → Targeted attack

3. **Block attacking IPs** (if concentrated attack):
   ```bash
   # Get top 10 IPs causing lockouts
   docker logs goimg-api --since 1h | grep "event=login_failure" | \
     awk '{print $6}' | sort | uniq -c | sort -rn | head -10

   # Block malicious IPs
   sudo iptables -A INPUT -s 203.0.113.10 -j DROP
   ```

#### Short-term Actions (Within 1 hour)

1. **Notify affected users**:
   - Send email notification about lockout
   - Explain lockout duration (15 minutes)
   - Provide guidance on securing account if compromise suspected
   - Include password reset link if needed

   ```bash
   # Example email content:
   # Subject: Security Alert - Account Temporarily Locked
   # Body:
   #   Your account has been temporarily locked due to multiple failed
   #   login attempts. This is a security measure to protect your account.
   #
   #   - Lockout Duration: 15 minutes
   #   - Time Remaining: Check dashboard
   #   - If this wasn't you: Reset password immediately
   #   - Source IP: XXX.XXX.XXX.XXX
   ```

2. **Analyze attack effectiveness**:
   ```bash
   # Check if any accounts were compromised before lockout
   docker logs goimg-api --since 2h | grep "event=login_success" | \
     grep -f <(docker logs goimg-api --since 2h | grep "event=account_lockout" | \
       grep -oP 'user_id=\K[^"]*')

   # Review sessions created around lockout time
   docker exec goimg-postgres psql -U goimg -c \
     "SELECT user_id, created_at, ip, user_agent FROM sessions
      WHERE created_at > NOW() - INTERVAL '2 hours'
      AND user_id IN (
        SELECT substring(key FROM 16) FROM redis_keys WHERE key LIKE 'goimg:lockout:%'
      );"
   ```

3. **Adjust security controls** (if needed):
   ```go
   // Consider tightening lockout policy during active attack
   // File: internal/infrastructure/security/account_lockout.go

   // Temporary during attack (reduce from 5 to 3 attempts)
   MaxFailedAttempts: 3
   LockoutDuration: 30 * time.Minute  // Increase from 15 to 30 min
   ```

#### Long-term Actions (Within 24 hours)

1. **If mass lockout attack (DoS attempt)**:
   - Implement CAPTCHA after 2 failed attempts
   - Add progressive delays between login attempts
   - Enable IP reputation checking
   - Consider temporary account lockout bypass for trusted IPs
   - Coordinate with infrastructure team for DDoS mitigation

2. **If credential stuffing attack**:
   - Force password reset for affected accounts
   - Check if passwords are in HIBP database
   - Implement breach password detection at registration
   - Enable mandatory 2FA for high-value accounts
   - Review authentication logs for successfully compromised accounts

3. **If targeted attack on high-value accounts**:
   - Escalate to security team immediately
   - Force password reset for targeted accounts
   - Enable mandatory 2FA
   - Review account activity for unauthorized access
   - Check for data exfiltration
   - Consider temporary account suspension pending investigation

4. **System improvements**:
   - Add honeypot accounts (fake admin accounts to detect attacks)
   - Implement adaptive lockout (longer duration for repeated lockouts)
   - Add device fingerprinting for anomaly detection
   - Enable location-based access controls
   - Implement risk-based authentication

### False Positive Scenarios

- **Legitimate user forgot password**: Single account, manual attempts
  - **Resolution**: User can wait 15 minutes or use password reset

- **API client misconfiguration**: Automated retries with wrong credentials
  - **Resolution**: Contact user, fix API client configuration, whitelist IP temporarily

- **Password manager sync issue**: User's password manager out of sync
  - **Resolution**: User updates password manager, wait for lockout to expire

- **Mobile app bug**: App sending wrong credentials repeatedly
  - **Resolution**: Fix app bug, temporarily increase lockout threshold

### Escalation Criteria

Escalate to security team lead if:
- More than 50 accounts locked in 1 hour (mass attack)
- Admin or moderator accounts targeted (high-value targets)
- Lockouts persist after IP blocking (distributed attack)
- Successful logins detected before lockout (accounts compromised)
- Attack coincides with other security events (coordinated attack)

### Metrics to Monitor

- Total lockouts per hour
- Unique source IPs causing lockouts
- Ratio of lockouts to total login attempts
- Time to lockout (should be consistent ~5 attempts)
- Lockout duration effectiveness (attack stops after lockout)

### Success Criteria

- Attack mitigated within 1 hour
- No accounts compromised during attack
- Legitimate users minimally impacted
- Attacking IPs blocked
- Users notified and accounts secured

---

## Alert Silencing

### When to Silence Alerts

Alerts should be silenced during:
1. **Planned maintenance windows**
2. **Known traffic events** (product launches, marketing campaigns)
3. **Testing in production** (with approval and limited scope)
4. **False positives** (after investigation confirms no threat)

### How to Silence Alerts

#### Via Grafana UI

1. Navigate to Alerting → Silences
2. Click "New Silence"
3. Add matchers:
   ```
   alertname = High Authentication Failure Rate
   severity = warning
   ```
4. Set duration (e.g., 1 hour)
5. Add comment explaining reason
6. Click "Create"

#### Via API

```bash
# Create silence for 1 hour
curl -X POST http://localhost:3000/api/alertmanager/grafana/api/v2/silences \
  -H "Content-Type: application/json" \
  -u admin:admin \
  -d '{
    "matchers": [
      {"name": "alertname", "value": "High Authentication Failure Rate", "isRegex": false}
    ],
    "startsAt": "2024-11-10T10:00:00Z",
    "endsAt": "2024-11-10T11:00:00Z",
    "createdBy": "ops-team",
    "comment": "Planned load testing"
  }'
```

### Silence Best Practices

1. **Always add a comment** explaining why alert was silenced
2. **Set expiration time** - never silence indefinitely
3. **Use specific matchers** - don't silence all security alerts
4. **Document in runbook** - add entry to incident log
5. **Review regularly** - audit silenced alerts weekly

---

## Incident Classification

### Incident Severity Matrix

| Incident Type | Data Breach? | System Compromise? | Severity | Response Time |
|---------------|--------------|-------------------|----------|---------------|
| Malware upload (single) | No | No | P1 | 30 min |
| Malware upload (multiple) | Possible | Possible | P0 | 15 min |
| Brute force (blocked) | No | No | P2 | 2 hours |
| Brute force (account compromised) | Yes | No | P0 | 15 min |
| Privilege escalation (failed) | No | No | P1 | 30 min |
| Privilege escalation (successful) | Yes | Yes | P0 | Immediate |
| Account enumeration | No | No | P3 | Best effort |
| Rate limit abuse | No | No | P3 | Best effort |

### Incident Response Phases

1. **Detection**: Alert fired, security team notified
2. **Triage**: Determine severity and scope (15-30 min)
3. **Containment**: Stop the attack, block IPs, lock accounts (30 min - 2 hours)
4. **Investigation**: Analyze logs, determine root cause (2-24 hours)
5. **Eradication**: Remove threat, patch vulnerabilities (24-72 hours)
6. **Recovery**: Restore normal operations, unlock accounts (72 hours - 1 week)
7. **Lessons Learned**: Post-mortem, update runbooks (1-2 weeks)

### Data Breach Notification

If incident involves data breach (unauthorized access to user data):

1. **Notify CISO immediately** (within 1 hour of confirmation)
2. **Document evidence**: Preserve logs, snapshots, forensics
3. **Legal review**: Consult legal team for notification requirements
4. **Regulatory compliance**:
   - **GDPR**: Notify within 72 hours if EU users affected
   - **CCPA**: Notify within "reasonable timeframe" if CA users affected
   - **State laws**: Vary by state, consult legal team
5. **User notification**: Email affected users with details and remediation steps
6. **Public disclosure**: If significant breach, prepare public statement

### Post-Incident Review

After every P0/P1 incident:

1. **Schedule post-mortem** within 3 business days
2. **Document timeline** of events
3. **Identify root cause** (5 Whys analysis)
4. **List action items** with owners and due dates
5. **Update runbooks** and alerting rules
6. **Share learnings** with engineering team

---

## Additional Resources

- [Grafana Alerting Documentation](https://grafana.com/docs/grafana/latest/alerting/)
- [Prometheus Query Examples](https://prometheus.io/docs/prometheus/latest/querying/examples/)
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [NIST Incident Handling Guide](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-61r2.pdf)
- [Security Events Dashboard](http://localhost:3000/d/goimg-security-events/security-events)

---

**Document Version**: 1.0
**Last Review**: 2025-12-05
**Next Review**: 2026-01-05
**Owner**: Security Team (security@goimg.dev)
