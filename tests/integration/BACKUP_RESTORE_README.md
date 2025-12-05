# Backup/Restore Validation Test Suite

Comprehensive test suite for validating PostgreSQL backup and restore operations for the goimg-datalayer project.

## Overview

This test suite validates that the backup and restore process:
- Creates complete, accurate backups
- Restores data with 100% integrity
- Maintains all foreign key relationships
- Preserves triggers, functions, and constraints
- Meets Recovery Time Objective (RTO) targets
- Handles edge cases (unicode, nulls, soft deletes)

**Security Gate**: S9-PROD-004 - RTO must be < 30 minutes

## Components

### 1. Seed Data Script (`backup-restore-seed.sql`)

Comprehensive SQL script that populates a test database with realistic data:

- **100 users** - Various roles (user, moderator, admin), statuses (active, pending, suspended, deleted)
- **200 sessions** - Active and revoked sessions with diverse user agents
- **500 images** - Multiple storage providers, statuses, visibility levels
- **2000 image variants** - Thumbnails, small, medium, large variants
- **50 albums** - With cover images and descriptions
- **300 album-image associations** - Many-to-many relationships with ordering
- **100 tags** - Reusable tags for image categorization
- **800 image-tag associations** - Tag relationships
- **2000 likes** - User-image interactions
- **1000 comments** - User comments on images

**Total**: 6000+ rows across 10 tables

**Features**:
- Unicode content (emoji, Cyrillic, Chinese characters)
- NULL values in optional fields
- Empty strings for edge case testing
- Soft-deleted records (deleted_at timestamps)
- Realistic relationships across all tables

**Usage**:
```bash
# Populate test database
psql -U goimg -d goimg_backup_test -f tests/integration/backup-restore-seed.sql
```

### 2. Validation Shell Script (`scripts/validate-backup-restore.sh`)

Bash script that performs end-to-end validation of backup/restore process:

**Process Flow**:
1. Create test database
2. Run migrations
3. Populate seed data
4. Calculate pre-backup checksums and row counts
5. Create backup using `backup-database.sh`
6. Destroy database (simulate disaster)
7. Restore from backup using `restore-database.sh` (measure RTO)
8. Calculate post-restore checksums and row counts
9. Compare checksums (data integrity)
10. Verify foreign key relationships
11. Verify triggers and functions
12. Validate RTO < 30 minutes
13. Generate validation report

**Exit Codes**:
- `0` - All validations passed
- `1` - General error
- `2` - Missing dependencies
- `3` - Configuration error
- `4` - Seed data failed
- `5` - Backup failed
- `6` - Restore failed
- `7` - Validation failed (data integrity)
- `8` - RTO exceeded (>30 minutes)

**Usage**:
```bash
# Basic validation
DB_PASSWORD=secret ./scripts/validate-backup-restore.sh

# With report generation
DB_PASSWORD=secret ./scripts/validate-backup-restore.sh --output-report /tmp/backup-validation-report.md

# Keep test database for debugging
DB_PASSWORD=secret ./scripts/validate-backup-restore.sh --no-cleanup
```

**Environment Variables**:
```bash
DB_HOST=localhost          # Test database host
DB_PORT=5432               # Test database port
DB_NAME=goimg_backup_test  # Test database name (will be created)
DB_USER=goimg              # Database user
DB_PASSWORD=secret         # Database password (REQUIRED)
USE_DOCKER=true            # Use Docker exec for pg_dump/pg_restore
DOCKER_CONTAINER=goimg-postgres  # Docker container name
```

### 3. Go Integration Test (`tests/integration/backup_restore_test.go`)

Programmatic integration test using testcontainers for isolated testing:

**Test Cases**:

1. **TestBackupRestore_FullCycle** - Complete validation cycle
   - Populates seed data
   - Calculates checksums
   - Creates backup
   - Destroys database
   - Restores from backup
   - Validates data integrity
   - Measures RTO

2. **TestBackupRestore_EmptyDatabase** - Edge case: empty database
   - Validates backup/restore works with no data
   - Ensures schema is preserved

