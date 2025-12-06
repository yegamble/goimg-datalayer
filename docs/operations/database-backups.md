# Database Backup Operations Guide

This document provides comprehensive guidance for database backup operations for the goimg-datalayer PostgreSQL database.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Configuration Requirements](#configuration-requirements)
3. [Backup Procedures](#backup-procedures)
4. [Restore Procedures](#restore-procedures)
5. [Monitoring and Alerting](#monitoring-and-alerting)
6. [Disaster Recovery](#disaster-recovery)
7. [Security Considerations](#security-considerations)
8. [Troubleshooting](#troubleshooting)

---

## Architecture Overview

### Backup Strategy

The goimg-datalayer backup system implements a **3-2-1 backup strategy**:

- **3** copies of data (production + 2 backups)
- **2** different storage media (local disk + S3-compatible cloud storage)
- **1** off-site copy (S3 backups)

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                     Backup Architecture                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐                                           │
│  │  PostgreSQL  │                                           │
│  │  Container   │                                           │
│  └──────┬───────┘                                           │
│         │                                                    │
│         │ pg_dump -Fc                                       │
│         │                                                    │
│  ┌──────▼───────────────────────────┐                      │
│  │  backup-database.sh               │                      │
│  │  - Creates compressed dump        │                      │
│  │  - Encrypts with GPG             │                      │
│  │  - Uploads to S3                 │                      │
│  └──────┬───────────────────────────┘                      │
│         │                                                    │
│         ├────────────────────┬──────────────────────────┐  │
│         │                    │                          │  │
│  ┌──────▼──────┐      ┌──────▼─────┐         ┌─────────▼──┐│
│  │   Local     │      │  AWS S3    │         │ DO Spaces  ││
│  │   Storage   │      │ Backblaze  │         │ Other S3   ││
│  │ /var/backups│      │    B2      │         │ Compatible ││
│  └─────────────┘      └────────────┘         └────────────┘│
│         │                                                    │
│  ┌──────▼───────────────────────────┐                      │
│  │  cleanup-old-backups.sh          │                      │
│  │  - Daily: 7 days retention       │                      │
│  │  - Weekly: 4 weeks retention     │                      │
│  │  - Monthly: 6 months retention   │                      │
│  └──────────────────────────────────┘                      │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Retention Policy

| Backup Type | Frequency | Retention Period | Storage Class |
|-------------|-----------|------------------|---------------|
| Daily | Every day at 2:00 AM | 7 days | STANDARD |
| Weekly | Sunday at 2:00 AM | 4 weeks (28 days) | STANDARD_IA |
| Monthly | 1st of month at 2:00 AM | 6 months (180 days) | GLACIER |

**Note**: Weekly backups are automatically identified as Sunday backups, and monthly backups are automatically identified as backups taken on the 1st of the month.

### Backup Naming Convention

Backups follow a consistent naming pattern:

```
goimg-backup-YYYYMMDD-HHMMSS.dump[.gpg]
```

Examples:
- `goimg-backup-20240315-020000.dump` - Unencrypted backup
- `goimg-backup-20240315-020000.dump.gpg` - GPG encrypted backup

---

## Configuration Requirements

### Prerequisites

1. **PostgreSQL Database**
   - PostgreSQL 16+ running in Docker or natively
   - Database user with backup privileges
   - Network access to database

2. **System Requirements**
   - Disk space: Minimum 2x database size for local storage
   - RAM: 512MB minimum for pg_dump
   - CPU: 1 core minimum

3. **Software Dependencies**
   - `pg_dump` and `pg_restore` (PostgreSQL client tools)
   - `docker` (if using Docker-based backups)
   - `gpg` (for encryption)
   - `aws-cli` (for S3 uploads)

### S3-Compatible Storage Setup

#### AWS S3

```bash
# Create S3 bucket
aws s3 mb s3://goimg-backups --region us-east-1

# Create IAM user for backups
aws iam create-user --user-name goimg-backup-user

# Attach policy (see policy below)
aws iam attach-user-policy \
  --user-name goimg-backup-user \
  --policy-arn arn:aws:iam::aws:policy/custom/GoImgBackupPolicy

# Create access keys
aws iam create-access-key --user-name goimg-backup-user
```

**IAM Policy** (`GoImgBackupPolicy`):
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::goimg-backups",
        "arn:aws:s3:::goimg-backups/*"
      ]
    }
  ]
}
```

#### DigitalOcean Spaces

```bash
# Create Space via DigitalOcean control panel or API
# Endpoint: https://nyc3.digitaloceanspaces.com
# Region: nyc3

# Generate Spaces access key from DigitalOcean control panel
# Settings > API > Spaces Keys > Generate New Key
```

#### Backblaze B2

```bash
# Create B2 bucket
b2 create-bucket goimg-backups allPrivate

# Create application key with restricted access
b2 create-key --bucket goimg-backups goimg-backup-key listBuckets,listFiles,readFiles,writeFiles,deleteFiles

# Note the keyID and applicationKey for configuration
```

### GPG Encryption Setup

**Security Gate S9-PROD-003 requires encrypted backups in production.**

#### Generate GPG Key Pair

```bash
# Generate new GPG key
gpg --full-generate-key

# Select:
# - Key type: RSA and RSA
# - Key size: 4096 bits
# - Expiration: 2 years (or as per policy)
# - Real name: GoImg Backup
# - Email: backup@example.com

# Export public key for backup server
gpg --armor --export backup@example.com > goimg-backup-public.key

# Export private key for restore operations (store securely!)
gpg --armor --export-secret-key backup@example.com > goimg-backup-private.key

# Import public key on backup server
gpg --import goimg-backup-public.key

# Trust the key
gpg --edit-key backup@example.com
# > trust
# > 5 (ultimate trust)
# > quit
```

#### Key Management Best Practices

1. **Store private keys securely**:
   - Use a password manager or secrets vault
   - Keep offline backup on encrypted USB drive
   - Store in company safe or secure location

2. **Key rotation**:
   - Rotate GPG keys annually or after security incidents
   - Re-encrypt existing backups with new key during transition

3. **Access control**:
   - Limit access to private keys to authorized personnel only
   - Maintain audit log of key usage

### Environment Configuration

Create `/etc/goimg/backup.env`:

```bash
# Database Connection
DB_HOST=localhost
DB_PORT=5432
DB_NAME=goimg
DB_USER=goimg
DB_PASSWORD=your_secure_password_here

# Backup Storage
BACKUP_DIR=/var/backups/postgres

# Docker Configuration
USE_DOCKER=true
DOCKER_CONTAINER=goimg-postgres

# S3 Configuration
S3_ENDPOINT=https://s3.amazonaws.com
S3_BUCKET=goimg-backups
S3_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE
S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# GPG Encryption (REQUIRED)
GPG_RECIPIENT=backup@example.com

# Retention Policy
DAILY_RETENTION_DAYS=7
WEEKLY_RETENTION_WEEKS=4
MONTHLY_RETENTION_MONTHS=6

# Optional: Debug mode
# DEBUG=true
```

**Secure the file**:
```bash
sudo chmod 600 /etc/goimg/backup.env
sudo chown root:root /etc/goimg/backup.env
```

---

## Backup Procedures

### Automated Backups

There are two approaches for automated backups, depending on your deployment model:

#### Option 1: Docker Container (Recommended for Containerized Deployments)

The backup service runs as a Docker container with cron scheduling built-in.

**Setup**:

```bash
# 1. Configure environment variables in .env or docker-compose.yml
export S3_ENDPOINT=https://s3.amazonaws.com
export S3_BUCKET=goimg-backups
export S3_ACCESS_KEY=your_access_key
export GPG_RECIPIENT=backup@example.com

# 2. Create secrets directory for GPG keys
mkdir -p docker/backup/gpg-keys
gpg --export backup@example.com > docker/backup/gpg-keys/public.key

# 3. Start backup service with docker-compose
docker-compose -f docker/docker-compose.prod.yml up -d backup

# 4. Verify backup service is running
docker logs goimg-backup

# 5. Monitor backup logs
docker exec goimg-backup tail -f /var/backups/postgres/logs/cron.log
```

**Configuration**:

The backup container runs on the following schedule:
- Daily backup: 2:00 AM every day
- Weekly cleanup: 3:00 AM every Sunday

Environment variables are configured in `docker-compose.prod.yml`:
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER` - Database connection
- `S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY` - S3 storage configuration
- `GPG_RECIPIENT` - Email/key ID for GPG encryption
- `DAILY_RETENTION_DAYS=7` - Daily backup retention
- `WEEKLY_RETENTION_WEEKS=4` - Weekly backup retention
- `MONTHLY_RETENTION_MONTHS=6` - Monthly backup retention

#### Option 2: Systemd Timers (Recommended for VM/Bare Metal)

For deployments on VMs or bare metal, systemd timers provide native system integration.

**Installation**:

See [systemd installation guide](../../docker/backup/README.md) for detailed setup instructions.

#### Verification (Docker Container)

```bash
# Check container status
docker ps | grep goimg-backup

# View backup schedule (cron jobs)
docker exec goimg-backup crontab -l

# Check recent backup logs
docker exec goimg-backup tail -n 50 /var/backups/postgres/logs/cron.log

# List backups
docker exec goimg-backup ls -lh /var/backups/postgres/

# Manually trigger backup (for testing)
docker exec goimg-backup /opt/goimg/scripts/backup-database.sh

# Manually trigger cleanup
docker exec goimg-backup /opt/goimg/scripts/cleanup-old-backups.sh --dry-run
```

#### Verification (Systemd Timers)

```bash
# Check timer status
sudo systemctl status goimg-backup.timer

# View next scheduled run
sudo systemctl list-timers goimg-backup.timer

# Check recent backup logs
sudo journalctl -u goimg-backup.service -n 50
```

### Manual Backup

#### Using the Backup Script

```bash
# Set required environment variables
export DB_PASSWORD=your_password
export S3_ENDPOINT=https://s3.amazonaws.com
export S3_BUCKET=goimg-backups
export S3_ACCESS_KEY=your_access_key
export S3_SECRET_KEY=your_secret_key
export GPG_RECIPIENT=backup@example.com

# Run backup
/opt/goimg/scripts/backup-database.sh
```

#### Using Docker Directly

```bash
# Create backup manually
docker exec -e PGPASSWORD=your_password goimg-postgres \
  pg_dump -U goimg -d goimg -Fc -Z 9 \
  > /var/backups/postgres/manual-backup-$(date +%Y%m%d-%H%M%S).dump

# Encrypt
gpg --encrypt \
  --recipient backup@example.com \
  --trust-model always \
  /var/backups/postgres/manual-backup-*.dump

# Upload to S3
aws s3 cp /var/backups/postgres/manual-backup-*.dump.gpg \
  s3://goimg-backups/postgres-backups/ \
  --endpoint-url https://s3.amazonaws.com
```

### Backup Verification

Always verify backups after creation:

```bash
# List backup contents
pg_restore --list /var/backups/postgres/goimg-backup-*.dump

# For encrypted backups, decrypt first
gpg --decrypt /var/backups/postgres/goimg-backup-*.dump.gpg | \
  pg_restore --list

# Verify file integrity
ls -lh /var/backups/postgres/
md5sum /var/backups/postgres/goimg-backup-*.dump
```

### Pre-Upgrade Backups

Before major upgrades, create a labeled backup:

```bash
# Create pre-upgrade backup
export DB_PASSWORD=your_password
BACKUP_DIR=/var/backups/postgres/upgrades /opt/goimg/scripts/backup-database.sh

# Tag the backup
mv /var/backups/postgres/upgrades/goimg-backup-*.dump \
   /var/backups/postgres/upgrades/goimg-backup-pre-v2.0-upgrade-$(date +%Y%m%d).dump
```

---

## Restore Procedures

### Full Database Restore

#### From Local Backup

```bash
# Set environment variables
export DB_PASSWORD=your_password

# List available backups
ls -lh /var/backups/postgres/goimg-backup-*.dump*

# Restore (will prompt for confirmation)
/opt/goimg/scripts/restore-database.sh \
  --file /var/backups/postgres/goimg-backup-20240315-020000.dump
```

#### From S3 Backup

```bash
# Set environment variables
export DB_PASSWORD=your_password
export S3_ENDPOINT=https://s3.amazonaws.com
export S3_BUCKET=goimg-backups
export S3_ACCESS_KEY=your_access_key
export S3_SECRET_KEY=your_secret_key

# List S3 backups
aws s3 ls s3://goimg-backups/postgres-backups/ --endpoint-url https://s3.amazonaws.com

# Restore from S3
/opt/goimg/scripts/restore-database.sh \
  --s3-key postgres-backups/goimg-backup-20240315-020000.dump.gpg
```

#### From Encrypted Backup

The restore script automatically detects and decrypts GPG-encrypted backups:

```bash
# GPG will prompt for private key passphrase
/opt/goimg/scripts/restore-database.sh \
  --file /var/backups/postgres/goimg-backup-20240315-020000.dump.gpg
```

### Dry Run (Validation Only)

Test restore without making changes:

```bash
/opt/goimg/scripts/restore-database.sh \
  --file /var/backups/postgres/goimg-backup-20240315-020000.dump \
  --dry-run
```

### Force Restore (Skip Confirmation)

For automated restore processes:

```bash
/opt/goimg/scripts/restore-database.sh \
  --file /var/backups/postgres/goimg-backup-20240315-020000.dump \
  --force
```

### Partial Restore

Restore specific tables or schemas:

```bash
# Extract table list
pg_restore --list /var/backups/postgres/goimg-backup-*.dump > restore.list

# Edit restore.list to comment out unwanted tables

# Restore with list
docker exec -i goimg-postgres \
  pg_restore -U goimg -d goimg \
  --use-list=restore.list \
  < /var/backups/postgres/goimg-backup-*.dump
```

### Point-in-Time Recovery

For point-in-time recovery, you need:
1. Most recent base backup before target time
2. WAL archives (if configured)

**Note**: WAL archiving is not currently configured. Consider implementing for critical environments.

---

## Monitoring and Alerting

### Backup Health Checks

#### Daily Verification Checklist

```bash
# 1. Check systemd timer status
sudo systemctl status goimg-backup.timer

# 2. Verify recent backup exists
ls -lh /var/backups/postgres/goimg-backup-$(date +%Y%m%d)*.dump*

# 3. Check backup logs for errors
sudo journalctl -u goimg-backup.service --since today | grep -i error

# 4. Verify S3 upload
aws s3 ls s3://goimg-backups/postgres-backups/ \
  --endpoint-url https://s3.amazonaws.com | grep $(date +%Y%m%d)

# 5. Check disk space
df -h /var/backups/postgres
```

#### Automated Monitoring Script

Create `/opt/goimg/scripts/check-backup-health.sh`:

```bash
#!/bin/bash
set -euo pipefail

TODAY=$(date +%Y%m%d)
BACKUP_DIR=/var/backups/postgres
ALERT_EMAIL=ops@example.com

# Check if today's backup exists
if ! ls ${BACKUP_DIR}/goimg-backup-${TODAY}*.dump* &> /dev/null; then
  echo "ERROR: No backup found for ${TODAY}" | mail -s "Backup Alert" ${ALERT_EMAIL}
  exit 1
fi

# Check backup age (should be < 30 hours)
LATEST_BACKUP=$(ls -t ${BACKUP_DIR}/goimg-backup-*.dump* | head -1)
BACKUP_AGE=$(($(date +%s) - $(stat -c %Y ${LATEST_BACKUP})))

if [ ${BACKUP_AGE} -gt 108000 ]; then
  echo "ERROR: Latest backup is older than 30 hours" | mail -s "Backup Alert" ${ALERT_EMAIL}
  exit 1
fi

echo "Backup health check passed"
exit 0
```

### Prometheus Metrics

Export backup metrics for Prometheus:

```bash
# Create metrics file
cat > /var/lib/prometheus/node-exporter/backup.prom <<EOF
# HELP postgres_backup_last_success_timestamp Last successful backup timestamp
# TYPE postgres_backup_last_success_timestamp gauge
postgres_backup_last_success_timestamp{database="goimg"} $(stat -c %Y $(ls -t /var/backups/postgres/goimg-backup-*.dump* | head -1))

# HELP postgres_backup_size_bytes Last backup size in bytes
# TYPE postgres_backup_size_bytes gauge
postgres_backup_size_bytes{database="goimg"} $(stat -c %s $(ls -t /var/backups/postgres/goimg-backup-*.dump* | head -1))

# HELP postgres_backup_count Total number of backups
# TYPE postgres_backup_count gauge
postgres_backup_count{database="goimg"} $(ls /var/backups/postgres/goimg-backup-*.dump* | wc -l)
EOF
```

### Alerting Rules

#### Prometheus Alert Rules

```yaml
groups:
  - name: database_backups
    interval: 5m
    rules:
      - alert: BackupTooOld
        expr: (time() - postgres_backup_last_success_timestamp) > 108000
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "PostgreSQL backup is too old"
          description: "Last backup was {{ $value | humanizeDuration }} ago"

      - alert: BackupSizeAnomalous
        expr: |
          abs(postgres_backup_size_bytes -
              avg_over_time(postgres_backup_size_bytes[7d])) /
              avg_over_time(postgres_backup_size_bytes[7d]) > 0.5
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Backup size anomalous"
          description: "Backup size changed by more than 50% from 7-day average"

      - alert: BackupFailed
        expr: up{job="goimg-backup"} == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Backup job failed"
          description: "Backup systemd service is not running"
```

---

## Disaster Recovery

### Disaster Recovery Runbook

This runbook covers complete database loss scenarios.

#### Prerequisites

- Access to S3 backup storage
- GPG private key for decryption
- Clean PostgreSQL instance or ability to create one

#### DR Steps

**Step 1: Assess Damage**

```bash
# Check if PostgreSQL is running
docker ps | grep goimg-postgres

# Check database accessibility
docker exec goimg-postgres psql -U goimg -d goimg -c "SELECT NOW();"

# If database is corrupted but container is running, proceed to restore
# If container is lost, proceed to container recreation
```

**Step 2: Stop Application Services**

```bash
# Stop API and worker to prevent write attempts
docker stop goimg-api goimg-worker

# Verify no active connections
docker exec goimg-postgres psql -U goimg -d postgres \
  -c "SELECT count(*) FROM pg_stat_activity WHERE datname = 'goimg';"
```

**Step 3: Identify Restore Point**

```bash
# List available S3 backups
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key

aws s3 ls s3://goimg-backups/postgres-backups/ \
  --endpoint-url https://s3.amazonaws.com | sort -r

# Select the most recent backup before the incident
# Example: goimg-backup-20240315-020000.dump.gpg
```

**Step 4: Perform Restore**

```bash
# Set environment variables
export DB_PASSWORD=your_password
export S3_ENDPOINT=https://s3.amazonaws.com
export S3_BUCKET=goimg-backups
export S3_ACCESS_KEY=your_access_key
export S3_SECRET_KEY=your_secret_key

# Execute restore
/opt/goimg/scripts/restore-database.sh \
  --s3-key postgres-backups/goimg-backup-20240315-020000.dump.gpg \
  --force

# Check restore logs
tail -f /var/backups/postgres/logs/restore-*.log
```

**Step 5: Verify Database Integrity**

```bash
# Check table counts
docker exec goimg-postgres psql -U goimg -d goimg -c \
  "SELECT schemaname, tablename, n_tup_ins, n_tup_upd, n_tup_del
   FROM pg_stat_user_tables;"

# Run application healthchecks
docker exec goimg-postgres psql -U goimg -d goimg -c \
  "SELECT COUNT(*) FROM users;"
docker exec goimg-postgres psql -U goimg -d goimg -c \
  "SELECT COUNT(*) FROM images;"

# Verify critical data
# (Add application-specific verification queries)
```

**Step 6: Restart Application**

```bash
# Start worker first
docker start goimg-worker

# Wait 30 seconds for background jobs to initialize
sleep 30

# Start API
docker start goimg-api

# Monitor logs
docker logs -f goimg-api
docker logs -f goimg-worker
```

**Step 7: Post-Recovery Validation**

```bash
# Test critical endpoints
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/images?limit=10

# Check for errors in application logs
docker logs goimg-api --since 10m | grep -i error

# Verify background jobs are processing
docker logs goimg-worker --since 5m
```

**Step 8: Document Incident**

Create incident report including:
- Incident timeline
- Root cause analysis
- Data loss assessment (if any)
- Restore point selected
- Recovery time (RTO achieved)
- Lessons learned
- Action items

#### Recovery Time Objective (RTO)

**Target RTO: 30 minutes** (Security Gate S9-PROD-004)

**Actual Measured RTO**: See validation reports from automated testing

Breakdown:
- Incident detection: 5 minutes
- Assessment and decision: 10 minutes
- Backup download from S3: 10 minutes
- Database restore: 20 minutes
- Verification and testing: 10 minutes
- Application restart: 5 minutes

#### Recovery Point Objective (RPO)

**Target RPO: 24 hours**

- Daily backups at 2:00 AM
- Maximum data loss: Data created/modified since last backup
- Consider implementing WAL archiving for RPO < 1 hour

### Testing Disaster Recovery

**Monthly DR drill procedure**:

```bash
# 1. Create test environment
docker-compose -f docker-compose.test.yml up -d postgres-test

# 2. Restore latest production backup to test environment
export DB_HOST=postgres-test
export DB_PORT=5433
/opt/goimg/scripts/restore-database.sh \
  --file /var/backups/postgres/goimg-backup-latest.dump \
  --force

# 3. Verify data integrity
docker exec postgres-test psql -U goimg -d goimg -f tests/verify-data.sql

# 4. Time the restore process
# 5. Document results
# 6. Cleanup
docker-compose -f docker-compose.test.yml down -v
```

---

## Security Considerations

### Encryption

**Security Gate S9-PROD-003: Backups must be encrypted**

1. **At Rest**:
   - All backups MUST be GPG encrypted
   - Use strong GPG keys (4096-bit RSA minimum)
   - Rotate encryption keys annually

2. **In Transit**:
   - S3 uploads use HTTPS/TLS
   - Verify S3 endpoint uses valid SSL certificate

3. **Key Management**:
   - Store GPG private keys in secure vault
   - Implement key rotation policy
   - Maintain offline backup of private keys

### Access Control

1. **Backup Script Permissions**:
   ```bash
   chmod 700 /opt/goimg/scripts/backup-database.sh
   chmod 700 /opt/goimg/scripts/restore-database.sh
   chown root:root /opt/goimg/scripts/*.sh
   ```

2. **Environment File Permissions**:
   ```bash
   chmod 600 /etc/goimg/backup.env
   chown root:root /etc/goimg/backup.env
   ```

3. **Backup Directory Permissions**:
   ```bash
   chmod 750 /var/backups/postgres
   chown root:postgres /var/backups/postgres
   ```

4. **S3 Bucket Policy**:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Deny",
         "Principal": "*",
         "Action": "s3:*",
         "Resource": [
           "arn:aws:s3:::goimg-backups/*"
         ],
         "Condition": {
           "Bool": {
             "aws:SecureTransport": "false"
           }
         }
       }
     ]
   }
   ```

### Audit Logging

Enable audit logging for backup operations:

```bash
# Add to /etc/audit/rules.d/backup.rules
-w /opt/goimg/scripts/backup-database.sh -p x -k backup_execution
-w /opt/goimg/scripts/restore-database.sh -p x -k restore_execution
-w /etc/goimg/backup.env -p r -k backup_config_access
-w /var/backups/postgres -p wa -k backup_file_changes

