package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	appcommunity "github.com/yegamble/goimg-datalayer/internal/application/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// JoinGroupCommand represents the intent for a user to join a group.
// Behavior depends on group type:
// - Public: instant join (status = Active)
// - InviteOnly: creates join request (status = Requested)
// - Private: forbidden (cannot join without invitation)
type JoinGroupCommand struct {
	GroupID community.GroupID
	UserID  identity.UserID
}

// Implement Command interface.
func (JoinGroupCommand) isCommand() {}

// JoinGroupHandler processes group join commands.
type JoinGroupHandler struct {
	groupRepo      community.GroupRepository
	membershipRepo community.GroupMembershipRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewJoinGroupHandler creates a new JoinGroupHandler with the given dependencies.
func NewJoinGroupHandler(
	groupRepo community.GroupRepository,
	membershipRepo community.GroupMembershipRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *JoinGroupHandler {
	return &JoinGroupHandler{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the join group use case.
//
// Process flow:
//  1. Load the group aggregate
//  2. Check if user is already a member
//  3. Verify group can accept new members (capacity check)
//  4. Check group type and create appropriate membership:
//     - Public: active membership (instant join)
//     - InviteOnly: requested membership (pending approval)
//     - Private: reject (cannot join without invitation)
//  5. Increment member count (only for active members)
//  6. Persist membership
//  7. Publish domain events
//
// Returns:
//   - *community.GroupMembership on successful join or request
//   - ErrAlreadyGroupMember if user is already a member
//   - ErrPrivateGroupNoAccess if group is private
//   - ErrMemberLimitReached if group is at capacity
//
//nolint:funlen // Sequential validation and join logic.
func (h *JoinGroupHandler) Handle(ctx context.Context, cmd JoinGroupCommand) (*community.GroupMembership, error) {
	// 1. Load the group aggregate
	group, err := h.groupRepo.FindByID(ctx, cmd.GroupID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Msg("group not found during join attempt")
		return nil, fmt.Errorf("find group: %w", err)
	}

	// 2. Check if user is already a member
	existingMembership, err := h.membershipRepo.FindByGroupAndUser(ctx, cmd.GroupID, cmd.UserID)
	if err != nil && !errors.Is(err, community.ErrMembershipNotFound) {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("user_id", cmd.UserID.String()).
			Msg("failed to check existing membership")
		return nil, fmt.Errorf("check existing membership: %w", err)
	}

	if existingMembership != nil {
		// User already has a membership (could be active, banned, or requested)
		if existingMembership.IsBanned() {
			h.logger.Warn().
				Str("group_id", cmd.GroupID.String()).
				Str("user_id", cmd.UserID.String()).
				Msg("banned user attempted to join group")
			return nil, community.ErrMemberBanned
		}

		if existingMembership.IsActive() || existingMembership.Status() == community.MemberStatusRequested {
			h.logger.Debug().
				Str("group_id", cmd.GroupID.String()).
				Str("user_id", cmd.UserID.String()).
				Msg("user already member or has pending request")
			return nil, community.ErrAlreadyGroupMember
		}
	}

	// 3. Verify group can accept new members (capacity check)
	if !group.CanAcceptNewMembers() {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("user_id", cmd.UserID.String()).
			Int("member_count", group.MemberCount()).
			Int("max_members", group.Settings().MaxMembers()).
			Msg("group at capacity during join attempt")
		return nil, community.ErrMemberLimitReached
	}

	// 4. Check group type and create appropriate membership
	var membership *community.GroupMembership

	switch {
	case group.IsPrivate():
		// Private groups cannot be joined without invitation
		h.logger.Debug().
			Str("group_id", cmd.GroupID.String()).
			Str("user_id", cmd.UserID.String()).
			Msg("attempted to join private group")
		return nil, community.ErrPrivateGroupNoAccess

	case group.IsPublic():
		// Public groups allow instant join
		membership, err = community.NewGroupMembership(cmd.GroupID, cmd.UserID, community.GroupRoleMember)
		if err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Str("user_id", cmd.UserID.String()).
				Msg("failed to create membership for public group")
			return nil, fmt.Errorf("create membership: %w", err)
		}

		// Increment member count for active members
		group.IncrementMemberCount()

	case group.IsInviteOnly():
		// Invite-only groups require a join request
		membership, err = community.NewRequestedMembership(cmd.GroupID, cmd.UserID)
		if err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Str("user_id", cmd.UserID.String()).
				Msg("failed to create requested membership")
			return nil, fmt.Errorf("create requested membership: %w", err)
		}
		// Do NOT increment member count for requested members

	default:
		// Should never happen if group types are valid
		h.logger.Error().
			Str("group_id", cmd.GroupID.String()).
			Str("group_type", group.GroupType().String()).
			Msg("unknown group type during join")
		return nil, fmt.Errorf("unknown group type: %s", group.GroupType())
	}

	// 5. Save group (if member count was incremented)
	if group.IsPublic() {
		if err := h.groupRepo.Save(ctx, group); err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Msg("failed to update group member count")
			return nil, fmt.Errorf("save group: %w", err)
		}
	}

	// 6. Persist membership
	if err := h.membershipRepo.Save(ctx, membership); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("user_id", cmd.UserID.String()).
			Msg("failed to save membership")
		return nil, fmt.Errorf("save membership: %w", err)
	}

	// 7. Publish domain events AFTER successful save
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

	// Publish membership events
	for _, event := range membership.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish membership domain event")
		}
	}
	membership.ClearEvents()

	h.logger.Info().
		Str("group_id", cmd.GroupID.String()).
		Str("user_id", cmd.UserID.String()).
		Str("status", membership.Status().String()).
		Msg("user joined group or requested membership")

	return membership, nil
}
