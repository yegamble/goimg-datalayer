package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// GetUserByEmailQuery retrieves a user by their email address.
// This is a read-only operation with no side effects.
type GetUserByEmailQuery struct {
	Email string
}

// GetUserByEmailHandler processes GetUserByEmailQuery requests.
// It retrieves a user by email and converts it to a DTO.
type GetUserByEmailHandler struct {
	userRepo identity.UserRepository
}

// NewGetUserByEmailHandler creates a new GetUserByEmailHandler with the given dependencies.
func NewGetUserByEmailHandler(userRepo identity.UserRepository) *GetUserByEmailHandler {
	return &GetUserByEmailHandler{
		userRepo: userRepo,
	}
}

// Handle executes the GetUserByEmailQuery and returns the user data.
// Returns:
//   - *dto.UserDTO: The user data excluding sensitive fields
//   - error: ErrUserNotFound if no user with that email exists, or validation/repository errors
func (h *GetUserByEmailHandler) Handle(ctx context.Context, q GetUserByEmailQuery) (*dto.UserDTO, error) {
	// Convert string to domain Email value object (validates format)
	email, err := identity.NewEmail(q.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	// Retrieve user from repository
	user, err := h.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	// Convert domain User to UserDTO (excludes password hash)
	userDTO := dto.FromDomain(user)
	return &userDTO, nil
}
