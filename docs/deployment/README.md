# Production Deployment Guide

Complete guide for deploying goimg-datalayer to production.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Infrastructure Setup](#infrastructure-setup)
- [SSL Certificate Setup](#ssl-certificate-setup)
- [Environment Configuration](#environment-configuration)
- [Database Setup](#database-setup)
- [Docker Deployment](#docker-deployment)
- [Monitoring & Logging](#monitoring--logging)
- [Backup & Restore](#backup--restore)
- [Security Hardening](#security-hardening)
- [Troubleshooting](#troubleshooting)
- [Updating & Maintenance](#updating--maintenance)

## Prerequisites

### Server Requirements

**Minimum:**
- 2 CPU cores
- 4 GB RAM
- 50 GB SSD storage
- Ubuntu 22.04 LTS or Debian 12

**Recommended:**
- 4 CPU cores
- 8 GB RAM
- 100 GB SSD storage
- Ubuntu 22.04 LTS

### Software Requirements

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Install Docker Compose
sudo apt install docker-compose-plugin -y

# Install additional tools
sudo apt install -y \
    git \
    make \
    curl \
    certbot \
    postgresql-client \
    awscli \
    gpg
```

### Domain & DNS

1. Register a domain name
2. Point A record to your server's IP address:
   ```
   yourdomain.com       A    123.45.67.89
   www.yourdomain.com   A    123.45.67.89
   ```
3. Wait for DNS propagation (5-60 minutes)

## Infrastructure Setup

### 1. Clone Repository

```bash
cd /opt
sudo git clone https://github.com/yourusername/goimg-datalayer.git
sudo chown -R $USER:$USER goimg-datalayer
cd goimg-datalayer
```

### 2. Create Data Directories

```bash
sudo mkdir -p /var/lib/goimg/{postgres,backups,uploads}
sudo chown -R 10001:10001 /var/lib/goimg
sudo chmod 750 /var/lib/goimg
```

### 3. Storage Backend Setup

Choose one of the following storage backends:

#### Option A: AWS S3

```bash
# Create S3 bucket
aws s3 mb s3://your-goimg-production-bucket --region us-east-1

# Enable versioning
aws s3api put-bucket-versioning \
    --bucket your-goimg-production-bucket \
    --versioning-configuration Status=Enabled

# Set lifecycle policy (optional - for cost optimization)
cat > lifecycle.json << 'EOF'
{
  "Rules": [
    {
      "Id": "DeleteOldVersions",
      "Status": "Enabled",
      "NoncurrentVersionExpiration": {
        "NoncurrentDays": 30
      }
    }
  ]
}
EOF

aws s3api put-bucket-lifecycle-configuration \
    --bucket your-goimg-production-bucket \
    --lifecycle-configuration file://lifecycle.json
```

#### Option B: DigitalOcean Spaces

```bash
# Install doctl (DigitalOcean CLI)
cd /tmp
wget https://github.com/digitalocean/doctl/releases/download/v1.98.1/doctl-1.98.1-linux-amd64.tar.gz
tar xf doctl-*.tar.gz
sudo mv doctl /usr/local/bin

# Create Spaces bucket
doctl spaces create your-goimg-production-bucket --region nyc3
```

#### Option C: Backblaze B2

```bash
# Install b2 CLI
pip3 install b2

# Create bucket via Backblaze web UI
# Or use b2 CLI after authentication
```

### 4. IPFS Pinning Service (Optional)

For production IPFS, use a pinning service:

#### Pinata Setup

1. Sign up at https://pinata.cloud
2. Create API key with permissions:
   - `pinFileToIPFS`
   - `pinJSONToIPFS`
   - `unpin`
3. Copy JWT token for environment configuration

#### Infura IPFS Setup

1. Sign up at https://infura.io
2. Create IPFS project
3. Copy Project ID and Secret

## SSL Certificate Setup

### Option 1: Let's Encrypt (Recommended)

```bash
# Create webroot directory for challenges
sudo mkdir -p /var/www/certbot
sudo chown -R www-data:www-data /var/www/certbot

# Obtain certificate
sudo certbot certonly --webroot \
    -w /var/www/certbot \
    -d yourdomain.com \
    -d www.yourdomain.com \
    --email admin@yourdomain.com \
    --agree-tos \
    --no-eff-email

# Copy certificates to nginx ssl directory
sudo cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem \
    /opt/goimg-datalayer/docker/nginx/ssl/fullchain.pem

sudo cp /etc/letsencrypt/live/yourdomain.com/privkey.pem \
    /opt/goimg-datalayer/docker/nginx/ssl/privkey.pem

# Set correct permissions
sudo chown 101:101 /opt/goimg-datalayer/docker/nginx/ssl/*.pem
sudo chmod 644 /opt/goimg-datalayer/docker/nginx/ssl/fullchain.pem
sudo chmod 600 /opt/goimg-datalayer/docker/nginx/ssl/privkey.pem

# Setup auto-renewal
sudo crontab -e
# Add this line:
0 0 * * * certbot renew --quiet --post-hook "docker exec goimg-nginx nginx -s reload"
```

### Option 2: CloudFlare Origin Certificate

1. Log in to CloudFlare dashboard
2. Go to SSL/TLS → Origin Server
3. Create certificate
4. Copy certificate and private key
5. Save to nginx ssl directory

### Option 3: Self-Signed (Development Only)

```bash
cd /opt/goimg-datalayer/docker/nginx/ssl

openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout privkey.pem \
    -out fullchain.pem \
    -subj "/C=US/ST=State/L=City/O=Org/CN=localhost"
```

## Environment Configuration

### 1. Create Production Environment File

```bash
cd /opt/goimg-datalayer/docker
cp .env.prod.example .env.prod
```

### 2. Generate Secrets

```bash
# Generate JWT secret
echo "JWT_SECRET=$(openssl rand -base64 64)" >> .env.prod

# Generate database password
echo "DB_PASSWORD=$(openssl rand -base64 32)" >> .env.prod

# Generate Redis password
echo "REDIS_PASSWORD=$(openssl rand -base64 32)" >> .env.prod
```

### 3. Edit Configuration

```bash
nano .env.prod
```

**Critical values to update:**

```bash
# Database
DB_USER=goimg_prod
DB_PASSWORD=<generated-password>
DB_NAME=goimg_production

# Redis
REDIS_PASSWORD=<generated-password>

# JWT
JWT_SECRET=<generated-secret>

# CORS
CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com

# Storage (choose one)
STORAGE_PROVIDER=s3
S3_BUCKET=your-goimg-production-bucket
S3_ACCESS_KEY=<your-access-key>
S3_SECRET_KEY=<your-secret-key>

# IPFS (if using Pinata)
IPFS_PINATA_JWT=<your-pinata-jwt>

# Data paths
DATA_ROOT=/var/lib/goimg
```

### 4. Secure Environment File

```bash
chmod 600 .env.prod
```

## Database Setup

### 1. Initialize Database

```bash
cd /opt/goimg-datalayer

# Start only PostgreSQL first
docker-compose -f docker/docker-compose.prod.yml up -d postgres

# Wait for PostgreSQL to be ready
sleep 10

# Run migrations
export $(cat docker/.env.prod | xargs)
make migrate-up
```

### 2. Verify Database

```bash
docker exec -it goimg-postgres psql -U goimg_prod -d goimg_production -c '\dt'
```

## Docker Deployment

### 1. Build Images

```bash
cd /opt/goimg-datalayer

# Build API and worker images
docker build -f docker/Dockerfile.api -t goimg-api:latest .
docker build -f docker/Dockerfile.worker -t goimg-worker:latest .
```

### 2. Start All Services

```bash
# Load environment variables
export $(cat docker/.env.prod | grep -v '^#' | xargs)

# Start all services
docker-compose -f docker/docker-compose.prod.yml --env-file docker/.env.prod up -d

# Check service status
docker-compose -f docker/docker-compose.prod.yml ps
```

### 3. Verify Deployment

```bash
# Check logs
docker-compose -f docker/docker-compose.prod.yml logs -f api

# Check health endpoints
curl http://localhost:8080/health
curl https://yourdomain.com/health

# Check metrics (from allowed IP)
curl http://localhost:9090/metrics
```

### 4. Enable IPFS (Optional)

```bash
# Start IPFS service
docker-compose -f docker/docker-compose.prod.yml --profile ipfs up -d ipfs
```

## Monitoring & Logging

### 1. View Logs

```bash
# All services
docker-compose -f docker/docker-compose.prod.yml logs -f

# Specific service
docker-compose -f docker/docker-compose.prod.yml logs -f api
docker-compose -f docker/docker-compose.prod.yml logs -f worker
docker-compose -f docker/docker-compose.prod.yml logs -f nginx

# Last 100 lines
docker-compose -f docker/docker-compose.prod.yml logs --tail=100
```

### 2. Resource Usage

```bash
# Container stats
docker stats

# Disk usage
docker system df
df -h /var/lib/goimg
```

### 3. Health Checks

```bash
# Check all container health
docker ps --format "table {{.Names}}\t{{.Status}}"

# API health
curl https://yourdomain.com/health

# Database health
docker exec goimg-postgres pg_isready -U goimg_prod
```

### 4. Application Metrics

Metrics are exposed at `http://localhost:9090/metrics` (restricted to internal IPs).

**Key metrics to monitor:**
- HTTP request rate and latency
- Database connection pool usage
- Redis cache hit rate
- Worker queue length
- ClamAV scan duration
- IPFS pin success rate

### 5. Set Up External Monitoring (Optional)

**Prometheus + Grafana:**

```bash
# Add to docker-compose.prod.yml
# See: https://prometheus.io/docs/prometheus/latest/installation/
```

**Cloud Monitoring:**
- AWS CloudWatch
- DigitalOcean Monitoring
- Datadog
- New Relic

## Backup & Restore

### 1. Manual Backup

```bash
cd /opt/goimg-datalayer

# Backup to local file
./scripts/backup-db.sh \
    -d goimg_production \
    -u goimg_prod \
    -p <db-password> \
    -o /var/lib/goimg/backups

# Backup with encryption
./scripts/backup-db.sh \
    -d goimg_production \
    -u goimg_prod \
    -e backup@yourdomain.com

# Backup to S3
./scripts/backup-db.sh \
    -d goimg_production \
    -u goimg_prod \
    -s your-backup-bucket \
    --s3-region us-east-1
```

### 2. Automated Backups

```bash
# Add to crontab
sudo crontab -e

# Daily backup at 2 AM
0 2 * * * cd /opt/goimg-datalayer && ./scripts/backup-db.sh -d goimg_production -u goimg_prod -s your-backup-bucket >> /var/log/goimg-backup.log 2>&1
```

### 3. Restore from Backup

```bash
# Restore from local file
./scripts/restore-db.sh \
    -f /var/lib/goimg/backups/goimg_production_20240101_120000.sql.gz \
    -d goimg_production \
    -u goimg_prod \
    --force

# Restore from S3
./scripts/restore-db.sh \
    -s s3://your-backup-bucket/postgres-backups/goimg_production_20240101_120000.sql.gz \
    -d goimg_production \
    -u goimg_prod
```

### 4. Backup Storage (Images)

If using local storage (not recommended), backup upload directory:

```bash
# Sync to S3
aws s3 sync /var/lib/goimg/uploads s3://your-backup-bucket/uploads/

# Or use rsync to remote server
rsync -avz /var/lib/goimg/uploads/ user@backup-server:/backups/goimg-uploads/
```

## Security Hardening

### 1. Firewall Configuration

```bash
# Install UFW
sudo apt install ufw

# Default policies
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH (IMPORTANT: Do this first!)
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Enable firewall
sudo ufw enable

# Check status
sudo ufw status verbose
```

### 2. Fail2Ban (Brute Force Protection)

```bash
# Install
sudo apt install fail2ban

# Configure
sudo cp /etc/fail2ban/jail.conf /etc/fail2ban/jail.local
sudo nano /etc/fail2ban/jail.local

# Add nginx rate limiting jail
cat | sudo tee /etc/fail2ban/jail.d/nginx-limit.conf << 'EOF'
[nginx-limit-req]
enabled = true
filter = nginx-limit-req
action = iptables-multiport[name=ReqLimit, port="http,https", protocol=tcp]
logpath = /var/log/nginx/error.log
findtime = 600
bantime = 7200
maxretry = 10
EOF

# Restart
sudo systemctl restart fail2ban
```

### 3. System Updates

```bash
# Enable automatic security updates
sudo apt install unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades
```

### 4. Docker Security

```bash
# Enable Docker content trust
export DOCKER_CONTENT_TRUST=1

# Scan images for vulnerabilities
docker scout cves goimg-api:latest
docker scout cves goimg-worker:latest
```

### 5. Secrets Management

For production, consider using a secrets manager:

**AWS Secrets Manager:**
```bash
# Store secret
aws secretsmanager create-secret \
    --name goimg/prod/db-password \
    --secret-string "<password>"

# Retrieve secret
aws secretsmanager get-secret-value \
    --secret-id goimg/prod/db-password \
    --query SecretString \
    --output text
```

**HashiCorp Vault:**
```bash
# Write secret
vault kv put secret/goimg/prod db_password="<password>"

# Read secret
vault kv get -field=db_password secret/goimg/prod
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs goimg-api
docker logs goimg-postgres

# Check resource usage
docker stats

# Verify environment variables
docker exec goimg-api env | grep DB_
```

### Database Connection Failed

```bash
# Check PostgreSQL is running
docker exec goimg-postgres pg_isready

# Test connection
docker exec goimg-postgres psql -U goimg_prod -d goimg_production -c 'SELECT 1;'

# Check network
docker network inspect goimg-network
```

### SSL Certificate Issues

```bash
# Verify certificate files
ls -l /opt/goimg-datalayer/docker/nginx/ssl/

# Test SSL configuration
openssl s_client -connect yourdomain.com:443 -servername yourdomain.com

# Check nginx config
docker exec goimg-nginx nginx -t

# Reload nginx
docker exec goimg-nginx nginx -s reload
```

### High Memory Usage

```bash
# Identify culprit
docker stats --no-stream

# Check container limits
docker inspect goimg-api | grep -A 10 Memory

# Restart heavy containers
docker-compose -f docker/docker-compose.prod.yml restart worker
```

### ClamAV Not Starting

ClamAV requires ~2 GB RAM and takes 3-5 minutes to start:

```bash
# Check logs
docker logs goimg-clamav

# Wait for database update
docker exec goimg-clamav tail -f /var/log/clamav/freshclam.log

# Check health
docker exec goimg-clamav clamdcheck.sh
```

### Worker Not Processing Jobs

```bash
# Check worker logs
docker logs goimg-worker

# Check Redis connection
docker exec goimg-redis redis-cli -a <redis-password> PING

# List queues
docker exec goimg-redis redis-cli -a <redis-password> KEYS '*'
```

## Updating & Maintenance

### 1. Update Application

```bash
cd /opt/goimg-datalayer

# Pull latest code
git pull origin main

# Backup database before update
./scripts/backup-db.sh -d goimg_production -u goimg_prod -s your-backup-bucket

# Run new migrations
make migrate-up

# Rebuild images
docker build -f docker/Dockerfile.api -t goimg-api:latest .
docker build -f docker/Dockerfile.worker -t goimg-worker:latest .

# Recreate containers (zero downtime with rolling restart)
docker-compose -f docker/docker-compose.prod.yml up -d --no-deps --build api
docker-compose -f docker/docker-compose.prod.yml up -d --no-deps --build worker

# Verify update
curl https://yourdomain.com/health
```

### 2. Update Docker Images

```bash
# Pull new base images
docker pull postgres:16-alpine
docker pull redis:7-alpine
docker pull nginx:1.25-alpine
docker pull clamav/clamav:1.3

# Recreate containers
docker-compose -f docker/docker-compose.prod.yml up -d --force-recreate
```

### 3. Database Maintenance

```bash
# Vacuum database
docker exec goimg-postgres psql -U goimg_prod -d goimg_production -c 'VACUUM ANALYZE;'

# Check database size
docker exec goimg-postgres psql -U goimg_prod -d goimg_production -c '\l+'

# Check table sizes
docker exec goimg-postgres psql -U goimg_prod -d goimg_production -c "
SELECT relname AS table, pg_size_pretty(pg_total_relation_size(relid)) AS size
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC;"
```

### 4. Clean Up

```bash
# Remove old images
docker image prune -a

# Remove unused volumes
docker volume prune

# Clean build cache
docker builder prune
```

### 5. Scale Services

```bash
# Scale workers
docker-compose -f docker/docker-compose.prod.yml up -d --scale worker=3

# Scale API (requires load balancer)
docker-compose -f docker/docker-compose.prod.yml up -d --scale api=2
```

## Performance Tuning

### PostgreSQL

Edit PostgreSQL configuration for better performance:

```bash
# Edit postgresql.conf
docker exec -it goimg-postgres vi /var/lib/postgresql/data/pgdata/postgresql.conf

# Recommended settings (adjust based on server specs):
shared_buffers = 2GB
effective_cache_size = 6GB
work_mem = 16MB
maintenance_work_mem = 512MB
max_connections = 100
```

### Redis

Already tuned in docker-compose.prod.yml. Monitor with:

```bash
docker exec goimg-redis redis-cli -a <password> INFO stats
```

### Nginx

Caching is configured. Monitor cache hit rate:

```bash
docker exec goimg-nginx tail -f /var/log/nginx/access.log | grep X-Cache-Status
```

## Support & Resources

- **Documentation**: `/docs/`
- **Issues**: GitHub Issues
- **Security**: security@yourdomain.com

## Next Steps

1. Set up monitoring (Prometheus/Grafana)
2. Configure CDN (CloudFlare/CloudFront)
3. Set up CI/CD pipeline
4. Performance testing
5. Disaster recovery plan
