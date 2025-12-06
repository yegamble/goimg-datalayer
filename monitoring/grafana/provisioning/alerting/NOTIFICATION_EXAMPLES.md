# Alert Notification Channel Examples

> Configuration examples for Grafana alert notification channels.

**Last Updated**: 2025-12-06
**Related**: [Contact Points Configuration](contact_points.yml)

---

## Overview

This document provides practical examples for configuring alert notification channels in Grafana. All examples use environment variables for sensitive credentials.

---

## Email Notifications

### SMTP Configuration (Gmail)

```yaml
# docker-compose.prod.yml
services:
  grafana:
    environment:
      - GF_SMTP_ENABLED=true
      - GF_SMTP_HOST=smtp.gmail.com:587
      - GF_SMTP_USER=alerts@goimg.dev
      - GF_SMTP_PASSWORD=${SMTP_PASSWORD}
      - GF_SMTP_FROM_ADDRESS=alerts@goimg.dev
      - GF_SMTP_FROM_NAME=goimg Security Alerts
      - GF_SMTP_SKIP_VERIFY=false
```

### Gmail App Password Setup

1. Go to https://myaccount.google.com/security
2. Enable 2-Step Verification
3. Go to App Passwords
4. Select "Mail" and "Other (Custom name)"
5. Enter "goimg Alerts"
6. Copy generated password
7. Set in `.env` file:

```bash
SMTP_PASSWORD=abcd1234efgh5678
```

### SMTP Configuration (AWS SES)

```yaml
services:
  grafana:
    environment:
      - GF_SMTP_ENABLED=true
      - GF_SMTP_HOST=email-smtp.us-east-1.amazonaws.com:587
      - GF_SMTP_USER=${AWS_SES_SMTP_USER}
      - GF_SMTP_PASSWORD=${AWS_SES_SMTP_PASSWORD}
      - GF_SMTP_FROM_ADDRESS=alerts@goimg.dev
      - GF_SMTP_FROM_NAME=goimg Security Alerts
```

AWS SES Setup:
1. Verify domain in SES console
2. Create SMTP credentials (IAM user)
3. Store credentials in `.env`

### Test Email Delivery

```bash
# Test via curl
curl -X POST http://localhost:3000/api/alertmanager/grafana/api/v2/alerts/test \
  -H "Authorization: Bearer $GRAFANA_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "receivers": [{
      "name": "security-team-email",
      "grafana_managed_receiver_configs": [{
        "type": "email",
        "settings": {
          "addresses": "security@goimg.dev"
        }
      }]
    }],
    "alerts": [{
      "labels": {"alertname": "Test", "severity": "critical"},
      "annotations": {"summary": "Test email notification"}
    }]
  }'
```

---

## Slack Notifications

### Webhook Setup

1. Create Slack App:
   - Go to https://api.slack.com/apps
   - Click "Create New App" → "From scratch"
   - Name: "goimg Security Alerts"
   - Workspace: Your workspace

2. Enable Incoming Webhooks:
   - Click "Incoming Webhooks" in sidebar
   - Toggle "Activate Incoming Webhooks" to ON
   - Click "Add New Webhook to Workspace"
   - Select channel: `#security-alerts`
   - Click "Allow"

3. Copy Webhook URL:
   ```
   https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX
   ```

4. Set in `.env`:
   ```bash
   SLACK_WEBHOOK_URL=https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX
   ```

### Docker Compose Configuration

```yaml
# docker-compose.prod.yml
services:
  grafana:
    environment:
      - SLACK_WEBHOOK_URL=${SLACK_WEBHOOK_URL}
```

### Custom Slack Message Format

```yaml
# contact_points.yml
- orgId: 1
  name: security-slack-channel
  receivers:
    - uid: slack-security
      type: slack
      settings:
        url: ${SLACK_WEBHOOK_URL}
        recipient: '#security-alerts'
        title: |
          {{ if eq .Status "firing" }}:rotating_light: ALERT{{ else }}:white_check_mark: RESOLVED{{ end }}: {{ .CommonLabels.alertname }}
        text: |
          *Severity:* {{ .CommonLabels.severity | upper }}
          *Category:* {{ .CommonLabels.subcategory }}

          {{ range .Alerts }}
          {{ .Annotations.description }}

          {{ if .Annotations.runbook_url }}:book: <{{ .Annotations.runbook_url }}|View Runbook>{{ end }}
          {{ end }}
        username: goimg Security Monitor
        icon_emoji: ':shield:'
```

