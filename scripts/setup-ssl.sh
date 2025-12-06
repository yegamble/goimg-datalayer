#!/usr/bin/env bash

# ============================================================================
# SSL Certificate Setup Script for goimg-datalayer
# ============================================================================
# This script manages Let's Encrypt SSL certificates using Certbot
#
# Features:
# - HTTP-01 and DNS-01 challenge support
# - Automatic renewal configuration (cron/systemd)
# - Certificate validity checking
# - Multi-domain support
# - Automatic nginx reload
# - Comprehensive error handling and logging
#
# Usage:
#   ./setup-ssl.sh --domain example.com --email admin@example.com --webroot /var/www/certbot
#   ./setup-ssl.sh --domain example.com --email admin@example.com --dns cloudflare
#   ./setup-ssl.sh --check-validity
#   ./setup-ssl.sh --setup-renewal
#   ./setup-ssl.sh --renew
#
# Requirements:
# - certbot installed
# - nginx running (for HTTP-01)
# - DNS provider credentials configured (for DNS-01)
# ============================================================================

set -euo pipefail

# ============================================================================
# Configuration Variables
# ============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
NGINX_SSL_DIR="${PROJECT_ROOT}/docker/nginx/ssl"
NGINX_CONF_DIR="${PROJECT_ROOT}/docker/nginx/conf.d"
LOG_FILE="${PROJECT_ROOT}/logs/ssl-setup.log"

# Default values
DOMAIN=""
EMAIL=""
WEBROOT="/var/www/certbot"
CHALLENGE_TYPE="http"  # http or dns
DNS_PLUGIN=""          # cloudflare, route53, google, etc.
STAGING=false          # Use Let's Encrypt staging for testing
FORCE_RENEWAL=false
DRY_RUN=false

# Certificate paths
LETSENCRYPT_DIR="/etc/letsencrypt"
CERT_LIVE_DIR="${LETSENCRYPT_DIR}/live"

# ============================================================================
# Color Output
# ============================================================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================================================
# Logging Functions
# ============================================================================
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    # Create log directory if it doesn't exist
    mkdir -p "$(dirname "$LOG_FILE")"

    # Log to file
    echo "[${timestamp}] [${level}] ${message}" >> "$LOG_FILE"

    # Log to console with color
    case "$level" in
        ERROR)
            echo -e "${RED}[ERROR]${NC} ${message}" >&2
            ;;
        WARN)
            echo -e "${YELLOW}[WARN]${NC} ${message}"
            ;;
        INFO)
            echo -e "${GREEN}[INFO]${NC} ${message}"
            ;;
        DEBUG)
            echo -e "${BLUE}[DEBUG]${NC} ${message}"
            ;;
        *)
            echo "[${level}] ${message}"
            ;;
    esac
}

# ============================================================================
# Error Handling
# ============================================================================
error_exit() {
    log ERROR "$1"
    exit 1
}

cleanup() {
    local exit_code=$?
    if [ $exit_code -ne 0 ]; then
        log ERROR "Script failed with exit code: $exit_code"
    fi
}

trap cleanup EXIT

# ============================================================================
# Validation Functions
# ============================================================================
check_root() {
    if [ "$EUID" -ne 0 ]; then
        error_exit "This script must be run as root (use sudo)"
    fi
}

check_certbot() {
    if ! command -v certbot &> /dev/null; then
        log ERROR "certbot is not installed"
        log INFO "Installing certbot..."

        if command -v apt-get &> /dev/null; then
            apt-get update
            apt-get install -y certbot
        elif command -v yum &> /dev/null; then
            yum install -y certbot
        else
            error_exit "Cannot install certbot automatically. Please install manually."
        fi
    fi

    log INFO "certbot version: $(certbot --version 2>&1)"
}

