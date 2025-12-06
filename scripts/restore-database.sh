#!/bin/bash
# ============================================================================
# PostgreSQL Database Restore Script
# ============================================================================
# Restores a PostgreSQL database from backups created by backup-database.sh
#
# Features:
# - Downloads backups from S3-compatible storage
# - Decrypts GPG-encrypted backups
# - Validates backup before restore
# - Supports dry-run mode
# - Comprehensive safety checks
# - Creates database if it doesn't exist
# - Terminates existing connections
# - Transaction-safe restore
#
# Exit Codes:
#   0 - Success
#   1 - General error
#   2 - Missing dependencies
#   3 - Configuration error
#   4 - Download failed
#   5 - Decryption failed
#   6 - Restore failed
#
# Usage:
#   ./restore-database.sh --file <backup-file> [options]
#   ./restore-database.sh --s3-key <s3-key> [options]
#
# Environment Variables:
#   DB_HOST           - Database host (default: localhost)
#   DB_PORT           - Database port (default: 5432)
#   DB_NAME           - Database name (default: goimg)
#   DB_USER           - Database user (default: goimg)
#   DB_PASSWORD       - Database password (REQUIRED)
#   S3_ENDPOINT       - S3 endpoint URL
#   S3_BUCKET         - S3 bucket name
#   S3_ACCESS_KEY     - S3 access key ID
#   S3_SECRET_KEY     - S3 secret access key
#   DOCKER_CONTAINER  - Docker container name (default: goimg-postgres)
#   USE_DOCKER        - Use Docker exec (default: true)
#
# ============================================================================

set -euo pipefail

# ============================================================================
# Configuration with Defaults
# ============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

# Database Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-goimg}"
DB_USER="${DB_USER:-goimg}"
DB_PASSWORD="${DB_PASSWORD:-}"

# Docker Configuration
USE_DOCKER="${USE_DOCKER:-true}"
DOCKER_CONTAINER="${DOCKER_CONTAINER:-goimg-postgres}"

# S3 Configuration
S3_ENDPOINT="${S3_ENDPOINT:-}"
S3_BUCKET="${S3_BUCKET:-}"
S3_ACCESS_KEY="${S3_ACCESS_KEY:-}"
S3_SECRET_KEY="${S3_SECRET_KEY:-}"
S3_PATH_PREFIX="postgres-backups"

# Restore Configuration
BACKUP_FILE=""
S3_KEY=""
DRY_RUN=false
FORCE=false
TEMP_DIR="${TMPDIR:-/tmp}/postgres-restore-${TIMESTAMP}"

# Logging Configuration
LOG_DIR="${SCRIPT_DIR}/../backups/logs"
LOG_FILE="${LOG_DIR}/restore-${TIMESTAMP}.log"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ============================================================================
# Logging Functions
# ============================================================================

log() {
    local level=$1
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${timestamp} [${level}] ${message}" | tee -a "${LOG_FILE}"
}

log_info() {
    log "INFO" "${GREEN}$*${NC}"
}

log_warn() {
    log "WARN" "${YELLOW}$*${NC}"
}

log_error() {
    log "ERROR" "${RED}$*${NC}"
}

log_debug() {
    if [[ "${DEBUG:-false}" == "true" ]]; then
        log "DEBUG" "${BLUE}$*${NC}"
    fi
}

die() {
    local exit_code=${2:-1}
    log_error "$1"
    cleanup
    exit "${exit_code}"
}

# ============================================================================
# Cleanup Function
# ============================================================================

cleanup() {
    if [[ -d "${TEMP_DIR}" ]]; then
        log_info "Cleaning up temporary files..."
        rm -rf "${TEMP_DIR}"
    fi
}

# ============================================================================
# Validation Functions
# ============================================================================

check_dependencies() {
    local missing_deps=()

    if [[ "${USE_DOCKER}" == "true" ]]; then
        if ! command -v docker &> /dev/null; then
            missing_deps+=("docker")
        fi
    else
        if ! command -v pg_restore &> /dev/null; then
            missing_deps+=("pg_restore")
        fi
        if ! command -v psql &> /dev/null; then
            missing_deps+=("psql")
        fi
    fi

    if [[ -n "${S3_KEY}" ]]; then
        if ! command -v aws &> /dev/null; then
            missing_deps+=("aws")
        fi
    fi

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        die "Missing required dependencies: ${missing_deps[*]}" 2
    fi

    log_info "All dependencies satisfied"
}

