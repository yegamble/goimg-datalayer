# Production Deployment Quick Start

Fast-track guide for deploying goimg-datalayer to production.

## Prerequisites

- Ubuntu 22.04 LTS server
- Domain name pointed to server IP
- Root or sudo access

## 10-Minute Deployment

### 1. Install Dependencies (2 minutes)

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
newgrp docker

# Install tools
sudo apt install -y git make certbot awscli
```

### 2. Clone & Setup (1 minute)

```bash
# Clone repository
cd /opt
sudo git clone https://github.com/yourusername/goimg-datalayer.git
sudo chown -R $USER:$USER goimg-datalayer
cd goimg-datalayer

# Create data directories
sudo mkdir -p /var/lib/goimg/{postgres,backups,uploads}
sudo chown -R 10001:10001 /var/lib/goimg
```

### 3. SSL Certificate (3 minutes)

```bash
# Create webroot
sudo mkdir -p /var/www/certbot

# Get certificate
sudo certbot certonly --standalone \
    -d yourdomain.com \
    --email admin@yourdomain.com \
    --agree-tos --non-interactive

# Copy to nginx
sudo cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem \
    docker/nginx/ssl/fullchain.pem
sudo cp /etc/letsencrypt/live/yourdomain.com/privkey.pem \
    docker/nginx/ssl/privkey.pem
sudo chown 101:101 docker/nginx/ssl/*.pem
sudo chmod 644 docker/nginx/ssl/fullchain.pem
sudo chmod 600 docker/nginx/ssl/privkey.pem
```

### 4. Configure Environment (2 minutes)

```bash
cd docker

# Copy template
cp .env.prod.example .env.prod

# Generate secrets
cat >> .env.prod << EOF
DB_PASSWORD=$(openssl rand -base64 32)
REDIS_PASSWORD=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -base64 64)
EOF

# Edit configuration
nano .env.prod
```

**Update these values:**
```bash
DB_USER=goimg_prod
DB_NAME=goimg_production
CORS_ALLOWED_ORIGINS=https://yourdomain.com
STORAGE_PROVIDER=s3
S3_BUCKET=your-bucket-name
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
```

### 5. Deploy (2 minutes)

```bash
cd /opt/goimg-datalayer

# Build images
docker build -f docker/Dockerfile.api -t goimg-api:latest .
docker build -f docker/Dockerfile.worker -t goimg-worker:latest .

# Start services
docker-compose -f docker/docker-compose.prod.yml --env-file docker/.env.prod up -d

# Wait for PostgreSQL
sleep 15

# Run migrations
export $(cat docker/.env.prod | grep -v '^#' | xargs)
make migrate-up

# Verify
curl https://yourdomain.com/health
```

## Done!

Your API is now running at `https://yourdomain.com`

## Next Steps

1. **Set up automated backups:**
   ```bash
   # Add to crontab
   0 2 * * * cd /opt/goimg-datalayer && ./scripts/backup-db.sh -d goimg_production -u goimg_prod -s your-backup-bucket
   ```

2. **Configure firewall:**
   ```bash
   sudo ufw allow 22,80,443/tcp
   sudo ufw enable
   ```

3. **Enable SSL auto-renewal:**
   ```bash
   sudo crontab -e
   # Add: 0 0 * * * certbot renew --quiet --post-hook "docker exec goimg-nginx nginx -s reload"
   ```

4. **Monitor logs:**
   ```bash
   docker-compose -f docker/docker-compose.prod.yml logs -f
   ```

## Troubleshooting

**SSL errors?**
```bash
# Verify certificate
openssl s_client -connect yourdomain.com:443
```

**Database connection failed?**
```bash
# Check PostgreSQL
docker exec goimg-postgres pg_isready
docker logs goimg-postgres
```

**API not responding?**
```bash
# Check logs
docker logs goimg-api
docker logs goimg-nginx
```

For detailed documentation, see [README.md](./README.md)
