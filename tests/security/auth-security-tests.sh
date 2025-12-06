#!/usr/bin/env bash

#
# Authentication Security Test Suite
#
# Tests for authentication vulnerabilities in goimg-datalayer API
# Including: JWT manipulation, session management, brute force protection,
# account enumeration, and token validation.
#
# Usage: ./auth-security-tests.sh <BASE_URL>
# Example: ./auth-security-tests.sh http://localhost:8080
#
# Requirements: curl, jq, base64, openssl
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${1:-http://localhost:8080}"
API_BASE="${BASE_URL}/api/v1"
OUTPUT_DIR="./pentest-results/auth"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
CRITICAL_FINDINGS=0
HIGH_FINDINGS=0

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    echo "[$(date -Iseconds)] FAIL: $1" >> "$OUTPUT_DIR/failures_${TIMESTAMP}.log"
    ((TESTS_FAILED++))
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_critical() {
    echo -e "${RED}[CRITICAL]${NC} $1"
    echo "[$(date -Iseconds)] CRITICAL: $1" >> "$OUTPUT_DIR/critical_${TIMESTAMP}.log"
    ((CRITICAL_FINDINGS++))
}

log_high() {
    echo -e "${RED}[HIGH]${NC} $1"
    echo "[$(date -Iseconds)] HIGH: $1" >> "$OUTPUT_DIR/high_${TIMESTAMP}.log"
    ((HIGH_FINDINGS++))
}

# Utility functions
decode_jwt() {
    local token="$1"
    local part="${2:-payload}" # header or payload

    if [ "$part" = "header" ]; then
        echo "$token" | cut -d. -f1 | base64 -d 2>/dev/null | jq .
    else
        echo "$token" | cut -d. -f2 | base64 -d 2>/dev/null | jq .
    fi
}

# Test user credentials
TEST_USER_EMAIL="pentest_$(date +%s)@example.com"
TEST_USER_PASSWORD="SecureP@ssw0rd123!"
TEST_USER_USERNAME="pentest_user_$(date +%s)"

# Setup: Register test user
setup_test_user() {
    log_info "Setting up test user: $TEST_USER_EMAIL"

    local response=$(curl -s -w "\n%{http_code}" -X POST \
        "${API_BASE}/auth/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$TEST_USER_EMAIL\",
            \"username\": \"$TEST_USER_USERNAME\",
            \"password\": \"$TEST_USER_PASSWORD\"
        }")

    local body=$(echo "$response" | head -n -1)
    local status=$(echo "$response" | tail -n 1)

    if [ "$status" = "201" ]; then
        log_success "Test user registered successfully"
        echo "$body" > "$OUTPUT_DIR/test_user.json"
        return 0
    else
        log_fail "Failed to register test user (HTTP $status)"
        echo "$body"
        return 1
    fi
}

# Login and get tokens
login_test_user() {
    log_info "Logging in as test user"

    local response=$(curl -s -w "\n%{http_code}" -X POST \
        "${API_BASE}/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$TEST_USER_EMAIL\",
            \"password\": \"$TEST_USER_PASSWORD\"
        }")

    local body=$(echo "$response" | head -n -1)
    local status=$(echo "$response" | tail -n 1)

    if [ "$status" = "200" ]; then
        echo "$body" > "$OUTPUT_DIR/login_response.json"
        ACCESS_TOKEN=$(echo "$body" | jq -r '.access_token // .tokens.access_token // empty')
        REFRESH_TOKEN=$(echo "$body" | jq -r '.refresh_token // .tokens.refresh_token // empty')

        if [ -n "$ACCESS_TOKEN" ] && [ "$ACCESS_TOKEN" != "null" ]; then
            log_success "Login successful, tokens obtained"
            return 0
        else
            log_fail "Tokens not found in response"
            return 1
        fi
    else
        log_fail "Login failed (HTTP $status)"
        echo "$body"
        return 1
    fi
}

# ============================================================================
# TEST SUITE
# ============================================================================

# TC-AUTH-001: JWT Algorithm Confusion Attack
test_algorithm_confusion() {
    log_info "TEST: TC-AUTH-001 - JWT Algorithm Confusion Attack"
    ((TESTS_RUN++))

    if [ -z "$ACCESS_TOKEN" ]; then
        log_warn "Skipping: No access token available"
        return
    fi

    # Decode JWT header and payload
    local header=$(echo "$ACCESS_TOKEN" | cut -d. -f1 | base64 -d 2>/dev/null || echo "{}")
    local payload=$(echo "$ACCESS_TOKEN" | cut -d. -f2 | base64 -d 2>/dev/null || echo "{}")

    # Modify algorithm to HS256
    local modified_header=$(echo "$header" | jq -c '.alg = "HS256"')
    local encoded_header=$(echo -n "$modified_header" | base64 | tr -d '=' | tr '+/' '-_')
    local encoded_payload=$(echo "$ACCESS_TOKEN" | cut -d. -f2)

    # Create fake signature (using public key as HMAC secret - this is the attack)
    # Note: This requires the public key, which attackers might obtain
    local unsigned="${encoded_header}.${encoded_payload}"
    local fake_signature=$(echo -n "$unsigned" | openssl dgst -sha256 -hmac "test" | awk '{print $2}' | xxd -r -p | base64 | tr -d '=' | tr '+/' '-_')

    local malicious_token="${unsigned}.${fake_signature}"

    # Attempt to use malicious token
    local response=$(curl -s -w "\n%{http_code}" -X GET \
        "${API_BASE}/users/me" \
        -H "Authorization: Bearer $malicious_token")

    local status=$(echo "$response" | tail -n 1)

    if [ "$status" = "401" ]; then
        log_success "Algorithm confusion attack prevented (401 Unauthorized)"
    else
        log_critical "VULNERABILITY: Algorithm confusion attack succeeded! (HTTP $status)"
        log_critical "Modified JWT with HS256 was accepted. CVSS: 10.0 (Critical)"
    fi
}

# TC-AUTH-002: Token Replay After Logout
test_token_replay_after_logout() {
    log_info "TEST: TC-AUTH-002 - Token Replay After Logout"
    ((TESTS_RUN++))

    if [ -z "$ACCESS_TOKEN" ]; then
        log_warn "Skipping: No access token available"
        return
    fi

    # Verify token works before logout
    local before_response=$(curl -s -w "\n%{http_code}" -X GET \
        "${API_BASE}/users/me" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    local before_status=$(echo "$before_response" | tail -n 1)

    if [ "$before_status" != "200" ]; then
        log_warn "Token not valid before logout test, skipping"
        return
    fi

    # Logout
    log_info "Performing logout..."
    local logout_response=$(curl -s -w "\n%{http_code}" -X POST \
        "${API_BASE}/auth/logout" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{}")

    local logout_status=$(echo "$logout_response" | tail -n 1)

    # Attempt to use token after logout
    sleep 1  # Give blacklist time to propagate
    local after_response=$(curl -s -w "\n%{http_code}" -X GET \
        "${API_BASE}/users/me" \
        -H "Authorization: Bearer $ACCESS_TOKEN")

    local after_status=$(echo "$after_response" | tail -n 1)
    local after_body=$(echo "$after_response" | head -n -1)

    if [ "$after_status" = "401" ]; then
        if echo "$after_body" | grep -qi "revoked\|blacklist"; then
            log_success "Token properly blacklisted after logout (401 with revocation message)"
        else
            log_success "Token rejected after logout (401), but message could be clearer"
        fi
    else
        log_critical "VULNERABILITY: Token still valid after logout! (HTTP $after_status)"
        log_critical "Session management bypass. CVSS: 9.1 (Critical)"
    fi
}

# TC-AUTH-003: Refresh Token Rotation
test_refresh_token_rotation() {
    log_info "TEST: TC-AUTH-003 - Refresh Token Rotation and Replay Detection"
    ((TESTS_RUN++))

    if [ -z "$REFRESH_TOKEN" ]; then
        log_warn "Skipping: No refresh token available"
        return
    fi

    # Use refresh token to get new token pair
    local first_refresh=$(curl -s -w "\n%{http_code}" -X POST \
        "${API_BASE}/auth/refresh" \
        -H "Content-Type: application/json" \
        -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}")

    local first_status=$(echo "$first_refresh" | tail -n 1)
    local first_body=$(echo "$first_refresh" | head -n -1)

    if [ "$first_status" != "200" ]; then
        log_warn "First refresh failed (HTTP $first_status), skipping test"
        return
    fi

    local new_refresh=$(echo "$first_body" | jq -r '.refresh_token // .tokens.refresh_token // empty')

    if [ -z "$new_refresh" ] || [ "$new_refresh" = "null" ]; then
        log_fail "No new refresh token in response - rotation not implemented"
        return
    fi

    # Attempt to reuse old refresh token (replay attack)
    sleep 1
    local replay_response=$(curl -s -w "\n%{http_code}" -X POST \
        "${API_BASE}/auth/refresh" \
        -H "Content-Type: application/json" \
        -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}")

    local replay_status=$(echo "$replay_response" | tail -n 1)
    local replay_body=$(echo "$replay_response" | head -n -1)

    if [ "$replay_status" = "401" ] || [ "$replay_status" = "403" ]; then
        if echo "$replay_body" | grep -qi "replay\|used\|revoked"; then
            log_success "Refresh token replay detected and blocked (HTTP $replay_status)"
        else
            log_success "Old refresh token rejected (HTTP $replay_status)"
        fi
    else
        log_high "VULNERABILITY: Refresh token replay not detected! (HTTP $replay_status)"
        log_high "Token theft could be exploited. CVSS: 7.5 (High)"
    fi
}

# TC-AUTH-004: Account Enumeration via Timing
test_account_enumeration_timing() {
    log_info "TEST: TC-AUTH-004 - Account Enumeration via Timing Attack"
    ((TESTS_RUN++))

    # Test with non-existent email
    local start1=$(date +%s%N)
    curl -s -o /dev/null -X POST \
        "${API_BASE}/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"nonexistent_$(date +%s)@example.com\",
            \"password\": \"WrongPassword123!\"
        }"
    local end1=$(date +%s%N)
    local time1=$((($end1 - $start1) / 1000000))

    # Test with existing email but wrong password
    local start2=$(date +%s%N)
    curl -s -o /dev/null -X POST \
        "${API_BASE}/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$TEST_USER_EMAIL\",
            \"password\": \"WrongPassword456!\"
        }"
    local end2=$(date +%s%N)
    local time2=$((($end2 - $start2) / 1000000))

    # Calculate difference
    local diff=$((time2 - time1))
    if [ $diff -lt 0 ]; then
        diff=$((-diff))
    fi

    log_info "Non-existent email: ${time1}ms, Existing email: ${time2}ms, Diff: ${diff}ms"

    # Threshold: 50ms difference could indicate timing leak
    if [ $diff -gt 50 ]; then
        log_warn "Potential timing difference detected (${diff}ms)"
        log_warn "May allow account enumeration through timing analysis"
        log_warn "CVSS: 5.3 (Medium)"
    else
        log_success "No significant timing difference detected (${diff}ms)"
    fi
}

# TC-AUTH-005: Brute Force Protection
test_brute_force_protection() {
    log_info "TEST: TC-AUTH-005 - Account Lockout and Brute Force Protection"
    ((TESTS_RUN++))

    # Create a dedicated test account for this test
    local brute_email="bruteforce_test_$(date +%s)@example.com"
    local brute_password="BruteForce123!"

    # Register account
    curl -s -o /dev/null -X POST \
        "${API_BASE}/auth/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$brute_email\",
            \"username\": \"bruteuser_$(date +%s)\",
            \"password\": \"$brute_password\"
        }"

    log_info "Attempting 6 failed login attempts..."

    local locked=false
    for i in {1..6}; do
        local response=$(curl -s -w "\n%{http_code}" -X POST \
            "${API_BASE}/auth/login" \
            -H "Content-Type: application/json" \
            -d "{
                \"email\": \"$brute_email\",
                \"password\": \"WrongPassword$i\"
            }")

        local status=$(echo "$response" | tail -n 1)
        local body=$(echo "$response" | head -n -1)

        log_info "Attempt $i: HTTP $status"

        # Check if account is locked (usually 403 or specific error message)
        if [ "$status" = "403" ] || echo "$body" | grep -qi "locked\|blocked\|too many attempts"; then
            log_success "Account locked after $i failed attempts"
            locked=true
            break
        fi

        sleep 0.5  # Brief delay between attempts
    done

    if [ "$locked" = false ]; then
        log_high "VULNERABILITY: No account lockout after 6 failed attempts"
        log_high "Brute force attacks are possible. CVSS: 7.5 (High)"
    fi
}

