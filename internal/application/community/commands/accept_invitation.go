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

// AcceptInvitationCommand represents the intent to accept a group invitation using a token.
// The accepting user must match the invitation target (email or userID).
type AcceptInvitationCommand struct {
	Token  community.InvitationToken
	UserID identity.UserID // The user accepting the invitation
}

// Implement Command interface.
func (AcceptInvitationCommand) isCommand() {}

// AcceptInvitationHandler processes invitation acceptance commands.
type AcceptInvitationHandler struct {
	groupRepo      community.GroupRepository
	membershipRepo community.GroupMembershipRepository
	invitationRepo community.GroupInvitationRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewAcceptInvitationHandler creates a new AcceptInvitationHandler with the given dependencies.
func NewAcceptInvitationHandler(
	groupRepo community.GroupRepository,
	membershipRepo community.GroupMembershipRepository,
	invitationRepo community.GroupInvitationRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *AcceptInvitationHandler {
	return &AcceptInvitationHandler{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		invitationRepo: invitationRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the accept invitation use case.
//
// Process flow:
//  1. Load the invitation by token
//  2. Validate invitation is not expired or already used
//  3. Load the group aggregate
//  4. Verify group can accept new members
//  5. Verify user is not already a member
//  6. Mark invitation as accepted
//  7. Create active group membership
//  8. Increment group member count
//  9. Persist all changes
//  10. Publish domain events
//
// Returns:
//   - *community.GroupMembership on successful acceptance
//   - ErrInvitationNotFound if token is invalid
//   - ErrInvitationExpired if invitation has expired
//   - ErrInvitationAlreadyUsed if invitation was already used
//   - ErrAlreadyGroupMember if user is already a member
//   - ErrMemberLimitReached if group is at capacity
//
//nolint:funlen // Sequential validation and acceptance logic.
func (h *AcceptInvitationHandler) Handle(ctx context.Context, cmd AcceptInvitationCommand) (*community.GroupMembership, error) {
	// 1. Load the invitation by token
	invitation, err := h.invitationRepo.FindByToken(ctx, cmd.Token)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID.String()).
			Msg("invitation not found by token")
		return nil, fmt.Errorf("find invitation: %w", err)
	}

	// 2. Validate invitation is not expired or already used
	if err := invitation.Accept(); err != nil {
		h.logger.Warn().
			Err(err).
			Str("invitation_id", invitation.ID().String()).
			Str("user_id", cmd.UserID.String()).
			Msg("cannot accept invitation")
		return nil, err
	}

	// 3. Load the group aggregate
	group, err := h.groupRepo.FindByID(ctx, invitation.GroupID())
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", invitation.GroupID().String()).
			Msg("group not found for invitation")
		return nil, fmt.Errorf("find group: %w", err)
	}

	// 4. Verify group can accept new members
	if !group.CanAcceptNewMembers() {
		h.logger.Warn().
			Str("group_id", invitation.GroupID().String()).
			Str("user_id", cmd.UserID.String()).
			Int("member_count", group.MemberCount()).
			Int("max_members", group.Settings().MaxMembers()).
			Msg("group at capacity during invitation acceptance")
		return nil, community.ErrMemberLimitReached
	}

	// 5. Verify user is not already a member
	existingMembership, err := h.membershipRepo.FindByGroupAndUser(ctx, invitation.GroupID(), cmd.UserID)
	if err != nil && !errors.Is(err, community.ErrMembershipNotFound) {
		h.logger.Error().
			Err(err).
			Str("group_id", invitation.GroupID().String()).
			Str("user_id", cmd.UserID.String()).
			Msg("failed to check existing membership")
		return nil, fmt.Errorf("check existing membership: %w", err)
	}

	if existingMembership != nil {
		if existingMembership.IsBanned() {
			h.logger.Warn().
				Str("group_id", invitation.GroupID().String()).
				Str("user_id", cmd.UserID.String()).
				Msg("banned user attempted to accept invitation")
			return nil, community.ErrMemberBanned
		}

		if existingMembership.IsActive() {
			h.logger.Debug().
				Str("group_id", invitation.GroupID().String()).
				Str("user_id", cmd.UserID.String()).
				Msg("user is already an active member")
			return nil, community.ErrAlreadyGroupMember
		}
	}

	// 6. Save the accepted invitation (marks used_at)
	if err := h.invitationRepo.Save(ctx, invitation); err != nil {
		h.logger.Error().
			Err(err).
			Str("invitation_id", invitation.ID().String()).
			Msg("failed to save accepted invitation")
		return nil, fmt.Errorf("save invitation: %w", err)
	}

	// 7. Create active group membership
	membership, err := community.NewGroupMembership(invitation.GroupID(), cmd.UserID, community.GroupRoleMember)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", invitation.GroupID().String()).
			Str("user_id", cmd.UserID.String()).
			Msg("failed to create membership")
		return nil, fmt.Errorf("create membership: %w", err)
	}

	// 8. Increment group member count
	group.IncrementMemberCount()

	// 9. Persist all changes
	if err := h.groupRepo.Save(ctx, group); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", invitation.GroupID().String()).
			Msg("failed to update group member count")
		return nil, fmt.Errorf("save group: %w", err)
	}

	if err := h.membershipRepo.Save(ctx, membership); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", invitation.GroupID().String()).
			Str("user_id", cmd.UserID.String()).
			Msg("failed to save membership")
		return nil, fmt.Errorf("save membership: %w", err)
	}

	// 10. Publish domain events AFTER successful save
	// Publish invitation events
	for _, event := range invitation.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("event_type", event.EventType()).
				Msg("failed to publish invitation event")
		}
	}
	invitation.ClearEvents()

	// Publish group events
	for _, event := range group.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("event_type", event.EventType()).
				Msg("failed to publish group event")
		}
	}
	group.ClearEvents()

	// Publish membership events
	for _, event := range membership.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("event_type", event.EventType()).
				Msg("failed to publish membership event")
		}
	}
	membership.ClearEvents()

	h.logger.Info().
		Str("group_id", invitation.GroupID().String()).
		Str("user_id", cmd.UserID.String()).
		Str("invitation_id", invitation.ID().String()).
		Msg("invitation accepted and membership created")

	return membership, nil
}
