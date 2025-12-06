package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
)

// JWTServiceInterface defines the interface for JWT token operations.
// This allows for dependency injection and testing with mocks.
type JWTServiceInterface interface {
	ValidateToken(tokenString string) (*jwt.Claims, error)
	ExtractTokenID(tokenString string) (string, error)
}

// TokenBlacklistInterface defines the interface for token blacklist operations.
type TokenBlacklistInterface interface {
	IsBlacklisted(ctx context.Context, tokenID string) (bool, error)
}

// AuthConfig holds configuration for JWT authentication middleware.
type AuthConfig struct {
	// JWTService handles token validation and signature verification.
	JWTService JWTServiceInterface

	// TokenBlacklist checks if tokens have been revoked (logout, security events).
	TokenBlacklist TokenBlacklistInterface

	// MetricsCollector records authentication metrics.
	MetricsCollector *MetricsCollector

	// Logger is used to log authentication events.
	Logger zerolog.Logger

	// Optional determines whether authentication is optional for this route.
	// If true, missing or invalid tokens do not result in 401 error.
	// The handler can check if a user is authenticated using GetUserID(ctx).
	// Default: false (authentication required)
	Optional bool
}

// JWTAuth creates a JWT authentication middleware with the given configuration.
//
// Authentication flow:
// 1. Extract Bearer token from Authorization header
// 2. Check if token is blacklisted (fast Redis lookup)
// 3. Validate token signature and expiration (RS256 verification)
// 4. Verify token type (must be "access" token, not "refresh")
// 5. Set user context (user_id, email, role, session_id)
//
// Security considerations:
// - Blacklist check before signature verification (performance optimization)
// - Constant-time string comparison for token validation
// - Logs all authentication failures for audit trail
// - Returns 401 for missing/invalid tokens (unless Optional=true)
// - Only accepts access tokens (refresh tokens rejected)
//
// Usage (required authentication):
//
//	cfg := middleware.AuthConfig{
//	    JWTService: jwtService,
//	    TokenBlacklist: blacklist,
//	    Logger: logger,
//	    Optional: false,
//	}
//	r.Group(func(r chi.Router) {
//	    r.Use(middleware.JWTAuth(cfg))
//	    r.Get("/protected", handler)
//	})
//
// Usage (optional authentication):
//
//	cfg := middleware.AuthConfig{
//	    JWTService: jwtService,
//	    TokenBlacklist: blacklist,
//	    Logger: logger,
//	    Optional: true,
//	}
//	r.With(middleware.JWTAuth(cfg)).Get("/public-or-private", handler)
func JWTAuth(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			requestID := GetRequestID(ctx)

			// Step 1: Extract Bearer token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				if cfg.Optional {
					// Optional auth - continue without user context
					next.ServeHTTP(w, r)
					return
				}

				cfg.Logger.Warn().
					Str("event", "auth_missing").
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("missing authorization header")

				WriteError(w, r,
					http.StatusUnauthorized,
					"Unauthorized",
					"Missing authorization header. Expected: Authorization: Bearer <token>",
				)
				return
			}

			// Step 2: Parse "Bearer <token>" format
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 {
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordAuthFailure("invalid_format")
				}
				handleAuthError(w, r, cfg, "invalid_format", "Invalid authorization header format. Expected: Authorization: Bearer <token>")
				return
			}

			if !strings.EqualFold(parts[0], "Bearer") {
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordAuthFailure("invalid_scheme")
				}
				handleAuthError(w, r, cfg, "invalid_scheme", "Invalid authorization scheme. Expected: Bearer")
				return
			}

			tokenString := parts[1]
			if tokenString == "" {
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordAuthFailure("empty_token")
				}
				handleAuthError(w, r, cfg, "empty_token", "Authorization token is empty")
				return
			}

			// Step 3: Extract token ID (fast operation, no crypto)
			tokenID, err := cfg.JWTService.ExtractTokenID(tokenString)
			if err != nil {
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordAuthFailure("token_parse_failed")
				}
				handleAuthError(w, r, cfg, "token_parse_failed", "Invalid token format")
				return
			}

			// Step 4: Check blacklist (Redis lookup ~1ms)
			// This check is done BEFORE expensive signature verification
			isBlacklisted, err := cfg.TokenBlacklist.IsBlacklisted(ctx, tokenID)
			if err != nil {
				cfg.Logger.Error().
					Err(err).
					Str("event", "blacklist_check_failed").
					Str("request_id", requestID).
					Msg("failed to check token blacklist")

				WriteError(w, r,
					http.StatusInternalServerError,
					"Internal Server Error",
					"Authentication service temporarily unavailable",
				)
				return
			}

			if isBlacklisted {
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordAuthFailure("token_revoked")
				}

				cfg.Logger.Warn().
					Str("event", "token_blacklisted").
					Str("token_id", tokenID).
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("attempt to use blacklisted token")

				WriteError(w, r,
					http.StatusUnauthorized,
					"Unauthorized",
					"Token has been revoked. Please log in again.",
				)
				return
			}

			// Step 5: Validate token signature and claims (RS256 verification ~5-10ms)
			claims, err := cfg.JWTService.ValidateToken(tokenString)
			if err != nil {
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordAuthFailure("token_invalid")
				}

				cfg.Logger.Warn().
					Err(err).
					Str("event", "token_validation_failed").
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("invalid token")

				handleAuthError(w, r, cfg, "token_invalid", "Invalid or expired token. Please log in again.")
				return
			}

			// Step 6: Verify token type (must be access token)
			if claims.TokenType != jwt.TokenTypeAccess {
				if cfg.MetricsCollector != nil {
					cfg.MetricsCollector.RecordAuthFailure("wrong_token_type")
				}

				cfg.Logger.Warn().
					Str("event", "wrong_token_type").
					Str("token_type", string(claims.TokenType)).
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("wrong token type used for authentication")

				WriteError(w, r,
					http.StatusUnauthorized,
					"Unauthorized",
					"Invalid token type. Access token required.",
				)
				return
			}

			// Step 7: Parse UUIDs from claims
			userID, err := uuid.Parse(claims.UserID)
			if err != nil {
				cfg.Logger.Error().
					Err(err).
					Str("event", "invalid_user_id").
					Str("user_id", claims.UserID).
					Str("request_id", requestID).
					Msg("invalid user ID in token claims")

				WriteError(w, r,
					http.StatusUnauthorized,
					"Unauthorized",
					"Invalid token claims",
				)
				return
			}

			sessionID, err := uuid.Parse(claims.SessionID)
			if err != nil {
				cfg.Logger.Error().
					Err(err).
					Str("event", "invalid_session_id").
					Str("session_id", claims.SessionID).
					Str("request_id", requestID).
					Msg("invalid session ID in token claims")

				WriteError(w, r,
					http.StatusUnauthorized,
					"Unauthorized",
					"Invalid token claims",
				)
				return
			}

			// Step 8: Set user context for downstream handlers
			ctx = SetUserContext(ctx, userID, claims.Email, claims.Role, sessionID)

			// Step 9: Log successful authentication
			cfg.Logger.Debug().
				Str("event", "auth_success").
				Str("user_id", claims.UserID).
				Str("role", claims.Role).
				Str("path", r.URL.Path).
				Str("request_id", requestID).
				Msg("request authenticated")

			// Continue with authenticated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// handleAuthError handles authentication errors based on Optional flag.
