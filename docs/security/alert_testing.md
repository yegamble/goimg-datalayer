# Security Alert Testing Procedures

> Comprehensive testing guide for Grafana security alerts in goimg-datalayer.

**Last Updated**: 2025-12-06
**Owner**: Security Team
**Related**: [Security Alerting Runbook](/docs/operations/security-alerting.md)

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Testing Environment Setup](#testing-environment-setup)
4. [Alert Rule Testing](#alert-rule-testing)
5. [Notification Channel Testing](#notification-channel-testing)
6. [End-to-End Alert Testing](#end-to-end-alert-testing)
7. [Verification Procedures](#verification-procedures)
8. [Troubleshooting](#troubleshooting)
9. [Test Automation](#test-automation)

---

## Overview

This document provides step-by-step procedures for testing security alert rules, notification channels, and end-to-end alert delivery. Testing should be performed:

- **Before production deployment** - Verify all alerts work correctly
- **After configuration changes** - Validate threshold adjustments, new rules
- **Quarterly** - Ensure alert fatigue hasn't degraded monitoring
- **Post-incident** - Verify alerts detected/would have detected the incident

### Alert Testing Matrix

| Alert Type | Severity | Test Frequency | Test Method |
|------------|----------|----------------|-------------|
| Authentication Failures | Warning | Weekly | Automated script |
| Rate Limit Violations | Warning | Weekly | Automated script |
| Privilege Escalation | Critical (P1) | Monthly | Manual simulation |
| Malware Detection | Critical (P0) | Monthly | EICAR test file |
| Account Lockout | High (P2) | Weekly | Automated script |
| Brute Force | High | Monthly | Automated script |
| Account Enumeration | Warning | Monthly | Automated script |

---

## Prerequisites

### Required Tools

```bash
# Install required tools
sudo apt-get install -y curl jq docker.io

# Install Prometheus query tool
go install github.com/prometheus/prometheus/cmd/promtool@latest

# Install newman for API testing
npm install -g newman
```

### Required Access

- Grafana admin credentials
- Prometheus access (port 9090)
- API test credentials (test user account)
- Email inbox for notification testing
- Slack webhook URL (optional)
- PagerDuty integration key (optional)

### Environment Variables

```bash
# Set these before running tests
export GRAFANA_URL="http://localhost:3000"
export GRAFANA_ADMIN_USER="admin"
export GRAFANA_ADMIN_PASS="admin"
export API_URL="http://localhost:8080"
export TEST_USER_EMAIL="test@example.com"
export TEST_USER_PASSWORD="TestPassword123!"
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
```

---

## Testing Environment Setup

### 1. Start Monitoring Stack

```bash
# Start Prometheus, Grafana, and API
docker-compose -f docker/docker-compose.prod.yml up -d

# Verify services are running
docker-compose -f docker/docker-compose.prod.yml ps

# Expected output:
# grafana       Up      0.0.0.0:3000->3000/tcp
# prometheus    Up      0.0.0.0:9090->9090/tcp
# goimg-api     Up      0.0.0.0:8080->8080/tcp
# goimg-redis   Up      6379/tcp
# goimg-postgres Up     5432/tcp
```

### 2. Verify Prometheus Metrics

```bash
# Check Prometheus is scraping metrics
curl -s http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job: .job, health: .health}'

# Verify security metrics exist
curl -s http://localhost:9090/api/v1/query?query=goimg_security_auth_failures_total | jq '.data.result'

# Expected output: Should show metric with labels
```

### 3. Verify Grafana Alert Rules Loaded

```bash
# Get Grafana API token (for testing)
GRAFANA_TOKEN=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"test-token\",\"role\":\"Admin\"}" \
  http://admin:admin@localhost:3000/api/auth/keys | jq -r '.key')

# List all alert rules
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  http://localhost:3000/api/v1/provisioning/alert-rules | jq '.[] | {uid: .uid, title: .title}'

# Expected output: Should show all security alert rules
```

---

## Alert Rule Testing

### Test 1: Authentication Failure Alert

**Alert**: High Authentication Failure Rate (>10/min)
**Threshold**: 10 failures per minute
**For**: 2 minutes

#### Trigger Alert

```bash
#!/bin/bash
# File: tests/security/trigger_auth_failure_alert.sh

API_URL="http://localhost:8080"
TEST_EMAIL="nonexistent@example.com"
TEST_PASSWORD="WrongPassword123"

echo "Triggering authentication failure alert..."
echo "This will take ~3 minutes (10 failures/min for 2 min + wait time)"

for i in {1..25}; do
  echo "Attempt $i/25..."

  curl -s -X POST "$API_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}" \
    -w "\nHTTP Status: %{http_code}\n" > /dev/null

  # 5 second delay = 12 requests per minute
  sleep 5
done

echo "Triggered 25 failed login attempts over ~2 minutes"
echo "Alert should fire within 2 minutes (for=2m)"
echo "Check Grafana: http://localhost:3000/alerting/list"
```

#### Verify Alert Fired

```bash
# Wait 2 minutes for alert to fire (for=2m)
sleep 120

# Check alert status via Grafana API
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://localhost:3000/api/v1/provisioning/alert-rules/auth_failures_high" | \
  jq '{uid: .uid, title: .title, state: .state}'

# Check Prometheus query
curl -s "http://localhost:9090/api/v1/query?query=rate(goimg_security_auth_failures_total[1m])*60" | \
  jq '.data.result[] | {value: .value[1]}'

# Expected: value > 10
```

#### Resolve Alert

```bash
# Stop triggering failures, wait 3 minutes for alert to resolve
sleep 180

# Verify alert resolved
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://localhost:3000/api/v1/provisioning/alert-rules/auth_failures_high" | \
  jq '{uid: .uid, title: .title, state: .state}'

# Expected: state = "Normal"
```

---

### Test 2: Rate Limit Violation Alert

**Alert**: High Rate Limit Violation Rate (>100/min globally)
**Threshold**: 100 violations per minute
**For**: 2 minutes

#### Trigger Alert

```bash
#!/bin/bash
# File: tests/security/trigger_rate_limit_alert.sh

API_URL="http://localhost:8080"

echo "Triggering rate limit violation alert..."
echo "Sending 150 requests in 60 seconds..."

for i in {1..150}; do
  curl -s -X GET "$API_URL/api/v1/health" > /dev/null &

  # Small delay to spread requests across 1 minute
  sleep 0.4
done

wait

echo "Triggered ~150 requests in 1 minute"
echo "Alert should fire after 2 minutes of sustained >100 req/min"
echo "Check rate limit metric:"
curl -s "http://localhost:9090/api/v1/query?query=rate(goimg_security_rate_limit_exceeded_total[1m])*60" | \
  jq '.data.result[] | {value: .value[1]}'
```

---

### Test 3: Privilege Escalation Alert

**Alert**: Privilege Escalation Attempt (ANY occurrence)
**Threshold**: Any increase
**For**: Immediate (0s)

#### Trigger Alert

```bash
#!/bin/bash
# File: tests/security/trigger_privilege_escalation_alert.sh

API_URL="http://localhost:8080"

# First, login as regular user
TOKEN=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"user@example.com\",\"password\":\"UserPassword123!\"}" | \
  jq -r '.accessToken')

echo "Logged in as regular user, attempting admin action..."

# Attempt to access admin-only endpoint
curl -s -X GET "$API_URL/api/v1/admin/users" \
  -H "Authorization: Bearer $TOKEN" \
  -w "\nHTTP Status: %{http_code}\n"

# Expected: 403 Forbidden + metric incremented

echo "Checking authorization denied metric..."
curl -s "http://localhost:9090/api/v1/query?query=goimg_security_authorization_denied_total" | \
  jq '.data.result[] | {labels: .metric, value: .value[1]}'

# Alert should fire immediately (for=0s)
sleep 30

# Check alert status
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://localhost:3000/api/v1/provisioning/alert-rules/privilege_escalation_attempt" | \
  jq '{uid: .uid, title: .title, state: .state}'
```

---

### Test 4: Malware Detection Alert

**Alert**: Malware Detected (ANY occurrence)
**Threshold**: Any increase
**For**: Immediate (0s)
**Priority**: P0 (Critical)

#### Trigger Alert

```bash
#!/bin/bash
# File: tests/security/trigger_malware_alert.sh

API_URL="http://localhost:8080"

# Login and get token
TOKEN=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"test@example.com\",\"password\":\"TestPassword123!\"}" | \
  jq -r '.accessToken')

# Create EICAR test file (harmless malware test file)
echo 'X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*' > /tmp/eicar.txt

echo "Uploading EICAR test file (will trigger ClamAV detection)..."

# Upload EICAR file
RESPONSE=$(curl -s -X POST "$API_URL/api/v1/images/upload" \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/tmp/eicar.txt" \
  -F "title=Test Upload" \
  -w "\nHTTP Status: %{http_code}\n")

echo "Upload response: $RESPONSE"

# Expected: 400 Bad Request with "Malware detected" error

# Check malware metric
sleep 10
curl -s "http://localhost:9090/api/v1/query?query=goimg_security_malware_detected_total" | \
  jq '.data.result[] | {value: .value[1]}'

# Alert should fire immediately (for=0s)
echo "Checking alert status..."
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://localhost:3000/api/v1/provisioning/alert-rules/malware_detection" | \
  jq '{uid: .uid, title: .title, state: .state}'

# Expected: state = "Alerting"

# Cleanup
rm /tmp/eicar.txt
```

---

### Test 5: Account Lockout Alert

**Alert**: Account Lockout Triggered (ANY occurrence)
**Threshold**: Any increase
**For**: Immediate (0s)
**Priority**: P2 (High)

#### Trigger Alert

```bash
#!/bin/bash
# File: tests/security/trigger_account_lockout_alert.sh

API_URL="http://localhost:8080"
TEST_EMAIL="lockout-test@example.com"
WRONG_PASSWORD="WrongPassword123"

echo "Triggering account lockout alert..."
echo "Making 5 failed login attempts to lock account..."

for i in {1..5}; do
  echo "Failed attempt $i/5..."

  curl -s -X POST "$API_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$WRONG_PASSWORD\"}" \
    -w "\nHTTP Status: %{http_code}\n"

  sleep 2
done

echo "Account should now be locked. Attempting 6th login..."

# 6th attempt should return 403 (account locked)
RESPONSE=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"CorrectPassword123!\"}" \
  -w "\nHTTP Status: %{http_code}\n")

echo "6th login attempt response: $RESPONSE"
# Expected: 403 Forbidden with "Account temporarily locked" message

# Check account lockout metric
sleep 10
curl -s "http://localhost:9090/api/v1/query?query=goimg_security_auth_failures_total\{reason=\"account_locked\"\}" | \
  jq '.data.result[] | {labels: .metric, value: .value[1]}'

# Alert should fire immediately (for=0s)
echo "Checking alert status..."
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://localhost:3000/api/v1/provisioning/alert-rules/account_lockout_detected" | \
  jq '{uid: .uid, title: .title, state: .state}'

# Expected: state = "Alerting"

# Verify lockout in Redis
docker exec goimg-redis redis-cli --scan --pattern "goimg:lockout:*"

echo "Account will auto-unlock after 15 minutes"
```

---

## Notification Channel Testing

### Test Email Notifications

#### 1. Configure SMTP Settings

```bash
# Edit docker-compose.prod.yml
vi docker/docker-compose.prod.yml

# Add under grafana service environment:
environment:
  - GF_SMTP_ENABLED=true
  - GF_SMTP_HOST=smtp.gmail.com:587
  - GF_SMTP_USER=alerts@goimg.dev
  - GF_SMTP_PASSWORD=${SMTP_PASSWORD}
  - GF_SMTP_FROM_ADDRESS=alerts@goimg.dev
  - GF_SMTP_FROM_NAME=goimg Security Alerts

# Restart Grafana
docker-compose -f docker/docker-compose.prod.yml restart grafana
```

#### 2. Test Email Delivery

```bash
# Via Grafana UI:
# 1. Go to http://localhost:3000/alerting/notifications
# 2. Find "security-team-email" contact point
# 3. Click "Test" button
# 4. Check inbox for test email

# Via API:
curl -X POST "http://localhost:3000/api/alertmanager/grafana/api/v2/alerts/test" \
  -H "Authorization: Bearer $GRAFANA_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "receivers": [{
      "name": "security-team-email",
      "grafana_managed_receiver_configs": [{
        "uid": "email-security-team",
        "name": "security-team-email",
        "type": "email",
        "settings": {
          "addresses": "security@goimg.dev"
        }
      }]
    }],
    "alerts": [{
      "labels": {
        "alertname": "Test Alert",
        "severity": "critical"
      },
      "annotations": {
        "summary": "This is a test alert",
        "description": "Testing email notification delivery"
      }
    }]
  }'
```

### Test Slack Notifications

```bash
# Test Slack webhook directly
curl -X POST "${SLACK_WEBHOOK_URL}" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Test alert from goimg security monitoring"
  }'

# Expected: Message appears in Slack channel #security-alerts
```

### Test PagerDuty Notifications

```bash
# Test PagerDuty Events API v2
curl -X POST "https://events.pagerduty.com/v2/enqueue" \
  -H "Content-Type: application/json" \
  -d '{
    "routing_key": "'${PAGERDUTY_INTEGRATION_KEY}'",
    "event_action": "trigger",
    "payload": {
      "summary": "Test alert from goimg security monitoring",
      "severity": "critical",
      "source": "goimg-api",
      "custom_details": {
        "alert_name": "Test Alert",
        "environment": "production"
      }
    }
  }'

# Expected: Incident created in PagerDuty, on-call engineer paged
```

---

## End-to-End Alert Testing

### Full Alert Lifecycle Test

```bash
#!/bin/bash
# File: tests/security/e2e_alert_test.sh
# Tests complete alert flow: trigger → fire → notify → resolve

set -e

echo "=== E2E Alert Test: Authentication Failure Alert ==="

# Step 1: Verify baseline (no alerts firing)
echo "Step 1: Checking baseline alert status..."
INITIAL_STATE=$(curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://localhost:3000/api/v1/provisioning/alert-rules/auth_failures_high" | \
  jq -r '.state')

echo "Initial alert state: $INITIAL_STATE"
# Expected: "Normal"

# Step 2: Trigger alert
echo "Step 2: Triggering authentication failures..."
for i in {1..25}; do
  echo "  Attempt $i/25"
  curl -s -X POST "http://localhost:8080/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"WrongPassword"}' > /dev/null
  sleep 5
done

# Step 3: Wait for alert to fire (for=2m)
echo "Step 3: Waiting 2 minutes for alert to fire (for=2m)..."
sleep 120

# Step 4: Verify alert is firing
echo "Step 4: Checking alert status..."
FIRING_STATE=$(curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://localhost:3000/api/v1/provisioning/alert-rules/auth_failures_high" | \
  jq -r '.state')

echo "Alert state after triggering: $FIRING_STATE"

if [ "$FIRING_STATE" == "Alerting" ]; then
  echo "✓ Alert fired successfully"
else
  echo "✗ Alert did not fire (expected: Alerting, got: $FIRING_STATE)"
  exit 1
fi

# Step 5: Verify notification sent
echo "Step 5: Check inbox/Slack/PagerDuty for alert notification"
echo "  Expected: Email to security@goimg.dev"
echo "  Expected: Slack message in #security-alerts"
echo "  Press Enter when notification confirmed..."
read

# Step 6: Wait for alert to resolve
echo "Step 6: Waiting 3 minutes for alert to resolve..."
sleep 180

# Step 7: Verify alert resolved
echo "Step 7: Checking alert resolution..."
RESOLVED_STATE=$(curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://localhost:3000/api/v1/provisioning/alert-rules/auth_failures_high" | \
  jq -r '.state')

echo "Alert state after resolution: $RESOLVED_STATE"

if [ "$RESOLVED_STATE" == "Normal" ]; then
  echo "✓ Alert resolved successfully"
else
  echo "✗ Alert did not resolve (expected: Normal, got: $RESOLVED_STATE)"
  exit 1
fi

# Step 8: Verify resolved notification sent
echo "Step 8: Check inbox/Slack for resolution notification"
echo "  Expected: Email/Slack message with 'RESOLVED' status"
echo "  Press Enter when notification confirmed..."
read

echo "=== E2E Test Completed Successfully ==="
```

---

## Verification Procedures

### Alert Rule Verification Checklist

- [ ] Alert UID is unique
- [ ] Alert title is descriptive
- [ ] Prometheus query syntax is valid
- [ ] Threshold is appropriate (not too sensitive, not too lax)
- [ ] `for` duration prevents alert flapping
- [ ] `noDataState` is set correctly (OK or Alerting)
- [ ] `execErrState` is set to Alerting
- [ ] Annotations include summary, description, runbook_url
- [ ] Labels include severity, category, subcategory, team
- [ ] Critical alerts have priority label (P0, P1)

### Notification Verification Checklist

- [ ] Email notifications delivered within 1 minute
- [ ] Slack messages appear in correct channel
- [ ] PagerDuty incidents created for P0/P1 alerts
- [ ] Notification templates render correctly (no template errors)
- [ ] Alert context variables populated (user_id, IP, etc.)
- [ ] Runbook links work (not 404)
- [ ] Dashboard links work
- [ ] Resolved notifications sent when alert clears
- [ ] Notification grouping works (batches related alerts)
- [ ] Repeat interval prevents notification spam

---

## Troubleshooting

### Alerts Not Firing

**Problem**: Alert rule shows "Normal" but should be "Alerting"

**Diagnosis**:

```bash
# 1. Check Prometheus query manually
curl -s "http://localhost:9090/api/v1/query?query=rate(goimg_security_auth_failures_total[1m])*60" | \
  jq '.data.result'

# 2. Check Grafana evaluation logs
docker logs goimg-grafana | grep -i "alert"

# 3. Verify metric exists
curl -s "http://localhost:9090/api/v1/query?query=goimg_security_auth_failures_total" | \
  jq '.data.result[] | {metric: .metric, value: .value[1]}'

# 4. Check alert rule evaluation interval
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://localhost:3000/api/v1/provisioning/alert-rules/auth_failures_high" | \
  jq '{interval: .interval, for: .for}'
```

**Solutions**:
- Verify Prometheus is scraping API metrics
- Check PromQL query syntax
- Reduce threshold temporarily to confirm alert works
- Check `for` duration (alert won't fire immediately)

---

### Notifications Not Sending

**Problem**: Alert fires but no email/Slack/PagerDuty notification

**Diagnosis**:

```bash
# 1. Check contact point configuration
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://localhost:3000/api/v1/provisioning/contact-points" | \
  jq '.[] | {name: .name, type: .type, settings: .settings}'

# 2. Check notification policy routing
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://localhost:3000/api/v1/provisioning/policies" | jq '.'

# 3. Check Grafana logs for delivery errors
docker logs goimg-grafana | grep -i "notification"
docker logs goimg-grafana | grep -i "email\|slack\|pagerduty"

# 4. Test contact point directly
# Grafana UI → Alerting → Contact points → Test
```

**Solutions**:
- Verify SMTP settings (email)
- Verify webhook URL (Slack, PagerDuty)
- Check environment variables are set
- Verify network connectivity (firewall, DNS)
- Check notification policy matchers (labels must match)

---

## Test Automation

### Automated Alert Testing Script

```bash
#!/bin/bash
# File: tests/security/run_all_alert_tests.sh
# Runs all alert tests and reports results

set -e

RESULTS_DIR="./test-results"
mkdir -p "$RESULTS_DIR"

echo "=== Running All Security Alert Tests ==="
echo "Results will be saved to: $RESULTS_DIR"

# Test 1: Authentication Failures
echo "Test 1: Authentication Failure Alert..."
bash tests/security/trigger_auth_failure_alert.sh > "$RESULTS_DIR/auth_failure_test.log" 2>&1
if [ $? -eq 0 ]; then
  echo "✓ Authentication Failure Alert: PASS"
else
  echo "✗ Authentication Failure Alert: FAIL"
fi

# Test 2: Rate Limit Violations
echo "Test 2: Rate Limit Violation Alert..."
bash tests/security/trigger_rate_limit_alert.sh > "$RESULTS_DIR/rate_limit_test.log" 2>&1
if [ $? -eq 0 ]; then
  echo "✓ Rate Limit Violation Alert: PASS"
else
  echo "✗ Rate Limit Violation Alert: FAIL"
fi

# Test 3: Privilege Escalation
echo "Test 3: Privilege Escalation Alert..."
bash tests/security/trigger_privilege_escalation_alert.sh > "$RESULTS_DIR/privilege_escalation_test.log" 2>&1
if [ $? -eq 0 ]; then
  echo "✓ Privilege Escalation Alert: PASS"
else
  echo "✗ Privilege Escalation Alert: FAIL"
fi

# Test 4: Malware Detection
echo "Test 4: Malware Detection Alert..."
bash tests/security/trigger_malware_alert.sh > "$RESULTS_DIR/malware_test.log" 2>&1
if [ $? -eq 0 ]; then
  echo "✓ Malware Detection Alert: PASS"
else
  echo "✗ Malware Detection Alert: FAIL"
fi

# Test 5: Account Lockout
echo "Test 5: Account Lockout Alert..."
bash tests/security/trigger_account_lockout_alert.sh > "$RESULTS_DIR/account_lockout_test.log" 2>&1
if [ $? -eq 0 ]; then
  echo "✓ Account Lockout Alert: PASS"
else
  echo "✗ Account Lockout Alert: FAIL"
fi

echo ""
echo "=== Test Summary ==="
echo "Detailed logs available in: $RESULTS_DIR"
echo "Review logs for any failures and investigate root cause"
```

### CI/CD Integration

```yaml
# File: .github/workflows/alert-testing.yml
name: Security Alert Testing

on:
  schedule:
    - cron: '0 0 * * 0'  # Weekly on Sunday
  workflow_dispatch:  # Manual trigger

jobs:
  test-alerts:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v3

      - name: Start monitoring stack
        run: |
          docker-compose -f docker/docker-compose.prod.yml up -d
          sleep 60  # Wait for services to start

      - name: Run alert tests
        run: |
          bash tests/security/run_all_alert_tests.sh

      - name: Upload test results
        uses: actions/upload-artifact@v3
        with:
          name: alert-test-results
          path: test-results/

      - name: Cleanup
        run: |
          docker-compose -f docker/docker-compose.prod.yml down -v
```

---

## Additional Resources

- [Grafana Alerting Documentation](https://grafana.com/docs/grafana/latest/alerting/)
- [Prometheus Query Examples](https://prometheus.io/docs/prometheus/latest/querying/examples/)
- [Security Alerting Runbook](/docs/operations/security-alerting.md)
- [Security Events Dashboard](http://localhost:3000/d/goimg-security-events/security-events)
- [Alert Configuration](/monitoring/grafana/provisioning/alerting/README.md)

---

**Document Version**: 1.0
**Last Review**: 2025-12-06
**Next Review**: 2026-01-06
**Owner**: Security Team (security@goimg.dev)