# Reload audit rules
sudo augenrules --load
```

### Compliance

For compliance requirements:
- **GDPR**: Ensure backups are encrypted and access is logged
- **HIPAA**: Implement encryption at rest and in transit
- **SOC 2**: Maintain audit logs and implement least privilege access
- **PCI DSS**: Use strong encryption and key management

---

## Troubleshooting

### Common Issues

#### Issue: Backup Script Fails with "Permission Denied"

**Symptoms**:
```
ERROR: permission denied for database
```

**Solution**:
```bash
# Verify database user has backup permissions
docker exec goimg-postgres psql -U postgres -c \
  "GRANT SELECT ON ALL TABLES IN SCHEMA public TO goimg;"

# For full backup capability, ensure user is superuser or has pg_dump privileges
docker exec goimg-postgres psql -U postgres -c \
  "ALTER USER goimg WITH SUPERUSER;"
```

#### Issue: S3 Upload Fails

**Symptoms**:
```
ERROR: S3 upload failed
```

**Solution**:
```bash
# Test S3 connectivity
aws s3 ls s3://goimg-backups/ --endpoint-url https://s3.amazonaws.com

# Verify credentials
echo $S3_ACCESS_KEY
echo $S3_SECRET_KEY

# Test with verbose output
aws s3 cp /tmp/test.txt s3://goimg-backups/ \
  --endpoint-url https://s3.amazonaws.com \
  --debug

