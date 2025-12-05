#!/bin/bash
# ============================================================================
# PostgreSQL Database Backup Script
# ============================================================================
# Features:
# - Compressed backups with gzip
# - Timestamp-based naming
# - Retention policy (configurable)
# - Optional GPG encryption
# - Local and S3 storage support
# - Logging and error handling
# - Docker and native PostgreSQL support
#
# Usage:
#   ./backup-db.sh [options]
#
# Options:
#   -h, --help              Show this help message
#   -c, --container NAME    Docker container name (default: goimg-postgres)
#   -d, --database NAME     Database name (required)
#   -u, --user NAME         Database user (required)
#   -p, --password PASS     Database password (or use PGPASSWORD env var)
#   -o, --output DIR        Output directory (default: ./backups)
#   -r, --retention DAYS    Keep backups for N days (default: 30)
#   -e, --encrypt EMAIL     GPG recipient email for encryption
#   -s, --s3-bucket BUCKET  Upload to S3 bucket
#   --s3-region REGION      S3 region (default: us-east-1)
#   --native                Use native pg_dump (not Docker)

set -euo pipefail

# ============================================================================
# Configuration
# ============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTAINER_NAME="${BACKUP_CONTAINER:-goimg-postgres}"
DATABASE="${BACKUP_DATABASE:-}"
DB_USER="${BACKUP_USER:-}"
DB_PASSWORD="${PGPASSWORD:-}"
OUTPUT_DIR="${BACKUP_DIR:-${SCRIPT_DIR}/../backups}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
GPG_RECIPIENT="${BACKUP_GPG_RECIPIENT:-}"
S3_BUCKET="${BACKUP_S3_BUCKET:-}"
S3_REGION="${BACKUP_S3_REGION:-us-east-1}"
USE_NATIVE="${BACKUP_USE_NATIVE:-false}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="${OUTPUT_DIR}/backup.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ============================================================================
# Functions
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

die() {
    log_error "$*"
    exit 1
}

show_help() {
    cat << EOF
PostgreSQL Database Backup Script

Usage: $0 [options]

Options:
  -h, --help              Show this help message
  -c, --container NAME    Docker container name (default: goimg-postgres)
  -d, --database NAME     Database name (required)
  -u, --user NAME         Database user (required)
  -p, --password PASS     Database password (or use PGPASSWORD env var)
  -o, --output DIR        Output directory (default: ./backups)
  -r, --retention DAYS    Keep backups for N days (default: 30)
  -e, --encrypt EMAIL     GPG recipient email for encryption
  -s, --s3-bucket BUCKET  Upload to S3 bucket
  --s3-region REGION      S3 region (default: us-east-1)
  --native                Use native pg_dump (not Docker)

Environment Variables:
  PGPASSWORD              Database password
  BACKUP_DATABASE         Database name
  BACKUP_USER             Database user
  BACKUP_DIR              Output directory
  BACKUP_RETENTION_DAYS   Retention period in days
  BACKUP_GPG_RECIPIENT    GPG encryption recipient
  BACKUP_S3_BUCKET        S3 bucket name
  BACKUP_S3_REGION        S3 region
  BACKUP_USE_NATIVE       Use native pg_dump (true/false)

Examples:
  # Basic backup
  $0 -d goimg -u postgres -p password123

  # Backup with encryption
  $0 -d goimg -u postgres -e backup@example.com

  # Backup to S3
  $0 -d goimg -u postgres -s my-backup-bucket

  # Use environment variables
  export PGPASSWORD=password123
  export BACKUP_DATABASE=goimg
  export BACKUP_USER=postgres
  $0

EOF
    exit 0
}

check_dependencies() {
    local deps=("gzip")

    if [[ "${USE_NATIVE}" == "true" ]]; then
        deps+=("pg_dump" "psql")
    else
        deps+=("docker")
    fi

    if [[ -n "${GPG_RECIPIENT}" ]]; then
        deps+=("gpg")
    fi

    if [[ -n "${S3_BUCKET}" ]]; then
        deps+=("aws")
    fi

    for cmd in "${deps[@]}"; do
        if ! command -v "${cmd}" &> /dev/null; then
            die "Required command '${cmd}' not found. Please install it first."
        fi
    done
}

create_backup() {
    local backup_file="${OUTPUT_DIR}/${DATABASE}_${TIMESTAMP}.sql.gz"

    log_info "Starting backup of database '${DATABASE}'..."

    if [[ "${USE_NATIVE}" == "true" ]]; then
        # Native pg_dump
        log_info "Using native pg_dump..."
        PGPASSWORD="${DB_PASSWORD}" pg_dump \
            -h localhost \
            -U "${DB_USER}" \
            -d "${DATABASE}" \
            --format=custom \
            --compress=9 \
            --verbose \
            --file="${backup_file%.gz}" 2>&1 | tee -a "${LOG_FILE}"

        # Compress
        gzip -9 "${backup_file%.gz}"
    else
        # Docker-based backup
        log_info "Using Docker container '${CONTAINER_NAME}'..."
        docker exec -e PGPASSWORD="${DB_PASSWORD}" "${CONTAINER_NAME}" \
            pg_dump \
            -U "${DB_USER}" \
            -d "${DATABASE}" \
            --format=plain \
            --verbose \
            --no-owner \
            --no-acl 2>&1 | gzip -9 > "${backup_file}" || die "Backup failed"
    fi

    if [[ ! -f "${backup_file}" ]]; then
        die "Backup file was not created: ${backup_file}"
    fi

    local backup_size=$(du -h "${backup_file}" | cut -f1)
    log_info "Backup created successfully: ${backup_file} (${backup_size})"

    echo "${backup_file}"
}

