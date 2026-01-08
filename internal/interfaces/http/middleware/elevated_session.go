// Package middleware provides HTTP security middleware for authentication, authorization, and request protection.
package middleware

import (
	"net/http"

	"github.com/rs/zerolog"
)

// RequireElevatedSession middleware checks if the session has 2FA verification (Sprint 11).
// This middleware should be used for sensitive operations that require additional security:
//   - Changing password
//   - Disabling 2FA
//   - Deleting account
//   - Accessing/modifying sensitive user data
//
// The middleware:
//  1. Checks if the JWT token has TwoFAVerified claim set to true
//  2. Returns 403 Forbidden if session is not elevated
//  3. Allows request to proceed if session is elevated
//
// Usage:
//
//	// Apply to specific sensitive routes
//	r.With(middleware.RequireElevatedSession(logger)).
//	    Post("/api/v1/auth/2fa/disable", handlers.TwoFA.Disable)
//
//	// Apply to a group of sensitive routes
//	r.Group(func(r chi.Router) {
//	    r.Use(middleware.JWTAuth(cfg))              // Authenticate first
//	    r.Use(middleware.RequireElevatedSession(logger)) // Then check elevation
//	    r.Put("/api/v1/users/me/password", handlers.User.ChangePassword)
//	    r.Delete("/api/v1/users/me", handlers.User.DeleteAccount)
//	})
//
// Security considerations:
//   - This middleware must be placed AFTER JWTAuth middleware
//   - Users without 2FA enabled will always have TwoFAVerified=false
//   - Users with 2FA must call POST /api/v1/auth/2fa/login-verify after login
//   - Elevated sessions inherit the same TTL as access tokens (15 minutes)
func RequireElevatedSession(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			requestID := GetRequestID(ctx)

			// Get user context from middleware context functions (must be set by JWTAuth middleware)
			userID, hasUserID := GetUserID(ctx)
			email, _ := GetUserEmail(ctx)
			if !hasUserID {
				logger.Error().
					Str("event", "elevated_session_no_context").
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("elevated session check called without user context")

				WriteError(w, r,
					http.StatusUnauthorized,
					"Unauthorized",
					"Authentication required for this operation",
				)
				return
			}

			// Check if session is elevated (TwoFAVerified flag)
			twoFAVerified, _ := Get2FAVerified(ctx)
			if !twoFAVerified {
				logger.Warn().
					Str("event", "session_not_elevated").
					Str("user_id", userID.String()).
					Str("email", email).
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("sensitive operation attempted without elevated session")

				// Return 403 with instructions on how to elevate
				WriteError(w, r,
					http.StatusForbidden,
					"Elevated Session Required",
					"This operation requires 2FA verification. Please verify your identity by calling POST /api/v1/auth/2fa/login-verify with your TOTP code.",
				)
				return
			}

			// Session is elevated - allow request
			logger.Debug().
				Str("event", "elevated_session_validated").
				Str("user_id", userID.String()).
				Str("path", r.URL.Path).
				Str("request_id", requestID).
				Msg("elevated session validated successfully")

			next.ServeHTTP(w, r)
		})
	}
}
