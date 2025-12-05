# Secret Rotation Procedures

> Zero-downtime credential rotation for goimg-datalayer production systems

## Overview

This document provides step-by-step procedures for rotating all security-sensitive credentials in the goimg-datalayer platform. Regular credential rotation is a critical security control that limits the impact of credential compromise and satisfies compliance requirements.

**Document Owner**: Security Operations Team
**Last Updated**: 2025-12-05
**Review Frequency**: Quarterly

---

## Rotation Schedule

### Routine Rotation

| Secret Type | Rotation Frequency | Last Rotated | Next Scheduled |
|-------------|-------------------|--------------|----------------|
| JWT Signing Keys | Every 6 months | 2025-06-01 | 2025-12-01 |
| PostgreSQL Password | Every 3 months | 2025-11-01 | 2026-02-01 |
| Redis Password | Every 3 months | 2025-11-01 | 2026-02-01 |
| S3/Storage Credentials | Every 6 months | 2025-09-01 | 2026-03-01 |
| API Keys (Third-Party) | Every 12 months | 2025-01-01 | 2026-01-01 |
| Session Encryption Key | Every 6 months | 2025-06-01 | 2025-12-01 |
| ClamAV Admin Password | Every 12 months | 2025-01-01 | 2026-01-01 |

### Emergency Rotation

Rotate immediately if:
- Credential leaked in logs, code, or public repository
- Employee with access leaves company
- Security incident confirms or suspects credential compromise
- Compliance audit requires immediate action
- Third-party provider reports breach

---

## General Rotation Principles

### Zero-Downtime Strategy

All rotations follow this pattern:
1. **Add new credential** alongside existing (dual-credential period)
2. **Deploy application** to accept both old and new credentials
3. **Update clients** to use new credential
4. **Validate** new credential is working in production
5. **Remove old credential** after grace period (typically 24-48 hours)

### Rollback Plan

Every rotation must have:
- Documented rollback steps
- Old credential retained for 48 hours minimum
- Monitoring to detect rotation-related failures
- Team on standby during rotation window

### Communication

Before rotation:
- Notify engineering team 24 hours in advance
- Schedule during low-traffic window (typically Tuesday/Wednesday 2-4 AM UTC)
- Create maintenance window in status page (if user-facing impact possible)
- Ensure on-call engineer is available

---

## JWT Signing Key Rotation

**Frequency**: Every 6 months (routine) or immediately (emergency)
**Downtime**: Zero (with dual-key validation)
**Complexity**: Medium

### Prerequisites

```bash
# Ensure you have required tools
which openssl || sudo apt-get install openssl
which jq || sudo apt-get install jq

# Backup current keys
cp /secrets/jwt_private.pem /backups/jwt_private_$(date +%Y%m%d).pem
cp /secrets/jwt_public.pem /backups/jwt_public_$(date +%Y%m%d).pem
```

### Step 1: Generate New RSA Key Pair

```bash
# Generate 4096-bit private key
openssl genrsa -out /secrets/jwt_private_new.pem 4096

# Extract public key
openssl rsa -in /secrets/jwt_private_new.pem -pubout -out /secrets/jwt_public_new.pem

# Verify key size
openssl rsa -in /secrets/jwt_private_new.pem -text -noout | grep "Private-Key"
# Expected output: Private-Key: (4096 bit, 2 primes)

# Set proper permissions
chmod 600 /secrets/jwt_private_new.pem
chmod 644 /secrets/jwt_public_new.pem
chown goimg-api:goimg-api /secrets/jwt_private_new.pem /secrets/jwt_public_new.pem
```

### Step 2: Update Application to Support Dual Keys

**Code Change** (`internal/infrastructure/security/jwt/jwt_service.go`):

