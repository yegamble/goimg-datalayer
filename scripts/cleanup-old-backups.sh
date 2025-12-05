#!/bin/bash
# ============================================================================
# PostgreSQL Backup Cleanup Script with Rotation Policy
# ============================================================================
# Implements intelligent backup retention policy:
#   - Daily backups: Retain for 7 days
#   - Weekly backups (Sunday): Retain for 4 weeks (28 days)
#   - Monthly backups (1st of month): Retain for 6 months (180 days)
#
# Features:
# - Cleans both local and S3 storage
# - Preserves weekly and monthly backups automatically
# - Dry-run mode for testing
# - Comprehensive logging
# - Safe deletion with verification
#
# Exit Codes:
#   0 - Success
#   1 - General error
#   2 - Missing dependencies
#   3 - Configuration error
#
# Usage:
#   ./cleanup-old-backups.sh [--dry-run]
#
# Environment Variables:
#   BACKUP_DIR                 - Local backup directory (default: /var/backups/postgres)
#   DAILY_RETENTION_DAYS       - Days to keep daily backups (default: 7)
#   WEEKLY_RETENTION_WEEKS     - Weeks to keep weekly backups (default: 4)
#   MONTHLY_RETENTION_MONTHS   - Months to keep monthly backups (default: 6)
#   S3_ENDPOINT                - S3 endpoint URL
#   S3_BUCKET                  - S3 bucket name
#   S3_ACCESS_KEY              - S3 access key ID
#   S3_SECRET_KEY              - S3 secret access key
#   S3_PATH_PREFIX             - S3 path prefix (default: postgres-backups)
#
# ============================================================================

set -euo pipefail

# ============================================================================
# Configuration with Defaults
# ============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Backup Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/postgres}"
BACKUP_PATTERN="goimg-backup-*.dump*"

# Retention Policy Configuration
DAILY_RETENTION_DAYS="${DAILY_RETENTION_DAYS:-7}"
WEEKLY_RETENTION_WEEKS="${WEEKLY_RETENTION_WEEKS:-4}"
MONTHLY_RETENTION_MONTHS="${MONTHLY_RETENTION_MONTHS:-6}"

# Calculate retention periods in days
WEEKLY_RETENTION_DAYS=$((WEEKLY_RETENTION_WEEKS * 7))
MONTHLY_RETENTION_DAYS=$((MONTHLY_RETENTION_MONTHS * 30))

# S3 Configuration
S3_ENDPOINT="${S3_ENDPOINT:-}"
S3_BUCKET="${S3_BUCKET:-}"
S3_ACCESS_KEY="${S3_ACCESS_KEY:-}"
S3_SECRET_KEY="${S3_SECRET_KEY:-}"
S3_PATH_PREFIX="${S3_PATH_PREFIX:-postgres-backups}"

# Logging Configuration
LOG_DIR="${BACKUP_DIR}/logs"
LOG_FILE="${LOG_DIR}/cleanup-$(date +%Y%m%d).log"

# Dry Run Mode
DRY_RUN=false

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
    exit "${exit_code}"
}

# ============================================================================
# Validation Functions
# ============================================================================

check_dependencies() {
    local missing_deps=()

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
    if [[ ! -d "${BACKUP_DIR}" ]]; then
        log_warn "Backup directory does not exist: ${BACKUP_DIR}"
        log_warn "Creating backup directory..."
        mkdir -p "${BACKUP_DIR}"
        mkdir -p "${LOG_DIR}"
    fi

    if [[ -n "${S3_BUCKET}" ]]; then
        if [[ -z "${S3_ENDPOINT}" ]] || [[ -z "${S3_ACCESS_KEY}" ]] || [[ -z "${S3_SECRET_KEY}" ]]; then
            die "S3 cleanup requires S3_ENDPOINT, S3_ACCESS_KEY, and S3_SECRET_KEY" 3
        fi
    fi

    log_info "Configuration validated"
}

# ============================================================================
# Helper Functions
# ============================================================================

# Extract date from backup filename (format: goimg-backup-YYYYMMDD-HHMMSS.dump)
extract_date_from_filename() {
    local filename=$1
    # Extract YYYYMMDD from filename
    if [[ ${filename} =~ goimg-backup-([0-9]{8})-[0-9]{6} ]]; then
        echo "${BASH_REMATCH[1]}"
        return 0
    fi
    return 1
}

