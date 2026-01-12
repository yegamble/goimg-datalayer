package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appcommunity "github.com/yegamble/goimg-datalayer/internal/application/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// UpdateGroupCommand represents the intent to update a group's metadata or settings.
type UpdateGroupCommand struct {
	GroupID     community.GroupID
	ActorID     identity.UserID          // User performing the action
	Description *string                  // Optional: update description
	Settings    *community.GroupSettings // Optional: update settings
}

// Implement Command interface.
func (UpdateGroupCommand) isCommand() {}

// UpdateGroupHandler processes group update commands.
type UpdateGroupHandler struct {
	groupRepo      community.GroupRepository
	membershipRepo community.GroupMembershipRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewUpdateGroupHandler creates a new UpdateGroupHandler with the given dependencies.
func NewUpdateGroupHandler(
	groupRepo community.GroupRepository,
	membershipRepo community.GroupMembershipRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *UpdateGroupHandler {
	return &UpdateGroupHandler{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the group update use case.
//
// Process flow:
//  1. Load the group aggregate
//  2. Check authorization (only owner or admin can update)
//  3. Apply updates via domain methods
//  4. Persist changes
//  5. Publish domain events
//
// Returns:
//   - *community.Group on successful update
//   - ErrGroupNotFound if the group doesn't exist
//   - ErrInsufficientGroupRole if actor lacks permission
func (h *UpdateGroupHandler) Handle(ctx context.Context, cmd UpdateGroupCommand) (*community.Group, error) {
	// 1. Load the group aggregate
	group, err := h.groupRepo.FindByID(ctx, cmd.GroupID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Msg("group not found during update")
		return nil, fmt.Errorf("find group: %w", err)
	}

	// 2. Check authorization - load actor's membership
	membership, err := h.membershipRepo.FindByGroupAndUser(ctx, cmd.GroupID, cmd.ActorID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Msg("actor is not a group member")
		return nil, fmt.Errorf("find membership: %w", err)
	}

	// Only admins and owner can update group settings
	if !membership.CanManageMembers() {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("role", membership.Role().String()).
			Msg("unauthorized group update attempt")
		return nil, community.ErrInsufficientGroupRole
	}

	// 3. Apply updates via domain methods
	if cmd.Description != nil {
		if err := group.UpdateDescription(*cmd.Description); err != nil {
			h.logger.Debug().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Msg("invalid description during update")
			return nil, fmt.Errorf("update description: %w", err)
		}
	}

	if cmd.Settings != nil {
		if err := group.UpdateSettings(*cmd.Settings); err != nil {
			h.logger.Debug().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Msg("invalid settings during update")
			return nil, fmt.Errorf("update settings: %w", err)
		}
	}

	// 4. Persist changes
	if err := h.groupRepo.Save(ctx, group); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Msg("failed to save group updates")
		return nil, fmt.Errorf("save group: %w", err)
	}

	// 5. Publish domain events AFTER successful save
	for _, event := range group.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish group domain event")
		}
	}
	group.ClearEvents()

	h.logger.Info().
		Str("group_id", cmd.GroupID.String()).
		Str("actor_id", cmd.ActorID.String()).
		Msg("group updated successfully")

	return group, nil
}