```go
type JWTService struct {
    // Signing (always use latest)
    signingKey *rsa.PrivateKey

    // Validation (support multiple for rotation)
    validationKeys []*rsa.PublicKey

    config Config
}

func NewJWTService(config Config) (*JWTService, error) {
    // Load new private key for signing
    signingKey, err := loadPrivateKey(config.PrivateKeyPath)
    if err != nil {
        return nil, fmt.Errorf("load signing key: %w", err)
    }

    // Load multiple public keys for validation (new + old during rotation)
    validationKeys := make([]*rsa.PublicKey, 0)
    for _, pubKeyPath := range config.PublicKeyPaths {
        pubKey, err := loadPublicKey(pubKeyPath)
        if err != nil {
            return nil, fmt.Errorf("load validation key %s: %w", pubKeyPath, err)
        }
        validationKeys = append(validationKeys, pubKey)
    }

    return &JWTService{
        signingKey:     signingKey,
        validationKeys: validationKeys,
        config:         config,
    }, nil
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
    // Try each public key until one succeeds
    var lastErr error
    for i, pubKey := range s.validationKeys {
        token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
            return pubKey, nil
        })

        if err == nil && token.Valid {
            return token.Claims.(*Claims), nil
        }
        lastErr = err
    }

    return nil, fmt.Errorf("token validation failed with all keys: %w", lastErr)
}
```

**Configuration Update** (`config/production.yaml`):

```yaml
jwt:
  # New key for signing
  private_key_path: /secrets/jwt_private_new.pem

  # Multiple keys for validation (new first, then old for grace period)
  public_key_paths:
    - /secrets/jwt_public_new.pem
    - /secrets/jwt_public.pem  # Old key, will be removed after 48h
```

### Step 3: Deploy Application with Dual-Key Support

```bash
# Build new version with dual-key support
make build

# Deploy to staging for validation
make deploy-staging

# Test token generation and validation
# Generate token with new key, validate with both keys
curl -X POST https://staging-api.goimg.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"testpass"}' | jq -r .access_token > /tmp/new_token.txt

# Validate new token works
curl -X GET https://staging-api.goimg.example.com/api/v1/users/me \
  -H "Authorization: Bearer $(cat /tmp/new_token.txt)"
# Expected: 200 OK with user data

# Validate old tokens still work (if you have one)
# This confirms backward compatibility
curl -X GET https://staging-api.goimg.example.com/api/v1/users/me \
  -H "Authorization: Bearer $OLD_TOKEN"
# Expected: 200 OK

# Deploy to production (blue-green deployment)
make deploy-production-bg
```

### Step 4: Monitor for Validation Errors

```bash
# Watch for JWT validation failures
watch -n 5 'curl -s http://prometheus:9090/api/v1/query?query=rate(jwt_validation_failures_total[5m])'

# Check application logs for key-related errors
tail -f /var/log/goimg-api/app.log | grep -i "jwt\|token"

# Verify both keys are being used
grep "jwt_public_new.pem\|jwt_public.pem" /var/log/goimg-api/app.log
```

**Expected Behavior**:
- New logins receive tokens signed with new key
- Old tokens (signed with old key) still validate successfully
- No increase in validation failure rate

### Step 5: Grace Period (48 Hours)

Wait 48 hours for all old tokens to expire naturally:
- Access tokens expire in 15 minutes (negligible)
- Refresh tokens expire in 7 days (must wait or force logout all users)

**Option A: Wait 7 Days** (no user impact)
- All old tokens expire naturally
- Users automatically get new tokens via refresh flow

**Option B: Force Logout All Users** (immediate, user impact)
```bash
# Invalidate all existing sessions
redis-cli FLUSHDB  # WARNING: Destroys all sessions, use carefully

# Notify users
curl -X POST https://api.goimg.example.com/admin/v1/notifications/broadcast \
  -H "Content-Type: application/json" \
  -d '{
    "type": "security_update",
    "message": "For your security, all sessions have been logged out. Please log in again.",
    "severity": "info"
  }'
```

### Step 6: Remove Old Key

After grace period (48 hours to 7 days):

**Configuration Update**:
```yaml
jwt:
  private_key_path: /secrets/jwt_private_new.pem
  public_key_paths:
    - /secrets/jwt_public_new.pem
    # OLD KEY REMOVED
```

**Deploy Updated Config**:
```bash
# Deploy configuration update
make deploy-production

# Verify only new key is loaded
grep "Loaded.*public keys" /var/log/goimg-api/app.log
# Expected: "Loaded 1 public keys for validation"

# Monitor for 2 hours
watch -n 10 'curl -s http://prometheus:9090/api/v1/query?query=rate(jwt_validation_failures_total[5m])'
```

