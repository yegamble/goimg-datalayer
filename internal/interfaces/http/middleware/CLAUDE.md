# Security Middleware Guide

> HTTP security middleware patterns for goimg-datalayer Sprint 4.

## Overview

This directory implements security middleware for the HTTP layer, providing defense-in-depth protection through layered security controls. All middleware functions follow the chi router pattern: `func(http.Handler) http.Handler`.

## Critical Security Rules

1. **Middleware order matters** - Security checks must execute in the correct sequence
2. **Fail securely** - Default deny on errors (return 401/403, not 200)
3. **No business logic** - Middleware only enforces security policies
4. **Constant-time operations** - Prevent timing attacks in security checks
5. **RFC 7807 errors** - All security errors use Problem Details format
6. **Rate limit before auth** - Prevent brute-force credential attacks
7. **Log security events** - Audit all authentication and authorization failures

## Middleware Execution Order

**CRITICAL**: Middleware executes in the order defined in `server.go`. Incorrect ordering creates security vulnerabilities.

### Global Middleware Stack (All Routes)

```go
r := chi.NewRouter()

// 1. Request ID - MUST BE FIRST
//    Generates correlation ID for tracing and audit logs
r.Use(middleware.RequestID)

// 2. Structured Logging - SECOND
//    Logs all requests/responses with request ID
//    Records: method, path, status, duration, IP
r.Use(middleware.Logger(logger))

// 3. Recovery - THIRD
//    Catches panics, logs stack trace, returns 500
//    Prevents information disclosure via stack traces
r.Use(middleware.Recoverer(logger))

// 4. Security Headers - BEFORE CORS
//    Sets defense headers: CSP, X-Frame-Options, etc.
r.Use(middleware.SecurityHeaders)

// 5. CORS - AFTER Security Headers
//    Allows cross-origin requests from approved domains
r.Use(middleware.CORS(cfg.CORS))

// 6. Rate Limiting (Global) - BEFORE Authentication
//    100 req/min per IP (prevents credential brute-force)
r.Use(middleware.RateLimitGlobal(redisClient, 100, time.Minute))

// 7. Timeout - LAST Global Middleware
//    Prevents long-running requests from exhausting resources
r.Use(middleware.Timeout(30 * time.Second))
```

### Protected Route Middleware

```go
// Protected routes group
r.Group(func(r chi.Router) {
    // 8. JWT Authentication - FIRST in protected routes
    //    Validates token, checks blacklist, sets user context
    r.Use(middleware.JWTAuth(jwtService, blacklist))

    // 9. Rate Limiting (Authenticated) - AFTER Authentication
    //    300 req/min per user (prevents API abuse)
    r.Use(middleware.RateLimitPerUser(redisClient, 300, time.Minute))

    // Routes here...
})
```

### Endpoint-Specific Middleware

```go
// Login endpoint - special rate limiting
r.With(middleware.RateLimitByIP(redisClient, 5, time.Minute)).
    Post("/auth/login", handlers.Auth.Login)

// Admin-only routes
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireRole("admin"))
    // Admin routes...
})

// RBAC for specific permissions
r.With(middleware.RequirePermission("image:moderate")).
    Post("/images/{id}/moderate", handlers.Image.Moderate)
```

### Why This Order?

1. **RequestID first**: All logs need correlation ID
2. **Logger second**: Needs request ID from context
3. **Recovery third**: Catches panics from all downstream middleware
4. **SecurityHeaders before CORS**: Defense headers apply before cross-origin logic
5. **Rate limiting before auth**: Prevents brute-force attacks on credentials
6. **Timeout last global**: Applies to all downstream processing
7. **Auth first in protected**: Must identify user before authorization checks
8. **Per-user rate limiting after auth**: Needs user ID from context

**Security Impact**: Wrong order can allow:
- Brute-force attacks (no rate limiting before auth)
- Timing attacks (auth before rate limiting reveals valid usernames)
- Resource exhaustion (no timeout)
- Information disclosure (recovery not first)

## Security Headers Configuration

### Required Headers (All Responses)