### Test Slack Webhook

```bash
# Test directly
curl -X POST "${SLACK_WEBHOOK_URL}" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Test alert from goimg security monitoring",
    "username": "goimg Security Monitor",
    "icon_emoji": ":shield:"
  }'

# Expected: Message appears in #security-alerts channel
```

---

## PagerDuty Notifications

### Integration Setup

1. Create PagerDuty Service:
   - Go to https://[your-subdomain].pagerduty.com
   - Services → Service Directory
   - Click "New Service"
   - Name: "goimg Security Alerts"
   - Escalation Policy: Select or create
   - Integration: Events API v2
   - Click "Create Service"

2. Get Integration Key:
   - Go to service settings
   - Click "Integrations" tab
   - Find "Events API v2" integration
   - Copy integration key:
   ```
   r1234567890abcdef1234567890abcdef
   ```

3. Set in `.env`:
   ```bash
   PAGERDUTY_INTEGRATION_KEY=r1234567890abcdef1234567890abcdef
   ```

### Docker Compose Configuration

```yaml
# docker-compose.prod.yml
services:
  grafana:
    environment:
      - PAGERDUTY_INTEGRATION_KEY=${PAGERDUTY_INTEGRATION_KEY}
```

### PagerDuty Alert Routing

Only critical alerts (P0/P1) trigger PagerDuty:

```yaml
# notification_policies.yml
routes:
  # P0 alerts (Malware) → PagerDuty
  - receiver: security-pagerduty
    matchers:
      - severity = critical
      - priority = P0
    group_wait: 10s
    repeat_interval: 30m

  # P1 alerts (Privilege Escalation) → PagerDuty
  - receiver: security-pagerduty
    matchers:
      - severity = critical
      - priority = P1
    group_wait: 30s
    repeat_interval: 1h
```

### Test PagerDuty Integration

```bash
# Test Events API v2
curl -X POST https://events.pagerduty.com/v2/enqueue \
  -H "Content-Type: application/json" \
  -d '{
    "routing_key": "'${PAGERDUTY_INTEGRATION_KEY}'",
    "event_action": "trigger",
    "payload": {
      "summary": "Test alert from goimg security monitoring",
      "severity": "critical",
      "source": "goimg-api",
      "component": "security",
      "custom_details": {
        "alert_name": "Test Alert",
        "description": "This is a test alert to verify PagerDuty integration"
      }
    }
  }'

# Expected: Incident created, on-call engineer paged
```

### Resolve Incident Programmatically

```bash
# Resolve incident
curl -X POST https://events.pagerduty.com/v2/enqueue \
  -H "Content-Type: application/json" \
  -d '{
    "routing_key": "'${PAGERDUTY_INTEGRATION_KEY}'",
    "event_action": "resolve",
    "dedup_key": "alert-uid-12345"
  }'
```

---

## Microsoft Teams Notifications

### Webhook Setup

1. Open Teams Channel:
   - Go to desired channel (e.g., "Security Alerts")
   - Click "..." (More options)
   - Select "Connectors"

2. Configure Incoming Webhook:
   - Search for "Incoming Webhook"
   - Click "Configure"
   - Name: "goimg Security Alerts"
   - Upload icon (optional)
   - Click "Create"

3. Copy Webhook URL:
   ```
   https://outlook.office.com/webhook/a1234567-89ab-cdef-0123-456789abcdef@...
   ```

4. Set in `.env`:
   ```bash
   TEAMS_WEBHOOK_URL=https://outlook.office.com/webhook/a1234567-89ab-cdef-0123-456789abcdef@...
   ```

### Docker Compose Configuration

```yaml
# docker-compose.prod.yml
services:
  grafana:
    environment:
      - TEAMS_WEBHOOK_URL=${TEAMS_WEBHOOK_URL}
```

### Test Teams Webhook

