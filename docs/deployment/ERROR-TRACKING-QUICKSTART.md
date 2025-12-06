# Error Tracking - Quick Start Guide

> 15-minute setup guide for error tracking integration

**For Full Documentation**: See `/home/user/goimg-datalayer/docs/deployment/error-tracking.md`

---

## Choose Your Solution

| Solution | Best For | Setup Time | Cost |
|----------|----------|------------|------|
| **GlitchTip (Self-Hosted)** | Data sovereignty, privacy-sensitive apps | 15 min | Free (infrastructure only) |
| **Sentry (Cloud)** | Quick setup, managed service | 5 min | Free tier: 10k errors/month |

**Recommendation for goimg-datalayer**: **GlitchTip** (image gallery is privacy-sensitive)

---

## Option A: GlitchTip (Self-Hosted)

### 1. Generate Secrets (30 seconds)

```bash
cd /home/user/goimg-datalayer

# Generate secrets
export GLITCHTIP_SECRET_KEY=$(openssl rand -hex 32)
export GLITCHTIP_DB_PASSWORD=$(openssl rand -base64 32)

# Add to .env.prod
cat >> docker/.env.prod <<EOF
GLITCHTIP_SECRET_KEY=$GLITCHTIP_SECRET_KEY
GLITCHTIP_DB_PASSWORD=$GLITCHTIP_DB_PASSWORD
GLITCHTIP_DOMAIN=https://glitchtip.goimg.example.com
GLITCHTIP_ADMIN_EMAIL=admin@goimg.example.com
EOF
```

### 2. Deploy GlitchTip (2 minutes)

```bash
cd /home/user/goimg-datalayer/docker

# Start GlitchTip stack
docker-compose -f docker-compose.glitchtip.yml up -d

# Wait for services to be healthy (30 seconds)
docker-compose -f docker-compose.glitchtip.yml ps

# Create admin user
docker exec -it goimg-glitchtip-web ./manage.py createsuperuser
# Username: admin
# Email: admin@goimg.example.com
# Password: [create strong password]
```

### 3. Get DSN (1 minute)

1. Open browser: http://localhost:8000 (or https://glitchtip.goimg.example.com)
2. Login with admin credentials
3. Create Organization: "goimg"
4. Create Team: "Engineering"
5. Create Project: "goimg-api" (Platform: Go)
6. Copy DSN from Settings → Client Keys (DSN)
7. Add to `.env.prod`:

```bash
echo "SENTRY_DSN=https://YOUR_KEY@glitchtip.goimg.example.com/1" >> docker/.env.prod
```

### 4. Configure App (3 minutes)

Add to `docker/.env.prod`:

```bash
# Error Tracking
SENTRY_ENVIRONMENT=production
SENTRY_RELEASE=goimg-api@1.0.0
SENTRY_SAMPLE_RATE=1.0
SENTRY_TRACES_SAMPLE_RATE=0.1
```

---

## Option B: Sentry (Cloud)

### 1. Sign Up (2 minutes)

1. Go to https://sentry.io
2. Sign up (free tier: 10,000 errors/month)
3. Create project: Platform = Go
4. Copy DSN

### 2. Configure App (1 minute)

Add to `docker/.env.prod`:

```bash
# Error Tracking
SENTRY_DSN=https://YOUR_KEY@YOUR_ORG.ingest.sentry.io/YOUR_PROJECT
SENTRY_ENVIRONMENT=production
SENTRY_RELEASE=goimg-api@1.0.0
SENTRY_SAMPLE_RATE=1.0
SENTRY_TRACES_SAMPLE_RATE=0.1
```

---

## Code Integration (10 minutes)

### 1. Install SDK (30 seconds)

```bash
cd /home/user/goimg-datalayer
go get github.com/getsentry/sentry-go
```

### 2. Create Error Tracking Package (3 minutes)

Create file: `/home/user/goimg-datalayer/internal/infrastructure/errortracking/sentry.go`

