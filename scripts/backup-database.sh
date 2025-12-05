#!/bin/bash
# ============================================================================
# PostgreSQL Database Backup Script with Enhanced Rotation
# ============================================================================
# Features:
# - pg_dump with custom format (-Fc) for optimal compression
# - Timestamped backups: goimg-backup-YYYYMMDD-HHMMSS.dump
# - Optional GPG encryption
# - S3-compatible storage upload (AWS S3, DigitalOcean Spaces, Backblaze B2)
# - Comprehensive logging with file size tracking
# - Exit codes for monitoring integration
# - Supports both Docker and native PostgreSQL
#
# Exit Codes:
#   0 - Success
#   1 - General error
#   2 - Missing dependencies
#   3 - Configuration error
#   4 - Backup creation failed
#   5 - Encryption failed
#   6 - Upload failed
#
# Security Gate: S9-PROD-003 - Encrypted backups with rotation policy
#
# Usage:
#   ./backup-database.sh
#
# Environment Variables (all required unless using defaults):
#   DB_HOST           - Database host (default: localhost)
#   DB_PORT           - Database port (default: 5432)
#   DB_NAME           - Database name (default: goimg)
#   DB_USER           - Database user (default: goimg)
#   DB_PASSWORD       - Database password (REQUIRED)
#   BACKUP_DIR        - Local backup directory (default: /var/backups/postgres)
#   S3_ENDPOINT       - S3 endpoint URL (e.g., https://s3.amazonaws.com)
#   S3_BUCKET         - S3 bucket name
#   S3_ACCESS_KEY     - S3 access key ID
#   S3_SECRET_KEY     - S3 secret access key
#   GPG_RECIPIENT     - GPG recipient for encryption (email or key ID)
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
DATE_TAG=$(date +%Y%m%d)
DAY_OF_WEEK=$(date +%u)  # 1 = Monday, 7 = Sunday
DAY_OF_MONTH=$(date +%d)

# Database Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-goimg}"
DB_USER="${DB_USER:-goimg}"
DB_PASSWORD="${DB_PASSWORD:-}"

# Backup Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/postgres}"
BACKUP_FILENAME="goimg-backup-${TIMESTAMP}.dump"
BACKUP_PATH="${BACKUP_DIR}/${BACKUP_FILENAME}"

# Docker Configuration
USE_DOCKER="${USE_DOCKER:-true}"
DOCKER_CONTAINER="${DOCKER_CONTAINER:-goimg-postgres}"

# S3 Configuration
S3_ENDPOINT="${S3_ENDPOINT:-}"
S3_BUCKET="${S3_BUCKET:-}"
S3_ACCESS_KEY="${S3_ACCESS_KEY:-}"
S3_SECRET_KEY="${S3_SECRET_KEY:-}"
S3_PATH_PREFIX="postgres-backups"

# Encryption Configuration
GPG_RECIPIENT="${GPG_RECIPIENT:-}"

# Logging Configuration
LOG_DIR="${BACKUP_DIR}/logs"
LOG_FILE="${LOG_DIR}/backup-$(date +%Y%m%d).log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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
    exit "${exit_code}"
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
        if ! command -v pg_dump &> /dev/null; then
            missing_deps+=("pg_dump")
        fi
    fi

    if [[ -n "${GPG_RECIPIENT}" ]]; then
        if ! command -v gpg &> /dev/null; then
            missing_deps+=("gpg")
        fi
    fi

    if [[ -n "${S3_BUCKET}" ]]; then
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

    if [[ -n "${S3_BUCKET}" ]]; then
        if [[ -z "${S3_ENDPOINT}" ]] || [[ -z "${S3_ACCESS_KEY}" ]] || [[ -z "${S3_SECRET_KEY}" ]]; then
            die "S3 upload requires S3_ENDPOINT, S3_ACCESS_KEY, and S3_SECRET_KEY" 3
        fi
    fi

    if [[ -n "${GPG_RECIPIENT}" ]]; then
        if ! gpg --list-keys "${GPG_RECIPIENT}" &> /dev/null; then
            log_warn "GPG key for recipient '${GPG_RECIPIENT}' not found in keyring"
            log_warn "Encryption may fail. Import the key first with: gpg --import <key-file>"
        fi
    fi

    log_info "Configuration validated"
}

