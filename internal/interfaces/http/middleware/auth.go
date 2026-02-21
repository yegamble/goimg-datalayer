package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
)

const (
	bearerTokenParts        = 2
	httpInternalServerError = 500
)

type JWTServiceInterface interface {
	ValidateToken(tokenString string) (*jwt.Claims, error)
	ExtractTokenID(tokenString string) (string, error)
}

type TokenBlacklistInterface interface {
	IsBlacklisted(ctx context.Context, tokenID string) (bool, error)
}

type AuthConfig struct {
	JWTService JWTServiceInterface

	TokenBlacklist TokenBlacklistInterface

	MetricsCollector *MetricsCollector

	Logger zerolog.Logger

	Optional bool
}

func JWTAuth(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			requestID := GetRequestID(ctx)

			tokenString, err := extractBearerToken(r, cfg)
			if err != nil {
				if cfg.Optional {
					next.ServeHTTP(w, r)
					return
				}
				logAndRespondAuthError(w, r, cfg, requestID, err)
				return
			}

			_, err = checkTokenBlacklist(ctx, tokenString, cfg)
			if err != nil {
				logAndRespondAuthError(w, r, cfg, requestID, err)
				return
			}

			claims, err := validateTokenAndType(tokenString, cfg, requestID)
			if err != nil {
				logAndRespondAuthError(w, r, cfg, requestID, err)
				return
			}

			userID, sessionID, err := parseClaimsUUIDs(claims, cfg.Logger, requestID)
			if err != nil {
				WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Invalid token claims")
				return
			}

			ctx = SetUserContext(ctx, userID, claims.Email, claims.Role, sessionID, claims.TwoFAVerified, claims.EmailVerified)
			cfg.Logger.Debug().
				Str("event", "auth_success").
				Str("user_id", claims.UserID).
				Str("role", claims.Role).
				Bool("twofa_verified", claims.TwoFAVerified).
				Str("path", r.URL.Path).
				Str("request_id", requestID).
				Msg("request authenticated")

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type authError struct {
	event   string
	message string
	status  int
}

func (e *authError) Error() string {
	return e.message
}

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

func parseClaimsUUIDs(claims *jwt.Claims, logger zerolog.Logger, requestID string) (uuid.UUID, uuid.UUID, error) {
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("event", "invalid_user_id").
			Str("user_id", claims.UserID).
			Str("request_id", requestID).
			Msg("invalid user ID in token claims")
		return uuid.UUID{}, uuid.UUID{}, fmt.Errorf("parse user ID: %w", err)
	}

	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("event", "invalid_session_id").
			Str("session_id", claims.SessionID).
			Str("request_id", requestID).
			Msg("invalid session ID in token claims")
		return uuid.UUID{}, uuid.UUID{}, fmt.Errorf("parse session ID: %w", err)
	}

	return userID, sessionID, nil
}

func logAndRespondAuthError(w http.ResponseWriter, r *http.Request, cfg AuthConfig, requestID string, err error) {
	var authErr *authError
	if !errors.As(err, &authErr) {
		cfg.Logger.Error().
			Err(err).
			Str("request_id", requestID).
			Msg("authentication error")
		WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication failed")
		return
	}

	if authErr.status >= httpInternalServerError {
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

func RequireRole(
	logger zerolog.Logger, metricsCollector *MetricsCollector, requiredRole string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			requestID := GetRequestID(ctx)

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

			if role != requiredRole {
				userID, _ := GetUserIDString(ctx)

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

func RequireAnyRole(
	logger zerolog.Logger, metricsCollector *MetricsCollector, allowedRoles ...string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			requestID := GetRequestID(ctx)

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

			for _, allowedRole := range allowedRoles {
				if role == allowedRole {
					next.ServeHTTP(w, r)
					return
				}
			}

			userID, _ := GetUserIDString(ctx)

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