```bash
# Test MessageCard format
curl -X POST "${TEAMS_WEBHOOK_URL}" \
  -H "Content-Type: application/json" \
  -d '{
    "@type": "MessageCard",
    "@context": "http://schema.org/extensions",
    "themeColor": "d32f2f",
    "summary": "Test Alert",
    "sections": [{
      "activityTitle": "Test alert from goimg security monitoring",
      "activitySubtitle": "Severity: CRITICAL",
      "facts": [
        {"name": "Category", "value": "security"},
        {"name": "Status", "value": "firing"}
      ],
      "text": "This is a test alert to verify Microsoft Teams integration."
    }]
  }'

# Expected: Card appears in Teams channel
```

---

## Webhook / SIEM Integration

### Generic Webhook

```yaml
# contact_points.yml
- orgId: 1
  name: security-webhook
  receivers:
    - uid: webhook-security
      type: webhook
      settings:
        url: ${SECURITY_WEBHOOK_URL}
        httpMethod: POST
        maxAlerts: 0
        # Send full alert payload as JSON
```

### Splunk HTTP Event Collector (HEC)

```bash
# .env
SECURITY_WEBHOOK_URL=https://splunk.goimg.dev:8088/services/collector/event
SPLUNK_HEC_TOKEN=12345678-1234-1234-1234-123456789012
```

```yaml
# contact_points.yml
- orgId: 1
  name: splunk-hec
  receivers:
    - uid: webhook-splunk
      type: webhook
      settings:
        url: ${SECURITY_WEBHOOK_URL}
        httpMethod: POST
        authorization_scheme: Splunk
        authorization_credentials: ${SPLUNK_HEC_TOKEN}
        maxAlerts: 0
```

### Elastic Security (Elasticsearch)

```bash
# .env
SECURITY_WEBHOOK_URL=https://elasticsearch.goimg.dev:9200/security-alerts/_doc
ELASTIC_API_KEY=VnVhQ2ZHY0JDZGJrUW0tZTVhT3g6dWkybHAyYXhUTm1zeWFrdzl0dk5udw==
```

```yaml
# contact_points.yml
- orgId: 1
  name: elasticsearch
  receivers:
    - uid: webhook-elastic
      type: webhook
      settings:
        url: ${SECURITY_WEBHOOK_URL}
        httpMethod: POST
        authorization_scheme: ApiKey
        authorization_credentials: ${ELASTIC_API_KEY}
        maxAlerts: 0
```

### Custom SIEM Payload Format

```yaml
# notification_templates.yml
- name: siem_json_template
  template: |
    {
      "timestamp": "{{ .FiringAt }}",
      "alert_name": "{{ .CommonLabels.alertname }}",
      "severity": "{{ .CommonLabels.severity }}",
      "category": "{{ .CommonLabels.category }}",
      "subcategory": "{{ .CommonLabels.subcategory }}",
      "status": "{{ .Status }}",
      "alerts": [
        {{ range $i, $alert := .Alerts }}
        {{ if $i }},{{ end }}
        {
          "description": "{{ $alert.Annotations.description }}",
          "labels": {{ $alert.Labels | toJson }},
          "started_at": "{{ $alert.StartsAt }}",
          "ends_at": "{{ $alert.EndsAt }}"
        }
        {{ end }}
      ]
    }
```

---

## Discord Notifications

### Webhook Setup

1. Open Discord Server Settings:
   - Go to server → Server Settings
   - Click "Integrations"
   - Click "Webhooks"
   - Click "New Webhook"

2. Configure Webhook:
   - Name: "goimg Security Alerts"
   - Channel: #security-alerts
   - Copy webhook URL

3. Set in `.env`:
   ```bash
   DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/123456789012345678/abcdefghijklmnopqrstuvwxyz
   ```

### Discord Webhook Configuration

```yaml
# contact_points.yml
- orgId: 1
  name: discord-alerts
  receivers:
    - uid: webhook-discord
      type: webhook
      settings:
        url: ${DISCORD_WEBHOOK_URL}
        httpMethod: POST
        maxAlerts: 10
```

### Test Discord Webhook