### Step 7: Archive Old Key

```bash
# Move old key to archive (encrypted S3)
tar czf jwt_keys_retired_$(date +%Y%m%d).tar.gz \
  /secrets/jwt_private.pem \
  /secrets/jwt_public.pem

# Encrypt archive
gpg --encrypt --recipient security@goimg.example.com \
  jwt_keys_retired_$(date +%Y%m%d).tar.gz

# Upload to secure archive
aws s3 cp jwt_keys_retired_$(date +%Y%m%d).tar.gz.gpg \
  s3://goimg-secrets-archive/jwt/

# Securely delete local copies
shred -vfz -n 3 /secrets/jwt_private.pem
shred -vfz -n 3 /secrets/jwt_public.pem
rm jwt_keys_retired_$(date +%Y%m%d).tar.gz*

# Rename new keys to standard names
mv /secrets/jwt_private_new.pem /secrets/jwt_private.pem
mv /secrets/jwt_public_new.pem /secrets/jwt_public.pem
```

### Rollback Procedure

If issues detected during rotation:

```bash
# Revert configuration to use old key only
# Update config/production.yaml:
# jwt:
#   private_key_path: /secrets/jwt_private.pem
#   public_key_paths:
#     - /secrets/jwt_public.pem

# Redeploy previous version
kubectl rollout undo deployment/goimg-api

# Verify rollback successful
curl -X GET https://api.goimg.example.com/health
```

---

## PostgreSQL Password Rotation

**Frequency**: Every 3 months (routine) or immediately (emergency)
**Downtime**: Zero (with connection pooling)
**Complexity**: Low

### Step 1: Generate New Password

```bash
# Generate cryptographically secure password (32 characters)
NEW_DB_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
echo "New password (save securely): $NEW_DB_PASSWORD"

# Store in secrets manager
aws secretsmanager create-secret \
  --name goimg/postgres/password-new \
  --secret-string "$NEW_DB_PASSWORD"
```

### Step 2: Create New PostgreSQL User (Temporary)

```bash
# Connect to PostgreSQL as admin
psql -h $DB_HOST -U postgres <<SQL

-- Create new user with same permissions
CREATE USER goimg_new WITH PASSWORD '$NEW_DB_PASSWORD';

-- Grant same permissions as existing user
GRANT ALL PRIVILEGES ON DATABASE goimg TO goimg_new;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO goimg_new;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO goimg_new;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO goimg_new;

-- Verify permissions match
SELECT * FROM information_schema.role_table_grants WHERE grantee = 'goimg_new';
SQL
```

**Alternative: Update Existing User Password**

```bash
psql -h $DB_HOST -U postgres <<SQL
ALTER USER goimg WITH PASSWORD '$NEW_DB_PASSWORD';
SQL
```

### Step 3: Update Application Configuration

**Using Environment Variables**:
```bash
# Update Kubernetes secret
kubectl create secret generic goimg-db-credentials \
  --from-literal=username=goimg_new \
  --from-literal=password=$NEW_DB_PASSWORD \
  --dry-run=client -o yaml | kubectl apply -f -

# Trigger rolling restart to pick up new secret
kubectl rollout restart deployment/goimg-api
kubectl rollout restart deployment/goimg-worker
```

**Using Secrets Manager**:
```go
// internal/infrastructure/persistence/postgres/connection.go

func loadDBCredentials(ctx context.Context) (string, string, error) {
    // Fetch from AWS Secrets Manager
    svc := secretsmanager.New(session.New())
    result, err := svc.GetSecretValueWithContext(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: aws.String("goimg/postgres/password-new"),
    })
    if err != nil {
        return "", "", fmt.Errorf("fetch secret: %w", err)
    }

    var creds struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }
    if err := json.Unmarshal([]byte(*result.SecretString), &creds); err != nil {
        return "", "", fmt.Errorf("parse secret: %w", err)
    }

    return creds.Username, creds.Password, nil
}
```

### Step 4: Validate New Credentials

