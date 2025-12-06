#!/usr/bin/env bash

#
# Access Control & Authorization Test Suite
#
# Tests for authorization vulnerabilities:
# - IDOR (Insecure Direct Object Reference)
# - Horizontal privilege escalation
# - Vertical privilege escalation
# - Missing function-level access control
# - Parameter tampering
#
# Usage: ./access-control-tests.sh <BASE_URL>
#

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'

BASE_URL="${1:-http://localhost:8080}"
API_BASE="${BASE_URL}/api/v1"
OUTPUT_DIR="./pentest-results/access-control"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

TESTS_RUN=0; TESTS_PASSED=0; TESTS_FAILED=0; CRITICAL_FINDINGS=0; HIGH_FINDINGS=0

mkdir -p "$OUTPUT_DIR"

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; ((TESTS_PASSED++)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; ((TESTS_FAILED++)); }
log_critical() { echo -e "${RED}[CRITICAL]${NC} $1"; echo "[$(date -Iseconds)] CRITICAL: $1" >> "$OUTPUT_DIR/critical_${TIMESTAMP}.log"; ((CRITICAL_FINDINGS++)); }
log_high() { echo -e "${RED}[HIGH]${NC} $1"; echo "[$(date -Iseconds)] HIGH: $1" >> "$OUTPUT_DIR/high_${TIMESTAMP}.log"; ((HIGH_FINDINGS++)); }

# Test users
USER_A_TOKEN=""
USER_A_ID=""
USER_A_IMAGE_ID=""

USER_B_TOKEN=""
USER_B_ID=""

create_user() {
    local username="$1"
    local email="accesstest_${username}_$(date +%s)@example.com"
    local password="TestPass123!"

    # Register
    curl -s -X POST "${API_BASE}/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$email\", \"username\": \"$username\", \"password\": \"$password\"}" > /dev/null

    # Login
    local response=$(curl -s -X POST "${API_BASE}/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$email\", \"password\": \"$password\"}")

    local token=$(echo "$response" | jq -r '.access_token // .tokens.access_token // empty')

    # Get user ID
    local user_info=$(curl -s -X GET "${API_BASE}/users/me" \
        -H "Authorization: Bearer $token")
    local user_id=$(echo "$user_info" | jq -r '.id // empty')

    echo "$token|$user_id"
}

upload_test_image() {
    local token="$1"
    local title="$2"
    local visibility="${3:-private}"

    local test_file="/tmp/test_image_$$.jpg"
    printf '\xFF\xD8\xFF\xE0\x00\x10JFIF\xFF\xD9' > "$test_file"

    local response=$(curl -s -X POST "${API_BASE}/images" \
        -H "Authorization: Bearer $token" \
        -F "image=@${test_file}" \
        -F "title=$title" \
        -F "visibility=$visibility")

    rm -f "$test_file"
    echo "$response" | jq -r '.id // empty'
}

setup_test_users() {
    log_info "Setting up test users A and B"

    # User A
    local user_a_data=$(create_user "user_a_$(date +%s)")
    USER_A_TOKEN=$(echo "$user_a_data" | cut -d'|' -f1)
    USER_A_ID=$(echo "$user_a_data" | cut -d'|' -f2)

    # User B
    local user_b_data=$(create_user "user_b_$(date +%s)")
    USER_B_TOKEN=$(echo "$user_b_data" | cut -d'|' -f1)
    USER_B_ID=$(echo "$user_b_data" | cut -d'|' -f2)

    # User A uploads a private image
    USER_A_IMAGE_ID=$(upload_test_image "$USER_A_TOKEN" "User A Private Image" "private")

    if [ -n "$USER_A_TOKEN" ] && [ -n "$USER_B_TOKEN" ]; then
        log_success "Test users created successfully"
    else
        log_fail "Failed to create test users"
        exit 1
    fi
}

