# Production Security Checklist

Complete this checklist before deploying to production.

## Pre-Deployment Security

### Environment & Secrets

- [ ] All secrets generated with cryptographically secure random data
- [ ] JWT_SECRET is at least 64 characters
- [ ] Database password is strong (32+ characters)
- [ ] Redis password is strong (32+ characters)
- [ ] `.env.prod` file has 600 permissions (not world-readable)
- [ ] No secrets committed to version control
- [ ] `.env.prod` added to `.gitignore`
- [ ] Consider using secrets manager (AWS Secrets Manager, Vault)

### SSL/TLS

- [ ] Valid SSL certificate installed (Let's Encrypt or commercial)
- [ ] Certificate auto-renewal configured
- [ ] TLS 1.2+ only (TLS 1.0/1.1 disabled)
- [ ] Strong cipher suite configured
- [ ] HSTS header enabled with appropriate max-age
- [ ] SSL Labs score A or A+ (test at https://www.ssllabs.com/ssltest/)
- [ ] Certificate expiration monitoring in place

### Database

- [ ] PostgreSQL password changed from default
- [ ] Database accessible only from backend network (not public)
- [ ] SSL mode set to 'require' or 'verify-full'
- [ ] Max connections limit set appropriately
- [ ] Regular automated backups configured
- [ ] Backup encryption enabled (if required)
- [ ] Backup restoration tested
- [ ] Database user has minimal required privileges

### Redis

- [ ] Redis password configured
- [ ] Redis accessible only from backend network (not public)
- [ ] Maxmemory policy configured
- [ ] Persistence (AOF) enabled
- [ ] Protected mode enabled

### Docker Security

- [ ] All containers run as non-root users
- [ ] Read-only root filesystems where possible
- [ ] Security options enabled (no-new-privileges)
- [ ] Capabilities dropped (cap_drop: ALL)
- [ ] Only required capabilities added back
- [ ] Resource limits (CPU, memory) configured
- [ ] Health checks configured for all services
- [ ] Secrets not passed as environment variables (use files/secrets)
- [ ] No privileged containers
- [ ] Docker images scanned for vulnerabilities
- [ ] Base images pinned to specific versions (not 'latest')

### Network Security

- [ ] Firewall enabled (UFW or iptables)
- [ ] Only required ports open (22, 80, 443)
- [ ] Internal network isolated from public
- [ ] Backend services not exposed to internet
- [ ] Rate limiting configured in nginx
- [ ] DDoS protection configured
- [ ] Fail2Ban installed and configured
- [ ] SSH key-based authentication only (password auth disabled)

### Application Security

- [ ] CORS origins restricted to actual frontend domain(s)
- [ ] JWT expiration configured (not too long)
- [ ] File upload size limits configured
- [ ] Allowed file types restricted (whitelist)
- [ ] ClamAV antivirus scanning enabled
- [ ] Image metadata stripping enabled (privacy)
- [ ] Rate limiting enabled on API endpoints
- [ ] Stricter rate limiting on auth endpoints
- [ ] SQL injection protection (parameterized queries)
- [ ] XSS protection headers configured
- [ ] CSRF protection implemented
- [ ] Input validation on all endpoints

### Security Headers

Verify all headers are set in nginx:

- [ ] `Strict-Transport-Security` (HSTS)
- [ ] `X-Frame-Options: SAMEORIGIN`
- [ ] `X-Content-Type-Options: nosniff`
- [ ] `X-XSS-Protection: 1; mode=block`
- [ ] `Referrer-Policy: strict-origin-when-cross-origin`
- [ ] `Permissions-Policy` (restrict features)
- [ ] `Content-Security-Policy` (if applicable)

### Logging & Monitoring

- [ ] Application logging configured (JSON format)
- [ ] Log level appropriate for production (info or warn)
- [ ] Sensitive data not logged (passwords, tokens)
- [ ] Log rotation configured
- [ ] Logs shipped to external service (optional)
- [ ] Error alerting configured
- [ ] Health check monitoring configured
- [ ] Uptime monitoring configured
- [ ] Metrics collection enabled
- [ ] Disk space monitoring

### Access Control

- [ ] Production server accessible only via SSH key
- [ ] SSH password authentication disabled
- [ ] Root SSH login disabled
- [ ] Non-root user with sudo for server management
- [ ] Docker socket not exposed to internet
- [ ] Admin endpoints restricted by IP (if any)
- [ ] Metrics endpoint restricted to internal network
- [ ] Database backups access restricted

### Updates & Patching

- [ ] Automatic security updates enabled
- [ ] Base Docker images updated to latest patches
- [ ] Go version is latest stable
- [ ] All dependencies updated to latest secure versions
- [ ] Update schedule defined
- [ ] Vulnerability scanning in CI/CD

## Post-Deployment Validation

### Automated Tests

```bash
# Run security tests
cd /opt/goimg-datalayer

# Check SSL configuration
sslscan yourdomain.com

# Check security headers
curl -I https://yourdomain.com | grep -E "Strict-Transport|X-Frame|X-Content-Type"

# Test rate limiting
ab -n 1000 -c 100 https://yourdomain.com/api/v1/health
```

### Manual Verification

- [ ] Test login/authentication flow
- [ ] Verify CORS (from frontend domain)
- [ ] Test file upload with malware sample
- [ ] Test file upload with oversized file
- [ ] Test file upload with disallowed type
- [ ] Verify JWT expiration
- [ ] Test invalid authentication
- [ ] Verify rate limiting kicks in
- [ ] Test SQL injection attempts (should fail)
- [ ] Test XSS attempts (should be blocked)

### External Scans

- [ ] SSL Labs test: https://www.ssllabs.com/ssltest/
- [ ] Security Headers test: https://securityheaders.com/
- [ ] Observatory Mozilla: https://observatory.mozilla.org/
- [ ] OWASP ZAP scan (optional)
- [ ] Nmap port scan (verify only 22,80,443 open)

### Penetration Testing (Optional)

- [ ] Hire security firm for pen test
- [ ] Run automated vulnerability scanners
- [ ] Test API endpoints with fuzzing
- [ ] Test authentication bypasses
- [ ] Test authorization bypasses

## Incident Response

- [ ] Incident response plan documented
- [ ] Security contact email configured
- [ ] Backup restoration procedure tested
- [ ] Rollback procedure documented
- [ ] Log retention policy defined
- [ ] Data breach notification procedure

## Compliance (if applicable)

- [ ] GDPR compliance verified
- [ ] PCI-DSS compliance (if handling payments)
- [ ] HIPAA compliance (if handling health data)
- [ ] SOC 2 requirements met
- [ ] Privacy policy updated
- [ ] Terms of service updated

## Regular Maintenance

Schedule these tasks:

- [ ] **Weekly**: Review application logs for anomalies
- [ ] **Weekly**: Check disk space and resource usage
- [ ] **Monthly**: Review firewall rules
- [ ] **Monthly**: Test backup restoration
- [ ] **Monthly**: Update dependencies
- [ ] **Quarterly**: Review access logs
- [ ] **Quarterly**: Security audit
- [ ] **Annually**: Penetration test
- [ ] **Annually**: SSL certificate renewal (if not auto-renewed)

## Emergency Contacts

Document these contacts:

- [ ] On-call engineer: _______________
- [ ] Security team: _______________
- [ ] Infrastructure team: _______________
- [ ] Hosting provider support: _______________
- [ ] Domain registrar: _______________
- [ ] SSL certificate provider: _______________

## Sign-Off

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Developer | | | |
| DevOps Lead | | | |
| Security Officer | | | |
| Technical Lead | | | |

---

**Note:** This checklist should be reviewed and updated regularly as new security best practices emerge.