```bash
# Test connection with new credentials
psql -h $DB_HOST -U goimg_new -d goimg <<SQL
SELECT 1;  -- Should return 1
SQL

# Verify application can connect
curl -X GET https://api.goimg.example.com/health/db
# Expected: {"status": "healthy", "database": "connected"}

# Monitor database connections
psql -h $DB_HOST -U postgres <<SQL
SELECT
  usename,
  COUNT(*) as connection_count,
  state
FROM pg_stat_activity
WHERE datname = 'goimg'
GROUP BY usename, state;
SQL
```

### Step 5: Remove Old Credentials

After grace period (24-48 hours):

```bash
# Revoke old user's access
psql -h $DB_HOST -U postgres <<SQL
REVOKE ALL PRIVILEGES ON DATABASE goimg FROM goimg;
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM goimg;
DROP USER goimg;

-- Rename new user to standard name (if using temporary user)
ALTER USER goimg_new RENAME TO goimg;
SQL

# Delete old secret
aws secretsmanager delete-secret \
  --secret-id goimg/postgres/password \
  --force-delete-without-recovery
```

### Rollback Procedure

```bash
# Revert to old credentials
kubectl create secret generic goimg-db-credentials \
  --from-literal=username=goimg \
  --from-literal=password=$OLD_DB_PASSWORD \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl rollout restart deployment/goimg-api
```

---

## Redis Password Rotation

**Frequency**: Every 3 months (routine) or immediately (emergency)
**Downtime**: ~5 seconds (during Redis restart)
**Complexity**: Low

### Step 1: Generate New Password

```bash
NEW_REDIS_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
echo "New Redis password: $NEW_REDIS_PASSWORD"
```

### Step 2: Update Redis Configuration

**If using Redis 6+ ACL** (preferred):

```bash
# Connect to Redis
redis-cli -h $REDIS_HOST -a $OLD_REDIS_PASSWORD

# Create new user with same permissions
ACL SETUSER goimg_new on >$NEW_REDIS_PASSWORD ~* +@all

# Verify new user
ACL LIST
```

**If using requirepass** (legacy):

```bash
# Update redis.conf
sed -i "s/^requirepass.*/requirepass $NEW_REDIS_PASSWORD/" /etc/redis/redis.conf

# Or via CLI (requires restart)
redis-cli -h $REDIS_HOST -a $OLD_REDIS_PASSWORD CONFIG SET requirepass "$NEW_REDIS_PASSWORD"
redis-cli -h $REDIS_HOST -a $NEW_REDIS_PASSWORD CONFIG REWRITE
```

### Step 3: Update Application Configuration

```bash
# Update Kubernetes secret
kubectl create secret generic goimg-redis-credentials \
  --from-literal=password=$NEW_REDIS_PASSWORD \
  --dry-run=client -o yaml | kubectl apply -f -

# Rolling restart
kubectl rollout restart deployment/goimg-api
kubectl rollout restart deployment/goimg-worker
```

### Step 4: Validate New Credentials

```bash
# Test connection
redis-cli -h $REDIS_HOST -a $NEW_REDIS_PASSWORD PING
# Expected: PONG

# Verify application connectivity
curl -X GET https://api.goimg.example.com/health/redis
# Expected: {"status": "healthy", "redis": "connected"}

# Check active connections
redis-cli -h $REDIS_HOST -a $NEW_REDIS_PASSWORD CLIENT LIST
```

### Step 5: Remove Old Credentials

```bash
# If using ACL, delete old user
redis-cli -h $REDIS_HOST -a $NEW_REDIS_PASSWORD ACL DELUSER goimg

# Rename new user (if temporary)
redis-cli -h $REDIS_HOST -a $NEW_REDIS_PASSWORD ACL SETUSER goimg on >$NEW_REDIS_PASSWORD ~* +@all
redis-cli -h $REDIS_HOST -a $NEW_REDIS_PASSWORD ACL DELUSER goimg_new
```

---

## S3/Object Storage Credentials Rotation

**Frequency**: Every 6 months (routine) or immediately (emergency)
**Downtime**: Zero (with dual-credential support)
**Complexity**: Medium

### Supported Storage Providers

- AWS S3
- DigitalOcean Spaces
- Backblaze B2
- Local storage (no credentials)

### Step 1: Generate New Access Keys

