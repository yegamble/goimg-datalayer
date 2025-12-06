# Data Retention and Privacy Policy

> Data lifecycle management, retention periods, and GDPR/CCPA compliance for goimg-datalayer

## Overview

This document defines data retention policies, deletion procedures, and compliance with privacy regulations (GDPR, CCPA) for the goimg-datalayer platform. Proper data lifecycle management balances operational needs, legal requirements, and user privacy rights.

**Document Owner**: Legal & Compliance Team, Security Operations Team
**Last Updated**: 2025-12-05
**Review Frequency**: Annually or when regulations change

---

## Regulatory Compliance

### GDPR (General Data Protection Regulation)

**Scope**: Applies to all users in the European Economic Area (EEA), UK, and Switzerland.

**Key Requirements**:
- **Article 5(1)(e)**: Data minimization - keep personal data no longer than necessary
- **Article 17**: Right to erasure ("right to be forgotten")
- **Article 20**: Right to data portability
- **Article 32**: Security of processing (encryption, pseudonymization)
- **Article 33/34**: Breach notification (72 hours to supervisory authority)

**Data Controller**: goimg-datalayer (your organization name)
**Data Protection Officer**: privacy@goimg-datalayer.example.com

### CCPA (California Consumer Privacy Act)

**Scope**: Applies to California residents.