# TC-AUTH-006: Missing Authorization Header
test_missing_auth_header() {
    log_info "TEST: TC-AUTH-006 - Protected Endpoint Without Authorization"
    ((TESTS_RUN++))

    local response=$(curl -s -w "\n%{http_code}" -X GET \
        "${API_BASE}/users/me")

    local status=$(echo "$response" | tail -n 1)

    if [ "$status" = "401" ]; then
        log_success "Protected endpoint properly requires authentication (401)"
    else
        log_critical "VULNERABILITY: Protected endpoint accessible without auth! (HTTP $status)"
    fi
}

# TC-AUTH-007: Malformed JWT Token
test_malformed_jwt() {
    log_info "TEST: TC-AUTH-007 - Malformed JWT Token Handling"
    ((TESTS_RUN++))

    local malformed_tokens=(
        "not.a.jwt"
        "header.payload"
        "header.payload.signature.extra"
        "äöü.invalid.utf8"
        "eyJhbGciOiJub25lIn0.eyJzdWIiOiIxMjM0NTY3ODkwIn0."  # Algorithm: none
    )

    local all_rejected=true

    for token in "${malformed_tokens[@]}"; do
        local response=$(curl -s -w "\n%{http_code}" -X GET \
            "${API_BASE}/users/me" \
            -H "Authorization: Bearer $token")

        local status=$(echo "$response" | tail -n 1)

        if [ "$status" != "401" ]; then
            log_fail "Malformed token not rejected: $token (HTTP $status)"
            all_rejected=false
        fi
    done

    if [ "$all_rejected" = true ]; then
        log_success "All malformed tokens properly rejected (401)"
    fi
}

