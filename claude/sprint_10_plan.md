# Sprint 10: Security Enhancements - Implementation Plan

> **Status**: Planning Phase
> **Start Date**: TBD (Post-Launch)
> **Duration**: 2 weeks
> **Focus**: Random Login Delay & HIBP Password Validation

---

## Executive Summary

Sprint 10 implements two critical security enhancements deferred from MVP:

1. **Random Login Delay**: Timing attack mitigation for authentication to prevent credential enumeration through response time analysis
2. **HIBP Password Check**: Integration with "Have I Been Pwned" API using k-anonymity to reject compromised passwords

Both features enhance defense-in-depth without significantly impacting legitimate user experience.

---

## Research Summary

### Random Login Delay - Best Practices

**Key Findings** ([source](https://www.slingacademy.com/article/avoiding-timing-attacks-with-constant-time-comparisons-in-go/)):
- Go's `crypto/subtle` package provides constant-time comparison functions
- Password comparison should always use hashed values (already implemented with Argon2id)
- Random delays mask timing variations across network requests
- Delays should be applied to BOTH successful and failed attempts

**Current Implementation Status**:
- ✅ Already using `crypto/subtle.ConstantTimeCompare` in `password.go:160`
- ✅ Argon2id hashing with constant-time verification
- ❌ No random delay to mask overall response time
- ❌ Failed logins return faster than successful ones (token generation time)

**Mitigation Strategy** ([source](https://github.com/globaleaks/GlobaLeaks/issues/264)):
- Add random delay (100-300ms) to all authentication attempts
- Apply delay AFTER credential verification but BEFORE response
- Delay range should be carefully tuned to balance security vs UX
- Log actual processing time separately for monitoring

### HIBP Password Check - Best Practices

**Official API** ([source](https://haveibeenpwned.com/API/v3)):
- **k-Anonymity Model**: Only sends first 5 characters of SHA-1 hash
- **No API Key Required**: Pwned Passwords API is freely accessible
- **Cache-Friendly**: Responses can be cached for 24 hours
- **No Rate Limiting**: Designed for high-volume usage

**Go Implementation Options**:
1. **[github.com/wneessen/go-hibp](https://pkg.go.dev/github.com/wneessen/go-hibp)** (Feb 2025, MIT License)
   - Complete HIBP API bindings
   - Supports all 3 APIs (Breaches, Pastes, Passwords)
   - Well-maintained, recent updates

2. **[github.com/supabase/hibp](https://github.com/supabase/hibp)** (Supabase official)
   - Pwned Passwords v3 only
   - Zero external dependencies (supply-chain security)
   - Concurrent request optimization
   - Built-in caching support

**Recommended**: Use `supabase/hibp` for supply-chain security and simplicity

**k-Anonymity Example**:
```
Password: "P@ssw0rd123"
SHA-1: "2AC9A6746ACA543AF8DFF39894CBC7B2D299FCAE"
API Call: GET https://api.pwnedpasswords.com/range/2AC9A
Response: (list of hash suffixes with counts)
Client checks if "6746ACA543AF8DFF39894CBC7B2D299FCAE" is in response
```

**Security Considerations**:
- Never send full password or hash to API
- API failures should NOT block registration (fail open)
- Log API failures for monitoring
- Consider rate limiting client-side to be good citizen
- Cache negative results for performance

---

## Feature 1: Random Login Delay

### 1.1 Technical Approach

**Goals**:
- Prevent timing-based credential enumeration
- Mask difference between failed credentials and successful login
- Minimal impact on legitimate users (<300ms delay)
- Maintain audit trail of actual processing time

**Implementation Strategy**:
1. Calculate random delay duration BEFORE authentication starts
2. Track actual processing time during authentication
3. Sleep for (random_delay - actual_time) if positive, else minimum delay
4. Apply to both successful and failed authentication attempts
5. Log both delays for monitoring and tuning

**Delay Configuration**:
```go
const (
    MinAuthDelay = 100 * time.Millisecond  // Minimum guaranteed delay
    MaxAuthDelay = 300 * time.Millisecond  // Maximum delay
)
```

**Rationale**:
- 100-300ms range is imperceptible to humans but significant for timing attacks
- Randomization prevents averaging attacks over multiple requests
- Logarithmic distribution favors shorter delays for better UX

### 1.2 Files to Modify

#### 1. `internal/application/identity/commands/login.go`

**Changes**:
- Add timing instrumentation to `LoginHandler.Handle()`
- Calculate random delay at function entry
- Track authentication processing time
- Apply compensating sleep before return

**Pseudocode**:
```go
func (h *LoginHandler) Handle(ctx context.Context, cmd LoginCommand) (*dto.AuthResponseDTO, error) {
    // 1. Calculate random delay target (100-300ms)
    targetDelay := calculateRandomDelay()
    startTime := time.Now()

    // 2. Defer sleep to ensure it happens on all return paths
    defer func() {
        actualDuration := time.Since(startTime)
        remainingDelay := targetDelay - actualDuration

        if remainingDelay > 0 {
            time.Sleep(remainingDelay)
        } else {
            // Processing took longer than target, apply minimum delay
            time.Sleep(MinAuthDelay)
        }

        h.logger.Debug().
            Dur("target_delay", targetDelay).
            Dur("actual_processing", actualDuration).
            Dur("applied_delay", remainingDelay).
            Msg("login timing completed")
    }()

    // 3. Existing authentication logic unchanged
    user, err := h.findUserByIdentifier(ctx, cmd.Identifier)
    // ... rest of existing logic
}
```

**Security Considerations**:
- Use `crypto/rand` for delay calculation, not `math/rand`
- Apply delay even if early returns (database errors, etc.)
- Log delays separately (not in user-facing logs)
- Consider context cancellation during sleep

#### 2. Create new helper: `internal/application/identity/timing.go`

**Purpose**: Centralize timing attack mitigation logic

**Functions**:
```go
package identity

import (
    "crypto/rand"
    "math/big"
    "time"
)

const (
    MinAuthDelay = 100 * time.Millisecond
    MaxAuthDelay = 300 * time.Millisecond
)

// CalculateRandomAuthDelay generates a cryptographically random delay
// within the configured range for timing attack mitigation.
func CalculateRandomAuthDelay() time.Duration {
    // Generate random number in range [0, 200]
    rangeMs := int64(MaxAuthDelay-MinAuthDelay) / int64(time.Millisecond)
    randomMs, err := rand.Int(rand.Reader, big.NewInt(rangeMs))
    if err != nil {
        // Fallback to max delay if randomness fails
        return MaxAuthDelay
    }

    return MinAuthDelay + time.Duration(randomMs.Int64())*time.Millisecond
}

// ApplyAuthDelay sleeps for the remaining time to reach the target delay,
// or applies a minimum delay if processing exceeded the target.
func ApplyAuthDelay(targetDelay time.Duration, actualProcessing time.Duration) time.Duration {
    remaining := targetDelay - actualProcessing

    if remaining > 0 {
        time.Sleep(remaining)
        return remaining
    }

    // Processing took longer than target, apply minimum delay
    time.Sleep(MinAuthDelay)
    return MinAuthDelay
}
```

**Testing Strategy**:
- Unit test `CalculateRandomAuthDelay()` with statistical distribution check
- Mock `time.Now()` in tests to verify delay calculation
- Integration test to ensure delays are applied in all code paths
- Verify delay distribution over 1000 samples falls within range

#### 3. Configuration (optional): `internal/infrastructure/config/config.go`

**Add to AuthConfig**:
```go
type AuthConfig struct {
    // Existing fields...
    JWTSecret          string
    AccessTokenTTL     time.Duration
    RefreshTokenTTL    time.Duration

    // New fields
    MinAuthDelay       time.Duration `envconfig:"AUTH_MIN_DELAY" default:"100ms"`
    MaxAuthDelay       time.Duration `envconfig:"AUTH_MAX_DELAY" default:"300ms"`
}
```

**Rationale**: Allows tuning in production without code changes

### 1.3 Testing Requirements

#### Unit Tests (`internal/application/identity/timing_test.go`)

```go
func TestCalculateRandomAuthDelay(t *testing.T) {
    t.Run("delay within range", func(t *testing.T) {
        for i := 0; i < 100; i++ {
            delay := CalculateRandomAuthDelay()
            assert.GreaterOrEqual(t, delay, MinAuthDelay)
            assert.LessOrEqual(t, delay, MaxAuthDelay)
        }
    })

    t.Run("distribution is random", func(t *testing.T) {
        delays := make([]time.Duration, 1000)
        for i := 0; i < 1000; i++ {
            delays[i] = CalculateRandomAuthDelay()
        }

        // Calculate standard deviation
        // Should be approximately (max-min)/sqrt(12) for uniform distribution
        stdDev := calculateStdDev(delays)
        expectedStdDev := float64(MaxAuthDelay-MinAuthDelay) / math.Sqrt(12)

        // Allow 20% variance from expected
        assert.InDelta(t, expectedStdDev, stdDev, expectedStdDev*0.2)
    })
}

func TestApplyAuthDelay(t *testing.T) {
    t.Run("applies remaining delay", func(t *testing.T) {
        target := 200 * time.Millisecond
        actual := 50 * time.Millisecond

        start := time.Now()
        applied := ApplyAuthDelay(target, actual)
        elapsed := time.Since(start)

        assert.Equal(t, 150*time.Millisecond, applied)
        assert.InDelta(t, 150*time.Millisecond, elapsed, float64(10*time.Millisecond))
    })

    t.Run("applies minimum delay when exceeded", func(t *testing.T) {
        target := 200 * time.Millisecond
        actual := 250 * time.Millisecond

        start := time.Now()
        applied := ApplyAuthDelay(target, actual)
        elapsed := time.Since(start)

        assert.Equal(t, MinAuthDelay, applied)
        assert.InDelta(t, MinAuthDelay, elapsed, float64(10*time.Millisecond))
    })
}
```

#### Integration Tests (`internal/application/identity/commands/login_test.go`)

**Add new test cases**:
```go
func TestLoginHandler_TimingAttackMitigation(t *testing.T) {
    t.Run("failed login has consistent timing", func(t *testing.T) {
        handler := setupLoginHandler(t)

        timings := make([]time.Duration, 10)
        for i := 0; i < 10; i++ {
            start := time.Now()
            _, err := handler.Handle(ctx, LoginCommand{
                Identifier: "nonexistent@example.com",
                Password:   "wrongpassword",
            })
            timings[i] = time.Since(start)

            assert.Error(t, err)
        }

        // All timings should be >= MinAuthDelay
        for _, timing := range timings {
            assert.GreaterOrEqual(t, timing, MinAuthDelay)
        }

        // Standard deviation should be relatively small
        stdDev := calculateStdDev(timings)
        assert.Less(t, stdDev, float64(50*time.Millisecond))
    })

    t.Run("successful login has similar timing to failed", func(t *testing.T) {
        // Compare timing distributions between success and failure
        // Should not be distinguishable without large sample size
    })
}
```

#### E2E Tests (Newman/Postman)

**Add to `tests/e2e/postman/goimg-api.postman_collection.json`**:

```json
{
    "name": "Login Timing Attack Protection",
    "event": [
        {
            "listen": "test",
            "script": {
                "exec": [
                    "// Measure response time",
                    "const responseTime = pm.response.responseTime;",
                    "",
                    "// Should have minimum delay (>= 100ms)",
                    "pm.test('Response time >= 100ms (timing protection)', function() {",
                    "    pm.expect(responseTime).to.be.at.least(100);",
                    "});",
                    "",
                    "// Should not be excessively slow (< 500ms)",
                    "pm.test('Response time < 500ms (reasonable UX)', function() {",
                    "    pm.expect(responseTime).to.be.below(500);",
                    "});"
                ]
            }
        }
    ],
    "request": {
        "method": "POST",
        "header": [],
        "body": {
            "mode": "raw",
            "raw": "{\n  \"email\": \"invalid@example.com\",\n  \"password\": \"wrongpassword\"\n}"
        },
        "url": "{{base_url}}/api/v1/auth/login"
    }
}
```

### 1.4 Security Validation

**Checklist**:
- [ ] Delay uses cryptographically secure randomness (`crypto/rand`)
- [ ] Delay applied to ALL authentication code paths (success, failure, error)
- [ ] Timing information logged separately from user-facing logs
- [ ] No timing information leaked in error messages
- [ ] Delay configuration cannot be disabled in production
- [ ] Integration tests verify consistent timing across code paths

**Penetration Test Scenarios**:
1. Measure response time for valid vs invalid credentials over 1000 attempts
2. Calculate statistical difference (should not be significant at p < 0.05)
3. Test with network latency variations
4. Verify delay persists under high concurrent load

### 1.5 Performance Impact

**Expected Impact**:
- **Successful logins**: +100-300ms (acceptable for security)
- **Failed logins**: +100-300ms (prevents brute force, acceptable)
- **Throughput**: Minimal impact (delay is per-request, not global)
- **Resource usage**: Negligible (sleep is non-blocking)

**Monitoring**:
- Add Prometheus metrics: `auth_login_delay_seconds` (histogram)
- Track actual processing time separately: `auth_login_processing_seconds`
- Alert if delays consistently exceed target (indicates slow backend)

---

## Feature 2: HIBP Password Check

### 2.1 Technical Approach

**Goals**:
- Reject passwords found in known data breaches
- Protect user privacy (never send full password or hash)
- Graceful degradation (API failures do not block registration)
- Performance-friendly (caching, timeouts)

**Implementation Strategy**:
1. Check passwords on registration and password change operations
2. Use k-anonymity API (only send first 5 SHA-1 hash characters)
3. Fail open with logging if API unavailable (don't block legitimate users)
4. Cache negative results (password not pwned) for 24 hours
5. Add configuration toggle for testing/development

**Password Check Flow**:
```
1. User submits password during registration
2. Hash password with SHA-1 (for HIBP only, NOT for storage)
3. Send first 5 chars of hash to HIBP API
4. Receive list of suffix matches
5. Check if our suffix is in list
   - If found: Reject with user-friendly error
   - If not found: Continue with registration
   - If API error: Log warning, allow registration
```

### 2.2 Files to Modify

#### 1. Add dependency to `go.mod`

```bash
go get github.com/supabase/hibp
```

**Rationale**: Zero external dependencies, cache support, concurrent optimization

#### 2. Create new service: `internal/infrastructure/security/hibp_client.go`

**Purpose**: Encapsulate HIBP API client with caching and error handling

```go
package security

import (
    "context"
    "crypto/sha1"
    "encoding/hex"
    "fmt"
    "strings"
    "time"

    "github.com/rs/zerolog"
    "github.com/supabase/hibp"
)

// HIBPConfig holds configuration for HIBP password checking
type HIBPConfig struct {
    Enabled        bool          `envconfig:"HIBP_ENABLED" default:"true"`
    Timeout        time.Duration `envconfig:"HIBP_TIMEOUT" default:"5s"`
    CacheTTL       time.Duration `envconfig:"HIBP_CACHE_TTL" default:"24h"`
    FailOpen       bool          `envconfig:"HIBP_FAIL_OPEN" default:"true"`
}

// PasswordChecker defines the interface for checking compromised passwords
type PasswordChecker interface {
    IsPasswordPwned(ctx context.Context, password string) (bool, error)
}

// HIBPClient wraps the HIBP API with caching and error handling
type HIBPClient struct {
    client  *hibp.Client
    cache   PasswordCache
    config  HIBPConfig
    logger  *zerolog.Logger
}

// NewHIBPClient creates a new HIBP client with caching
func NewHIBPClient(cache PasswordCache, config HIBPConfig, logger *zerolog.Logger) *HIBPClient {
    return &HIBPClient{
        client: hibp.New(),
        cache:  cache,
        config: config,
        logger: logger,
    }
}

// IsPasswordPwned checks if a password has been compromised in a data breach
// Uses k-anonymity (only sends first 5 hash characters)
func (c *HIBPClient) IsPasswordPwned(ctx context.Context, password string) (bool, error) {
    if !c.config.Enabled {
        c.logger.Debug().Msg("HIBP check disabled")
        return false, nil
    }

    // 1. Generate SHA-1 hash (HIBP uses SHA-1, NOT for storage)
    hash := sha1.Sum([]byte(password))
    hashStr := strings.ToUpper(hex.EncodeToString(hash[:]))

    // 2. Check cache first (prefix:suffix format)
    prefix := hashStr[:5]
    suffix := hashStr[5:]

    if cached, found := c.cache.Get(ctx, prefix, suffix); found {
        c.logger.Debug().
            Str("cache_result", fmt.Sprintf("%v", cached)).
            Msg("HIBP cache hit")
        return cached, nil
    }

    // 3. Call HIBP API with timeout
    apiCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
    defer cancel()

    pwned, err := c.client.IsPwned(apiCtx, password)
    if err != nil {
        c.logger.Warn().
            Err(err).
            Bool("fail_open", c.config.FailOpen).
            Msg("HIBP API request failed")

        // Fail open: don't block registration on API errors
        if c.config.FailOpen {
            return false, nil
        }
        return false, fmt.Errorf("password breach check failed: %w", err)
    }

    // 4. Cache result (negative results for 24h, positive indefinitely)
    ttl := c.config.CacheTTL
    if pwned {
        ttl = 0 // Cache forever for pwned passwords
    }
    c.cache.Set(ctx, prefix, suffix, pwned, ttl)

    c.logger.Info().
        Bool("pwned", pwned).
        Msg("HIBP check completed")

    return pwned, nil
}
```

#### 3. Create cache interface: `internal/infrastructure/security/password_cache.go`

**Purpose**: Abstract caching for testability and flexibility

```go
package security

import (
    "context"
    "time"
)

// PasswordCache defines the interface for caching HIBP results
type PasswordCache interface {
    Get(ctx context.Context, prefix, suffix string) (bool, bool)
    Set(ctx context.Context, prefix, suffix string, pwned bool, ttl time.Duration) error
}

// RedisPasswordCache implements PasswordCache using Redis
type RedisPasswordCache struct {
    client *redis.Client
}

func NewRedisPasswordCache(client *redis.Client) *RedisPasswordCache {
    return &RedisPasswordCache{client: client}
}

func (c *RedisPasswordCache) Get(ctx context.Context, prefix, suffix string) (bool, bool) {
    key := fmt.Sprintf("goimg:hibp:%s:%s", prefix, suffix)

    val, err := c.client.Get(ctx, key).Result()
    if err == redis.Nil {
        return false, false // Not cached
    }
    if err != nil {
        return false, false // Error, treat as miss
    }

    return val == "1", true
}

func (c *RedisPasswordCache) Set(ctx context.Context, prefix, suffix string, pwned bool, ttl time.Duration) error {
    key := fmt.Sprintf("goimg:hibp:%s:%s", prefix, suffix)
    value := "0"
    if pwned {
        value = "1"
    }

    return c.client.Set(ctx, key, value, ttl).Err()
}
```

**Redis Key Pattern**: `goimg:hibp:{prefix}:{suffix}` → `"0"` or `"1"`

#### 4. Update domain errors: `internal/domain/identity/errors.go`

**Add new error**:
```go
var (
    // Existing errors...
    ErrPasswordEmpty        = errors.New("password cannot be empty")
    ErrPasswordTooShort     = errors.New("password must be at least 12 characters")
    ErrPasswordTooLong      = errors.New("password exceeds 128 characters")
    ErrPasswordWeak         = errors.New("password is too common or weak")

    // New error
    ErrPasswordCompromised  = errors.New("password has been found in a data breach and cannot be used")
)
```

#### 5. Update registration handler: `internal/application/identity/commands/register_user.go`

**Add HIBP check before password hashing**:

```go
type RegisterUserHandler struct {
    users           identity.UserRepository
    eventPublisher  appidentity.EventPublisher
    passwordChecker security.PasswordChecker  // NEW
    logger          *zerolog.Logger
}

func NewRegisterUserHandler(
    users identity.UserRepository,
    eventPublisher appidentity.EventPublisher,
    passwordChecker security.PasswordChecker,  // NEW
    logger *zerolog.Logger,
) *RegisterUserHandler {
    return &RegisterUserHandler{
        users:           users,
        eventPublisher:  eventPublisher,
        passwordChecker: passwordChecker,  // NEW
        logger:          logger,
    }
}

func (h *RegisterUserHandler) Handle(ctx context.Context, cmd RegisterUserCommand) (*dto.UserDTO, error) {
    // ... existing email/username validation ...

    // NEW: Check if password is compromised BEFORE hashing
    if h.passwordChecker != nil {
        pwned, err := h.passwordChecker.IsPasswordPwned(ctx, cmd.Password)
        if err != nil {
            // Log error but don't fail registration if configured for fail-open
            h.logger.Warn().
                Err(err).
                Msg("HIBP check failed during registration")
        }
        if pwned {
            h.logger.Warn().
                Str("email", email.String()).
                Msg("registration blocked due to compromised password")
            return nil, identity.ErrPasswordCompromised
        }
    }

    // Existing: Hash password using Argon2id
    passwordHash, err := identity.NewPasswordHash(cmd.Password)
    // ... rest of existing logic ...
}
```

#### 6. Update password change handler (if exists)

**Similar changes to any password change operations**

#### 7. Update HTTP error mapping: `internal/interfaces/http/handlers/auth_handler.go`

**Add case for compromised password**:

```go
func (h *AuthHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
    // ... existing cases ...

    case errors.Is(err, identity.ErrPasswordCompromised):
        middleware.WriteError(w, r,
            http.StatusBadRequest,
            "Password Compromised",
            "This password has been found in a data breach and cannot be used. Please choose a different password.",
        )

    // ... rest of cases ...
}
```

### 2.3 Testing Requirements

#### Unit Tests (`internal/infrastructure/security/hibp_client_test.go`)

```go
func TestHIBPClient_IsPasswordPwned(t *testing.T) {
    t.Run("detects known compromised password", func(t *testing.T) {
        cache := NewMockPasswordCache()
        config := HIBPConfig{
            Enabled:  true,
            Timeout:  5 * time.Second,
            FailOpen: true,
        }
        client := NewHIBPClient(cache, config, &zerolog.Nop())

        // Test with known compromised password
        pwned, err := client.IsPasswordPwned(context.Background(), "password123")

        assert.NoError(t, err)
        assert.True(t, pwned)
    })

    t.Run("accepts strong unique password", func(t *testing.T) {
        cache := NewMockPasswordCache()
        client := setupHIBPClient(cache)

        // Test with strong random password (very unlikely to be pwned)
        strongPassword := generateRandomPassword(32)
        pwned, err := client.IsPasswordPwned(context.Background(), strongPassword)

        assert.NoError(t, err)
        assert.False(t, pwned)
    })

    t.Run("uses cache on second check", func(t *testing.T) {
        cache := NewMockPasswordCache()
        client := setupHIBPClient(cache)

        password := "testpassword123"

        // First check
        _, _ = client.IsPasswordPwned(context.Background(), password)

        // Second check should hit cache
        cache.ResetCallCount()
        pwned, err := client.IsPasswordPwned(context.Background(), password)

        assert.NoError(t, err)
        assert.Equal(t, 1, cache.GetCallCount())
        assert.Equal(t, 0, cache.APICallCount) // No API call
    })

    t.Run("fails open when API unavailable", func(t *testing.T) {
        cache := NewMockPasswordCache()
        config := HIBPConfig{
            Enabled:  true,
            Timeout:  1 * time.Millisecond, // Force timeout
            FailOpen: true,
        }
        client := NewHIBPClient(cache, config, &zerolog.Nop())

        pwned, err := client.IsPasswordPwned(context.Background(), "anypassword")

        assert.NoError(t, err)
        assert.False(t, pwned) // Fails open, allows password
    })

    t.Run("respects disabled configuration", func(t *testing.T) {
        cache := NewMockPasswordCache()
        config := HIBPConfig{Enabled: false}
        client := NewHIBPClient(cache, config, &zerolog.Nop())

        pwned, err := client.IsPasswordPwned(context.Background(), "password")

        assert.NoError(t, err)
        assert.False(t, pwned)
        assert.Equal(t, 0, cache.APICallCount)
    })
}
```

#### Integration Tests (`internal/application/identity/commands/register_user_test.go`)

**Add new test cases**:

```go
func TestRegisterUserHandler_HIBPIntegration(t *testing.T) {
    t.Run("rejects compromised password", func(t *testing.T) {
        handler := setupHandlerWithHIBP(t)

        cmd := RegisterUserCommand{
            Email:    "test@example.com",
            Username: "testuser",
            Password: "password123", // Known compromised password
        }

        _, err := handler.Handle(context.Background(), cmd)

        assert.ErrorIs(t, err, identity.ErrPasswordCompromised)
    })

    t.Run("accepts strong unique password", func(t *testing.T) {
        handler := setupHandlerWithHIBP(t)

        cmd := RegisterUserCommand{
            Email:    "test@example.com",
            Username: "testuser",
            Password: generateRandomPassword(16),
        }

        user, err := handler.Handle(context.Background(), cmd)

        assert.NoError(t, err)
        assert.NotNil(t, user)
    })

    t.Run("allows registration when HIBP unavailable", func(t *testing.T) {
        // Configure handler with unreachable HIBP endpoint
        handler := setupHandlerWithFailingHIBP(t)

        cmd := RegisterUserCommand{
            Email:    "test@example.com",
            Username: "testuser",
            Password: "anypassword",
        }

        user, err := handler.Handle(context.Background(), cmd)

        assert.NoError(t, err) // Fails open
        assert.NotNil(t, user)
    })
}
```

#### E2E Tests (Newman/Postman)

**Add to `tests/e2e/postman/goimg-api.postman_collection.json`**:

```json
{
    "name": "Register with Compromised Password",
    "event": [
        {
            "listen": "test",
            "script": {
                "exec": [
                    "pm.test('Status code is 400', function() {",
                    "    pm.response.to.have.status(400);",
                    "});",
                    "",
                    "pm.test('Error indicates compromised password', function() {",
                    "    const json = pm.response.json();",
                    "    pm.expect(json.title).to.include('Password Compromised');",
                    "    pm.expect(json.detail).to.include('data breach');",
                    "});"
                ]
            }
        }
    ],
    "request": {
        "method": "POST",
        "header": [],
        "body": {
            "mode": "raw",
            "raw": "{\n  \"email\": \"newuser@example.com\",\n  \"username\": \"newuser\",\n  \"password\": \"password123\"\n}"
        },
        "url": "{{base_url}}/api/v1/auth/register"
    }
},
{
    "name": "Register with Strong Password",
    "event": [
        {
            "listen": "test",
            "script": {
                "exec": [
                    "pm.test('Status code is 201', function() {",
                    "    pm.response.to.have.status(201);",
                    "});",
                    "",
                    "pm.test('User created successfully', function() {",
                    "    const json = pm.response.json();",
                    "    pm.expect(json.id).to.exist;",
                    "    pm.expect(json.email).to.equal(pm.variables.get('test_email'));",
                    "});"
                ]
            }
        }
    ],
    "request": {
        "method": "POST",
        "header": [],
        "body": {
            "mode": "raw",
            "raw": "{\n  \"email\": \"{{test_email}}\",\n  \"username\": \"secureuser\",\n  \"password\": \"{{strong_password}}\"\n}"
        },
        "url": "{{base_url}}/api/v1/auth/register"
    }
}
```

### 2.4 Security Validation

**Checklist**:
- [ ] Never send full password or complete hash to API (only 5 chars)
- [ ] SHA-1 hash is ONLY for HIBP, NOT for password storage (still Argon2id)
- [ ] API failures fail open (don't block legitimate users)
- [ ] Cache implementation prevents timing attacks (constant-time lookup)
- [ ] Error messages don't reveal if API is down
- [ ] Compromised password error is user-friendly and actionable
- [ ] No PII logged in HIBP requests
- [ ] Rate limiting exists (client-side and server-side)

**Privacy Considerations**:
- k-anonymity ensures HIBP never sees full password
- Local cache reduces API calls (fewer exposure points)
- Redis cache keys use prefix:suffix (no full hashes)
- Logs never contain passwords (only "pwned" boolean)

### 2.5 Performance Impact

**Expected Impact**:
- **Cache hit**: <1ms (Redis lookup)
- **Cache miss**: 100-500ms (HIBP API call)
- **API timeout**: 5s max (configurable)
- **Overall**: 95% cache hit rate expected, minimal user impact

**Caching Strategy**:
- Negative results: 24 hours (passwords not pwned)
- Positive results: Indefinite (once pwned, always pwned)
- Cache invalidation: Not needed (HIBP data only grows)

**Monitoring**:
- Add Prometheus metrics:
  - `hibp_checks_total{result="pwned|clean|error"}`
  - `hibp_cache_hits_total`
  - `hibp_api_latency_seconds` (histogram)
  - `hibp_api_errors_total`
- Alert on high error rate (>5%) or slow API (>2s p95)

---

## Implementation Timeline

### Week 1: Random Login Delay

| Task | Duration | Owner | Dependencies |
|------|----------|-------|--------------|
| **Day 1-2**: Implement timing helpers | 0.5 day | backend-developer | None |
| **Day 2**: Integrate into LoginHandler | 0.5 day | backend-developer | Timing helpers |
| **Day 3**: Unit tests for timing logic | 0.5 day | backend-test-architect | Implementation |
| **Day 3-4**: Integration tests | 0.5 day | backend-test-architect | Unit tests |
| **Day 4**: E2E tests (Newman) | 0.5 day | test-strategist | Integration tests |
| **Day 5**: Security review | 0.5 day | senior-secops-engineer | All tests passing |
| **Day 5**: Performance validation | 0.5 day | senior-go-architect | Security review |

### Week 2: HIBP Password Check

| Task | Duration | Owner | Dependencies |
|------|----------|-------|--------------|
| **Day 1**: Add supabase/hibp dependency | 0.25 day | backend-developer | None |
| **Day 1-2**: Implement HIBPClient | 1 day | backend-developer | Dependency |
| **Day 2-3**: Implement Redis cache | 0.5 day | backend-developer | HIBPClient |
| **Day 3-4**: Integrate into RegisterHandler | 0.5 day | backend-developer | Cache |
| **Day 4-5**: Unit tests (HIBP + cache) | 1 day | backend-test-architect | Implementation |
| **Day 5-6**: Integration tests | 0.5 day | backend-test-architect | Unit tests |
| **Day 6**: E2E tests (Newman) | 0.5 day | test-strategist | Integration tests |
| **Day 7**: Security review | 0.5 day | senior-secops-engineer | All tests passing |
| **Day 7**: Performance validation | 0.5 day | senior-go-architect | Security review |

### Sprint Wrap-Up

| Task | Duration | Owner |
|------|----------|-------|
| Documentation updates | 0.5 day | senior-go-architect |
| Configuration guide | 0.5 day | senior-go-architect |
| Deployment runbook | 0.5 day | cicd-guardian |
| Sprint retrospective | 0.5 day | scrum-master |

---

## Configuration Reference

### Environment Variables

```bash
# Random Login Delay
AUTH_MIN_DELAY=100ms              # Minimum authentication delay
AUTH_MAX_DELAY=300ms              # Maximum authentication delay

# HIBP Password Check
HIBP_ENABLED=true                 # Enable/disable HIBP checks
HIBP_TIMEOUT=5s                   # API request timeout
HIBP_CACHE_TTL=24h                # Cache TTL for negative results
HIBP_FAIL_OPEN=true               # Allow registration if API fails
```

### Production Recommendations

```bash
# Production (strict security)
AUTH_MIN_DELAY=150ms
AUTH_MAX_DELAY=300ms
HIBP_ENABLED=true
HIBP_TIMEOUT=5s
HIBP_FAIL_OPEN=true              # Don't break registration

# Development (faster feedback)
AUTH_MIN_DELAY=50ms
AUTH_MAX_DELAY=100ms
HIBP_ENABLED=false               # Optional: disable for speed
HIBP_FAIL_OPEN=true

# Testing (deterministic)
AUTH_MIN_DELAY=100ms
AUTH_MAX_DELAY=100ms             # Fixed delay for tests
HIBP_ENABLED=true
HIBP_TIMEOUT=2s
```

---

## Security Gate S10 Requirements

| Control ID | Requirement | Evidence | Owner |
|------------|-------------|----------|-------|
| S10-AUTH-001 | Random delay uses crypto/rand | Code review + unit tests | senior-secops-engineer |
| S10-AUTH-002 | Delay applied to all auth paths | Integration tests | backend-test-architect |
| S10-AUTH-003 | No timing leak in logs | Manual verification | senior-secops-engineer |
| S10-HIBP-001 | k-anonymity enforced (5 chars only) | Code review | senior-secops-engineer |
| S10-HIBP-002 | SHA-1 only for HIBP, not storage | Code review | senior-secops-engineer |
| S10-HIBP-003 | API failures fail open | Unit tests | backend-test-architect |
| S10-HIBP-004 | No PII in HIBP logs | Manual verification | senior-secops-engineer |
| S10-HIBP-005 | Cache prevents timing attacks | Performance tests | senior-go-architect |
| S10-TEST-001 | 85%+ test coverage | Coverage report | backend-test-architect |
| S10-PERF-001 | <500ms p95 login latency | Load tests | test-strategist |

**Approval Required From**:
- senior-secops-engineer (security validation)
- backend-test-architect (test coverage)
- senior-go-architect (code quality)

---

## Monitoring & Alerting

### Key Metrics

```promql
# Authentication Delays
rate(auth_login_delay_seconds_sum[5m]) / rate(auth_login_delay_seconds_count[5m])

# HIBP Checks
rate(hibp_checks_total{result="pwned"}[1h])
rate(hibp_checks_total{result="error"}[5m])
hibp_api_latency_seconds{quantile="0.95"}

# Cache Performance
rate(hibp_cache_hits_total[5m]) / rate(hibp_checks_total[5m])
```

### Alert Rules

```yaml
# Grafana Alert: HIBP API Errors
- alert: HIBPHighErrorRate
  expr: rate(hibp_checks_total{result="error"}[5m]) > 0.05
  for: 10m
  annotations:
    summary: "HIBP API error rate > 5%"
    description: "Check HIBP service status and fail-open configuration"

# Grafana Alert: Slow Authentication
- alert: SlowAuthenticationDelays
  expr: histogram_quantile(0.95, rate(auth_login_processing_seconds_bucket[5m])) > 1
  for: 15m
  annotations:
    summary: "95th percentile auth processing > 1s"
    description: "Backend performance degradation, check database and Redis"
```

---

## Rollout Strategy

### Phase 1: Canary Deployment (10% traffic)

**Metrics to Watch**:
- Login success rate (should not decrease)
- P95 login latency (should increase by 100-300ms)
- HIBP API success rate (should be >95%)
- User complaints about slow login

**Rollback Criteria**:
- Login success rate drops >5%
- P95 latency >1s
- HIBP API errors >10%

### Phase 2: Full Rollout (100% traffic)

**Prerequisites**:
- Canary successful for 48 hours
- No critical issues reported
- Monitoring dashboards validated
- Alerts configured and tested

### Phase 3: Post-Deployment Validation

**Tasks**:
- [ ] Run penetration test on timing attack mitigation
- [ ] Verify HIBP cache hit rate >90%
- [ ] Review user feedback on password rejection
- [ ] Tune delay configuration if needed
- [ ] Document lessons learned

---

## Documentation Updates

### Files to Update

1. **`/docs/api/README.md`**
   - Add error documentation for `ErrPasswordCompromised`
   - Update authentication endpoint examples

2. **`/docs/deployment/environment_variables.md`**
   - Add configuration options for random delay
   - Add HIBP configuration section

3. **`/docs/security/authentication.md`** (new)
   - Document timing attack mitigation
   - Explain HIBP integration
   - Privacy guarantees (k-anonymity)

4. **`/SECURITY.md`**
   - Add section on compromised password rejection
   - Update security features list

5. **`/claude/api_security.md`**
   - Update authentication security controls
   - Add timing attack mitigation patterns

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| HIBP API downtime | Medium | Low | Fail-open configuration, local caching |
| Excessive delays annoy users | Low | Medium | Tunable config, user education |
| Cache poisoning | Low | High | Redis authentication, key expiration |
| Timing attack still possible | Low | Medium | Statistical analysis in testing |
| Strong passwords rejected | Medium | Low | Clear error message, alternative suggestions |

---

## Success Criteria

### Functional Requirements

- [ ] All authentication attempts have random delay (100-300ms)
- [ ] Compromised passwords rejected during registration
- [ ] Compromised passwords rejected during password change
- [ ] API failures do not block legitimate users
- [ ] Cache hit rate >90% after warm-up

### Non-Functional Requirements

- [ ] Test coverage ≥85% for new code
- [ ] P95 login latency <500ms
- [ ] HIBP API success rate >95%
- [ ] Zero timing information leaked in logs
- [ ] Documentation complete and accurate

### Security Validation

- [ ] Timing attack penetration test passed
- [ ] k-anonymity verified (network inspection)
- [ ] No PII logged in HIBP requests
- [ ] Cache timing is constant (no cache-timing attacks)
- [ ] Security gate S10 approved

---

## References

### Research Sources

- [Sling Academy - Constant-Time Comparisons in Go](https://www.slingacademy.com/article/avoiding-timing-attacks-with-constant-time-comparisons-in-go/)
- [Alex Edwards - Basic Authentication in Go](https://www.alexedwards.net/blog/basic-authentication-in-go)
- [StackHawk - Golang Broken Authentication Guide](https://www.stackhawk.com/blog/golang-broken-authentication-guide-examples-and-prevention/)
- [Go Security Best Practices](https://go.dev/doc/security/best-practices)
- [GlobaLeaks - Timing Attack Prevention](https://github.com/globaleaks/GlobaLeaks/issues/264)
- [HIBP API Documentation](https://haveibeenpwned.com/API/v3)
- [go-hibp Package Documentation](https://pkg.go.dev/github.com/wneessen/go-hibp)
- [Supabase HIBP Library](https://github.com/supabase/hibp)

### Internal Documentation

- `/home/user/goimg-datalayer/claude/architecture.md`
- `/home/user/goimg-datalayer/claude/api_security.md`
- `/home/user/goimg-datalayer/claude/test_strategy.md`
- `/home/user/goimg-datalayer/internal/domain/identity/password.go`
- `/home/user/goimg-datalayer/internal/application/identity/commands/login.go`
- `/home/user/goimg-datalayer/internal/application/identity/commands/register_user.go`

---

## Appendix A: Code Examples

### Example: Testing Timing Distribution

```go
func TestLoginTimingDistribution(t *testing.T) {
    handler := setupLoginHandler(t)

    successTimings := make([]time.Duration, 100)
    failureTimings := make([]time.Duration, 100)

    // Measure successful logins
    for i := 0; i < 100; i++ {
        start := time.Now()
        _, _ = handler.Handle(ctx, validLoginCommand())
        successTimings[i] = time.Since(start)
    }

    // Measure failed logins
    for i := 0; i < 100; i++ {
        start := time.Now()
        _, _ = handler.Handle(ctx, invalidLoginCommand())
        failureTimings[i] = time.Since(start)
    }

    // Statistical analysis
    successMean := mean(successTimings)
    failureMean := mean(failureTimings)

    // Difference should not be statistically significant
    // Use Mann-Whitney U test or similar
    pValue := mannWhitneyUTest(successTimings, failureTimings)
    assert.Greater(t, pValue, 0.05, "Timing difference is statistically significant")
}
```

### Example: HIBP Mock for Testing

```go
type MockPasswordChecker struct {
    mock.Mock
}

func (m *MockPasswordChecker) IsPasswordPwned(ctx context.Context, password string) (bool, error) {
    args := m.Called(ctx, password)
    return args.Bool(0), args.Error(1)
}

// Usage in tests
func TestRegisterWithMockHIBP(t *testing.T) {
    mockChecker := new(MockPasswordChecker)
    mockChecker.On("IsPasswordPwned", mock.Anything, "password123").
        Return(true, nil)
    mockChecker.On("IsPasswordPwned", mock.Anything, "StrongP@ssw0rd!").
        Return(false, nil)

    handler := NewRegisterUserHandler(repo, publisher, mockChecker, logger)
    // ... test cases
}
```

---

## Appendix B: Performance Benchmarks

### Benchmark: Random Delay Generation

```go
func BenchmarkCalculateRandomAuthDelay(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _ = CalculateRandomAuthDelay()
    }
}

// Expected: <1µs per operation
```

### Benchmark: HIBP Cache Lookup

```go
func BenchmarkHIBPCacheHit(b *testing.B) {
    cache := setupRedisCache()
    prefix, suffix := "2AC9A", "6746ACA543AF8DFF39894CBC7B2D299FCAE"
    cache.Set(context.Background(), prefix, suffix, false, 24*time.Hour)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = cache.Get(context.Background(), prefix, suffix)
    }
}

// Expected: <500µs per operation
```

---

**End of Sprint 10 Implementation Plan**
