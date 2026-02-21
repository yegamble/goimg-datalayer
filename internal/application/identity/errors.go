package identity

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")

	ErrAccountLocked = errors.New("account temporarily locked due to multiple failed attempts")

	ErrAccountSuspended = errors.New("account suspended")

	ErrAccountDeleted = errors.New("account has been deleted")

	ErrEmailAlreadyExists = errors.New("email address already registered")

	ErrUsernameAlreadyExists = errors.New("username already taken")

	ErrTokenExpired = errors.New("token has expired")

	ErrTokenRevoked = errors.New("token has been revoked")

	ErrTokenReplayDetected = errors.New("token replay detected - all tokens in family revoked")

	ErrSessionNotFound = errors.New("session not found")

	ErrSessionExpired = errors.New("session has expired")

	ErrInvalidToken = errors.New("invalid token")

	ErrTokenBlacklisted = errors.New("token has been blacklisted")

	ErrUnauthorized = errors.New("unauthorized")

	ErrForbidden = errors.New("forbidden - insufficient permissions")

	ErrPasswordCompromised = errors.New("password has been found in data breaches")

	Err2FAAlreadyEnabled = errors.New("two-factor authentication is already enabled")

	Err2FANotEnabled = errors.New("two-factor authentication is not enabled")

	Err2FASetupPending = errors.New("two-factor authentication setup pending verification")

	Err2FAInvalidCode = errors.New("invalid two-factor authentication code")

	Err2FARequired = errors.New("two-factor authentication verification required")

	ErrBackupCodeInvalid = errors.New("invalid or already used backup code")

	ErrBackupCodesExhausted = errors.New("all backup codes have been used - please regenerate")

	ErrPasswordRequired = errors.New("password confirmation required for this operation")

	ErrPasswordResetTokenInvalid = errors.New("password reset token is invalid or has expired")

	ErrPasswordResetTokenUsed = errors.New("password reset token has already been used")

	ErrSMTPDisabled = errors.New("smtp is disabled")
)