```go
package errortracking

import (
	"fmt"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

type Config struct {
	DSN              string
	Environment      string
	Release          string
	SampleRate       float64
	TracesSampleRate float64
	Debug            bool
}

func InitSentry(cfg Config, logger *zerolog.Logger) error {
	if cfg.DSN == "" {
		logger.Warn().Msg("SENTRY_DSN not configured, error tracking disabled")
		return nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		SampleRate:       cfg.SampleRate,
		TracesSampleRate: cfg.TracesSampleRate,
		Debug:            cfg.Debug,
		EnableTracing:    true,
		ServerName:       getHostname(),
	})

	if err != nil {
		return fmt.Errorf("failed to initialize sentry: %w", err)
	}

	logger.Info().
		Str("environment", cfg.Environment).
		Str("release", cfg.Release).
		Msg("error tracking initialized")

	return nil
}

func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}
```

### 3. Initialize in main.go (2 minutes)

Update `/home/user/goimg-datalayer/cmd/api/main.go`:

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"goimg-datalayer/internal/infrastructure/errortracking"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Initialize error tracking
	sentryConfig := errortracking.Config{
		DSN:              os.Getenv("SENTRY_DSN"),
		Environment:      os.Getenv("SENTRY_ENVIRONMENT"),
		Release:          os.Getenv("SENTRY_RELEASE"),
		SampleRate:       1.0,
		TracesSampleRate: 0.1,
		Debug:            os.Getenv("SENTRY_ENVIRONMENT") == "development",
	}

	if err := errortracking.InitSentry(sentryConfig, &logger); err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize error tracking")
	}

	defer errortracking.Flush(5 * time.Second)

	// ... rest of application ...
}
```

### 4. Enhance Recovery Middleware (5 minutes)

Update `/home/user/goimg-datalayer/internal/interfaces/http/middleware/recovery.go`:

Add import:
```go
import (
	"github.com/getsentry/sentry-go"
)
```

In the `defer func()` block, add after logging:

```go
// Capture to Sentry/GlitchTip
hub := sentry.GetHubFromContext(r.Context())
if hub == nil {
	hub = sentry.CurrentHub().Clone()
}

hub.ConfigureScope(func(scope *sentry.Scope) {
	scope.SetTag("request_id", requestID)
	scope.SetTag("http.method", r.Method)
	scope.SetTag("http.path", r.URL.Path)
	scope.SetUser(sentry.User{ID: userID})
	scope.SetRequest(r)
	scope.SetLevel(sentry.LevelFatal)
})

eventID := hub.RecoverWithContext(r.Context(), err)
logger.Info().Str("sentry_event_id", string(*eventID)).Msg("panic reported to error tracking")
```

---

## Test It (2 minutes)

### 1. Create Test Panic

Add temporary test endpoint:

```go
// In your router setup
r.Get("/test-error", func(w http.ResponseWriter, r *http.Request) {
	panic("Test panic for error tracking")
})
```

### 2. Trigger Error

```bash
curl http://localhost:8080/test-error
```

### 3. Verify in Dashboard

1. Open GlitchTip/Sentry dashboard
2. Navigate to Issues
3. You should see: "Test panic for error tracking"
4. Click issue to see:
   - Stack trace
   - Request context (method, path, request ID)
   - User information
   - Breadcrumbs

### 4. Remove Test Endpoint

Delete the test endpoint after verification.

---

## Configure Alerts (5 minutes)

### Email Alerts

**GlitchTip**:
1. Settings → Alerts → New Alert
2. Trigger: "A new issue is created"
3. Environment: Production
4. Send to: team@goimg.example.com

**Sentry**:
1. Settings → Alerts → Create Alert
2. When: "The issue is first seen"
3. Environment: production
4. Action: Send notification to email

### Slack Integration

**GlitchTip**:
1. Create Slack incoming webhook: https://api.slack.com/apps
2. Settings → Integrations → Webhook
3. Add webhook URL
4. Channel: #engineering-alerts

**Sentry**:
1. Settings → Integrations → Slack
2. Connect workspace
3. Choose channel: #engineering-alerts
4. Configure alert rules

---

## PII Scrubbing (REQUIRED for Production)

### Quick Implementation

Add to `errortracking/sentry.go`:

```go
import "regexp"

