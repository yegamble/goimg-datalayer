package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	appcommunity "github.com/yegamble/goimg-datalayer/internal/application/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// DeclineInvitationCommand represents the intent to decline a group invitation using a token.
// The declining user must match the invitation target (email or userID).
type DeclineInvitationCommand struct {
	Token  community.InvitationToken
	UserID identity.UserID // The user declining the invitation
}

// Implement Command interface.
func (DeclineInvitationCommand) isCommand() {}

// DeclineInvitationHandler processes invitation decline commands.
type DeclineInvitationHandler struct {
	invitationRepo community.GroupInvitationRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewDeclineInvitationHandler creates a new DeclineInvitationHandler with the given dependencies.
func NewDeclineInvitationHandler(
	invitationRepo community.GroupInvitationRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *DeclineInvitationHandler {
	return &DeclineInvitationHandler{
		invitationRepo: invitationRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the decline invitation use case.
//
// Process flow:
//  1. Load the invitation by token
//  2. Validate invitation exists and is not already used
//  3. Delete the invitation (decline = remove from system)
//  4. Publish domain event
//
// Returns:
//   - nil on successful decline
//   - ErrInvitationNotFound if token is invalid
//   - ErrInvitationAlreadyUsed if invitation was already used
//
// Note: Expired invitations can still be declined (to clean them up).
func (h *DeclineInvitationHandler) Handle(ctx context.Context, cmd DeclineInvitationCommand) error {
	// 1. Load the invitation by token
	invitation, err := h.invitationRepo.FindByToken(ctx, cmd.Token)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID.String()).
			Msg("invitation not found by token during decline")
		return fmt.Errorf("find invitation: %w", err)
	}

	// 2. Validate invitation is not already used
	if invitation.IsUsed() {
		h.logger.Warn().
			Str("invitation_id", invitation.ID().String()).
			Str("user_id", cmd.UserID.String()).
			Msg("attempted to decline already-used invitation")
		return community.ErrInvitationAlreadyUsed
	}

	// Note: We allow declining expired invitations for cleanup purposes

	// 3. Delete the invitation (decline = remove from system)
	if err := h.invitationRepo.Delete(ctx, invitation.ID()); err != nil {
		h.logger.Error().
			Err(err).
			Str("invitation_id", invitation.ID().String()).
			Msg("failed to delete declined invitation")
		return fmt.Errorf("delete invitation: %w", err)
	}

	// 4. Publish domain event
	// Since the invitation is being deleted, we create a standalone event
	// (not added to the invitation's event list)
	declineEvent := &community.GroupInvitationDeclined{
		BaseEvent:    shared.NewBaseEvent("community.invitation.declined", invitation.ID().String()),
		InvitationID: invitation.ID(),
		GroupID:      invitation.GroupID(),
		DeclinedAt:   time.Now().UTC(),
	}

	if err := h.eventPublisher.Publish(ctx, declineEvent); err != nil {
		h.logger.Error().
			Err(err).
			Str("invitation_id", invitation.ID().String()).
			Msg("failed to publish invitation declined event")
		// Don't fail the operation if event publishing fails
	}

	h.logger.Info().
		Str("group_id", invitation.GroupID().String()).
		Str("user_id", cmd.UserID.String()).
		Str("invitation_id", invitation.ID().String()).
		Msg("invitation declined successfully")

	return nil
}