encrypt_backup() {
    local backup_file=$1
    local encrypted_file="${backup_file}.gpg"

    log_info "Encrypting backup with GPG for recipient '${GPG_RECIPIENT}'..."

    gpg --encrypt \
        --recipient "${GPG_RECIPIENT}" \
        --trust-model always \
        --output "${encrypted_file}" \
        "${backup_file}" || die "Encryption failed"

    # Remove unencrypted file
    rm -f "${backup_file}"

    log_info "Backup encrypted: ${encrypted_file}"
    echo "${encrypted_file}"
}

upload_to_s3() {
    local backup_file=$1
    local s3_path="s3://${S3_BUCKET}/postgres-backups/$(basename "${backup_file}")"

    log_info "Uploading backup to S3: ${s3_path}..."

    aws s3 cp \
        "${backup_file}" \
        "${s3_path}" \
        --region "${S3_REGION}" \
        --storage-class STANDARD_IA || die "S3 upload failed"

    log_info "Backup uploaded to S3 successfully"
}

cleanup_old_backups() {
    log_info "Cleaning up backups older than ${RETENTION_DAYS} days..."

    local count=0
    while IFS= read -r -d '' file; do
        log_info "Removing old backup: ${file}"
        rm -f "${file}"
        ((count++))
    done < <(find "${OUTPUT_DIR}" -name "${DATABASE}_*.sql.gz*" -type f -mtime +"${RETENTION_DAYS}" -print0)

    if [[ ${count} -gt 0 ]]; then
        log_info "Removed ${count} old backup(s)"
    else
        log_info "No old backups to remove"
    fi

    # Cleanup old S3 backups if configured
    if [[ -n "${S3_BUCKET}" ]]; then
        log_info "Cleaning up old S3 backups..."

        local cutoff_date=$(date -d "${RETENTION_DAYS} days ago" +%Y%m%d)

        aws s3 ls "s3://${S3_BUCKET}/postgres-backups/" --region "${S3_REGION}" | \
        awk '{print $4}' | \
        while read -r file; do
            # Extract date from filename (assuming YYYYMMDD format)
            if [[ ${file} =~ ${DATABASE}_([0-9]{8}) ]]; then
                local file_date=${BASH_REMATCH[1]}
                if [[ ${file_date} -lt ${cutoff_date} ]]; then
                    log_info "Removing old S3 backup: ${file}"
                    aws s3 rm "s3://${S3_BUCKET}/postgres-backups/${file}" --region "${S3_REGION}"
                fi
            fi
        done
    fi
}

verify_backup() {
    local backup_file=$1

    log_info "Verifying backup integrity..."

    if [[ "${backup_file}" == *.gpg ]]; then
        log_info "Skipping integrity check for encrypted backup (GPG will verify on decrypt)"
        return 0
    fi

    # Test gzip integrity
    if gzip -t "${backup_file}" 2>&1 | tee -a "${LOG_FILE}"; then
        log_info "Backup integrity verified"
        return 0
    else
        die "Backup integrity check failed"
    fi
}

# ============================================================================
# Main
# ============================================================================

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                ;;
            -c|--container)
                CONTAINER_NAME="$2"
                shift 2
                ;;
            -d|--database)
                DATABASE="$2"
                shift 2
                ;;
            -u|--user)
                DB_USER="$2"
                shift 2
                ;;
            -p|--password)
                DB_PASSWORD="$2"
                shift 2
                ;;
            -o|--output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            -r|--retention)
                RETENTION_DAYS="$2"
                shift 2
                ;;
            -e|--encrypt)
                GPG_RECIPIENT="$2"
                shift 2
                ;;
            -s|--s3-bucket)
                S3_BUCKET="$2"
                shift 2
                ;;
            --s3-region)
                S3_REGION="$2"
                shift 2
                ;;
            --native)
                USE_NATIVE=true
                shift
                ;;
            *)
                die "Unknown option: $1. Use --help for usage information."
                ;;
        esac
    done

    # Validate required parameters
    if [[ -z "${DATABASE}" ]]; then
        die "Database name is required. Use -d or set BACKUP_DATABASE."
    fi

    if [[ -z "${DB_USER}" ]]; then
        die "Database user is required. Use -u or set BACKUP_USER."
    fi

    if [[ -z "${DB_PASSWORD}" ]]; then
        die "Database password is required. Use -p or set PGPASSWORD."
    fi

    # Create output directory
    mkdir -p "${OUTPUT_DIR}"

    # Check dependencies
    check_dependencies

    # Log backup start
    log_info "========================================"
    log_info "Database Backup Started"
    log_info "========================================"
    log_info "Database: ${DATABASE}"
    log_info "User: ${DB_USER}"
    log_info "Output: ${OUTPUT_DIR}"
    log_info "Retention: ${RETENTION_DAYS} days"
    log_info "Timestamp: ${TIMESTAMP}"

    # Create backup
    local backup_file=$(create_backup)

    # Verify backup
    verify_backup "${backup_file}"

    # Encrypt if requested
    if [[ -n "${GPG_RECIPIENT}" ]]; then
        backup_file=$(encrypt_backup "${backup_file}")
    fi

    # Upload to S3 if requested
    if [[ -n "${S3_BUCKET}" ]]; then
        upload_to_s3 "${backup_file}"
    fi

    # Cleanup old backups
    cleanup_old_backups

    # Log completion
    log_info "========================================"
    log_info "Backup Completed Successfully"
    log_info "Final backup: ${backup_file}"
    log_info "========================================"

    exit 0
}

# Run main function
main "$@"