3. **TestBackupRestore_PartialData** - Edge case: partial data
   - Only some tables populated
   - Validates selective data backup/restore

4. **TestBackupRestore_ChecksumCalculation** - Validates checksum function
   - Ensures checksums are deterministic
   - Tests accuracy of integrity checking

5. **TestBackupRestore_LargeDataset** - Performance test
   - Tests with 1000 users
   - Validates performance at scale
   - Measures RTO for larger datasets

6. **BenchmarkBackupRestore** - Performance benchmark
   - Measures backup/restore performance
   - Helps identify regressions

**Usage**:
```bash
# Run all backup/restore integration tests
go test -v -tags=integration ./tests/integration -run TestBackupRestore

# Run specific test
go test -v -tags=integration ./tests/integration -run TestBackupRestore_FullCycle

# Run with race detector
go test -race -v -tags=integration ./tests/integration -run TestBackupRestore

# Run benchmark
go test -bench=BenchmarkBackupRestore -tags=integration ./tests/integration
```

### 4. Validation Report Template (`docs/operations/backup-restore-validation-template.md`)

Comprehensive report template for documenting validation results:

**Sections**:
- Executive Summary
- Test Configuration
- Test Execution Summary (timeline)
- Recovery Time Objective (RTO) analysis
- Data Integrity Verification (row counts, checksums)
- Foreign Key Relationship Verification
- Database Objects Verification (triggers, functions, constraints)
- Edge Cases Tested
- Issues and Anomalies
- Pass/Fail Criteria
- Recommendations
- Security Gate Compliance (S9-PROD-004)
- Sign-Off

**Usage**: Fill in template after running validation tests for official documentation.

## Quick Start

### Prerequisites

- PostgreSQL 16+
- Docker (if using containerized approach)
- Go 1.25+ (for integration tests)
- Bash 4.0+
- Access to goimg database user

### Running Validation

**Option 1: Shell Script (Recommended for Production)**
```bash
# Set required environment variable
export DB_PASSWORD=your_secure_password

# Run validation with report
./scripts/validate-backup-restore.sh --output-report /tmp/validation-report.md

# Check exit code
echo $?  # Should be 0 for success
```

**Option 2: Go Integration Tests**
```bash
# Run full test suite
go test -v -tags=integration ./tests/integration -run TestBackupRestore

# Run with coverage
go test -v -tags=integration -coverprofile=coverage.out ./tests/integration -run TestBackupRestore
go tool cover -html=coverage.out
```

**Option 3: Manual Validation**
```bash
# 1. Create test database
createdb -U goimg goimg_backup_test

# 2. Run migrations
goose -dir migrations postgres "user=goimg password=secret dbname=goimg_backup_test" up

# 3. Populate seed data
psql -U goimg -d goimg_backup_test -f tests/integration/backup-restore-seed.sql

# 4. Create backup
DB_NAME=goimg_backup_test ./scripts/backup-database.sh

# 5. Drop database
dropdb -U goimg goimg_backup_test

# 6. Restore from backup
./scripts/restore-database.sh --file /var/backups/postgres/goimg-backup-*.dump --force
```

## Validation Criteria

### Pass/Fail Thresholds

| Criterion | Threshold | Critical |
|-----------|-----------|----------|
| RTO | < 30 minutes | ✓ YES |
| Row Count Accuracy | 100% | ✓ YES |
| Checksum Match | 100% | ✓ YES |
| Foreign Key Integrity | 100% | ✓ YES |
| Trigger Restoration | 100% | ✓ YES |
| Function Restoration | 100% | ✓ YES |
| Constraint Enforcement | 100% | ✓ YES |

**Overall**: All criteria must pass for validation to be successful.

### Security Gate S9-PROD-004

**Requirement**: Database backup and restore process must be validated with measured RTO < 30 minutes.

**Evidence**:
- Automated test results from `validate-backup-restore.sh`
- Go integration test results
- Completed validation report using template

**Approval**: Requires sign-off from:
- Senior Backend Engineer
- DevOps Engineer
- Security Team Representative

