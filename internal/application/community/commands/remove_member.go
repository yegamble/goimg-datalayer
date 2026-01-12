package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appcommunity "github.com/yegamble/goimg-datalayer/internal/application/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// RemoveMemberCommand represents the intent to remove a member from a group.
// Only admins and owners can remove members.
// Owners cannot be removed.
type RemoveMemberCommand struct {
	GroupID  community.GroupID
	ActorID  identity.UserID // User performing the removal
	TargetID identity.UserID // User being removed
}

// Implement Command interface.
func (RemoveMemberCommand) isCommand() {}

// RemoveMemberHandler processes member removal commands.
type RemoveMemberHandler struct {
	groupRepo      community.GroupRepository
	membershipRepo community.GroupMembershipRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewRemoveMemberHandler creates a new RemoveMemberHandler with the given dependencies.
func NewRemoveMemberHandler(
	groupRepo community.GroupRepository,
	membershipRepo community.GroupMembershipRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *RemoveMemberHandler {
	return &RemoveMemberHandler{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the member removal use case.
//
// Process flow:
//  1. Load actor's and target's memberships
//  2. Verify actor has permission to remove members
//  3. Verify target can be removed (not owner)
//  4. Delete membership
//  5. Decrement member count (if target was active)
//  6. Persist changes
//  7. Publish domain events
//
// Returns:
//   - nil on successful removal
//   - ErrInsufficientGroupRole if actor lacks permission
//   - ErrCannotRemoveOwner if trying to remove the owner
//
func (h *RemoveMemberHandler) Handle(ctx context.Context, cmd RemoveMemberCommand) error {
	// 1. Load actor's membership
	actorMembership, err := h.membershipRepo.FindByGroupAndUser(ctx, cmd.GroupID, cmd.ActorID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Msg("actor membership not found")
		return fmt.Errorf("find actor membership: %w", err)
	}

	// 2. Verify actor has permission to remove members
	if err := ValidateAdminOrOwner(actorMembership); err != nil {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("actor_role", actorMembership.Role().String()).
			Msg("unauthorized member removal attempt")
		return err
	}

	// 3. Load target's membership
	targetMembership, err := h.membershipRepo.FindByGroupAndUser(ctx, cmd.GroupID, cmd.TargetID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("target_id", cmd.TargetID.String()).
			Msg("target membership not found")
		return fmt.Errorf("find target membership: %w", err)
	}

	// Verify target can be removed (not owner)
	if targetMembership.IsOwner() {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("target_id", cmd.TargetID.String()).
			Msg("attempted to remove group owner")
		return community.ErrCannotRemoveOwner
	}

	// Admins cannot remove other admins (only owner can)
	if targetMembership.IsAdmin() && actorMembership.IsAdmin() && !actorMembership.IsOwner() {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("target_id", cmd.TargetID.String()).
			Msg("admin attempted to remove another admin")
		return community.ErrInsufficientGroupRole
	}

	// 4. Delete membership
	if err := h.membershipRepo.Delete(ctx, targetMembership.ID()); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("target_id", cmd.TargetID.String()).
			Msg("failed to delete membership")
		return fmt.Errorf("delete membership: %w", err)
	}

	// 5. Decrement member count (only for active members)
	if targetMembership.IsActive() {
		group, err := h.groupRepo.FindByID(ctx, cmd.GroupID)
		if err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Msg("failed to load group for member count update")
			return fmt.Errorf("find group: %w", err)
		}

		group.DecrementMemberCount()

		// 6. Persist group changes
		if err := h.groupRepo.Save(ctx, group); err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Msg("failed to update group member count")
			return fmt.Errorf("save group: %w", err)
		}

		// Publish group events
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
	}

	h.logger.Info().
		Str("group_id", cmd.GroupID.String()).
		Str("actor_id", cmd.ActorID.String()).
		Str("target_id", cmd.TargetID.String()).
		Msg("member removed successfully")

	return nil
}
