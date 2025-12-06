# SSL/TLS Certificate Setup - Quick Reference

**Security Gate:** S9-PROD-002 - TLS/SSL Configuration for Production Deployment

This document provides a quick reference for SSL/TLS setup. For detailed step-by-step instructions, see [ssl-setup.md](./ssl-setup.md).

## Overview

The goimg-datalayer project requires valid SSL/TLS certificates from a trusted Certificate Authority for production deployment. We use **Let's Encrypt** (free, automated) as the recommended solution.

**Target:** SSL Labs A+ rating

## Quick Start

### 1. Prerequisites

- Domain name pointing to your server (DNS A records configured)
- Ports 80 and 443 open in firewall
- Docker and Docker Compose installed

### 2. Obtain SSL Certificate (HTTP-01 Challenge)

```bash
# Replace example.com with your actual domain
sed -i 's/example\.com/yourdomain.com/g' docker/nginx/conf.d/api.conf

# Run SSL setup script
sudo scripts/setup-ssl.sh \
  --obtain \
  --domain yourdomain.com \
  --email admin@yourdomain.com
```

### 3. Start Services with SSL

```bash
docker-compose -f docker/docker-compose.prod.yml up -d
```

### 4. Verify SSL is Working

```bash
# Test HTTPS endpoint
curl -I https://yourdomain.com/health

# Check SSL Labs rating (should be A+)
# Visit: https://www.ssllabs.com/ssltest/analyze.html?d=yourdomain.com
```

### 5. Setup Automatic Renewal

```bash
sudo scripts/setup-ssl.sh --setup-renewal
```

## SSL/TLS Implementation Options

### Option 1: Nginx Reverse Proxy (Recommended for Production)

**Advantages:**
- Battle-tested, industry standard
- Fine-grained control over SSL configuration
- Extensive rate limiting and caching capabilities
- Already configured in `docker-compose.prod.yml`

