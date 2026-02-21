package middleware

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	RequestIDKey contextKey = "requestID"

	UserIDKey contextKey = "userID"

	UserEmailKey contextKey = "userEmail"

	UserRoleKey contextKey = "userRole"

	SessionIDKey contextKey = "sessionID"

	TwoFAVerifiedKey contextKey = "twofaVerified"

	EmailVerifiedKey contextKey = "emailVerified"
)

func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	if userID, ok := ctx.Value(UserIDKey).(uuid.UUID); ok {
		return userID, true
	}
	return uuid.Nil, false
}

func GetUserIDString(ctx context.Context) (string, bool) {
	if userID, ok := GetUserID(ctx); ok {
		return userID.String(), true
	}
	return "", false
}

func GetUserEmail(ctx context.Context) (string, bool) {
	if email, ok := ctx.Value(UserEmailKey).(string); ok {
		return email, true
	}
	return "", false
}

func GetUserRole(ctx context.Context) (string, bool) {
	if role, ok := ctx.Value(UserRoleKey).(string); ok {
		return role, true
	}
	return "", false
}

func GetSessionID(ctx context.Context) (uuid.UUID, bool) {
	if sessionID, ok := ctx.Value(SessionIDKey).(uuid.UUID); ok {
		return sessionID, true
	}
	return uuid.Nil, false
}

func GetSessionIDString(ctx context.Context) (string, bool) {
	if sessionID, ok := GetSessionID(ctx); ok {
		return sessionID.String(), true
	}
	return "", false
}

func Get2FAVerified(ctx context.Context) (bool, bool) {
	if verified, ok := ctx.Value(TwoFAVerifiedKey).(bool); ok {
		return verified, true
	}
	return false, false
}

func GetEmailVerified(ctx context.Context) (bool, bool) {
	if verified, ok := ctx.Value(EmailVerifiedKey).(bool); ok {
		return verified, true
	}
	return false, false
}

func SetUserContext(ctx context.Context, userID uuid.UUID, email, role string, sessionID uuid.UUID, twofaVerified, emailVerified bool) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, userID)
	ctx = context.WithValue(ctx, UserEmailKey, email)
	ctx = context.WithValue(ctx, UserRoleKey, role)
	ctx = context.WithValue(ctx, SessionIDKey, sessionID)
	ctx = context.WithValue(ctx, TwoFAVerifiedKey, twofaVerified)
	ctx = context.WithValue(ctx, EmailVerifiedKey, emailVerified)
	return ctx
}

func MustGetUserID(ctx context.Context) uuid.UUID {
	userID, ok := GetUserID(ctx)
	if !ok {
		panic("user_id not found in context - did you forget JWTAuth middleware?")
	}
	return userID
}

func MustGetUserIDString(ctx context.Context) string {
	return MustGetUserID(ctx).String()
}

func MustGetUserEmail(ctx context.Context) string {
	email, ok := GetUserEmail(ctx)
	if !ok {
		panic("user_email not found in context - did you forget JWTAuth middleware?")
	}
	return email
}

func MustGetUserRole(ctx context.Context) string {
	role, ok := GetUserRole(ctx)
	if !ok {
		panic("user_role not found in context - did you forget JWTAuth middleware?")
	}
	return role
}

func MustGetSessionID(ctx context.Context) uuid.UUID {
	sessionID, ok := GetSessionID(ctx)
	if !ok {
		panic("session_id not found in context - did you forget JWTAuth middleware?")
	}
	return sessionID
}
