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
		GlobalLimit: 100,
		AuthLimit:   300,
		LoginLimit:  5,
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
					fmt.Sprintf("You have exceeded the login attempt limit of %d per %s. Please try again later.", cfg.LoginLimit, cfg.WindowSize),
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
					fmt.Sprintf("You have exceeded the upload limit of %d uploads per %s. Please try again later.", uploadLimit, uploadWindow),
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
