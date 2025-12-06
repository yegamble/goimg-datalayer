# Caddy Reverse Proxy Configuration

This directory contains Caddy configuration for automatic HTTPS with Let's Encrypt.

## Why Caddy?

Caddy offers the simplest path to production HTTPS:

- **Automatic HTTPS**: Zero-config SSL/TLS with Let's Encrypt
- **Auto-renewal**: Certificates renew automatically (no cron jobs)
- **Modern defaults**: TLS 1.2/1.3, strong ciphers, HTTP/2, HTTP/3
- **SSL Labs A+**: Achieves A+ rating out of the box
- **Simple syntax**: Easier to configure than Nginx

## Quick Start

### 1. Create Caddyfile

```bash
cd /home/user/goimg-datalayer

# Copy example configuration
cp docker/caddy/Caddyfile.example docker/caddy/Caddyfile

# Replace domain placeholder
sed -i 's/example\.com/yourdomain.com/g' docker/caddy/Caddyfile
```

### 2. Update Email Address

Edit `docker/caddy/Caddyfile` and replace `admin@example.com` with your email:

```caddyfile
{
    email admin@yourdomain.com  # Change this!
}

example.com, www.example.com {
    tls admin@yourdomain.com {  # And this!
        # ...
    }
}
```

### 3. Deploy with Docker Compose

```bash
# Start with Caddy instead of Nginx
docker-compose -f docker/docker-compose.prod.yml \
  -f docker/docker-compose.caddy.yml up -d
```

Caddy will:
1. Start listening on ports 80 and 443
2. Automatically obtain SSL certificate from Let's Encrypt
3. Redirect all HTTP traffic to HTTPS
4. Serve your API over HTTPS

### 4. Verify SSL is Working

```bash
# Test HTTPS endpoint
curl -I https://yourdomain.com/health

# Check SSL Labs rating (should be A+)
# Visit: https://www.ssllabs.com/ssltest/analyze.html?d=yourdomain.com
```

## Configuration

### Caddyfile Structure

```caddyfile
# Global options
{
    email admin@yourdomain.com
    # More global settings...
}

# HTTP to HTTPS redirect
http://example.com {
    redir https://{host}{uri} permanent
}

# HTTPS server
example.com, www.example.com {
    # TLS settings (automatic HTTPS)
    tls admin@example.com {
        protocols tls1.2 tls1.3
    }

    # Security headers
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        # More headers...
    }

    # Reverse proxy to API
    handle /api/* {
        reverse_proxy api:8080
    }
}
```

### Customization

Edit `docker/caddy/Caddyfile` to:

1. **Change domain**: Replace `example.com` with your domain
2. **Add subdomains**: Add more domains to the server block
3. **Adjust rate limits**: Configure per-endpoint limits
4. **Serve frontend**: Add static file serving
5. **Add middleware**: Enable additional Caddy modules

## Certificate Management

### Automatic Renewal

Caddy automatically renews certificates **30 days before expiration**.

No cron jobs or manual intervention required!

### Certificate Location

**Inside container:**
```
/data/caddy/certificates/acme-v02.api.letsencrypt.org-directory/
└── yourdomain.com/
    ├── yourdomain.com.crt  # Certificate
    ├── yourdomain.com.key  # Private key
    └── yourdomain.com.json # Metadata
```

**On host (Docker volume):**
```
/var/lib/docker/volumes/caddy_data/_data/certificates/
```

### View Certificates

```bash
# List all certificates
docker exec goimg-caddy caddy list-certificates

# View certificate details
docker exec goimg-caddy ls -lh /data/caddy/certificates/acme-v02.api.letsencrypt.org-directory/yourdomain.com/
```

### Manual Certificate Renewal

```bash
# Reload configuration (triggers renewal check)
docker exec goimg-caddy caddy reload --config /etc/caddy/Caddyfile

# Force reload
docker restart goimg-caddy
```

## Testing

### Test Caddyfile Syntax

```bash
# Validate configuration
docker run --rm \
  -v $(pwd)/docker/caddy/Caddyfile:/etc/caddy/Caddyfile \
  caddy:2-alpine caddy validate --config /etc/caddy/Caddyfile
```

### Dry Run (Local Testing)

```bash
# Run Caddy in foreground
docker run --rm -it \
  -p 80:80 -p 443:443 \
  -v $(pwd)/docker/caddy/Caddyfile:/etc/caddy/Caddyfile \
  caddy:2-alpine caddy run --config /etc/caddy/Caddyfile
```

### Staging Let's Encrypt

For testing, use Let's Encrypt staging to avoid rate limits:

Edit `Caddyfile`, add to global options:

```caddyfile
{
    acme_ca https://acme-staging-v02.api.letsencrypt.org/directory
}
```

**Note:** Staging certificates are not trusted by browsers (expect warnings).

## Monitoring

### Logs

```bash
# View Caddy logs
docker logs goimg-caddy -f

# View access logs
docker exec goimg-caddy tail -f /var/log/caddy/access.log

# View error logs
docker exec goimg-caddy tail -f /var/log/caddy/caddy.log
```