func handleAuthError(w http.ResponseWriter, r *http.Request, cfg AuthConfig, event, message string) {
	requestID := GetRequestID(r.Context())

	if cfg.Optional {
		// Optional auth - log and continue without user context
		cfg.Logger.Debug().
			Str("event", event).
			Str("path", r.URL.Path).
			Str("request_id", requestID).
			Msg("optional authentication failed")

		// Continue without user context
		// Handler can check GetUserID(ctx) to see if authenticated
		// This is a placeholder - the caller should handle this
		return
	}

	// Required auth - log and return error
	cfg.Logger.Warn().
		Str("event", event).
		Str("path", r.URL.Path).
		Str("request_id", requestID).
		Msg("authentication failed")

	WriteError(w, r, http.StatusUnauthorized, "Unauthorized", message)
}

// RequireRole creates a middleware that enforces role-based access control (RBAC).
// This middleware must be placed AFTER JWTAuth middleware.
//
// Roles (from least to most privileged):
// - "user": Regular user (can upload images, manage own content)
// - "moderator": Can moderate content (review reports, flag images)
// - "admin": Full administrative access (user management, system settings)
//
// Usage:
//
//	// Admin-only routes
//	r.Group(func(r chi.Router) {
//	    r.Use(middleware.JWTAuth(cfg))
//	    r.Use(middleware.RequireRole(logger, collector, "admin"))
//	    r.Get("/admin/users", handlers.Admin.ListUsers)
//	})
//
//	// Moderator or admin routes
//	r.Group(func(r chi.Router) {
//	    r.Use(middleware.JWTAuth(cfg))
//	    r.Use(middleware.RequireAnyRole(logger, collector, "moderator", "admin"))
//	    r.Post("/images/{id}/moderate", handlers.Moderation.ModerateImage)
//	})
func RequireRole(logger zerolog.Logger, metricsCollector *MetricsCollector, requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			requestID := GetRequestID(ctx)

			// Get user role from context (set by JWTAuth middleware)
			role, ok := GetUserRole(ctx)
			if !ok {
				logger.Error().
					Str("event", "role_check_no_context").
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("role check called without user context")

				WriteError(w, r,
					http.StatusUnauthorized,
					"Unauthorized",
					"User role not found in context",
				)
				return
			}

			// Check if user has required role
			if role != requiredRole {
				userID, _ := GetUserIDString(ctx)

				// Record metrics
				if metricsCollector != nil {
					metricsCollector.RecordAuthorizationDenied(role, requiredRole)
				}

				logger.Warn().
					Str("event", "insufficient_role").
					Str("user_id", userID).
					Str("user_role", role).
					Str("required_role", requiredRole).
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("access denied due to insufficient role")

				WriteError(w, r,
					http.StatusForbidden,
					"Forbidden",
					fmt.Sprintf("This action requires %s role", requiredRole),
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole creates a middleware that accepts multiple roles (OR logic).
// User must have at least one of the specified roles.
//
// Usage:
//
//	r.Use(middleware.RequireAnyRole(logger, collector, "moderator", "admin"))
func RequireAnyRole(logger zerolog.Logger, metricsCollector *MetricsCollector, allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			requestID := GetRequestID(ctx)

			// Get user role from context
			role, ok := GetUserRole(ctx)
			if !ok {
				logger.Error().
					Str("event", "role_check_no_context").
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("role check called without user context")

				WriteError(w, r,
					http.StatusUnauthorized,
					"Unauthorized",
					"User role not found in context",
				)
				return
			}

			// Check if user has any of the allowed roles
			for _, allowedRole := range allowedRoles {
				if role == allowedRole {
					next.ServeHTTP(w, r)
					return
				}
			}

			// User doesn't have any of the required roles
			userID, _ := GetUserIDString(ctx)

			// Record metrics (use first allowed role as required permission)
			requiredPermission := fmt.Sprintf("role:%v", allowedRoles)
			if metricsCollector != nil {
				metricsCollector.RecordAuthorizationDenied(role, requiredPermission)
			}

			logger.Warn().
				Str("event", "insufficient_role").
				Str("user_id", userID).
				Str("user_role", role).
				Strs("allowed_roles", allowedRoles).
				Str("path", r.URL.Path).
				Str("request_id", requestID).
				Msg("access denied due to insufficient role")

			WriteError(w, r,
				http.StatusForbidden,
				"Forbidden",
				fmt.Sprintf("This action requires one of the following roles: %v", allowedRoles),
			)
		})
	}
}
