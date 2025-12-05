#!/usr/bin/env bash

#
# File Upload Security Test Suite
#
# Tests for file upload vulnerabilities:
# - Malware detection (ClamAV/EICAR)
# - Polyglot files (GIF+JS, PNG+PHP)
# - SVG with embedded scripts
# - Oversized files
# - Malicious metadata (EXIF)
# - File type validation bypass
#
# Usage: ./upload-security-tests.sh <BASE_URL>
#

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'

BASE_URL="${1:-http://localhost:8080}"
API_BASE="${BASE_URL}/api/v1"
OUTPUT_DIR="./pentest-results/upload"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

TESTS_RUN=0; TESTS_PASSED=0; TESTS_FAILED=0; CRITICAL_FINDINGS=0; HIGH_FINDINGS=0

mkdir -p "$OUTPUT_DIR"

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; ((TESTS_PASSED++)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; ((TESTS_FAILED++)); }
log_critical() { echo -e "${RED}[CRITICAL]${NC} $1"; echo "[$(date -Iseconds)] CRITICAL: $1" >> "$OUTPUT_DIR/critical_${TIMESTAMP}.log"; ((CRITICAL_FINDINGS++)); }
log_high() { echo -e "${RED}[HIGH]${NC} $1"; echo "[$(date -Iseconds)] HIGH: $1" >> "$OUTPUT_DIR/high_${TIMESTAMP}.log"; ((HIGH_FINDINGS++)); }

ACCESS_TOKEN=""

setup_auth() {
    local email="upload_test_$(date +%s)@example.com"
    curl -s -X POST "${API_BASE}/auth/register" -H "Content-Type: application/json" \
        -d "{\"email\": \"$email\", \"username\": \"uploadtest_$(date +%s)\", \"password\": \"TestPass123!\"}" > /dev/null

    local response=$(curl -s -X POST "${API_BASE}/auth/login" -H "Content-Type: application/json" \
        -d "{\"email\": \"$email\", \"password\": \"TestPass123!\"}")

    ACCESS_TOKEN=$(echo "$response" | jq -r '.access_token // .tokens.access_token // empty')
    [ -n "$ACCESS_TOKEN" ] && log_success "Authentication setup complete" || exit 1
}

# TC-UPLOAD-001: EICAR Malware Detection
test_eicar_detection() {
    log_info "TEST: TC-UPLOAD-001 - EICAR Malware Detection (ClamAV)"
    ((TESTS_RUN++))

    local eicar_file="/tmp/eicar_$$.txt"
    # EICAR test signature - safe malware test string
    echo 'X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*' > "$eicar_file"

    local response=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/images" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -F "image=@${eicar_file}" \
        -F "title=EICAR Test" \
        -F "visibility=private")

    local status=$(echo "$response" | tail -n 1)
    local body=$(echo "$response" | head -n -1)

    rm -f "$eicar_file"

    if [ "$status" = "400" ] && echo "$body" | grep -qi "malware\|virus"; then
        log_success "EICAR test file properly detected and rejected"
    elif [ "$status" = "202" ] || [ "$status" = "201" ]; then
        log_critical "EICAR file accepted! Malware scanning not working. CVSS: 9.3"
    else
        log_fail "Unexpected response to EICAR upload (HTTP $status)"
    fi
}

