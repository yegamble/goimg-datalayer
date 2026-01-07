// Package identity implements the Identity bounded context for user authentication and authorization.
package identity

import "errors"

// Domain-specific errors for the Identity bounded context.
var (
	// ErrEmailEmpty indicates the email address is empty.
	ErrEmailEmpty = errors.New("email cannot be empty")
	// ErrEmailInvalid indicates the email format is invalid.
	ErrEmailInvalid = errors.New("email format is invalid")
	// ErrEmailTooLong indicates the email exceeds the maximum length.
	ErrEmailTooLong = errors.New("email exceeds 255 characters")
	// ErrEmailDisposable indicates a disposable email address was provided.
	ErrEmailDisposable = errors.New("disposable email addresses not allowed")

	// ErrUsernameEmpty indicates the username is empty.
	ErrUsernameEmpty = errors.New("username cannot be empty")
	// ErrUsernameTooShort indicates the username is too short.
	ErrUsernameTooShort = errors.New("username must be at least 3 characters")
	// ErrUsernameTooLong indicates the username is too long.
	ErrUsernameTooLong = errors.New("username cannot exceed 32 characters")
	// ErrUsernameInvalid indicates the username contains invalid characters.
	ErrUsernameInvalid = errors.New("username must be alphanumeric with underscores")
	// ErrUsernameReserved indicates the username is reserved.
	ErrUsernameReserved = errors.New("username is reserved")

	// ErrPasswordEmpty indicates the password is empty.
	ErrPasswordEmpty = errors.New("password cannot be empty")
	// ErrPasswordTooShort indicates the password is too short.
	ErrPasswordTooShort = errors.New("password must be at least 12 characters")
	// ErrPasswordTooLong indicates the password is too long.
	ErrPasswordTooLong = errors.New("password cannot exceed 128 characters")
	// ErrPasswordWeak indicates the password is too common.
	ErrPasswordWeak = errors.New("password is too common")
	// ErrPasswordCompromised indicates the password has been found in a data breach.
	ErrPasswordCompromised = errors.New("password has been found in a data breach and cannot be used")
	// ErrPasswordMismatch indicates the password does not match the stored hash.
	ErrPasswordMismatch = errors.New("password does not match")

	// ErrUserNotFound indicates a user was not found.
	ErrUserNotFound = errors.New("user not found")
	// ErrEmailExists indicates an email is already registered.
	ErrEmailExists = errors.New("email already registered")
	// ErrUsernameExists indicates a username is already taken.
	ErrUsernameExists = errors.New("username already taken")
	// ErrUserSuspended indicates a user account is suspended.
	ErrUserSuspended = errors.New("user account is suspended")
	// ErrInvalidCredentials indicates authentication credentials are invalid.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserDeleted indicates a user account is deleted.
	ErrUserDeleted = errors.New("user account is deleted")
	// ErrInvalidUserStatus indicates an invalid user status transition.
	ErrInvalidUserStatus = errors.New("invalid user status transition")

	// 2FA-related errors
	// ErrTOTPAlreadyEnabled indicates 2FA is already enabled for the user.
	ErrTOTPAlreadyEnabled = errors.New("two-factor authentication is already enabled")
	// ErrTOTPNotEnabled indicates 2FA is not enabled for the user.
	ErrTOTPNotEnabled = errors.New("two-factor authentication is not enabled")
	// ErrTOTPNotVerified indicates 2FA setup was started but not verified.
	ErrTOTPNotVerified = errors.New("two-factor authentication setup not verified")
	// ErrTOTPSecretEmpty indicates the TOTP secret is empty.
	ErrTOTPSecretEmpty = errors.New("TOTP secret cannot be empty")
	// ErrTOTPCodeInvalid indicates the TOTP code is invalid.
	ErrTOTPCodeInvalid = errors.New("invalid two-factor authentication code")
	// ErrTOTPCodeExpired indicates the TOTP code has expired.
	ErrTOTPCodeExpired = errors.New("two-factor authentication code has expired")
	// ErrBackupCodeInvalid indicates the backup code is invalid or already used.
	ErrBackupCodeInvalid = errors.New("invalid or already used backup code")
	// ErrBackupCodesExhausted indicates all backup codes have been used.
	ErrBackupCodesExhausted = errors.New("all backup codes have been used")
	// Err2FARequired indicates 2FA verification is required to complete login.
	Err2FARequired = errors.New("two-factor authentication verification required")

	// OAuth-related errors
	// ErrProviderUserIDEmpty indicates the provider user ID is empty.
	ErrProviderUserIDEmpty = errors.New("provider user ID cannot be empty")
	// ErrProviderUserIDTooLong indicates the provider user ID exceeds maximum length.
	ErrProviderUserIDTooLong = errors.New("provider user ID exceeds 255 characters")
	// ErrOAuthAccountNotFound indicates an OAuth account was not found.
	ErrOAuthAccountNotFound = errors.New("OAuth account not found")
	// ErrOAuthAccountExists indicates an OAuth account already exists for this provider.
	ErrOAuthAccountExists = errors.New("OAuth account already exists for this provider")
	// ErrOAuthProviderNotLinked indicates the user has not linked this OAuth provider.
	ErrOAuthProviderNotLinked = errors.New("OAuth provider not linked to user account")

	// Follow-related errors
	// ErrCannotFollowSelf indicates a user attempted to follow themselves.
	ErrCannotFollowSelf = errors.New("cannot follow yourself")
	// ErrFollowAlreadyExists indicates a follow relationship already exists.
	ErrFollowAlreadyExists = errors.New("already following this user")
	// ErrFollowNotFound indicates a follow relationship was not found.
	ErrFollowNotFound = errors.New("follow relationship not found")
)
