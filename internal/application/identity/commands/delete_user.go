package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/yegamble/goimg-datalayer/internal/application/identity"
	domainIdentity "github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// DeleteUserCommand soft-deletes a user account.
// This is a write operation that marks the user as deleted.
// Requires password confirmation to prevent accidental deletion.
type DeleteUserCommand struct {
	UserID      uuid.UUID
	RequestorID uuid.UUID
	Password    string // Require password confirmation for safety
}

// Implement Command interface
func (DeleteUserCommand) isCommand() {}

// DeleteUserHandler processes DeleteUserCommand requests.
// It verifies authorization and password, then soft-deletes the user and revokes all sessions.
type DeleteUserHandler struct {
	userRepo     domainIdentity.UserRepository
	sessionStore identity.SessionStore
}

// NewDeleteUserHandler creates a new DeleteUserHandler with the given dependencies.
func NewDeleteUserHandler(
	userRepo domainIdentity.UserRepository,
	sessionStore identity.SessionStore,
) *DeleteUserHandler {
	return &DeleteUserHandler{
		userRepo:     userRepo,
		sessionStore: sessionStore,
	}
}

// Handle executes the DeleteUserCommand.
// Authorization: The requestor must own the user ID and provide correct password.
// This operation:
//  1. Verifies authorization and password
//  2. Soft-deletes the user (via repository Delete method)
//  3. Revokes all active sessions
//
// Returns:
//   - error: Authorization errors, invalid password, or repository errors
func (h *DeleteUserHandler) Handle(ctx context.Context, cmd DeleteUserCommand) error {
	// Authorization: Verify requestor owns the user ID
	if cmd.RequestorID != cmd.UserID {
		return fmt.Errorf("unauthorized: cannot delete another user's account")
	}

	// Convert UUID to domain UserID value object
	userID, err := domainIdentity.ParseUserID(cmd.UserID.String())
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	// Retrieve user from repository
	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	// Verify password before allowing deletion
	if err := user.VerifyPassword(cmd.Password); err != nil {
		return fmt.Errorf("password verification failed: %w", domainIdentity.ErrInvalidCredentials)
	}

	// Soft delete the user (repository implementation should set deleted_at)
	if err := h.userRepo.Delete(ctx, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	// Revoke all sessions to immediately log out the user from all devices
	if err := h.sessionStore.RevokeAll(ctx, cmd.UserID); err != nil {
		// Log the error but don't fail the operation since user is already deleted
		// In production, this should be logged with a proper logger
		return fmt.Errorf("revoke user sessions: %w", err)
	}

	return nil
}