# Check if backup is a weekly backup (Sunday)
is_weekly_backup() {
    local date_str=$1  # YYYYMMDD format
    local day_of_week=$(date -d "${date_str}" +%u 2>/dev/null || date -j -f "%Y%m%d" "${date_str}" +%u 2>/dev/null)
    [[ "${day_of_week}" == "7" ]]
}

# Check if backup is a monthly backup (1st of month)
is_monthly_backup() {
    local date_str=$1  # YYYYMMDD format
    local day_of_month=$(date -d "${date_str}" +%d 2>/dev/null || date -j -f "%Y%m%d" "${date_str}" +%d 2>/dev/null)
    [[ "${day_of_month}" == "01" ]]
}

# Calculate age in days
get_backup_age_days() {
    local date_str=$1  # YYYYMMDD format
    local backup_date=$(date -d "${date_str}" +%s 2>/dev/null || date -j -f "%Y%m%d" "${date_str}" +%s 2>/dev/null)
    local current_date=$(date +%s)
    local age_seconds=$((current_date - backup_date))
    echo $((age_seconds / 86400))
}

# Determine if backup should be kept based on retention policy
should_keep_backup() {
    local filename=$1
    local date_str

    date_str=$(extract_date_from_filename "${filename}")
    if [[ $? -ne 0 ]]; then
        log_warn "Could not extract date from filename: ${filename}"
        return 0  # Keep if we can't parse date
    fi

    local age_days
    age_days=$(get_backup_age_days "${date_str}")

    # Monthly backups: keep for MONTHLY_RETENTION_DAYS
    if is_monthly_backup "${date_str}"; then
        if [[ ${age_days} -le ${MONTHLY_RETENTION_DAYS} ]]; then
            log_debug "Keeping monthly backup (${age_days} days old): ${filename}"
            return 0
        else
            log_debug "Monthly backup expired (${age_days} > ${MONTHLY_RETENTION_DAYS} days): ${filename}"
            return 1
        fi
    fi

    # Weekly backups: keep for WEEKLY_RETENTION_DAYS
    if is_weekly_backup "${date_str}"; then
        if [[ ${age_days} -le ${WEEKLY_RETENTION_DAYS} ]]; then
            log_debug "Keeping weekly backup (${age_days} days old): ${filename}"
            return 0
        else
            log_debug "Weekly backup expired (${age_days} > ${WEEKLY_RETENTION_DAYS} days): ${filename}"
            return 1
        fi
    fi

    # Daily backups: keep for DAILY_RETENTION_DAYS
    if [[ ${age_days} -le ${DAILY_RETENTION_DAYS} ]]; then
        log_debug "Keeping daily backup (${age_days} days old): ${filename}"
        return 0
    else
        log_debug "Daily backup expired (${age_days} > ${DAILY_RETENTION_DAYS} days): ${filename}"
        return 1
    fi
}

# ============================================================================
# Cleanup Functions
# ============================================================================

cleanup_local_backups() {
    log_info "Cleaning up local backups in: ${BACKUP_DIR}"
    log_info "Retention policy:"
    log_info "  - Daily backups: ${DAILY_RETENTION_DAYS} days"
    log_info "  - Weekly backups (Sunday): ${WEEKLY_RETENTION_WEEKS} weeks (${WEEKLY_RETENTION_DAYS} days)"
    log_info "  - Monthly backups (1st): ${MONTHLY_RETENTION_MONTHS} months (${MONTHLY_RETENTION_DAYS} days)"

    local total_count=0
    local kept_count=0
    local deleted_count=0
    local deleted_size=0

    # Find all backup files
    while IFS= read -r -d '' filepath; do
        ((total_count++))

        local filename=$(basename "${filepath}")

        if should_keep_backup "${filename}"; then
            ((kept_count++))
        else
            local filesize=$(stat -f%z "${filepath}" 2>/dev/null || stat -c%s "${filepath}" 2>/dev/null || echo 0)
            deleted_size=$((deleted_size + filesize))

            if [[ "${DRY_RUN}" == "true" ]]; then
                log_info "[DRY RUN] Would delete: ${filename}"
            else
                log_info "Deleting: ${filename}"
                rm -f "${filepath}" || log_error "Failed to delete: ${filepath}"
            fi
            ((deleted_count++))
        fi
    done < <(find "${BACKUP_DIR}" -name "${BACKUP_PATTERN}" -type f -print0 2>/dev/null)

    local deleted_size_human=$(numfmt --to=iec-i --suffix=B "${deleted_size}" 2>/dev/null || echo "${deleted_size} bytes")

    log_info "Local cleanup summary:"
    log_info "  Total backups: ${total_count}"
    log_info "  Kept: ${kept_count}"
    log_info "  Deleted: ${deleted_count}"
    log_info "  Space freed: ${deleted_size_human}"
}

