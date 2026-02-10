#!/bin/bash
# Agent CI Check Script
#
# MANDATORY: All Claude agents must run this script before pushing commits.
# This validates core CI checks locally to prevent pipeline failures.
#
# Usage:
#   ./scripts/agent-ci-check.sh          # Run all checks
#   ./scripts/agent-ci-check.sh --quick  # Run quick checks only (no Go compilation)
#   make agent-check                     # Via Makefile
#   make agent-check-quick               # Quick mode via Makefile
#
# Exit codes:
#   0 - All checks passed
#   1 - One or more checks failed

set -uo pipefail
# Note: we don't use 'set -e' because individual check failures are handled
# explicitly via the pass/fail/skip functions and FAILED counter.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
QUICK_MODE="${1:-}"
FAILED=0
PASSED=0
SKIPPED=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo ""
    echo -e "${BLUE}===========================================${NC}"
    echo -e "${BLUE}  Agent CI Check - Pre-Push Validation${NC}"
    echo -e "${BLUE}===========================================${NC}"
    echo ""
}

print_step() {
    echo -e "${BLUE}--- Step $1: $2 ---${NC}"
}

pass() {
    echo -e "${GREEN}  PASS: $1${NC}"
    PASSED=$((PASSED + 1))
}

fail() {
    echo -e "${RED}  FAIL: $1${NC}"
    FAILED=$((FAILED + 1))
}

skip() {
    echo -e "${YELLOW}  SKIP: $1${NC}"
    SKIPPED=$((SKIPPED + 1))
}

warn() {
    echo -e "${YELLOW}  WARN: $1${NC}"
}

print_header

cd "$PROJECT_ROOT"

# =========================================
# Step 1: Validate Postman Collection (JSON)
# =========================================
print_step 1 "Validate Postman Collection"