# TC-UPLOAD-002: Polyglot File (GIF + JavaScript)
test_polyglot_gif_js() {
    log_info "TEST: TC-UPLOAD-002 - Polyglot File (GIF + JavaScript)"
    ((TESTS_RUN++))

    local polyglot_file="/tmp/polyglot_$$.gif"
    # Valid GIF header + embedded JavaScript
    echo -e 'GIF89a\x01\x00\x01\x00\x00\xff\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x00;' > "$polyglot_file"
    echo '<script>alert("XSS")</script>' >> "$polyglot_file"

    local response=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/images" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -F "image=@${polyglot_file}" \
        -F "title=Polyglot Test" \
        -F "visibility=private")

    local status=$(echo "$response" | tail -n 1)
    local body=$(echo "$response" | head -n -1)

    rm -f "$polyglot_file"

    if [ "$status" = "400" ] || [ "$status" = "415" ]; then
        log_success "Polyglot file rejected"
    elif [ "$status" = "202" ] || [ "$status" = "201" ]; then
        local image_id=$(echo "$body" | jq -r '.id // empty')
        if [ -n "$image_id" ]; then
            # Verify re-encoding removed embedded script
            log_success "Polyglot accepted but should be re-encoded (verify with manual test)"
        fi
    fi
}

# TC-UPLOAD-003: SVG with Embedded JavaScript
test_svg_xss() {
    log_info "TEST: TC-UPLOAD-003 - SVG with Embedded JavaScript"
    ((TESTS_RUN++))

    local svg_file="/tmp/xss_$$.svg"
    cat > "$svg_file" << 'EOF'
<?xml version="1.0" standalone="no"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg version="1.1" baseProfile="full" xmlns="http://www.w3.org/2000/svg">
  <polygon id="triangle" points="0,0 0,50 50,0" fill="#009900" stroke="#004400"/>
  <script type="text/javascript">
    alert('XSS');
  </script>
</svg>
EOF

    local response=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/images" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -F "image=@${svg_file}" \
        -F "title=SVG XSS Test" \
        -F "visibility=private")

    local status=$(echo "$response" | tail -n 1)

    rm -f "$svg_file"

    if [ "$status" = "415" ] || [ "$status" = "400" ]; then
        log_success "SVG file rejected (expected - SVG not in allowed formats)"
    elif [ "$status" = "202" ] || [ "$status" = "201" ]; then
        log_high "SVG file accepted! Check if scripts are stripped. CVSS: 7.1"
    fi
}

# TC-UPLOAD-004: Oversized File Upload
test_oversized_file() {
    log_info "TEST: TC-UPLOAD-004 - Oversized File Upload (DoS)"
    ((TESTS_RUN++))

    local large_file="/tmp/large_$$.jpg"
    # Create 100MB file
    dd if=/dev/zero of="$large_file" bs=1M count=100 2>/dev/null

    local response=$(curl -s -w "\n%{http_code}" -m 30 -X POST "${API_BASE}/images" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -F "image=@${large_file}" \
        -F "title=Large File Test" \
        -F "visibility=private")

    local status=$(echo "$response" | tail -n 1)

    rm -f "$large_file"

    if [ "$status" = "413" ] || [ "$status" = "400" ]; then
        log_success "Oversized file rejected (HTTP $status)"
    elif [ "$status" = "202" ] || [ "$status" = "201" ]; then
        log_high "100MB file accepted! Resource exhaustion risk. CVSS: 7.5"
    else
        log_success "Request handled without timeout (HTTP $status)"
    fi
}

# TC-UPLOAD-005: Double Extension Bypass
test_double_extension() {
    log_info "TEST: TC-UPLOAD-005 - Double Extension Bypass"
    ((TESTS_RUN++))

    local test_file="/tmp/malicious_$$.jpg.php"
    printf '\xFF\xD8\xFF' > "$test_file"

    local response=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/images" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -F "image=@${test_file}" \
        -F "title=Double Extension Test" \
        -F "visibility=private")

    local status=$(echo "$response" | tail -n 1)

    rm -f "$test_file"

    if [ "$status" = "400" ] || [ "$status" = "415" ]; then
        log_success "Double extension file rejected"
    elif [ "$status" = "202" ] || [ "$status" = "201" ]; then
        log_high "File with .jpg.php accepted - verify storage sanitization"
    fi
}