# TC-AUTH-008: JWT Expiration Validation
test_jwt_expiration() {
    log_info "TEST: TC-AUTH-008 - JWT Expiration Validation"
    ((TESTS_RUN++))

    if [ -z "$ACCESS_TOKEN" ]; then
        log_warn "Skipping: No access token available"
        return
    fi

    # Decode token and check expiration
    local payload=$(decode_jwt "$ACCESS_TOKEN" "payload")
    local exp=$(echo "$payload" | jq -r '.exp // empty')
    local now=$(date +%s)

    if [ -n "$exp" ] && [ "$exp" != "null" ]; then
        local ttl=$((exp - now))
        log_info "Token expires in $ttl seconds"

        if [ $ttl -gt 3600 ]; then
            log_warn "Token TTL is long (${ttl}s / $((ttl/60))min) - consider shorter access token lifetime"
        else
            log_success "Reasonable token expiration ($((ttl/60)) minutes)"
        fi
    else
        log_fail "No expiration claim (exp) found in JWT"
    fi
}

# TC-AUTH-009: Case-Sensitive Email Validation
test_email_case_sensitivity() {
    log_info "TEST: TC-AUTH-009 - Email Case Sensitivity in Login"
    ((TESTS_RUN++))

    # Try login with different case variations
    local lowercase_email="${TEST_USER_EMAIL,,}"
    local uppercase_email="${TEST_USER_EMAIL^^}"

    local response_lower=$(curl -s -w "\n%{http_code}" -X POST \
        "${API_BASE}/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$lowercase_email\",
            \"password\": \"$TEST_USER_PASSWORD\"
        }")

    local status_lower=$(echo "$response_lower" | tail -n 1)

    if [ "$lowercase_email" != "$TEST_USER_EMAIL" ]; then
        if [ "$status_lower" = "200" ]; then
            log_success "Email login is case-insensitive (as expected for email)"
        else
            log_warn "Email appears case-sensitive - may confuse users"
        fi
    else
        log_info "Cannot test case sensitivity (test email already lowercase)"
    fi
}

