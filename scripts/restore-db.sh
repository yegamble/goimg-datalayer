#!/bin/bash
# ============================================================================
# PostgreSQL Database Restore Script
# ============================================================================
# Restores a database from a backup created by backup-db.sh
#
# Usage:
#   ./restore-db.sh -f <backup-file> -d <database> -u <user> -p <password>
#
# Features:
# - Supports gzip compressed backups
# - Supports GPG encrypted backups
# - Supports S3 download
# - Docker and native PostgreSQL support
# - Safety checks before restore

set -euo pipefail

# ============================================================================
# Configuration
# ============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTAINER_NAME="${RESTORE_CONTAINER:-goimg-postgres}"
DATABASE="${RESTORE_DATABASE:-}"
DB_USER="${RESTORE_USER:-}"
DB_PASSWORD="${PGPASSWORD:-}"
BACKUP_FILE="${RESTORE_FILE:-}"
S3_BUCKET="${RESTORE_S3_BUCKET:-}"
S3_REGION="${RESTORE_S3_REGION:-us-east-1}"
USE_NATIVE="${RESTORE_USE_NATIVE:-false}"
FORCE="${RESTORE_FORCE:-false}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# ============================================================================
# Functions
# ============================================================================

log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

die() {
    log_error "$*"
    exit 1
}

