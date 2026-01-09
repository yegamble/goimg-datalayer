# Secret Management Guide

This document describes how to configure and manage secrets in the goimg-datalayer application for both development and production environments.

## Table of Contents

- [Overview](#overview)
- [Secret Providers](#secret-providers)
- [Required Secrets](#required-secrets)
- [Development Setup](#development-setup)
- [Production Setup](#production-setup)
- [Secret Rotation](#secret-rotation)
- [Security Best Practices](#security-best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The goimg-datalayer application uses a flexible secret management system that supports multiple backends:

- **Environment Variables**: Simple, suitable for local development
- **Docker Secrets**: Secure, recommended for production deployments
- **Extensible**: Easily add support for HashiCorp Vault, AWS Secrets Manager, etc.

Secrets are loaded at application startup via the `SecretProvider` interface defined in `/internal/infrastructure/secrets/`.

## Secret Providers

### Environment Variable Provider (`env`)

**Use Case**: Local development, CI/CD pipelines

**How it Works**: Reads secrets from environment variables using `os.Getenv()`.

**Configuration**:
```bash
export SECRET_PROVIDER=env
export JWT_SECRET="your-jwt-secret-here"
export DB_PASSWORD="your-db-password"
```

**Pros**:
- Simple to set up
- Works everywhere
- Standard 12-factor app pattern

**Cons**:
- Visible in process listings (`ps aux`)
- Can leak via logs or error messages
- Not recommended for production

### Docker Secrets Provider (`docker`)

**Use Case**: Production deployments with Docker Compose or Docker Swarm

**How it Works**: Reads secrets from files mounted at `/run/secrets/` by Docker.

**Configuration**:
```yaml
# docker-compose.prod.yml
services:
  api:
    environment:
      - SECRET_PROVIDER=docker
    secrets:
      - JWT_SECRET
      - DB_PASSWORD

secrets:
  JWT_SECRET:
    file: /etc/goimg/secrets/jwt_secret
  DB_PASSWORD:
    file: /etc/goimg/secrets/db_password
```

**Pros**:
- Secrets not visible in `docker inspect` or process listings
- Filesystem permissions control access
- Native Docker/Kubernetes integration
- Supports secret rotation without container rebuilds

**Cons**:
- Requires initial setup of secret files
- More complex than environment variables

## Required Secrets

The following secrets **MUST** be configured for the application to start:

| Secret Name | Description | Minimum Length | Generation Command |
|-------------|-------------|----------------|-------------------|
| `JWT_SECRET` | JWT token signing key | 64 characters | `openssl rand -base64 64` |
| `DB_PASSWORD` | PostgreSQL password | 32 characters | `openssl rand -base64 32` |

## Optional Secrets

These secrets enable additional features but are not required for basic operation:

| Secret Name | Description | When Required |
|-------------|-------------|---------------|
| `REDIS_PASSWORD` | Redis authentication | Production (recommended) |
| `S3_SECRET_KEY` | S3-compatible storage secret | When using S3/Spaces/B2 |
| `IPFS_PINATA_JWT` | Pinata IPFS pinning service | When using Pinata |
| `IPFS_INFURA_PROJECT_SECRET` | Infura IPFS service | When using Infura |
| `OAUTH_GOOGLE_CLIENT_SECRET` | Google OAuth | When enabling Google login |
| `OAUTH_GITHUB_CLIENT_SECRET` | GitHub OAuth | When enabling GitHub login |
| `SMTP_PASSWORD` | Email server password | When enabling email notifications |
| `GRAFANA_ADMIN_PASSWORD` | Grafana admin password | Production monitoring |

## Development Setup

### Method 1: Environment Variables (Recommended for Local Dev)

1. **Copy the example environment file**:
   ```bash
   cd docker
   cp .env.example .env
   ```

2. **Edit `.env` and set your secrets**:
   ```bash
   # For development, you can use weak secrets
   JWT_SECRET=dev_secret_change_in_production_min_32_chars
   DB_PASSWORD=goimg_dev_password
   REDIS_PASSWORD=  # Empty = no auth in dev
   ```

3. **Start the application**:
   ```bash
   docker-compose -f docker/docker-compose.yml up -d
   make run
   ```

### Method 2: Direct Environment Variables

```bash
export SECRET_PROVIDER=env
export JWT_SECRET=$(openssl rand -base64 64)
export DB_PASSWORD=$(openssl rand -base64 32)
export REDIS_PASSWORD=  # Optional in dev

make run
```

### Verifying Development Setup

```bash
# Check that secrets are loaded
docker logs goimg-api 2>&1 | grep "secret provider"
# Should show: "Initialized environment variable secret provider"
```

## Production Setup

### Step 1: Generate Strong Secrets

```bash
# Create secrets directory with restricted permissions
sudo mkdir -p /etc/goimg/secrets
sudo chmod 700 /etc/goimg/secrets

# Generate required secrets
openssl rand -base64 64 > /tmp/jwt_secret
openssl rand -base64 32 > /tmp/db_password
openssl rand -base64 32 > /tmp/redis_password

# Move to secrets directory with proper permissions
sudo mv /tmp/jwt_secret /etc/goimg/secrets/jwt_secret
sudo mv /tmp/db_password /etc/goimg/secrets/db_password
sudo mv /tmp/redis_password /etc/goimg/secrets/redis_password

# Set restrictive permissions (only root can read)
sudo chmod 600 /etc/goimg/secrets/*

# Verify permissions
ls -la /etc/goimg/secrets/
# Should show: -rw------- 1 root root
```

### Step 2: Generate Optional Secrets

```bash
# S3-compatible storage
echo "your-s3-secret-key-here" | sudo tee /etc/goimg/secrets/s3_secret_key
sudo chmod 600 /etc/goimg/secrets/s3_secret_key

# IPFS Pinata (get from https://pinata.cloud)
echo "your-pinata-jwt-token" | sudo tee /etc/goimg/secrets/ipfs_pinata_jwt
sudo chmod 600 /etc/goimg/secrets/ipfs_pinata_jwt

# Grafana admin password
openssl rand -base64 32 | sudo tee /etc/goimg/secrets/grafana_admin_password
sudo chmod 600 /etc/goimg/secrets/grafana_admin_password
```

### Step 3: Configure Docker Compose

The production `docker-compose.prod.yml` is already configured to use Docker Secrets. Verify the configuration:

```bash
# Check secrets are defined
grep -A 2 "^secrets:" docker/docker-compose.prod.yml

# Verify services reference secrets
grep -A 5 "secrets:" docker/docker-compose.prod.yml
```

### Step 4: Deploy with Docker Compose

```bash
cd docker
docker-compose -f docker-compose.prod.yml up -d
```

### Step 5: Verify Production Deployment

```bash
# Check logs for secret provider initialization
docker logs goimg-api 2>&1 | grep "secret provider"
# Should show: "Initialized Docker Secrets provider"

# Verify secrets are mounted
docker exec goimg-api ls -la /run/secrets/
# Should list: JWT_SECRET, DB_PASSWORD, REDIS_PASSWORD, etc.

# Test secret is readable (DO NOT log the actual value!)
docker exec goimg-api test -r /run/secrets/JWT_SECRET && echo "JWT_SECRET is readable"
```

### Docker Swarm (Advanced)

For Docker Swarm orchestration:

```bash
# Create secrets in Swarm
docker secret create JWT_SECRET /etc/goimg/secrets/jwt_secret
docker secret create DB_PASSWORD /etc/goimg/secrets/db_password
docker secret create REDIS_PASSWORD /etc/goimg/secrets/redis_password

# Deploy stack
docker stack deploy -c docker-compose.prod.yml goimg

# Verify secrets
docker secret ls
```

## Secret Rotation

Regular secret rotation is a critical security practice. Follow these procedures:

### JWT Secret Rotation

**WARNING**: Rotating JWT secrets will invalidate all existing user sessions.

```bash
# 1. Generate new secret
openssl rand -base64 64 > /tmp/jwt_secret_new

# 2. Update secret file
sudo mv /tmp/jwt_secret_new /etc/goimg/secrets/jwt_secret
sudo chmod 600 /etc/goimg/secrets/jwt_secret

# 3. Restart application (triggers re-read of secrets)
docker-compose -f docker/docker-compose.prod.yml restart api

# 4. Verify new secret is loaded
docker logs goimg-api 2>&1 | grep "secret provider"
```

### Database Password Rotation

See [docs/security/secret_rotation.md](../security/secret_rotation.md) for detailed database password rotation procedures.

**Quick Steps**:
1. Create new database user with new password
2. Update `/etc/goimg/secrets/db_password`
3. Grant permissions to new user
4. Restart application
5. Verify connectivity
6. Remove old user

### Redis Password Rotation

```bash
# 1. Generate new password
openssl rand -base64 32 > /tmp/redis_password_new

# 2. Update secret file
sudo mv /tmp/redis_password_new /etc/goimg/secrets/redis_password
sudo chmod 600 /etc/goimg/secrets/redis_password

# 3. Update Redis configuration and restart
docker-compose -f docker/docker-compose.prod.yml restart redis

# 4. Restart dependent services
docker-compose -f docker/docker-compose.prod.yml restart api worker
```

### Rotation Best Practices

- **Frequency**: Rotate critical secrets (JWT, DB) every 90 days minimum
- **Automation**: Use scripts or secret management tools for rotation
- **Monitoring**: Log rotation events for audit trails
- **Testing**: Test secret rotation in staging before production
- **Backups**: Keep encrypted backups of old secrets for rollback

## Security Best Practices

### 1. Never Commit Secrets to Version Control

```bash
# Verify no secrets in git history
git grep -i "password\|secret\|token" docker/.env.prod
# Should return no results

# Ensure .gitignore blocks secret files
cat .gitignore | grep -E "\.env$|\.env\.prod$|secrets/"
```

### 2. Use Strong, Random Secrets

```bash
# GOOD: Cryptographically random
openssl rand -base64 64

# GOOD: High entropy
pwgen -s 64 1

# BAD: Weak, predictable
echo "password123"  # DO NOT USE
```

### 3. Restrict File Permissions

```bash
# Secrets directory: only root/admin access
chmod 700 /etc/goimg/secrets

# Secret files: only owner can read
chmod 600 /etc/goimg/secrets/*

# Verify no world-readable files
find /etc/goimg/secrets -type f -perm /o+r
# Should return nothing
```

### 4. Encrypt Secrets at Rest

For sensitive production environments, encrypt secret files:

```bash
# Encrypt with GPG
gpg --encrypt --recipient ops@yourdomain.com /etc/goimg/secrets/jwt_secret

# Decrypt on deployment
gpg --decrypt /etc/goimg/secrets/jwt_secret.gpg > /etc/goimg/secrets/jwt_secret
```

### 5. Use Separate Secrets Per Environment

```
/etc/goimg/secrets/
├── dev/
│   ├── jwt_secret
│   └── db_password
├── staging/
│   ├── jwt_secret
│   └── db_password
└── prod/
    ├── jwt_secret
    └── db_password
```

### 6. Monitor Secret Access

```bash
# Enable audit logging for secret files
sudo auditctl -w /etc/goimg/secrets/ -p war -k goimg-secrets

# Check audit logs
sudo ausearch -k goimg-secrets
```

### 7. Implement Least Privilege

- Only grant secret access to services that need them
- Use read-only mounts in Docker:
  ```yaml
  secrets:
    - source: JWT_SECRET
      mode: 0400  # Read-only
  ```

## Troubleshooting

### Error: "Required secret not found"

**Symptom**: Application crashes on startup with error message:
```
FATAL Required secret JWT_SECRET not found
```

**Solution**:
1. Check SECRET_PROVIDER is set correctly:
   ```bash
   docker exec goimg-api env | grep SECRET_PROVIDER
   ```

2. For `env` provider, verify environment variable:
   ```bash
   docker exec goimg-api env | grep JWT_SECRET
   ```

3. For `docker` provider, verify secret file exists:
   ```bash
   docker exec goimg-api ls -la /run/secrets/JWT_SECRET
   ```

### Error: "Permission denied reading secret"

**Symptom**:
```
ERROR Failed to read secret JWT_SECRET: permission denied
```

**Solution**:
```bash
# Check file permissions
ls -la /etc/goimg/secrets/jwt_secret

# Fix permissions
sudo chmod 600 /etc/goimg/secrets/jwt_secret

# Restart container
docker-compose -f docker/docker-compose.prod.yml restart api
```

### Error: "Secret is empty"

**Symptom**:
```
ERROR Secret JWT_SECRET is empty at /run/secrets/JWT_SECRET
```

**Solution**:
```bash
# Check file content (on host)
sudo cat /etc/goimg/secrets/jwt_secret

# Regenerate if empty
openssl rand -base64 64 | sudo tee /etc/goimg/secrets/jwt_secret

# Restart
docker-compose restart api
```

### Debugging Secret Loading

Enable debug logging to see secret loading details:

```bash
# Set log level to debug
docker-compose -f docker/docker-compose.prod.yml up -d \
  -e LOG_LEVEL=debug

# Check logs
docker logs goimg-api 2>&1 | grep -i secret
```

### Listing Available Secrets

The application logs which secrets are available at startup (without logging values):

```
INFO All required secrets validated count=2 provider=docker-secrets
```

## Migration from Environment Variables to Docker Secrets

If you're currently using environment variables and want to migrate to Docker Secrets:

### Step 1: Export Current Secrets

```bash
# Extract from .env.prod (DO THIS SECURELY!)
source docker/.env.prod

# Create secret files
echo "$JWT_SECRET" | sudo tee /etc/goimg/secrets/jwt_secret
echo "$DB_PASSWORD" | sudo tee /etc/goimg/secrets/db_password

sudo chmod 600 /etc/goimg/secrets/*
```

### Step 2: Update Docker Compose

```yaml
# Change from:
environment:
  - JWT_SECRET=${JWT_SECRET}

# To:
environment:
  - SECRET_PROVIDER=docker
secrets:
  - JWT_SECRET
```

### Step 3: Test Migration

```bash
# Stop services
docker-compose -f docker/docker-compose.prod.yml down

# Start with new configuration
docker-compose -f docker/docker-compose.prod.yml up -d

# Verify
docker logs goimg-api 2>&1 | grep "Docker Secrets provider"
```

### Step 4: Remove Old .env.prod

```bash
# Securely wipe old environment file
shred -u docker/.env.prod

# Or use secure delete
srm docker/.env.prod
```

## Advanced: HashiCorp Vault Integration

To add HashiCorp Vault support, implement a new provider:

```go
// internal/infrastructure/secrets/vault_provider.go
type VaultProvider struct {
    client *vault.Client
}

func (p *VaultProvider) GetSecret(ctx context.Context, name string) (string, error) {
    secret, err := p.client.Logical().Read("secret/goimg/" + name)
    // ... implementation
}
```

Update the factory in `provider.go`:

```go
case "vault":
    return NewVaultProvider(config.VaultAddr, config.VaultToken), nil
```

## Security Gate Compliance

This secret management implementation satisfies:

- **S9-PROD-001**: No hardcoded secrets in config files or code
  - All secrets loaded dynamically at runtime
  - Provider abstraction supports multiple backends
  - Fail-fast on missing required secrets

## References

- [Docker Secrets Documentation](https://docs.docker.com/engine/swarm/secrets/)
- [12-Factor App: Config](https://12factor.net/config)
- [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html)
- [NIST Special Publication 800-57: Key Management](https://csrc.nist.gov/publications/detail/sp/800-57-part-1/rev-5/final)
- Project-specific rotation procedures: [docs/security/secret_rotation.md](../security/secret_rotation.md)
