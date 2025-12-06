# Error Tracking Setup Guide

> Production error tracking integration for goimg-datalayer (Security Gate S9-MON-002)

**Document Status**: DRAFT
**Security Gate**: S9-MON-002 (Error Tracking Integration)
**Priority**: P1 (Launch Blocker)
**Last Updated**: 2025-12-06

---

## Table of Contents

1. [Overview](#overview)
2. [Solution Options](#solution-options)
3. [Sentry Integration (Cloud)](#sentry-integration-cloud)
4. [GlitchTip Integration (Self-Hosted)](#glitchtip-integration-self-hosted)
5. [Go SDK Integration](#go-sdk-integration)
6. [PII Scrubbing Configuration](#pii-scrubbing-configuration)
7. [Operational Procedures](#operational-procedures)
8. [Performance Impact](#performance-impact)
9. [Security Considerations](#security-considerations)

---

## Overview

Error tracking provides real-time visibility into production errors, enabling rapid detection, diagnosis, and resolution of issues before they impact users at scale.

### Why Error Tracking is Required

**Security Gate S9-MON-002** requires error tracking integration to:

1. **Detect Security Issues**: Identify failed authentication, authorization bypass attempts, validation failures
2. **Monitor Application Health**: Track error rates, patterns, and trends
3. **Enable Rapid Response**: Alert on-call engineers to critical failures within minutes
4. **Support Forensics**: Preserve error context for security incident investigation
5. **Compliance**: Document system stability for SOC 2 CC7.2 (System Monitoring)

### Key Features Required

- **Error Capture**: Automatic capture of all unhandled errors and panics
- **Context Enrichment**: Request ID, user ID, IP address, user agent, tags
- **PII Scrubbing**: Automatic removal of sensitive data (passwords, tokens, emails)
- **Sampling**: 100% errors, 10% transactions/performance monitoring
- **Alerting**: Notifications for error spikes and new error types
- **Release Tracking**: Correlate errors to specific deployments

---

## Solution Options

### Option 1: Sentry (Cloud SaaS)

**Pros**:
- Managed infrastructure (no maintenance)
- Advanced features (performance monitoring, session replay)
- 10,000 errors/month free tier
- Excellent Go SDK support
- Real-time alerting and integrations (Slack, PagerDuty)

**Cons**:
- Data leaves your infrastructure (GDPR considerations)
- Cost scales with volume (~$26/month for 50k errors)
- Vendor lock-in

**Best For**: Teams prioritizing ease-of-use and advanced features

---

### Option 2: GlitchTip (Self-Hosted)

**Pros**:
- 100% open source (MIT license)
- Sentry-compatible API (drop-in replacement)
- Full data control (GDPR-friendly)
- No per-event pricing
- Self-hosted on your infrastructure

**Cons**:
- Requires infrastructure maintenance
- Fewer features than Sentry (no session replay, limited performance monitoring)
- Needs PostgreSQL + Redis

**Best For**: Teams with data sovereignty requirements or budget constraints

---

### Recommendation

**Use GlitchTip for goimg-datalayer** because:

1. **Data Sovereignty**: Image gallery data is user-generated and privacy-sensitive
2. **Cost Predictability**: No surprise bills with traffic spikes
3. **Existing Infrastructure**: We already run PostgreSQL and Redis
4. **Compliance**: Easier GDPR Article 32 compliance with on-premise data
5. **Sentry Compatibility**: Can migrate to Sentry later if needed (same SDK)

---

## Sentry Integration (Cloud)

If you choose Sentry as your error tracking provider:

### 1. Create Sentry Project

```bash
# Sign up at https://sentry.io
# Create new project: Go
# Copy DSN: https://abc123@o123456.ingest.sentry.io/789012
```

### 2. Install Go SDK

```bash
go get github.com/getsentry/sentry-go
```

### 3. Environment Configuration

Add to `/home/user/goimg-datalayer/docker/.env.prod.example`:

```bash
# Sentry Error Tracking
SENTRY_DSN=https://YOUR_KEY@YOUR_ORG.ingest.sentry.io/YOUR_PROJECT
SENTRY_ENVIRONMENT=production
SENTRY_RELEASE=goimg-api@1.0.0
SENTRY_SAMPLE_RATE=1.0          # 100% of errors
SENTRY_TRACES_SAMPLE_RATE=0.1   # 10% of transactions
```

### 4. Integration Code

See [Go SDK Integration](#go-sdk-integration) section below.

### 5. Configure Alerts

In Sentry dashboard:

1. **Settings → Alerts → New Alert Rule**
2. **When**: "The issue is first seen" OR "Number of events > 10 in 1 minute"
3. **Then**: Send notification to Slack/PagerDuty/Email
4. **For**: All environments OR production only

---

## GlitchTip Integration (Self-Hosted)

### 1. Architecture

```
┌─────────────────┐
│   goimg-api     │
│  (Sentry SDK)   │
└────────┬────────┘
         │ HTTP POST /api/1/envelope/
         ▼
┌─────────────────┐
│   GlitchTip     │
│  (Django app)   │
├─────────────────┤
│  PostgreSQL DB  │
│  Redis Cache    │
└─────────────────┘
```

### 2. Deploy GlitchTip with Docker Compose

Create `/home/user/goimg-datalayer/docker/docker-compose.glitchtip.yml`:

```yaml
version: "3.9"

services:
  # PostgreSQL database for GlitchTip
  glitchtip-postgres:
    image: postgres:16-alpine
    container_name: goimg-glitchtip-postgres
    environment:
      POSTGRES_USER: glitchtip
      POSTGRES_PASSWORD: ${GLITCHTIP_DB_PASSWORD}
      POSTGRES_DB: glitchtip
    volumes:
      - glitchtip_postgres_data:/var/lib/postgresql/data
    networks:
      - glitchtip-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U glitchtip"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  # Redis for caching and Celery task queue
  glitchtip-redis:
    image: redis:7-alpine
    container_name: goimg-glitchtip-redis
    networks:
      - glitchtip-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  # GlitchTip web application
  glitchtip-web:
    image: glitchtip/glitchtip:latest
    container_name: goimg-glitchtip-web
    depends_on:
      glitchtip-postgres:
        condition: service_healthy
      glitchtip-redis:
        condition: service_healthy
    environment:
      # Database configuration
      DATABASE_URL: postgresql://glitchtip:${GLITCHTIP_DB_PASSWORD}@glitchtip-postgres:5432/glitchtip
      REDIS_URL: redis://glitchtip-redis:6379/0

      # Security settings
      SECRET_KEY: ${GLITCHTIP_SECRET_KEY}  # Generate with: openssl rand -hex 32
      ENABLE_OPEN_USER_REGISTRATION: "False"

      # Email configuration (for alerts)
      EMAIL_BACKEND: django.core.mail.backends.smtp.EmailBackend
      EMAIL_HOST: ${SMTP_HOST}
      EMAIL_PORT: ${SMTP_PORT}
      EMAIL_HOST_USER: ${SMTP_USER}
      EMAIL_HOST_PASSWORD: ${SMTP_PASSWORD}
      EMAIL_USE_TLS: "True"
      DEFAULT_FROM_EMAIL: alerts@goimg.example.com

      # Application settings
      GLITCHTIP_DOMAIN: https://glitchtip.goimg.example.com
      ENABLE_ORGANIZATION_CREATION: "True"
      MAINTENANCE_EVENT_FREEZE: "False"

      # Performance
      WEB_CONCURRENCY: "4"  # Gunicorn workers

    ports:
      - "8000:8000"
    networks:
      - glitchtip-network
      - goimg-network  # Connect to main app network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/api/0/"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
    restart: unless-stopped

  # Celery worker for async tasks (email sending, cleanup)
  glitchtip-worker:
    image: glitchtip/glitchtip:latest
    container_name: goimg-glitchtip-worker
    command: ./bin/run-celery-with-beat.sh
    depends_on:
      glitchtip-postgres:
        condition: service_healthy
      glitchtip-redis:
        condition: service_healthy
    environment:
      DATABASE_URL: postgresql://glitchtip:${GLITCHTIP_DB_PASSWORD}@glitchtip-postgres:5432/glitchtip
      REDIS_URL: redis://glitchtip-redis:6379/0
      SECRET_KEY: ${GLITCHTIP_SECRET_KEY}
      EMAIL_BACKEND: django.core.mail.backends.smtp.EmailBackend
      EMAIL_HOST: ${SMTP_HOST}
      EMAIL_PORT: ${SMTP_PORT}
      EMAIL_HOST_USER: ${SMTP_USER}
      EMAIL_HOST_PASSWORD: ${SMTP_PASSWORD}
      EMAIL_USE_TLS: "True"
    networks:
      - glitchtip-network
    restart: unless-stopped

volumes:
  glitchtip_postgres_data:
    driver: local

networks:
  glitchtip-network:
    name: glitchtip-network
  goimg-network:
    external: true
    name: goimg-network
```

### 3. Environment Variables

Add to `/home/user/goimg-datalayer/docker/.env.prod.example`:

```bash
# GlitchTip Error Tracking (Self-Hosted)
GLITCHTIP_DB_PASSWORD=your-secure-database-password-here
GLITCHTIP_SECRET_KEY=your-64-char-random-secret-key-generate-with-openssl
SENTRY_DSN=https://YOUR_KEY@glitchtip.goimg.example.com/1

# SMTP for GlitchTip alerts
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASSWORD=SG.your-sendgrid-api-key

# Sentry SDK Configuration (same for Sentry or GlitchTip)
SENTRY_ENVIRONMENT=production
SENTRY_RELEASE=goimg-api@1.0.0
SENTRY_SAMPLE_RATE=1.0
SENTRY_TRACES_SAMPLE_RATE=0.1
```

### 4. Deploy GlitchTip

```bash
# Generate secure secrets
export GLITCHTIP_SECRET_KEY=$(openssl rand -hex 32)
export GLITCHTIP_DB_PASSWORD=$(openssl rand -base64 32)

# Add to .env file
echo "GLITCHTIP_SECRET_KEY=$GLITCHTIP_SECRET_KEY" >> docker/.env.prod
echo "GLITCHTIP_DB_PASSWORD=$GLITCHTIP_DB_PASSWORD" >> docker/.env.prod

# Start GlitchTip
cd /home/user/goimg-datalayer/docker
docker-compose -f docker-compose.glitchtip.yml up -d

# Check status
docker-compose -f docker-compose.glitchtip.yml ps

# View logs
docker-compose -f docker-compose.glitchtip.yml logs -f glitchtip-web

# Create superuser account
docker exec -it goimg-glitchtip-web ./manage.py createsuperuser
```

### 5. Initial Setup

1. **Access GlitchTip**: http://localhost:8000 (or https://glitchtip.goimg.example.com)
2. **Login**: Use superuser credentials
3. **Create Organization**: "goimg"
4. **Create Team**: "Engineering"
5. **Create Project**: "goimg-api" (Platform: Go)
6. **Copy DSN**: Settings → Client Keys (DSN)
7. **Configure Alerts**: Settings → Alerts → New Alert

### 6. Nginx Reverse Proxy Configuration

Add to `/home/user/goimg-datalayer/docker/nginx/conf.d/glitchtip.conf`:

```nginx
upstream glitchtip {
    server glitchtip-web:8000;
}

server {
    listen 443 ssl http2;
    server_name glitchtip.goimg.example.com;

    ssl_certificate /etc/letsencrypt/live/glitchtip.goimg.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/glitchtip.goimg.example.com/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    client_max_body_size 10M;

    location / {
        proxy_pass http://glitchtip;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support for live updates
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name glitchtip.goimg.example.com;
    return 301 https://$server_name$request_uri;
}
```

---

## Go SDK Integration

Both Sentry and GlitchTip use the same Go SDK (Sentry SDK is compatible with GlitchTip).

### 1. Install Dependencies

```bash
cd /home/user/goimg-datalayer
go get github.com/getsentry/sentry-go
```

### 2. Initialize Sentry in main.go

Create `/home/user/goimg-datalayer/internal/infrastructure/errortracking/sentry.go`:

```go
package errortracking

import (
	"fmt"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

// Config holds error tracking configuration
type Config struct {
	DSN              string
	Environment      string
	Release          string
	SampleRate       float64
	TracesSampleRate float64
	Debug            bool
}

// InitSentry initializes Sentry/GlitchTip error tracking
func InitSentry(cfg Config, logger *zerolog.Logger) error {
	if cfg.DSN == "" {
		logger.Warn().Msg("Sentry DSN not configured, error tracking disabled")
		return nil
	}

	err := sentry.Init(sentry.ClientOptions{
		// Connection
		Dsn: cfg.DSN,

		// Environment and release tracking
		Environment: cfg.Environment,
		Release:     cfg.Release,

		// Sampling configuration
		SampleRate:       cfg.SampleRate,       // 1.0 = 100% of errors
		TracesSampleRate: cfg.TracesSampleRate, // 0.1 = 10% of transactions

		// Error filtering
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Apply PII scrubbing (see PII section below)
			event = scrubbPII(event)

			// Filter out non-production environments if needed
			if cfg.Environment == "development" || cfg.Environment == "test" {
				return nil
			}

			return event
		},

		// Performance monitoring
		EnableTracing: true,

		// Debugging (only in dev)
		Debug: cfg.Debug,

		// Timeout for sending events
		ServerName: getHostname(),
		Transport:  sentry.NewHTTPSyncTransport(),
	})

	if err != nil {
		return fmt.Errorf("failed to initialize sentry: %w", err)
	}

	logger.Info().
		Str("environment", cfg.Environment).
		Str("release", cfg.Release).
		Float64("sample_rate", cfg.SampleRate).
		Msg("sentry error tracking initialized")

	return nil
}

// Flush ensures all pending events are sent before shutdown
func Flush(timeout time.Duration) {
	if !sentry.Flush(timeout) {
		// Log warning but don't fail shutdown
		fmt.Fprintf(os.Stderr, "Sentry flush timed out after %v\n", timeout)
	}
}

// getHostname returns the hostname for server identification
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// scrubbPII removes sensitive data from error events
func scrubbPII(event *sentry.Event) *sentry.Event {
	// See "PII Scrubbing Configuration" section below
	return event
}
```

### 3. Initialize in cmd/api/main.go

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"goimg-datalayer/internal/infrastructure/errortracking"
	// ... other imports
)

func main() {
	// Setup logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Load configuration
	cfg := loadConfig()

	// Initialize error tracking
	sentryConfig := errortracking.Config{
		DSN:              cfg.SentryDSN,
		Environment:      cfg.Environment,
		Release:          fmt.Sprintf("goimg-api@%s", cfg.Version),
		SampleRate:       1.0,  // 100% of errors
		TracesSampleRate: 0.1,  // 10% of transactions
		Debug:            cfg.Environment == "development",
	}

	if err := errortracking.InitSentry(sentryConfig, &logger); err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize error tracking")
	}

	// Ensure events are flushed on shutdown
	defer errortracking.Flush(5 * time.Second)

	// Run application
	if err := run(&logger, cfg); err != nil {
		logger.Fatal().Err(err).Msg("application failed")
	}
}

func run(logger *zerolog.Logger, cfg Config) error {
	// ... application initialization ...

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("http server shutdown error")
	}

	logger.Info().Msg("shutdown complete")
	return nil
}
```

### 4. Update Recovery Middleware

Enhance `/home/user/goimg-datalayer/internal/interfaces/http/middleware/recovery.go`:

```go
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

// Recovery middleware with Sentry integration
func Recovery(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					// Capture stack trace
					stackTrace := debug.Stack()

					// Get request context
					requestID := GetRequestID(r.Context())
					userID, _ := GetUserIDString(r.Context())

					// Build error message
					var errMsg string
					if err, ok := rvr.(error); ok {
						errMsg = err.Error()
					} else {
						errMsg = fmt.Sprintf("%v", rvr)
					}

					// Log to zerolog (existing behavior)
					logEvent := logger.Error().
						Str("request_id", requestID).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Str("remote_addr", getClientIP(r)).
						Str("panic", errMsg).
						Bytes("stack_trace", stackTrace)

					if userID != "" {
						logEvent = logEvent.Str("user_id", userID)
					}

					logEvent.Msg("panic recovered in http handler")

					// Capture to Sentry/GlitchTip
					hub := sentry.GetHubFromContext(r.Context())
					if hub == nil {
						hub = sentry.CurrentHub().Clone()
					}

					// Set context tags
					hub.ConfigureScope(func(scope *sentry.Scope) {
						scope.SetTag("request_id", requestID)
						scope.SetTag("http.method", r.Method)
						scope.SetTag("http.path", r.URL.Path)
						scope.SetUser(sentry.User{
							ID:        userID,
							IPAddress: getClientIP(r),
						})
						scope.SetRequest(r)
						scope.SetLevel(sentry.LevelFatal)
					})

					// Capture panic
					var err error
					if e, ok := rvr.(error); ok {
						err = e
					} else {
						err = fmt.Errorf("panic: %v", rvr)
					}

					eventID := hub.RecoverWithContext(r.Context(), err)

					logger.Info().
						Str("sentry_event_id", string(*eventID)).
						Msg("panic reported to error tracking")

					// Return generic error to client
					if !isResponseWritten(w) {
						WriteError(w, r,
							http.StatusInternalServerError,
							"Internal Server Error",
							"An unexpected error occurred. Please try again later.",
						)
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
```

### 5. Capture Errors in Handlers

Example handler with error capture:

```go
package handlers

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"

	"goimg-datalayer/internal/application/commands"
	"goimg-datalayer/internal/interfaces/http/middleware"
)

type ImageHandler struct {
	uploadImage *commands.UploadImageHandler
	logger      *zerolog.Logger
}

func (h *ImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ... parse request ...

	// Execute command
	image, err := h.uploadImage.Handle(ctx, cmd)
	if err != nil {
		// Capture non-user errors to Sentry
		if !isUserError(err) {
			hub := sentry.GetHubFromContext(ctx)
			if hub == nil {
				hub = sentry.CurrentHub().Clone()
			}

			hub.ConfigureScope(func(scope *sentry.Scope) {
				scope.SetTag("handler", "image.upload")
				scope.SetTag("request_id", middleware.GetRequestID(ctx))
				scope.SetUser(sentry.User{
					ID: middleware.MustGetUserID(ctx),
				})
				scope.SetLevel(sentry.LevelError)
				scope.SetContext("upload", map[string]interface{}{
					"filename":     req.Filename,
					"content_type": req.ContentType,
					"size":         req.Size,
				})
			})

			hub.CaptureException(err)
		}

		// Map to HTTP error
		problem := h.mapError(err)
		middleware.RespondProblem(w, r, problem)
		return
	}

	// ... success response ...
}

// isUserError checks if error is caused by user input (don't report to Sentry)
func isUserError(err error) bool {
	// Validation errors, not found, etc.
	// Don't report user mistakes to error tracking
	switch {
	case errors.Is(err, domain.ErrValidation):
		return true
	case errors.Is(err, domain.ErrNotFound):
		return true
	case errors.Is(err, domain.ErrConflict):
		return true
	default:
		return false
	}
}
```

### 6. Performance Monitoring (Transactions)

```go
package handlers

import (
	"net/http"

	"github.com/getsentry/sentry-go"
)

func (h *ImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Start transaction for performance monitoring
	span := sentry.StartSpan(r.Context(), "http.server",
		sentry.TransactionName("/api/v1/images"),
	)
	defer span.Finish()

	ctx := span.Context()

	// Create child span for specific operation
	uploadSpan := sentry.StartSpan(ctx, "image.upload")
	image, err := h.uploadImage.Handle(ctx, cmd)
	uploadSpan.Finish()

	// Create child span for storage
	storageSpan := sentry.StartSpan(ctx, "storage.save")
	// ... storage operation ...
	storageSpan.Finish()

	// ... rest of handler ...
}
```

---

## PII Scrubbing Configuration

**CRITICAL**: Error tracking MUST NOT capture personally identifiable information (GDPR Article 32).

### 1. Data Classification

| Data Type | Capture? | Scrubbing Method |
|-----------|----------|------------------|
| **User ID** | Yes | UUID is not PII (pseudonymous identifier) |
| **Email Address** | No | Scrub from error messages, breadcrumbs, request data |
| **IP Address** | Partial | Last octet masked (192.168.1.xxx) |
| **Passwords** | Never | Scrub from all contexts |
| **Access Tokens** | Never | Scrub from headers, cookies, request bodies |
| **Request ID** | Yes | Not PII, needed for correlation |
| **User Agent** | Yes | Not PII (device fingerprint acceptable) |
| **File Names** | Partial | Scrub if contains PII (e.g., "john-doe-selfie.jpg" → "user-image.jpg") |
| **Stack Traces** | Yes | Safe if no PII in variable names |

### 2. Implement PII Scrubber

Update `/home/user/goimg-datalayer/internal/infrastructure/errortracking/sentry.go`:

```go
package errortracking

import (
	"net"
	"regexp"
	"strings"

	"github.com/getsentry/sentry-go"
)

var (
	// Email regex for scrubbing
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	// Password-like patterns
	passwordRegex = regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token|api[_-]?key)["\s:=]+[^\s"&]+`)

	// Bearer tokens
	bearerTokenRegex = regexp.MustCompile(`Bearer\s+[A-Za-z0-9\-\._~\+\/]+=*`)

	// Credit card numbers (basic pattern)
	creditCardRegex = regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`)

	// Sensitive headers to scrub
	sensitiveHeaders = map[string]bool{
		"authorization":  true,
		"cookie":         true,
		"set-cookie":     true,
		"x-api-key":      true,
		"x-auth-token":   true,
		"x-csrf-token":   true,
		"x-access-token": true,
	}

	// Sensitive form fields
	sensitiveFields = map[string]bool{
		"password":         true,
		"passwd":           true,
		"pwd":              true,
		"secret":           true,
		"api_key":          true,
		"apikey":           true,
		"token":            true,
		"access_token":     true,
		"refresh_token":    true,
		"credit_card":      true,
		"card_number":      true,
		"cvv":              true,
		"ssn":              true,
		"social_security":  true,
	}
)

// scrubbPII removes sensitive data from Sentry events
func scrubbPII(event *sentry.Event) *sentry.Event {
	// Scrub exception messages
	for i := range event.Exception {
		event.Exception[i].Value = scrubString(event.Exception[i].Value)
	}

	// Scrub log message
	if event.Message != "" {
		event.Message = scrubString(event.Message)
	}

	// Scrub breadcrumbs
	for i := range event.Breadcrumbs {
		event.Breadcrumbs[i].Message = scrubString(event.Breadcrumbs[i].Message)
		if event.Breadcrumbs[i].Data != nil {
			event.Breadcrumbs[i].Data = scrubMap(event.Breadcrumbs[i].Data)
		}
	}

	// Scrub request data
	if event.Request != nil {
		event.Request = scrubRequest(event.Request)
	}

	// Scrub user data (keep ID, mask IP)
	if event.User.IPAddress != "" {
		event.User.IPAddress = maskIP(event.User.IPAddress)
	}
	// Remove email if accidentally set
	event.User.Email = ""
	event.User.Username = ""

	// Scrub extra context
	if event.Extra != nil {
		event.Extra = scrubMap(event.Extra)
	}

	// Scrub contexts
	for key, context := range event.Contexts {
		if contextMap, ok := context.(map[string]interface{}); ok {
			event.Contexts[key] = scrubMap(contextMap)
		}
	}

	return event
}

// scrubString removes PII from a string
func scrubString(s string) string {
	// Remove emails
	s = emailRegex.ReplaceAllString(s, "[EMAIL_REDACTED]")

	// Remove passwords and tokens
	s = passwordRegex.ReplaceAllString(s, "$1=[REDACTED]")

	// Remove bearer tokens
	s = bearerTokenRegex.ReplaceAllString(s, "Bearer [TOKEN_REDACTED]")

	// Remove credit cards
	s = creditCardRegex.ReplaceAllString(s, "[CARD_REDACTED]")

	return s
}

// scrubMap removes sensitive keys from maps
func scrubMap(m map[string]interface{}) map[string]interface{} {
	scrubbed := make(map[string]interface{})

	for key, value := range m {
		lowerKey := strings.ToLower(key)

		// Check if key is sensitive
		if sensitiveFields[lowerKey] || sensitiveHeaders[lowerKey] {
			scrubbed[key] = "[REDACTED]"
			continue
		}

		// Recursively scrub nested maps
		switch v := value.(type) {
		case map[string]interface{}:
			scrubbed[key] = scrubMap(v)
		case string:
			scrubbed[key] = scrubString(v)
		default:
			scrubbed[key] = value
		}
	}

	return scrubbed
}

// scrubRequest removes PII from HTTP request data
func scrubRequest(req *sentry.Request) *sentry.Request {
	// Scrub headers
	if req.Headers != nil {
		scrubbedHeaders := make(map[string]string)
		for key, value := range req.Headers {
			lowerKey := strings.ToLower(key)
			if sensitiveHeaders[lowerKey] {
				scrubbedHeaders[key] = "[REDACTED]"
			} else {
				scrubbedHeaders[key] = scrubString(value)
			}
		}
		req.Headers = scrubbedHeaders
	}

	// Scrub cookies (if captured)
	req.Cookies = ""

	// Scrub query string
	if req.QueryString != "" {
		req.QueryString = scrubString(req.QueryString)
	}

	// Scrub POST data
	if req.Data != nil {
		if dataMap, ok := req.Data.(map[string]interface{}); ok {
			req.Data = scrubMap(dataMap)
		} else if dataStr, ok := req.Data.(string); ok {
			req.Data = scrubString(dataStr)
		}
	}

	// Scrub environment variables (if captured)
	if req.Env != nil {
		scrubbedEnv := make(map[string]string)
		for key, value := range req.Env {
			if strings.Contains(strings.ToLower(key), "secret") ||
				strings.Contains(strings.ToLower(key), "password") ||
				strings.Contains(strings.ToLower(key), "token") {
				scrubbedEnv[key] = "[REDACTED]"
			} else {
				scrubbedEnv[key] = value
			}
		}
		req.Env = scrubbedEnv
	}

	return req
}

// maskIP masks the last octet of an IP address
func maskIP(ip string) string {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "[INVALID_IP]"
	}

	// IPv4: mask last octet
	if ipv4 := parsedIP.To4(); ipv4 != nil {
		ipv4[3] = 0
		return ipv4.String() // e.g., 192.168.1.0
	}

	// IPv6: mask last 64 bits
	if ipv6 := parsedIP.To16(); ipv6 != nil {
		for i := 8; i < 16; i++ {
			ipv6[i] = 0
		}
		return parsedIP.String() // e.g., 2001:db8::/64
	}

	return "[INVALID_IP]"
}
```

### 3. Test PII Scrubbing

Create test file `/home/user/goimg-datalayer/internal/infrastructure/errortracking/sentry_test.go`:

```go
package errortracking

import (
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
)

func TestScrubbPII_EmailRedaction(t *testing.T) {
	event := &sentry.Event{
		Message: "User john.doe@example.com failed to login",
	}

	scrubbed := scrubbPII(event)

	assert.Contains(t, scrubbed.Message, "[EMAIL_REDACTED]")
	assert.NotContains(t, scrubbed.Message, "john.doe@example.com")
}

func TestScrubbPII_PasswordRedaction(t *testing.T) {
	event := &sentry.Event{
		Exception: []sentry.Exception{
			{Value: "authentication failed with password=SecretPass123"},
		},
	}

	scrubbed := scrubbPII(event)

	assert.Contains(t, scrubbed.Exception[0].Value, "[REDACTED]")
	assert.NotContains(t, scrubbed.Exception[0].Value, "SecretPass123")
}

func TestScrubbPII_BearerTokenRedaction(t *testing.T) {
	event := &sentry.Event{
		Request: &sentry.Request{
			Headers: map[string]string{
				"Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature",
			},
		},
	}

	scrubbed := scrubbPII(event)

	assert.Equal(t, "[REDACTED]", scrubbed.Request.Headers["Authorization"])
}

func TestScrubbPII_IPMasking(t *testing.T) {
	event := &sentry.Event{
		User: sentry.User{
			IPAddress: "192.168.1.42",
		},
	}

	scrubbed := scrubbPII(event)

	assert.Equal(t, "192.168.1.0", scrubbed.User.IPAddress)
}

func TestScrubbPII_SensitiveFormFields(t *testing.T) {
	event := &sentry.Event{
		Request: &sentry.Request{
			Data: map[string]interface{}{
				"email":    "user@example.com",
				"password": "SecretPass123",
				"username": "johndoe",
			},
		},
	}

	scrubbed := scrubbPII(event)

	dataMap := scrubbed.Request.Data.(map[string]interface{})
	assert.Equal(t, "[REDACTED]", dataMap["password"])
	assert.Contains(t, dataMap["email"], "[EMAIL_REDACTED]")
	assert.Equal(t, "johndoe", dataMap["username"]) // Username is OK
}
```

Run tests:

```bash
cd /home/user/goimg-datalayer
go test -v ./internal/infrastructure/errortracking/...
```

---

## Operational Procedures

### Accessing the Dashboard

**GlitchTip**:
- URL: https://glitchtip.goimg.example.com
- Login: Use your organization credentials

**Sentry**:
- URL: https://sentry.io
- Login: Use your Sentry account

### Viewing Errors

#### 1. Issues List

**Path**: Projects → goimg-api → Issues

**Columns**:
- Error type and message
- First seen / Last seen timestamps
- Event count
- Users affected
- Assigned to

**Filters**:
- Environment (production, staging, development)
- Release version
- Status (unresolved, resolved, ignored)
- Assigned to team member

#### 2. Issue Detail View

Click on any issue to see:

**Overview Tab**:
- Error message and type
- Stack trace
- Breadcrumbs (events leading up to error)
- Tags (request_id, user_id, http.method, etc.)

**Activity Tab**:
- Comments from team
- Assignment history
- Status changes

**Similar Issues Tab**:
- Related errors (same root cause)

**Tags Tab**:
- All tags and their values
- Filter by specific tag values

#### 3. Stack Trace Navigation

GlitchTip/Sentry shows:
```
File: /home/user/goimg-datalayer/internal/application/commands/upload_image.go
Line: 42
Function: (*UploadImageHandler).Handle
Code context:
  40:   if err := h.validator.Validate(cmd); err != nil {
  41:       return nil, fmt.Errorf("validation failed: %w", err)
→ 42:       panic("unexpected nil repository")  ← Error occurred here
  43:   }
  44:   return image, nil
```

**Actions**:
- Click filename to open in GitHub (if integration configured)
- View full stack trace
- See local variables (if source maps configured)

### Configuring Alerts

#### Email Alerts

**GlitchTip**: Settings → Alerts → New Alert

**Configuration**:
- **Trigger**: "A new issue is created" OR "Issue frequency is above 10 events in 1 minute"
- **Environment**: Production
- **Send to**: team@goimg.example.com
- **Frequency**: Immediately (for new issues), Digest daily (for known issues)

#### Slack Integration

**GlitchTip**: Settings → Integrations → Slack

**Setup**:
1. Create Slack app: https://api.slack.com/apps
2. Add incoming webhook
3. Copy webhook URL
4. In GlitchTip: Add webhook URL
5. Configure channel: #engineering-alerts

**Alert Format**:
```
🔴 New Error in goimg-api (production)
DatabaseConnectionError: failed to connect to postgres
Affected users: 23
First seen: 2 minutes ago
View in GlitchTip: https://glitchtip.goimg.example.com/issues/12345
```

#### PagerDuty Integration (Sentry only)

**Sentry**: Settings → Integrations → PagerDuty

**Routing**:
- P0 (Critical): Page on-call immediately
- P1 (High): Page during business hours
- P2 (Medium): Create ticket, no page
- P3 (Low): Ignore

### Searching and Filtering Errors

#### Search Syntax

**By environment**:
```
environment:production
```

**By release**:
```
release:goimg-api@1.2.3
```

**By user**:
```
user.id:550e8400-e29b-41d4-a716-446655440000
```

**By tag**:
```
request_id:req-abc123
http.method:POST
```

**By error type**:
```
error.type:DatabaseConnectionError
```

**Combined**:
```
environment:production release:goimg-api@1.2.3 error.type:ValidationError
```

#### Advanced Filters

**Time range**:
- Last hour
- Last 24 hours
- Last 7 days
- Custom range

**Event count**:
- More than 100 events
- Less than 10 events

**User impact**:
- Affecting >10 users
- Affecting <5 users

### Assigning and Resolving Issues

#### Assignment

1. Open issue
2. Click "Assign to" → Select team member
3. Add comment explaining context
4. Set priority label (P0, P1, P2, P3)

#### Resolution

**Mark as Resolved**:
- Click "Resolve" button
- Select resolution type:
  - "In next release" (fix deployed)
  - "In current release" (already fixed)
  - "Ignored" (known issue, not fixing)

**Resolution Notes**:
- Add comment explaining fix
- Link to GitHub PR/commit
- Mention if issue needs monitoring

**Auto-resolve**:
- Configure auto-resolve after 30 days of no new events
- Or after specific release deployed

### Monitoring Error Trends

#### Metrics to Track

**Daily**:
- Total errors (should trend down over time)
- New error types introduced (investigate immediately)
- Users affected (high user impact = P0)

**Weekly**:
- Error rate by release (which releases are stable?)
- Top 10 errors (focus on high-frequency issues)
- MTTR (Mean Time To Resolution)

**Monthly**:
- Error reduction % (goal: -20% month-over-month)
- Release stability score
- Team response time

#### Dashboards

Create custom dashboard in GlitchTip/Sentry:

**Production Health Dashboard**:
- Total errors (7-day trend)
- New vs returning errors
- Top 5 errors by frequency
- Top 5 errors by user impact
- Errors by release version
- Error rate by environment

---

## Performance Impact

### SDK Overhead

**Negligible** for error capture:
- Error capture: <1ms per error (asynchronous)
- Transaction sampling: ~2-5ms per sampled request (10% sample = 0.2-0.5ms average)

**Total API latency impact**: <1ms average (within noise margin)

### Network Traffic

**Outbound to Sentry/GlitchTip**:
- Per error event: ~5-20 KB (depending on stack trace size)
- Per transaction: ~1-3 KB

**Example**: 1000 errors/day = ~10 MB/day outbound

**Impact**: Minimal (API already sends logs, metrics)

### Memory Usage

- SDK initialization: ~5 MB
- Event queue buffer: ~10 MB (holds 100 events before dropping)

**Total**: ~15 MB per instance (acceptable for API server)

### Recommendations

1. **Keep 100% error sampling**: Errors are infrequent, capture all
2. **Use 10% transaction sampling**: Reduces overhead while maintaining visibility
3. **Set timeout**: 5 seconds for event sending (don't block shutdown)
4. **Monitor**: Track Sentry API latency (should be <100ms p95)

---

## Security Considerations

### Data Privacy (GDPR)

**Compliance Requirements**:
- Article 5.1.c: Data minimization (don't collect unnecessary PII)
- Article 32: Technical measures (PII scrubbing, encryption in transit)
- Article 33: Breach notification (errors may contain breach indicators)

**Implementation**:
- ✅ PII scrubbing implemented (see PII section)
- ✅ TLS for data in transit (Sentry/GlitchTip use HTTPS)
- ✅ Data retention: 90 days (configurable)
- ✅ Right to be forgotten: Can delete user's error events by user.id

### Self-Hosted vs Cloud

**GlitchTip (Self-Hosted)**:
- ✅ Data stays in your infrastructure (GDPR-friendly)
- ✅ No third-party processors (simpler compliance)
- ⚠️ You are responsible for security patches

**Sentry (Cloud)**:
- ⚠️ Data sent to US servers (requires Data Processing Agreement)
- ✅ SOC 2 Type II certified
- ✅ Automatic security updates

**Recommendation**: Use GlitchTip for goimg-datalayer (privacy-sensitive image gallery)

### Access Control

**GlitchTip Roles**:
- **Admin**: Full access, can delete data
- **Manager**: Can manage projects and team
- **Member**: Can view errors, assign issues
- **Billing**: Can view billing only

**Best Practices**:
- Use SSO (SAML, OAuth) if available
- Enable 2FA for all users
- Review access quarterly
- Audit log all admin actions

### Secret Management

**DO NOT** hardcode Sentry DSN in code:

```go
// ❌ BAD: Hardcoded DSN
sentry.Init(sentry.ClientOptions{
    Dsn: "https://abc123@o123456.ingest.sentry.io/789012",
})

// ✅ GOOD: From environment variable
sentry.Init(sentry.ClientOptions{
    Dsn: os.Getenv("SENTRY_DSN"),
})
```

**Environment variable management**:
```bash
# Production: Use Kubernetes secrets
kubectl create secret generic sentry-config \
  --from-literal=SENTRY_DSN='https://...'

# Docker Compose: Use .env file (not committed)
echo "SENTRY_DSN=https://..." >> docker/.env.prod
```

---

## Verification Checklist (Security Gate S9-MON-002)

Security gate S9-MON-002 is **VERIFIED** when:

- [ ] Error tracking solution deployed (GlitchTip or Sentry)
- [ ] Go SDK integrated in API and worker services
- [ ] Panic recovery middleware captures to error tracking
- [ ] PII scrubbing implemented and tested
- [ ] Sample rates configured: 100% errors, 10% transactions
- [ ] Alerts configured for error spikes and new error types
- [ ] Team trained on dashboard usage and issue triage
- [ ] Documentation complete (this guide)
- [ ] Access control configured (team members assigned roles)
- [ ] Data retention policy set (90 days default)
- [ ] Integration tested in staging environment
- [ ] Production deployment verified (errors appearing in dashboard)

---

## Troubleshooting

### Errors Not Appearing in Dashboard

**Check 1**: Is SDK initialized?

```bash
# Look for initialization log
docker logs goimg-api 2>&1 | grep "sentry error tracking initialized"
```

**Check 2**: Is DSN correct?

```bash
# Test DSN connectivity
curl -X POST "https://YOUR_ORG.ingest.sentry.io/api/1/store/" \
  -H "Content-Type: application/json" \
  -H "X-Sentry-Auth: Sentry sentry_version=7, sentry_key=YOUR_KEY" \
  -d '{"message": "test"}'

# Expected: HTTP 200 OK
```

**Check 3**: Is environment filtering events?

```go
// Check BeforeSend filter
BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
    if cfg.Environment == "development" {
        return nil  // ← Events filtered out in dev
    }
    return event
},
```

### GlitchTip Container Issues

**Issue**: GlitchTip web not starting

```bash
# Check logs
docker-compose -f docker-compose.glitchtip.yml logs glitchtip-web

# Common issues:
# - Database not ready: Wait for healthcheck
# - SECRET_KEY not set: Check .env file
# - Migration failed: Run manually:
docker exec goimg-glitchtip-web ./manage.py migrate
```

**Issue**: Database connection failed

```bash
# Test PostgreSQL connection
docker exec goimg-glitchtip-postgres psql -U glitchtip -d glitchtip -c "SELECT 1;"

# Check DATABASE_URL format
echo $DATABASE_URL
# Expected: postgresql://glitchtip:password@glitchtip-postgres:5432/glitchtip
```

### High Memory Usage

**Symptom**: Sentry SDK using >100 MB memory

**Cause**: Event queue backup (network issues)

**Fix**:
```go
sentry.Init(sentry.ClientOptions{
    // Limit queue size
    MaxQueueSize: 100,  // Drop events if queue full

    // Reduce timeout
    Transport: &sentry.HTTPSyncTransport{
        Timeout: 5 * time.Second,
    },
})
```

### PII Leaking

**Check**: Review recent events for PII

```bash
# In GlitchTip dashboard
1. Open any error
2. Check "Additional Data" tab
3. Search for: @, password, token, secret
4. If found: Fix PII scrubber and rotate affected credentials
```

---

## Next Steps

After completing error tracking setup:

1. **Monitor for 7 days**: Ensure errors are captured correctly
2. **Tune alert thresholds**: Reduce noise, increase signal
3. **Create runbooks**: Document response procedures for common errors
4. **Integrate with incident response**: Link to `/home/user/goimg-datalayer/docs/security/incident_response.md`
5. **Performance monitoring**: Enable transaction tracing for slow endpoints
6. **Custom instrumentation**: Add spans for critical business logic

---

## References

- **Sentry Documentation**: https://docs.sentry.io/platforms/go/
- **GlitchTip Documentation**: https://glitchtip.com/documentation
- **Sentry Go SDK**: https://github.com/getsentry/sentry-go
- **GDPR Article 32**: https://gdpr-info.eu/art-32-gdpr/
- **SOC 2 CC7.2**: AICPA Trust Services Criteria

---

## Document Control

**Version**: 1.0
**Created**: 2025-12-06
**Author**: cicd-guardian agent
**Security Gate**: S9-MON-002

**Related Documents**:
- `/home/user/goimg-datalayer/docs/security/monitoring.md`
- `/home/user/goimg-datalayer/docs/security/incident_response.md`
- `/home/user/goimg-datalayer/docs/deployment/secret-management.md`

**Approval Required**:
- [ ] Security Team Lead
- [ ] Engineering Manager
- [ ] Privacy Officer (for PII scrubbing review)