# TC-AUTHZ-001: IDOR - Image Retrieval
test_idor_image_retrieval() {
    log_info "TEST: TC-AUTHZ-001 - IDOR on Private Image Retrieval"
    ((TESTS_RUN++))

    if [ -z "$USER_A_IMAGE_ID" ]; then
        log_info "Skipping: No image ID available"
        return
    fi

    # User B attempts to access User A's private image
    local response=$(curl -s -w "\n%{http_code}" -X GET \
        "${API_BASE}/images/${USER_A_IMAGE_ID}" \
        -H "Authorization: Bearer $USER_B_TOKEN")

    local status=$(echo "$response" | tail -n 1)
    local body=$(echo "$response" | head -n -1)

    if [ "$status" = "403" ] || [ "$status" = "404" ]; then
        log_success "IDOR prevented: User B cannot access User A's private image (HTTP $status)"
    else
        log_critical "IDOR VULNERABILITY: User B accessed User A's private image! (HTTP $status)"
        log_critical "Unauthorized data access. CVSS: 9.1 (Critical)"
        echo "$body" > "$OUTPUT_DIR/idor_evidence_image_${TIMESTAMP}.json"
    fi
}

# TC-AUTHZ-002: IDOR - Image Deletion
test_idor_image_deletion() {
    log_info "TEST: TC-AUTHZ-002 - IDOR on Image Deletion"
    ((TESTS_RUN++))

    # User A creates another image
    local delete_test_image=$(upload_test_image "$USER_A_TOKEN" "To Be Deleted Test" "private")

    if [ -z "$delete_test_image" ] || [ "$delete_test_image" = "null" ]; then
        log_info "Skipping: Failed to create test image"
        return
    fi

    # User B attempts to delete User A's image
    local response=$(curl -s -w "\n%{http_code}" -X DELETE \
        "${API_BASE}/images/${delete_test_image}" \
        -H "Authorization: Bearer $USER_B_TOKEN")

    local status=$(echo "$response" | tail -n 1)

    if [ "$status" = "403" ] || [ "$status" = "404" ]; then
        log_success "IDOR prevented: User B cannot delete User A's image (HTTP $status)"
    else
        log_critical "IDOR VULNERABILITY: User B deleted User A's image! (HTTP $status)"
        log_critical "Unauthorized data modification. CVSS: 9.1"
    fi
}