**Key Requirements**:
- **Right to Know**: Users can request what personal data is collected
- **Right to Delete**: Users can request deletion of personal data
- **Right to Opt-Out**: Users can opt-out of data sale (not applicable - we don't sell data)
- **Non-Discrimination**: Cannot discriminate against users exercising privacy rights

**Privacy Contact**: privacy@goimg-datalayer.example.com

### Additional Regulations

- **UK GDPR**: Substantially similar to EU GDPR
- **PIPEDA** (Canada): Similar data protection requirements
- **LGPD** (Brazil): Brazilian data protection law

---

## Data Classification

### Personal Data (Personally Identifiable Information - PII)

Data that can identify an individual:

| Data Type | Examples | GDPR Category | Storage Location |
|-----------|----------|---------------|------------------|
| **Direct Identifiers** | Email address, username, full name | Personal data | PostgreSQL `users` table |
| **Authentication** | Password hash (Argon2id), session tokens | Personal data | PostgreSQL `users`, Redis sessions |
| **IP Addresses** | Login IP, upload IP | Personal data | PostgreSQL `audit_log`, application logs |
| **User-Generated Content** | Image titles, descriptions, comments | Personal data | PostgreSQL `images`, `comments` |
| **Metadata** | Account creation date, last login, preferences | Personal data | PostgreSQL `users` |
| **Device Information** | User-Agent strings | Personal data | Application logs |

### Non-Personal Data

Data that cannot identify individuals:

| Data Type | Examples | Storage Location |
|-----------|----------|------------------|
| **Aggregated Statistics** | Total uploads, average image size | Analytics database |
| **System Metrics** | API response times, error rates | Prometheus, Grafana |
| **Anonymized Logs** | IP-stripped security events (after anonymization) | Long-term archive |

### Special Categories (Sensitive Data)

Data requiring extra protection under GDPR Article 9:

| Category | Handling | Notes |
|----------|----------|-------|
| **Images with faces** | Not explicitly collected, but users may upload | EXIF stripped, no facial recognition |
| **Biometric data** | Not collected | N/A |
| **Health data** | Not collected | N/A |
| **Political/religious content** | User-generated (optional tags) | No automatic processing |

**Policy**: We do not intentionally collect special category data. If users upload such content, it is treated as user-generated content with no special processing.

---

## Retention Periods

### Active User Data

**Retention**: While account is active + retention period after account deletion.

| Data Type | Active Retention | Post-Deletion Retention | Rationale |
|-----------|------------------|-------------------------|-----------|
| **User Profile** | Indefinite (while active) | 30 days | Grace period for account recovery |
| **Email Address** | Indefinite (while active) | 90 days (hashed) | Prevent duplicate account creation |
| **Password Hash** | Indefinite (while active) | 0 days (immediate deletion) | Security - no need to retain |
| **Uploaded Images** | Indefinite (while active) | 30 days | Grace period for account recovery |
| **Comments/Likes** | Indefinite (while active) | 0 days (anonymized) | Convert to "deleted user" |
| **Session Tokens** | 7 days (refresh token TTL) | 0 days | Expire naturally |

### Security and Audit Logs

**Retention**: Based on security, compliance, and forensic needs.

| Log Type | Hot Storage | Cold Storage | Total Retention | Rationale |
|----------|-------------|--------------|-----------------|-----------|
| **Security Events** (auth failures, malware, escalation) | 90 days (PostgreSQL) | 7 years (S3 Glacier, encrypted) | 7 years | Legal/compliance, forensics |
| **Audit Logs** (user actions) | 90 days (PostgreSQL) | 1 year (S3 Standard-IA) | 1 year + 90 days | Compliance, investigations |
| **Application Logs** | 30 days (Elasticsearch) | N/A | 30 days | Debugging, operational |
| **Access Logs** (nginx) | 30 days (local disk) | N/A | 30 days | Performance, debugging |
| **Forensic Snapshots** | N/A | 7 years (S3 Glacier, encrypted) | 7 years | Legal hold, evidence |

**Automated Archival**: Cron job runs daily at 2 AM UTC to move logs from hot to cold storage.

### Backups

**Retention**: For disaster recovery and compliance.

| Backup Type | Frequency | Retention Period | Storage Location |
|-------------|-----------|------------------|------------------|
| **Full Database Backup** | Daily | 30 days | S3 Standard (encrypted) |
| **Incremental Backup** | Every 6 hours | 7 days | S3 Standard (encrypted) |
| **Point-in-Time Recovery** | Continuous (WAL) | 7 days | S3 Standard (encrypted) |
| **Annual Archive** | Yearly | 7 years | S3 Glacier (encrypted) |

**Backup Encryption**: All backups encrypted with AES-256 using AWS KMS.

**Note**: Backups may contain deleted user data during retention window. Backups older than retention period are purged according to schedule.

### Analytics and Metrics

| Metric Type | Retention Period | Storage |
|-------------|------------------|---------|
| **Prometheus Metrics** | 90 days | Prometheus TSDB |
| **Grafana Dashboards** | Indefinite (no PII) | Grafana DB |
| **Aggregated Statistics** | Indefinite (anonymized) | Analytics DB |

---

## User Data Rights (GDPR/CCPA Compliance)

### Right to Access (GDPR Article 15, CCPA Right to Know)

**User Request**: "What personal data do you have about me?"

**Response Process**:

1. **Verification**: Verify user identity (login + email confirmation)
2. **Data Export**: Generate comprehensive data export within **30 days**
3. **Format**: Machine-readable JSON format
4. **Delivery**: Secure download link (expires in 7 days)

**Data Export Contents**:
```json
{
  "user_profile": {
    "id": "uuid",
    "email": "user@example.com",
    "username": "username",
    "created_at": "2025-01-01T00:00:00Z",
    "last_login": "2025-12-05T14:32:18Z"
  },
  "uploaded_images": [
    {
      "id": "image-uuid",
      "title": "My Image",
      "description": "Description",
      "uploaded_at": "2025-11-01T10:00:00Z",
      "visibility": "public",
      "view_count": 42,
      "download_url": "https://..."
    }
  ],
  "comments": [...],
  "likes": [...],
  "audit_log": [...]
}
```

**Implementation**:
```bash
# Generate data export
curl -X POST https://api.goimg.example.com/api/v1/users/me/export \
  -H "Authorization: Bearer $USER_TOKEN"

# Response: { "export_id": "uuid", "status": "processing" }

# Check export status
curl -X GET https://api.goimg.example.com/api/v1/users/me/exports/{export_id} \
  -H "Authorization: Bearer $USER_TOKEN"

# Download when ready
curl -X GET https://api.goimg.example.com/api/v1/users/me/exports/{export_id}/download \
  -H "Authorization: Bearer $USER_TOKEN" > my_data.json
```

### Right to Erasure / Deletion (GDPR Article 17, CCPA Right to Delete)

**User Request**: "Delete all my personal data."

**Response Process**:

1. **Verification**: Verify user identity (login + email confirmation + password re-entry)
2. **Grace Period**: 30-day grace period for accidental deletions
3. **Soft Delete**: Mark account as `deleted`, hide from public
4. **Hard Delete**: After grace period, permanently delete data
5. **Notification**: Email confirmation of deletion

**Deletion Procedure**:

```sql
-- Step 1: Soft delete (immediate)
UPDATE users
SET status = 'deleted',
    deleted_at = NOW(),
    email = CONCAT('deleted_', id, '@deleted.local'),  -- Anonymize
    username = CONCAT('deleted_user_', id)
WHERE id = 'user-uuid';

-- Revoke all sessions immediately
DELETE FROM sessions WHERE user_id = 'user-uuid';

-- Step 2: Hard delete (after 30-day grace period)
-- Run nightly cleanup job

-- Delete user-generated content
DELETE FROM comments WHERE user_id = 'user-uuid';
DELETE FROM likes WHERE user_id = 'user-uuid';

-- Delete images and variants
DELETE FROM image_variants WHERE image_id IN (
  SELECT id FROM images WHERE owner_id = 'user-uuid'
);
DELETE FROM images WHERE owner_id = 'user-uuid';

-- Delete albums
DELETE FROM album_images WHERE album_id IN (
  SELECT id FROM albums WHERE owner_id = 'user-uuid'
);
DELETE FROM albums WHERE owner_id = 'user-uuid';

-- Delete user record
DELETE FROM users WHERE id = 'user-uuid' AND deleted_at < NOW() - INTERVAL '30 days';
```

**Storage Cleanup**:
```bash
# Delete user's images from S3
aws s3 rm s3://goimg-uploads/users/{user-uuid}/ --recursive

# Delete from IPFS (if pinned)
# Unpin all user's CIDs from Pinata/Infura
```

**Exceptions to Deletion** (GDPR Article 17(3)):

We may retain data despite deletion request if:
- **Legal obligation**: Compliance with laws (tax records, anti-fraud)
- **Legal claims**: Defense of legal claims (7-year retention for legal hold)
- **Public interest**: Archival, scientific research (anonymized only)

**Retention After Deletion**:
- Hashed email (90 days) - prevent duplicate account creation
- Anonymized audit logs (per retention policy) - IP removed, user_id pseudonymized
- Backups (per backup retention policy) - purged when backup expires

### Right to Rectification (GDPR Article 16)

**User Request**: "Correct inaccurate data about me."

**Self-Service**: Users can update profile via UI (`/settings/profile`)

**Admin Correction** (if needed):
```sql
UPDATE users
SET email = 'corrected@example.com',
    updated_at = NOW()
WHERE id = 'user-uuid';
```

### Right to Data Portability (GDPR Article 20)

**User Request**: "Provide my data in a portable format."

**Implementation**: Same as Right to Access (JSON export)

**Additional Format Support** (future):
- CSV for images/albums
- ZIP archive with original image files

### Right to Restrict Processing (GDPR Article 18)

**User Request**: "Stop processing my data temporarily."

**Implementation**:
```sql
UPDATE users
SET status = 'restricted',
    restriction_reason = 'user_request',
    updated_at = NOW()
WHERE id = 'user-uuid';
```

**Effect**: Account remains active but:
- Images set to private automatically
- No email notifications sent
- Data not used for analytics
- Can log in to unrestrict

### Right to Object (GDPR Article 21)

**User Request**: "I object to processing for direct marketing."

**Implementation**: We do not currently engage in direct marketing. If added:
```sql
UPDATE users
SET email_preferences = jsonb_set(email_preferences, '{marketing}', 'false')
WHERE id = 'user-uuid';
```

---

## Anonymization and Pseudonymization

### Anonymization Techniques

**Definition**: Irreversibly removing PII so data cannot be linked to an individual.

**Use Cases**:
- Long-term analytics
- Public statistics
- Research datasets

**Techniques**:

1. **IP Address Anonymization**:
   ```sql
   -- Anonymize IP addresses in audit logs (remove last octet)
   UPDATE audit_log
   SET client_ip = host(network(client_ip) + '0.0.0.0/24')::inet
   WHERE timestamp < NOW() - INTERVAL '90 days';
   ```

2. **User ID Pseudonymization**:
   ```sql
   -- Replace user_id with pseudonymous hash
   UPDATE audit_log_archive
   SET user_id_hash = encode(digest(user_id::text, 'sha256'), 'hex'),
       user_id = NULL
   WHERE timestamp < NOW() - INTERVAL '1 year';
   ```

3. **Comment Anonymization** (deleted users):
   ```sql
   UPDATE comments
   SET user_id = '00000000-0000-0000-0000-000000000000',  -- Sentinel "deleted user"
       username = 'deleted_user'
   WHERE user_id IN (SELECT id FROM users WHERE status = 'deleted');
   ```

### Pseudonymization (GDPR Article 4)

**Definition**: Processing data so it cannot be attributed to an individual without additional information (key).

**Use Case**: Separate identifying data from content data.

**Example**:
- Store user ID as UUID (not sequential integer)
- Hash email addresses for duplicate detection
- Use session IDs instead of user IDs in Redis

---

## Data Breach Notification

### GDPR Breach Notification (Article 33/34)

**Timeline**:
- **72 hours**: Notify supervisory authority (if high risk to rights)
- **Without undue delay**: Notify affected users (if high risk)

**Breach Assessment**:

| Risk Level | Criteria | Notification Required |
|------------|----------|----------------------|
| **High** | Password hashes exposed, full database breach, sensitive images leaked | Users + Authority |
| **Medium** | Email addresses exposed, IP addresses leaked | Authority only |
| **Low** | Anonymized data only, no PII exposed | None (document internally) |

**Notification Process**:

1. **Detect**: Security monitoring alerts (see `incident_response.md`)
2. **Assess**: Determine scope, impact, risk level (within 24 hours)
3. **Contain**: Stop breach, revoke credentials (within 1 hour for P0)
4. **Notify Authority** (within 72 hours):
   ```
   To: supervisory-authority@gdpr.eu
   Subject: Personal Data Breach Notification - goimg-datalayer

   1. Nature of breach: [SQL injection exposing user emails]
   2. Categories and number of data subjects: [247 users, EU residents]
   3. Categories and number of records: [Email addresses, IP addresses]
   4. Likely consequences: [Account enumeration, phishing risk]
   5. Measures taken: [Patched vulnerability, rotated credentials]
   6. Contact: privacy@goimg-datalayer.example.com
   ```

5. **Notify Users** (if high risk):
   ```
   Subject: Important Security Notice - Data Breach Notification

   We are writing to inform you of a security incident that may have affected
   your personal data. On [date], we discovered [brief description].

   What happened: [Clear explanation]
   What data was affected: [Specific data types]
   What we did: [Immediate actions]
   What you should do: [User actions - change password, etc.]

   Your rights: Under GDPR, you have the right to lodge a complaint with
   a supervisory authority.

   Contact: privacy@goimg-datalayer.example.com
   ```

6. **Document**: Maintain breach register (GDPR Article 33(5))

### Breach Register

**Location**: `/home/user/goimg-datalayer/docs/security/breach_register.md`

**Contents**:
```markdown
## Breach Register (GDPR Article 33(5))

| Date | Breach Type | Data Affected | Users Affected | Risk Level | Notification | Resolution |
|------|-------------|---------------|----------------|------------|--------------|------------|
| 2025-12-05 | SQL Injection | Emails, IPs | 247 | Medium | Authority (72h) | Patched SEC-2025-0042 |
```

---

## Data Processing Agreements (DPA)

### Third-Party Processors

When using third-party services, we maintain Data Processing Agreements:

| Service | Purpose | Data Shared | DPA Status | GDPR Compliant |
|---------|---------|-------------|------------|----------------|
| **AWS S3** | Image storage | Image files (no PII metadata) | ✅ Signed | ✅ Yes |
| **Pinata (IPFS)** | Decentralized storage | Image CIDs | ✅ Signed | ✅ Yes |
| **Backblaze B2** | Backup storage | Encrypted backups | ✅ Signed | ✅ Yes |
| **Sentry** (if used) | Error tracking | Stack traces (may include IPs) | ⚠️ Configure IP anonymization | ✅ Yes |

**International Transfers**: AWS, Pinata, Backblaze use EU Standard Contractual Clauses (SCCs) for GDPR compliance.

---

## User Transparency

### Privacy Policy

**Location**: Public-facing website `/privacy-policy`

**Contents** (summary):
1. What data we collect and why
2. How we use data
3. How long we retain data
4. User rights (access, delete, rectify, etc.)
5. How to exercise rights
6. Contact information

**Updates**: Notify users 30 days before material changes take effect.

### Cookie Policy

**Cookies Used**:

| Cookie | Purpose | Retention | Required |
|--------|---------|-----------|----------|
| `session_id` | Authentication | 7 days | Yes (functional) |
| `csrf_token` | Security | Session | Yes (functional) |
| `preferences` | UI preferences | 1 year | No (optional) |

**Note**: We do not use tracking or advertising cookies. No consent banner required (functional cookies only).

---

## Compliance Checklist

### GDPR Compliance

```markdown
- [x] Legal basis for processing identified (legitimate interest, consent)
- [x] Privacy policy published and accessible
- [x] Data retention periods defined
- [x] User rights implemented (access, delete, rectify, portability)
- [x] Breach notification procedures documented
- [x] Data Processing Agreements with third parties
- [x] Security measures in place (encryption, access controls)
- [ ] Data Protection Impact Assessment (DPIA) - Conduct if high-risk processing
- [x] Data Protection Officer contact published
- [x] Audit logging for compliance
```

### CCPA Compliance

```markdown
- [x] Privacy policy includes CCPA disclosures
- [x] "Do Not Sell My Personal Information" link (not applicable - we don't sell)
- [x] Right to know procedure implemented (data export)
- [x] Right to delete procedure implemented
- [x] Non-discrimination policy (no penalties for exercising rights)
- [ ] Consumer request verification process documented
- [x] Privacy contact published
```

---

## Automated Retention Management

### Daily Cleanup Job

**Script**: `/home/user/goimg-datalayer/scripts/data_retention_cleanup.sh`

```bash
#!/bin/bash
# Daily data retention cleanup job
# Runs at 2:00 AM UTC via cron

set -euo pipefail

DB_HOST="${DB_HOST:-localhost}"
DB_USER="${DB_USER:-goimg}"
DB_NAME="${DB_NAME:-goimg}"

echo "=== Data Retention Cleanup $(date -u) ==="

# 1. Hard delete soft-deleted users (30-day grace period expired)
echo "Deleting users past grace period..."
psql -h $DB_HOST -U $DB_USER -d $DB_NAME <<SQL
DELETE FROM users
WHERE status = 'deleted'
AND deleted_at < NOW() - INTERVAL '30 days';
SQL

# 2. Archive old audit logs to S3
echo "Archiving audit logs older than 90 days..."
psql -h $DB_HOST -U $DB_USER -d $DB_NAME <<SQL
COPY (
  SELECT * FROM audit_log
  WHERE timestamp < NOW() - INTERVAL '90 days'
) TO STDOUT (FORMAT CSV, HEADER)
SQL | gzip | aws s3 cp - s3://goimg-audit-archive/audit-logs-$(date +%Y%m).csv.gz

# 3. Delete archived audit logs from hot storage
psql -h $DB_HOST -U $DB_USER -d $DB_NAME <<SQL
DELETE FROM audit_log
WHERE timestamp < NOW() - INTERVAL '90 days';
SQL

# 4. Anonymize IP addresses in old logs
echo "Anonymizing IP addresses in logs older than 90 days..."
psql -h $DB_HOST -U $DB_USER -d $DB_NAME <<SQL
UPDATE audit_log
SET client_ip = host(network(client_ip) || '/24')::inet
WHERE timestamp < NOW() - INTERVAL '90 days'
AND client_ip IS NOT NULL;
SQL

# 5. Delete expired session tokens
echo "Deleting expired sessions..."
redis-cli SCAN 0 MATCH "goimg:session:*" | while read key; do
  TTL=$(redis-cli TTL "$key")
  if [ "$TTL" -eq "-1" ]; then
    redis-cli DEL "$key"
  fi
done

# 6. Purge old backups
echo "Deleting backups older than 30 days..."
aws s3 ls s3://goimg-backups/daily/ | awk '{print $4}' | while read backup; do
  BACKUP_DATE=$(echo $backup | grep -oP '\d{8}')
  DAYS_OLD=$(( ($(date +%s) - $(date -d $BACKUP_DATE +%s)) / 86400 ))
  if [ $DAYS_OLD -gt 30 ]; then
    aws s3 rm s3://goimg-backups/daily/$backup
  fi
done

echo "=== Cleanup Complete ==="
```

**Cron Schedule**:
```cron
# /etc/cron.d/goimg-data-retention
0 2 * * * /home/user/goimg-datalayer/scripts/data_retention_cleanup.sh >> /var/log/goimg-retention.log 2>&1
```

---

## User Deletion Request Handling

### Self-Service Deletion (UI)

**Endpoint**: `DELETE /api/v1/users/me`

**Requirements**:
1. User must be authenticated
2. Password re-entry required for confirmation
3. Email confirmation link sent

**Implementation**:
```go
// internal/application/identity/commands/delete_user.go

func (h *DeleteUserHandler) Handle(ctx context.Context, cmd DeleteUserCommand) error {
    // 1. Verify password
    user, err := h.userRepo.GetByID(ctx, cmd.UserID)
    if err != nil {
        return fmt.Errorf("get user: %w", err)
    }

    if !user.PasswordHash.Compare(cmd.Password) {
        return ErrInvalidPassword
    }

    // 2. Send confirmation email
    confirmToken := generateDeletionToken()
    h.emailService.SendDeletionConfirmation(user.Email, confirmToken)

    // 3. Schedule deletion (confirmed via email link)
    h.scheduler.ScheduleDeletion(user.ID, time.Now().Add(24*time.Hour))

    return nil
}
```

### Manual Deletion Request (Email)

**Process**:
1. User emails privacy@goimg-datalayer.example.com
2. Support verifies identity (ask for account email + last login date)
3. Support initiates deletion via admin panel
4. System follows same deletion procedure as self-service

**SLA**: Complete within **30 days** of verified request (GDPR requirement)

---

## Document Control

**Version History**:
- v1.0 (2025-12-05): Initial creation for Sprint 9

**Related Documents**:
- `/home/user/goimg-datalayer/SECURITY.md` - Vulnerability disclosure policy
- `/home/user/goimg-datalayer/docs/security/incident_response.md` - Incident response procedures
- `/home/user/goimg-datalayer/docs/security/monitoring.md` - Security monitoring (includes log retention)
- `/home/user/goimg-datalayer/claude/security_gates.md` - Security gate S9-COMP-001

**Approval**:
- Legal/Compliance Officer: [Pending]
- Data Protection Officer: [Pending]
- Security Operations Lead: [Pending]
- Engineering Director: [Pending]

**Next Review**: 2025-12-05 + 12 months = 2026-12-05

---

**Last Updated**: 2025-12-05 (Sprint 9)