validate_email() {
    local email="$1"
    if [[ ! "$email" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
        error_exit "Invalid email address: $email"
    fi
}

validate_domain() {
    local domain="$1"
    if [[ ! "$domain" =~ ^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$ ]]; then
        error_exit "Invalid domain name: $domain"
    fi
}

check_dns_resolution() {
    local domain="$1"
    log INFO "Checking DNS resolution for $domain..."

    if ! dig +short "$domain" &> /dev/null; then
        log WARN "DNS resolution failed for $domain. Ensure domain is properly configured."
        return 1
    fi

    local resolved_ip
    resolved_ip=$(dig +short "$domain" | head -n1)
    log INFO "Domain $domain resolves to: $resolved_ip"
}

check_port_80() {
    log INFO "Checking if port 80 is accessible..."

    if ! command -v nc &> /dev/null; then
        log WARN "netcat not installed, skipping port check"
        return 0
    fi

    if ! timeout 5 nc -z -v localhost 80 2>&1 | grep -q succeeded; then
        log WARN "Port 80 is not accessible. Ensure nginx is running for HTTP-01 challenge."
        return 1
    fi

    log INFO "Port 80 is accessible"
}

# ============================================================================
# Certificate Management Functions
# ============================================================================
obtain_certificate_http01() {
    local domain="$1"
    local email="$2"
    local webroot="$3"

    log INFO "Obtaining SSL certificate via HTTP-01 challenge..."
    log INFO "Domain: $domain"
    log INFO "Email: $email"
    log INFO "Webroot: $webroot"

    # Ensure webroot directory exists
    mkdir -p "$webroot"

    # Build certbot command
    local certbot_cmd="certbot certonly"
    certbot_cmd+=" --webroot"
    certbot_cmd+=" -w $webroot"
    certbot_cmd+=" -d $domain"
    certbot_cmd+=" --email $email"
    certbot_cmd+=" --agree-tos"
    certbot_cmd+=" --no-eff-email"
    certbot_cmd+=" --non-interactive"

    if [ "$STAGING" = true ]; then
        certbot_cmd+=" --staging"
        log WARN "Using Let's Encrypt STAGING environment (certificates will not be trusted)"
    fi

    if [ "$FORCE_RENEWAL" = true ]; then
        certbot_cmd+=" --force-renewal"
    fi

    if [ "$DRY_RUN" = true ]; then
        certbot_cmd+=" --dry-run"
        log INFO "DRY RUN: $certbot_cmd"
        return 0
    fi

    log DEBUG "Executing: $certbot_cmd"

    if eval "$certbot_cmd"; then
        log INFO "Certificate obtained successfully!"
        copy_certificates "$domain"
    else
        error_exit "Failed to obtain certificate"
    fi
}

obtain_certificate_dns01() {
    local domain="$1"
    local email="$2"
    local dns_plugin="$3"

    log INFO "Obtaining SSL certificate via DNS-01 challenge..."
    log INFO "Domain: $domain"
    log INFO "Email: $email"
    log INFO "DNS Plugin: $dns_plugin"

    # Check if DNS plugin is installed
    local plugin_pkg="python3-certbot-dns-${dns_plugin}"
    if ! dpkg -l | grep -q "$plugin_pkg" 2>/dev/null; then
        log WARN "DNS plugin not installed: $plugin_pkg"
        log INFO "Installing $plugin_pkg..."

        if command -v apt-get &> /dev/null; then
            apt-get update
            apt-get install -y "$plugin_pkg"
        else
            error_exit "Cannot install $plugin_pkg automatically. Please install manually."
        fi
    fi

    # Build certbot command
    local certbot_cmd="certbot certonly"
    certbot_cmd+=" --dns-${dns_plugin}"
    certbot_cmd+=" -d $domain"
    certbot_cmd+=" -d *.$domain"  # Wildcard support
    certbot_cmd+=" --email $email"
    certbot_cmd+=" --agree-tos"
    certbot_cmd+=" --no-eff-email"
    certbot_cmd+=" --non-interactive"

    if [ "$STAGING" = true ]; then
        certbot_cmd+=" --staging"
        log WARN "Using Let's Encrypt STAGING environment"
    fi

    if [ "$FORCE_RENEWAL" = true ]; then
        certbot_cmd+=" --force-renewal"
    fi

    if [ "$DRY_RUN" = true ]; then
        certbot_cmd+=" --dry-run"
        log INFO "DRY RUN: $certbot_cmd"
        return 0
    fi

    log DEBUG "Executing: $certbot_cmd"

    if eval "$certbot_cmd"; then
        log INFO "Certificate obtained successfully!"
        copy_certificates "$domain"
    else
        error_exit "Failed to obtain certificate"
    fi
}

copy_certificates() {
    local domain="$1"
    local cert_dir="${CERT_LIVE_DIR}/${domain}"

    log INFO "Copying certificates to nginx SSL directory..."

    if [ ! -d "$cert_dir" ]; then
        error_exit "Certificate directory not found: $cert_dir"
    fi

    # Ensure nginx SSL directory exists
    mkdir -p "$NGINX_SSL_DIR"

    # Copy certificates
    cp -f "${cert_dir}/fullchain.pem" "${NGINX_SSL_DIR}/fullchain.pem"
    cp -f "${cert_dir}/privkey.pem" "${NGINX_SSL_DIR}/privkey.pem"
    cp -f "${cert_dir}/chain.pem" "${NGINX_SSL_DIR}/chain.pem"
    cp -f "${cert_dir}/cert.pem" "${NGINX_SSL_DIR}/cert.pem"

    # Set proper permissions
    chmod 644 "${NGINX_SSL_DIR}/fullchain.pem"
    chmod 644 "${NGINX_SSL_DIR}/chain.pem"
    chmod 644 "${NGINX_SSL_DIR}/cert.pem"
    chmod 600 "${NGINX_SSL_DIR}/privkey.pem"

    log INFO "Certificates copied successfully"
    log INFO "Certificate: ${NGINX_SSL_DIR}/fullchain.pem"
    log INFO "Private Key: ${NGINX_SSL_DIR}/privkey.pem"

    # Update nginx configuration with domain
    update_nginx_config "$domain"
}

update_nginx_config() {
    local domain="$1"
    local api_conf="${NGINX_CONF_DIR}/api.conf"

    log INFO "Updating nginx configuration with domain: $domain"

    if [ ! -f "$api_conf" ]; then
        log WARN "Nginx config not found: $api_conf"
        return 1
    fi

    # Replace example.com with actual domain
    sed -i "s/example\.com/$domain/g" "$api_conf"

    log INFO "Nginx configuration updated"
}

reload_nginx() {
    log INFO "Reloading nginx configuration..."

    # Check if running in Docker
    if command -v docker &> /dev/null && docker ps | grep -q goimg-nginx; then
        log INFO "Reloading nginx in Docker container..."
        docker exec goimg-nginx nginx -t
        docker exec goimg-nginx nginx -s reload
        log INFO "Nginx reloaded successfully (Docker)"
    elif command -v systemctl &> /dev/null && systemctl is-active --quiet nginx; then
        log INFO "Reloading nginx via systemd..."
        nginx -t
        systemctl reload nginx
        log INFO "Nginx reloaded successfully (systemd)"
    elif command -v service &> /dev/null; then
        log INFO "Reloading nginx via service..."
        nginx -t
        service nginx reload
        log INFO "Nginx reloaded successfully (service)"
    else
        log WARN "Cannot reload nginx automatically. Please reload manually."
        return 1
    fi
}

# ============================================================================
# Certificate Renewal Functions
# ============================================================================
renew_certificates() {
    log INFO "Renewing SSL certificates..."

    if [ "$DRY_RUN" = true ]; then
        log INFO "DRY RUN: certbot renew --dry-run"
        certbot renew --dry-run
        return 0
    fi

    if certbot renew --quiet --post-hook "$(realpath "$0") --post-renewal"; then
        log INFO "Certificate renewal completed successfully"
    else
        log ERROR "Certificate renewal failed"
        return 1
    fi
}

post_renewal_hook() {
    log INFO "Running post-renewal hook..."

    # Copy renewed certificates
    for cert_dir in "${CERT_LIVE_DIR}"/*; do
        if [ -d "$cert_dir" ]; then
            local domain
            domain=$(basename "$cert_dir")
            log INFO "Copying renewed certificates for: $domain"
            copy_certificates "$domain"
        fi
    done

    # Reload nginx
    reload_nginx
}

setup_auto_renewal() {
    log INFO "Setting up automatic certificate renewal..."

    # Prefer systemd timer over cron
    if command -v systemctl &> /dev/null; then
        setup_systemd_renewal
    else
        setup_cron_renewal
    fi
}

setup_systemd_renewal() {
    log INFO "Configuring systemd timer for automatic renewal..."

    # Create systemd service
    cat > /etc/systemd/system/certbot-renewal.service <<EOF
[Unit]
Description=Certbot SSL Certificate Renewal
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/bin/certbot renew --quiet --post-hook "$(realpath "$0") --post-renewal"
EOF

    # Create systemd timer (runs twice daily)
    cat > /etc/systemd/system/certbot-renewal.timer <<EOF
[Unit]
Description=Certbot SSL Certificate Renewal Timer

[Timer]
OnCalendar=*-*-* 00,12:00:00
RandomizedDelaySec=3600
Persistent=true

[Install]
WantedBy=timers.target
EOF

    # Enable and start timer
    systemctl daemon-reload
    systemctl enable certbot-renewal.timer
    systemctl start certbot-renewal.timer

    log INFO "Systemd timer configured successfully"
    log INFO "Timer status: $(systemctl is-active certbot-renewal.timer)"

    # Show next run time
    systemctl list-timers certbot-renewal.timer --no-pager
}

setup_cron_renewal() {
    log INFO "Configuring cron job for automatic renewal..."

    local cron_cmd="0 0,12 * * * /usr/bin/certbot renew --quiet --post-hook '$(realpath "$0") --post-renewal'"

    # Add to root crontab if not already present
    if ! crontab -l 2>/dev/null | grep -q "certbot renew"; then
        (crontab -l 2>/dev/null; echo "$cron_cmd") | crontab -
        log INFO "Cron job added successfully"
    else
        log INFO "Cron job already exists"
    fi

    log INFO "Automatic renewal configured (runs twice daily at 00:00 and 12:00)"
}

# ============================================================================
# Certificate Validation Functions
# ============================================================================
check_certificate_validity() {
    log INFO "Checking certificate validity..."

    local cert_file="${NGINX_SSL_DIR}/fullchain.pem"

    if [ ! -f "$cert_file" ]; then
        log ERROR "Certificate not found: $cert_file"
        return 1
    fi

    # Extract certificate information
    local expiry_date
    expiry_date=$(openssl x509 -enddate -noout -in "$cert_file" | cut -d= -f2)

    local expiry_epoch
    expiry_epoch=$(date -d "$expiry_date" +%s)

    local current_epoch
    current_epoch=$(date +%s)

    local days_remaining
    days_remaining=$(( (expiry_epoch - current_epoch) / 86400 ))

    log INFO "Certificate: $cert_file"
    log INFO "Expires: $expiry_date"
    log INFO "Days remaining: $days_remaining"

    # Extract subject and issuer
    local subject
    subject=$(openssl x509 -subject -noout -in "$cert_file" | sed 's/subject=//')

    local issuer
    issuer=$(openssl x509 -issuer -noout -in "$cert_file" | sed 's/issuer=//')

    log INFO "Subject: $subject"
    log INFO "Issuer: $issuer"

    # Check if renewal is needed (< 30 days)
    if [ "$days_remaining" -lt 30 ]; then
        log WARN "Certificate expires in less than 30 days. Renewal recommended."
        return 2
    elif [ "$days_remaining" -lt 7 ]; then
        log ERROR "Certificate expires in less than 7 days! Renewal URGENT."
        return 3
    fi

    log INFO "Certificate is valid and does not require renewal yet"
    return 0
}

test_ssl_configuration() {
    local domain="$1"

    log INFO "Testing SSL configuration for: $domain"

    # Test with openssl
    if command -v openssl &> /dev/null; then
        log INFO "Testing SSL connection..."
        if echo | openssl s_client -connect "${domain}:443" -servername "$domain" 2>/dev/null | grep -q "Verify return code: 0"; then
            log INFO "SSL connection test: PASSED"
        else
            log WARN "SSL connection test: FAILED"
        fi
    fi

    # Suggest SSL Labs test
    log INFO "For comprehensive SSL testing, visit:"
    log INFO "https://www.ssllabs.com/ssltest/analyze.html?d=$domain"
}

generate_dhparam() {
    local dhparam_file="${NGINX_SSL_DIR}/dhparam.pem"

    if [ -f "$dhparam_file" ]; then
        log INFO "DH parameters already exist: $dhparam_file"
        return 0
    fi

    log INFO "Generating DH parameters (2048-bit)..."
    log WARN "This may take several minutes..."

    if openssl dhparam -out "$dhparam_file" 2048; then
        chmod 644 "$dhparam_file"
        log INFO "DH parameters generated successfully: $dhparam_file"
    else
        log ERROR "Failed to generate DH parameters"
        return 1
    fi
}

# ============================================================================
# Usage and Help
# ============================================================================
usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

SSL Certificate Management Script for goimg-datalayer

OPTIONS:
    -d, --domain DOMAIN          Domain name for the certificate (required for obtain)
    -e, --email EMAIL            Email address for Let's Encrypt notifications (required for obtain)
    -w, --webroot PATH           Webroot path for HTTP-01 challenge (default: /var/www/certbot)
    --dns PLUGIN                 Use DNS-01 challenge with specified plugin (cloudflare, route53, etc.)
    --staging                    Use Let's Encrypt staging environment (for testing)
    --force                      Force certificate renewal even if not expired
    --dry-run                    Test certificate operations without making changes

COMMANDS:
    --obtain                     Obtain new SSL certificate
    --renew                      Renew existing certificates
    --check-validity             Check certificate expiration and validity
    --setup-renewal              Configure automatic renewal (cron/systemd)
    --post-renewal               Post-renewal hook (internal use)
    --generate-dhparam           Generate DH parameters for enhanced security
    --test                       Test SSL configuration
    -h, --help                   Show this help message

EXAMPLES:
    # Obtain certificate using HTTP-01 challenge:
    sudo $0 --obtain --domain example.com --email admin@example.com

    # Obtain certificate using DNS-01 challenge (with wildcard):
    sudo $0 --obtain --domain example.com --email admin@example.com --dns cloudflare

    # Test certificate acquisition (staging):
    sudo $0 --obtain --domain example.com --email admin@example.com --staging --dry-run

    # Check certificate validity:
    sudo $0 --check-validity

    # Manual renewal:
    sudo $0 --renew

    # Setup automatic renewal:
    sudo $0 --setup-renewal

    # Generate DH parameters:
    sudo $0 --generate-dhparam

    # Test SSL configuration:
    sudo $0 --test --domain example.com

ENVIRONMENT VARIABLES:
    SSL_DOMAIN                   Override domain (alternative to --domain)
    SSL_EMAIL                    Override email (alternative to --email)

EOF
}

# ============================================================================
# Main Function
# ============================================================================
main() {
    local action=""

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -d|--domain)
                DOMAIN="$2"
                shift 2
                ;;
            -e|--email)
                EMAIL="$2"
                shift 2
                ;;
            -w|--webroot)
                WEBROOT="$2"
                shift 2
                ;;
            --dns)
                CHALLENGE_TYPE="dns"
                DNS_PLUGIN="$2"
                shift 2
                ;;
            --staging)
                STAGING=true
                shift
                ;;
            --force)
                FORCE_RENEWAL=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --obtain)
                action="obtain"
                shift
                ;;
            --renew)
                action="renew"
                shift
                ;;
            --check-validity)
                action="check-validity"
                shift
                ;;
            --setup-renewal)
                action="setup-renewal"
                shift
                ;;
            --post-renewal)
                action="post-renewal"
                shift
                ;;
            --generate-dhparam)
                action="generate-dhparam"
                shift
                ;;
            --test)
                action="test"
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                error_exit "Unknown option: $1 (use --help for usage)"
                ;;
        esac
    done

    # Environment variable overrides
    DOMAIN="${SSL_DOMAIN:-$DOMAIN}"
    EMAIL="${SSL_EMAIL:-$EMAIL}"

    # Execute action
    case "$action" in
        obtain)
            check_root
            check_certbot

            [ -z "$DOMAIN" ] && error_exit "Domain is required (use --domain or set SSL_DOMAIN)"
            [ -z "$EMAIL" ] && error_exit "Email is required (use --email or set SSL_EMAIL)"

            validate_domain "$DOMAIN"
            validate_email "$EMAIL"

            if [ "$CHALLENGE_TYPE" = "http" ]; then
                check_dns_resolution "$DOMAIN"
                check_port_80
                obtain_certificate_http01 "$DOMAIN" "$EMAIL" "$WEBROOT"
            else
                obtain_certificate_dns01 "$DOMAIN" "$EMAIL" "$DNS_PLUGIN"
            fi

            reload_nginx
            check_certificate_validity

            log INFO "SSL certificate setup completed successfully!"
            log INFO "Next steps:"
            log INFO "1. Update your nginx configuration if needed"
            log INFO "2. Test SSL: $0 --test --domain $DOMAIN"
            log INFO "3. Setup auto-renewal: $0 --setup-renewal"
            log INFO "4. Verify at https://www.ssllabs.com/ssltest/analyze.html?d=$DOMAIN"
            ;;
        renew)
            check_root
            check_certbot
            renew_certificates
            ;;
        check-validity)
            check_certificate_validity
            ;;
        setup-renewal)
            check_root
            setup_auto_renewal
            ;;
        post-renewal)
            post_renewal_hook
            ;;
        generate-dhparam)
            check_root
            generate_dhparam
            ;;
        test)
            [ -z "$DOMAIN" ] && error_exit "Domain is required for testing (use --domain)"
            test_ssl_configuration "$DOMAIN"
            ;;
        "")
            log ERROR "No action specified"
            usage
            exit 1
            ;;
        *)
            error_exit "Unknown action: $action"
            ;;
    esac
}

# ============================================================================
# Script Entry Point
# ============================================================================
main "$@"