var (
	emailRegex    = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	passwordRegex = regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token)["\s:=]+[^\s"&]+`)
)

func scrubPII(event *sentry.Event) *sentry.Event {
	// Scrub exception messages
	for i := range event.Exception {
		event.Exception[i].Value = scrubString(event.Exception[i].Value)
	}

	// Scrub log message
	if event.Message != "" {
		event.Message = scrubString(event.Message)
	}

	// Remove email from user
	event.User.Email = ""
	event.User.Username = ""

	return event
}

func scrubString(s string) string {
	s = emailRegex.ReplaceAllString(s, "[EMAIL_REDACTED]")
	s = passwordRegex.ReplaceAllString(s, "$1=[REDACTED]")
	return s
}
```

Add to `sentry.Init()` options:

```go
sentry.Init(sentry.ClientOptions{
	// ... existing options ...
	BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
		return scrubPII(event)
	},
})
```

**For complete PII scrubbing**: See `/home/user/goimg-datalayer/docs/deployment/error-tracking.md` section "PII Scrubbing Configuration"

---

## Verification Checklist

- [ ] Error tracking deployed (GlitchTip or Sentry)
- [ ] DSN configured in `.env.prod`
- [ ] Go SDK installed (`go get github.com/getsentry/sentry-go`)
- [ ] SDK initialized in `main.go`
- [ ] Recovery middleware enhanced with Sentry capture
- [ ] PII scrubbing implemented in `BeforeSend` hook
- [ ] Test error triggered and appeared in dashboard
- [ ] Alerts configured (email and/or Slack)
- [ ] Test endpoint removed
- [ ] Team trained on dashboard usage

---

## Common Issues

### Errors Not Appearing

**Check 1**: Is DSN set?
```bash
echo $SENTRY_DSN
# Should output: https://...
```

**Check 2**: Is SDK initialized?
```bash
docker logs goimg-api 2>&1 | grep "error tracking initialized"
# Should show: "error tracking initialized"
```

**Check 3**: Test DSN connectivity
```bash
curl -X POST "https://YOUR_ORG.ingest.sentry.io/api/1/store/" \
  -H "X-Sentry-Auth: Sentry sentry_version=7, sentry_key=YOUR_KEY" \
  -d '{"message": "test"}'
# Should return: HTTP 200
```

### GlitchTip Not Starting

```bash
# Check logs
docker-compose -f docker-compose.glitchtip.yml logs glitchtip-web

# Common issues:
# - Database not ready: Wait 30 seconds
# - SECRET_KEY not set: Check .env.prod
# - Migration failed: Run manually:
docker exec -it goimg-glitchtip-web ./manage.py migrate
```

---

## Next Steps

1. **Review full documentation**: `/home/user/goimg-datalayer/docs/deployment/error-tracking.md`
2. **Implement complete PII scrubbing**: See section 6 in main docs
3. **Configure performance monitoring**: Transaction tracing for slow endpoints
4. **Create runbooks**: Document response procedures for common errors
5. **Train team**: Workshop on dashboard usage and issue triage

---

## Support

**Documentation**: `/home/user/goimg-datalayer/docs/deployment/error-tracking.md`
**Integration Examples**: `/home/user/goimg-datalayer/docs/deployment/error-tracking-integration-example.go`
**Docker Compose**: `/home/user/goimg-datalayer/docker/docker-compose.glitchtip.yml`

**External Resources**:
- Sentry Go SDK: https://docs.sentry.io/platforms/go/
- GlitchTip Docs: https://glitchtip.com/documentation
- Sentry Go GitHub: https://github.com/getsentry/sentry-go

---

**Setup Time**: 15 minutes
**Status**: Ready to implement
**Security Gate**: S9-MON-002
