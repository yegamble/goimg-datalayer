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

// InviteToGroupCommand represents the intent to invite a user or email to join a group.
// Either email OR userID must be provided (but not both).
// The invitation token is cryptographically secure and expires in 7 days.
type InviteToGroupCommand struct {
	GroupID   community.GroupID
	InvitedBy identity.UserID
	Email     *string          // Email address for non-registered users
	UserID    *identity.UserID // User ID for existing users
}

// Implement Command interface.
func (InviteToGroupCommand) isCommand() {}

// InviteToGroupHandler processes group invitation commands.
type InviteToGroupHandler struct {
	groupRepo      community.GroupRepository
	membershipRepo community.GroupMembershipRepository
	invitationRepo community.GroupInvitationRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewInviteToGroupHandler creates a new InviteToGroupHandler with the given dependencies.
func NewInviteToGroupHandler(
	groupRepo community.GroupRepository,
	membershipRepo community.GroupMembershipRepository,
	invitationRepo community.GroupInvitationRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *InviteToGroupHandler {
	return &InviteToGroupHandler{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		invitationRepo: invitationRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the invite to group use case.
//
// Process flow:
//  1. Load the group aggregate
//  2. Verify inviter is a member with admin or owner role
//  3. Check if group allows member invites (or if inviter is admin+)
//  4. Verify the invitee is not already a member
//  5. Create secure invitation with 7-day expiry
//  6. Persist invitation
//  7. Publish domain events
//
// Returns:
//   - *community.GroupInvitation on successful invitation creation
//   - ErrInsufficientGroupRole if inviter lacks permissions
//   - ErrAlreadyGroupMember if invitee is already a member
//   - ErrInvalidInput if both email and userID are provided or neither
//
//nolint:funlen // Sequential validation and invitation logic.
func (h *InviteToGroupHandler) Handle(ctx context.Context, cmd InviteToGroupCommand) (*community.GroupInvitation, error) {
	// 1. Load the group aggregate
	group, err := h.groupRepo.FindByID(ctx, cmd.GroupID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Msg("group not found during invitation attempt")
		return nil, fmt.Errorf("find group: %w", err)
	}

	// 2. Verify inviter is a member with admin or owner role
	inviterMembership, err := h.membershipRepo.FindByGroupAndUser(ctx, cmd.GroupID, cmd.InvitedBy)
	if err != nil {
		if errors.Is(err, community.ErrMembershipNotFound) {
			h.logger.Warn().
				Str("group_id", cmd.GroupID.String()).
				Str("inviter_id", cmd.InvitedBy.String()).
				Msg("non-member attempted to invite to group")
			return nil, community.ErrNotGroupMember
		}
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("inviter_id", cmd.InvitedBy.String()).
			Msg("failed to check inviter membership")
		return nil, fmt.Errorf("check inviter membership: %w", err)
	}

	// Check if inviter has permission to invite
	isAdminOrOwner := inviterMembership.Role() == community.GroupRoleAdmin || inviterMembership.Role() == community.GroupRoleOwner

	if !isAdminOrOwner && !group.Settings().AllowMemberInvites() {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("inviter_id", cmd.InvitedBy.String()).
			Str("role", inviterMembership.Role().String()).
			Msg("member without invite permission attempted to invite")
		return nil, community.ErrInsufficientGroupRole
	}

	// 3. Validate that exactly one of email or userID is provided
	if cmd.Email == nil && cmd.UserID == nil {
		return nil, fmt.Errorf("either email or user ID must be provided")
	}
	if cmd.Email != nil && cmd.UserID != nil {
		return nil, fmt.Errorf("cannot specify both email and user ID")
	}

	// 4. If inviting by userID, verify the user is not already a member
	if cmd.UserID != nil {
		existingMembership, err := h.membershipRepo.FindByGroupAndUser(ctx, cmd.GroupID, *cmd.UserID)
		if err != nil && !errors.Is(err, community.ErrMembershipNotFound) {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Str("user_id", cmd.UserID.String()).
				Msg("failed to check if invitee is already a member")
			return nil, fmt.Errorf("check invitee membership: %w", err)
		}

		if existingMembership != nil {
			h.logger.Debug().
				Str("group_id", cmd.GroupID.String()).
				Str("user_id", cmd.UserID.String()).
				Msg("attempted to invite user who is already a member")
			return nil, community.ErrAlreadyGroupMember
		}
	}

	// 5. Create secure invitation with 7-day expiry
	invitation, err := community.NewGroupInvitation(cmd.GroupID, cmd.InvitedBy, cmd.Email, cmd.UserID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Msg("failed to create group invitation")
		return nil, fmt.Errorf("create invitation: %w", err)
	}

	// 6. Persist invitation
	if err := h.invitationRepo.Save(ctx, invitation); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("invitation_id", invitation.ID().String()).
			Msg("failed to save invitation")
		return nil, fmt.Errorf("save invitation: %w", err)
	}

	// 7. Publish domain events AFTER successful save
	for _, event := range invitation.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish invitation domain event")
		}
	}
	invitation.ClearEvents()

	h.logger.Info().
		Str("group_id", cmd.GroupID.String()).
		Str("inviter_id", cmd.InvitedBy.String()).
		Str("invitation_id", invitation.ID().String()).
		Msg("group invitation created successfully")

	return invitation, nil
}