# Check bucket permissions and IAM policy
```

#### Issue: GPG Encryption Fails

**Symptoms**:
```
ERROR: Encryption failed
gpg: recipient not found
```

**Solution**:
```bash
# List available GPG keys
gpg --list-keys

# Import public key if missing
gpg --import goimg-backup-public.key

# Trust the key
gpg --edit-key backup@example.com
# > trust
# > 5
# > quit

# Test encryption manually
echo "test" | gpg --encrypt --recipient backup@example.com
```

#### Issue: Restore Fails with "Database in Use"

**Symptoms**:
```
ERROR: database "goimg" is being accessed by other users
```

**Solution**:
```bash
# Terminate all connections
docker exec goimg-postgres psql -U goimg -d postgres -c \
  "SELECT pg_terminate_backend(pid) FROM pg_stat_activity
   WHERE datname = 'goimg' AND pid <> pg_backend_pid();"

# Stop application services
docker stop goimg-api goimg-worker

# Retry restore
```

#### Issue: Backup Size Unexpectedly Large

**Symptoms**:
Backup files are significantly larger than expected.

**Solution**:
```bash
# Analyze database size
docker exec goimg-postgres psql -U goimg -d goimg -c \
  "SELECT pg_size_pretty(pg_database_size('goimg'));"

