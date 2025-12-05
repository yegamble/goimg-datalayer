# SSL Certificates

This directory should contain your SSL certificates for HTTPS.

## Production (Let's Encrypt)

Use Certbot to obtain free SSL certificates from Let's Encrypt:

```bash
# Install Certbot
sudo apt-get update
sudo apt-get install certbot

# Obtain certificate (ensure nginx is running and port 80 is accessible)
sudo certbot certonly --webroot \
  -w /var/www/certbot \
  -d yourdomain.com \
  -d www.yourdomain.com \
  --email admin@yourdomain.com \
  --agree-tos \
  --no-eff-email

# Copy certificates to nginx ssl directory
sudo cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem ./fullchain.pem
sudo cp /etc/letsencrypt/live/yourdomain.com/privkey.pem ./privkey.pem
sudo chmod 644 fullchain.pem
sudo chmod 600 privkey.pem
```

### Auto-renewal with Certbot

Add to crontab:
```bash
0 0 * * * certbot renew --quiet --post-hook "docker exec goimg-nginx nginx -s reload"
```

## Development (Self-Signed)

For local development, generate self-signed certificates:

```bash
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout selfsigned.key \
  -out selfsigned.crt \
  -subj "/C=US/ST=State/L=City/O=Org/CN=localhost"
```

Then update `docker/nginx/conf.d/api.conf` to use:
```nginx
ssl_certificate /etc/nginx/ssl/selfsigned.crt;
ssl_certificate_key /etc/nginx/ssl/selfsigned.key;
```

## Diffie-Hellman Parameters (Recommended)

Generate strong DH parameters for enhanced security:

```bash
openssl dhparam -out dhparam.pem 2048
```

Uncomment the `ssl_dhparam` line in `nginx.conf`.

## Security Best Practices

1. Never commit private keys to version control
2. Use 2048-bit or 4096-bit keys
3. Rotate certificates before expiration
4. Monitor certificate expiration dates
5. Use HSTS after validating HTTPS works correctly
6. Test SSL configuration at https://www.ssllabs.com/ssltest/

## File Permissions

Recommended permissions:
- `fullchain.pem` (certificate): 644
- `privkey.pem` (private key): 600
- Directory: 755
