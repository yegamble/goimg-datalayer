package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// GetUserQuery retrieves a user by their unique ID.
// This is a read-only operation with no side effects.
type GetUserQuery struct {
	UserID      uuid.UUID
	RequestorID uuid.UUID // Who is requesting (for authorization/audit)
}

// Implement Query interface
func (GetUserQuery) isQuery() {}

// GetUserHandler processes GetUserQuery requests.
// It retrieves a user from the repository and converts it to a DTO.
type GetUserHandler struct {
	userRepo identity.UserRepository
}

// NewGetUserHandler creates a new GetUserHandler with the given dependencies.
func NewGetUserHandler(userRepo identity.UserRepository) *GetUserHandler {
	return &GetUserHandler{
		userRepo: userRepo,
	}
}

// Handle executes the GetUserQuery and returns the user data.
// Returns:
//   - *dto.UserDTO: The user data excluding sensitive fields
//   - error: ErrUserNotFound if the user does not exist, or other repository errors
func (h *GetUserHandler) Handle(ctx context.Context, q GetUserQuery) (*dto.UserDTO, error) {
	// Convert UUID to domain UserID value object
	userID, err := identity.ParseUserID(q.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// Retrieve user from a repository
	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	// Convert domain User to UserDTO (excludes password hash)
	userDTO := dto.FromDomain(user)
	return &userDTO, nil
}
