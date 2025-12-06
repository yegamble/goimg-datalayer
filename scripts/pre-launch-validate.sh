#!/bin/bash
# Pre-Launch Validation Script
# Verifies all P0 launch requirements are in place
#
# Usage: ./scripts/pre-launch-validate.sh [--full]
#   --full: Run full validation including infrastructure tests (requires running services)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

FULL_MODE="${1:-}"
ERRORS=0
WARNINGS=0

print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_check() {
    echo -e "${BLUE}[CHECK]${NC} $1"
}

print_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((ERRORS++))
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((WARNINGS++))
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# ============================================================================
# 1. Secret Management Validation
# ============================================================================
validate_secrets() {
    print_header "1. Secret Management Configuration"

    # Check secret provider files exist
    print_check "Checking secret provider implementation..."
    if [[ -f "internal/infrastructure/secrets/provider.go" ]] && \
       [[ -f "internal/infrastructure/secrets/docker_secrets_provider.go" ]] && \
       [[ -f "internal/infrastructure/secrets/env_provider.go" ]]; then
        print_pass "Secret provider files exist"
    else
        print_fail "Missing secret provider files"
    fi

    # Check tests exist
    if [[ -f "internal/infrastructure/secrets/docker_secrets_provider_test.go" ]]; then
        print_pass "Secret provider tests exist"
    else
        print_warn "Missing secret provider tests"
    fi

    # Check documentation
    if [[ -f "docs/deployment/secret-management.md" ]]; then
        print_pass "Secret management documentation exists"
    else
        print_warn "Missing secret management documentation"
    fi

    # Verify expected secrets are documented
    print_check "Verifying required secrets are defined..."
    local required_secrets=("SecretDBPassword" "SecretJWT" "SecretRedisPassword" "SecretS3AccessKey")
    for secret in "${required_secrets[@]}"; do
        if grep -q "$secret" "internal/infrastructure/secrets/provider.go" 2>/dev/null; then
            print_pass "Secret '$secret' is defined"
        else
            print_warn "Secret '$secret' may not be defined in provider"
        fi
    done
}

# ============================================================================
# 2. SSL/TLS Configuration Validation
# ============================================================================
validate_ssl() {
    print_header "2. SSL/TLS Configuration"

    # Check SSL setup script
    print_check "Checking SSL setup script..."
    if [[ -x "scripts/setup-ssl.sh" ]]; then
        print_pass "SSL setup script exists and is executable"
    else
        print_fail "SSL setup script missing or not executable"
    fi

    # Check nginx configuration
    print_check "Checking nginx SSL configuration..."
    if [[ -f "docker/nginx/nginx.conf" ]]; then
        if grep -q "ssl_protocols TLSv1.2 TLSv1.3" "docker/nginx/nginx.conf"; then
            print_pass "TLS 1.2/1.3 protocols configured"
        else
            print_warn "TLS protocol configuration may need review"
        fi

        if grep -q "ssl_prefer_server_ciphers" "docker/nginx/nginx.conf"; then
            print_pass "Server cipher preference configured"
        else
            print_warn "Server cipher preference not found"
        fi
    else
        print_fail "nginx.conf not found"
    fi

    # Check SSL documentation
    if [[ -f "docs/deployment/ssl-setup.md" ]]; then
        print_pass "SSL setup documentation exists"
    else
        print_warn "Missing SSL setup documentation"
    fi
}

# ============================================================================
# 3. Database Backup Configuration
# ============================================================================
validate_backups() {
    print_header "3. Database Backup Configuration"

    # Check backup scripts
    local backup_scripts=("backup-database.sh" "restore-database.sh" "cleanup-old-backups.sh" "validate-backup-restore.sh")
    for script in "${backup_scripts[@]}"; do
        print_check "Checking scripts/$script..."
        if [[ -x "scripts/$script" ]]; then
            print_pass "$script exists and is executable"
        else
            print_fail "$script missing or not executable"
        fi
    done

    # Check systemd units
    print_check "Checking systemd timer units..."
    if [[ -f "docker/backup/goimg-backup.timer" ]] && [[ -f "docker/backup/goimg-backup.service" ]]; then
        print_pass "Backup systemd units exist"
    else
        print_warn "Backup systemd units missing (manual scheduling required)"
    fi

    # Check backup documentation
    if [[ -f "docs/operations/database-backups.md" ]]; then
        print_pass "Database backup documentation exists"
    else
        print_warn "Missing database backup documentation"
    fi
}

