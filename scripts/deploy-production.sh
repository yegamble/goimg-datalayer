#!/bin/bash
# ============================================================================
# Production Deployment Automation Script
# ============================================================================
# Automates the deployment of goimg-datalayer to production
#
# Usage:
#   ./deploy-production.sh [options]
#
# Prerequisites:
#   - Docker and Docker Compose installed
#   - Domain name configured with DNS pointing to server
#   - .env.prod file created and configured
#   - SSL certificates obtained

set -euo pipefail

# ============================================================================
# Configuration
# ============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "${SCRIPT_DIR}")"
DOCKER_DIR="${PROJECT_ROOT}/docker"
ENV_FILE="${DOCKER_DIR}/.env.prod"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_step() {
    echo -e "${BLUE}[STEP]${NC} $*"
}

die() {
    log_error "$*"
    exit 1
}

show_help() {
    cat << EOF
Production Deployment Script

Usage: $0 [options]

Options:
  -h, --help              Show this help message
  -s, --skip-build        Skip Docker image build
  -m, --skip-migrations   Skip database migrations
  -b, --backup            Create backup before deployment
  -d, --dry-run          Show what would be done without executing

Examples:
  # Full deployment
  $0

  # Deploy with backup
  $0 --backup

  # Skip build (use existing images)
  $0 --skip-build

  # Dry run
  $0 --dry-run

EOF
    exit 0
}

check_prerequisites() {
    log_step "Checking prerequisites..."

    # Check if running as root
    if [[ $EUID -eq 0 ]]; then
        log_warn "Running as root. Consider using a non-root user with Docker permissions."
    fi

    # Check Docker
    if ! command -v docker &> /dev/null; then
        die "Docker is not installed. Please install Docker first."
    fi

    # Check Docker Compose
    if ! docker compose version &> /dev/null && ! command -v docker-compose &> /dev/null; then
        die "Docker Compose is not installed. Please install Docker Compose first."
    fi

    # Check .env.prod file
    if [[ ! -f "${ENV_FILE}" ]]; then
        die ".env.prod file not found at ${ENV_FILE}. Please create it from .env.prod.example"
    fi

    # Check SSL certificates
    if [[ ! -f "${DOCKER_DIR}/nginx/ssl/fullchain.pem" ]] || [[ ! -f "${DOCKER_DIR}/nginx/ssl/privkey.pem" ]]; then
        log_warn "SSL certificates not found. HTTPS will not work."
        log_warn "Generate certificates with: sudo certbot certonly --standalone -d yourdomain.com"
        read -p "Continue without SSL? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 0
        fi
    fi

    # Load environment
    set -a
    source "${ENV_FILE}"
    set +a

    # Check critical environment variables
    local required_vars=(
        "DB_USER"
        "DB_PASSWORD"
        "DB_NAME"
        "REDIS_PASSWORD"
        "JWT_SECRET"
        "CORS_ALLOWED_ORIGINS"
    )

    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            die "Required environment variable ${var} is not set in .env.prod"
        fi
    done

    log_info "Prerequisites check passed"
}

create_backup() {
    log_step "Creating database backup before deployment..."

    if [[ ! -f "${SCRIPT_DIR}/backup-db.sh" ]]; then
        log_warn "Backup script not found. Skipping backup."
        return 0
    fi

    local backup_dir="${PROJECT_ROOT}/backups/pre-deploy"
    mkdir -p "${backup_dir}"

    "${SCRIPT_DIR}/backup-db.sh" \
        -d "${DB_NAME}" \
        -u "${DB_USER}" \
        -p "${DB_PASSWORD}" \
        -o "${backup_dir}" || log_warn "Backup failed, but continuing..."

    log_info "Backup created successfully"
}

build_images() {
    log_step "Building Docker images..."

    cd "${PROJECT_ROOT}"

    log_info "Building API image..."
    docker build \
        -f "${DOCKER_DIR}/Dockerfile.api" \
        -t goimg-api:latest \
        -t "goimg-api:$(date +%Y%m%d-%H%M%S)" \
        .

    log_info "Building Worker image..."
    docker build \
        -f "${DOCKER_DIR}/Dockerfile.worker" \
        -t goimg-worker:latest \
        -t "goimg-worker:$(date +%Y%m%d-%H%M%S)" \
        .

    log_info "Images built successfully"
}

run_migrations() {
    log_step "Running database migrations..."

    cd "${PROJECT_ROOT}"

    # Ensure PostgreSQL is running
    docker compose -f "${DOCKER_DIR}/docker-compose.prod.yml" up -d postgres

    # Wait for PostgreSQL
    log_info "Waiting for PostgreSQL to be ready..."
    for i in {1..30}; do
        if docker exec goimg-postgres pg_isready -U "${DB_USER}" &> /dev/null; then
            log_info "PostgreSQL is ready"
            break
        fi
        if [[ $i -eq 30 ]]; then
            die "PostgreSQL did not become ready in time"
        fi
        sleep 2
    done

    # Run migrations
    log_info "Applying migrations..."
    export GOOSE_DRIVER=postgres
    export GOOSE_DBSTRING="host=localhost port=5432 user=${DB_USER} password=${DB_PASSWORD} dbname=${DB_NAME} sslmode=${DB_SSL_MODE:-require}"

    if command -v goose &> /dev/null; then
        goose -dir migrations up
    elif command -v make &> /dev/null; then
        make migrate-up
    else
        log_warn "Goose not found. Skipping migrations."
        log_warn "Run migrations manually with: make migrate-up"
    fi

    log_info "Migrations completed"
}