```go
// Prevents MIME-type sniffing attacks
w.Header().Set("X-Content-Type-Options", "nosniff")

// Prevents clickjacking attacks via iframe embedding
w.Header().Set("X-Frame-Options", "DENY")

// Legacy XSS protection (modern browsers use CSP)
w.Header().Set("X-XSS-Protection", "1; mode=block")

// Controls referrer information leakage
w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

// Content Security Policy - prevents XSS, injection attacks
w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: https:; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; connect-src 'self'; frame-ancestors 'none'")

// Enforces HTTPS (only in production, NOT in development)
if isProd {
    w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
}

// Restricts browser feature access
w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=()")
```

### Header-by-Header Analysis

| Header | Attack Prevented | Rationale |
|--------|------------------|-----------|
| `X-Content-Type-Options: nosniff` | MIME confusion attacks | Forces browser to respect Content-Type, prevents executing HTML as JS |
| `X-Frame-Options: DENY` | Clickjacking | Prevents embedding in iframe (use `SAMEORIGIN` if embedding needed) |
| `X-XSS-Protection: 1; mode=block` | Reflected XSS | Legacy header, still useful for older browsers |
| `Referrer-Policy: strict-origin-when-cross-origin` | Information leakage | Sends origin only on cross-origin, full URL on same-origin |
| `Content-Security-Policy` | XSS, injection, inline scripts | Whitelist allowed content sources, blocks inline scripts |
| `Strict-Transport-Security` | MITM, SSL stripping | Forces HTTPS for 1 year (includeSubDomains applies to all subdomains) |
| `Permissions-Policy` | Unauthorized feature access | Disables dangerous browser APIs (geolocation, camera, etc.) |

### CSP Directives Explained

```
default-src 'self'              → Default: only load from same origin
img-src 'self' data: https:     → Images: same origin + data URIs + HTTPS sources
script-src 'self'               → Scripts: only same origin (NO inline scripts)
style-src 'self' 'unsafe-inline'→ Styles: same origin + inline (needed for dynamic styles)
font-src 'self'                 → Fonts: only same origin
connect-src 'self'              → XHR/WebSocket: only same origin
frame-ancestors 'none'          → Cannot be embedded in any frame (same as X-Frame-Options: DENY)
```

**Security Trade-off**: `'unsafe-inline'` in `style-src` weakens CSP but is often required for CSS-in-JS libraries. Consider using nonces or hashes for inline styles in production.

### HSTS Considerations

**Development**: DO NOT set HSTS (breaks local HTTP testing)

