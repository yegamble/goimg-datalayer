package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	appmoderation "github.com/yegamble/goimg-datalayer/internal/application/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// BanUserCommand represents the intent to ban a user from the platform.
// The ban can be temporary (with duration) or permanent (nil duration).
type BanUserCommand struct {
	UserID        string // User to ban
	BannedBy      string // Admin/moderator issuing the ban
	Reason        string // Reason for the ban
	DurationHours *int   // Optional: Duration in hours (nil = permanent)
}

// Implement Command interface
func (BanUserCommand) isCommand() {}

// BanUserResult represents the result of successfully banning a user.
type BanUserResult struct {
	BanID       string
	UserID      string
	IsPermanent bool
	ExpiresAt   *time.Time
}

// BanUserHandler processes user ban commands.
// It orchestrates the workflow of creating a ban and publishing events.
type BanUserHandler struct {
	bans           moderation.BanRepository
	eventPublisher appmoderation.EventPublisher
	logger         *zerolog.Logger
}

// NewBanUserHandler creates a new BanUserHandler with the given dependencies.
func NewBanUserHandler(
	bans moderation.BanRepository,
	eventPublisher appmoderation.EventPublisher,
	logger *zerolog.Logger,
) *BanUserHandler {
	return &BanUserHandler{
		bans:           bans,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the ban user use case.
//
// Process flow:
//  1. Parse and validate user ID
//  2. Parse and validate banned by ID
//  3. Convert duration hours to time.Duration (if provided)
//  4. Create Ban aggregate via domain factory
//  5. Persist ban via repository
//  6. Publish domain events after successful save
//
// Returns:
//   - BanUserResult on successful ban creation
//   - Validation errors from domain value objects
//   - ErrReasonRequired if reason is empty
//   - ErrReasonTooLong if reason exceeds max length
func (h *BanUserHandler) Handle(ctx context.Context, cmd BanUserCommand) (*BanUserResult, error) {
	// 1. Parse and validate user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id during ban")
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Parse and validate banned by ID
	bannedBy, err := identity.ParseUserID(cmd.BannedBy)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("banned_by", cmd.BannedBy).
			Msg("invalid banned by id during ban")
		return nil, fmt.Errorf("invalid banned by id: %w", err)
	}

	// 3. Convert duration hours to time.Duration
	var duration *time.Duration
	if cmd.DurationHours != nil {
		d := time.Duration(*cmd.DurationHours) * time.Hour
		duration = &d
	}

	// 4. Create Ban aggregate via domain factory
	ban, err := moderation.NewBan(userID, bannedBy, cmd.Reason, duration)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", userID.String()).
			Str("banned_by", bannedBy.String()).
			Msg("failed to create ban aggregate")
		return nil, fmt.Errorf("create ban: %w", err)
	}

	// 5. Persist ban to repository
	if err := h.bans.Save(ctx, ban); err != nil {
		h.logger.Error().
			Err(err).
			Str("ban_id", ban.ID().String()).
			Str("user_id", userID.String()).
			Msg("failed to save ban")
		return nil, fmt.Errorf("save ban: %w", err)
	}

	// 6. Publish domain events AFTER successful save
	for _, event := range ban.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("ban_id", ban.ID().String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after ban creation")
		}
	}
	ban.ClearEvents()

	h.logger.Info().
		Str("ban_id", ban.ID().String()).
		Str("user_id", userID.String()).
		Str("banned_by", bannedBy.String()).
		Bool("is_permanent", ban.IsPermanent()).
		Msg("user banned successfully")

	return &BanUserResult{
		BanID:       ban.ID().String(),
		UserID:      userID.String(),
		IsPermanent: ban.IsPermanent(),
		ExpiresAt:   ban.ExpiresAt(),
	}, nil
}