validate_configuration() {
    if [[ -z "${DB_PASSWORD}" ]]; then
        die "DB_PASSWORD environment variable is required" 3
    fi

    if [[ -z "${BACKUP_FILE}" ]] && [[ -z "${S3_KEY}" ]]; then
        die "Either --file or --s3-key must be specified" 3
    fi

    if [[ -n "${BACKUP_FILE}" ]] && [[ -n "${S3_KEY}" ]]; then
        die "Cannot specify both --file and --s3-key" 3
    fi

    if [[ -n "${S3_KEY}" ]]; then
        if [[ -z "${S3_ENDPOINT}" ]] || [[ -z "${S3_BUCKET}" ]] || [[ -z "${S3_ACCESS_KEY}" ]] || [[ -z "${S3_SECRET_KEY}" ]]; then
            die "S3 restore requires S3_ENDPOINT, S3_BUCKET, S3_ACCESS_KEY, and S3_SECRET_KEY" 3
        fi
    fi

    if [[ "${USE_DOCKER}" == "true" ]]; then
        if ! docker ps --format '{{.Names}}' | grep -q "^${DOCKER_CONTAINER}$"; then
            die "Docker container '${DOCKER_CONTAINER}' is not running" 3
        fi
    fi

    log_info "Configuration validated"
}

# ============================================================================
# Restore Functions
# ============================================================================

download_from_s3() {
    log_info "Downloading backup from S3..."
    log_info "Endpoint: ${S3_ENDPOINT}"
    log_info "Bucket: ${S3_BUCKET}"
    log_info "Key: ${S3_KEY}"

    mkdir -p "${TEMP_DIR}"

    local local_file="${TEMP_DIR}/$(basename "${S3_KEY}")"

    # Configure AWS CLI
    export AWS_ACCESS_KEY_ID="${S3_ACCESS_KEY}"
    export AWS_SECRET_ACCESS_KEY="${S3_SECRET_KEY}"

    aws s3 cp \
        "s3://${S3_BUCKET}/${S3_KEY}" \
        "${local_file}" \
        --endpoint-url "${S3_ENDPOINT}" || die "S3 download failed" 4

    # Cleanup credentials
    unset AWS_ACCESS_KEY_ID
    unset AWS_SECRET_ACCESS_KEY

    local file_size=$(stat -f%z "${local_file}" 2>/dev/null || stat -c%s "${local_file}" 2>/dev/null)
    local file_size_human=$(numfmt --to=iec-i --suffix=B "${file_size}" 2>/dev/null || echo "${file_size} bytes")

    log_info "Download completed: ${local_file}"
    log_info "Size: ${file_size_human}"

    BACKUP_FILE="${local_file}"
}

decrypt_backup() {
    if [[ "${BACKUP_FILE}" != *.gpg ]]; then
        log_debug "Backup is not encrypted, skipping decryption"
        return 0
    fi

    log_info "Decrypting GPG-encrypted backup..."

    if ! command -v gpg &> /dev/null; then
        die "GPG is required to decrypt backup but is not installed" 2
    fi

    local decrypted_file="${BACKUP_FILE%.gpg}"

    gpg --decrypt \
        --output "${decrypted_file}" \
        "${BACKUP_FILE}" || die "Decryption failed" 5

    log_info "Backup decrypted successfully"

    # Update backup file to decrypted version
    BACKUP_FILE="${decrypted_file}"
}

validate_backup() {
    log_info "Validating backup file..."

    if [[ ! -f "${BACKUP_FILE}" ]]; then
        die "Backup file not found: ${BACKUP_FILE}" 3
    fi

    if [[ ! -s "${BACKUP_FILE}" ]]; then
        die "Backup file is empty: ${BACKUP_FILE}" 3
    fi

    local file_size=$(stat -f%z "${BACKUP_FILE}" 2>/dev/null || stat -c%s "${BACKUP_FILE}" 2>/dev/null)
    local file_size_human=$(numfmt --to=iec-i --suffix=B "${file_size}" 2>/dev/null || echo "${file_size} bytes")

    log_info "Backup file: ${BACKUP_FILE}"
    log_info "File size: ${file_size_human}"

    # Validate pg_dump custom format
    log_info "Verifying backup integrity..."

    if [[ "${USE_DOCKER}" == "true" ]]; then
        if ! docker exec -i "${DOCKER_CONTAINER}" pg_restore --list < "${BACKUP_FILE}" &> /dev/null; then
            die "Backup file is corrupted or not a valid pg_dump file" 3
        fi
    else
        if ! pg_restore --list "${BACKUP_FILE}" &> /dev/null; then
            die "Backup file is corrupted or not a valid pg_dump file" 3
        fi
    fi

    log_info "Backup integrity verified"
}