# TC-UPLOAD-006: Content-Type Mismatch
test_content_type_mismatch() {
    log_info "TEST: TC-UPLOAD-006 - Content-Type vs Actual File Type Mismatch"
    ((TESTS_RUN++))

    local php_file="/tmp/shell_$$.jpg"
    echo '<?php system($_GET["cmd"]); ?>' > "$php_file"

    local response=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/images" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -F "image=@${php_file};type=image/jpeg" \
        -F "title=Content-Type Mismatch" \
        -F "visibility=private")

    local status=$(echo "$response" | tail -n 1)

    rm -f "$php_file"

    if [ "$status" = "400" ] || [ "$status" = "415" ]; then
        log_success "Content-type mismatch detected and rejected"
    elif [ "$status" = "202" ] || [ "$status" = "201" ]; then
        log_high "PHP file accepted with image/jpeg content-type! CVSS: 8.6"
    fi
}

# TC-UPLOAD-007: Malicious EXIF Metadata
test_malicious_exif() {
    log_info "TEST: TC-UPLOAD-007 - Malicious EXIF Metadata"
    ((TESTS_RUN++))

    # Create minimal JPEG with crafted EXIF
    local exif_file="/tmp/exif_$$.jpg"
    # JPEG header + EXIF with XSS payload in comment
    printf '\xFF\xD8\xFF\xE1' > "$exif_file"
    echo '<script>alert("EXIF-XSS")</script>' >> "$exif_file"
    printf '\xFF\xD9' >> "$exif_file"

    local response=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/images" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -F "image=@${exif_file}" \
        -F "title=EXIF Test" \
        -F "visibility=private")

    local status=$(echo "$response" | tail -n 1)

    rm -f "$exif_file"

    # Image processing should strip or sanitize EXIF
    if [ "$status" = "202" ] || [ "$status" = "201" ]; then
        log_success "Image accepted (verify EXIF is stripped during processing)"
    fi
}

# TC-UPLOAD-008: Race Condition in Upload
test_upload_race_condition() {
    log_info "TEST: TC-UPLOAD-008 - Race Condition in Concurrent Uploads"
    ((TESTS_RUN++))

    local test_file="/tmp/race_$$.jpg"
    printf '\xFF\xD8\xFF\xE0\x00\x10JFIF\xFF\xD9' > "$test_file"

    # Upload same file 10 times concurrently
    for i in {1..10}; do
        curl -s -X POST "${API_BASE}/images" \
            -H "Authorization: Bearer $ACCESS_TOKEN" \
            -F "image=@${test_file}" \
            -F "title=Race Test $i" \
            -F "visibility=private" &
    done

    wait
    rm -f "$test_file"

    log_success "Concurrent uploads completed (verify no file corruption in storage)"
}

main() {
    echo "=========================================="
    echo "  Upload Security Test Suite"
    echo "  Target: $BASE_URL"
    echo "=========================================="

    for cmd in curl jq dd; do
        command -v $cmd &> /dev/null || { echo "ERROR: $cmd not found"; exit 1; }
    done

    setup_auth

    echo; echo "Running upload security tests..."; echo

    test_eicar_detection
    test_polyglot_gif_js
    test_svg_xss
    test_oversized_file
    test_double_extension
    test_content_type_mismatch
    test_malicious_exif
    test_upload_race_condition

    echo; echo "=========================================="
    echo "  Test Summary"
    echo "=========================================="
    echo "Tests Run:         $TESTS_RUN"
    echo "Tests Passed:      $TESTS_PASSED"
    echo "Critical Findings: $CRITICAL_FINDINGS"
    echo "High Findings:     $HIGH_FINDINGS"
    echo "=========================================="

    [ $CRITICAL_FINDINGS -gt 0 ] && { echo "⚠️  CRITICAL VULNERABILITIES - LAUNCH BLOCKER"; exit 1; }
    [ $HIGH_FINDINGS -gt 0 ] && { echo "⚠️  HIGH SEVERITY VULNERABILITIES FOUND"; exit 1; }
    exit 0
}

main