**Production**:
- Start with `max-age=300` (5 minutes) to test
- Gradually increase to `max-age=31536000` (1 year)
- Add `includeSubDomains` if all subdomains use HTTPS
- Add `preload` only after submitting to [HSTS preload list](https://hstspreload.org/)

**Warning**: HSTS preload is irreversible. Users cannot access site via HTTP until max-age expires.

## Rate Limiting Strategy

### Redis-Backed Sliding Window

**Algorithm**: Fixed window counter with Redis TTL

**Key Pattern**:
```
goimg:ratelimit:{scope}:{identifier}
```

**Examples**:
- Global (by IP): `goimg:ratelimit:global:192.168.1.1`
- Authenticated (by user): `goimg:ratelimit:user:550e8400-e29b-41d3-a456-426614174000`
- Login (by IP): `goimg:ratelimit:login:192.168.1.1`

### Rate Limit Tiers

| Scope | Limit | Window | Key | Use Case |
|-------|-------|--------|-----|----------|
| Global (IP) | 100 req/min | 1 minute | IP address | Prevents API abuse from single source |
| Authenticated | 300 req/min | 1 minute | User ID | Higher limit for logged-in users |
| Login | 5 req/min | 1 minute | IP address | Prevents credential brute-force |
| Password Reset | 3 req/hour | 1 hour | Email hash | Prevents enumeration and spam |
| Upload | 50 uploads/hour | 1 hour | User ID | Prevents storage abuse |

### Implementation Pattern

```go
type RateLimiter struct {
    redis  *redis.Client
    limit  int           // Max requests
    window time.Duration // Time window
}

func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, *RateLimitInfo, error) {
    fullKey := "goimg:ratelimit:" + key
    now := time.Now().Unix()

    // Increment counter
    pipe := rl.redis.Pipeline()
    incr := pipe.Incr(ctx, fullKey)
    pipe.Expire(ctx, fullKey, rl.window)
    _, err := pipe.Exec(ctx)
    if err != nil {
        return false, nil, fmt.Errorf("redis pipeline: %w", err)
    }

    count := incr.Val()
    allowed := count <= int64(rl.limit)

    // Get TTL for Retry-After
    ttl, _ := rl.redis.TTL(ctx, fullKey).Result()
    resetAt := time.Now().Add(ttl)

    info := &RateLimitInfo{
        Limit:     rl.limit,
        Remaining: max(0, rl.limit-int(count)),
        Reset:     resetAt.Unix(),
        RetryAfter: int(ttl.Seconds()),
    }

    return allowed, info, nil
}
```

### Response Headers

**On Success (200-399)**:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1699564800  (Unix timestamp)
```

**On Rate Limit Exceeded (429)**:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1699564800
Retry-After: 42  (seconds until reset)
```

**RFC 7807 Error Body**:
```json
{
    "type": "https://api.goimg.dev/problems/rate-limit-exceeded",
    "title": "Rate Limit Exceeded",
    "status": 429,
    "detail": "You have exceeded the rate limit of 100 requests per minute",
    "retryAfter": 42,
    "traceId": "550e8400-e29b-41d3-a456-426614174000"
}
```

### Security Considerations

1. **IP Spoofing**: Use `X-Forwarded-For` only if behind trusted proxy
   ```go
   // Get real IP
   ip := r.Header.Get("X-Forwarded-For")
   if ip == "" || !cfg.TrustProxy {
       ip = r.RemoteAddr
   }
   ```

2. **Distributed Rate Limiting**: Redis ensures consistent rate limits across multiple API servers

3. **Bypass for Internal Services**: Whitelist internal IPs or use service tokens

4. **Rate Limit Poisoning**: Sanitize keys to prevent Redis key injection
   ```go
   // Bad: goimg:ratelimit:user:USER_INPUT
   // Good: goimg:ratelimit:user:{SHA256(USER_INPUT)}
   ```

## Account Security Controls

### Account Lockout Policy

**Objective**: Prevent online password guessing attacks while minimizing legitimate user impact.

**Policy**:
- Threshold: 5 failed login attempts
- Lockout Duration: 15 minutes
- Scope: Per account (not IP-based to prevent lockout attacks)

**Implementation**:

```go
// Redis keys
goimg:lockout:{user_id}          → Lockout expiry timestamp (TTL: 15 min)
goimg:failed_attempts:{user_id}  → Failed attempt counter (TTL: 15 min)

// On failed login
func (h *AuthHandler) handleFailedLogin(ctx context.Context, userID string) error {
    key := "goimg:failed_attempts:" + userID

    // Increment counter
    attempts, err := h.redis.Incr(ctx, key).Result()
    if err != nil {
        return err
    }

    // Set TTL on first attempt
    if attempts == 1 {
        h.redis.Expire(ctx, key, 15*time.Minute)
    }

    // Lock account after 5 attempts
    if attempts >= 5 {
        lockoutKey := "goimg:lockout:" + userID
        h.redis.Set(ctx, lockoutKey, time.Now().Add(15*time.Minute).Unix(), 15*time.Minute)

        // Log security event
        h.logger.Warn().
            Str("user_id", userID).
            Int64("attempts", attempts).
            Msg("account locked due to failed login attempts")

        // Send email notification
        h.notifier.SendAccountLockoutEmail(ctx, userID)
    }

    return nil
}

// On successful login
func (h *AuthHandler) handleSuccessfulLogin(ctx context.Context, userID string) error {
    // Clear failed attempts counter
    h.redis.Del(ctx, "goimg:failed_attempts:"+userID)
    return nil
}

// Check lockout before authentication
func (h *AuthHandler) isLocked(ctx context.Context, userID string) (bool, time.Time, error) {
    lockoutKey := "goimg:lockout:" + userID

    expiryUnix, err := h.redis.Get(ctx, lockoutKey).Int64()
    if err != nil {
        if errors.Is(err, redis.Nil) {
            return false, time.Time{}, nil // Not locked
        }
        return false, time.Time{}, err
    }

    expiry := time.Unix(expiryUnix, 0)
    return true, expiry, nil
}
```

**RFC 7807 Error Response** (Account Locked):
```json
{
    "type": "https://api.goimg.dev/problems/account-locked",
    "title": "Account Temporarily Locked",
    "status": 403,
    "detail": "Your account has been locked due to multiple failed login attempts. Please try again in 15 minutes.",
    "lockedUntil": "2024-11-10T15:45:00Z",
    "traceId": "550e8400-e29b-41d3-a456-426614174000"
}
```

**Security Properties**:
- Exponential lockout (future): 5 min → 15 min → 1 hour → 24 hour
- Email notification alerts legitimate users of attacks
- Admin unlock capability for support tickets
- Logs all lockout events for security monitoring

**Attack Mitigation**:
- **Brute Force**: 5 attempts = 0.00001% chance to guess 8-char password
- **Credential Stuffing**: Slows down automated attacks
- **Account Lockout DoS**: Short duration (15 min) limits impact

### Account Enumeration Prevention

**Vulnerability**: Attackers can determine valid email addresses by observing different responses.

**Bad Implementation**:
```go
// WRONG: Reveals whether email exists
if user == nil {
    return ProblemNotFound("Email not found")
}
if !passwordValid {
    return ProblemUnauthorized("Invalid password")
}
```

**Secure Implementation**:
```go
// CORRECT: Same error message for all failures
if user == nil || !passwordValid {
    return ProblemUnauthorized("Invalid email or password")
}
```

**Timing Attack Mitigation**:
```go
// WRONG: Fast return reveals missing user
user := repo.FindByEmail(email)
if user == nil {
    return ErrInvalidCredentials
}
if !bcrypt.Compare(password, user.PasswordHash) {
    return ErrInvalidCredentials
}

// CORRECT: Constant-time comparison
user := repo.FindByEmail(email)

// Always hash password, even if user not found
passwordHash := user.PasswordHash
if user == nil {
    // Use dummy hash to maintain constant time
    passwordHash = "$2a$12$dummyHashForConstantTimePaddingToMaintainLength"
}

valid := bcrypt.Compare(password, passwordHash)
if user == nil || !valid {
    return ErrInvalidCredentials
}
```

**Additional Protections**:
1. **Constant-time string comparison**:
   ```go
   import "crypto/subtle"

   // WRONG: Early return on length difference
   if len(a) != len(b) {
       return false
   }

   // CORRECT: Constant-time comparison
   return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
   ```

2. **Generic error messages**: Never reveal:
   - "Email not found"
   - "Password incorrect"
   - "Account disabled"
   - Use: "Invalid email or password" for all cases

3. **Password reset enumeration**: Same response for existing/non-existing emails
   ```go
   // Always return 200 OK
   // Send email only if account exists
   // Generic message: "If an account exists, you will receive an email"
   ```

4. **Registration enumeration**: Use email confirmation to prevent enumeration via signup

### Session ID Regeneration

**Vulnerability**: Session fixation attack - attacker sets victim's session ID

**Protection**: Generate new session ID on authentication state changes

```go
// On login - create NEW session
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    // ... authenticate user ...

    // Generate new session ID (don't reuse any existing ID)
    sessionID := uuid.New().String()

    session := &Session{
        SessionID: sessionID,
        UserID:    user.ID().String(),
        Email:     user.Email().String(),
        Role:      user.Role().String(),
        IP:        getClientIP(r),
        UserAgent: r.UserAgent(),
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
    }

    h.sessionStore.Create(ctx, session)

    // Issue tokens with new session ID
    accessToken := h.jwtService.GenerateAccessToken(
        user.ID().String(),
        user.Email().String(),
        user.Role().String(),
        sessionID, // New session ID in token
    )

    // ... return tokens ...
}

// On privilege escalation - regenerate session
func (h *AdminHandler) ElevatePrivileges(w http.ResponseWriter, r *http.Request) {
    // Verify user credentials again
    // ... authentication check ...

    // Revoke old session
    oldSessionID := GetSessionID(r.Context())
    h.sessionStore.Revoke(ctx, oldSessionID)

    // Create new elevated session
    newSessionID := uuid.New().String()
    // ... create new session with admin role ...
}
```

**When to Regenerate**:
- Login (authentication)
- Logout (invalidate old session)
- Privilege change (user → admin)
- Password change (invalidate all sessions)
- Critical action completion (payment, etc.)

**Session Fixation Attack Flow** (Before Mitigation):
1. Attacker obtains session ID: `SESSION=abc123`
2. Attacker tricks victim into using that session ID
3. Victim logs in, same session ID persists
4. Attacker uses `SESSION=abc123` to access victim's account

**After Mitigation**:
1. Attacker obtains session ID: `SESSION=abc123`
2. Victim logs in, **NEW session ID generated**: `SESSION=xyz789`
3. Attacker's `SESSION=abc123` is now invalid

### Audit Logging

**Objective**: Record security-relevant events for forensics and compliance.

**Events to Log**:

| Event | Severity | Data to Include |
|-------|----------|-----------------|
| Login success | INFO | User ID, IP, User-Agent, Timestamp |
| Login failure | WARN | Email (hashed), IP, Reason, Timestamp |
| Account lockout | WARN | User ID, IP, Failed attempts count |
| Logout | INFO | User ID, Session ID, Timestamp |
| Token refresh | INFO | User ID, Session ID, Timestamp |
| Password change | WARN | User ID, IP, Timestamp |
| Password reset request | WARN | Email (hashed), IP, Timestamp |
| Account created | INFO | User ID, IP, Email (hashed) |
| Account deleted | WARN | User ID, Admin ID, Reason |
| Role change | WARN | User ID, Old role, New role, Admin ID |
| Permission denied | WARN | User ID, Resource, Required permission |
| Session hijacking detected | ERROR | User ID, Old IP, New IP, Session ID |
| Token replay detected | ERROR | User ID, Token ID, Timestamp |
| Rate limit exceeded | INFO | IP or User ID, Endpoint, Limit |

**What NOT to Log** (Sensitive Data):
- Passwords (plaintext or hashed)
- Access tokens or refresh tokens
- Password reset tokens
- Email verification tokens
- Full credit card numbers
- Social security numbers
- Answers to security questions

**Structured Logging Example**:
```go
// Login success
h.logger.Info().
    Str("event", "login_success").
    Str("user_id", userID).
    Str("ip", clientIP).
    Str("user_agent", userAgent).
    Str("session_id", sessionID).
    Str("trace_id", traceID).
    Msg("user logged in successfully")

// Login failure
h.logger.Warn().
    Str("event", "login_failure").
    Str("email_hash", hashEmail(email)). // Hash email for privacy
    Str("ip", clientIP).
    Str("reason", "invalid_credentials").
    Str("trace_id", traceID).
    Msg("login attempt failed")

// Account lockout
h.logger.Warn().
    Str("event", "account_lockout").
    Str("user_id", userID).
    Str("ip", clientIP).
    Int("failed_attempts", attempts).
    Dur("lockout_duration", 15*time.Minute).
    Str("trace_id", traceID).
    Msg("account locked due to failed login attempts")

// Permission denied
h.logger.Warn().
    Str("event", "permission_denied").
    Str("user_id", userID).
    Str("resource", resourceID).
    Str("required_permission", permission).
    Str("user_role", role).
    Str("trace_id", traceID).
    Msg("access denied due to insufficient permissions")
```

**Email Hashing for Privacy**:
```go
import "crypto/sha256"

func hashEmail(email string) string {
    hash := sha256.Sum256([]byte(email))
    return fmt.Sprintf("%x", hash[:8]) // First 8 bytes for brevity
}
```

**Log Aggregation**: Forward logs to centralized system (ELK, Splunk, Datadog) for:
- Security monitoring and alerting
- Anomaly detection (ML-based)
- Compliance audits (SOC 2, GDPR)
- Incident response and forensics

**Log Retention**:
- Security logs: 1 year minimum (compliance requirement)
- Access logs: 90 days
- Error logs: 30 days

## Authentication Middleware

### JWT Validation Flow

```go
package middleware

import (
    "context"
    "net/http"
    "strings"

    "github.com/rs/zerolog"

    "goimg-datalayer/internal/infrastructure/security/jwt"
    httputil "goimg-datalayer/internal/interfaces/http"
)

type contextKey string

const (
    UserIDKey    contextKey = "user_id"
    EmailKey     contextKey = "email"
    RoleKey      contextKey = "role"
    SessionIDKey contextKey = "session_id"
)

type JWTAuthMiddleware struct {
    jwtService *jwt.Service
    blacklist  jwt.Blacklist
    logger     *zerolog.Logger
}

func JWTAuth(jwtService *jwt.Service, blacklist jwt.Blacklist, logger *zerolog.Logger) func(http.Handler) http.Handler {
    m := &JWTAuthMiddleware{
        jwtService: jwtService,
        blacklist:  blacklist,
        logger:     logger,
    }
    return m.Handler
}

func (m *JWTAuthMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Step 1: Extract Bearer token from Authorization header
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            m.logger.Warn().
                Str("event", "auth_missing").
                Str("path", r.URL.Path).
                Msg("missing authorization header")

            httputil.RespondProblem(w, r, httputil.ProblemUnauthorized(
                "Missing authorization header",
            ))
            return
        }

        // Step 2: Parse "Bearer <token>" format
        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
            m.logger.Warn().
                Str("event", "auth_invalid_format").
                Str("path", r.URL.Path).
                Msg("invalid authorization header format")

            httputil.RespondProblem(w, r, httputil.ProblemUnauthorized(
                "Invalid authorization header format. Expected: Bearer <token>",
            ))
            return
        }

        tokenString := parts[1]

        // Step 3: Check blacklist (before expensive signature verification)
        tokenID, err := m.jwtService.ExtractTokenID(tokenString)
        if err == nil {
            isBlacklisted, err := m.blacklist.IsBlacklisted(ctx, tokenID)
            if err != nil {
                m.logger.Error().
                    Err(err).
                    Str("event", "blacklist_check_failed").
                    Msg("failed to check token blacklist")

                httputil.RespondProblem(w, r, httputil.ProblemInternalError())
                return
            }

            if isBlacklisted {
                m.logger.Warn().
                    Str("event", "token_blacklisted").
                    Str("token_id", tokenID).
                    Str("path", r.URL.Path).
                    Msg("attempt to use blacklisted token")

                httputil.RespondProblem(w, r, httputil.ProblemUnauthorized(
                    "Token has been revoked",
                ))
                return
            }
        }

        // Step 4: Validate token signature and claims
        claims, err := m.jwtService.ValidateToken(tokenString)
        if err != nil {
            m.logger.Warn().
                Err(err).
                Str("event", "token_validation_failed").
                Str("path", r.URL.Path).
                Msg("invalid token")

            httputil.RespondProblem(w, r, httputil.ProblemUnauthorized(
                "Invalid or expired token",
            ))
            return
        }

        // Step 5: Verify token type (must be access token)
        if claims.TokenType != jwt.TokenTypeAccess {
            m.logger.Warn().
                Str("event", "wrong_token_type").
                Str("token_type", string(claims.TokenType)).
                Str("path", r.URL.Path).
                Msg("wrong token type used for authentication")

            httputil.RespondProblem(w, r, httputil.ProblemUnauthorized(
                "Invalid token type",
            ))
            return
        }

        // Step 6: Set user context for downstream handlers
        ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
        ctx = context.WithValue(ctx, EmailKey, claims.Email)
        ctx = context.WithValue(ctx, RoleKey, claims.Role)
        ctx = context.WithValue(ctx, SessionIDKey, claims.SessionID)

        // Step 7: Log successful authentication
        m.logger.Debug().
            Str("event", "auth_success").
            Str("user_id", claims.UserID).
            Str("role", claims.Role).
            Str("path", r.URL.Path).
            Msg("request authenticated")

        // Pass to next handler
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Helper functions to extract user info from context
func GetUserID(ctx context.Context) (string, bool) {
    userID, ok := ctx.Value(UserIDKey).(string)
    return userID, ok
}

func GetEmail(ctx context.Context) (string, bool) {
    email, ok := ctx.Value(EmailKey).(string)
    return email, ok
}

func GetRole(ctx context.Context) (string, bool) {
    role, ok := ctx.Value(RoleKey).(string)
    return role, ok
}

func GetSessionID(ctx context.Context) (string, bool) {
    sessionID, ok := ctx.Value(SessionIDKey).(string)
    return sessionID, ok
}

// MustGetUserID panics if user ID not in context (use in protected routes only)
func MustGetUserID(ctx context.Context) string {
    userID, ok := GetUserID(ctx)
    if !ok {
        panic("user_id not found in context - did you forget JWT middleware?")
    }
    return userID
}
```

### RBAC Middleware

```go
// RequireRole enforces role-based access control
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            role, ok := GetRole(r.Context())
            if !ok {
                httputil.RespondProblem(w, r, httputil.ProblemUnauthorized(
                    "User role not found",
                ))
                return
            }

            if role != requiredRole {
                httputil.RespondProblem(w, r, httputil.ProblemForbidden(
                    fmt.Sprintf("Requires %s role", requiredRole),
                ))
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// RequireAnyRole accepts multiple roles (OR logic)
func RequireAnyRole(allowedRoles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            role, ok := GetRole(r.Context())
            if !ok {
                httputil.RespondProblem(w, r, httputil.ProblemUnauthorized(
                    "User role not found",
                ))
                return
            }

            for _, allowedRole := range allowedRoles {
                if role == allowedRole {
                    next.ServeHTTP(w, r)
                    return
                }
            }

            httputil.RespondProblem(w, r, httputil.ProblemForbidden(
                fmt.Sprintf("Requires one of: %v", allowedRoles),
            ))
        })
    }
}

// RequirePermission checks fine-grained permissions
func RequirePermission(permission string) func(http.Handler) http.Handler {
    // Map roles to permissions
    rolePermissions := map[string][]string{
        "admin": {
            "image:upload", "image:delete", "image:delete:any", "image:moderate",
            "user:read", "user:update", "user:ban", "user:manage:roles",
            "report:view", "report:resolve",
        },
        "moderator": {
            "image:upload", "image:delete", "image:moderate",
            "report:view", "report:resolve",
        },
        "user": {
            "image:upload", "image:delete",
        },
    }

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            role, ok := GetRole(r.Context())
            if !ok {
                httputil.RespondProblem(w, r, httputil.ProblemUnauthorized(
                    "User role not found",
                ))
                return
            }

            permissions, exists := rolePermissions[role]
            if !exists {
                httputil.RespondProblem(w, r, httputil.ProblemForbidden(
                    "Unknown role",
                ))
                return
            }

            // Check if user has required permission
            for _, perm := range permissions {
                if perm == permission {
                    next.ServeHTTP(w, r)
                    return
                }
            }

            httputil.RespondProblem(w, r, httputil.ProblemForbidden(
                fmt.Sprintf("Requires permission: %s", permission),
            ))
        })
    }
}
```

## Error Handling (RFC 7807)

### Security-Specific Problem Details

All security errors use RFC 7807 Problem Details format. See `internal/interfaces/http/CLAUDE.md` for general error handling patterns.

**Unauthorized (401)**:
```go
func ProblemUnauthorized(detail string) ProblemDetail {
    return ProblemDetail{
        Type:   "https://api.goimg.dev/problems/unauthorized",
        Title:  "Unauthorized",
        Status: http.StatusUnauthorized,
        Detail: detail,
    }
}

// Usage examples:
ProblemUnauthorized("Missing authorization header")
ProblemUnauthorized("Invalid or expired token")
ProblemUnauthorized("Token has been revoked")
```

**Forbidden (403)**:
```go
func ProblemForbidden(detail string) ProblemDetail {
    return ProblemDetail{
        Type:   "https://api.goimg.dev/problems/forbidden",
        Title:  "Forbidden",
        Status: http.StatusForbidden,
        Detail: detail,
    }
}

// Usage examples:
ProblemForbidden("Requires admin role")
ProblemForbidden("Insufficient permissions")
ProblemForbidden("Account temporarily locked")
```

**Rate Limit Exceeded (429)**:
```go
func ProblemRateLimitExceeded(limit int, window time.Duration, retryAfter int) ProblemDetail {
    return ProblemDetail{
        Type:   "https://api.goimg.dev/problems/rate-limit-exceeded",
        Title:  "Rate Limit Exceeded",
        Status: http.StatusTooManyRequests,
        Detail: fmt.Sprintf("You have exceeded the rate limit of %d requests per %s", limit, window),
        Extras: map[string]interface{}{
            "limit":      limit,
            "window":     window.String(),
            "retryAfter": retryAfter,
        },
    }
}
```

### Information Disclosure Prevention

**NEVER return**:
- Stack traces (caught by recovery middleware)
- Database error details (e.g., "duplicate key violation")
- Internal service names or versions
- File system paths
- User existence confirmation (timing attacks)

**DO return**:
- Generic error messages
- Trace ID for support correlation
- Expected error types (validation, not found, etc.)
- Rate limit information (helps legitimate clients)

**Example - Bad**:
```go
// WRONG: Reveals internal implementation
return ProblemInternalError(fmt.Sprintf("postgres error: %v", err))
```

**Example - Good**:
```go
// CORRECT: Generic message, log details internally
logger.Error().Err(err).Msg("database query failed")
return ProblemInternalError() // Generic "internal server error"
```

## Testing Requirements

### Unit Tests

Test each middleware in isolation:

```go
func TestJWTAuth_ValidToken(t *testing.T) {
    // Create test JWT service with test keys
    // Generate valid token
    // Create middleware
    // Make request with token
    // Assert: context has user info
}

func TestJWTAuth_MissingToken(t *testing.T) {
    // Make request without Authorization header
    // Assert: 401 response
}

func TestJWTAuth_BlacklistedToken(t *testing.T) {
    // Generate token
    // Add to blacklist
    // Make request with token
    // Assert: 401 response with "token revoked" message
}

func TestRateLimiter_Exceeded(t *testing.T) {
    // Make (limit + 1) requests
    // Assert: last request returns 429
    // Assert: X-RateLimit-* headers present
}

func TestSecurityHeaders_AllPresent(t *testing.T) {
    // Make request through middleware
    // Assert: all required headers present
}
```

### Integration Tests

Test middleware combinations:

```go
func TestAuthFlow_EndToEnd(t *testing.T) {
    // 1. Login (rate limited)
    // 2. Access protected route (JWT + rate limit)
    // 3. Logout (blacklist token)
    // 4. Attempt access with logged-out token (should fail)
}

func TestAccountLockout_EndToEnd(t *testing.T) {
    // 1. Make 5 failed login attempts
    // 2. Assert: 6th attempt returns 403 (locked)
    // 3. Wait 15 minutes (or mock time)
    // 4. Assert: login succeeds
}
```

### Coverage Target

- **Middleware**: 90% (security-critical code)
- Focus on error paths (invalid tokens, rate limits, blacklists)
- Test boundary conditions (exactly at limit, token expires now, etc.)

## Security Checklist

Before deploying middleware:

- [ ] All middleware follows correct execution order
- [ ] Rate limiting prevents brute-force (5 login attempts/min)
- [ ] JWT blacklist checked before signature verification
- [ ] Account lockout implemented (5 attempts, 15 min)
- [ ] Constant-time comparison for sensitive operations
- [ ] Generic error messages (no enumeration)
- [ ] Session ID regenerated on login
- [ ] Security headers set on all responses
- [ ] HSTS disabled in development, enabled in production
- [ ] Audit logging covers all security events
- [ ] No sensitive data in logs (passwords, tokens)
- [ ] Rate limit headers included in responses
- [ ] RFC 7807 error format for all security errors
- [ ] Unit tests cover all error paths
- [ ] Integration tests validate end-to-end flows

## Performance Considerations

### Redis Operations

- **Token validation**: 1 blacklist check (GET) per request → ~1ms
- **Rate limiting**: 1-2 operations (INCR + EXPIRE) per request → ~1-2ms
- **Session lookup**: Optional, not required for JWT validation

**Total overhead**: ~2-3ms per authenticated request

### Optimization Strategies

1. **Connection pooling**: Reuse Redis connections (go-redis handles this)
2. **Pipeline operations**: Batch Redis commands when possible
3. **Cache JWT public key**: Avoid file I/O on every validation
4. **Skip blacklist for expired tokens**: If token.exp < now, no need to check blacklist

### Monitoring

Track middleware performance:
- JWT validation latency (p50, p95, p99)
- Redis operation latency
- Rate limit check latency
- Blacklist hit rate (should be <1% in normal operation)

Alert on:
- JWT validation latency > 50ms
- Redis operation failures
- High blacklist hit rate (>5% = potential attack)

## References

- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [RFC 7807: Problem Details for HTTP APIs](https://datatracker.ietf.org/doc/html/rfc7807)
- [RFC 6749: OAuth 2.0 Authorization Framework](https://datatracker.ietf.org/doc/html/rfc6749)
- [NIST SP 800-63B: Digital Identity Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html)

## Agent Responsibilities

- **senior-secops-engineer** (YOU): Reviews all middleware for security vulnerabilities
- **senior-go-architect**: Reviews middleware design patterns and performance
- **backend-developer**: Implements middleware following this guide
- **test-strategist**: Ensures 90% test coverage with comprehensive security test cases

## See Also

- JWT implementation: `internal/infrastructure/security/CLAUDE.md`
- HTTP layer patterns: `internal/interfaces/http/CLAUDE.md`
- API security: `claude/api_security.md`
- Security testing: `claude/security_testing.md`