# ============================================================================
# 4. Security Alerting Configuration
# ============================================================================
validate_alerting() {
    print_header "4. Security Alerting Configuration"

    # Check Grafana alerting files
    local alert_files=("security_alerts.yml" "contact_points.yml" "notification_policies.yml")
    for file in "${alert_files[@]}"; do
        print_check "Checking alerting/$file..."
        if [[ -f "monitoring/grafana/provisioning/alerting/$file" ]]; then
            print_pass "$file exists"
        else
            print_fail "$file missing"
        fi
    done

    # Check metrics middleware
    print_check "Checking security metrics in middleware..."
    if grep -q "auth_failures_total" "internal/interfaces/http/middleware/metrics.go" 2>/dev/null; then
        print_pass "Security metrics defined in middleware"
    else
        print_warn "Security metrics may not be defined"
    fi

    # Check alerting documentation
    if [[ -f "docs/operations/security-alerting.md" ]]; then
        print_pass "Security alerting documentation exists"
    else
        print_warn "Missing security alerting documentation"
    fi
}

# ============================================================================
# 5. Load Testing Configuration
# ============================================================================
validate_load_tests() {
    print_header "5. Load Testing Configuration"

    # Check k6 test files
    local load_tests=("auth-flow.js" "browse-flow.js" "upload-flow.js" "social-flow.js" "mixed-traffic.js")
    for test in "${load_tests[@]}"; do
        print_check "Checking tests/load/$test..."
        if [[ -f "tests/load/$test" ]]; then
            print_pass "$test exists"
        else
            print_fail "$test missing"
        fi
    done

    # Check helper files
    if [[ -d "tests/load/helpers" ]]; then
        print_pass "Load test helpers directory exists"
    else
        print_warn "Load test helpers directory missing"
    fi

    # Check Makefile targets
    print_check "Checking Makefile load test targets..."
    if grep -q "load-test:" "Makefile" 2>/dev/null; then
        print_pass "load-test target exists in Makefile"
    else
        print_warn "load-test target not found in Makefile"
    fi

    # Check load test documentation
    if [[ -f "docs/performance/load-testing.md" ]]; then
        print_pass "Load testing documentation exists"
    else
        print_warn "Missing load testing documentation"
    fi
}

# ============================================================================
# 6. Security Testing Configuration
# ============================================================================
validate_security_tests() {
    print_header "6. Security Testing Configuration"

    # Check security test scripts
    local security_tests=("auth-security-tests.sh" "injection-tests.sh" "upload-security-tests.sh" "access-control-tests.sh")
    for test in "${security_tests[@]}"; do
        print_check "Checking tests/security/$test..."
        if [[ -x "tests/security/$test" ]]; then
            print_pass "$test exists and is executable"
        else
            print_fail "$test missing or not executable"
        fi
    done

    # Check pentest documentation
    if [[ -f "docs/security/pentest-plan.md" ]] || [[ -f "tests/security/README_PENTEST.md" ]]; then
        print_pass "Penetration testing documentation exists"
    else
        print_warn "Missing penetration testing documentation"
    fi

    # Check security checklist
    if [[ -f "docs/security/pre-launch-security-checklist.md" ]]; then
        print_pass "Pre-launch security checklist exists"
    else
        print_fail "Pre-launch security checklist missing"
    fi
}