**AWS S3**:
```bash
# Create new IAM access key for goimg-storage user
aws iam create-access-key --user-name goimg-storage | tee /tmp/new_s3_key.json

# Extract credentials
NEW_ACCESS_KEY=$(jq -r .AccessKey.AccessKeyId /tmp/new_s3_key.json)
NEW_SECRET_KEY=$(jq -r .AccessKey.SecretAccessKey /tmp/new_s3_key.json)

echo "Access Key: $NEW_ACCESS_KEY"
echo "Secret Key: $NEW_SECRET_KEY"
```

**DigitalOcean Spaces**:
```bash
# Generate via DO Console or API
# Spaces > Settings > Generate New Key
# Manually create and note down
```

**Backblaze B2**:
```bash
# Via B2 CLI
b2 create-key goimg-storage-new goimg-uploads-bucket read,write

# Note application key ID and key
```

### Step 2: Update Application Configuration

```bash
# Store in Kubernetes secrets
kubectl create secret generic goimg-s3-credentials \
  --from-literal=access_key_id=$NEW_ACCESS_KEY \
  --from-literal=secret_access_key=$NEW_SECRET_KEY \
  --dry-run=client -o yaml | kubectl apply -f -

# Update config (dual-credential period not needed for S3 since SDKs handle this)
# Just update and restart
kubectl rollout restart deployment/goimg-api
kubectl rollout restart deployment/goimg-worker
```

### Step 3: Validate New Credentials

```bash
# Test upload with new credentials
aws s3 cp /tmp/test-file.jpg s3://goimg-uploads/test/ \
  --access-key $NEW_ACCESS_KEY \
  --secret-key $NEW_SECRET_KEY

# Test via API
curl -X POST https://api.goimg.example.com/api/v1/images \
  -H "Authorization: Bearer $USER_TOKEN" \
  -F "image=@test-image.jpg"

# Verify upload succeeded
curl -X GET https://api.goimg.example.com/api/v1/images/{image_id}
```

### Step 4: Delete Old Access Keys

After grace period (24 hours):

```bash
# List existing keys
aws iam list-access-keys --user-name goimg-storage

# Delete old key
aws iam delete-access-key \
  --user-name goimg-storage \
  --access-key-id $OLD_ACCESS_KEY

# Verify only new key exists
aws iam list-access-keys --user-name goimg-storage
```

### Step 5: Monitor for Errors

```bash
# Watch for S3 authentication errors
tail -f /var/log/goimg-api/app.log | grep -i "s3\|storage\|SignatureDoesNotMatch"

# Check CloudTrail for access denied events (AWS)
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=EventName,AttributeValue=AccessDenied \
  --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
  --max-results 50
```

---

## IPFS Pinning Service Credentials Rotation

**Frequency**: Every 6 months or when API key exposed
**Downtime**: Zero
**Complexity**: Low

### Pinata API Key Rotation

```bash
# Generate new API key via Pinata dashboard
# Account > API Keys > New Key
# Permissions: pinFileToIPFS, pinJSONToIPFS, unpin

# Update configuration
kubectl create secret generic goimg-ipfs-credentials \
  --from-literal=pinata_api_key=$NEW_PINATA_KEY \
  --from-literal=pinata_secret_key=$NEW_PINATA_SECRET \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart services
kubectl rollout restart deployment/goimg-api
kubectl rollout restart deployment/goimg-worker

# Test pinning
curl -X POST https://api.goimg.example.com/api/v1/images \
  -H "Authorization: Bearer $USER_TOKEN" \
  -F "image=@test.jpg" \
  -F "storage_provider=ipfs"

# Verify IPFS hash returned
# Delete old API key from Pinata dashboard
```

---

## Emergency Rotation Checklist

When credentials are compromised:

