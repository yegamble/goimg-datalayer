package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// UpdateUserCommand updates a user's profile information.
// This is a write operation that modifies the user aggregate.
// Optional fields use pointers (nil = no change).
type UpdateUserCommand struct {
	UserID      uuid.UUID
	RequestorID uuid.UUID
	DisplayName *string // Optional: nil means no change
	Bio         *string // Optional: nil means no change
}

// UpdateUserHandler processes UpdateUserCommand requests.
// It retrieves a user, applies updates, and persists changes.
type UpdateUserHandler struct {
	userRepo identity.UserRepository
}

// NewUpdateUserHandler creates a new UpdateUserHandler with the given dependencies.
func NewUpdateUserHandler(userRepo identity.UserRepository) *UpdateUserHandler {
	return &UpdateUserHandler{
		userRepo: userRepo,
	}
}

// Handle executes the UpdateUserCommand and returns the updated user data.
// Authorization: The requestor must own the user ID (or be an admin in future).
// Returns:
//   - *dto.UserDTO: The updated user data
//   - error: Authorization errors, validation errors, or repository errors
func (h *UpdateUserHandler) Handle(ctx context.Context, cmd UpdateUserCommand) (*dto.UserDTO, error) {
	// Authorization: Verify requestor owns the user ID
	// In a production system, this would also check for admin role override
	if cmd.RequestorID != cmd.UserID {
		return nil, fmt.Errorf("unauthorized: cannot update another user's profile")
	}

	// Convert UUID to domain UserID value object
	userID, err := identity.ParseUserID(cmd.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// Retrieve user from repository
	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	// Apply updates only if fields are provided (non-nil)
	updateNeeded := false

	displayName := user.DisplayName()
	if cmd.DisplayName != nil {
		displayName = *cmd.DisplayName
		updateNeeded = true
	}

	bio := user.Bio()
	if cmd.Bio != nil {
		bio = *cmd.Bio
		updateNeeded = true
	}

	// If no fields to update, return current state
	if !updateNeeded {
		userDTO := dto.FromDomain(user)
		return &userDTO, nil
	}

	// Update profile via domain method (validates constraints)
	if err := user.UpdateProfile(displayName, bio); err != nil {
		return nil, fmt.Errorf("update user profile: %w", err)
	}

	// Persist changes
	if err := h.userRepo.Save(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	// Convert updated user to DTO
	userDTO := dto.FromDomain(user)
	return &userDTO, nil
}
