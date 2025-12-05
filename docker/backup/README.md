# GoImg Database Backup Systemd Units

This directory contains systemd service and timer units for automated PostgreSQL database backups.

## Files

- `goimg-backup.service` - Backup service unit
- `goimg-backup.timer` - Daily backup timer (2:00 AM)
- `goimg-cleanup.service` - Cleanup service unit
- `goimg-cleanup.timer` - Weekly cleanup timer (Sunday 3:00 AM)

## Installation

### 1. Copy systemd units to system directory

```bash
sudo cp docker/backup/*.service /etc/systemd/system/
sudo cp docker/backup/*.timer /etc/systemd/system/
```

### 2. Create environment configuration file

```bash
sudo mkdir -p /etc/goimg
sudo nano /etc/goimg/backup.env
```

Add the following configuration:

```bash
# Database Configuration
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

# S3 Configuration (AWS S3, DigitalOcean Spaces, Backblaze B2)
S3_ENDPOINT=https://s3.amazonaws.com
S3_BUCKET=my-backup-bucket
S3_ACCESS_KEY=your_access_key_here
S3_SECRET_KEY=your_secret_key_here

# GPG Encryption (REQUIRED for security gate S9-PROD-003)
GPG_RECIPIENT=backup@example.com

# Retention Policy
DAILY_RETENTION_DAYS=7
WEEKLY_RETENTION_WEEKS=4
MONTHLY_RETENTION_MONTHS=6
```

### 3. Secure the environment file

```bash
sudo chmod 600 /etc/goimg/backup.env
sudo chown root:root /etc/goimg/backup.env
```

### 4. Create backup directory

```bash
sudo mkdir -p /var/backups/postgres/logs
sudo chown -R root:root /var/backups/postgres
```

### 5. Copy backup scripts to system location

```bash
sudo mkdir -p /opt/goimg/scripts
sudo cp scripts/backup-database.sh /opt/goimg/scripts/
sudo cp scripts/cleanup-old-backups.sh /opt/goimg/scripts/
sudo cp scripts/restore-database.sh /opt/goimg/scripts/
sudo chmod +x /opt/goimg/scripts/*.sh
```

### 6. Reload systemd and enable timers

```bash
# Reload systemd daemon
sudo systemctl daemon-reload

# Enable and start backup timer
sudo systemctl enable goimg-backup.timer
sudo systemctl start goimg-backup.timer

# Enable and start cleanup timer
sudo systemctl enable goimg-cleanup.timer
sudo systemctl start goimg-cleanup.timer

# Verify timers are active
sudo systemctl list-timers --all | grep goimg
```

## Usage

### Manual backup

```bash
# Run backup manually
sudo systemctl start goimg-backup.service

# Check backup status
sudo systemctl status goimg-backup.service

# View backup logs
sudo journalctl -u goimg-backup.service -f
```

### Manual cleanup

```bash
# Run cleanup manually
sudo systemctl start goimg-cleanup.service

# Check cleanup status
sudo systemctl status goimg-cleanup.service

# View cleanup logs
sudo journalctl -u goimg-cleanup.service -f
```

### Check timer schedules

```bash
# List all timers
sudo systemctl list-timers

# Show next scheduled run
sudo systemctl list-timers goimg-backup.timer
sudo systemctl list-timers goimg-cleanup.timer

# View timer status
sudo systemctl status goimg-backup.timer
sudo systemctl status goimg-cleanup.timer
```

### View logs

```bash
# Backup service logs
sudo journalctl -u goimg-backup.service -n 100

# Cleanup service logs
sudo journalctl -u goimg-cleanup.service -n 100

# Follow logs in real-time
sudo journalctl -u goimg-backup.service -f
```

## Monitoring

### Check backup success

```bash
# View recent backup log files
ls -lh /var/backups/postgres/logs/

# Check last backup
tail -n 50 /var/backups/postgres/logs/backup-$(date +%Y%m%d).log

# List local backups
ls -lh /var/backups/postgres/goimg-backup-*.dump*
```

### Check S3 backups

```bash
# List S3 backups (requires AWS CLI configured)
aws s3 ls s3://my-backup-bucket/postgres-backups/ --endpoint-url https://s3.amazonaws.com
```

### Monitor systemd service status

```bash
# Check if services have failed
sudo systemctl --failed | grep goimg

# Check service status
sudo systemctl status goimg-backup.service
sudo systemctl status goimg-cleanup.service
```

## Troubleshooting

### Backup fails

```bash
# Check service logs
sudo journalctl -u goimg-backup.service -n 100 --no-pager

# Verify environment configuration
sudo cat /etc/goimg/backup.env

# Test backup script manually
sudo bash -x /opt/goimg/scripts/backup-database.sh

# Check database connectivity
docker exec goimg-postgres pg_isready -U goimg
```

### Timer not running

```bash
# Check if timer is enabled
sudo systemctl is-enabled goimg-backup.timer

# Check timer status
sudo systemctl status goimg-backup.timer

# Re-enable timer
sudo systemctl enable goimg-backup.timer
sudo systemctl start goimg-backup.timer
```

### S3 upload fails

```bash
# Verify S3 credentials in /etc/goimg/backup.env
# Test S3 connectivity manually
aws s3 ls s3://my-backup-bucket/ --endpoint-url https://s3.amazonaws.com
```

## Uninstallation

```bash
# Stop and disable timers
sudo systemctl stop goimg-backup.timer goimg-cleanup.timer
sudo systemctl disable goimg-backup.timer goimg-cleanup.timer

# Remove systemd units
sudo rm /etc/systemd/system/goimg-backup.service
sudo rm /etc/systemd/system/goimg-backup.timer
sudo rm /etc/systemd/system/goimg-cleanup.service
sudo rm /etc/systemd/system/goimg-cleanup.timer

# Reload systemd
sudo systemctl daemon-reload

# Optionally remove backup files and configuration
# sudo rm -rf /var/backups/postgres
# sudo rm -rf /etc/goimg
# sudo rm -rf /opt/goimg
```

## Security Notes

- The `/etc/goimg/backup.env` file contains sensitive credentials and must be protected with `chmod 600`
- GPG encryption is **required** for production backups (Security Gate S9-PROD-003)
- S3 credentials should use IAM roles with minimum required permissions
- Regular restore testing should be performed to verify backup integrity
- Consider using AWS KMS or similar for additional encryption at rest

## See Also

- [Database Backup Operations Guide](/docs/operations/database-backups.md)
- [Disaster Recovery Runbook](/docs/operations/database-backups.md#disaster-recovery)
