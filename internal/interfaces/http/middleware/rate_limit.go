package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

const (
	// defaultGlobalLimit is the default rate limit for unauthenticated requests (per IP).
	defaultGlobalLimit = 100
	// defaultAuthLimit is the default rate limit for authenticated requests (per user).
	defaultAuthLimit = 300
	// defaultLoginLimit is the default rate limit for login attempts (per IP).
	defaultLoginLimit = 5
)

// RateLimiterConfig holds configuration for rate limiting middleware.
type RateLimiterConfig struct {
	// RedisClient is the Redis client for storing rate limit counters.
	RedisClient *redis.Client

	// MetricsCollector records rate limit metrics.
	MetricsCollector *MetricsCollector

	// GlobalLimit is the maximum requests per window for unauthenticated requests (per IP).
	// Default: 100 requests per minute
	GlobalLimit int

	// AuthLimit is the maximum requests per window for authenticated requests (per user).
	// Default: 300 requests per minute (higher limit for authenticated users)
	AuthLimit int

	// LoginLimit is the maximum login attempts per window (per IP).
	// Default: 5 requests per minute (prevents brute-force attacks)
	LoginLimit int

	// WindowSize is the time window for rate limiting.
	// Default: 1 minute
	WindowSize time.Duration

	// Logger is used to log rate limit events.
	Logger zerolog.Logger

	// TrustProxy determines whether to trust X-Forwarded-For header for IP extraction.
	// Only enable if behind a trusted reverse proxy (nginx, ALB, etc.)
	// Default: false (safer for security)
	TrustProxy bool
}

// DefaultRateLimiterConfig returns a configuration with secure defaults.
func DefaultRateLimiterConfig(redisClient *redis.Client, logger zerolog.Logger) RateLimiterConfig {
	return RateLimiterConfig{
		RedisClient: redisClient,
		GlobalLimit: defaultGlobalLimit,
		AuthLimit:   defaultAuthLimit,
		LoginLimit:  defaultLoginLimit,
		WindowSize:  time.Minute,
		Logger:      logger,
		TrustProxy:  false,
	}
}

// RateLimitInfo holds rate limit information for response headers.
type RateLimitInfo struct {
	Limit      int   // Maximum requests allowed
	Remaining  int   // Remaining requests in current window
	Reset      int64 // Unix timestamp when the window resets
	RetryAfter int   // Seconds until retry (only set when limit exceeded)
}