# ============================================================================
# 7. OpenAPI Validation
# ============================================================================
validate_openapi() {
    print_header "7. OpenAPI Specification"

    # Check OpenAPI file exists
    print_check "Checking OpenAPI specification..."
    if [[ -f "api/openapi/openapi.yaml" ]]; then
        print_pass "OpenAPI specification exists"

        # Check key endpoints are defined
        local endpoints=("/auth/register" "/auth/login" "/images" "/albums")
        for endpoint in "${endpoints[@]}"; do
            if grep -q "$endpoint" "api/openapi/openapi.yaml"; then
                print_pass "Endpoint $endpoint is defined"
            else
                print_warn "Endpoint $endpoint may not be defined"
            fi
        done
    else
        print_fail "OpenAPI specification missing"
    fi

    # Check if validate-openapi target exists
    if grep -q "validate-openapi:" "Makefile" 2>/dev/null; then
        print_pass "validate-openapi target exists in Makefile"
    else
        print_warn "validate-openapi target not found in Makefile"
    fi
}

# ============================================================================
# 8. Documentation Validation
# ============================================================================
validate_documentation() {
    print_header "8. Launch Documentation"

    local required_docs=(
        "docs/LAUNCH_REQUIREMENTS.md"
        "docs/security/pre-launch-security-checklist.md"
        "README.md"
    )

    for doc in "${required_docs[@]}"; do
        print_check "Checking $doc..."
        if [[ -f "$doc" ]]; then
            print_pass "$doc exists"
        else
            print_fail "$doc missing"
        fi
    done

    # Check Claude documentation
    if [[ -f "CLAUDE.md" ]]; then
        print_pass "Agent documentation (CLAUDE.md) exists"
    else
        print_warn "CLAUDE.md missing"
    fi
}

# ============================================================================
# 9. Full Infrastructure Tests (optional)
# ============================================================================
run_infrastructure_tests() {
    if [[ "$FULL_MODE" != "--full" ]]; then
        print_header "9. Infrastructure Tests (SKIPPED)"
        print_info "Run with --full flag to execute infrastructure tests"
        print_info "Requires: Docker, PostgreSQL, Redis running"
        return
    fi

    print_header "9. Infrastructure Tests"

    # Check if services are running
    print_check "Checking Docker services..."
    if docker ps &>/dev/null; then
        print_pass "Docker is running"
    else
        print_fail "Docker is not running or not accessible"
        return
    fi

    # Try to run tests
    print_check "Running unit tests..."
    if make test 2>/dev/null; then
        print_pass "Unit tests passed"
    else
        print_fail "Unit tests failed"
    fi

    # Validate OpenAPI
    print_check "Validating OpenAPI spec..."
    if make validate-openapi 2>/dev/null; then
        print_pass "OpenAPI validation passed"
    else
        print_fail "OpenAPI validation failed"
    fi
}

# ============================================================================
# Main
# ============================================================================
main() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║          GoImg Pre-Launch Validation Script                ║"
    echo "║                                                            ║"
    echo "║  Verifying P0 launch requirements are in place             ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    validate_secrets
    validate_ssl
    validate_backups
    validate_alerting
    validate_load_tests
    validate_security_tests
    validate_openapi
    validate_documentation
    run_infrastructure_tests

    # Summary
    print_header "VALIDATION SUMMARY"

    if [[ $ERRORS -eq 0 ]] && [[ $WARNINGS -eq 0 ]]; then
        echo -e "${GREEN}All checks passed! Ready for launch.${NC}"
    elif [[ $ERRORS -eq 0 ]]; then
        echo -e "${YELLOW}Passed with $WARNINGS warning(s). Review before launch.${NC}"
    else
        echo -e "${RED}Failed with $ERRORS error(s) and $WARNINGS warning(s).${NC}"
        echo -e "${RED}Address errors before launching.${NC}"
    fi

    echo ""
    echo "Next steps:"
    echo "  1. Review docs/security/pre-launch-security-checklist.md"
    echo "  2. Configure production secrets"
    echo "  3. Run: make load-test"
    echo "  4. Run: tests/security/auth-security-tests.sh"
    echo "  5. Validate backups: scripts/validate-backup-restore.sh"

    exit $ERRORS
}

main "$@"
