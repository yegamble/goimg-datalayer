# Environment Variables Reference Guide

> Complete catalog of environment variables for goimg-datalayer configuration
>
> **Sprint 9 - Task 1.4**: Environment Configuration Documentation

This guide provides a comprehensive reference for all environment variables used in the goimg-datalayer application. Variables are organized by category with descriptions, default values, and security implications.

## Table of Contents

- [Quick Start](#quick-start)
- [Application Settings](#application-settings)
- [Database Configuration](#database-configuration)
- [Redis Configuration](#redis-configuration)
- [JWT Authentication](#jwt-authentication)
- [API Server Configuration](#api-server-configuration)
- [CORS Configuration](#cors-configuration)
- [Storage Providers](#storage-providers)
- [IPFS Configuration](#ipfs-configuration)
- [ClamAV Antivirus](#clamav-antivirus)
- [Image Processing](#image-processing)
- [Rate Limiting](#rate-limiting)
- [Worker Configuration](#worker-configuration)
- [Logging Configuration](#logging-configuration)
- [Observability](#observability)
- [Example Configuration Files](#example-configuration-files)
- [Troubleshooting](#troubleshooting)

## Quick Start

**Development**: Use `.env` file in `docker/` directory:
```bash
cd docker
cp .env.example .env
# Edit .env with your local settings
docker-compose up -d
```

**Production**: Use Docker Secrets (recommended) or `.env.prod`:
```bash
# Set SECRET_PROVIDER=docker to use Docker Secrets
export SECRET_PROVIDER=docker

# Sensitive values stored in /etc/goimg/secrets/
# See docs/deployment/secrets.md
```

## Application Settings

Configuration for application-wide behavior.

| Variable | Description | Type | Default | Required | Example |
|----------|-------------|------|---------|----------|---------|
| `APP_NAME` | Application name for logging/metrics | String | `goimg-api` | No | `goimg-api` |
| `APP_ENV` | Environment name | String | `development` | No | `production`, `staging`, `development` |
| `ENVIRONMENT` | Alias for APP_ENV | String | `development` | No | `production` |
| `SECRET_PROVIDER` | Secret management backend | String | `env` | No | `env`, `docker`, `vault` |

**Security Notes**:
- Set `APP_ENV=production` in production to disable debug features
- Use `SECRET_PROVIDER=docker` in production (not `env`)

## Database Configuration

PostgreSQL database connection settings.

| Variable | Description | Type | Default | Required | Example |
|----------|-------------|------|---------|----------|---------|
| `DB_HOST` | Database hostname | String | `localhost` | Yes | `postgres`, `db.example.com` |
| `DB_PORT` | Database port | Integer | `5432` | Yes | `5432` |
| `DB_USER` | Database username | String | `goimg` | Yes | `goimg` |
| `DB_PASSWORD` | Database password | Secret | - | Yes | Use Docker Secret |
| `DB_NAME` | Database name | String | `goimg` | Yes | `goimg` |
| `DB_SSL_MODE` | SSL/TLS mode | String | `disable` | No | `disable`, `require`, `verify-full` |

### Connection Pooling

| Variable | Description | Type | Default | Example |
|----------|-------------|------|---------|---------|
| `DB_MAX_OPEN_CONNS` | Maximum open connections | Integer | `25` | `50`, `100` |
| `DB_MAX_IDLE_CONNS` | Maximum idle connections | Integer | `5` | `10`, `25` |
| `DB_CONN_MAX_LIFETIME` | Connection max lifetime | Duration | `5m` | `5m`, `15m`, `1h` |

### Performance Tuning (PostgreSQL)

| Variable | Description | Type | Default | Example |
|----------|-------------|------|---------|---------|
| `DB_MAX_CONNECTIONS` | PostgreSQL max_connections | Integer | `100` | `200`, `500` |
| `DB_SHARED_BUFFERS` | PostgreSQL shared_buffers | Size | `256MB` | `512MB`, `1GB`, `4GB` |
| `DB_EFFECTIVE_CACHE_SIZE` | PostgreSQL effective_cache_size | Size | `1GB` | `4GB`, `16GB` |
| `DB_WORK_MEM` | PostgreSQL work_mem | Size | `4MB` | `16MB`, `64MB` |
| `DB_MAINTENANCE_WORK_MEM` | PostgreSQL maintenance_work_mem | Size | `64MB` | `256MB`, `512MB` |

**Security Notes**:
- **NEVER set `DB_PASSWORD` in `.env` for production** - use Docker Secrets
- Use `DB_SSL_MODE=require` in production
- Use `DB_SSL_MODE=verify-full` for highest security (requires CA certificate)

**Performance Recommendations**:
- **Shared buffers**: 25% of system RAM (max 8GB)
- **Effective cache size**: 50-75% of system RAM
- **Work mem**: Total RAM / max_connections / 4
- **Max connections**: Based on load (100-500 typical)

## Redis Configuration

Redis cache and session storage settings.

| Variable | Description | Type | Default | Required | Example |
|----------|-------------|------|---------|----------|---------|
| `REDIS_HOST` | Redis hostname | String | `localhost` | Yes | `redis`, `cache.example.com` |
| `REDIS_PORT` | Redis port | Integer | `6379` | Yes | `6379` |
| `REDIS_PASSWORD` | Redis authentication password | Secret | - | No (dev), Yes (prod) | Use Docker Secret |
| `REDIS_DB` | Redis database number | Integer | `0` | No | `0`, `1`, `2` |
| `REDIS_URL` | Redis connection URL (alternative) | String | - | No | `redis://localhost:6379/0` |
| `REDIS_MAX_RETRIES` | Connection retry attempts | Integer | `3` | No | `3`, `5` |
| `REDIS_MAX_MEMORY` | Redis maxmemory limit | Size | `512mb` | No | `512mb`, `1gb`, `2gb` |

**Security Notes**:
- **ALWAYS set `REDIS_PASSWORD` in production**
- Redis without password is acceptable ONLY for local development
- Never expose Redis port (6379) to internet

**Performance Recommendations**:
- Set `REDIS_MAX_MEMORY` to prevent OOM
- Use `allkeys-lru` eviction policy for cache
- Monitor memory usage via `INFO memory` command

## JWT Authentication

JSON Web Token configuration for user authentication.

| Variable | Description | Type | Default | Required | Example |
|----------|-------------|------|---------|----------|---------|
| `JWT_SECRET` | HS256 symmetric secret | Secret | - | Yes (MVP) | Use Docker Secret (64+ chars) |
| `JWT_PRIVATE_KEY_PATH` | RS256 private key path | Path | - | Yes (recommended) | `/run/secrets/JWT_PRIVATE_KEY` |
| `JWT_PUBLIC_KEY_PATH` | RS256 public key path | Path | - | Yes (recommended) | `/run/secrets/JWT_PUBLIC_KEY` |
| `JWT_EXPIRATION` | Token expiration duration | Duration | `24h` | No | `15m`, `1h`, `24h` |
| `JWT_REFRESH_EXPIRATION` | Refresh token expiration | Duration | `168h` | No | `7d`, `30d` |

**Security Notes**:
- **MVP uses HS256** (`JWT_SECRET`) - symmetric key
- **Production should use RS256** (`JWT_PRIVATE_KEY_PATH`, `JWT_PUBLIC_KEY_PATH`) - asymmetric keypair
- **JWT_SECRET**: Minimum 64 characters, cryptographically random
- **RS256 keys**: 4096-bit RSA keypair (see [docs/deployment/secrets.md](./secrets.md))
- Rotate JWT keys every 6 months

**Migration Path** (HS256 → RS256):
```bash
# Generate 4096-bit RSA keypair
openssl genrsa -out /etc/goimg/secrets/jwt_private.pem 4096
openssl rsa -in /etc/goimg/secrets/jwt_private.pem -pubout -out /etc/goimg/secrets/jwt_public.pem
chmod 600 /etc/goimg/secrets/jwt_private.pem
chmod 644 /etc/goimg/secrets/jwt_public.pem

# Update environment variables
JWT_PRIVATE_KEY_PATH=/run/secrets/JWT_PRIVATE_KEY
JWT_PUBLIC_KEY_PATH=/run/secrets/JWT_PUBLIC_KEY
# Remove JWT_SECRET
```

## API Server Configuration

HTTP server settings for the API service.

| Variable | Description | Type | Default | Required | Example |
|----------|-------------|------|---------|----------|---------|
| `API_PORT` | HTTP server port | Integer | `8080` | No | `8080`, `3000` |
| `HTTP_PORT` | Alias for API_PORT | Integer | `8080` | No | `8080` |
| `API_HOST` | Bind address | String | `0.0.0.0` | No | `0.0.0.0`, `127.0.0.1` |
| `API_READ_TIMEOUT` | Read timeout | Duration | `30s` | No | `30s`, `60s` |
| `API_WRITE_TIMEOUT` | Write timeout | Duration | `30s` | No | `30s`, `60s` |
| `API_IDLE_TIMEOUT` | Idle connection timeout | Duration | `120s` | No | `120s`, `300s` |
| `API_MAX_HEADER_BYTES` | Max header size | Integer | `1048576` | No | `1048576` (1MB) |

**Recommendations**:
- Use `0.0.0.0` to accept connections from all interfaces (required in Docker)
- Use `127.0.0.1` for local-only access (development)
- Set timeouts based on expected workload:
  - Image upload: `API_WRITE_TIMEOUT=60s` or higher
  - Normal API requests: `API_READ_TIMEOUT=30s`

## CORS Configuration

Cross-Origin Resource Sharing settings.

| Variable | Description | Type | Default | Example |
|----------|-------------|------|---------|---------|
| `CORS_ALLOWED_ORIGINS` | Allowed origins (comma-separated) | String | `*` | `https://example.com,https://www.example.com` |
| `CORS_ALLOWED_METHODS` | Allowed HTTP methods | String | `GET,POST,PUT,DELETE,OPTIONS` | `GET,POST,PUT,DELETE,OPTIONS` |
| `CORS_ALLOWED_HEADERS` | Allowed headers | String | `Accept,Authorization,Content-Type` | `Accept,Authorization,Content-Type,X-CSRF-Token` |
| `CORS_ALLOW_CREDENTIALS` | Allow credentials (cookies) | Boolean | `true` | `true`, `false` |

**Security Notes**:
- **NEVER use `*` (wildcard) in production** for `CORS_ALLOWED_ORIGINS`
- List specific domains only
- Set `CORS_ALLOW_CREDENTIALS=true` only if using cookies/sessions

**Example (Production)**:
```bash
CORS_ALLOWED_ORIGINS=https://api.example.com,https://www.example.com,https://app.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Accept,Authorization,Content-Type,X-CSRF-Token
CORS_ALLOW_CREDENTIALS=true
```

## Storage Providers

Object storage configuration for uploaded images.

### Storage Provider Selection

| Variable | Description | Type | Default | Required | Options |
|----------|-------------|------|---------|----------|---------|
| `STORAGE_PROVIDER` | Storage backend | String | `local` | Yes | `local`, `s3`, `do-spaces`, `b2` |

### Local Storage (Development Only)

| Variable | Description | Type | Default | Example |
|----------|-------------|------|---------|---------|
| `STORAGE_LOCAL_PATH` | Local filesystem path | Path | `/tmp/goimg-uploads` | `/var/lib/goimg/uploads` |

**WARNING**: Local storage NOT recommended for production (no redundancy, scaling issues)

### S3-Compatible Storage (AWS S3, DigitalOcean Spaces, Backblaze B2)

| Variable | Description | Type | Default | Required | Example |
|----------|-------------|------|---------|----------|---------|
| `S3_ENDPOINT` | S3 API endpoint | String | - | Yes | See below |
| `S3_REGION` | S3 region | String | `us-east-1` | Yes | `us-east-1`, `nyc3` |
| `S3_BUCKET` | S3 bucket name | String | - | Yes | `goimg-production` |
| `S3_ACCESS_KEY` | S3 access key ID | Secret | - | Yes | Use Docker Secret |
| `S3_SECRET_KEY` | S3 secret access key | Secret | - | Yes | Use Docker Secret |
| `S3_USE_SSL` | Use HTTPS for S3 | Boolean | `true` | No | `true`, `false` |

**Provider-Specific Endpoints**:

| Provider | S3_ENDPOINT | S3_REGION | Example |
|----------|-------------|-----------|---------|
| **AWS S3** | `s3.amazonaws.com` or `s3.REGION.amazonaws.com` | AWS region | `s3.us-east-1.amazonaws.com` |
| **DigitalOcean Spaces** | `REGION.digitaloceanspaces.com` | DO region | `nyc3.digitaloceanspaces.com` |
| **Backblaze B2** | `s3.REGION.backblazeb2.com` | B2 region | `s3.us-west-004.backblazeb2.com` |
| **MinIO (self-hosted)** | `minio.example.com:9000` | Custom | `localhost:9000` |

**Examples**:

AWS S3:
```bash
STORAGE_PROVIDER=s3
S3_ENDPOINT=s3.us-east-1.amazonaws.com
S3_REGION=us-east-1
S3_BUCKET=goimg-production
S3_ACCESS_KEY=  # Use Docker Secret: /run/secrets/S3_ACCESS_KEY
S3_SECRET_KEY=  # Use Docker Secret: /run/secrets/S3_SECRET_KEY
S3_USE_SSL=true
```

DigitalOcean Spaces:
```bash
STORAGE_PROVIDER=do-spaces
S3_ENDPOINT=nyc3.digitaloceanspaces.com
S3_REGION=us-east-1  # Always use us-east-1 for DO Spaces
S3_BUCKET=goimg-production
S3_ACCESS_KEY=  # Use Docker Secret
S3_SECRET_KEY=  # Use Docker Secret
S3_USE_SSL=true
```

Backblaze B2:
```bash
STORAGE_PROVIDER=b2
S3_ENDPOINT=s3.us-west-004.backblazeb2.com
S3_REGION=us-west-004
S3_BUCKET=goimg-production
S3_ACCESS_KEY=  # Use Docker Secret
S3_SECRET_KEY=  # Use Docker Secret
S3_USE_SSL=true
```

**Security Notes**:
- **NEVER commit `S3_ACCESS_KEY` or `S3_SECRET_KEY` to version control**
- Use Docker Secrets in production
- Restrict S3 bucket access with IAM policies
- Enable bucket versioning for disaster recovery

## IPFS Configuration

InterPlanetary File System (decentralized storage) settings.

| Variable | Description | Type | Default | Required | Example |
|----------|-------------|------|---------|----------|---------|
| `IPFS_ENABLED` | Enable IPFS storage | Boolean | `false` | No | `true`, `false` |
| `IPFS_API_URL` | IPFS Kubo API endpoint | URL | `http://localhost:5001` | Yes (if enabled) | `http://ipfs:5001` |
| `IPFS_GATEWAY_URL` | IPFS HTTP gateway | URL | `https://ipfs.io` | No | `https://ipfs.io`, `http://localhost:8080` |

### IPFS Pinning Services (Optional)

| Variable | Description | Type | Required | Example |
|----------|-------------|------|----------|---------|
| `IPFS_PINATA_JWT` | Pinata JWT token | Secret | No | Use Docker Secret |
| `IPFS_INFURA_PROJECT_ID` | Infura IPFS project ID | String | No | Use Docker Secret |
| `IPFS_INFURA_PROJECT_SECRET` | Infura IPFS project secret | Secret | No | Use Docker Secret |

**Use Cases**:
- **Local IPFS node**: Development, self-hosted deployments
- **Pinata**: Production pinning service (recommended)
- **Infura IPFS**: Alternative pinning service

**Recommendations**:
- Use Pinata or Infura in production for reliable pinning
- Local IPFS node requires significant storage and bandwidth
- Enable IPFS only if decentralized storage is required

## ClamAV Antivirus

Virus scanning configuration for uploaded files.

| Variable | Description | Type | Default | Required | Example |
|----------|-------------|------|---------|----------|---------|
| `CLAMAV_HOST` | ClamAV daemon hostname | String | `localhost` | Yes | `clamav`, `av.example.com` |
| `CLAMAV_PORT` | ClamAV daemon port | Integer | `3310` | Yes | `3310` |
| `CLAMAV_TIMEOUT` | Scan timeout | Duration | `30s` | No | `30s`, `60s` |
| `CLAMAV_UPDATE_FREQUENCY` | Virus DB update frequency (hours) | Integer | `24` | No | `24`, `12`, `6` |

**Performance Notes**:
- ClamAV requires 2-4GB RAM
- First-time virus DB download takes 5-10 minutes
- Scan time: ~1-5 seconds per file

**Security Recommendations**:
- Update virus definitions daily (default: 24 hours)
- Increase `CLAMAV_UPDATE_FREQUENCY=12` for high-security environments
- Monitor ClamAV logs for malware detections

## Image Processing

Configuration for image upload and processing.

| Variable | Description | Type | Default | Example |
|----------|-------------|------|---------|---------|
| `IMAGE_MAX_FILE_SIZE` | Maximum upload size (bytes) | Integer | `10485760` | `10485760` (10MB), `52428800` (50MB) |
| `IMAGE_ALLOWED_TYPES` | Allowed MIME types | String | `image/jpeg,image/png,image/webp,image/gif` | `image/jpeg,image/png,image/webp` |
| `IMAGE_QUALITY` | JPEG/WebP quality (1-100) | Integer | `85` | `80`, `90`, `95` |
| `IMAGE_STRIP_METADATA` | Strip EXIF metadata | Boolean | `true` | `true`, `false` |

**Quality Recommendations**:
- **85**: Balanced quality/size (default)
- **80**: Smaller files, slight quality loss
- **90-95**: High quality, larger files
- **100**: Lossless (not recommended - very large files)

**Security Notes**:
- **ALWAYS set `IMAGE_STRIP_METADATA=true` in production** (privacy - removes GPS, camera info)
- Limit `IMAGE_MAX_FILE_SIZE` to prevent DoS attacks
- Validate `IMAGE_ALLOWED_TYPES` to prevent malicious uploads

## Rate Limiting

API rate limiting configuration.

| Variable | Description | Type | Default | Example |
|----------|-------------|------|---------|---------|
| `RATE_LIMIT_ENABLED` | Enable rate limiting | Boolean | `true` | `true`, `false` |
| `RATE_LIMIT_REQUESTS` | Requests per window | Integer | `100` | `100`, `1000` |
| `RATE_LIMIT_WINDOW` | Time window | Duration | `1m` | `1m`, `1h` |

**Recommendations**:
- **Public API**: `RATE_LIMIT_REQUESTS=100` per minute
- **Authenticated users**: `RATE_LIMIT_REQUESTS=1000` per minute
- **Image upload**: Lower limit (e.g., 10 per minute)

**Implementation**:
- Rate limiting uses Redis for distributed tracking
- Separate limits per endpoint (configured in code)

## Worker Configuration

Background job worker settings.

| Variable | Description | Type | Default | Example |
|----------|-------------|------|---------|---------|
| `WORKER_CONCURRENCY` | Concurrent job workers | Integer | `10` | `10`, `20`, `50` |
| `WORKER_QUEUES` | Queue priorities | String | `critical:6,default:3,low:1` | `critical:6,default:3,low:1` |

**Queue Priorities**:
- **critical:6** - High priority (image processing, virus scanning)
- **default:3** - Normal priority (notifications, emails)
- **low:1** - Low priority (cleanup, analytics)

**Concurrency Recommendations**:
- **Development**: `WORKER_CONCURRENCY=5`
- **Production (4 CPU)**: `WORKER_CONCURRENCY=10-20`
- **Production (8 CPU)**: `WORKER_CONCURRENCY=20-50`

## Logging Configuration

Application logging settings.

| Variable | Description | Type | Default | Options |
|----------|-------------|------|---------|---------|
| `LOG_LEVEL` | Logging level | String | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | Log output format | String | `json` | `json`, `text` |

**Log Levels**:
- **debug**: Verbose (development only)
- **info**: Normal operations (production default)
- **warn**: Warnings and errors only
- **error**: Errors only

**Format**:
- **json**: Structured JSON (recommended for production, log aggregation)
- **text**: Human-readable (development, debugging)

**Examples**:

Development:
```bash
LOG_LEVEL=debug
LOG_FORMAT=text
```

Production:
```bash
LOG_LEVEL=info
LOG_FORMAT=json
```

## Observability

Metrics, tracing, and monitoring configuration.

### Metrics (Prometheus)

| Variable | Description | Type | Default | Example |
|----------|-------------|------|---------|---------|
| `METRICS_ENABLED` | Enable Prometheus metrics | Boolean | `true` | `true`, `false` |
| `METRICS_PORT` | Metrics HTTP port | Integer | `9090` | `9090`, `2112` |
| `PROMETHEUS_PORT` | Alias for METRICS_PORT | Integer | `9090` | `9090` |

**Metrics Endpoint**: `http://localhost:9090/metrics`

### Tracing (OpenTelemetry)

| Variable | Description | Type | Default | Example |
|----------|-------------|------|---------|---------|
| `TRACING_ENABLED` | Enable distributed tracing | Boolean | `false` | `true`, `false` |
| `TRACING_ENDPOINT` | OpenTelemetry collector endpoint | URL | - | `http://otel-collector:4318` |

### Error Tracking (Sentry)

| Variable | Description | Type | Required | Example |
|----------|-------------|------|----------|---------|
| `SENTRY_DSN` | Sentry Data Source Name | Secret | No | `https://xxx@sentry.io/xxx` |
| `SENTRY_ENVIRONMENT` | Environment tag | String | No | `production`, `staging` |

**Sentry Configuration** (optional):
```bash
SENTRY_DSN=https://xxxxxx@o12345.ingest.sentry.io/67890
SENTRY_ENVIRONMENT=production
```

## Example Configuration Files

### .env.example (Development)

```bash
# ============================================================================
# Development Environment Configuration
# ============================================================================

# Application
APP_NAME=goimg-api
ENVIRONMENT=development
SECRET_PROVIDER=env

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=goimg
DB_PASSWORD=goimg_dev_password
DB_NAME=goimg
DB_SSL_MODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_SECRET=dev_secret_change_in_production_min_64_chars_abcdefghijklmnopqrstuvwxyz
JWT_EXPIRATION=24h
JWT_REFRESH_EXPIRATION=168h

# API Server
API_PORT=8080
API_HOST=0.0.0.0
API_READ_TIMEOUT=30s
API_WRITE_TIMEOUT=30s
API_IDLE_TIMEOUT=120s

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Accept,Authorization,Content-Type,X-CSRF-Token
CORS_ALLOW_CREDENTIALS=true

# Storage (local for dev)
STORAGE_PROVIDER=local
STORAGE_LOCAL_PATH=/tmp/goimg-uploads

# IPFS
IPFS_ENABLED=true
IPFS_API_URL=http://localhost:5001
IPFS_GATEWAY_URL=https://ipfs.io

# ClamAV
CLAMAV_HOST=localhost
CLAMAV_PORT=3310
CLAMAV_TIMEOUT=30s

# Image Processing
IMAGE_MAX_FILE_SIZE=10485760
IMAGE_ALLOWED_TYPES=image/jpeg,image/png,image/webp,image/gif
IMAGE_QUALITY=85
IMAGE_STRIP_METADATA=true

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Worker
WORKER_CONCURRENCY=5
WORKER_QUEUES=critical:6,default:3,low:1

# Logging
LOG_LEVEL=debug
LOG_FORMAT=text

# Observability
METRICS_ENABLED=true
METRICS_PORT=9090
TRACING_ENABLED=false
```

### .env.staging.example (Staging)

```bash
# ============================================================================
# Staging Environment Configuration
# ============================================================================

# Application
APP_NAME=goimg-api-staging
ENVIRONMENT=staging
SECRET_PROVIDER=docker

# Database (uses Docker Secret for password)
DB_HOST=postgres
DB_PORT=5432
DB_USER=goimg
DB_NAME=goimg
DB_SSL_MODE=require

# Redis (uses Docker Secret for password)
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_DB=0

# JWT (uses Docker Secret)
JWT_EXPIRATION=24h
JWT_REFRESH_EXPIRATION=168h

# API Server
API_PORT=8080
API_HOST=0.0.0.0
API_READ_TIMEOUT=30s
API_WRITE_TIMEOUT=60s
API_IDLE_TIMEOUT=120s

# CORS
CORS_ALLOWED_ORIGINS=https://staging.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Accept,Authorization,Content-Type,X-CSRF-Token
CORS_ALLOW_CREDENTIALS=true

# Storage (S3)
STORAGE_PROVIDER=s3
S3_ENDPOINT=s3.us-east-1.amazonaws.com
S3_REGION=us-east-1
S3_BUCKET=goimg-staging
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
IMAGE_MAX_FILE_SIZE=10485760
IMAGE_ALLOWED_TYPES=image/jpeg,image/png,image/webp,image/gif
IMAGE_QUALITY=85
IMAGE_STRIP_METADATA=true

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Worker
WORKER_CONCURRENCY=10
WORKER_QUEUES=critical:6,default:3,low:1

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Observability
METRICS_ENABLED=true
METRICS_PORT=9090
TRACING_ENABLED=false
```

### .env.production.example (Production)

```bash
# ============================================================================
# Production Environment Configuration
# ============================================================================
# WARNING: This is a template with placeholder values
# NEVER use these values in production!
# Use Docker Secrets for all sensitive values
# ============================================================================

# Application
APP_NAME=goimg-api
ENVIRONMENT=production
SECRET_PROVIDER=docker

# Database (uses Docker Secret: /run/secrets/DB_PASSWORD)
DB_HOST=postgres
DB_PORT=5432
DB_USER=goimg
DB_NAME=goimg
DB_SSL_MODE=require
DB_MAX_OPEN_CONNS=100
DB_MAX_IDLE_CONNS=25
DB_CONN_MAX_LIFETIME=5m

# Redis (uses Docker Secret: /run/secrets/REDIS_PASSWORD)
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_DB=0
REDIS_MAX_RETRIES=3
REDIS_MAX_MEMORY=1gb

# JWT (uses Docker Secret: /run/secrets/JWT_SECRET or JWT_PRIVATE_KEY)
JWT_EXPIRATION=24h
JWT_REFRESH_EXPIRATION=168h

# API Server
API_PORT=8080
API_HOST=0.0.0.0
API_READ_TIMEOUT=30s
API_WRITE_TIMEOUT=60s
API_IDLE_TIMEOUT=120s
API_MAX_HEADER_BYTES=1048576

# CORS (REPLACE WITH YOUR DOMAINS!)
CORS_ALLOWED_ORIGINS=https://api.example.com,https://www.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Accept,Authorization,Content-Type,X-CSRF-Token
CORS_ALLOW_CREDENTIALS=true

# Storage Provider (choose: s3, do-spaces, b2)
STORAGE_PROVIDER=s3
S3_ENDPOINT=s3.us-east-1.amazonaws.com
S3_REGION=us-east-1
S3_BUCKET=goimg-production
S3_USE_SSL=true
# S3_ACCESS_KEY - uses Docker Secret: /run/secrets/S3_ACCESS_KEY
# S3_SECRET_KEY - uses Docker Secret: /run/secrets/S3_SECRET_KEY

# IPFS
IPFS_ENABLED=true
IPFS_API_URL=http://ipfs:5001
IPFS_GATEWAY_URL=https://ipfs.io
# IPFS_PINATA_JWT - uses Docker Secret: /run/secrets/IPFS_PINATA_JWT

# ClamAV
CLAMAV_HOST=clamav
CLAMAV_PORT=3310
CLAMAV_TIMEOUT=30s
CLAMAV_UPDATE_FREQUENCY=24

# Image Processing
IMAGE_MAX_FILE_SIZE=10485760
IMAGE_ALLOWED_TYPES=image/jpeg,image/png,image/webp,image/gif
IMAGE_QUALITY=85
IMAGE_STRIP_METADATA=true

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Worker
WORKER_CONCURRENCY=20
WORKER_QUEUES=critical:6,default:3,low:1

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Observability
METRICS_ENABLED=true
METRICS_PORT=9090
TRACING_ENABLED=false
SENTRY_DSN=  # Optional: your Sentry DSN
SENTRY_ENVIRONMENT=production

# Data Storage
DATA_ROOT=/var/lib/goimg
```

## Troubleshooting

### Common Configuration Errors

#### 1. Database Connection Failed

**Error**: `dial tcp: connect: connection refused`

**Solutions**:
```bash
# Check DB_HOST is correct
echo $DB_HOST  # Should be "postgres" in Docker Compose

# Verify database is running
docker ps | grep postgres
docker logs goimg-postgres

# Check network connectivity
docker exec -it goimg-api nc -zv postgres 5432
```

#### 2. Redis Connection Failed

**Error**: `dial tcp: connect: connection refused` or `NOAUTH Authentication required`

**Solutions**:
```bash
# Check REDIS_HOST is correct
echo $REDIS_HOST  # Should be "redis" in Docker Compose

# Verify Redis is running
docker ps | grep redis
docker logs goimg-redis

# Check password is set (production)
docker exec -it goimg-redis redis-cli -a "$(sudo cat /etc/goimg/secrets/redis_password)" ping
```

#### 3. JWT Authentication Failed

**Error**: `invalid token` or `signature is invalid`

**Solutions**:
```bash
# Verify JWT_SECRET is set
docker exec -it goimg-api env | grep JWT_SECRET

# Check secret length (must be 64+ characters)
cat /etc/goimg/secrets/jwt_secret | wc -c

# Regenerate if needed
openssl rand -base64 64 | sudo tee /etc/goimg/secrets/jwt_secret > /dev/null
docker-compose -f docker/docker-compose.prod.yml restart api
```

#### 4. S3 Storage Failed

**Error**: `The AWS Access Key Id you provided does not exist` or `SignatureDoesNotMatch`

**Solutions**:
```bash
# Verify S3_ENDPOINT is correct for provider
echo $S3_ENDPOINT

# Check credentials are set
docker exec -it goimg-api ls -la /run/secrets/ | grep S3

# Test S3 connectivity (using AWS CLI)
docker exec -it goimg-api aws s3 ls s3://$S3_BUCKET --endpoint-url https://$S3_ENDPOINT
```

#### 5. ClamAV Timeout

**Error**: `clamav scan timeout` or `connection refused`

**Solutions**:
```bash
# Check ClamAV is running
docker ps | grep clamav
docker logs goimg-clamav | grep "Daemon started"

# Increase timeout if needed
CLAMAV_TIMEOUT=60s  # Increase from 30s

# Check connectivity
docker exec -it goimg-api nc -zv clamav 3310
```

### Validation Checklist

Before deploying, verify all required variables:

```bash
# Required environment variables
required_vars=(
  "DB_HOST"
  "DB_PORT"
  "DB_USER"
  "DB_NAME"
  "REDIS_HOST"
  "REDIS_PORT"
  "API_PORT"
  "STORAGE_PROVIDER"
  "CLAMAV_HOST"
  "CLAMAV_PORT"
)

# Check each variable
for var in "${required_vars[@]}"; do
  if [ -z "${!var}" ]; then
    echo "ERROR: $var is not set"
  else
    echo "OK: $var is set"
  fi
done
```

**Required Docker Secrets** (production):
```bash
# Check secrets exist
for secret in jwt_secret db_password redis_password; do
  if [ -f /etc/goimg/secrets/$secret ]; then
    echo "✓ $secret exists"
  else
    echo "✗ $secret MISSING"
  fi
done
```

## References

- [Production Deployment Guide](./production.md)
- [Secret Management Guide](./secrets.md)
- [SSL/TLS Setup](./ssl.md)
- [Docker Compose Configuration](../../docker/docker-compose.prod.yml)

---

**Document Version**: 1.0
**Last Updated**: 2025-12-07 (Sprint 9 - Task 1.4)
**Next Review**: When adding new configuration options