// RateLimiter creates a rate limiting middleware with the given configuration.
// This is the general-purpose rate limiter for unauthenticated requests (per IP).
//
// Redis key pattern: goimg:ratelimit:global:{ip}
//
// Response headers set:
//   - X-RateLimit-Limit: Maximum requests allowed
//   - X-RateLimit-Remaining: Requests remaining in current window
//   - X-RateLimit-Reset: Unix timestamp when window resets
//   - Retry-After: Seconds until retry (only when 429 returned)
//
// Usage:
//
//	cfg := middleware.DefaultRateLimiterConfig(redisClient, logger)
//	r.Use(middleware.RateLimiter(cfg))
func RateLimiter(cfg RateLimiterConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract client IP
			clientIP := extractClientIP(r, cfg.TrustProxy)

			// Build rate limit key
			key := fmt.Sprintf("goimg:ratelimit:global:%s", clientIP)

			// Check rate limit
			allowed, info, err := checkRateLimit(ctx, cfg.RedisClient, key, cfg.GlobalLimit, cfg.WindowSize)
			if err != nil {
				// Log error but allow request to proceed (fail open for availability)
				cfg.Logger.Error().
					Err(err).
					Str("ip", clientIP).
					Str("request_id", GetRequestID(ctx)).
					Msg("rate limit check failed")

				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			setRateLimitHeaders(w, info)

			// Deny request if rate limit exceeded
			if !allowed {
				// Record metrics
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordRateLimitExceeded("global")
				}

				cfg.Logger.Warn().
					Str("ip", clientIP).
					Str("path", r.URL.Path).
					Int("limit", cfg.GlobalLimit).
					Str("request_id", GetRequestID(ctx)).
					Msg("rate limit exceeded")

				// Set Retry-After header
				w.Header().Set("Retry-After", strconv.Itoa(info.RetryAfter))

				// Return 429 Too Many Requests
				WriteErrorWithExtensions(w, r,
					http.StatusTooManyRequests,
					"Rate Limit Exceeded",
					fmt.Sprintf("You have exceeded the rate limit of %d requests per %s", cfg.GlobalLimit, cfg.WindowSize),
					map[string]interface{}{
						"limit":      info.Limit,
						"remaining":  info.Remaining,
						"reset":      info.Reset,
						"retryAfter": info.RetryAfter,
					},
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthRateLimiter creates a rate limiting middleware for authenticated requests (per user).
// This should be placed AFTER the JWT authentication middleware.
//
// Redis key pattern: goimg:ratelimit:auth:{user_id}
//
// Higher rate limit than global limiter (300 vs 100 req/min) as authenticated
// users are trusted and have legitimate higher usage patterns.
//
// Usage:
//
//	r.Group(func(r chi.Router) {
//	    r.Use(middleware.JWTAuth(cfg))
//	    r.Use(middleware.AuthRateLimiter(cfg))
//	    // Protected routes here
//	})
//
//nolint:funlen // Rate limiting middleware with Redis.
func AuthRateLimiter(cfg RateLimiterConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract user ID from context (set by JWT middleware)
			userID, ok := GetUserIDString(ctx)
			if !ok {
				// No user ID in context - should not happen if JWTAuth middleware is present
				cfg.Logger.Error().
					Str("request_id", GetRequestID(ctx)).
					Msg("auth rate limiter called without user context")

				WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Rate limiter configuration error")
				return
			}

			// Build rate limit key
			key := fmt.Sprintf("goimg:ratelimit:auth:%s", userID)

			// Check rate limit
			allowed, info, err := checkRateLimit(ctx, cfg.RedisClient, key, cfg.AuthLimit, cfg.WindowSize)
			if err != nil {
				cfg.Logger.Error().
					Err(err).
					Str("user_id", userID).
					Str("request_id", GetRequestID(ctx)).
					Msg("auth rate limit check failed")

				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			setRateLimitHeaders(w, info)

			// Deny request if rate limit exceeded
			if !allowed {
				// Record metrics
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordRateLimitExceeded("auth")
				}

				cfg.Logger.Warn().
					Str("user_id", userID).
					Str("path", r.URL.Path).
					Int("limit", cfg.AuthLimit).
					Str("request_id", GetRequestID(ctx)).
					Msg("auth rate limit exceeded")

				w.Header().Set("Retry-After", strconv.Itoa(info.RetryAfter))

				WriteErrorWithExtensions(w, r,
					http.StatusTooManyRequests,
					"Rate Limit Exceeded",
					fmt.Sprintf("You have exceeded the rate limit of %d requests per %s", cfg.AuthLimit, cfg.WindowSize),
					map[string]interface{}{
						"limit":      info.Limit,
						"remaining":  info.Remaining,
						"reset":      info.Reset,
						"retryAfter": info.RetryAfter,
					},
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoginRateLimiter creates a rate limiting middleware specifically for login endpoints.
// Uses a much stricter limit (5 req/min) to prevent brute-force attacks.
//
// Redis key pattern: goimg:ratelimit:login:{ip}
//
// This should be applied specifically to login endpoints using chi's With() method:
//
// Usage:
//
//	r.With(middleware.LoginRateLimiter(cfg)).Post("/auth/login", handlers.Auth.Login)
func LoginRateLimiter(cfg RateLimiterConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract client IP
			clientIP := extractClientIP(r, cfg.TrustProxy)

			// Build rate limit key
			key := fmt.Sprintf("goimg:ratelimit:login:%s", clientIP)

			// Check rate limit
			allowed, info, err := checkRateLimit(ctx, cfg.RedisClient, key, cfg.LoginLimit, cfg.WindowSize)
			if err != nil {
				cfg.Logger.Error().
					Err(err).
					Str("ip", clientIP).
					Str("request_id", GetRequestID(ctx)).
					Msg("login rate limit check failed")

				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			setRateLimitHeaders(w, info)

			// Deny request if rate limit exceeded
			if !allowed {
				// Record metrics
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordRateLimitExceeded("login")
				}

				cfg.Logger.Warn().
					Str("ip", clientIP).
					Int("limit", cfg.LoginLimit).
					Str("request_id", GetRequestID(ctx)).
					Msg("login rate limit exceeded - potential brute force attack")

				w.Header().Set("Retry-After", strconv.Itoa(info.RetryAfter))

				WriteErrorWithExtensions(w, r,
					http.StatusTooManyRequests,
					"Too Many Login Attempts",
					fmt.Sprintf(
						"You have exceeded the login attempt limit of %d per %s. Please try again later.",
						cfg.LoginLimit, cfg.WindowSize,
					),
					map[string]interface{}{
						"limit":      info.Limit,
						"remaining":  info.Remaining,
						"reset":      info.Reset,
						"retryAfter": info.RetryAfter,
					},
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// checkRateLimit performs the rate limit check using Redis fixed window algorithm.
// Returns: (allowed bool, info *RateLimitInfo, error).
func checkRateLimit(
	ctx context.Context,
	redisClient *redis.Client,
	key string,
	limit int,
	window time.Duration,
) (bool, *RateLimitInfo, error) {
	// Use Redis pipeline for atomic operations
	pipe := redisClient.Pipeline()

	// Increment counter
	incrCmd := pipe.Incr(ctx, key)

	// Set expiration on first increment
	pipe.Expire(ctx, key, window)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("redis pipeline failed: %w", err)
	}

	// Get current count
	count := int(incrCmd.Val())

	// Check if limit exceeded
	allowed := count <= limit

	// Calculate remaining requests
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	// Get TTL for reset time
	ttl, err := redisClient.TTL(ctx, key).Result()
	if err != nil {
		return false, nil, fmt.Errorf("redis ttl failed: %w", err)
	}

	// Calculate reset timestamp
	resetAt := time.Now().Add(ttl)

	info := &RateLimitInfo{
		Limit:      limit,
		Remaining:  remaining,
		Reset:      resetAt.Unix(),
		RetryAfter: int(ttl.Seconds()),
	}

	return allowed, info, nil
}

// setRateLimitHeaders sets standard rate limit response headers.
func setRateLimitHeaders(w http.ResponseWriter, info *RateLimitInfo) {
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(info.Reset, 10))
}

// UploadRateLimiter creates a rate limiting middleware specifically for upload endpoints.
// Uses a stricter limit (50 uploads/hour per user) to prevent storage abuse.
//
// Redis key pattern: goimg:ratelimit:upload:{user_id}
//
// This should be applied specifically to upload endpoints using chi's With() method:
//
// Usage:
//
//	r.With(middleware.UploadRateLimiter(cfg)).Post("/api/v1/images", handlers.Image.Upload)
//
//nolint:funlen // Rate limiting middleware with Redis.
func UploadRateLimiter(cfg RateLimiterConfig) func(http.Handler) http.Handler {
	// Default upload limit: 50 uploads per hour
	uploadLimit := 50
	uploadWindow := time.Hour

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract user ID from context (set by JWT middleware)
			userID, ok := GetUserIDString(ctx)
			if !ok {
				// No user ID in context - should not happen if JWTAuth middleware is present
				cfg.Logger.Error().
					Str("request_id", GetRequestID(ctx)).
					Msg("upload rate limiter called without user context")

				WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Rate limiter configuration error")
				return
			}

			// Build rate limit key
			key := fmt.Sprintf("goimg:ratelimit:upload:%s", userID)

			// Check rate limit
			allowed, info, err := checkRateLimit(ctx, cfg.RedisClient, key, uploadLimit, uploadWindow)
			if err != nil {
				cfg.Logger.Error().
					Err(err).
					Str("user_id", userID).
					Str("request_id", GetRequestID(ctx)).
					Msg("upload rate limit check failed")

				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			setRateLimitHeaders(w, info)

			// Deny request if rate limit exceeded
			if !allowed {
				// Record metrics
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordRateLimitExceeded("upload")
				}

				cfg.Logger.Warn().
					Str("user_id", userID).
					Int("limit", uploadLimit).
					Str("request_id", GetRequestID(ctx)).
					Msg("upload rate limit exceeded - potential storage abuse")

				w.Header().Set("Retry-After", strconv.Itoa(info.RetryAfter))

				WriteErrorWithExtensions(w, r,
					http.StatusTooManyRequests,
					"Upload Limit Exceeded",
					fmt.Sprintf(
						"You have exceeded the upload limit of %d uploads per %s. Please try again later.",
						uploadLimit, uploadWindow,
					),
					map[string]interface{}{
						"limit":      info.Limit,
						"remaining":  info.Remaining,
						"reset":      info.Reset,
						"retryAfter": info.RetryAfter,
					},
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TwoFARateLimiter creates a rate limiting middleware specifically for 2FA verification endpoints.
// Uses a strict limit (5 attempts/min per user) to prevent brute-force attacks on TOTP codes.
//
// Redis key pattern: goimg:ratelimit:2fa:{user_id}
//
// Security: TOTP codes have only 6 digits (1,000,000 possibilities). Without rate limiting,
// an attacker could try all combinations in minutes. With 5 attempts/min, a brute-force
// attack would take ~139 days on average.
//
// This should be applied to 2FA verification endpoints using chi's With() method:
//
// Usage:
//
//	r.With(middleware.TwoFARateLimiter(cfg)).Post("/api/v1/auth/2fa/verify", handlers.TwoFA.Verify)
//
//nolint:funlen // Rate limiting middleware with Redis.
func TwoFARateLimiter(cfg RateLimiterConfig) func(http.Handler) http.Handler {
	// 2FA rate limit: 5 attempts per minute (same as login)
	twoFALimit := 5
	twoFAWindow := time.Minute

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract user ID from context (set by JWT middleware)
			userID, ok := GetUserIDString(ctx)
			if !ok {
				// No user ID in context - should not happen if JWTAuth middleware is present
				cfg.Logger.Error().
					Str("request_id", GetRequestID(ctx)).
					Msg("2FA rate limiter called without user context")

				WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Rate limiter configuration error")
				return
			}

			// Build rate limit key
			key := fmt.Sprintf("goimg:ratelimit:2fa:%s", userID)

			// Check rate limit
			allowed, info, err := checkRateLimit(ctx, cfg.RedisClient, key, twoFALimit, twoFAWindow)
			if err != nil {
				cfg.Logger.Error().
					Err(err).
					Str("user_id", userID).
					Str("request_id", GetRequestID(ctx)).
					Msg("2FA rate limit check failed")

				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			setRateLimitHeaders(w, info)

			// Deny request if rate limit exceeded
			if !allowed {
				// Record metrics
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordRateLimitExceeded("2fa")
				}

				cfg.Logger.Warn().
					Str("user_id", userID).
					Int("limit", twoFALimit).
					Str("request_id", GetRequestID(ctx)).
					Msg("2FA rate limit exceeded - potential brute force attack")

				w.Header().Set("Retry-After", strconv.Itoa(info.RetryAfter))

				WriteErrorWithExtensions(w, r,
					http.StatusTooManyRequests,
					"Too Many 2FA Attempts",
					fmt.Sprintf(
						"You have exceeded the 2FA verification limit of %d attempts per %s. Please try again later.",
						twoFALimit, twoFAWindow,
					),
					map[string]interface{}{
						"limit":      info.Limit,
						"remaining":  info.Remaining,
						"reset":      info.Reset,
						"retryAfter": info.RetryAfter,
					},
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ReportRateLimiter creates a rate limiting middleware for abuse report creation.
// Uses a limit of 10 reports/hour per user to prevent report spam and abuse.
//
// Redis key pattern: goimg:ratelimit:report:{user_id}
//
// This should be applied to the report creation endpoint:
//
// Usage:
//
//	r.With(middleware.ReportRateLimiter(cfg)).Post("/reports", handlers.Moderation.CreateReport)
//
//nolint:funlen // Rate limiting middleware with Redis.
func ReportRateLimiter(cfg RateLimiterConfig) func(http.Handler) http.Handler {
	// Report rate limit: 10 reports per hour
	reportLimit := 10
	reportWindow := time.Hour

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract user ID from context (set by JWT middleware)
			userID, ok := GetUserIDString(ctx)
			if !ok {
				// No user ID in context - should not happen if JWTAuth middleware is present
				cfg.Logger.Error().
					Str("request_id", GetRequestID(ctx)).
					Msg("report rate limiter called without user context")

				WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Rate limiter configuration error")
				return
			}

			// Build rate limit key
			key := fmt.Sprintf("goimg:ratelimit:report:%s", userID)

			// Check rate limit
			allowed, info, err := checkRateLimit(ctx, cfg.RedisClient, key, reportLimit, reportWindow)
			if err != nil {
				cfg.Logger.Error().
					Err(err).
					Str("user_id", userID).
					Str("request_id", GetRequestID(ctx)).
					Msg("report rate limit check failed")

				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			setRateLimitHeaders(w, info)

			// Deny request if rate limit exceeded
			if !allowed {
				// Record metrics
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordRateLimitExceeded("report")
				}

				cfg.Logger.Warn().
					Str("user_id", userID).
					Int("limit", reportLimit).
					Str("request_id", GetRequestID(ctx)).
					Msg("report rate limit exceeded - potential report spam")

				w.Header().Set("Retry-After", strconv.Itoa(info.RetryAfter))

				WriteErrorWithExtensions(w, r,
					http.StatusTooManyRequests,
					"Report Limit Exceeded",
					fmt.Sprintf(
						"You have exceeded the report limit of %d reports per %s. Please try again later.",
						reportLimit, reportWindow,
					),
					map[string]interface{}{
						"limit":      info.Limit,
						"remaining":  info.Remaining,
						"reset":      info.Reset,
						"retryAfter": info.RetryAfter,
					},
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GuestSessionRateLimiter creates a rate limiting middleware for guest session creation.
// Uses IP-based limiting of 10 sessions/hour to prevent abuse of anonymous upload feature.
//
// Redis key pattern: goimg:ratelimit:guest:{ip}
//
// Security: Guest sessions allow anonymous uploads. Without rate limiting, attackers
// could create unlimited guest accounts for storage abuse or spam.
//
// This should be applied to the guest session creation endpoint:
//
// Usage:
//
//	r.With(middleware.GuestSessionRateLimiter(cfg)).Post("/auth/guest", handlers.Auth.CreateGuestSession)
//
//nolint:funlen // Rate limiting middleware with Redis.
func GuestSessionRateLimiter(cfg RateLimiterConfig) func(http.Handler) http.Handler {
	// Guest session rate limit: 10 sessions per hour per IP
	guestLimit := 10
	guestWindow := time.Hour

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract client IP
			clientIP := extractClientIP(r, cfg.TrustProxy)

			// Build rate limit key
			key := fmt.Sprintf("goimg:ratelimit:guest:%s", clientIP)

			// Check rate limit
			allowed, info, err := checkRateLimit(ctx, cfg.RedisClient, key, guestLimit, guestWindow)
			if err != nil {
				cfg.Logger.Error().
					Err(err).
					Str("ip", clientIP).
					Str("request_id", GetRequestID(ctx)).
					Msg("guest session rate limit check failed")

				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			setRateLimitHeaders(w, info)

			// Deny request if rate limit exceeded
			if !allowed {
				// Record metrics
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordRateLimitExceeded("guest")
				}

				cfg.Logger.Warn().
					Str("ip", clientIP).
					Int("limit", guestLimit).
					Str("request_id", GetRequestID(ctx)).
					Msg("guest session rate limit exceeded - potential abuse")

				w.Header().Set("Retry-After", strconv.Itoa(info.RetryAfter))

				WriteErrorWithExtensions(w, r,
					http.StatusTooManyRequests,
					"Guest Session Limit Exceeded",
					fmt.Sprintf(
						"You have exceeded the guest session limit of %d per %s. Please try again later.",
						guestLimit, guestWindow,
					),
					map[string]interface{}{
						"limit":      info.Limit,
						"remaining":  info.Remaining,
						"reset":      info.Reset,
						"retryAfter": info.RetryAfter,
					},
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractClientIP extracts the client IP address from the request.
// If trustProxy is true, it checks X-Forwarded-For and X-Real-IP headers.
// Otherwise, it uses RemoteAddr directly.
func extractClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		return getClientIP(r) // Uses X-Forwarded-For logic
	}

	// Don't trust proxy headers - use RemoteAddr directly
	remoteAddr := r.RemoteAddr

	// Strip port if present
	for i := len(remoteAddr) - 1; i >= 0; i-- {
		if remoteAddr[i] == ':' {
			// IPv6 addresses are wrapped in brackets [::1]:8080
			if i > 0 && remoteAddr[0] == '[' {
				return remoteAddr[1 : i-1]
			}
			return remoteAddr[:i]
		}
	}

	return remoteAddr
}

// VariantGenerationRateLimiter creates a rate limiting middleware for custom variant generation.
// Uses a limit of 20 variants/hour per user to prevent DoS attacks on CPU-intensive processing.
//
// Redis key pattern: goimg:ratelimit:variant:{user_id}
//
// Security: Custom variant generation is CPU-intensive. Without rate limiting, attackers
// could overwhelm the server by requesting many large custom variants simultaneously.
//
// This should be applied to the variant generation endpoint:
//
// Usage:
//
//	r.With(middleware.VariantGenerationRateLimiter(cfg)).Post("/{imageID}/variants", handlers.Image.GenerateCustomVariant)
//
//nolint:funlen // Rate limiting middleware with Redis.
func VariantGenerationRateLimiter(cfg RateLimiterConfig) func(http.Handler) http.Handler {
	// Variant generation rate limit: 20 variants per hour
	variantLimit := 20
	variantWindow := time.Hour

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract user ID from context (set by JWT middleware)
			userID, ok := GetUserIDString(ctx)
			if !ok {
				// No user context - this shouldn't happen as endpoint requires auth
				// Fall through to handler which will return 401
				next.ServeHTTP(w, r)
				return
			}

			// Redis key for per-user variant generation rate limiting
			key := fmt.Sprintf("goimg:ratelimit:variant:%s", userID)

			// Check rate limit using shared helper
			allowed, info, err := checkRateLimit(ctx, cfg.RedisClient, key, variantLimit, variantWindow)
			if err != nil {
				// Redis error - log and allow request (fail-open for availability)
				cfg.Logger.Warn().
					Err(err).
					Str("user_id", userID).
					Str("request_id", GetRequestID(ctx)).
					Msg("variant generation rate limiter: redis error, allowing request")
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			setRateLimitHeaders(w, info)

			// Deny request if rate limit exceeded
			if !allowed {
				// Record metrics
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordRateLimitExceeded("variant_generation")
				}

				cfg.Logger.Warn().
					Str("user_id", userID).
					Str("path", r.URL.Path).
					Int("limit", variantLimit).
					Str("request_id", GetRequestID(ctx)).
					Msg("variant generation rate limit exceeded")

				w.Header().Set("Retry-After", strconv.Itoa(info.RetryAfter))

				WriteErrorWithExtensions(w, r,
					http.StatusTooManyRequests,
					"Rate Limit Exceeded",
					fmt.Sprintf("Variant generation rate limit exceeded. Limit: %d per hour", variantLimit),
					map[string]interface{}{
						"limit":      info.Limit,
						"remaining":  info.Remaining,
						"reset":      info.Reset,
						"retryAfter": info.RetryAfter,
					},
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
