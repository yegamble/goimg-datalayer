package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appcommunity "github.com/yegamble/goimg-datalayer/internal/application/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// UpdateMemberRoleCommand represents the intent to change a member's role within a group.
// Only admins and owners can promote/demote members.
// Only owners can promote to admin.
type UpdateMemberRoleCommand struct {
	GroupID  community.GroupID
	ActorID  identity.UserID // User performing the action
	TargetID identity.UserID // User whose role is being changed
	NewRole  community.GroupRole
}

// Implement Command interface.
func (UpdateMemberRoleCommand) isCommand() {}

// UpdateMemberRoleHandler processes member role update commands.
type UpdateMemberRoleHandler struct {
	membershipRepo community.GroupMembershipRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewUpdateMemberRoleHandler creates a new UpdateMemberRoleHandler with the given dependencies.
func NewUpdateMemberRoleHandler(
	membershipRepo community.GroupMembershipRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *UpdateMemberRoleHandler {
	return &UpdateMemberRoleHandler{
		membershipRepo: membershipRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the member role update use case.
//
// Process flow:
//  1. Load actor's membership (for authorization)
//  2. Load target's membership
//  3. Verify actor has permission to change roles
//  4. Verify target role change is allowed (e.g., only owner can promote to admin)
//  5. Update role via domain method
//  6. Persist changes
//  7. Publish domain events
//
// Returns:
//   - *community.GroupMembership on successful role change
//   - ErrInsufficientGroupRole if actor lacks permission
//   - ErrCannotDemoteOwner if trying to demote the owner
func (h *UpdateMemberRoleHandler) Handle(ctx context.Context, cmd UpdateMemberRoleCommand) (*community.GroupMembership, error) {
	// 1. Load actor's membership
	actorMembership, err := h.membershipRepo.FindByGroupAndUser(ctx, cmd.GroupID, cmd.ActorID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Msg("actor membership not found")
		return nil, fmt.Errorf("find actor membership: %w", err)
	}

	// 2. Load target's membership
	targetMembership, err := h.membershipRepo.FindByGroupAndUser(ctx, cmd.GroupID, cmd.TargetID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("target_id", cmd.TargetID.String()).
			Msg("target membership not found")
		return nil, fmt.Errorf("find target membership: %w", err)
	}

	// 3. Verify actor has permission to change roles
	// Only admins and owners can manage members
	if err := ValidateAdminOrOwner(actorMembership); err != nil {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("actor_role", actorMembership.Role().String()).
			Msg("unauthorized role update attempt")
		return nil, err
	}

	// 4. Verify target role change is allowed
	// Only owners can promote to admin
	if cmd.NewRole == community.GroupRoleAdmin && !actorMembership.IsOwner() {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("actor_role", actorMembership.Role().String()).
			Msg("non-owner attempted to promote member to admin")
		return nil, community.ErrInsufficientGroupRole
	}

	// Admins cannot demote other admins (only owner can)
	if targetMembership.IsAdmin() && actorMembership.IsAdmin() && !actorMembership.IsOwner() {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("target_id", cmd.TargetID.String()).
			Msg("admin attempted to demote another admin")
		return nil, community.ErrInsufficientGroupRole
	}

	// 5. Update role via domain method
	if err := targetMembership.ChangeRole(cmd.NewRole); err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("target_id", cmd.TargetID.String()).
			Str("new_role", cmd.NewRole.String()).
			Msg("invalid role change")
		return nil, fmt.Errorf("change role: %w", err)
	}

	// 6. Persist changes
	if err := h.membershipRepo.Save(ctx, targetMembership); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("target_id", cmd.TargetID.String()).
			Msg("failed to save membership role update")
		return nil, fmt.Errorf("save membership: %w", err)
	}

	// 7. Publish domain events AFTER successful save
	for _, event := range targetMembership.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish membership domain event")
		}
	}
	targetMembership.ClearEvents()

	h.logger.Info().
		Str("group_id", cmd.GroupID.String()).
		Str("actor_id", cmd.ActorID.String()).
		Str("target_id", cmd.TargetID.String()).
		Str("new_role", cmd.NewRole.String()).
		Msg("member role updated successfully")

	return targetMembership, nil
}