# TC-AUTHZ-003: IDOR - Image Update
test_idor_image_update() {
    log_info "TEST: TC-AUTHZ-003 - IDOR on Image Metadata Update"
    ((TESTS_RUN++))

    if [ -z "$USER_A_IMAGE_ID" ]; then
        log_info "Skipping: No image ID available"
        return
    fi

    # User B attempts to update User A's image title
    local response=$(curl -s -w "\n%{http_code}" -X PUT \
        "${API_BASE}/images/${USER_A_IMAGE_ID}" \
        -H "Authorization: Bearer $USER_B_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"title": "Hijacked by User B", "visibility": "public"}')

    local status=$(echo "$response" | tail -n 1)

    if [ "$status" = "403" ] || [ "$status" = "404" ]; then
        log_success "IDOR prevented: User B cannot update User A's image (HTTP $status)"
    else
        log_critical "IDOR VULNERABILITY: User B updated User A's image! (HTTP $status)"
        log_critical "CVSS: 8.1 (High)"
    fi
}

# TC-AUTHZ-004: Horizontal Privilege Escalation - User Profile
test_horizontal_privilege_escalation() {
    log_info "TEST: TC-AUTHZ-004 - Horizontal Privilege Escalation (User Profile)"
    ((TESTS_RUN++))

    # User B attempts to access/modify User A's profile
    local response=$(curl -s -w "\n%{http_code}" -X PATCH \
        "${API_BASE}/users/${USER_A_ID}" \
        -H "Authorization: Bearer $USER_B_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"email": "hijacked@example.com"}')

    local status=$(echo "$response" | tail -n 1)

    if [ "$status" = "403" ] || [ "$status" = "404" ] || [ "$status" = "401" ]; then
        log_success "Horizontal escalation prevented (HTTP $status)"
    else
        log_critical "HORIZONTAL PRIVILEGE ESCALATION: User B modified User A's profile! (HTTP $status)"
        log_critical "CVSS: 9.1"
    fi
}

# TC-AUTHZ-005: Vertical Privilege Escalation - Role Modification
test_vertical_privilege_escalation() {
    log_info "TEST: TC-AUTHZ-005 - Vertical Privilege Escalation (Role Modification)"
    ((TESTS_RUN++))

    # User A attempts to elevate own role to admin via parameter tampering
    local response=$(curl -s -w "\n%{http_code}" -X PATCH \
        "${API_BASE}/users/me" \
        -H "Authorization: Bearer $USER_A_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"role": "admin"}')

    local status=$(echo "$response" | tail -n 1)
    local body=$(echo "$response" | head -n -1)

    # Verify role did not change
    local user_info=$(curl -s -X GET "${API_BASE}/users/me" \
        -H "Authorization: Bearer $USER_A_TOKEN")
    local current_role=$(echo "$user_info" | jq -r '.role // empty')

    if [ "$current_role" = "admin" ]; then
        log_critical "VERTICAL PRIVILEGE ESCALATION: User elevated to admin! CVSS: 10.0"
    else
        log_success "Vertical escalation prevented: Role modification rejected"
    fi
}

# TC-AUTHZ-006: Missing Function-Level Access Control
test_missing_function_level_access() {
    log_info "TEST: TC-AUTHZ-006 - Missing Function-Level Access Control (Admin Endpoints)"
    ((TESTS_RUN++))

    # Regular user attempts to access admin endpoints (if they exist)
    local admin_endpoints=(
        "/api/v1/admin/users"
        "/api/v1/admin/images"
        "/api/v1/admin/stats"
    )

    local unauthorized_access=false

    for endpoint in "${admin_endpoints[@]}"; do
        local response=$(curl -s -w "\n%{http_code}" -X GET \
            "${BASE_URL}${endpoint}" \
            -H "Authorization: Bearer $USER_A_TOKEN")

        local status=$(echo "$response" | tail -n 1)

        if [ "$status" = "200" ]; then
            log_critical "Missing access control: Regular user accessed admin endpoint: $endpoint"
            unauthorized_access=true
        fi
    done

    if [ "$unauthorized_access" = false ]; then
        log_success "Admin endpoints properly protected or non-existent"
    fi
}

# TC-AUTHZ-007: JWT Token Manipulation - User ID
test_jwt_user_id_tampering() {
    log_info "TEST: TC-AUTHZ-007 - JWT Token Manipulation (User ID Tampering)"
    ((TESTS_RUN++))

    # Decode User A's token
    local header=$(echo "$USER_A_TOKEN" | cut -d. -f1 | base64 -d 2>/dev/null || echo "{}")
    local payload=$(echo "$USER_A_TOKEN" | cut -d. -f2 | base64 -d 2>/dev/null || echo "{}")

    # Modify user_id in payload to User B's ID
    local modified_payload=$(echo "$payload" | jq -c ".user_id = \"$USER_B_ID\"")
    local encoded_payload=$(echo -n "$modified_payload" | base64 | tr -d '=' | tr '+/' '-_')

    # Create tampered token (keep original signature - it will be invalid)
    local original_signature=$(echo "$USER_A_TOKEN" | cut -d. -f3)
    local tampered_token="$(echo "$USER_A_TOKEN" | cut -d. -f1).${encoded_payload}.${original_signature}"

    # Attempt to use tampered token
    local response=$(curl -s -w "\n%{http_code}" -X GET \
        "${API_BASE}/users/me" \
        -H "Authorization: Bearer $tampered_token")

    local status=$(echo "$response" | tail -n 1)

    if [ "$status" = "401" ]; then
        log_success "Token tampering detected: Invalid signature rejected (401)"
    else
        log_critical "JWT SIGNATURE BYPASS: Tampered token accepted! (HTTP $status) CVSS: 10.0"
    fi
}

# TC-AUTHZ-008: Like Bombing (Rate Limit + Idempotency)
test_like_bombing() {
    log_info "TEST: TC-AUTHZ-008 - Like Bombing (Idempotency Check)"
    ((TESTS_RUN++))

    # Create a public image
    local public_image=$(upload_test_image "$USER_A_TOKEN" "Public Image" "public")

    if [ -z "$public_image" ] || [ "$public_image" = "null" ]; then
        log_info "Skipping: Failed to create public image"
        return
    fi

    # User B attempts to like the same image 5 times
    local like_count=0
    for i in {1..5}; do
        local response=$(curl -s -w "\n%{http_code}" -X POST \
            "${API_BASE}/images/${public_image}/likes" \
            -H "Authorization: Bearer $USER_B_TOKEN")

        local status=$(echo "$response" | tail -n 1)

        [ "$status" = "200" ] || [ "$status" = "201" ] && ((like_count++))
    done

    # Verify like count on image
    local image_data=$(curl -s -X GET \
        "${API_BASE}/images/${public_image}" \
        -H "Authorization: Bearer $USER_B_TOKEN")

    local actual_likes=$(echo "$image_data" | jq -r '.likes_count // .like_count // 0')

    if [ "$actual_likes" -eq 1 ]; then
        log_success "Like operation is idempotent (5 attempts = 1 like)"
    elif [ "$actual_likes" -gt 1 ]; then
        log_fail "Like bombing possible: User liked image $actual_likes times"
    fi
}

# TC-AUTHZ-009: Comment Deletion Authorization
test_comment_deletion_authorization() {
    log_info "TEST: TC-AUTHZ-009 - Comment Deletion Authorization"
    ((TESTS_RUN++))

    # Create public image (User A)
    local public_image=$(upload_test_image "$USER_A_TOKEN" "Comment Test Image" "public")

    if [ -z "$public_image" ] || [ "$public_image" = "null" ]; then
        log_info "Skipping: Failed to create image"
        return
    fi

    # User B adds a comment
    local comment_response=$(curl -s -X POST \
        "${API_BASE}/images/${public_image}/comments" \
        -H "Authorization: Bearer $USER_B_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"text": "User B Comment"}')

    local comment_id=$(echo "$comment_response" | jq -r '.id // empty')

    if [ -z "$comment_id" ] || [ "$comment_id" = "null" ]; then
        log_info "Skipping: Failed to create comment"
        return
    fi

    # User A (image owner) attempts to delete User B's comment
    local delete_response=$(curl -s -w "\n%{http_code}" -X DELETE \
        "${API_BASE}/comments/${comment_id}" \
        -H "Authorization: Bearer $USER_A_TOKEN")

    local status=$(echo "$delete_response" | tail -n 1)

    # Expected: Only comment owner or moderator/admin can delete
    # Image owner should NOT be able to delete others' comments (unless they're a moderator)
    if [ "$status" = "403" ]; then
        log_success "Comment deletion properly restricted to owner/moderators"
    elif [ "$status" = "204" ] || [ "$status" = "200" ]; then
        log_info "Image owner can delete comments (verify this is intended behavior)"
    fi
}

main() {
    echo "=========================================="
    echo "  Access Control Test Suite"
    echo "  Target: $BASE_URL"
    echo "=========================================="

    for cmd in curl jq base64; do
        command -v $cmd &> /dev/null || { echo "ERROR: $cmd not found"; exit 1; }
    done

    setup_test_users

    echo; echo "Running access control tests..."; echo

    test_idor_image_retrieval
    test_idor_image_deletion
    test_idor_image_update
    test_horizontal_privilege_escalation
    test_vertical_privilege_escalation
    test_missing_function_level_access
    test_jwt_user_id_tampering
    test_like_bombing
    test_comment_deletion_authorization

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