deploy_services() {
    log_step "Deploying services..."

    cd "${DOCKER_DIR}"

    # Pull any external images
    log_info "Pulling external images..."
    docker compose -f docker-compose.prod.yml pull postgres redis clamav nginx || true

    # Start services
    log_info "Starting services..."
    docker compose -f docker-compose.prod.yml --env-file .env.prod up -d

    log_info "Services started successfully"
}

verify_deployment() {
    log_step "Verifying deployment..."

    # Wait for services to be ready
    log_info "Waiting for services to be healthy..."
    sleep 10

    # Check container status
    log_info "Container status:"
    docker compose -f "${DOCKER_DIR}/docker-compose.prod.yml" ps

    # Check health endpoints
    log_info "Checking health endpoints..."

    local max_retries=30
    local retry=0
    local health_ok=false

    while [[ $retry -lt $max_retries ]]; do
        if curl -sf http://localhost:8080/health &> /dev/null; then
            health_ok=true
            break
        fi
        ((retry++))
        sleep 2
    done

    if [[ $health_ok == true ]]; then
        log_info "Health check passed: http://localhost:8080/health"
    else
        log_warn "Health check failed. Check logs with: docker logs goimg-api"
    fi

    # Show resource usage
    log_info "Resource usage:"
    docker stats --no-stream

    log_info "Deployment verification completed"
}

show_next_steps() {
    cat << EOF

${GREEN}========================================
Deployment Completed Successfully!
========================================${NC}

Next Steps:
-----------

1. Verify the API is accessible:
   ${BLUE}curl https://yourdomain.com/health${NC}

2. Check logs:
   ${BLUE}docker compose -f docker/docker-compose.prod.yml logs -f${NC}

3. Set up automated backups (if not already done):
   ${BLUE}sudo crontab -e${NC}
   Add: 0 2 * * * cd ${PROJECT_ROOT} && ./scripts/backup-db.sh -d ${DB_NAME} -u ${DB_USER} -s your-backup-bucket

4. Configure SSL auto-renewal:
   ${BLUE}sudo crontab -e${NC}
   Add: 0 0 * * * certbot renew --quiet --post-hook "docker exec goimg-nginx nginx -s reload"

5. Set up monitoring:
   - Prometheus: http://localhost:9090/metrics
   - Consider: Grafana, Datadog, or New Relic

6. Complete security checklist:
   ${BLUE}docs/deployment/SECURITY-CHECKLIST.md${NC}

7. Review logs for any issues:
   ${BLUE}docker logs goimg-api${NC}
   ${BLUE}docker logs goimg-worker${NC}

For detailed documentation, see:
${BLUE}docs/deployment/README.md${NC}

EOF
}

cleanup() {
    # Remove old images
    log_info "Cleaning up old images..."
    docker image prune -f || true
}

# ============================================================================
# Main
# ============================================================================

main() {
    local skip_build=false
    local skip_migrations=false
    local create_backup=false
    local dry_run=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                ;;
            -s|--skip-build)
                skip_build=true
                shift
                ;;
            -m|--skip-migrations)
                skip_migrations=true
                shift
                ;;
            -b|--backup)
                create_backup=true
                shift
                ;;
            -d|--dry-run)
                dry_run=true
                shift
                ;;
            *)
                die "Unknown option: $1. Use --help for usage information."
                ;;
        esac
    done

    echo -e "${GREEN}"
    cat << 'EOF'
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║   goimg-datalayer Production Deployment                  ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"

    if [[ $dry_run == true ]]; then
        log_warn "DRY RUN MODE - No changes will be made"
    fi

    # Step 1: Check prerequisites
    check_prerequisites

    # Step 2: Create backup if requested
    if [[ $create_backup == true ]]; then
        if [[ $dry_run == false ]]; then
            create_backup
        else
            log_info "[DRY RUN] Would create backup"
        fi
    fi

    # Step 3: Build images
    if [[ $skip_build == false ]]; then
        if [[ $dry_run == false ]]; then
            build_images
        else
            log_info "[DRY RUN] Would build Docker images"
        fi
    else
        log_info "Skipping image build"
    fi

    # Step 4: Run migrations
    if [[ $skip_migrations == false ]]; then
        if [[ $dry_run == false ]]; then
            run_migrations
        else
            log_info "[DRY RUN] Would run database migrations"
        fi
    else
        log_info "Skipping migrations"
    fi

    # Step 5: Deploy services
    if [[ $dry_run == false ]]; then
        deploy_services
    else
        log_info "[DRY RUN] Would deploy services"
    fi

    # Step 6: Verify deployment
    if [[ $dry_run == false ]]; then
        verify_deployment
    else
        log_info "[DRY RUN] Would verify deployment"
    fi

    # Step 7: Cleanup
    if [[ $dry_run == false ]]; then
        cleanup
    else
        log_info "[DRY RUN] Would cleanup old images"
    fi

    # Show next steps
    if [[ $dry_run == false ]]; then
        show_next_steps
    else
        log_info "[DRY RUN] Deployment simulation completed"
    fi

    exit 0
}

# Run main function
main "$@"