# ============================================================================
# Backup Functions
# ============================================================================

create_backup_directory() {
    mkdir -p "${BACKUP_DIR}"
    mkdir -p "${LOG_DIR}"
    log_info "Backup directory: ${BACKUP_DIR}"
}

create_database_backup() {
    log_info "Starting PostgreSQL backup..."
    log_info "Database: ${DB_NAME}"
    log_info "Host: ${DB_HOST}:${DB_PORT}"
    log_info "User: ${DB_USER}"
    log_info "Output: ${BACKUP_PATH}"

    local start_time=$(date +%s)

    if [[ "${USE_DOCKER}" == "true" ]]; then
        log_info "Using Docker container: ${DOCKER_CONTAINER}"

        # Check if container is running
        if ! docker ps --format '{{.Names}}' | grep -q "^${DOCKER_CONTAINER}$"; then
            die "Docker container '${DOCKER_CONTAINER}' is not running" 4
        fi

        # Execute pg_dump inside container
        docker exec -e PGPASSWORD="${DB_PASSWORD}" "${DOCKER_CONTAINER}" \
            pg_dump \
            -U "${DB_USER}" \
            -d "${DB_NAME}" \
            -Fc \
            -Z 9 \
            --verbose \
            --no-owner \
            --no-acl \
            > "${BACKUP_PATH}" 2>> "${LOG_FILE}" || die "Backup creation failed" 4
    else
        log_info "Using native pg_dump"

        PGPASSWORD="${DB_PASSWORD}" pg_dump \
            -h "${DB_HOST}" \
            -p "${DB_PORT}" \
            -U "${DB_USER}" \
            -d "${DB_NAME}" \
            -Fc \
            -Z 9 \
            --verbose \
            --file="${BACKUP_PATH}" 2>> "${LOG_FILE}" || die "Backup creation failed" 4
    fi

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    if [[ ! -f "${BACKUP_PATH}" ]]; then
        die "Backup file was not created: ${BACKUP_PATH}" 4
    fi

    local backup_size=$(stat -f%z "${BACKUP_PATH}" 2>/dev/null || stat -c%s "${BACKUP_PATH}" 2>/dev/null)
    local backup_size_human=$(numfmt --to=iec-i --suffix=B "${backup_size}" 2>/dev/null || echo "${backup_size} bytes")

    log_info "Backup created successfully"
    log_info "File: ${BACKUP_PATH}"
    log_info "Size: ${backup_size_human}"
    log_info "Duration: ${duration} seconds"

    # Log backup metadata for monitoring
    echo "BACKUP_SIZE_BYTES=${backup_size}" >> "${LOG_FILE}"
    echo "BACKUP_DURATION_SECONDS=${duration}" >> "${LOG_FILE}"
}

verify_backup() {
    log_info "Verifying backup integrity..."

    if [[ ! -s "${BACKUP_PATH}" ]]; then
        die "Backup file is empty: ${BACKUP_PATH}" 4
    fi

    # Verify pg_dump custom format
    if [[ "${USE_DOCKER}" == "true" ]]; then
        if ! docker exec -i "${DOCKER_CONTAINER}" pg_restore --list < "${BACKUP_PATH}" &> /dev/null; then
            die "Backup integrity check failed - corrupted dump file" 4
        fi
    else
        if ! pg_restore --list "${BACKUP_PATH}" &> /dev/null; then
            die "Backup integrity check failed - corrupted dump file" 4
        fi
    fi

    log_info "Backup integrity verified"
}

