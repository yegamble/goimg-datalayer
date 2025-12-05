# SSL/TLS Certificate Setup Guide

This guide walks you through setting up SSL/TLS certificates for the goimg-datalayer production deployment using Let's Encrypt and Certbot.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Step-by-Step Setup](#step-by-step-setup)
  - [HTTP-01 Challenge](#method-1-http-01-challenge-recommended)
  - [DNS-01 Challenge](#method-2-dns-01-challenge-for-wildcards)
- [Certificate Renewal](#certificate-renewal)
- [Verification](#verification)
- [Troubleshooting](#troubleshooting)
- [Security Best Practices](#security-best-practices)

## Prerequisites

Before setting up SSL certificates, ensure you have:

### 1. Domain Name

- A registered domain name (e.g., `example.com`)
- DNS access to create/modify DNS records

### 2. DNS Configuration

Your domain must be properly configured to point to your server:

```bash
# Verify DNS resolution
dig +short example.com
nslookup example.com

# Should return your server's public IP address
```

**Required DNS Records:**

| Type | Name | Value | TTL |
|------|------|-------|-----|
| A | example.com | YOUR_SERVER_IP | 300 |
| A | www.example.com | YOUR_SERVER_IP | 300 |

### 3. Server Requirements

- Ubuntu 20.04+ or Debian 11+ (recommended)
- Root or sudo access
- Ports 80 and 443 open in firewall
- Docker and Docker Compose installed

### 4. Firewall Configuration

Ensure ports 80 and 443 are accessible:

```bash
# UFW (Ubuntu)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw reload

# iptables
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
sudo iptables-save > /etc/iptables/rules.v4

# Verify
sudo netstat -tlnp | grep -E ':(80|443)'
```

### 5. Email Address

Valid email address for Let's Encrypt notifications (certificate expiration alerts).

## Quick Start

For most users, the HTTP-01 challenge is the simplest method:

```bash
# 1. Navigate to project directory
cd /home/user/goimg-datalayer

# 2. Start the application stack (without nginx initially)
docker-compose -f docker/docker-compose.prod.yml up -d api postgres redis clamav ipfs

# 3. Wait for services to be healthy
docker-compose -f docker/docker-compose.prod.yml ps

# 4. Run SSL setup script
sudo scripts/setup-ssl.sh \
  --obtain \
  --domain example.com \
  --email admin@example.com

# 5. Start nginx with SSL
docker-compose -f docker/docker-compose.prod.yml up -d nginx

# 6. Verify SSL is working
curl -I https://example.com/health

# 7. Setup automatic renewal
sudo scripts/setup-ssl.sh --setup-renewal
```

## Step-by-Step Setup

### Method 1: HTTP-01 Challenge (Recommended)

The HTTP-01 challenge verifies domain ownership by placing a file on your web server.

**Advantages:**
- Simple setup, no DNS provider configuration needed
- Works for single domains and www subdomains
- Fast verification (seconds)

**Limitations:**
- Requires port 80 to be accessible
- Cannot obtain wildcard certificates
- Must be run on the web server itself

#### Steps:

1. **Update domain in nginx configuration:**

```bash
# Replace placeholder domain with your actual domain
sed -i 's/example\.com/yourdomain.com/g' docker/nginx/conf.d/api.conf
```

2. **Create webroot directory:**

```bash
sudo mkdir -p /var/www/certbot
sudo chown -R $USER:$USER /var/www/certbot
```

3. **Start application services (without nginx):**

```bash
docker-compose -f docker/docker-compose.prod.yml up -d \
  api postgres redis clamav ipfs worker
```

4. **Start nginx temporarily on port 80 only:**

Create a temporary nginx config for certificate acquisition:

```bash
# Create temporary nginx container for HTTP-01 challenge
docker run -d --name nginx-temp \
  -p 80:80 \
  -v /var/www/certbot:/var/www/certbot:ro \
  -v $(pwd)/docker/nginx/conf.d/api.conf:/etc/nginx/conf.d/default.conf:ro \
  nginx:1.25-alpine
```

5. **Obtain SSL certificate:**

```bash
sudo scripts/setup-ssl.sh \
  --obtain \
  --domain yourdomain.com \
  --email admin@yourdomain.com \
  --webroot /var/www/certbot
```

The script will:
- Validate your domain and email
- Check DNS resolution
- Verify port 80 accessibility
- Request certificate from Let's Encrypt
- Copy certificates to `docker/nginx/ssl/`
- Update nginx configuration

6. **Stop temporary nginx and start production nginx:**

```bash
# Stop temporary nginx
docker stop nginx-temp
docker rm nginx-temp

# Start production nginx with SSL
docker-compose -f docker/docker-compose.prod.yml up -d nginx
```

7. **Verify SSL is working:**

```bash
# Test HTTPS endpoint
curl -I https://yourdomain.com/health

# Check certificate details
openssl s_client -connect yourdomain.com:443 -servername yourdomain.com < /dev/null
```

### Method 2: DNS-01 Challenge (For Wildcards)

The DNS-01 challenge verifies domain ownership via DNS TXT records.

**Advantages:**
- Can obtain wildcard certificates (`*.example.com`)
- Works without requiring port 80/443 access
- Can be run from any machine with DNS API access

**Limitations:**
- Requires DNS provider API credentials
- Slightly more complex setup
- Slower verification (DNS propagation can take minutes)

**Supported DNS Providers:**
- Cloudflare
- Amazon Route53
- Google Cloud DNS
- DigitalOcean
- Many others (see [Certbot DNS plugins](https://eff-certbot.readthedocs.io/en/stable/using.html#dns-plugins))

#### Steps (Cloudflare Example):

1. **Install Cloudflare DNS plugin:**

```bash
sudo apt-get update
sudo apt-get install python3-certbot-dns-cloudflare
```

2. **Create Cloudflare API credentials file:**

```bash
sudo mkdir -p /etc/letsencrypt/cloudflare
sudo nano /etc/letsencrypt/cloudflare/credentials.ini
```

Add your Cloudflare API token:

```ini
# Cloudflare API credentials
dns_cloudflare_api_token = YOUR_CLOUDFLARE_API_TOKEN
```

**How to get Cloudflare API token:**
1. Log in to Cloudflare dashboard
2. Go to My Profile > API Tokens
3. Create Token > Use template "Edit zone DNS"
4. Select your domain zone
5. Copy the generated token

Set proper permissions:

```bash
sudo chmod 600 /etc/letsencrypt/cloudflare/credentials.ini
```

3. **Obtain wildcard certificate:**

```bash
sudo scripts/setup-ssl.sh \
  --obtain \
  --domain yourdomain.com \
  --email admin@yourdomain.com \
  --dns cloudflare
```

This will obtain a certificate valid for:
- `yourdomain.com`
- `*.yourdomain.com` (wildcard)

4. **Start nginx with SSL:**

```bash
docker-compose -f docker/docker-compose.prod.yml up -d nginx
```

## Certificate Renewal

Let's Encrypt certificates are valid for 90 days. Automatic renewal is configured to run twice daily.

### Automatic Renewal

Setup automatic renewal (one-time):

```bash
sudo scripts/setup-ssl.sh --setup-renewal
```

This configures either:
- **Systemd timer** (preferred on systemd systems)
- **Cron job** (fallback on older systems)

**Systemd Timer:**
- Runs at 00:00 and 12:00 daily
- Random delay up to 1 hour to spread load
- View status: `systemctl status certbot-renewal.timer`
- View logs: `journalctl -u certbot-renewal.service`

**Cron Job:**
- Runs at 00:00 and 12:00 daily
- View crontab: `sudo crontab -l`
- Logs to: `/home/user/goimg-datalayer/logs/ssl-setup.log`

### Manual Renewal

Renew certificates manually:

```bash
# Dry run (test renewal without making changes)
sudo scripts/setup-ssl.sh --renew --dry-run

# Actual renewal
sudo scripts/setup-ssl.sh --renew
```

Certbot automatically:
- Checks if renewal is needed (< 30 days remaining)
- Requests new certificate
- Copies to nginx SSL directory
- Reloads nginx

### Check Certificate Validity

Check when your certificate expires:

```bash
sudo scripts/setup-ssl.sh --check-validity
```

Output example:

```
[INFO] Checking certificate validity...
[INFO] Certificate: /home/user/goimg-datalayer/docker/nginx/ssl/fullchain.pem
[INFO] Expires: Mar 15 12:00:00 2025 GMT
[INFO] Days remaining: 73
[INFO] Subject: CN=example.com
[INFO] Issuer: C=US, O=Let's Encrypt, CN=R3
[INFO] Certificate is valid and does not require renewal yet
```

### Post-Renewal Process

When certificates are renewed, the script automatically:

1. Copies new certificates to `docker/nginx/ssl/`
2. Sets proper file permissions
3. Reloads nginx (Docker exec or systemctl)

No manual intervention required.

## Verification

### 1. Check Certificate Files

Verify certificate files exist:

```bash
ls -lh docker/nginx/ssl/

# Should show:
# -rw-r--r-- cert.pem         (certificate only)
# -rw-r--r-- chain.pem        (intermediate certificates)
# -rw-r--r-- fullchain.pem    (cert + chain)
# -rw------- privkey.pem      (private key)
```

### 2. Test HTTPS Connection

```bash
# Test with curl
curl -I https://yourdomain.com/health

# Expected output:
# HTTP/2 200
# strict-transport-security: max-age=31536000; includeSubDomains; preload
# x-content-type-options: nosniff
# x-frame-options: SAMEORIGIN
# ...
```

### 3. Verify SSL Configuration

```bash
# Test SSL handshake
openssl s_client -connect yourdomain.com:443 -servername yourdomain.com < /dev/null

# Check for:
# - Protocol: TLSv1.2 or TLSv1.3
# - Cipher: Strong cipher (ECDHE-RSA-AES128-GCM-SHA256 or better)
# - Verify return code: 0 (ok)
```

### 4. Check Security Headers

```bash
curl -I https://yourdomain.com/health | grep -i "strict-transport-security\|x-content-type\|x-frame-options\|content-security-policy"
```

Expected headers:

```
strict-transport-security: max-age=31536000; includeSubDomains; preload
x-content-type-options: nosniff
x-frame-options: SAMEORIGIN
content-security-policy: default-src 'self'; ...
```

### 5. SSL Labs Test (Comprehensive)

Test your SSL configuration for an A+ rating:

```bash
# Visit this URL (replace example.com with your domain)
https://www.ssllabs.com/ssltest/analyze.html?d=yourdomain.com
```

**Expected SSL Labs Grade: A+**

To achieve A+:
- TLS 1.2+ only ✓
- Strong cipher suites ✓
- HSTS enabled ✓
- No protocol vulnerabilities ✓
- Forward secrecy enabled ✓

### 6. Test HTTP to HTTPS Redirect

```bash
# Should redirect to HTTPS
curl -I http://yourdomain.com/health

# Expected:
# HTTP/1.1 301 Moved Permanently
# Location: https://yourdomain.com/health
```

### 7. Verify in Browser

Visit https://yourdomain.com in a web browser:

1. Check for padlock icon in address bar
2. Click padlock > Certificate > Details
3. Verify:
   - Issued by: Let's Encrypt Authority X3
   - Valid from/to dates
   - Subject Alternative Names (SAN) includes your domain

## Troubleshooting

### Certificate Acquisition Failed

**Error: "Failed to obtain certificate"**

**Solutions:**

1. **Check DNS resolution:**

```bash
dig +short yourdomain.com

# Must return your server's public IP
# If not, update DNS A record and wait for propagation
```

2. **Verify port 80 is accessible:**

```bash
# From another machine
curl http://YOUR_SERVER_IP/.well-known/acme-challenge/test

# Should return nginx 404, not connection refused
```

3. **Check nginx is running:**

```bash
docker ps | grep nginx
# Should show goimg-nginx running
```

4. **Review certbot logs:**

```bash
sudo tail -f /var/log/letsencrypt/letsencrypt.log
```

5. **Test with staging environment:**

```bash
# Use Let's Encrypt staging to avoid rate limits during testing
sudo scripts/setup-ssl.sh \
  --obtain \
  --domain yourdomain.com \
  --email admin@yourdomain.com \
  --staging
```

### Rate Limit Exceeded

**Error: "too many certificates already issued"**

Let's Encrypt has rate limits:
- 50 certificates per registered domain per week
- 5 duplicate certificates per week

**Solutions:**

1. **Wait one week** for rate limit to reset
2. **Use staging environment** for testing:

```bash
sudo scripts/setup-ssl.sh --obtain --staging --domain yourdomain.com --email admin@yourdomain.com
```

3. **Check rate limits:**

Visit https://crt.sh/?q=yourdomain.com to see issued certificates.

### Certificate Not Trusted in Browser

**Error: "Your connection is not private" or "NET::ERR_CERT_AUTHORITY_INVALID"**

**Causes:**

1. **Using staging certificate** (not trusted by browsers)
   - Solution: Obtain production certificate (remove `--staging` flag)

2. **Wrong certificate file** (using `cert.pem` instead of `fullchain.pem`)
   - Solution: Ensure nginx uses `fullchain.pem`:
   ```nginx
   ssl_certificate /etc/nginx/ssl/fullchain.pem;  # Correct
   ssl_certificate /etc/nginx/ssl/cert.pem;       # Wrong - missing intermediate
   ```

3. **Certificate expired**
   - Check: `sudo scripts/setup-ssl.sh --check-validity`
   - Solution: Renew certificate

### Nginx Won't Start After SSL Setup

**Error: "nginx: [emerg] cannot load certificate"**

**Solutions:**

1. **Check certificate files exist:**

```bash
ls -l docker/nginx/ssl/
# Must have fullchain.pem and privkey.pem
```

2. **Verify file permissions:**

```bash
# Certificate should be readable
sudo chmod 644 docker/nginx/ssl/fullchain.pem

# Private key should be private
sudo chmod 600 docker/nginx/ssl/privkey.pem
```

3. **Test nginx configuration:**

```bash
docker run --rm \
  -v $(pwd)/docker/nginx/nginx.conf:/etc/nginx/nginx.conf:ro \
  -v $(pwd)/docker/nginx/conf.d:/etc/nginx/conf.d:ro \
  nginx:1.25-alpine nginx -t
```

4. **Check nginx error logs:**

```bash
docker logs goimg-nginx
```

### Auto-Renewal Not Working

**Issue: Certificate expired despite automatic renewal being configured**

**Diagnosis:**

1. **Check systemd timer status:**

```bash
systemctl status certbot-renewal.timer
systemctl list-timers certbot-renewal.timer

# View last run logs
journalctl -u certbot-renewal.service --since "1 week ago"
```

2. **Check cron job:**

```bash
sudo crontab -l | grep certbot
```

3. **Test renewal manually:**

```bash
sudo scripts/setup-ssl.sh --renew --dry-run
```

**Solutions:**

1. **Re-setup auto-renewal:**

```bash
sudo scripts/setup-ssl.sh --setup-renewal
```

2. **Check renewal logs:**

```bash
sudo tail -f /var/log/letsencrypt/letsencrypt.log
sudo tail -f /home/user/goimg-datalayer/logs/ssl-setup.log
```

### DNS-01 Challenge Failed

**Error: "DNS problem: NXDOMAIN looking up TXT"**

**Solutions:**

1. **Verify DNS API credentials:**

```bash
# Test Cloudflare API token
curl -X GET "https://api.cloudflare.com/client/v4/user/tokens/verify" \
  -H "Authorization: Bearer YOUR_API_TOKEN"
```

2. **Check DNS plugin installation:**

```bash
dpkg -l | grep certbot-dns
# Should show python3-certbot-dns-cloudflare (or your provider)
```

3. **Wait for DNS propagation:**

DNS changes can take 5-60 minutes to propagate globally.

```bash
# Check TXT record propagation
dig +short TXT _acme-challenge.yourdomain.com @8.8.8.8
```

### Mixed Content Warnings

**Issue: Some resources load over HTTP instead of HTTPS**

**Solution:**

1. **Update Content Security Policy to enforce HTTPS:**

```nginx
# In nginx.conf
add_header Content-Security-Policy "upgrade-insecure-requests; default-src 'self' https:; ...";
```

2. **Check for hardcoded HTTP URLs** in your application code

3. **Use relative URLs** instead of absolute URLs where possible

## Security Best Practices

### 1. Strong TLS Configuration

The nginx configuration includes:

```nginx
# TLS 1.2 and 1.3 only (no TLS 1.0/1.1 - vulnerable)
ssl_protocols TLSv1.2 TLSv1.3;

# Strong cipher suites (forward secrecy)
ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:...';
ssl_prefer_server_ciphers off;  # Let client choose (TLS 1.3)

# OCSP stapling (faster handshake, improved privacy)
ssl_stapling on;
ssl_stapling_verify on;

# Session cache (performance)
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 10m;
ssl_session_tickets off;  # Prevent session resumption attacks
```

### 2. HSTS (HTTP Strict Transport Security)

Force HTTPS for 1 year (prevents SSL stripping attacks):

```nginx
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;
```

**HSTS Preload:**
- Submit to [HSTS Preload List](https://hstspreload.org/) after verifying HTTPS works
- Once preloaded, browsers will NEVER connect via HTTP (irreversible)
- Test thoroughly with `max-age=300` (5 minutes) first

### 3. Diffie-Hellman Parameters

Generate custom DH parameters for enhanced security:

```bash
sudo scripts/setup-ssl.sh --generate-dhparam
```

This creates `docker/nginx/ssl/dhparam.pem` with 2048-bit parameters.

Uncomment in `docker/nginx/nginx.conf`:

```nginx
ssl_dhparam /etc/nginx/ssl/dhparam.pem;
```

### 4. Certificate Pinning (Advanced)

For critical applications, consider HTTP Public Key Pinning (HPKP).

**Warning:** HPKP can lock users out if misconfigured. Not recommended for most applications.

### 5. Monitor Certificate Expiration

Set up monitoring alerts:

```bash
# Add to cron (daily check)
0 9 * * * /home/user/goimg-datalayer/scripts/setup-ssl.sh --check-validity >> /var/log/ssl-check.log 2>&1
```

Send alert if < 7 days remaining:

```bash
#!/bin/bash
DAYS_REMAINING=$(sudo /home/user/goimg-datalayer/scripts/setup-ssl.sh --check-validity 2>&1 | grep "Days remaining" | awk '{print $4}')

if [ "$DAYS_REMAINING" -lt 7 ]; then
    echo "SSL certificate expires in $DAYS_REMAINING days!" | mail -s "SSL Alert" admin@yourdomain.com
fi
```

### 6. Regular Security Audits

**Monthly:**
- Run SSL Labs test: https://www.ssllabs.com/ssltest/
- Target grade: **A+**

**Quarterly:**
- Review nginx security configuration
- Update cipher suites as needed
- Check for nginx/openssl security updates

**Annually:**
- Rotate private keys (optional but recommended)
- Review HSTS/CSP policies
- Audit access logs for suspicious HTTPS handshake patterns

### 7. Backup Private Keys

**Important:** Store private keys securely in case of disaster recovery.

```bash
# Backup certificates and keys (encrypted)
sudo tar -czf ssl-backup-$(date +%Y%m%d).tar.gz \
  docker/nginx/ssl/ \
  /etc/letsencrypt/

# Encrypt backup
gpg --symmetric --cipher-algo AES256 ssl-backup-$(date +%Y%m%d).tar.gz

# Store encrypted backup in secure location (not on same server)
# Delete unencrypted tar.gz
rm ssl-backup-$(date +%Y%m%d).tar.gz
```

### 8. Restrict Private Key Access

```bash
# Private key should only be readable by root and nginx user
sudo chown root:root docker/nginx/ssl/privkey.pem
sudo chmod 600 docker/nginx/ssl/privkey.pem

# Verify
sudo ls -l docker/nginx/ssl/privkey.pem
# Should show: -rw------- 1 root root
```

## Additional Resources

- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [Certbot User Guide](https://eff-certbot.readthedocs.io/)
- [SSL Labs Best Practices](https://github.com/ssllabs/research/wiki/SSL-and-TLS-Deployment-Best-Practices)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)
- [OWASP TLS Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Transport_Layer_Protection_Cheat_Sheet.html)

## Quick Reference

### Common Commands

```bash
# Obtain certificate (HTTP-01)
sudo scripts/setup-ssl.sh --obtain --domain example.com --email admin@example.com

# Obtain wildcard certificate (DNS-01)
sudo scripts/setup-ssl.sh --obtain --domain example.com --email admin@example.com --dns cloudflare

# Check certificate validity
sudo scripts/setup-ssl.sh --check-validity

# Manual renewal
sudo scripts/setup-ssl.sh --renew

# Setup auto-renewal
sudo scripts/setup-ssl.sh --setup-renewal

# Test renewal (dry run)
sudo scripts/setup-ssl.sh --renew --dry-run

# Generate DH parameters
sudo scripts/setup-ssl.sh --generate-dhparam

# Test SSL configuration
sudo scripts/setup-ssl.sh --test --domain example.com
```

### Important Files

| File | Purpose |
|------|---------|
| `docker/nginx/ssl/fullchain.pem` | Certificate + intermediate chain |
| `docker/nginx/ssl/privkey.pem` | Private key (keep secure!) |
| `docker/nginx/nginx.conf` | Main nginx configuration |
| `docker/nginx/conf.d/api.conf` | API server configuration with SSL |
| `/etc/letsencrypt/live/example.com/` | Let's Encrypt certificate directory |
| `/home/user/goimg-datalayer/logs/ssl-setup.log` | SSL setup script logs |
| `/var/log/letsencrypt/letsencrypt.log` | Certbot logs |

### SSL Labs Grading Criteria

| Grade | Requirements |
|-------|--------------|
| A+ | All A requirements + HSTS with long max-age |
| A | TLS 1.2+, strong ciphers, forward secrecy, no vulnerabilities |
| B | TLS 1.0 or weak ciphers present |
| C | Obsolete protocols or weak keys |
| F | Severe vulnerabilities (expired cert, untrusted CA, etc.) |

### Support

For issues or questions:
1. Check logs: `/home/user/goimg-datalayer/logs/ssl-setup.log`
2. Review [Troubleshooting](#troubleshooting) section
3. Consult [Let's Encrypt Community](https://community.letsencrypt.org/)
4. Open GitHub issue with logs attached