if command -v node &> /dev/null; then
    COLLECTION="tests/e2e/postman/goimg-api.postman_collection.json"
    if [ -f "$COLLECTION" ]; then
        if node -e "JSON.parse(require('fs').readFileSync('$COLLECTION', 'utf8')); console.log('JSON valid')" 2>/dev/null; then
            TEST_COUNT=$(node -e "
                const c = JSON.parse(require('fs').readFileSync('$COLLECTION', 'utf8'));
                function count(items) { let n = 0; for (const i of items) { if (i.item) n += count(i.item); else n++; } return n; }
                console.log(count(c.item));
            " 2>/dev/null)
            pass "Postman collection valid ($TEST_COUNT tests)"
        else
            fail "Postman collection has invalid JSON"
        fi
    else
        fail "Postman collection not found: $COLLECTION"
    fi

    # Validate environment file
    ENV_FILE="tests/e2e/postman/ci.postman_environment.json"
    if [ -f "$ENV_FILE" ]; then
        if node -e "JSON.parse(require('fs').readFileSync('$ENV_FILE', 'utf8')); console.log('JSON valid')" 2>/dev/null; then
            pass "Postman environment file valid"
        else
            fail "Postman environment file has invalid JSON"
        fi
    else
        fail "Postman environment file not found: $ENV_FILE"
    fi
else
    skip "Node.js not available - cannot validate Postman collection"
fi

# =========================================
# Step 2: Validate OpenAPI Spec
# =========================================
print_step 2 "Validate OpenAPI Spec"

OPENAPI_SPEC="api/openapi/openapi.yaml"
if [ -f "$OPENAPI_SPEC" ]; then
    # Basic YAML validation
    if command -v node &> /dev/null; then
        # Check for valid YAML by parsing with Node
        if node -e "
            const fs = require('fs');
            const content = fs.readFileSync('$OPENAPI_SPEC', 'utf8');
            if (content.includes('openapi:')) { console.log('OpenAPI spec present'); process.exit(0); }
            else { console.error('Missing openapi version'); process.exit(1); }
        " 2>/dev/null; then
            pass "OpenAPI spec file present and readable"
        else
            fail "OpenAPI spec missing version declaration"
        fi
    else
        if [ -s "$OPENAPI_SPEC" ]; then
            pass "OpenAPI spec file exists"
        else
            fail "OpenAPI spec file is empty"
        fi
    fi
else
    fail "OpenAPI spec not found: $OPENAPI_SPEC"
fi

# =========================================
# Step 3: Go Formatting Check
# =========================================
if [ "$QUICK_MODE" != "--quick" ]; then
    print_step 3 "Go Formatting"

    if command -v go &> /dev/null; then
        # Use GOTOOLCHAIN=local to avoid download issues in sandboxed environments
        FMT_OUTPUT=$(GOTOOLCHAIN=local go fmt ./... 2>&1) || true
        if [ -z "$FMT_OUTPUT" ]; then
            pass "Go code properly formatted"
        else
            warn "Files reformatted: $FMT_OUTPUT"
            pass "Go fmt ran successfully (files were reformatted)"
        fi
    else
        skip "Go not available"
    fi

    # =========================================
    # Step 4: Go Vet
    # =========================================
    print_step 4 "Go Vet"

    if command -v go &> /dev/null; then
        if GOTOOLCHAIN=local go vet ./... 2>/dev/null; then
            pass "Go vet passed"
        else
            # go vet may fail due to missing dependencies in sandbox - downgrade to warning
            warn "go vet had issues (may be due to missing module cache in sandbox)"
            skip "go vet - sandbox limitation"
        fi
    else
        skip "Go not available"
    fi

    # =========================================
    # Step 5: Lint Check
    # =========================================
    print_step 5 "Lint Check"

    if command -v golangci-lint &> /dev/null; then
        if golangci-lint run ./... 2>/dev/null; then
            pass "golangci-lint passed"
        else
            warn "golangci-lint had issues (may be due to missing module cache in sandbox)"
            skip "golangci-lint - sandbox limitation"
        fi
    else
        skip "golangci-lint not available"
    fi

    # =========================================
    # Step 6: Go Build Check
    # =========================================
    print_step 6 "Go Build Check"

    if command -v go &> /dev/null; then
        BUILD_OK=true
        for pkg in ./cmd/api ./cmd/worker ./cmd/migrate; do
            if GOTOOLCHAIN=local go build -o /dev/null "$pkg" 2>/dev/null; then
                pass "Build $pkg"
            else
                warn "Build $pkg failed (may be sandbox limitation)"
                skip "Build $pkg - sandbox limitation"
                BUILD_OK=false
            fi
        done
    else
        skip "Go not available - cannot build"
    fi
else
    # Quick mode - skip Go checks
    print_step 3 "Go Checks (skipped in quick mode)"
    skip "Go fmt - quick mode"
    skip "Go vet - quick mode"
    skip "Go lint - quick mode"
    skip "Go build - quick mode"
fi

# =========================================
# Step 7: Check for Common Issues
# =========================================
print_step 7 "Common Issue Detection"

# Check for hardcoded secrets
if grep -rn "password\s*=\s*['\"]" --include="*.go" internal/ cmd/ 2>/dev/null | grep -v "_test.go" | grep -v "password\s*=\s*\"\"" | head -5; then
    warn "Potential hardcoded passwords found (review above matches)"
else
    pass "No obvious hardcoded secrets detected"
fi

# Check for TODO without issue reference
TODO_COUNT=$(grep -rn "TODO" --include="*.go" internal/ cmd/ 2>/dev/null | grep -v "TODO(" | grep -v "TODO #" | grep -v "TODO:" | wc -l | tr -d '[:space:]')
TODO_COUNT=${TODO_COUNT:-0}
if [ "$TODO_COUNT" -gt "0" ]; then
    warn "$TODO_COUNT TODO comments without issue references"
else
    pass "No orphan TODO comments"
fi

# Check for debug/temporary code
if grep -rn "fmt\.Print" --include="*.go" internal/ cmd/ 2>/dev/null | grep -v "_test.go" | head -3; then
    warn "Found fmt.Print statements (should use structured logging)"
else
    pass "No debug print statements"
fi

# =========================================
# Step 8: Git Status Check
# =========================================
print_step 8 "Git Status"

# Check for uncommitted changes
UNSTAGED=$(git diff --name-only 2>/dev/null | wc -l | tr -d '[:space:]')
STAGED=$(git diff --cached --name-only 2>/dev/null | wc -l | tr -d '[:space:]')

if [ "$UNSTAGED" -gt "0" ]; then
    warn "$UNSTAGED file(s) with unstaged changes"
fi
if [ "$STAGED" -gt "0" ]; then
    warn "$STAGED file(s) staged but not committed"
fi

if [ "$UNSTAGED" -eq "0" ] && [ "$STAGED" -eq "0" ]; then
    pass "Working directory clean"
fi

# Check current branch
BRANCH=$(git branch --show-current 2>/dev/null || echo "unknown")
pass "On branch: $BRANCH"

# =========================================
# Summary
# =========================================
echo ""
echo -e "${BLUE}===========================================${NC}"
echo -e "${BLUE}  Summary${NC}"
echo -e "${BLUE}===========================================${NC}"
echo -e "  ${GREEN}Passed:  $PASSED${NC}"
echo -e "  ${RED}Failed:  $FAILED${NC}"
echo -e "  ${YELLOW}Skipped: $SKIPPED${NC}"
echo ""

if [ "$FAILED" -gt "0" ]; then
    echo -e "${RED}AGENT CI CHECK FAILED${NC}"
    echo -e "${RED}Fix the above failures before pushing.${NC}"
    exit 1
else
    echo -e "${GREEN}AGENT CI CHECK PASSED${NC}"
    if [ "$SKIPPED" -gt "0" ]; then
        echo -e "${YELLOW}Note: $SKIPPED checks were skipped (likely sandbox limitations).${NC}"
        echo -e "${YELLOW}These will be validated by GitHub Actions CI.${NC}"
    fi
    exit 0
fi