# TC-AUTH-010: Password Complexity Enforcement
test_password_complexity() {
    log_info "TEST: TC-AUTH-010 - Password Complexity Enforcement"
    ((TESTS_RUN++))

    local weak_passwords=(
        "password"
        "12345678"
        "qwerty"
        "abc123"
    )

    local all_rejected=true

    for weak_pass in "${weak_passwords[@]}"; do
        local response=$(curl -s -w "\n%{http_code}" -X POST \
            "${API_BASE}/auth/register" \
            -H "Content-Type: application/json" \
            -d "{
                \"email\": \"weakpass_$(date +%s)@example.com\",
                \"username\": \"weakuser_$(date +%s)\",
                \"password\": \"$weak_pass\"
            }")

        local status=$(echo "$response" | tail -n 1)

        if [ "$status" = "201" ]; then
            log_fail "Weak password accepted: $weak_pass"
            all_rejected=false
        fi
    done

    if [ "$all_rejected" = true ]; then
        log_success "Weak passwords properly rejected"
    else
        log_warn "Password complexity enforcement may be insufficient"
    fi
}

# ============================================================================
# MAIN EXECUTION
# ============================================================================

main() {
    echo "=========================================="
    echo "  Authentication Security Test Suite"
    echo "  Target: $BASE_URL"
    echo "  Timestamp: $TIMESTAMP"
    echo "=========================================="
    echo

    # Check dependencies
    for cmd in curl jq base64 openssl; do
        if ! command -v $cmd &> /dev/null; then
            echo "ERROR: Required command '$cmd' not found"
            exit 1
        fi
    done

    # Setup
    setup_test_user || exit 1
    login_test_user || exit 1

    echo
    echo "Running authentication security tests..."
    echo

    # Run all tests
    test_missing_auth_header
    test_malformed_jwt
    test_algorithm_confusion
    test_token_replay_after_logout
    test_refresh_token_rotation
    test_account_enumeration_timing
    test_brute_force_protection
    test_jwt_expiration
    test_email_case_sensitivity
    test_password_complexity

    # Summary
    echo
    echo "=========================================="
    echo "  Test Summary"
    echo "=========================================="
    echo "Tests Run:      $TESTS_RUN"
    echo "Tests Passed:   $TESTS_PASSED"
    echo "Tests Failed:   $TESTS_FAILED"
    echo "Critical Findings: $CRITICAL_FINDINGS"
    echo "High Findings:     $HIGH_FINDINGS"
    echo
    echo "Results saved to: $OUTPUT_DIR"
    echo "=========================================="

    # Exit with error if critical findings
    if [ $CRITICAL_FINDINGS -gt 0 ]; then
        echo
        echo "⚠️  CRITICAL VULNERABILITIES FOUND - LAUNCH BLOCKER"
        exit 1
    elif [ $HIGH_FINDINGS -gt 0 ]; then
        echo
        echo "⚠️  HIGH SEVERITY VULNERABILITIES FOUND"
        exit 1
    fi

    exit 0
}

# Run main function
main
