package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appmoderation "github.com/yegamble/goimg-datalayer/internal/application/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// UnbanUserCommand represents the intent to revoke a user's ban before its expiration.
// This manually lifts a ban that was previously issued.
type UnbanUserCommand struct {
	UserID    string // User to unban
	RevokedBy string // Admin/moderator revoking the ban
}

// Implement Command interface
func (UnbanUserCommand) isCommand() {}

// UnbanUserResult represents the result of successfully unbanning a user.
type UnbanUserResult struct {
	BanID  string
	UserID string
	Status string
}

// UnbanUserHandler processes user unban commands.
// It orchestrates the workflow of revoking an active ban.
type UnbanUserHandler struct {
	bans           moderation.BanRepository
	eventPublisher appmoderation.EventPublisher
	logger         *zerolog.Logger
}

// NewUnbanUserHandler creates a new UnbanUserHandler with the given dependencies.
func NewUnbanUserHandler(
	bans moderation.BanRepository,
	eventPublisher appmoderation.EventPublisher,
	logger *zerolog.Logger,
) *UnbanUserHandler {
	return &UnbanUserHandler{
		bans:           bans,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the unban user use case.
//
// Process flow:
//  1. Parse and validate user ID
//  2. Parse and validate revoked by ID
//  3. Find the user's most recent ban
//  4. Call domain method to revoke the ban
//  5. Persist updated ban
//  6. Publish domain events after successful save
//
// Returns:
//   - UnbanUserResult on successful ban revocation
//   - ErrBanNotFound if the user has no active ban
//   - ErrBanAlreadyRevoked if the ban is already revoked
//   - ErrBanExpired if the ban has naturally expired
func (h *UnbanUserHandler) Handle(ctx context.Context, cmd UnbanUserCommand) (*UnbanUserResult, error) {
	// 1. Parse and validate user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id during unban")
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Parse and validate revoked by ID
	revokedBy, err := identity.ParseUserID(cmd.RevokedBy)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("revoked_by", cmd.RevokedBy).
			Msg("invalid revoked by id during unban")
		return nil, fmt.Errorf("invalid revoked by id: %w", err)
	}

	// 3. Find the user's most recent ban
	ban, err := h.bans.FindByUserID(ctx, userID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", userID.String()).
			Msg("ban not found during unban")
		return nil, fmt.Errorf("find ban: %w", err)
	}

	// 4. Revoke via domain method
	if err := ban.Revoke(revokedBy); err != nil {
		h.logger.Debug().
			Err(err).
			Str("ban_id", ban.ID().String()).
			Str("user_id", userID.String()).
			Str("revoked_by", revokedBy.String()).
			Msg("failed to revoke ban")
		return nil, fmt.Errorf("revoke ban: %w", err)
	}

	// 5. Persist updated ban
	if err := h.bans.Save(ctx, ban); err != nil {
		h.logger.Error().
			Err(err).
			Str("ban_id", ban.ID().String()).
			Str("user_id", userID.String()).
			Msg("failed to save ban after revocation")
		return nil, fmt.Errorf("save ban: %w", err)
	}

	// 6. Publish domain events AFTER successful save
	for _, event := range ban.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("ban_id", ban.ID().String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after ban revocation")
		}
	}
	ban.ClearEvents()

	h.logger.Info().
		Str("ban_id", ban.ID().String()).
		Str("user_id", userID.String()).
		Str("revoked_by", revokedBy.String()).
		Msg("user unbanned successfully")

	return &UnbanUserResult{
		BanID:  ban.ID().String(),
		UserID: userID.String(),
		Status: "revoked",
	}, nil
}