### Admin API

Caddy exposes an admin API on `localhost:2019`:

```bash
# View current configuration
curl http://localhost:2019/config/ | jq

# View metrics
curl http://localhost:2019/metrics

# Health check
curl http://localhost:2019/health
```

### Certificate Expiry

Caddy handles renewal automatically, but you can monitor expiry:

```bash
# Check certificate validity
docker exec goimg-caddy caddy list-certificates

# Extract expiry date
docker exec goimg-caddy sh -c "cd /data/caddy/certificates/acme-v02.api.letsencrypt.org-directory/yourdomain.com && cat yourdomain.com.crt | openssl x509 -noout -enddate"
```

## Troubleshooting

### Certificate Acquisition Failed

**Symptoms:**
- Caddy logs show "failed to obtain certificate"
- Browser shows "connection refused" or "SSL error"

**Solutions:**

1. **Check DNS resolution:**
   ```bash
   dig +short yourdomain.com
   # Must return your server's public IP
   ```

2. **Check ports 80/443 are accessible:**
   ```bash
   # From another machine
   curl http://YOUR_SERVER_IP
   ```

3. **Check firewall:**
   ```bash
   sudo ufw status
   # Ensure ports 80 and 443 are allowed
   ```

4. **Review Caddy logs:**
   ```bash
   docker logs goimg-caddy
   ```

5. **Use staging environment:**
   ```caddyfile
   {
       acme_ca https://acme-staging-v02.api.letsencrypt.org/directory
   }
   ```

### Rate Limit Exceeded

Let's Encrypt has rate limits:
- 50 certificates per domain per week
- 5 duplicate certificates per week

**Solutions:**
- Wait one week for rate limit to reset
- Use staging environment for testing
- Check issued certificates at https://crt.sh/?q=yourdomain.com

### Configuration Errors

**Test configuration:**
```bash
docker run --rm \
  -v $(pwd)/docker/caddy/Caddyfile:/etc/caddy/Caddyfile \
  caddy:2-alpine caddy validate --config /etc/caddy/Caddyfile
```

**Common errors:**
- Missing email address
- Invalid domain format
- Syntax errors in Caddyfile

### Caddy Won't Start

**Check logs:**
```bash
docker logs goimg-caddy
```

**Test configuration:**
```bash
docker run --rm -v $(pwd)/docker/caddy/Caddyfile:/etc/caddy/Caddyfile caddy:2-alpine caddy validate --config /etc/caddy/Caddyfile
```

**Check port conflicts:**
```bash
sudo netstat -tlnp | grep -E ':(80|443)'
# Ensure nginx or other services aren't using these ports
```

## Switching Between Caddy and Nginx

### Switch to Caddy (from Nginx)

```bash
# Stop nginx, start Caddy
docker-compose -f docker/docker-compose.prod.yml \
  -f docker/docker-compose.caddy.yml up -d
```

### Switch to Nginx (from Caddy)

```bash
# Stop Caddy, start nginx
docker-compose -f docker/docker-compose.prod.yml up -d
```

## SSL Labs A+ Rating

Caddy's default configuration achieves **SSL Labs A+** rating:

- **Certificate**: 100% (Let's Encrypt trusted CA)
- **Protocol Support**: 100% (TLS 1.2/1.3 only)
- **Key Exchange**: 90%+ (ECDHE with strong curves)
- **Cipher Strength**: 90%+ (AES-GCM, ChaCha20-Poly1305)

**Additional features:**
- Forward secrecy (ECDHE)
- HSTS enabled (with Caddyfile config)
- OCSP stapling (automatic)
- Session tickets disabled (automatic)
- Modern TLS 1.3 support

**Test your domain:**
https://www.ssllabs.com/ssltest/analyze.html?d=yourdomain.com

## Advantages vs Nginx

| Feature | Caddy | Nginx |
|---------|-------|-------|
| Automatic HTTPS | ✓ Built-in | ✗ Manual |
| Auto-renewal | ✓ Built-in | ✗ Cron job needed |
| Configuration | Simple | Complex |
| HTTP/3 (QUIC) | ✓ Built-in | Module required |
| Default security | Excellent | Good (requires tuning) |
| Learning curve | Easy | Moderate |
| Ecosystem | Smaller | Larger |
| Performance | Excellent | Excellent |

## Additional Resources

- [Caddy Documentation](https://caddyserver.com/docs/)
- [Caddyfile Syntax](https://caddyserver.com/docs/caddyfile)
- [Automatic HTTPS](https://caddyserver.com/docs/automatic-https)
- [Caddy Docker Image](https://hub.docker.com/_/caddy)
- [Let's Encrypt Rate Limits](https://letsencrypt.org/docs/rate-limits/)

## Support

For issues or questions:
1. Check Caddy logs: `docker logs goimg-caddy`
2. Review [Troubleshooting](#troubleshooting) section
3. Consult [Caddy Community Forum](https://caddy.community/)
4. Open GitHub issue with logs attached