## Troubleshooting

### Common Issues

**Issue**: `DB_PASSWORD environment variable is required`
**Solution**: Set the password before running: `export DB_PASSWORD=your_password`

**Issue**: `Docker container 'goimg-postgres' is not running`
**Solution**: Start the container: `docker-compose -f docker/docker-compose.yml up -d`

**Issue**: `Seed data SQL file not found`
**Solution**: Ensure you're running from project root or adjust path in script

**Issue**: `RTO exceeded`
**Solution**:
- Check database size (may need optimization)
- Verify no other processes consuming resources
- Consider incremental backups for very large databases
- Check network latency if using remote storage

**Issue**: `Checksum mismatch`
**Solution**:
- Investigate which table(s) failed
- Check for non-deterministic data (timestamps, random UUIDs)
- Verify triggers are functioning correctly
- Check for concurrent modifications during test

### Debug Mode

Run with debug output:
```bash
DEBUG=true ./scripts/validate-backup-restore.sh --no-cleanup
```

This will:
- Print verbose logging
- Preserve test database for inspection
- Keep temporary files for analysis

## CI/CD Integration

Add to GitHub Actions workflow:

```yaml
name: Backup/Restore Validation

on:
  schedule:
    - cron: '0 2 * * 0'  # Weekly on Sunday at 2 AM
  workflow_dispatch:

jobs:
  validate-backup-restore:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: goimg
          POSTGRES_PASSWORD: testpass
          POSTGRES_DB: goimg_backup_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4

      - name: Run backup/restore validation
        env:
          DB_PASSWORD: testpass
          USE_DOCKER: false
          DB_HOST: localhost
        run: |
          ./scripts/validate-backup-restore.sh --output-report validation-report.md

      - name: Upload validation report
        uses: actions/upload-artifact@v3
        with:
          name: validation-report
          path: validation-report.md

      - name: Fail if RTO exceeded
        run: |
          if [ $? -eq 8 ]; then
            echo "RTO target not met"
            exit 1
          fi
```

## Performance Benchmarks

Expected performance on standard hardware (8GB RAM, 4 CPU cores):

| Dataset Size | Backup Time | Restore Time | Total RTO | Status |
|--------------|-------------|--------------|-----------|--------|
| Empty DB | ~5s | ~5s | ~10s | ✓ |
| 1K rows | ~10s | ~10s | ~20s | ✓ |
| 10K rows | ~30s | ~30s | ~60s | ✓ |
| 100K rows | ~2m | ~3m | ~5m | ✓ |
| 1M rows | ~10m | ~15m | ~25m | ✓ |
| 10M rows | ~30m+ | ~45m+ | ~75m | ⚠ May exceed RTO |

**Note**: For datasets >1M rows, consider:
- Incremental backups
- Point-in-time recovery (PITR)
- Parallel restore
- Hardware optimization

## Best Practices

1. **Run Regularly**: Schedule weekly validation tests in CI/CD
2. **Document Results**: Use the validation report template for each test
3. **Monitor RTO**: Track RTO trends over time as data grows
4. **Test Recovery Scenarios**: Test various failure scenarios (corruption, partial restore)
5. **Validate Encryption**: Test both encrypted and unencrypted backups
6. **Test Remote Storage**: Validate S3/DO Spaces/B2 upload/download
7. **Update Playbooks**: Keep disaster recovery documentation current
8. **Practice Restores**: Run restore drills quarterly with operations team

## Contributing

When modifying the backup/restore process:

1. Update seed data if schema changes
2. Add new validation checks to shell script
3. Add corresponding Go integration tests
4. Update validation report template
5. Run full validation suite before committing
6. Update this README with any new requirements

## References

- Backup Script: `scripts/backup-database.sh`
- Restore Script: `scripts/restore-database.sh`
- Database Migrations: `migrations/`
- Operations Documentation: `docs/operations/`
- Security Gates: `claude/security_gates.md`
- Test Strategy: `claude/test_strategy.md`

## License

Part of the goimg-datalayer project. See project LICENSE file.
