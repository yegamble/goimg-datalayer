# Grafana Alerting Configuration

> Automated security event alerting for goimg-datalayer using Grafana 10.x Unified Alerting.

**Last Updated**: 2025-12-05
**Grafana Version**: 10.x+
**Alerting Format**: Unified Alerting (YAML provisioning)

---

## Overview

This directory contains Grafana alert rules, contact points, notification policies, and templates for security event monitoring. All alerts are provisioned automatically when Grafana starts.

### Security Gate Compliance

All alerts implement security gate **S9-MON-001**: All security events must generate alerts.

---

## Directory Structure

```
monitoring/grafana/provisioning/alerting/
├── security_alerts.yml           # Alert rules and thresholds
├── contact_points.yml            # Notification destinations
├── notification_policies.yml     # Alert routing logic
├── notification_templates.yml    # Email, Slack, PagerDuty templates
└── README.md                     # This file
```

---

## Alert Rules

### File: `security_alerts.yml`

Defines 8 alert rules across 5 categories:

#### 1. Authentication Security

| Alert | Threshold | Severity | Priority | Description |
|-------|-----------|----------|----------|-------------|
| High Authentication Failure Rate | >10/min | Warning | P2 | Brute-force or credential stuffing detection |
| Account Lockout Triggered | ANY occurrence | High | P2 | Account locked due to failed login attempts |
| Potential Brute Force Attack | >50 from single IP in 10min | High | P2 | Targeted brute force attack |
| Potential Account Enumeration | >200 from single IP in 1h | Warning | P3 | Account enumeration attempt |

**Metrics Used**:
- `goimg_security_auth_failures_total{reason="invalid_credentials"}`
- `goimg_security_auth_failures_total{reason="account_locked"}`

#### 2. Rate Limiting

| Alert | Threshold | Severity | Priority | Description |
|-------|-----------|----------|----------|-------------|
| High Rate Limit Violation Rate | >100/min globally | Warning | P2 | API abuse or DDoS attack |

**Metrics Used**:
- `goimg_security_rate_limit_exceeded_total`

#### 3. Authorization

| Alert | Threshold | Severity | Priority | Description |
|-------|-----------|----------|----------|-------------|
| Privilege Escalation Attempt | ANY occurrence | Critical | P1 | Unauthorized access attempt |

**Metrics Used**:
- `goimg_security_authorization_denied_total`

#### 4. Malware Detection

| Alert | Threshold | Severity | Priority | Description |
|-------|-----------|----------|----------|-------------|
| Malware Detected | ANY occurrence | Critical | P0 | ClamAV detected malicious file |

**Metrics Used**:
- `goimg_security_malware_detected_total`

---

## Contact Points

### File: `contact_points.yml`

Defines 5 notification channels:

| Contact Point | Type | Use Case | Configuration Required |
|---------------|------|----------|----------------------|
| security-team-email | Email | All alerts | SMTP settings in Grafana config |
| security-slack-channel | Slack | Real-time notifications | `SLACK_WEBHOOK_URL` env var |
| security-pagerduty | PagerDuty | Critical alerts (P0/P1) | `PAGERDUTY_INTEGRATION_KEY` env var |
| security-webhook | Webhook | SIEM integration | `SECURITY_WEBHOOK_URL` env var |
| security-teams-channel | Microsoft Teams | Alternative to Slack | `TEAMS_WEBHOOK_URL` env var |

### Required Environment Variables

Set these in `docker-compose.prod.yml`:

```yaml
grafana:
  environment:
    # Email (SMTP)
    - GF_SMTP_ENABLED=true
    - GF_SMTP_HOST=smtp.gmail.com:587
    - GF_SMTP_USER=alerts@goimg.dev
    - GF_SMTP_PASSWORD=${SMTP_PASSWORD}
    - GF_SMTP_FROM_ADDRESS=alerts@goimg.dev
    - GF_SMTP_FROM_NAME=goimg Security Alerts

    # Slack
    - SLACK_WEBHOOK_URL=https://hooks.slack.com/services/T00000000/B00000000/XXXX

    # PagerDuty
    - PAGERDUTY_INTEGRATION_KEY=your-integration-key-here

    # Custom webhook (SIEM)
    - SECURITY_WEBHOOK_URL=https://siem.goimg.dev/api/v1/alerts

    # Microsoft Teams
    - TEAMS_WEBHOOK_URL=https://outlook.office.com/webhook/...
```

