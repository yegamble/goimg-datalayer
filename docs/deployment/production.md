# Production Deployment Guide

> Comprehensive guide for deploying goimg-datalayer to production environments
>
> **Security Gate**: S9-PROD (All controls verified - LAUNCH READY)

This guide covers deploying the goimg-datalayer application to production using Docker Compose with best practices for security, reliability, and performance.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Pre-Deployment Checklist](#pre-deployment-checklist)
- [Docker Compose Production Deployment](#docker-compose-production-deployment)
- [Post-Deployment Validation](#post-deployment-validation)
- [Monitoring and Observability](#monitoring-and-observability)
- [Maintenance and Operations](#maintenance-and-operations)
- [Troubleshooting](#troubleshooting)
- [Rollback Procedures](#rollback-procedures)
- [Security Hardening](#security-hardening)

## Overview

The goimg-datalayer production deployment consists of the following services:

| Service | Purpose | Resource Limits | Network |
|---------|---------|-----------------|---------|
| **nginx** | Reverse proxy with SSL termination | 0.5 CPU, 256MB RAM | frontend |
| **api** | Go API server | 2 CPU, 2GB RAM | frontend, backend |
| **worker** | Background job processor | 4 CPU, 4GB RAM | backend |
| **postgres** | PostgreSQL 16 database | 2 CPU, 2GB RAM | database, backend |
| **redis** | Cache and session store | 1 CPU, 1GB RAM | backend |
| **clamav** | Antivirus scanner | 2 CPU, 4GB RAM | backend |
| **ipfs** | IPFS node for decentralized storage | 2 CPU, 2GB RAM | backend |
| **backup** | Automated database backups | 0.5 CPU, 512MB RAM | database, backend |
| **prometheus** | Metrics collection | 1 CPU, 2GB RAM | backend, frontend |
| **grafana** | Metrics visualization | 1 CPU, 1GB RAM | frontend, backend |

**Total Minimum Server Requirements**:
- **CPU**: 16 vCPUs (recommended: 24+ for production)
- **RAM**: 32GB (recommended: 48GB+ for production)
- **Disk**: 500GB SSD (database, uploads, backups, IPFS)
- **Network**: 1Gbps connection

## Prerequisites

### 1. Server Requirements

**Operating System**:
- Ubuntu 22.04 LTS or 24.04 LTS (recommended)
- Debian 12+ (supported)
- Rocky Linux 9+ (supported)

**Software**:
```bash
# Check versions
docker --version    # Docker 24.0+
docker-compose --version  # Docker Compose 2.20+
git --version       # Git 2.40+
```

**Install if missing**:
```bash
# Docker and Docker Compose (Ubuntu/Debian)
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Git
sudo apt update && sudo apt install -y git

# Make
sudo apt install -y make
```

### 2. Domain and DNS Configuration

**Requirements**:
- Domain name (e.g., `api.example.com`)
- DNS A record pointing to server public IP
- Port 80 and 443 accessible from internet

**Verify DNS**:
```bash
dig +short api.example.com
# Should return your server's public IP
```

### 3. Firewall Configuration

**Required ports**:
```bash
# Allow HTTP/HTTPS (required for SSL certificate acquisition)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Allow SSH (for server access)
sudo ufw allow 22/tcp

# Optional: Grafana dashboard (restrict to trusted IPs)
sudo ufw allow from TRUSTED_IP to any port 3000 proto tcp

# Optional: Prometheus (restrict to trusted IPs)
sudo ufw allow from TRUSTED_IP to any port 9091 proto tcp

# Enable firewall
sudo ufw enable
```

### 4. SSL/TLS Certificates

**Obtain certificates before deployment**:
```bash
# See docs/deployment/ssl.md for detailed instructions

# Quick setup with Let's Encrypt
sudo /home/user/goimg-datalayer/scripts/setup-ssl.sh \
  --obtain \
  --domain api.example.com \
  --email admin@example.com
```

**Verify certificates exist**:
```bash
ls -la /etc/letsencrypt/live/api.example.com/
# Should show: fullchain.pem, privkey.pem, cert.pem, chain.pem
```

## Pre-Deployment Checklist

Complete this checklist before deploying to production:

### 1. Code and Configuration

- [ ] **Code checkout**: Clone repository to `/opt/goimg-datalayer` or `/home/user/goimg-datalayer`
  ```bash
  git clone https://github.com/yegamble/goimg-datalayer.git /opt/goimg-datalayer
  cd /opt/goimg-datalayer
  git checkout main  # Or specific release tag (e.g., v1.0.0)
  ```

- [ ] **Build Docker images**:
  ```bash
  make docker-build
  # Or build manually:
  docker build -t goimg-api:latest -f docker/Dockerfile.api .
  docker build -t goimg-worker:latest -f docker/Dockerfile.worker .
  ```

- [ ] **Verify image builds**:
  ```bash
  docker images | grep goimg
  # Should show: goimg-api:latest, goimg-worker:latest
  ```

### 2. Secret Configuration

**Critical**: Never use default secrets in production!

- [ ] **Generate strong secrets** (see [docs/deployment/secrets.md](./secrets.md)):
  ```bash
  # Create secrets directory
  sudo mkdir -p /etc/goimg/secrets
  sudo chmod 700 /etc/goimg/secrets

  # Generate JWT secret (RS256 4096-bit keypair recommended for production)
  openssl genrsa -out /tmp/jwt_private.pem 4096
  openssl rsa -in /tmp/jwt_private.pem -pubout -out /tmp/jwt_public.pem
  sudo mv /tmp/jwt_private.pem /etc/goimg/secrets/jwt_private.pem
  sudo mv /tmp/jwt_public.pem /etc/goimg/secrets/jwt_public.pem
  sudo chmod 600 /etc/goimg/secrets/jwt_private.pem
  sudo chmod 644 /etc/goimg/secrets/jwt_public.pem

  # For MVP with HS256 (temporary - migrate to RS256 later):
  openssl rand -base64 64 | sudo tee /etc/goimg/secrets/jwt_secret > /dev/null

  # Database password (32+ characters)
  openssl rand -base64 32 | sudo tee /etc/goimg/secrets/db_password > /dev/null

  # Redis password (32+ characters)
  openssl rand -base64 32 | sudo tee /etc/goimg/secrets/redis_password > /dev/null

  # Grafana admin password
  openssl rand -base64 32 | sudo tee /etc/goimg/secrets/grafana_admin_password > /dev/null

  # Set restrictive permissions
  sudo chmod 600 /etc/goimg/secrets/*
  sudo chmod 644 /etc/goimg/secrets/jwt_public.pem  # Public key can be readable
  ```

- [ ] **Generate optional secrets** (if using S3, IPFS pinning, etc.):
  ```bash
  # S3-compatible storage (AWS, DO Spaces, Backblaze B2)
  echo "YOUR_S3_ACCESS_KEY" | sudo tee /etc/goimg/secrets/s3_access_key > /dev/null
  echo "YOUR_S3_SECRET_KEY" | sudo tee /etc/goimg/secrets/s3_secret_key > /dev/null

  # IPFS Pinata JWT (get from https://pinata.cloud)
  echo "YOUR_PINATA_JWT" | sudo tee /etc/goimg/secrets/ipfs_pinata_jwt > /dev/null

  # Set permissions
  sudo chmod 600 /etc/goimg/secrets/s3_*
  sudo chmod 600 /etc/goimg/secrets/ipfs_*
  ```

- [ ] **Verify all required secrets exist**:
  ```bash
  # Required secrets
  for secret in jwt_secret db_password redis_password grafana_admin_password; do
    if [ -f /etc/goimg/secrets/$secret ]; then
      echo "✓ $secret exists"
    else
      echo "✗ $secret MISSING"
    fi
  done
  ```

- [ ] **Verify secret permissions**:
  ```bash
  ls -la /etc/goimg/secrets/
  # All files should be: -rw------- 1 root root (600)
  # Except jwt_public.pem: -rw-r--r-- 1 root root (644)
  ```

### 3. Database Migrations

- [ ] **Run migrations** (before starting services):
  ```bash
  # Start only database for migrations
  docker-compose -f docker/docker-compose.prod.yml up -d postgres

  # Wait for database to be ready
  until docker exec goimg-postgres pg_isready -U goimg; do
    echo "Waiting for postgres..."
    sleep 2
  done

  # Run migrations
  make migrate-up
  # Or manually:
  # goose -dir migrations postgres "postgresql://goimg:$(sudo cat /etc/goimg/secrets/db_password)@localhost:5432/goimg?sslmode=disable" up

  # Verify migration status
  make migrate-status
  ```

- [ ] **Verify database schema**:
  ```bash
  docker exec -it goimg-postgres psql -U goimg -d goimg -c "\dt"
  # Should show all tables (users, images, variants, albums, etc.)
  ```

### 4. SSL/TLS Setup

- [ ] **SSL certificates obtained** (see [Prerequisites](#3-ssltls-certificates))
- [ ] **Certificate permissions verified**:
  ```bash
  sudo chmod 644 /etc/letsencrypt/live/api.example.com/fullchain.pem
  sudo chmod 600 /etc/letsencrypt/live/api.example.com/privkey.pem
  ```

- [ ] **Nginx configuration updated** with your domain:
  ```bash
  # Update domain in Nginx config
  sed -i 's/example\.com/api.example.com/g' docker/nginx/conf.d/api.conf
  ```

- [ ] **Auto-renewal configured**:
  ```bash
  sudo /home/user/goimg-datalayer/scripts/setup-ssl.sh --setup-renewal
  ```

### 5. Backup Configuration

- [ ] **Backup directory created**:
  ```bash
  sudo mkdir -p /var/backups/goimg
  sudo chmod 700 /var/backups/goimg
  ```

- [ ] **S3 backup bucket configured** (optional but recommended):
  ```bash
  # Configure in docker/docker-compose.prod.yml:
  # backup service environment:
  #   S3_ENDPOINT=s3.us-east-1.amazonaws.com
  #   S3_BUCKET=goimg-backups
  ```

- [ ] **GPG keys for backup encryption** (see [docs/operations/backup_restore.md](../operations/backup_restore.md)):
  ```bash
  gpg --gen-key  # Follow prompts
  gpg --export-secret-keys backup@goimg.local > /root/backup-gpg-key.asc
  ```

## Docker Compose Production Deployment

### 1. Review docker-compose.prod.yml

The production compose file is located at `/home/user/goimg-datalayer/docker/docker-compose.prod.yml`.

**Key features**:
- Docker Secrets for sensitive data (not environment variables)
- Resource limits for all services
- Network segmentation (frontend, backend, database isolated)
- Health checks for all services
- Logging with rotation
- Volume persistence

**Network Architecture**:
```
┌─────────────────────────────────────────────────────┐
│ frontend (172.20.0.0/24) - Public-facing            │
│  - nginx (SSL termination)                          │
│  - api (internal port 8080)                         │
│  - grafana (dashboard)                              │
│  - prometheus (metrics)                             │
└─────────────────────────────────────────────────────┘
                         │
┌─────────────────────────────────────────────────────┐
│ backend (172.21.0.0/24) - Internal application      │
│  - api                                              │
│  - worker                                           │
│  - redis                                            │
│  - clamav                                           │
│  - ipfs                                             │
│  - prometheus                                       │
│  - grafana                                          │
│  - backup (partial access to database)             │
└─────────────────────────────────────────────────────┘
                         │
┌─────────────────────────────────────────────────────┐
│ database (172.22.0.0/24) - Isolated, internal only  │
│  - postgres (NO external access)                    │
│  - backup (read-only database access)               │
└─────────────────────────────────────────────────────┘
```

### 2. Environment Configuration

Create production environment file:

```bash
cd /opt/goimg-datalayer/docker
cp .env.example .env.prod
```

**Edit `.env.prod`** with production values (see [docs/deployment/environment_variables.md](./environment_variables.md) for complete reference):

```bash
# Application
ENVIRONMENT=production
APP_NAME=goimg-api
LOG_LEVEL=info
LOG_FORMAT=json

# Database (uses Docker Secret for password)
DB_HOST=postgres
DB_PORT=5432
DB_USER=goimg
DB_NAME=goimg
DB_SSL_MODE=require  # Enable SSL for production

# Redis (uses Docker Secret for password)
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_DB=0

# API Server
API_PORT=8080
API_HOST=0.0.0.0
API_READ_TIMEOUT=30s
API_WRITE_TIMEOUT=30s
API_IDLE_TIMEOUT=120s

# CORS
CORS_ALLOWED_ORIGINS=https://api.example.com,https://www.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Accept,Authorization,Content-Type,X-CSRF-Token
CORS_ALLOW_CREDENTIALS=true

# Storage Provider (choose one)
STORAGE_PROVIDER=s3  # Options: local, s3, do-spaces, b2

# S3 Configuration (AWS, DigitalOcean Spaces, Backblaze B2)
S3_ENDPOINT=s3.us-east-1.amazonaws.com
S3_REGION=us-east-1
S3_BUCKET=goimg-production
S3_ACCESS_KEY=  # Use Docker Secret
S3_SECRET_KEY=  # Use Docker Secret
S3_USE_SSL=true

# IPFS
IPFS_ENABLED=true
IPFS_API_URL=http://ipfs:5001
IPFS_GATEWAY_URL=https://ipfs.io

# ClamAV
CLAMAV_HOST=clamav
CLAMAV_PORT=3310
CLAMAV_TIMEOUT=30s

# Image Processing
IMAGE_MAX_FILE_SIZE=10485760  # 10MB
IMAGE_ALLOWED_TYPES=image/jpeg,image/png,image/webp,image/gif
IMAGE_QUALITY=85
IMAGE_STRIP_METADATA=true

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Observability
METRICS_ENABLED=true
METRICS_PORT=9090
TRACING_ENABLED=false

# Backup
S3_ENDPOINT=s3.us-east-1.amazonaws.com
S3_BUCKET=goimg-backups
GPG_RECIPIENT=backup@goimg.local
DAILY_RETENTION_DAYS=7
WEEKLY_RETENTION_WEEKS=4
MONTHLY_RETENTION_MONTHS=6
```

### 3. Deploy Services

**Start all services**:
```bash
cd /opt/goimg-datalayer

# Pull latest images (if using registry)
docker-compose -f docker/docker-compose.prod.yml pull

# Start services
docker-compose -f docker/docker-compose.prod.yml up -d

# View logs
docker-compose -f docker/docker-compose.prod.yml logs -f
```

**Start services one by one** (recommended for first deployment):
```bash
# 1. Database and cache
docker-compose -f docker/docker-compose.prod.yml up -d postgres redis

# Wait for health checks to pass
docker-compose -f docker/docker-compose.prod.yml ps
# STATUS should show "healthy"

# 2. ClamAV and IPFS (these take time to initialize)
docker-compose -f docker/docker-compose.prod.yml up -d clamav ipfs

# Wait for ClamAV to update virus definitions (can take 5-10 minutes)
docker logs -f goimg-clamav
# Wait for: "Daemon started"

# 3. API and Worker
docker-compose -f docker/docker-compose.prod.yml up -d api worker

# 4. Monitoring stack
docker-compose -f docker/docker-compose.prod.yml up -d prometheus grafana

# 5. Nginx (SSL termination)
docker-compose -f docker/docker-compose.prod.yml up -d nginx

# 6. Backup service
docker-compose -f docker/docker-compose.prod.yml up -d backup
```

### 4. Resource Limits

All services have resource limits configured in `docker-compose.prod.yml`:

| Service | CPU Limit | Memory Limit | CPU Reservation | Memory Reservation |
|---------|-----------|--------------|-----------------|-------------------|
| nginx | 0.5 | 256MB | 0.1 | 64MB |
| api | 2.0 | 2GB | 0.5 | 512MB |
| worker | 4.0 | 4GB | 1.0 | 1GB |
| postgres | 2.0 | 2GB | 0.5 | 512MB |
| redis | 1.0 | 1GB | 0.25 | 256MB |
| clamav | 2.0 | 4GB | 0.5 | 2GB |
| ipfs | 2.0 | 2GB | 0.5 | 512MB |
| prometheus | 1.0 | 2GB | 0.25 | 512MB |
| grafana | 1.0 | 1GB | 0.25 | 256MB |
| backup | 0.5 | 512MB | 0.1 | 128MB |

**Monitor resource usage**:
```bash
docker stats
```

### 5. Volume Persistence

All data is persisted in Docker volumes:

```bash
# List volumes
docker volume ls | grep goimg

# Inspect volume (find mount point)
docker volume inspect goimg_postgres_data

# Backup volumes (see docs/operations/backup_restore.md)
```

**Critical volumes**:
- `postgres_data` - Database (CRITICAL - backup daily)
- `redis_data` - Cache/sessions (important - can be recreated)
- `api_uploads` - Uploaded images (CRITICAL - backup daily)
- `ipfs_data` - IPFS blocks (important - can be re-pinned)
- `backup_data` - Database backups (CRITICAL - sync to S3)

## Post-Deployment Validation

### 1. Health Check Verification

**Check all service health**:
```bash
docker-compose -f docker/docker-compose.prod.yml ps

# Expected output (all "healthy"):
# NAME              STATUS
# goimg-nginx       Up (healthy)
# goimg-api         Up (healthy)
# goimg-worker      Up (healthy)
# goimg-postgres    Up (healthy)
# goimg-redis       Up (healthy)
# goimg-clamav      Up (healthy)
# goimg-ipfs        Up (healthy)
# goimg-prometheus  Up (healthy)
# goimg-grafana     Up (healthy)
# goimg-backup      Up (healthy)
```

**Test health endpoints**:
```bash
# Internal health check (from server)
curl http://localhost:8080/health
# Expected: {"status":"healthy","database":"connected","redis":"connected"}

# External health check (HTTPS via nginx)
curl -I https://api.example.com/health
# Expected: HTTP/2 200
```

**Ready endpoint** (checks all dependencies):
```bash
curl https://api.example.com/health/ready
# Expected: {"status":"ready","checks":{"database":"ok","redis":"ok","clamav":"ok","ipfs":"ok"}}
```

### 2. SSL/TLS Validation

**Verify SSL certificate**:
```bash
openssl s_client -connect api.example.com:443 -servername api.example.com < /dev/null

# Check:
# - Certificate chain is valid
# - Issuer: Let's Encrypt
# - Expires: 90 days from issue date
```

**Test SSL Labs** (should achieve A+ rating):
```bash
# Visit: https://www.ssllabs.com/ssltest/analyze.html?d=api.example.com
# Expected: Overall Rating: A+
```

**Verify security headers**:
```bash
curl -I https://api.example.com/health | grep -E "Strict-Transport|X-Frame|X-Content-Type|Content-Security"

# Expected headers:
# Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
# X-Frame-Options: DENY
# X-Content-Type-Options: nosniff
# Content-Security-Policy: default-src 'self'
```

### 3. Database Connectivity

**Verify database connection**:
```bash
docker exec -it goimg-postgres psql -U goimg -d goimg -c "SELECT version();"
# Should show: PostgreSQL 16.x

# Check tables exist
docker exec -it goimg-postgres psql -U goimg -d goimg -c "\dt"
# Should list: users, images, variants, albums, tags, etc.

# Check migration status
docker exec -it goimg-api goose -dir /app/migrations status
```

### 4. Redis Connectivity

**Verify Redis connection**:
```bash
docker exec -it goimg-redis redis-cli -a "$(sudo cat /etc/goimg/secrets/redis_password)" ping
# Expected: PONG

# Check memory usage
docker exec -it goimg-redis redis-cli -a "$(sudo cat /etc/goimg/secrets/redis_password)" INFO memory
```

### 5. ClamAV Virus Scanner

**Verify ClamAV is running**:
```bash
docker logs goimg-clamav | grep "Daemon started"
# Expected: "Daemon started"

# Test virus scanning (API endpoint)
curl -X POST https://api.example.com/api/v1/health/clamav
# Expected: {"status":"healthy","version":"ClamAV 1.x.x"}
```

### 6. IPFS Node

**Verify IPFS is running**:
```bash
docker exec -it goimg-ipfs ipfs id
# Should show peer ID and addresses

# Test pinning (API endpoint)
curl https://api.example.com/api/v1/health/ipfs
# Expected: {"status":"healthy","peerID":"QmXXXXX..."}
```

### 7. Monitoring Setup

**Access Prometheus** (http://SERVER_IP:9091):
```bash
# Check targets are up
curl http://localhost:9091/api/v1/targets | jq '.data.activeTargets[] | {job, health}'

# Expected: All targets "up"
```

**Access Grafana** (http://SERVER_IP:3000):
```bash
# Default credentials (CHANGE IMMEDIATELY):
# Username: admin
# Password: (from /etc/goimg/secrets/grafana_admin_password)

# Import dashboards from monitoring/grafana/dashboards/
```

### 8. Log Verification

**Check API logs**:
```bash
docker logs --tail 100 goimg-api

# Should show:
# - "Starting API server" (startup)
# - "Database connection established" (database)
# - "Redis connection established" (redis)
# - "Server listening on :8080" (ready)
```

**Check for errors**:
```bash
docker-compose -f docker/docker-compose.prod.yml logs --tail 100 | grep -i error
# Should be empty or only expected errors
```

### 9. Functional Testing

**Test authentication flow**:
```bash
# Register new user
curl -X POST https://api.example.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "username": "testuser",
    "password": "TestPassword123!"
  }'

# Expected: {"id":"...","email":"test@example.com","username":"testuser"}

# Login
curl -X POST https://api.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "TestPassword123!"
  }'

# Expected: {"token":"eyJhbGc...","expiresIn":86400}
```

**Test image upload** (requires JWT token from login):
```bash
TOKEN="eyJhbGc..."  # From login response

curl -X POST https://api.example.com/api/v1/images \
  -H "Authorization: Bearer $TOKEN" \
  -F "image=@/path/to/test-image.jpg" \
  -F "title=Test Image" \
  -F "description=Production test upload" \
  -F "visibility=private"

# Expected: {"id":"...","status":"processing","message":"Image uploaded successfully"}
```

## Monitoring and Observability

### 1. Prometheus Metrics

**Key metrics to monitor**:

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `up{job="api"}` | API service availability | < 1 (down) |
| `http_requests_total` | Total HTTP requests | Rate change > 200% |
| `http_request_duration_seconds` | Request latency | p95 > 1s |
| `database_connections_active` | Active DB connections | > 80% of max |
| `redis_connected_clients` | Redis client count | > 100 |
| `image_processing_duration_seconds` | Image processing time | p95 > 30s |
| `clamav_scan_duration_seconds` | Virus scan duration | p95 > 10s |

**Access Prometheus**:
```bash
# Port forward if not exposed
ssh -L 9091:localhost:9091 user@SERVER_IP

# Open browser: http://localhost:9091
```

### 2. Grafana Dashboards

**Pre-configured dashboards** (in `monitoring/grafana/dashboards/`):

1. **API Overview**
   - Request rate, latency, error rate
   - Top endpoints by volume
   - HTTP status code distribution

2. **Database Performance**
   - Connection pool usage
   - Query duration (p50, p95, p99)
   - Slow queries
   - Cache hit rate

3. **Image Processing**
   - Upload rate
   - Processing queue depth
   - ClamAV scan results
   - IPFS pinning status

4. **Infrastructure**
   - CPU, memory, disk usage per service
   - Network I/O
   - Container health status

**Import dashboards**:
```bash
# Dashboards auto-provisioned from:
# monitoring/grafana/provisioning/dashboards/
```

### 3. Log Aggregation

**Centralized logging** (optional but recommended):

Install **Loki** for log aggregation:
```bash
# Add to docker-compose.prod.yml
loki:
  image: grafana/loki:latest
  ports:
    - "3100:3100"
  volumes:
    - loki_data:/loki
  command: -config.file=/etc/loki/local-config.yaml

promtail:
  image: grafana/promtail:latest
  volumes:
    - /var/lib/docker/containers:/var/lib/docker/containers:ro
    - /var/log:/var/log:ro
  command: -config.file=/etc/promtail/config.yml
```

**Query logs in Grafana** (Explore > Loki):
```promql
{container_name="goimg-api"} |= "error"
{container_name="goimg-api"} | json | level="error"
```

### 4. Alerting

**Configure Prometheus alerting** (`monitoring/prometheus/alerts.yml`):

```yaml
groups:
  - name: api
    interval: 30s
    rules:
      - alert: APIDown
        expr: up{job="api"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "API service is down"

      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "API latency p95 > 1s"

      - alert: DatabaseConnectionPoolExhausted
        expr: database_connections_active / database_connections_max > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Database connection pool > 80% utilized"
```

**Send alerts** (configure in `prometheus.yml`):
```yaml
alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - alertmanager:9093

# Or use Grafana alerting (recommended)
# Configure in Grafana UI: Alerting > Alert rules
```

## Maintenance and Operations

### 1. Backup Procedures

**Automated backups** (configured via backup service):

```bash
# Daily automated backups at 02:00 AM
# Retention:
#   - Daily: 7 days
#   - Weekly: 4 weeks
#   - Monthly: 6 months

# Manual backup
docker exec goimg-backup /backup.sh manual

# Verify backup exists
ls -lh /var/backups/goimg/
docker exec goimg-backup ls -lh /var/backups/postgres/

# Restore from backup (see docs/operations/backup_restore.md)
```

**S3 backup sync**:
```bash
# Backups automatically sync to S3 bucket
# Check S3 bucket
aws s3 ls s3://goimg-backups/postgres/
```

### 2. Secret Rotation

**Rotate secrets every 90 days** (see [docs/deployment/secrets.md](./secrets.md)):

```bash
# JWT secret rotation (invalidates all sessions!)
openssl rand -base64 64 | sudo tee /etc/goimg/secrets/jwt_secret > /dev/null
sudo chmod 600 /etc/goimg/secrets/jwt_secret
docker-compose -f docker/docker-compose.prod.yml restart api

# Database password rotation (zero-downtime)
# See docs/security/secret_rotation.md for detailed procedure
```

### 3. SSL Certificate Renewal

**Automatic renewal** (configured via systemd timer or cron):

```bash
# Check renewal timer status
sudo systemctl status certbot-renewal.timer

# Manual renewal test (dry run)
sudo /home/user/goimg-datalayer/scripts/setup-ssl.sh --renew --dry-run

# Force renewal
sudo /home/user/goimg-datalayer/scripts/setup-ssl.sh --renew

# Reload nginx to pick up new certificates
docker-compose -f docker/docker-compose.prod.yml exec nginx nginx -s reload
```

### 4. Database Maintenance

**Vacuum and analyze** (weekly):
```bash
docker exec -it goimg-postgres psql -U goimg -d goimg -c "VACUUM ANALYZE;"
```

**Reindex** (monthly):
```bash
docker exec -it goimg-postgres psql -U goimg -d goimg -c "REINDEX DATABASE goimg;"
```

**Check database size**:
```bash
docker exec -it goimg-postgres psql -U goimg -d goimg -c "
  SELECT pg_database.datname,
         pg_size_pretty(pg_database_size(pg_database.datname)) AS size
  FROM pg_database
  ORDER BY pg_database_size(pg_database.datname) DESC;
"
```

### 5. Log Rotation

**Docker logs** (configured in docker-compose.prod.yml):
```yaml
logging:
  driver: json-file
  options:
    max-size: "10m"  # 10MB per log file
    max-file: "3"    # Keep 3 rotated files (total: 30MB per container)
```

**Manual log cleanup**:
```bash
# Clear old logs
docker-compose -f docker/docker-compose.prod.yml logs --tail 0
```

### 6. Updates and Patches

**Update application**:
```bash
cd /opt/goimg-datalayer
git pull origin main  # Or specific release tag

# Rebuild images
make docker-build

# Run migrations
make migrate-up

# Rolling update (zero downtime)
docker-compose -f docker/docker-compose.prod.yml up -d --no-deps api worker

# Verify health
curl https://api.example.com/health
```

**Update dependencies** (Docker images):
```bash
docker-compose -f docker/docker-compose.prod.yml pull postgres redis clamav ipfs
docker-compose -f docker/docker-compose.prod.yml up -d postgres redis clamav ipfs
```

## Troubleshooting

### Service Won't Start

**Check logs**:
```bash
docker logs goimg-api --tail 100

# Common issues:
# 1. Missing secrets - check /run/secrets/ in container
# 2. Database connection failed - verify postgres is healthy
# 3. Port already in use - check with: sudo lsof -i :8080
```

**Verify health checks**:
```bash
docker inspect goimg-api | jq '.[0].State.Health'
```

### Database Connection Issues

**Check connectivity from API container**:
```bash
docker exec -it goimg-api nc -zv postgres 5432
# Expected: Connection to postgres 5432 port [tcp/postgresql] succeeded!

# Check database logs
docker logs goimg-postgres --tail 50
```

### SSL Certificate Issues

**Certificate not trusted**:
```bash
# Check certificate chain
openssl s_client -connect api.example.com:443 -showcerts

# Renew certificate
sudo /home/user/goimg-datalayer/scripts/setup-ssl.sh --renew
docker-compose -f docker/docker-compose.prod.yml restart nginx
```

### Out of Memory

**Check container memory usage**:
```bash
docker stats --no-stream

# Increase limits if needed (edit docker-compose.prod.yml)
# api:
#   deploy:
#     resources:
#       limits:
#         memory: 4G  # Increased from 2G
```

### Slow Performance

**Check resource usage**:
```bash
docker stats
htop
iotop
```

**Database performance**:
```bash
# Check slow queries
docker exec -it goimg-postgres psql -U goimg -d goimg -c "
  SELECT query, calls, total_time, mean_time
  FROM pg_stat_statements
  ORDER BY mean_time DESC
  LIMIT 10;
"

# Check connections
docker exec -it goimg-postgres psql -U goimg -d goimg -c "
  SELECT count(*) FROM pg_stat_activity;
"
```

## Rollback Procedures

### Application Rollback

**Rollback to previous version**:
```bash
cd /opt/goimg-datalayer

# 1. Checkout previous version
git log --oneline -10  # Find previous commit/tag
git checkout v1.0.0  # Or commit hash

# 2. Rebuild images
make docker-build

# 3. Rollback database migrations (if needed)
make migrate-down

# 4. Restart services
docker-compose -f docker/docker-compose.prod.yml up -d --force-recreate api worker

# 5. Verify health
curl https://api.example.com/health
```

### Database Rollback

**Restore from backup**:
```bash
# See docs/operations/backup_restore.md for detailed procedure

# Quick restore:
docker exec goimg-backup /restore.sh /var/backups/postgres/goimg-backup-YYYYMMDD-HHMMSS.sql.gz.gpg
```

## Security Hardening

### 1. Principle of Least Privilege

**Run containers as non-root** (where possible):
```yaml
# Add to docker-compose.prod.yml
api:
  user: "1000:1000"  # Non-root user
```

### 2. Network Isolation

**Database is isolated** (internal network only):
```yaml
networks:
  database:
    driver: bridge
    internal: true  # No external access
```

**Only nginx exposed to internet**:
```bash
# Firewall rules
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw deny 5432/tcp  # Block direct database access
sudo ufw deny 6379/tcp  # Block direct Redis access
```

### 3. Security Scanning

**Scan Docker images**:
```bash
# Install Trivy
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh

# Scan images for vulnerabilities
trivy image goimg-api:latest
trivy image goimg-worker:latest
```

### 4. Secrets Audit

**Verify no secrets in logs**:
```bash
docker-compose -f docker/docker-compose.prod.yml logs | grep -iE "password|secret|token|key"
# Should return no sensitive values
```

### 5. Regular Updates

**Keep base images updated**:
```bash
# Update monthly
docker-compose -f docker/docker-compose.prod.yml pull
docker-compose -f docker/docker-compose.prod.yml up -d
```

## Production Readiness Summary

After completing this deployment guide, your production environment should have:

- ✅ All services running with health checks passing
- ✅ SSL/TLS with A+ rating from SSL Labs
- ✅ Docker Secrets for all sensitive configuration
- ✅ Automated backups with S3 sync
- ✅ Prometheus metrics collection
- ✅ Grafana dashboards configured
- ✅ SSL auto-renewal configured
- ✅ Network segmentation enforced
- ✅ Resource limits configured
- ✅ Log rotation enabled

## References

- [Environment Variables Reference](./environment_variables.md)
- [Secret Management Guide](./secrets.md)
- [SSL/TLS Setup Guide](./ssl.md)
- [Backup and Restore Procedures](../operations/backup_restore.md)
- [Security Gates Verification](../../claude/security_gates.md)
- [CDN Configuration](./cdn.md)

---

**Document Version**: 1.0
**Last Updated**: 2025-12-07 (Sprint 9 - Task 1.2)
**Security Gate**: S9-PROD - VERIFIED (LAUNCH READY)
**Next Review**: After first production deployment

**Deployment Checklist**: See [Pre-Deployment Checklist](#pre-deployment-checklist)
