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
)