### Creating Webhooks

#### Slack Webhook
1. Go to https://api.slack.com/messaging/webhooks
2. Create new app or use existing
3. Enable "Incoming Webhooks"
4. Click "Add New Webhook to Workspace"
5. Select channel: `#security-alerts`
6. Copy webhook URL

#### PagerDuty Integration
1. Go to PagerDuty service settings
2. Click "Integrations" tab
3. Click "Add Integration"
4. Select "Events API v2"
5. Copy integration key

#### Microsoft Teams Webhook
1. Open Teams channel
2. Click "..." → "Connectors"
3. Search for "Incoming Webhook"
4. Click "Configure"
5. Provide name and image
6. Copy webhook URL

---

## Notification Policies

### File: `notification_policies.yml`

Routes alerts to appropriate contact points based on severity and labels.

### Routing Logic

```
┌─────────────────────────────────────────────┐
│ Alert Fired                                 │
└────────────────┬────────────────────────────┘
                 │
    ┌────────────▼────────────┐
    │ Priority = P0?          │
    │ (Malware Detection)     │
    └─────┬──────────────┬────┘
          │ Yes          │ No
          ▼              │
    ┌─────────────┐      │
    │ PagerDuty   │      │
    │ + Slack     │      │
    │ + Email     │      │
    └─────────────┘      │
                         │
           ┌─────────────▼──────────┐
           │ Priority = P1?         │
           │ (Privilege Escalation) │
           └─────┬──────────────┬───┘
                 │ Yes          │ No
                 ▼              │
           ┌─────────────┐      │
           │ PagerDuty   │      │
           │ + Slack     │      │
           └─────────────┘      │
                                │
                  ┌─────────────▼────────┐
                  │ Severity = Warning?  │
                  │ (Auth/Rate Limits)   │
                  └─────┬──────────────┬─┘
                        │ Yes          │ No
                        ▼              │
                  ┌─────────────┐      │
                  │ Slack       │      │
                  │ + Email     │      │
                  └─────────────┘      │
                                       │
                         ┌─────────────▼────────┐
                         │ All Security Alerts  │
                         └─────┬────────────────┘
                               ▼
                         ┌─────────────┐
                         │ Webhook     │
                         │ (SIEM)      │
                         └─────────────┘
```

### Grouping and Timing

| Setting | Value | Purpose |
|---------|-------|---------|
| group_by | alertname, severity, category | Batch related alerts together |
| group_wait | 10s - 2m | Wait for additional alerts before sending |
| group_interval | 1m - 10m | Re-evaluate grouped alerts |
| repeat_interval | 30m - 24h | Resend unresolved alerts |

---

## Notification Templates

### File: `notification_templates.yml`

Defines 4 templates:

| Template | Format | Channel | Features |
|----------|--------|---------|----------|
| security_email_template | HTML | Email | Rich formatting, remediation steps, severity colors |
| security_slack_template | Markdown | Slack | Emoji indicators, compact format, action links |
| security_pagerduty_template | Plain text | PagerDuty | Concise for mobile, essential info only |
| security_teams_template | JSON (MessageCard) | Microsoft Teams | Actionable cards, view buttons |

### Template Variables

All templates have access to:

```go
.Status              // "firing" or "resolved"
.CommonLabels        // Labels shared by all alerts in group
.CommonAnnotations   // Annotations shared by all alerts
.Alerts[]            // Array of individual alerts
  .Labels            // Alert-specific labels
  .Annotations       // Alert-specific annotations
  .StartsAt          // When alert started firing
  .EndsAt            // When alert resolved
  .Duration          // How long alert was firing
```

---

## Testing Alerts

### 1. Validate Configuration Syntax

```bash
# Install promtool (Prometheus toolkit)
go install github.com/prometheus/prometheus/cmd/promtool@latest

# Validate alert rules
promtool check rules monitoring/grafana/provisioning/alerting/security_alerts.yml
```