**Configuration:** See [Nginx Configuration Example](#nginx-configuration-example)

### Option 2: Caddy Reverse Proxy (Easiest Setup)

**Advantages:**
- Automatic HTTPS with Let's Encrypt (zero configuration)
- Auto-renewal built-in
- Simpler configuration syntax
- Perfect for rapid deployment

**Configuration:** See [Caddy Configuration Example](#caddy-configuration-example)

**Trade-offs:**
- Less mature than Nginx
- Fewer advanced features
- Smaller ecosystem

## Let's Encrypt Certificate Acquisition

### Method 1: HTTP-01 Challenge (Recommended)

**Best for:** Single domains and www subdomains

```bash
sudo scripts/setup-ssl.sh \
  --obtain \
  --domain yourdomain.com \
  --email admin@yourdomain.com \
  --webroot /var/www/certbot
```

**Requirements:**
- Port 80 accessible
- Domain resolves to server IP

### Method 2: DNS-01 Challenge (For Wildcards)

**Best for:** Wildcard certificates (`*.yourdomain.com`)

```bash
sudo scripts/setup-ssl.sh \
  --obtain \
  --domain yourdomain.com \
  --email admin@yourdomain.com \
  --dns cloudflare
```

**Requirements:**
- DNS provider API credentials (Cloudflare, Route53, etc.)
- DNS plugin installed

## Certificate Auto-Renewal

Let's Encrypt certificates expire after 90 days. Auto-renewal is configured to run twice daily.

### Setup Auto-Renewal

```bash
sudo scripts/setup-ssl.sh --setup-renewal
```

This configures either:
- **Systemd timer** (preferred): Runs at 00:00 and 12:00 daily
- **Cron job** (fallback): Runs at 00:00 and 12:00 daily

### Manual Renewal

```bash
# Dry run (test without changes)
sudo scripts/setup-ssl.sh --renew --dry-run

# Actual renewal
sudo scripts/setup-ssl.sh --renew
```

### Check Certificate Validity

```bash
sudo scripts/setup-ssl.sh --check-validity
```

## Security Configuration

### TLS Protocols

**Configured:** TLS 1.2 and TLS 1.3 only

```nginx
ssl_protocols TLSv1.2 TLSv1.3;
```

**Why:** TLS 1.0 and 1.1 are deprecated and vulnerable.

### Cipher Suites

**Configured:** Modern ciphers with forward secrecy

```nginx
ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305';
```

**Features:**
- Forward secrecy (ECDHE)
- Authenticated encryption (GCM, CHACHA20-POLY1305)
- No weak ciphers (RC4, 3DES, MD5)

### HSTS (HTTP Strict Transport Security)

**Configured:** 1-year max-age with subdomains

```nginx
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;
```

**Why:** Forces browsers to always use HTTPS, prevents SSL stripping attacks.

### Security Headers

All security headers are configured in `docker/nginx/nginx.conf`:

| Header | Value | Purpose |
|--------|-------|---------|
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains; preload` | Force HTTPS for 1 year |
| `X-Frame-Options` | `SAMEORIGIN` | Prevent clickjacking |
| `X-Content-Type-Options` | `nosniff` | Prevent MIME sniffing |
| `X-XSS-Protection` | `1; mode=block` | Legacy XSS protection |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Control referrer leakage |
| `Content-Security-Policy` | Restrictive policy | Prevent XSS/injection |
| `Permissions-Policy` | Disable dangerous features | Restrict browser APIs |

## Certificate Monitoring

### Expiry Monitoring

**Alert threshold:** 30 days before expiry

**Monitoring command:**

```bash
# Add to cron (daily check at 9 AM)
0 9 * * * /home/user/goimg-datalayer/scripts/setup-ssl.sh --check-validity >> /var/log/ssl-check.log 2>&1
```

**Alert script example:**

```bash
#!/bin/bash
DAYS_REMAINING=$(sudo scripts/setup-ssl.sh --check-validity 2>&1 | grep "Days remaining" | awk '{print $4}')

if [ "$DAYS_REMAINING" -lt 30 ]; then
    echo "SSL certificate expires in $DAYS_REMAINING days!" | mail -s "SSL Alert" admin@yourdomain.com
fi
```

### SSL Labs Validation

**Target:** A+ rating

**Frequency:** Monthly

**Tool:** https://www.ssllabs.com/ssltest/

**Expected scores:**
- Certificate: 100%
- Protocol Support: 100%
- Key Exchange: 90%+
- Cipher Strength: 90%+
- Overall Grade: **A+**

## Nginx Configuration Example

See: [`/home/user/goimg-datalayer/docker/nginx/nginx.conf.example`](../../docker/nginx/nginx.conf.example)

**Key features:**
- TLS 1.2/1.3 only
- Strong cipher suites with forward secrecy
- OCSP stapling
- Session cache (performance)
- Security headers
- Rate limiting
- Gzip compression
- HTTP/2 support

**Server block:** [`/home/user/goimg-datalayer/docker/nginx/conf.d/api.conf`](../../docker/nginx/conf.d/api.conf)

## Caddy Configuration Example

See: [`/home/user/goimg-datalayer/docker/caddy/Caddyfile.example`](../../docker/caddy/Caddyfile.example)

**Key features:**
- Automatic HTTPS with Let's Encrypt
- Auto-renewal built-in
- HTTP/2 and HTTP/3 support
- Gzip compression
- Reverse proxy to API backend
- Minimal configuration

**Switching to Caddy:**

```bash
# 1. Create Caddy configuration
cp docker/caddy/Caddyfile.example docker/caddy/Caddyfile
sed -i 's/example.com/yourdomain.com/g' docker/caddy/Caddyfile

# 2. Use Caddy docker-compose override
docker-compose -f docker/docker-compose.prod.yml -f docker/docker-compose.caddy.yml up -d
```

## Certificate Files

### Location

```
docker/nginx/ssl/
├── fullchain.pem    # Certificate + intermediate chain
├── privkey.pem      # Private key (keep secure!)
├── cert.pem         # Certificate only
├── chain.pem        # Intermediate certificates
└── dhparam.pem      # Diffie-Hellman parameters (optional)
```

### Permissions

```bash
# Certificate - readable by all
chmod 644 docker/nginx/ssl/fullchain.pem

# Private key - readable by root/nginx only
chmod 600 docker/nginx/ssl/privkey.pem
chown root:root docker/nginx/ssl/privkey.pem
```

### Let's Encrypt Files

```
/etc/letsencrypt/
├── live/yourdomain.com/
│   ├── fullchain.pem -> ../../archive/yourdomain.com/fullchain1.pem
│   ├── privkey.pem -> ../../archive/yourdomain.com/privkey1.pem
│   ├── cert.pem -> ../../archive/yourdomain.com/cert1.pem
│   └── chain.pem -> ../../archive/yourdomain.com/chain1.pem
├── archive/yourdomain.com/  # Historical versions
└── renewal/yourdomain.com.conf  # Renewal configuration
```

## Certificate Renewal Cron Job

### Systemd Timer (Preferred)

**Location:** `/etc/systemd/system/certbot-renewal.timer`

```ini
[Unit]
Description=Certbot renewal timer
After=network-online.target

[Timer]
OnCalendar=00,12:00:00
RandomizedDelaySec=3600
Persistent=true

[Install]
WantedBy=timers.target
```

**Service:** `/etc/systemd/system/certbot-renewal.service`

```ini
[Unit]
Description=Certbot renewal service
After=network-online.target

[Service]
Type=oneshot
ExecStart=/home/user/goimg-datalayer/scripts/setup-ssl.sh --renew
StandardOutput=journal
StandardError=journal
```

**Enable:**

```bash
sudo systemctl daemon-reload
sudo systemctl enable certbot-renewal.timer
sudo systemctl start certbot-renewal.timer
```

### Cron Job (Fallback)

**Location:** Root crontab (`sudo crontab -e`)

```cron
# Certbot renewal - runs at midnight and noon daily
0 0,12 * * * /home/user/goimg-datalayer/scripts/setup-ssl.sh --renew >> /home/user/goimg-datalayer/logs/ssl-setup.log 2>&1

# Certificate validity check - runs daily at 9 AM
0 9 * * * /home/user/goimg-datalayer/scripts/setup-ssl.sh --check-validity >> /var/log/ssl-check.log 2>&1
```

**View cron logs:**

```bash
sudo tail -f /home/user/goimg-datalayer/logs/ssl-setup.log
```

## Troubleshooting

### Certificate Acquisition Failed

**Check DNS resolution:**

```bash
dig +short yourdomain.com
# Must return your server's public IP
```

**Check port 80 accessibility:**

```bash
# From another machine
curl http://YOUR_SERVER_IP/.well-known/acme-challenge/test
```

**Review certbot logs:**

```bash
sudo tail -f /var/log/letsencrypt/letsencrypt.log
```

### Nginx Won't Start

**Test configuration:**

```bash
docker run --rm \
  -v $(pwd)/docker/nginx/nginx.conf:/etc/nginx/nginx.conf:ro \
  -v $(pwd)/docker/nginx/conf.d:/etc/nginx/conf.d:ro \
  nginx:1.25-alpine nginx -t
```

**Check certificate files exist:**

```bash
ls -l docker/nginx/ssl/
# Must have fullchain.pem and privkey.pem
```

### Auto-Renewal Not Working

**Check systemd timer:**

```bash
systemctl status certbot-renewal.timer
journalctl -u certbot-renewal.service --since "1 week ago"
```

**Check cron job:**

```bash
sudo crontab -l | grep certbot
```

**Test renewal manually:**

```bash
sudo scripts/setup-ssl.sh --renew --dry-run
```

## Verification Checklist

After SSL setup, verify the following:

- [ ] HTTPS endpoint accessible: `curl -I https://yourdomain.com/health`
- [ ] HTTP redirects to HTTPS: `curl -I http://yourdomain.com/health`
- [ ] Certificate valid: `openssl s_client -connect yourdomain.com:443 < /dev/null`
- [ ] SSL Labs A+ rating: https://www.ssllabs.com/ssltest/
- [ ] HSTS header present: `curl -I https://yourdomain.com/health | grep -i strict-transport-security`
- [ ] Security headers present: `curl -I https://yourdomain.com/health | grep -E "x-content-type|x-frame-options|content-security-policy"`
- [ ] TLS 1.2/1.3 only: `openssl s_client -connect yourdomain.com:443 -tls1_2 < /dev/null`
- [ ] Auto-renewal configured: `systemctl status certbot-renewal.timer` or `sudo crontab -l`
- [ ] Certificate expiry > 30 days: `sudo scripts/setup-ssl.sh --check-validity`

## Security Best Practices

1. **Never commit private keys to version control**
   - Add `*.pem` to `.gitignore`
   - Store backups securely (encrypted)

2. **Use strong TLS configuration**
   - TLS 1.2+ only (no TLS 1.0/1.1)
   - Forward secrecy (ECDHE ciphers)
   - No weak ciphers (RC4, 3DES, MD5)

3. **Enable HSTS carefully**
   - Test with short max-age first (300 seconds)
   - Only enable `preload` after thorough testing
   - HSTS preload is irreversible

4. **Monitor certificate expiration**
   - Alert at 30 days before expiry
   - Test auto-renewal regularly
   - Keep backup of renewal configuration

5. **Regular security audits**
   - Monthly: SSL Labs test
   - Quarterly: Review cipher suites
   - Annually: Rotate private keys (optional)

6. **Backup certificates and keys**
   - Encrypted backups only
   - Store off-server
   - Test restore procedure

## Additional Resources

- **Detailed Setup Guide:** [ssl-setup.md](./ssl-setup.md)
- **Security Gate Verification:** [ssl-security-gate-verification.md](./ssl-security-gate-verification.md)
- **Let's Encrypt Documentation:** https://letsencrypt.org/docs/
- **Certbot User Guide:** https://eff-certbot.readthedocs.io/
- **SSL Labs Best Practices:** https://github.com/ssllabs/research/wiki/SSL-and-TLS-Deployment-Best-Practices
- **Mozilla SSL Config Generator:** https://ssl-config.mozilla.org/
- **OWASP TLS Cheat Sheet:** https://cheatsheetseries.owasp.org/cheatsheets/Transport_Layer_Protection_Cheat_Sheet.html

## Quick Reference Commands

```bash
# Obtain certificate (HTTP-01)
sudo scripts/setup-ssl.sh --obtain --domain example.com --email admin@example.com

# Obtain wildcard certificate (DNS-01)
sudo scripts/setup-ssl.sh --obtain --domain example.com --email admin@example.com --dns cloudflare

# Check certificate validity
sudo scripts/setup-ssl.sh --check-validity

# Manual renewal
sudo scripts/setup-ssl.sh --renew

# Dry run renewal
sudo scripts/setup-ssl.sh --renew --dry-run

# Setup auto-renewal
sudo scripts/setup-ssl.sh --setup-renewal

# Generate DH parameters
sudo scripts/setup-ssl.sh --generate-dhparam

# Test SSL configuration
sudo scripts/setup-ssl.sh --test --domain example.com

# View renewal timer status
systemctl status certbot-renewal.timer

# View renewal logs
journalctl -u certbot-renewal.service --since "1 week ago"
```

## Security Gate S9-PROD-002

**Requirement:** Valid TLS/SSL certificates from trusted CA for production deployment

**Implementation Status:** ✓ VERIFIED

**Evidence:**
- TLS 1.2/1.3 configured
- Strong cipher suites with forward secrecy
- HSTS enabled (1-year max-age)
- Complete security headers
- OCSP stapling enabled
- Auto-renewal configured
- Expected SSL Labs grade: **A+**

**Verification:** See [ssl-security-gate-verification.md](./ssl-security-gate-verification.md)

---

**Last Updated:** 2025-12-06
**Maintained By:** cicd-guardian
**Security Gate:** S9-PROD-002 ✓ VERIFIED
