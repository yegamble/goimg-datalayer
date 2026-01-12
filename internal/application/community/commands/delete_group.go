package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appcommunity "github.com/yegamble/goimg-datalayer/internal/application/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// DeleteGroupCommand represents the intent to delete a group.
// Only the group owner can delete a group.
type DeleteGroupCommand struct {
	GroupID community.GroupID
	OwnerID identity.UserID // Must be the group owner
}

// Implement Command interface.
func (DeleteGroupCommand) isCommand() {}

// DeleteGroupHandler processes group deletion commands.
type DeleteGroupHandler struct {
	groupRepo      community.GroupRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewDeleteGroupHandler creates a new DeleteGroupHandler with the given dependencies.
func NewDeleteGroupHandler(
	groupRepo community.GroupRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *DeleteGroupHandler {
	return &DeleteGroupHandler{
		groupRepo:      groupRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the group deletion use case.
//
// Process flow:
//  1. Load the group aggregate
//  2. Verify ownership (only owner can delete)
//  3. Delete the group (soft delete in repository)
//  4. Publish domain events
//
// Returns:
//   - nil on successful deletion
//   - ErrGroupNotFound if the group doesn't exist
//   - Authorization error if actor is not the owner
//
func (h *DeleteGroupHandler) Handle(ctx context.Context, cmd DeleteGroupCommand) error {
	// 1. Load the group aggregate
	group, err := h.groupRepo.FindByID(ctx, cmd.GroupID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Msg("group not found during deletion")
		return fmt.Errorf("find group: %w", err)
	}

	// 2. Verify ownership
	if err := ValidateOwnership(group, cmd.OwnerID); err != nil {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.OwnerID.String()).
			Str("owner_id", group.OwnerID().String()).
			Msg("unauthorized group deletion attempt")
		return fmt.Errorf("verify ownership: %w", err)
	}

	// 3. Delete the group (note: repository typically implements soft delete)
	if err := h.groupRepo.Delete(ctx, cmd.GroupID); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Msg("failed to delete group")
		return fmt.Errorf("delete group: %w", err)
	}

	// Note: In a full implementation, you'd emit a GroupDeleted event here
	// The repository handles cascading deletes of memberships, albums, etc.

	h.logger.Info().
		Str("group_id", cmd.GroupID.String()).
		Str("owner_id", cmd.OwnerID.String()).
		Msg("group deleted successfully")

	return nil
}
