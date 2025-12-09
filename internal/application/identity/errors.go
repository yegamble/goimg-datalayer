// Package identity implements application-layer use cases for authentication and user management.
package identity

import "errors"

// Application-level errors for the Identity application layer.
// These errors wrap domain errors with additional context or represent
// application-specific concerns like token management and session handling.
var (
	// ErrInvalidCredentials is returned when authentication fails.
	// This generic error prevents user enumeration attacks by not revealing
	// whether the email/username exists or if the password is wrong.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrAccountLocked is returned when an account is temporarily locked
	// due to too many failed login attempts (rate limiting/brute force protection).
	ErrAccountLocked = errors.New("account temporarily locked due to multiple failed attempts")

	// ErrAccountSuspended is returned when attempting to login to a suspended account.
	// This is a business rule enforcement at the application layer.
	ErrAccountSuspended = errors.New("account suspended")

	// ErrAccountDeleted is returned when attempting operations on a deleted account.
	ErrAccountDeleted = errors.New("account has been deleted")

	// ErrEmailAlreadyExists is returned when attempting to register with an email
	// that already exists in the system.
	ErrEmailAlreadyExists = errors.New("email address already registered")

	// ErrUsernameAlreadyExists is returned when attempting to register with a username
	// that is already taken.
	ErrUsernameAlreadyExists = errors.New("username already taken")

	// ErrTokenExpired is returned when a token has expired.
	ErrTokenExpired = errors.New("token has expired")

	// ErrTokenRevoked is returned when a token has been explicitly revoked.
	ErrTokenRevoked = errors.New("token has been revoked")

	// ErrTokenReplayDetected is returned when a refresh token is reused,
	// indicating a potential replay attack. This triggers revocation of the entire
	// token family as a security measure.
	ErrTokenReplayDetected = errors.New("token replay detected - all tokens in family revoked")

	// ErrSessionNotFound is returned when a session cannot be found.
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionExpired is returned when a session has expired.
	ErrSessionExpired = errors.New("session has expired")

	// ErrInvalidToken is returned when a token is malformed or cannot be validated.
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenBlacklisted is returned when a token has been blacklisted.
	ErrTokenBlacklisted = errors.New("token has been blacklisted")

	// ErrUnauthorized is returned when a user lacks permission for an operation.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when a user is authenticated but lacks permission
	// for the requested resource or operation.
	ErrForbidden = errors.New("forbidden - insufficient permissions")
)
