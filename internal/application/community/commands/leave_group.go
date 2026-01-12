package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appcommunity "github.com/yegamble/goimg-datalayer/internal/application/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// LeaveGroupCommand represents the intent for a user to leave a group.
// Group owners cannot leave without first transferring ownership.
type LeaveGroupCommand struct {
	GroupID community.GroupID
	UserID  identity.UserID
}

// Implement Command interface.
func (LeaveGroupCommand) isCommand() {}

// LeaveGroupHandler processes group leave commands.
type LeaveGroupHandler struct {
	groupRepo      community.GroupRepository
	membershipRepo community.GroupMembershipRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewLeaveGroupHandler creates a new LeaveGroupHandler with the given dependencies.
func NewLeaveGroupHandler(
	groupRepo community.GroupRepository,
	membershipRepo community.GroupMembershipRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *LeaveGroupHandler {
	return &LeaveGroupHandler{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the leave group use case.
//
// Process flow:
//  1. Load group and membership
//  2. Verify user is not the owner (owners must transfer ownership first)
//  3. Delete membership
//  4. Decrement member count
//  5. Persist changes
//  6. Publish domain events
//
// Returns:
//   - nil on successful leave
//   - ErrCannotLeaveAsOwner if user is the owner
//   - ErrMembershipNotFound if user is not a member
func (h *LeaveGroupHandler) Handle(ctx context.Context, cmd LeaveGroupCommand) error {
	// 1. Load group and membership
	group, err := h.groupRepo.FindByID(ctx, cmd.GroupID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Msg("group not found during leave attempt")
		return fmt.Errorf("find group: %w", err)
	}

	membership, err := h.membershipRepo.FindByGroupAndUser(ctx, cmd.GroupID, cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("user_id", cmd.UserID.String()).
			Msg("membership not found during leave attempt")
		return fmt.Errorf("find membership: %w", err)
	}

	// 2. Verify user is not the owner
	if membership.IsOwner() {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("user_id", cmd.UserID.String()).
			Msg("owner attempted to leave without transferring ownership")
		return community.ErrCannotLeaveAsOwner
	}

	// 3. Delete membership
	if err := h.membershipRepo.Delete(ctx, membership.ID()); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("user_id", cmd.UserID.String()).
			Msg("failed to delete membership")
		return fmt.Errorf("delete membership: %w", err)
	}

	// 4. Decrement member count (only for active members)
	if membership.IsActive() {
		group.DecrementMemberCount()

		// 5. Persist group changes
		if err := h.groupRepo.Save(ctx, group); err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Msg("failed to update group member count")
			return fmt.Errorf("save group: %w", err)
		}
	}

	// 6. Publish domain events AFTER successful save
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
		Str("user_id", cmd.UserID.String()).
		Msg("user left group successfully")

	return nil
}
