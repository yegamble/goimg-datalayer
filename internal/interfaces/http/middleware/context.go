package middleware

import (
	"context"

	"github.com/google/uuid"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey contextKey = "requestID"

	// UserIDKey is the context key for authenticated user ID.
	UserIDKey contextKey = "userID"

	// UserEmailKey is the context key for authenticated user email.
	UserEmailKey contextKey = "userEmail"

	// UserRoleKey is the context key for authenticated user role.
	UserRoleKey contextKey = "userRole"

	// SessionIDKey is the context key for session ID.
	SessionIDKey contextKey = "sessionID"
)

// GetRequestID retrieves the request ID from the context.
// Returns empty string if not found.
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// SetRequestID adds a request ID to the context.
func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetUserID retrieves the user ID from the context.
// Returns zero UUID and false if not found or invalid.
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	if userID, ok := ctx.Value(UserIDKey).(uuid.UUID); ok {
		return userID, true
	}
	return uuid.Nil, false
}

// GetUserIDString retrieves the user ID as a string from the context.
// Returns empty string and false if not found.
func GetUserIDString(ctx context.Context) (string, bool) {
	if userID, ok := GetUserID(ctx); ok {
		return userID.String(), true
	}
	return "", false
}

// GetUserEmail retrieves the user email from the context.
// Returns empty string and false if not found.
func GetUserEmail(ctx context.Context) (string, bool) {
	if email, ok := ctx.Value(UserEmailKey).(string); ok {
		return email, true
	}
	return "", false
}

// GetUserRole retrieves the user role from the context.
// Returns empty string and false if not found.
func GetUserRole(ctx context.Context) (string, bool) {
	if role, ok := ctx.Value(UserRoleKey).(string); ok {
		return role, true
	}
	return "", false
}

// GetSessionID retrieves the session ID from the context.
// Returns zero UUID and false if not found or invalid.
func GetSessionID(ctx context.Context) (uuid.UUID, bool) {
	if sessionID, ok := ctx.Value(SessionIDKey).(uuid.UUID); ok {
		return sessionID, true
	}
	return uuid.Nil, false
}

// GetSessionIDString retrieves the session ID as a string from the context.
// Returns empty string and false if not found.
func GetSessionIDString(ctx context.Context) (string, bool) {
	if sessionID, ok := GetSessionID(ctx); ok {
		return sessionID.String(), true
	}
	return "", false
}

// SetUserContext sets all user-related context values from JWT claims.
// This is a convenience function used by the authentication middleware.
func SetUserContext(ctx context.Context, userID uuid.UUID, email, role string, sessionID uuid.UUID) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, userID)
	ctx = context.WithValue(ctx, UserEmailKey, email)
	ctx = context.WithValue(ctx, UserRoleKey, role)
	ctx = context.WithValue(ctx, SessionIDKey, sessionID)
	return ctx
}

// MustGetUserID retrieves the user ID from context or panics.
// Use only in protected routes where authentication middleware guarantees user context exists.
func MustGetUserID(ctx context.Context) uuid.UUID {
	userID, ok := GetUserID(ctx)
	if !ok {
		panic("user_id not found in context - did you forget JWTAuth middleware?")
	}
	return userID
}

// MustGetUserIDString retrieves the user ID as string from context or panics.
// Use only in protected routes where authentication middleware guarantees user context exists.
func MustGetUserIDString(ctx context.Context) string {
	return MustGetUserID(ctx).String()
}

// MustGetUserEmail retrieves the user email from context or panics.
// Use only in protected routes where authentication middleware guarantees user context exists.
func MustGetUserEmail(ctx context.Context) string {
	email, ok := GetUserEmail(ctx)
	if !ok {
		panic("user_email not found in context - did you forget JWTAuth middleware?")
	}
	return email
}

// MustGetUserRole retrieves the user role from context or panics.
// Use only in protected routes where authentication middleware guarantees user context exists.
func MustGetUserRole(ctx context.Context) string {
	role, ok := GetUserRole(ctx)
	if !ok {
		panic("user_role not found in context - did you forget JWTAuth middleware?")
	}
	return role
}

// MustGetSessionID retrieves the session ID from context or panics.
// Use only in protected routes where authentication middleware guarantees user context exists.
func MustGetSessionID(ctx context.Context) uuid.UUID {
	sessionID, ok := GetSessionID(ctx)
	if !ok {
		panic("session_id not found in context - did you forget JWTAuth middleware?")
	}
	return sessionID
}