### 2. Test Alert Rules Locally

```bash
# Start Prometheus and Grafana
docker-compose -f docker/docker-compose.prod.yml up -d

# Trigger authentication failure alert (simulate attack)
for i in {1..15}; do
  curl -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"wrong"}' \
    -w "\n"
  sleep 5
done

# Wait 2 minutes for alert to fire (for=2m)
# Check Grafana Alerting UI: http://localhost:3000/alerting/list
```

### 3. Test Notification Channels

```bash
# Test email via Grafana UI
# 1. Go to Alerting → Contact points
# 2. Find "security-team-email"
# 3. Click "Test" button
# 4. Check inbox

# Test Slack webhook manually
curl -X POST ${SLACK_WEBHOOK_URL} \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Test alert from goimg security monitoring"
  }'

# Test PagerDuty integration
curl -X POST https://events.pagerduty.com/v2/enqueue \
  -H "Content-Type: application/json" \
  -d '{
    "routing_key": "'${PAGERDUTY_INTEGRATION_KEY}'",
    "event_action": "trigger",
    "payload": {
      "summary": "Test alert from goimg",
      "severity": "critical",
      "source": "goimg-api"
    }
  }'
```

### 4. Simulate Security Events

```bash
# Trigger malware detection alert
# Upload EICAR test file (harmless malware test file)
echo 'X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*' > /tmp/eicar.txt
curl -X POST http://localhost:8080/api/v1/images/upload \
  -H "Authorization: Bearer ${TOKEN}" \
  -F "file=@/tmp/eicar.txt"

# Wait 30 seconds for alert to fire (for=0s)
# Check PagerDuty for critical alert
```

---

## Troubleshooting

### Alerts Not Firing

1. **Check Prometheus is scraping metrics**:
   ```bash
   # Verify Prometheus can reach API
   curl http://localhost:9091/api/v1/targets

   # Check if metrics exist
   curl http://localhost:9090/metrics | grep goimg_security
   ```

2. **Verify alert rules are loaded**:
   ```bash
   # Check Grafana logs
   docker logs goimg-grafana | grep -i "alert"

   # Check for errors in provisioning
   docker logs goimg-grafana | grep -i "error"
   ```

3. **Test PromQL query manually**:
   ```bash
   # Go to Grafana → Explore
   # Run: rate(goimg_security_auth_failures_total[1m]) * 60
   # Should return data if metrics exist
   ```

### Alerts Firing Too Often (Flapping)

1. **Increase `for` duration**:
   ```yaml
   # Change from:
   for: 2m
   # To:
   for: 5m
   ```

2. **Adjust thresholds**:
   ```yaml
   # Change from:
   params: [10, 0]  # >10
   # To:
   params: [20, 0]  # >20
   ```

3. **Add more context to `group_by`**:
   ```yaml
   # Add more labels to reduce grouping
   group_by: [alertname, severity, category, ip]
   ```

### Notifications Not Sending

1. **Check contact point status**:
   ```bash
   # Grafana UI → Alerting → Contact points
   # Look for "Last delivery attempt" column
   ```

2. **Test contact point**:
   ```bash
   # Click "Test" button on contact point
   # Check logs for errors
   docker logs goimg-grafana | grep -i "notification"
   ```

3. **Verify webhook URLs**:
   ```bash
   # Check environment variables are set
   docker exec goimg-grafana env | grep SLACK_WEBHOOK_URL
   docker exec goimg-grafana env | grep PAGERDUTY_INTEGRATION_KEY
   ```

4. **Check SMTP settings**:
   ```bash
   # Verify SMTP is configured
   docker exec goimg-grafana grep -A 10 "\[smtp\]" /etc/grafana/grafana.ini

   # Test SMTP connection
   docker exec goimg-grafana nc -zv smtp.gmail.com 587
   ```

### Alert Templates Not Rendering

1. **Check template syntax**:
   ```bash
   # Look for template errors in logs
   docker logs goimg-grafana | grep -i "template"
   ```

