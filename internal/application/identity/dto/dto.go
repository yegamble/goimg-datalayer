// Package dto provides data transfer objects for identity operations.
package dto

import (
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// UserDTO represents a user in API responses.
// It excludes sensitive fields like password hash and is safe for external consumption.
type UserDTO struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	DisplayName string    `json:"display_name"`
	Bio         string    `json:"bio"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FromDomain converts a domain User aggregate to a UserDTO.
func FromDomain(user *identity.User) UserDTO {
	return UserDTO{
		ID:          user.ID().String(),
		Email:       user.Email().String(),
		Username:    user.Username().String(),
		Role:        user.Role().String(),
		Status:      user.Status().String(),
		DisplayName: user.DisplayName(),
		Bio:         user.Bio(),
		CreatedAt:   user.CreatedAt(),
		UpdatedAt:   user.UpdatedAt(),
	}
}

// TokenPairDTO contains access and refresh tokens returned after authentication.
type TokenPairDTO struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"` // Always "Bearer"
	ExpiresIn    int64     `json:"expires_in"` // Access token expiry in seconds
	ExpiresAt    time.Time `json:"expires_at"` // Access token expiry timestamp
}

// NewTokenPairDTO creates a TokenPairDTO with the given tokens and expiry.
func NewTokenPairDTO(accessToken, refreshToken string, expiresAt time.Time) TokenPairDTO {
	now := time.Now().UTC()
	expiresIn := int64(expiresAt.Sub(now).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}

	return TokenPairDTO{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		ExpiresAt:    expiresAt,
	}
}

// CreateUserDTO represents the request to create a new user account.
type CreateUserDTO struct {
	Email    string `json:"email"    validate:"required,email,max=255"`
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

// LoginDTO represents the request to authenticate a user.
type LoginDTO struct {
	// Identifier can be either email or username
	Identifier string `json:"identifier" validate:"required"`
	Password   string `json:"password"   validate:"required"`
	IP         string `json:"-"` // Not from request body, set by middleware
	UserAgent  string `json:"-"` // Not from request body, set by middleware
}

// RefreshTokenDTO represents the request to refresh an access token.
type RefreshTokenDTO struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
	IP           string `json:"-"` // Not from request body, set by middleware
	UserAgent    string `json:"-"` // Not from request body, set by middleware
}

// UpdateUserDTO represents the request to update user profile information.
// All fields are optional (pointer types indicate this).
type UpdateUserDTO struct {
	DisplayName *string `json:"display_name,omitempty" validate:"omitempty,max=100"`
	Bio         *string `json:"bio,omitempty"          validate:"omitempty,max=500"`
}

// ChangePasswordDTO represents the request to change a user's password.
type ChangePasswordDTO struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password"     validate:"required,min=8,max=128"`
}

// SessionDTO represents an active user session in API responses.
type SessionDTO struct {
	SessionID string    `json:"session_id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IsCurrent bool      `json:"is_current"` // Whether this is the current session
}

// ListUsersDTO represents the request to list users with filters and pagination.
type ListUsersDTO struct {
	Role   *string `json:"role"   validate:"omitempty,oneof=user moderator admin"`
	Status *string `json:"status" validate:"omitempty,oneof=pending active suspended deleted"`
	Search string  `json:"search" validate:"omitempty,max=255"`
	Offset int     `json:"offset" validate:"min=0"`
	Limit  int     `json:"limit"  validate:"min=1,max=100"`
}

// ListUsersResultDTO represents the paginated response for listing users.
type ListUsersResultDTO struct {
	Users      []UserDTO `json:"users"`
	TotalCount int       `json:"total_count"`
	Offset     int       `json:"offset"`
	Limit      int       `json:"limit"`
}

// AuthResponseDTO represents the response after successful authentication or registration.
// It includes both the user data and token pair.
type AuthResponseDTO struct {
	User   UserDTO      `json:"user"`
	Tokens TokenPairDTO `json:"tokens"`
}

// NewAuthResponseDTO creates an AuthResponseDTO from a domain User and token pair.
func NewAuthResponseDTO(user *identity.User, tokens TokenPairDTO) AuthResponseDTO {
	return AuthResponseDTO{
		User:   FromDomain(user),
		Tokens: tokens,
	}
}

// MessageDTO represents a simple message response (e.g., success confirmations).
type MessageDTO struct {
	Message string `json:"message"`
}

// NewMessageDTO creates a MessageDTO with the given message.
func NewMessageDTO(message string) MessageDTO {
	return MessageDTO{
		Message: message,
	}
}