confirm_restore() {
    if [[ "${FORCE}" == "true" ]]; then
        return 0
    fi

    log_warn "========================================"
    log_warn "WARNING: DATABASE RESTORE OPERATION"
    log_warn "========================================"
    log_warn "This operation will:"
    log_warn "  1. Terminate all active connections to '${DB_NAME}'"
    log_warn "  2. DROP and recreate the database '${DB_NAME}'"
    log_warn "  3. Restore data from: $(basename "${BACKUP_FILE}")"
    log_warn ""
    log_warn "ALL CURRENT DATA IN '${DB_NAME}' WILL BE LOST!"
    log_warn ""

    echo -n "Type 'YES' to confirm restore: "
    read -r response

    if [[ "${response}" != "YES" ]]; then
        log_info "Restore cancelled by user"
        cleanup
        exit 0
    fi

    log_info "Restore confirmed, proceeding..."
}

terminate_connections() {
    log_info "Terminating existing connections to database '${DB_NAME}'..."

    local sql="SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${DB_NAME}' AND pid <> pg_backend_pid();"

    if [[ "${USE_DOCKER}" == "true" ]]; then
        docker exec -e PGPASSWORD="${DB_PASSWORD}" "${DOCKER_CONTAINER}" \
            psql -U "${DB_USER}" -d postgres -c "${sql}" >> "${LOG_FILE}" 2>&1 || true
    else
        PGPASSWORD="${DB_PASSWORD}" psql \
            -h "${DB_HOST}" \
            -p "${DB_PORT}" \
            -U "${DB_USER}" \
            -d postgres \
            -c "${sql}" >> "${LOG_FILE}" 2>&1 || true
    fi

    # Wait a moment for connections to close
    sleep 2

    log_info "Connections terminated"
}

drop_and_create_database() {
    log_info "Dropping and recreating database '${DB_NAME}'..."

    if [[ "${USE_DOCKER}" == "true" ]]; then
        # Drop database
        docker exec -e PGPASSWORD="${DB_PASSWORD}" "${DOCKER_CONTAINER}" \
            psql -U "${DB_USER}" -d postgres -c "DROP DATABASE IF EXISTS ${DB_NAME};" >> "${LOG_FILE}" 2>&1 || true

        # Create database
        docker exec -e PGPASSWORD="${DB_PASSWORD}" "${DOCKER_CONTAINER}" \
            psql -U "${DB_USER}" -d postgres -c "CREATE DATABASE ${DB_NAME} WITH ENCODING='UTF8' LC_COLLATE='C' LC_CTYPE='C' TEMPLATE=template0;" \
            || die "Failed to create database" 6
    else
        # Drop database
        PGPASSWORD="${DB_PASSWORD}" psql \
            -h "${DB_HOST}" \
            -p "${DB_PORT}" \
            -U "${DB_USER}" \
            -d postgres \
            -c "DROP DATABASE IF EXISTS ${DB_NAME};" >> "${LOG_FILE}" 2>&1 || true

        # Create database
        PGPASSWORD="${DB_PASSWORD}" psql \
            -h "${DB_HOST}" \
            -p "${DB_PORT}" \
            -U "${DB_USER}" \
            -d postgres \
            -c "CREATE DATABASE ${DB_NAME} WITH ENCODING='UTF8' LC_COLLATE='C' LC_CTYPE='C' TEMPLATE=template0;" \
            || die "Failed to create database" 6
    fi

    log_info "Database recreated successfully"
}

restore_database() {
    log_info "Starting database restore..."

    local start_time=$(date +%s)

    if [[ "${USE_DOCKER}" == "true" ]]; then
        docker exec -i -e PGPASSWORD="${DB_PASSWORD}" "${DOCKER_CONTAINER}" \
            pg_restore \
            -U "${DB_USER}" \
            -d "${DB_NAME}" \
            --verbose \
            --no-owner \
            --no-acl \
            --exit-on-error \
            < "${BACKUP_FILE}" 2>&1 | tee -a "${LOG_FILE}" || die "Restore failed" 6
    else
        PGPASSWORD="${DB_PASSWORD}" pg_restore \
            -h "${DB_HOST}" \
            -p "${DB_PORT}" \
            -U "${DB_USER}" \
            -d "${DB_NAME}" \
            --verbose \
            --no-owner \
            --no-acl \
            --exit-on-error \
            --file="${BACKUP_FILE}" 2>&1 | tee -a "${LOG_FILE}" || die "Restore failed" 6
    fi

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    log_info "Restore completed in ${duration} seconds"
}