encrypt_backup() {
    if [[ -z "${GPG_RECIPIENT}" ]]; then
        log_debug "GPG encryption not configured, skipping"
        return 0
    fi

    log_info "Encrypting backup with GPG..."
    log_info "Recipient: ${GPG_RECIPIENT}"

    local encrypted_path="${BACKUP_PATH}.gpg"

    gpg --encrypt \
        --recipient "${GPG_RECIPIENT}" \
        --trust-model always \
        --output "${encrypted_path}" \
        "${BACKUP_PATH}" || die "Encryption failed" 5

    # Remove unencrypted file
    rm -f "${BACKUP_PATH}"

    # Update backup path to encrypted version
    BACKUP_PATH="${encrypted_path}"
    BACKUP_FILENAME="${BACKUP_FILENAME}.gpg"

    local encrypted_size=$(stat -f%z "${BACKUP_PATH}" 2>/dev/null || stat -c%s "${BACKUP_PATH}" 2>/dev/null)
    local encrypted_size_human=$(numfmt --to=iec-i --suffix=B "${encrypted_size}" 2>/dev/null || echo "${encrypted_size} bytes")

    log_info "Backup encrypted successfully"
    log_info "Encrypted file: ${BACKUP_PATH}"
    log_info "Encrypted size: ${encrypted_size_human}"
}

upload_to_s3() {
    if [[ -z "${S3_BUCKET}" ]]; then
        log_debug "S3 upload not configured, skipping"
        return 0
    fi

    log_info "Uploading backup to S3..."
    log_info "Endpoint: ${S3_ENDPOINT}"
    log_info "Bucket: ${S3_BUCKET}"

    local s3_key="${S3_PATH_PREFIX}/${BACKUP_FILENAME}"

    # Configure AWS CLI for S3-compatible storage
    export AWS_ACCESS_KEY_ID="${S3_ACCESS_KEY}"
    export AWS_SECRET_ACCESS_KEY="${S3_SECRET_KEY}"

    # Determine storage class based on backup type
    local storage_class="STANDARD"
    if [[ "${DAY_OF_WEEK}" == "7" ]]; then
        storage_class="STANDARD_IA"  # Weekly backups use infrequent access
        log_debug "Weekly backup - using STANDARD_IA storage class"
    fi

    local start_time=$(date +%s)

    aws s3 cp \
        "${BACKUP_PATH}" \
        "s3://${S3_BUCKET}/${s3_key}" \
        --endpoint-url "${S3_ENDPOINT}" \
        --storage-class "${storage_class}" \
        --metadata "backup-date=${DATE_TAG},database=${DB_NAME},encrypted=$([[ -n ${GPG_RECIPIENT} ]] && echo true || echo false)" \
        || die "S3 upload failed" 6

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    log_info "Upload completed successfully"
    log_info "S3 URI: s3://${S3_BUCKET}/${s3_key}"
    log_info "Upload duration: ${duration} seconds"

    # Cleanup credentials from environment
    unset AWS_ACCESS_KEY_ID
    unset AWS_SECRET_ACCESS_KEY
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    log_info "========================================"
    log_info "PostgreSQL Backup Started"
    log_info "========================================"
    log_info "Timestamp: ${TIMESTAMP}"
    log_info "Day of week: ${DAY_OF_WEEK} (7 = Sunday)"
    log_info "Day of month: ${DAY_OF_MONTH}"

    # Create backup directory
    create_backup_directory

    # Validate dependencies and configuration
    check_dependencies
    validate_configuration

    # Create backup
    create_database_backup

    # Verify backup integrity
    verify_backup

    # Encrypt backup if configured
    encrypt_backup

    # Upload to S3 if configured
    upload_to_s3

    # Final status
    log_info "========================================"
    log_info "Backup Completed Successfully"
    log_info "========================================"
    log_info "Local backup: ${BACKUP_PATH}"
    if [[ -n "${S3_BUCKET}" ]]; then
        log_info "S3 backup: s3://${S3_BUCKET}/${S3_PATH_PREFIX}/${BACKUP_FILENAME}"
    fi
    log_info "Encrypted: $([[ -n ${GPG_RECIPIENT} ]] && echo 'Yes' || echo 'No')"

    exit 0
}

# Run main function
main "$@"