show_help() {
    cat << EOF
PostgreSQL Database Restore Script

Usage: $0 [options]

Options:
  -h, --help              Show this help message
  -f, --file PATH         Backup file path (required, unless using S3)
  -c, --container NAME    Docker container name (default: goimg-postgres)
  -d, --database NAME     Database name (required)
  -u, --user NAME         Database user (required)
  -p, --password PASS     Database password (or use PGPASSWORD env var)
  -s, --s3-path PATH      Download from S3 (e.g., s3://bucket/path/backup.sql.gz)
  --s3-region REGION      S3 region (default: us-east-1)
  --native                Use native psql (not Docker)
  --force                 Skip confirmation prompt

Examples:
  # Restore from local file
  $0 -f backups/goimg_20240101_120000.sql.gz -d goimg -u postgres -p password

  # Restore from S3
  $0 -s s3://my-bucket/backups/goimg_20240101_120000.sql.gz -d goimg -u postgres

  # Restore encrypted backup
  $0 -f backups/goimg_20240101_120000.sql.gz.gpg -d goimg -u postgres

EOF
    exit 0
}

check_dependencies() {
    local deps=("gzip")

    if [[ "${USE_NATIVE}" == "true" ]]; then
        deps+=("psql")
    else
        deps+=("docker")
    fi

    if [[ -n "${S3_BUCKET}" ]]; then
        deps+=("aws")
    fi

    for cmd in "${deps[@]}"; do
        if ! command -v "${cmd}" &> /dev/null; then
            die "Required command '${cmd}' not found."
        fi
    done
}

download_from_s3() {
    local s3_path=$1
    local local_file="${SCRIPT_DIR}/tmp_restore_$(date +%s).sql.gz"

    log_info "Downloading backup from S3: ${s3_path}..."

    aws s3 cp \
        "${s3_path}" \
        "${local_file}" \
        --region "${S3_REGION}" || die "S3 download failed"

    log_info "Downloaded to: ${local_file}"
    echo "${local_file}"
}

decrypt_backup() {
    local encrypted_file=$1
    local decrypted_file="${encrypted_file%.gpg}"

    log_info "Decrypting backup with GPG..."

    gpg --decrypt \
        --output "${decrypted_file}" \
        "${encrypted_file}" || die "Decryption failed"

    log_info "Backup decrypted: ${decrypted_file}"
    echo "${decrypted_file}"
}

confirm_restore() {
    if [[ "${FORCE}" == "true" ]]; then
        return 0
    fi

    log_warn "WARNING: This will REPLACE all data in database '${DATABASE}'"
    log_warn "Backup file: ${BACKUP_FILE}"
    echo -n "Are you sure you want to continue? [y/N] "
    read -r response

    if [[ ! "${response}" =~ ^[Yy]$ ]]; then
        log_info "Restore cancelled by user"
        exit 0
    fi
}

create_database_if_not_exists() {
    log_info "Checking if database '${DATABASE}' exists..."

    if [[ "${USE_NATIVE}" == "true" ]]; then
        PGPASSWORD="${DB_PASSWORD}" psql -h localhost -U "${DB_USER}" -lqt | \
            cut -d \| -f 1 | grep -qw "${DATABASE}" && return 0

        log_info "Creating database '${DATABASE}'..."
        PGPASSWORD="${DB_PASSWORD}" psql -h localhost -U "${DB_USER}" -c "CREATE DATABASE ${DATABASE};"
    else
        docker exec -e PGPASSWORD="${DB_PASSWORD}" "${CONTAINER_NAME}" \
            psql -U "${DB_USER}" -lqt | cut -d \| -f 1 | grep -qw "${DATABASE}" && return 0

        log_info "Creating database '${DATABASE}'..."
        docker exec -e PGPASSWORD="${DB_PASSWORD}" "${CONTAINER_NAME}" \
            psql -U "${DB_USER}" -c "CREATE DATABASE ${DATABASE};"
    fi
}

restore_backup() {
    local backup_file=$1

    log_info "Starting restore of database '${DATABASE}'..."
    log_info "Backup file: ${backup_file}"

    # Ensure database exists
    create_database_if_not_exists

    # Drop existing connections to the database
    log_warn "Terminating existing connections to '${DATABASE}'..."
    if [[ "${USE_NATIVE}" == "true" ]]; then
        PGPASSWORD="${DB_PASSWORD}" psql -h localhost -U "${DB_USER}" -d postgres -c \
            "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${DATABASE}' AND pid <> pg_backend_pid();" || true
    else
        docker exec -e PGPASSWORD="${DB_PASSWORD}" "${CONTAINER_NAME}" \
            psql -U "${DB_USER}" -d postgres -c \
            "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${DATABASE}' AND pid <> pg_backend_pid();" || true
    fi

    # Perform restore
    if [[ "${USE_NATIVE}" == "true" ]]; then
        log_info "Restoring with native psql..."
        gunzip -c "${backup_file}" | PGPASSWORD="${DB_PASSWORD}" psql \
            -h localhost \
            -U "${DB_USER}" \
            -d "${DATABASE}" \
            --single-transaction || die "Restore failed"
    else
        log_info "Restoring with Docker container '${CONTAINER_NAME}'..."
        gunzip -c "${backup_file}" | docker exec -i -e PGPASSWORD="${DB_PASSWORD}" "${CONTAINER_NAME}" \
            psql -U "${DB_USER}" -d "${DATABASE}" || die "Restore failed"
    fi

    log_info "Restore completed successfully"
}

cleanup() {
    # Clean up temporary files
    if [[ -n "${TEMP_FILE:-}" ]] && [[ -f "${TEMP_FILE}" ]]; then
        log_info "Cleaning up temporary files..."
        rm -f "${TEMP_FILE}"
    fi
}

# ============================================================================
# Main
# ============================================================================

main() {
    # Setup cleanup trap
    trap cleanup EXIT

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                ;;
            -f|--file)
                BACKUP_FILE="$2"
                shift 2
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
            -s|--s3-path)
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
            --force)
                FORCE=true
                shift
                ;;
            *)
                die "Unknown option: $1"
                ;;
        esac
    done

    # Validate required parameters
    if [[ -z "${DATABASE}" ]]; then
        die "Database name is required. Use -d or set RESTORE_DATABASE."
    fi

    if [[ -z "${DB_USER}" ]]; then
        die "Database user is required. Use -u or set RESTORE_USER."
    fi

    if [[ -z "${DB_PASSWORD}" ]]; then
        die "Database password is required. Use -p or set PGPASSWORD."
    fi

    if [[ -z "${BACKUP_FILE}" ]] && [[ -z "${S3_BUCKET}" ]]; then
        die "Backup file or S3 path is required. Use -f or -s."
    fi

    # Check dependencies
    check_dependencies

    # Download from S3 if needed
    if [[ -n "${S3_BUCKET}" ]]; then
        BACKUP_FILE=$(download_from_s3 "${S3_BUCKET}")
        TEMP_FILE="${BACKUP_FILE}"
    fi

    # Verify backup file exists
    if [[ ! -f "${BACKUP_FILE}" ]]; then
        die "Backup file not found: ${BACKUP_FILE}"
    fi

    # Decrypt if GPG encrypted
    if [[ "${BACKUP_FILE}" == *.gpg ]]; then
        BACKUP_FILE=$(decrypt_backup "${BACKUP_FILE}")
        TEMP_FILE="${BACKUP_FILE}"
    fi

    # Confirm restore
    confirm_restore

    # Perform restore
    restore_backup "${BACKUP_FILE}"

    log_info "========================================"
    log_info "Database Restored Successfully"
    log_info "Database: ${DATABASE}"
    log_info "========================================"

    exit 0
}

# Run main function
main "$@"