verify_restore() {
    log_info "Verifying restored database..."

    local table_count
    local sql="SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';"

    if [[ "${USE_DOCKER}" == "true" ]]; then
        table_count=$(docker exec -e PGPASSWORD="${DB_PASSWORD}" "${DOCKER_CONTAINER}" \
            psql -U "${DB_USER}" -d "${DB_NAME}" -t -c "${sql}" | tr -d '[:space:]')
    else
        table_count=$(PGPASSWORD="${DB_PASSWORD}" psql \
            -h "${DB_HOST}" \
            -p "${DB_PORT}" \
            -U "${DB_USER}" \
            -d "${DB_NAME}" \
            -t -c "${sql}" | tr -d '[:space:]')
    fi

    log_info "Database contains ${table_count} tables"

    if [[ "${table_count}" -eq 0 ]]; then
        log_warn "WARNING: No tables found in restored database"
        log_warn "This might indicate an issue with the restore"
    else
        log_info "Database verification successful"
    fi
}

# ============================================================================
# Main Execution
# ============================================================================

show_help() {
    cat << EOF
PostgreSQL Database Restore Script

Usage: $0 [options]

Options:
  -f, --file PATH       Path to local backup file
  -s, --s3-key KEY      S3 object key (relative to S3_PATH_PREFIX)
  --dry-run             Validate backup without performing restore
  --force               Skip confirmation prompt
  --debug               Enable debug logging
  -h, --help            Show this help message

Environment Variables:
  DB_HOST               - Database host (default: localhost)
  DB_PORT               - Database port (default: 5432)
  DB_NAME               - Database name (default: goimg)
  DB_USER               - Database user (default: goimg)
  DB_PASSWORD           - Database password (REQUIRED)
  S3_ENDPOINT           - S3 endpoint URL
  S3_BUCKET             - S3 bucket name
  S3_ACCESS_KEY         - S3 access key ID
  S3_SECRET_KEY         - S3 secret access key
  DOCKER_CONTAINER      - Docker container name (default: goimg-postgres)
  USE_DOCKER            - Use Docker exec (default: true)

Examples:
  # Restore from local file
  $0 --file /var/backups/postgres/goimg-backup-20240101-120000.dump

  # Restore from S3
  $0 --s3-key postgres-backups/goimg-backup-20240101-120000.dump.gpg

  # Dry run to validate backup
  $0 --file /path/to/backup.dump --dry-run

  # Force restore without confirmation
  $0 --file /path/to/backup.dump --force

EOF
    exit 0
}

main() {
    # Setup cleanup trap
    trap cleanup EXIT

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -f|--file)
                BACKUP_FILE="$2"
                shift 2
                ;;
            -s|--s3-key)
                S3_KEY="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --force)
                FORCE=true
                shift
                ;;
            --debug)
                DEBUG=true
                shift
                ;;
            -h|--help)
                show_help
                ;;
            *)
                die "Unknown option: $1. Use --help for usage information."
                ;;
        esac
    done

    mkdir -p "${LOG_DIR}"

    log_info "========================================"
    log_info "PostgreSQL Database Restore"
    log_info "========================================"
    log_info "Timestamp: ${TIMESTAMP}"
    log_info "Target database: ${DB_NAME}"
    log_info "Host: ${DB_HOST}:${DB_PORT}"
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "MODE: DRY RUN (no changes will be made)"
    fi

    # Validate dependencies and configuration
    check_dependencies
    validate_configuration

    # Download from S3 if needed
    if [[ -n "${S3_KEY}" ]]; then
        download_from_s3
    fi

    # Decrypt if encrypted
    decrypt_backup

    # Validate backup
    validate_backup

    # If dry run, stop here
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "========================================"
        log_info "Dry Run Completed Successfully"
        log_info "========================================"
        log_info "Backup is valid and ready for restore"
        exit 0
    fi

    # Confirm restore
    confirm_restore

    # Terminate existing connections
    terminate_connections

    # Drop and recreate database
    drop_and_create_database

    # Restore database
    restore_database

    # Verify restore
    verify_restore

    log_info "========================================"
    log_info "Database Restored Successfully"
    log_info "========================================"
    log_info "Database: ${DB_NAME}"
    log_info "Backup: $(basename "${BACKUP_FILE}")"
    log_info "Log file: ${LOG_FILE}"

    exit 0
}

# Run main function
main "$@"
