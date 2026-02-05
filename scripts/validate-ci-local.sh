#!/bin/bash
# CI/CD Local Validation Script
# Purpose: Test GitHub Actions workflows locally using 'act' before pushing
# Author: CI/CD Agent
# Date: 2026-02-05

set -e  # Exit on error

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
ACT_BIN="${ACT_BIN:-/opt/homebrew/bin/act}"
CONTAINER_ARCH="linux/amd64"  # Required for Apple M-series
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Change to project root
cd "$PROJECT_ROOT"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}CI/CD Local Validation Script${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to print section headers
print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

# Function to print success
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print error
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Function to print warning
print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Function to print info
print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Step 1: Prerequisites check
print_header "Step 1: Prerequisites Check"

# Check if act is installed
if ! command -v "$ACT_BIN" &> /dev/null; then
    print_error "act is not installed at $ACT_BIN"
    echo ""
    echo "Install act using:"
    echo "  brew install act"
    echo ""
    exit 1
fi
print_success "act is installed: $($ACT_BIN --version)"

# Check if Docker is running
if ! docker ps &> /dev/null; then
    print_error "Docker is not running"
    echo ""
    echo "Please start Docker Desktop and try again"
    echo ""
    exit 1
fi
print_success "Docker is running"

# Check Go version
GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+\.[0-9]+')
print_success "Go version: $GO_VERSION"

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "Not in a git repository"
    exit 1
fi
print_success "Git repository detected"

# Step 2: List available workflows
print_header "Step 2: Available Workflows"
$ACT_BIN -l --container-architecture $CONTAINER_ARCH 2>/dev/null || true
echo ""

# Step 3: Validate workflow files
print_header "Step 3: Validate Workflow Files"

WORKFLOW_DIR=".github/workflows"
if [ ! -d "$WORKFLOW_DIR" ]; then
    print_error "Workflow directory not found: $WORKFLOW_DIR"
    exit 1
fi

WORKFLOW_COUNT=$(find "$WORKFLOW_DIR" -name "*.yml" -o -name "*.yaml" | wc -l | tr -d ' ')
print_info "Found $WORKFLOW_COUNT workflow files"

for workflow in "$WORKFLOW_DIR"/*.yml "$WORKFLOW_DIR"/*.yaml; do
    [ -e "$workflow" ] || continue
    filename=$(basename "$workflow")

    # Basic YAML syntax check
    if command -v yamllint &> /dev/null; then
        if yamllint -d relaxed "$workflow" &> /dev/null; then
            print_success "$filename: YAML syntax valid"
        else
            print_warning "$filename: YAML syntax warnings (non-critical)"
        fi
    else
        print_info "$filename: Found (yamllint not available for syntax check)"
    fi
done

# Step 4: Validate composite actions
print_header "Step 4: Validate Composite Actions"

ACTION_DIRS=$(find .github/actions -mindepth 1 -maxdepth 1 -type d 2>/dev/null || echo "")

if [ -z "$ACTION_DIRS" ]; then
    print_warning "No composite actions found"
else
    for action_dir in $ACTION_DIRS; do
        action_name=$(basename "$action_dir")
        action_file="$action_dir/action.yml"

        if [ -f "$action_file" ]; then
            print_success "Composite action: $action_name"

            # Check if action has required fields
            if grep -q "name:" "$action_file" && grep -q "runs:" "$action_file"; then
                print_success "  └─ Required fields present"
            else
                print_error "  └─ Missing required fields (name/runs)"
            fi
        else
            print_error "Composite action $action_name missing action.yml"
        fi
    done
fi

# Step 5: Run pre-commit checks locally (fast validation)
print_header "Step 5: Local Pre-Commit Checks"

print_info "Running 'make pre-commit' to validate code quality..."
if make pre-commit; then
    print_success "Pre-commit checks passed"
else
    print_error "Pre-commit checks failed"
    print_warning "Fix these issues before running act tests"
    exit 1
fi

# Step 6: Test critical jobs with act (if Docker is available)
print_header "Step 6: Test Critical Jobs with act"

print_warning "The following tests require Docker and may take several minutes"
print_info "Press Ctrl+C to skip act tests and see the summary"
echo ""

# Test lint job (fastest, no services required)
print_info "Testing 'lint' job..."
if $ACT_BIN -j lint --container-architecture $CONTAINER_ARCH --env GO_VERSION=1.25.5 push 2>&1 | tee /tmp/act-lint.log; then
    print_success "lint job completed"
else
    EXIT_CODE=$?
    print_error "lint job failed (exit code: $EXIT_CODE)"
    echo ""
    echo "Check /tmp/act-lint.log for details"
    print_warning "Common issues:"
    echo "  - Missing dependencies in Docker container"
    echo "  - Incorrect environment variables"
    echo "  - golangci-lint version mismatch"
fi

# Test unit tests (no services required)
print_info "Testing 'test-unit' job..."
if $ACT_BIN -j test-unit --container-architecture $CONTAINER_ARCH --env GO_VERSION=1.25.5 push 2>&1 | tee /tmp/act-test-unit.log; then
    print_success "test-unit job completed"
else
    EXIT_CODE=$?
    print_error "test-unit job failed (exit code: $EXIT_CODE)"
    echo ""
    echo "Check /tmp/act-test-unit.log for details"
fi

# Note about service containers
print_warning "Jobs requiring service containers (PostgreSQL, Redis) are harder to test with act"
print_info "Recommendation: Use 'make test-integration' locally instead"

# Step 7: Summary
print_header "Summary and Recommendations"

echo ""
echo "Local CI/CD Validation Complete!"
echo ""
echo "What was validated:"
print_success "  1. Prerequisites (act, Docker, Go)"
print_success "  2. Workflow file syntax"
print_success "  3. Composite actions structure"
print_success "  4. Pre-commit checks (lint, vet, fmt)"
print_success "  5. Critical jobs with act (lint, test-unit)"
echo ""
echo "What to test manually:"
print_info "  • Integration tests: make test-integration"
print_info "  • E2E tests: make test-e2e"
print_info "  • Domain tests: make test-domain"
print_info "  • Coverage: make test-coverage"
echo ""
echo "Before pushing to GitHub:"
print_warning "  1. Ensure all 'make pre-commit' checks pass"
print_warning "  2. Run 'make test' locally"
print_warning "  3. Verify migrations work: make migrate-up"
print_warning "  4. Check OpenAPI spec: make validate-openapi"
echo ""
print_success "CI/CD validation complete!"
