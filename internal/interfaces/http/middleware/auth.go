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

const (
	// Expected number of parts in Bearer token header.
	bearerTokenParts = 2
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

			// Step 1: Extract and parse Bearer token
			tokenString, err := extractBearerToken(r, cfg)
			if err != nil {
				if cfg.Optional {
					next.ServeHTTP(w, r)
					return
				}
				logAndRespondAuthError(w, r, cfg, requestID, err)
				return
			}

			// Step 2: Extract token ID and check blacklist
			_, err = checkTokenBlacklist(ctx, tokenString, cfg)
			if err != nil {
				logAndRespondAuthError(w, r, cfg, requestID, err)
				return
			}

			// Step 3: Validate token and verify type
			claims, err := validateTokenAndType(tokenString, cfg, requestID)
			if err != nil {
				logAndRespondAuthError(w, r, cfg, requestID, err)
				return
			}

			// Step 4: Parse UUIDs from claims
			userID, sessionID, err := parseClaimsUUIDs(claims, cfg.Logger, requestID)
			if err != nil {
				WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Invalid token claims")
				return
			}

			// Step 5: Set user context and continue
			ctx = SetUserContext(ctx, userID, claims.Email, claims.Role, sessionID)
			cfg.Logger.Debug().
				Str("event", "auth_success").
				Str("user_id", claims.UserID).
				Str("role", claims.Role).
				Str("path", r.URL.Path).
				Str("request_id", requestID).
				Msg("request authenticated")

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// authError represents an authentication error with event type and message.
type authError struct {
	event   string
	message string
	status  int
}

func (e *authError) Error() string {
	return e.message
}

// extractBearerToken extracts and validates the Bearer token from the Authorization header.
func extractBearerToken(r *http.Request, cfg AuthConfig) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", &authError{
			event:   "auth_missing",
			message: "Missing authorization header. Expected: Authorization: Bearer <token>",
			status:  http.StatusUnauthorized,
		}
	}

	parts := strings.SplitN(authHeader, " ", bearerTokenParts)
	if len(parts) != bearerTokenParts {
		if cfg.MetricsCollector != nil {
			cfg.MetricsCollector.RecordAuthFailure("invalid_format")
		}
		return "", &authError{
			event:   "invalid_format",
			message: "Invalid authorization header format. Expected: Authorization: Bearer <token>",
			status:  http.StatusUnauthorized,
		}
	}

	if !strings.EqualFold(parts[0], "Bearer") {
		if cfg.MetricsCollector != nil {
			cfg.MetricsCollector.RecordAuthFailure("invalid_scheme")
		}
		return "", &authError{
			event:   "invalid_scheme",
			message: "Invalid authorization scheme. Expected: Bearer",
			status:  http.StatusUnauthorized,
		}
	}

	tokenString := parts[1]
	if tokenString == "" {
		if cfg.MetricsCollector != nil {
			cfg.MetricsCollector.RecordAuthFailure("empty_token")
		}
		return "", &authError{
			event:   "empty_token",
			message: "Authorization token is empty",
			status:  http.StatusUnauthorized,
		}
	}

	return tokenString, nil
}

// checkTokenBlacklist extracts token ID and verifies it's not blacklisted.
func checkTokenBlacklist(ctx context.Context, tokenString string, cfg AuthConfig) (string, error) {
	tokenID, err := cfg.JWTService.ExtractTokenID(tokenString)
	if err != nil {
		if cfg.MetricsCollector != nil {
			cfg.MetricsCollector.RecordAuthFailure("token_parse_failed")
		}
		return "", &authError{
			event:   "token_parse_failed",
			message: "Invalid token format",
			status:  http.StatusUnauthorized,
		}
	}

	isBlacklisted, err := cfg.TokenBlacklist.IsBlacklisted(ctx, tokenID)
	if err != nil {
		return "", &authError{
			event:   "blacklist_check_failed",
			message: "Authentication service temporarily unavailable",
			status:  http.StatusInternalServerError,
		}
	}

	if isBlacklisted {
		if cfg.MetricsCollector != nil {
			cfg.MetricsCollector.RecordAuthFailure("token_revoked")
		}
		return "", &authError{
			event:   "token_blacklisted",
			message: "Token has been revoked. Please log in again.",
			status:  http.StatusUnauthorized,
		}
	}

	return tokenID, nil
}

// validateTokenAndType validates the JWT token and verifies it's an access token.
func validateTokenAndType(tokenString string, cfg AuthConfig, _ string) (*jwt.Claims, error) {
	claims, err := cfg.JWTService.ValidateToken(tokenString)
	if err != nil {
		if cfg.MetricsCollector != nil {
			cfg.MetricsCollector.RecordAuthFailure("token_invalid")
		}
		return nil, &authError{
			event:   "token_validation_failed",
			message: "Invalid or expired token. Please log in again.",
			status:  http.StatusUnauthorized,
		}
	}

	if claims.TokenType != jwt.TokenTypeAccess {
		if cfg.MetricsCollector != nil {
			cfg.MetricsCollector.RecordAuthFailure("wrong_token_type")
		}
		return nil, &authError{
			event:   "wrong_token_type",
			message: "Invalid token type. Access token required.",
			status:  http.StatusUnauthorized,
		}
	}

	return claims, nil
}

// parseClaimsUUIDs parses and validates UUIDs from JWT claims.
func parseClaimsUUIDs(claims *jwt.Claims, logger zerolog.Logger, requestID string) (uuid.UUID, uuid.UUID, error) {
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("event", "invalid_user_id").
			Str("user_id", claims.UserID).
			Str("request_id", requestID).
			Msg("invalid user ID in token claims")
		return uuid.UUID{}, uuid.UUID{}, err
	}

	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("event", "invalid_session_id").
			Str("session_id", claims.SessionID).
			Str("request_id", requestID).
			Msg("invalid session ID in token claims")
		return uuid.UUID{}, uuid.UUID{}, err
	}

	return userID, sessionID, nil
}

// logAndRespondAuthError logs and responds with appropriate auth error.
func logAndRespondAuthError(w http.ResponseWriter, r *http.Request, cfg AuthConfig, requestID string, err error) {
	authErr, ok := err.(*authError)
	if !ok {
		// Generic error handling
		cfg.Logger.Error().
			Err(err).
			Str("request_id", requestID).
			Msg("authentication error")
		WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication failed")
		return
	}

	// Log based on severity
	if authErr.status >= 500 {
		cfg.Logger.Error().
			Str("event", authErr.event).
			Str("path", r.URL.Path).
			Str("request_id", requestID).
			Msg(authErr.message)
	} else {
		cfg.Logger.Warn().
			Str("event", authErr.event).
			Str("path", r.URL.Path).
			Str("request_id", requestID).
			Msg(authErr.message)
	}

	WriteError(w, r, authErr.status, getErrorTitle(authErr.status), authErr.message)
}

// getErrorTitle returns appropriate error title for HTTP status code.
func getErrorTitle(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "Unauthorized"
	case http.StatusForbidden:
		return "Forbidden"
	case http.StatusInternalServerError:
		return "Internal Server Error"
	default:
		return "Error"
	}
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