2. **Verify template name matches**:
   ```yaml
   # In contact_points.yml, template reference:
   template: security_email_template

   # In notification_templates.yml, template name:
   - name: security_email_template
   ```

3. **Test template variables**:
   ```go
   # Use {{ . | jsonify }} in template to see all available variables
   {{ . | jsonify }}
   ```

---

## Customization

### Adding New Alert Rules

1. Edit `security_alerts.yml`
2. Add new rule under appropriate group:
   ```yaml
   - name: security_custom
     interval: 1m
     folder: Security
     rules:
       - uid: my_custom_alert
         title: My Custom Alert
         condition: C
         data:
           - refId: A
             relativeTimeRange:
               from: 300
               to: 0
             datasourceUid: prometheus
             model:
               expr: my_custom_metric > 100
               refId: A
         annotations:
           summary: Custom alert triggered
           description: My custom security alert
         labels:
           severity: warning
           category: security
   ```

3. Restart Grafana:
   ```bash
   docker-compose -f docker/docker-compose.prod.yml restart grafana
   ```

### Modifying Alert Thresholds

1. Edit `security_alerts.yml`
2. Find alert rule by `uid` or `title`
3. Update threshold in `params`:
   ```yaml
   # Change from >10 to >20
   conditions:
     - evaluator:
         params: [10, 0]  # Old threshold
         # Change to:
         params: [20, 0]  # New threshold
   ```

4. Reload Grafana (no restart needed - provisioning watches for changes)

### Adding New Contact Points

1. Edit `contact_points.yml`
2. Add new receiver:
   ```yaml
   - orgId: 1
     name: security-custom
     receivers:
       - uid: custom-integration
         type: webhook
         settings:
           url: https://my-service.com/alerts
           httpMethod: POST
   ```

3. Update `notification_policies.yml` to route alerts:
   ```yaml
   routes:
     - receiver: security-custom
       matchers:
         - severity = critical
   ```

---

## Maintenance

### Regular Tasks

| Task | Frequency | Owner |
|------|-----------|-------|
| Review alert thresholds | Monthly | Security Team |
| Test notification channels | Weekly | On-call Engineer |
| Update runbook documentation | After each incident | Incident Commander |
| Review silenced alerts | Weekly | Security Team Lead |
| Audit alert response times | Quarterly | Engineering Manager |

### Alert Tuning Checklist

- [ ] Review alert firing frequency (target: <5 false positives/day)
- [ ] Analyze alert-to-incident ratio (target: >80%)
- [ ] Check notification delivery success rate (target: >99%)
- [ ] Verify runbook accuracy (test procedures quarterly)
- [ ] Update severity classifications based on real incidents
- [ ] Adjust thresholds based on production metrics
- [ ] Remove or consolidate redundant alerts

---

## Security Considerations

1. **Protect webhook URLs**: Never commit webhook URLs to git
   - Use environment variables
   - Store in secrets management (HashiCorp Vault, AWS Secrets Manager)

2. **Restrict Grafana access**:
   - Change default admin password
   - Enable authentication (OAuth, LDAP, SAML)
   - Use RBAC for alert management

3. **Encrypt notification traffic**:
   - Use HTTPS for all webhooks
   - Enable TLS for SMTP

4. **Audit alert modifications**:
   - Track changes to alert rules via git
   - Log alert silencing actions
   - Review contact point changes

5. **Test disaster recovery**:
   - Backup Grafana database regularly
   - Document restoration procedures
   - Test failover to secondary Grafana instance

---

## References

- [Grafana Unified Alerting](https://grafana.com/docs/grafana/latest/alerting/unified-alerting/)
- [Prometheus Alerting Rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
- [Slack Incoming Webhooks](https://api.slack.com/messaging/webhooks)
- [PagerDuty Events API v2](https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgw-events-api-v2-overview)
- [Security Alerting Runbook](/docs/operations/security-alerting.md)
- [Alert Testing Procedures](/docs/security/alert_testing.md)

---

**Document Version**: 1.0
**Last Updated**: 2025-12-05
**Next Review**: 2026-01-05
**Maintainer**: Security Team (security@goimg.dev)