cleanup_s3_backups() {
    if [[ -z "${S3_BUCKET}" ]]; then
        log_debug "S3 cleanup not configured, skipping"
        return 0
    fi

    log_info "Cleaning up S3 backups"
    log_info "Endpoint: ${S3_ENDPOINT}"
    log_info "Bucket: ${S3_BUCKET}"
    log_info "Path: ${S3_PATH_PREFIX}/"

    # Configure AWS CLI
    export AWS_ACCESS_KEY_ID="${S3_ACCESS_KEY}"
    export AWS_SECRET_ACCESS_KEY="${S3_SECRET_KEY}"

    local total_count=0
    local kept_count=0
    local deleted_count=0

    # List all backup files in S3
    local s3_files
    s3_files=$(aws s3 ls "s3://${S3_BUCKET}/${S3_PATH_PREFIX}/" \
        --endpoint-url "${S3_ENDPOINT}" \
        --recursive | awk '{print $4}' | grep "${BACKUP_PATTERN//\*/}" || true)

    if [[ -z "${s3_files}" ]]; then
        log_info "No S3 backups found"
        unset AWS_ACCESS_KEY_ID
        unset AWS_SECRET_ACCESS_KEY
        return 0
    fi

    while IFS= read -r s3_key; do
        ((total_count++))

        local filename=$(basename "${s3_key}")

        if should_keep_backup "${filename}"; then
            ((kept_count++))
        else
            if [[ "${DRY_RUN}" == "true" ]]; then
                log_info "[DRY RUN] Would delete S3: ${filename}"
            else
                log_info "Deleting S3: ${filename}"
                aws s3 rm "s3://${S3_BUCKET}/${s3_key}" \
                    --endpoint-url "${S3_ENDPOINT}" || log_error "Failed to delete S3: ${s3_key}"
            fi
            ((deleted_count++))
        fi
    done <<< "${s3_files}"

    log_info "S3 cleanup summary:"
    log_info "  Total backups: ${total_count}"
    log_info "  Kept: ${kept_count}"
    log_info "  Deleted: ${deleted_count}"

    # Cleanup credentials
    unset AWS_ACCESS_KEY_ID
    unset AWS_SECRET_ACCESS_KEY
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --debug)
                DEBUG=true
                shift
                ;;
            -h|--help)
                cat << EOF
PostgreSQL Backup Cleanup Script

Usage: $0 [options]

Options:
  --dry-run    Show what would be deleted without actually deleting
  --debug      Enable debug logging
  -h, --help   Show this help message

Retention Policy:
  - Daily backups: ${DAILY_RETENTION_DAYS} days
  - Weekly backups (Sunday): ${WEEKLY_RETENTION_WEEKS} weeks
  - Monthly backups (1st): ${MONTHLY_RETENTION_MONTHS} months

Environment Variables:
  BACKUP_DIR                 - Local backup directory
  DAILY_RETENTION_DAYS       - Days to keep daily backups
  WEEKLY_RETENTION_WEEKS     - Weeks to keep weekly backups
  MONTHLY_RETENTION_MONTHS   - Months to keep monthly backups
  S3_ENDPOINT                - S3 endpoint URL
  S3_BUCKET                  - S3 bucket name
  S3_ACCESS_KEY              - S3 access key ID
  S3_SECRET_KEY              - S3 secret access key

EOF
                exit 0
                ;;
            *)
                die "Unknown option: $1. Use --help for usage information."
                ;;
        esac
    done

    mkdir -p "${LOG_DIR}"

    log_info "========================================"
    log_info "Backup Cleanup Started"
    log_info "========================================"
    log_info "Timestamp: $(date '+%Y-%m-%d %H:%M:%S')"
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "MODE: DRY RUN (no files will be deleted)"
    fi

    # Validate dependencies and configuration
    check_dependencies
    validate_configuration

    # Cleanup local backups
    cleanup_local_backups

    # Cleanup S3 backups
    cleanup_s3_backups

    log_info "========================================"
    log_info "Cleanup Completed Successfully"
    log_info "========================================"

    exit 0
}

# Run main function
main "$@"