```bash
curl -X POST "${DISCORD_WEBHOOK_URL}" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Test alert from goimg security monitoring",
    "username": "goimg Security Monitor",
    "embeds": [{
      "title": "Security Alert",
      "description": "This is a test alert",
      "color": 15158332,
      "fields": [
        {"name": "Severity", "value": "CRITICAL", "inline": true},
        {"name": "Status", "value": "Firing", "inline": true}
      ]
    }]
  }'
```

---

## Multi-Channel Configuration

### Example: Critical Alerts to All Channels

```yaml
# notification_policies.yml
routes:
  # P0 Critical: PagerDuty + Slack + Email
  - receiver: security-pagerduty
    matchers:
      - severity = critical
      - priority = P0
    continue: true  # Continue to next routes

  - receiver: security-slack-channel
    matchers:
      - severity = critical
      - priority = P0
    continue: true

  - receiver: security-team-email
    matchers:
      - severity = critical
      - priority = P0
    continue: false  # Stop here
```

### Business Hours vs After Hours

```yaml
# notification_policies.yml
mute_time_intervals:
  - name: business-hours
    time_intervals:
      - times:
          - start_time: '09:00'
            end_time: '17:00'
        weekdays: ['monday', 'tuesday', 'wednesday', 'thursday', 'friday']

routes:
  # After hours: Page on-call
  - receiver: security-pagerduty
    matchers:
      - severity = critical
    mute_time_intervals:
      - business-hours  # Don't page during business hours

  # Business hours: Slack only
  - receiver: security-slack-channel
    matchers:
      - severity = critical
```

---

## Security Best Practices

### Credential Management

1. **Never commit secrets to git**:
   ```bash
   # .gitignore
   .env
   docker-compose.prod.yml  # If it contains secrets
   ```

2. **Use environment variables**:
   ```yaml
   # Good
   url: ${SLACK_WEBHOOK_URL}

   # Bad
   url: https://hooks.slack.com/services/T00/B00/XXX
   ```

3. **Rotate credentials quarterly**:
   - Create new webhook URLs
   - Update `.env` file
   - Restart Grafana
   - Verify alerts still work
   - Deactivate old webhooks

### Webhook URL Protection

1. **Use HTTPS only** (never HTTP)
2. **Validate webhook signatures** (if supported)
3. **Restrict IP access** (firewall rules)
4. **Monitor webhook access logs**
5. **Set rate limits on webhook endpoints**

### Testing in Production

```bash
# Use label to differentiate test alerts
curl -X POST http://localhost:3000/api/alertmanager/grafana/api/v2/alerts \
  -H "Authorization: Bearer $GRAFANA_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "labels": {
      "alertname": "TEST - Ignore",
      "severity": "info",
      "test": "true"
    },
    "annotations": {
      "summary": "This is a test alert - please ignore"
    }
  }'
```

---

## Troubleshooting

### Email Not Sending

```bash
# Check SMTP settings
docker exec goimg-grafana grep -A 10 "\[smtp\]" /etc/grafana/grafana.ini

# Test SMTP connection
docker exec goimg-grafana nc -zv smtp.gmail.com 587

# Check Grafana logs
docker logs goimg-grafana | grep -i smtp
```

### Slack Webhook 404

- Verify webhook URL is correct
- Check if webhook was deleted in Slack
- Confirm app is installed in workspace
- Test webhook with curl

### PagerDuty Not Paging

- Verify integration key is correct
- Check escalation policy is active
- Ensure on-call schedule has assigned user
- Verify user has phone number / push notifications enabled
- Check PagerDuty service status

---

## Additional Resources

- [Grafana Contact Points Documentation](https://grafana.com/docs/grafana/latest/alerting/manage-notifications/create-contact-point/)
- [Slack Incoming Webhooks Guide](https://api.slack.com/messaging/webhooks)
- [PagerDuty Events API v2](https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgw-events-api-v2-overview)
- [Microsoft Teams Message Cards](https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference#office-365-connector-card)
- [Contact Points Configuration](contact_points.yml)

---

**Document Version**: 1.0
**Last Updated**: 2025-12-06
**Maintainer**: Security Team (security@goimg.dev)