# Find large tables
docker exec goimg-postgres psql -U goimg -d goimg -c \
  "SELECT schemaname, tablename,
          pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
   FROM pg_tables
   ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
   LIMIT 10;"

# Check for bloat
docker exec goimg-postgres psql -U goimg -d goimg -c \
  "SELECT schemaname, tablename,
          pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) as size,
          pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) -
                         pg_relation_size(schemaname||'.'||tablename)) as external_size
   FROM pg_tables
   ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"

# Run VACUUM to reclaim space
docker exec goimg-postgres psql -U goimg -d goimg -c "VACUUM FULL VERBOSE;"
```

#### Issue: Systemd Timer Not Running

**Symptoms**:
```bash
$ sudo systemctl status goimg-backup.timer
● goimg-backup.timer - inactive (dead)
```

**Solution**:
```bash
# Enable the timer
sudo systemctl enable goimg-backup.timer

# Start the timer
sudo systemctl start goimg-backup.timer

# Verify status
sudo systemctl status goimg-backup.timer

# Check timer schedule
sudo systemctl list-timers goimg-backup.timer

# View logs
sudo journalctl -u goimg-backup.timer -n 50
```

### Debug Mode

Enable debug logging:

```bash
# Add to /etc/goimg/backup.env
DEBUG=true

# Run backup manually to see debug output
sudo bash -x /opt/goimg/scripts/backup-database.sh
```

### Support Contacts

- **Infrastructure Team**: infrastructure@example.com
- **On-Call Engineer**: oncall@example.com
- **Escalation**: CTO or VP Engineering

---

## Appendix

### Backup Script Exit Codes

| Exit Code | Meaning |
|-----------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Missing dependencies |
| 3 | Configuration error |
| 4 | Backup creation failed |
| 5 | Encryption failed |
| 6 | Upload failed |

### Restore Script Exit Codes

| Exit Code | Meaning |
|-----------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Missing dependencies |
| 3 | Configuration error |
| 4 | Download failed |
| 5 | Decryption failed |
| 6 | Restore failed |

### Useful Commands Reference

```bash
# List all backups with sizes
ls -lhS /var/backups/postgres/

# Find backups older than 30 days
find /var/backups/postgres -name "goimg-backup-*.dump*" -mtime +30

# Calculate total backup storage usage
du -sh /var/backups/postgres/

# List S3 backups sorted by date
aws s3 ls s3://goimg-backups/postgres-backups/ --endpoint-url https://s3.amazonaws.com | sort

# Download specific S3 backup
aws s3 cp s3://goimg-backups/postgres-backups/goimg-backup-20240315-020000.dump.gpg \
  /tmp/ --endpoint-url https://s3.amazonaws.com

# Decrypt backup manually
gpg --decrypt /var/backups/postgres/goimg-backup-20240315-020000.dump.gpg \
  > /tmp/decrypted-backup.dump

# View backup metadata
pg_restore --list /tmp/decrypted-backup.dump | head -20

# Test restore to temporary database
createdb -U goimg goimg_test
pg_restore -U goimg -d goimg_test /tmp/decrypted-backup.dump
psql -U goimg -d goimg_test -c "SELECT COUNT(*) FROM users;"
dropdb -U goimg goimg_test
```

---

**Document Version**: 1.0
**Last Updated**: 2024-03-15
**Maintained By**: Infrastructure Team
**Review Cycle**: Quarterly