```markdown
### Immediate Actions (within 15 minutes)
- [ ] Confirm credential type and scope of exposure
- [ ] Create incident ticket (SEC-YYYY-NNNN)
- [ ] Page on-call security engineer
- [ ] Begin incident response (see incident_response.md)

### Containment (within 1 hour)
- [ ] Generate new credential
- [ ] Update application configuration with new credential
- [ ] Deploy to production (expedited, skip staging if critical)
- [ ] Revoke/delete old credential immediately (no grace period)
- [ ] Block any unauthorized access detected

### Validation (within 2 hours)
- [ ] Verify application using new credential
- [ ] Monitor for authentication errors
- [ ] Check for unauthorized access using old credential
- [ ] Confirm old credential fully revoked

### Investigation (within 24 hours)
- [ ] Determine how credential was exposed (logs, git, third-party breach)
- [ ] Identify if credential was used maliciously
- [ ] Assess data access/modification during exposure window
- [ ] Document timeline for post-mortem

### Remediation (within 48 hours)
- [ ] Fix root cause (remove from logs, update .gitignore, etc.)
- [ ] Notify affected parties if personal data accessed
- [ ] Update credential rotation procedures if gaps found
- [ ] Conduct post-mortem review
```

---

## Automation Scripts

### Database Password Rotation Script

```bash
#!/bin/bash
# /home/user/goimg-datalayer/scripts/rotate_db_password.sh

set -euo pipefail

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-goimg}"
CURRENT_USER="${DB_USER:-goimg}"
ADMIN_USER="${DB_ADMIN_USER:-postgres}"

# Generate new password
NEW_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)

echo "=== PostgreSQL Password Rotation ==="
echo "Database: $DB_NAME"
echo "User: $CURRENT_USER"
echo ""

read -p "Continue with rotation? (yes/no): " CONFIRM
if [[ "$CONFIRM" != "yes" ]]; then
    echo "Aborted."
    exit 1
fi

# Update password
echo "Updating password..."
PGPASSWORD=$ADMIN_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $ADMIN_USER <<SQL
ALTER USER $CURRENT_USER WITH PASSWORD '$NEW_PASSWORD';
SQL

# Update Kubernetes secret
echo "Updating Kubernetes secret..."
kubectl create secret generic goimg-db-credentials \
  --from-literal=username=$CURRENT_USER \
  --from-literal=password=$NEW_PASSWORD \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart pods
echo "Restarting pods to pick up new credentials..."
kubectl rollout restart deployment/goimg-api
kubectl rollout restart deployment/goimg-worker

# Wait for rollout
kubectl rollout status deployment/goimg-api
kubectl rollout status deployment/goimg-worker

echo ""
echo "=== Rotation Complete ==="
echo "New password has been set and application restarted."
echo "Monitor logs for any connection issues:"
echo "  kubectl logs -f deployment/goimg-api | grep -i postgres"
echo ""
echo "Rollback if needed:"
echo "  ALTER USER $CURRENT_USER WITH PASSWORD '<old_password>';"
```

### JWT Key Rotation Script

```bash
#!/bin/bash
# /home/user/goimg-datalayer/scripts/rotate_jwt_keys.sh

set -euo pipefail

SECRETS_DIR="${SECRETS_DIR:-/secrets}"
BACKUP_DIR="${BACKUP_DIR:-/backups}"

echo "=== JWT Key Rotation ==="

# Backup current keys
echo "Backing up current keys..."
cp $SECRETS_DIR/jwt_private.pem $BACKUP_DIR/jwt_private_$(date +%Y%m%d).pem
cp $SECRETS_DIR/jwt_public.pem $BACKUP_DIR/jwt_public_$(date +%Y%m%d).pem

# Generate new key pair
echo "Generating new 4096-bit RSA key pair..."
openssl genrsa -out $SECRETS_DIR/jwt_private_new.pem 4096
openssl rsa -in $SECRETS_DIR/jwt_private_new.pem -pubout -out $SECRETS_DIR/jwt_public_new.pem

# Verify key size
KEY_SIZE=$(openssl rsa -in $SECRETS_DIR/jwt_private_new.pem -text -noout | grep "Private-Key" | grep -oP '\d+')
if [[ "$KEY_SIZE" != "4096" ]]; then
    echo "ERROR: Generated key is not 4096 bits!"
    exit 1
fi

# Set permissions
chmod 600 $SECRETS_DIR/jwt_private_new.pem
chmod 644 $SECRETS_DIR/jwt_public_new.pem

echo "New keys generated successfully."
echo ""
echo "Next steps:"
echo "1. Update application config to use dual-key validation:"
echo "   - private_key_path: $SECRETS_DIR/jwt_private_new.pem"
echo "   - public_key_paths: [$SECRETS_DIR/jwt_public_new.pem, $SECRETS_DIR/jwt_public.pem]"
echo "2. Deploy application"
echo "3. Wait 48 hours for grace period"
echo "4. Remove old key from config and redeploy"
echo "5. Archive old keys: $BACKUP_DIR/jwt_private_$(date +%Y%m%d).pem"
```

