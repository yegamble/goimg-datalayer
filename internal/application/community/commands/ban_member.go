package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appcommunity "github.com/yegamble/goimg-datalayer/internal/application/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// BanMemberCommand represents the intent to ban a member from a group.
// Banned members cannot rejoin until unbanned.
// Only admins and owners can ban members.
// Owners cannot be banned.
type BanMemberCommand struct {
	GroupID  community.GroupID
	ActorID  identity.UserID // User performing the ban
	TargetID identity.UserID // User being banned
	Reason   string          // Reason for the ban
}

// Implement Command interface.
func (BanMemberCommand) isCommand() {}

// BanMemberHandler processes member ban commands.
type BanMemberHandler struct {
	groupRepo      community.GroupRepository
	membershipRepo community.GroupMembershipRepository
	activityRepo   community.GroupActivityRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewBanMemberHandler creates a new BanMemberHandler with the given dependencies.
func NewBanMemberHandler(
	groupRepo community.GroupRepository,
	membershipRepo community.GroupMembershipRepository,
	activityRepo community.GroupActivityRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *BanMemberHandler {
	return &BanMemberHandler{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		activityRepo:   activityRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the member ban use case.
//
// Process flow:
//  1. Load actor's and target's memberships
//  2. Verify actor has permission to ban members
//  3. Verify target can be banned (not owner, not already banned)
//  4. Ban the member via domain method
//  5. Decrement member count (if target was active)
//  6. Persist changes
//  7. Publish domain events
//
// Returns:
//   - nil on successful ban
//   - ErrInsufficientGroupRole if actor lacks permission
//   - ErrCannotBanOwner if trying to ban the owner
//   - ErrMemberAlreadyBanned if member is already banned
func (h *BanMemberHandler) Handle(ctx context.Context, cmd BanMemberCommand) error {
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

	// 2. Verify actor has permission to ban members
	if err := ValidateAdminOrOwner(actorMembership); err != nil {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("actor_role", actorMembership.Role().String()).
			Msg("unauthorized ban attempt")
		return fmt.Errorf("validate ban permission: %w", err)
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

	// Admins cannot ban other admins (only owner can)
	if targetMembership.IsAdmin() && actorMembership.IsAdmin() && !actorMembership.IsOwner() {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("target_id", cmd.TargetID.String()).
			Msg("admin attempted to ban another admin")
		return community.ErrInsufficientGroupRole
	}

	// Check if member was active (for member count update)
	wasActive := targetMembership.IsActive()

	// 4. Ban the member via domain method
	if err := targetMembership.Ban(cmd.ActorID, cmd.Reason); err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("target_id", cmd.TargetID.String()).
			Msg("failed to ban member")
		return fmt.Errorf("ban member: %w", err)
	}

	// 5. Persist membership changes
	if err := h.membershipRepo.Save(ctx, targetMembership); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("target_id", cmd.TargetID.String()).
			Msg("failed to save banned membership")
		return fmt.Errorf("save membership: %w", err)
	}

	// 6. Decrement member count if target was active
	if wasActive {
		group, err := h.groupRepo.FindByID(ctx, cmd.GroupID)
		if err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Msg("failed to load group for member count update")
			return fmt.Errorf("find group: %w", err)
		}

		group.DecrementMemberCount()

		// Persist group changes
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

	// 7. Publish membership domain events AFTER successful save
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

	// 8. Create audit log activity for the ban action
	activity, err := community.NewMemberBannedActivity(
		cmd.GroupID,
		cmd.ActorID,
		cmd.TargetID,
		cmd.Reason,
	)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("target_id", cmd.TargetID.String()).
			Msg("failed to create ban activity")
		// Don't fail the command if audit logging fails
	} else {
		if err := h.activityRepo.Save(ctx, activity); err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Str("activity_id", activity.ID().String()).
				Msg("failed to save ban activity")
			// Don't fail the command if audit logging fails
		}
	}

	h.logger.Info().
		Str("group_id", cmd.GroupID.String()).
		Str("actor_id", cmd.ActorID.String()).
		Str("target_id", cmd.TargetID.String()).
		Str("reason", cmd.Reason).
		Msg("member banned successfully")

	return nil
}
