package handlers

// HTTP-specific request DTOs for the handlers layer.
// These DTOs are separate from application-layer DTOs and represent the HTTP contract.
// They include JSON tags and validation rules using go-playground/validator.

// RegisterRequest represents the HTTP request body for user registration.
// POST /api/v1/auth/register
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Password string `json:"password" validate:"required,min=12,max=128"`
}

// LoginRequest represents the HTTP request body for user login.
// POST /api/v1/auth/login
//
// Identifier can be either email or username.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest represents the HTTP request body for token refresh.
// POST /api/v1/auth/refresh
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest represents the HTTP request body for logout.
// POST /api/v1/auth/logout
//
// Both fields are optional. If neither is provided, logout uses the session from JWT context.
// If logout_all is true, all sessions for the user are revoked.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
	LogoutAll    bool   `json:"logout_all,omitempty"`
}

// UpdateUserRequest represents the HTTP request body for updating user profile.
// PUT /api/v1/users/{id}
//
// All fields are optional (use pointers to indicate "no change").
// Only provided fields will be updated.
type UpdateUserRequest struct {
	DisplayName *string `json:"display_name,omitempty" validate:"omitempty,max=100"`
	Bio         *string `json:"bio,omitempty" validate:"omitempty,max=500"`
}

// DeleteUserRequest represents the HTTP request body for deleting a user account.
// DELETE /api/v1/users/{id}
//
// Requires password confirmation to prevent accidental deletion.
type DeleteUserRequest struct {
	Password string `json:"password" validate:"required"`
}