---

## Post-Rotation Verification

After any rotation, verify:

```bash
# Health checks pass
curl -X GET https://api.goimg.example.com/health | jq

# No authentication errors
kubectl logs -f deployment/goimg-api | grep -i "auth\|credential" | grep -i error

# No database connection errors
kubectl logs -f deployment/goimg-api | grep -i "postgres\|database" | grep -i error

# Metrics show normal operation
curl -s http://prometheus:9090/api/v1/query?query=up{job="goimg-api"} | jq '.data.result[].value[1]'
# Expected: "1"

# Error rate normal
curl -s http://prometheus:9090/api/v1/query?query=rate(http_requests_total{status=~"5.."}[5m]) | jq
# Expected: very low or 0
```

---

## Compliance and Audit

### Documentation Requirements

For each rotation, document:
- Date and time of rotation
- Credential type rotated
- Person who performed rotation
- Reason (routine or emergency)
- Any incidents during rotation
- Rollback performed (yes/no)

**Audit Log Entry**:
```json
{
  "timestamp": "2025-12-05T02:00:00Z",
  "event": "credential_rotation",
  "credential_type": "postgresql_password",
  "performed_by": "security-ops@example.com",
  "reason": "routine_3_month",
  "success": true,
  "downtime_seconds": 0,
  "rollback_required": false
}
```

### SOC 2 Control Mapping

| Control | Activity |
|---------|----------|
| **CC6.1** (Logical Access) | Regular password rotation for database and Redis |
| **CC6.3** (Key Management) | JWT key rotation every 6 months, documented procedures |
| **CC6.6** (Unauthorized Access) | Emergency rotation on suspected compromise |
| **CC7.2** (Monitoring) | Post-rotation verification and monitoring |

---

## Troubleshooting

### Issue: Application Cannot Connect After Rotation

**Symptoms**:
- Health checks failing
- Database connection errors in logs
- 500 errors on API endpoints

**Resolution**:
```bash
# Verify new credential is correct
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT 1"

# Check Kubernetes secret was updated
kubectl get secret goimg-db-credentials -o jsonpath='{.data.password}' | base64 -d

# Verify pods picked up new secret (check restart time)
kubectl get pods -l app=goimg-api -o jsonpath='{.items[*].status.startTime}'

# Force restart if secret not picked up
kubectl rollout restart deployment/goimg-api

# Rollback if persists
# (see specific rotation section for rollback steps)
```

### Issue: Old Credential Still Works After Deletion

**Symptoms**:
- Old password still authenticates
- Old JWT key still validates tokens

**Resolution**:
```bash
# Verify credential deletion
aws iam list-access-keys --user-name goimg-storage  # Should show only new key
redis-cli ACL LIST  # Should show only new user

# Force database connection pool refresh
kubectl rollout restart deployment/goimg-api

# Verify old credential truly deleted
psql -h $DB_HOST -U $OLD_USER -d $DB_NAME
# Expected: authentication failed
```

### Issue: Monitoring Alerts After Rotation

**Symptoms**:
- High error rate alerts
- Authentication failure spike
- JWT validation errors

**Resolution**:
1. Check if error rate is transient (during pod restart)
2. Verify new credential is working (manual test)
3. Review logs for specific error messages
4. Rollback if errors persist beyond 5 minutes

---

## Document Control

**Version History**:
- v1.0 (2025-12-05): Initial creation for Sprint 9

**Related Documents**:
- `/home/user/goimg-datalayer/SECURITY.md` - Vulnerability disclosure policy
- `/home/user/goimg-datalayer/docs/security/incident_response.md` - Incident response procedures
- `/home/user/goimg-datalayer/docs/security/monitoring.md` - Security monitoring guide
- `/home/user/goimg-datalayer/claude/security_gates.md` - Security gate requirements

**Approval**:
- Security Operations Lead: [Pending]
- Engineering Director: [Pending]
- Infrastructure Team Lead: [Pending]
