package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// CleanupExpiredGuestsCommand represents the intent to delete expired guest accounts.
// This command is typically executed by a background job (Asynq) on a daily schedule.
type CleanupExpiredGuestsCommand struct {
	// No parameters needed - cleans up all expired guests
}

// CleanupExpiredGuestsHandler processes guest cleanup commands.
// It deletes guest accounts that have passed their expiration date.
type CleanupExpiredGuestsHandler struct {
	users  identity.UserRepository
	logger *zerolog.Logger
}

// NewCleanupExpiredGuestsHandler creates a new CleanupExpiredGuestsHandler.
func NewCleanupExpiredGuestsHandler(
	users identity.UserRepository,
	logger *zerolog.Logger,
) *CleanupExpiredGuestsHandler {
	return &CleanupExpiredGuestsHandler{
		users:  users,
		logger: logger,
	}
}

// Handle executes the expired guest cleanup use case.
//
// Process flow:
//  1. Query all guest users with expires_at <= NOW()
//  2. Delete each expired guest user
//  3. Log results for monitoring
//
// Returns:
//   - CleanupResult with count of deleted users
//   - Error if query or deletion fails
func (h *CleanupExpiredGuestsHandler) Handle(
	ctx context.Context,
	cmd CleanupExpiredGuestsCommand,
) (*CleanupResult, error) {
	h.logger.Info().Msg("starting expired guest cleanup job")

	startTime := time.Now()
	now := time.Now().UTC()

	deletedCount := 0
	errorCount := 0

	// Query all guest users with expires_at <= NOW()
	// Process in batches to avoid loading too many records at once
	const batchSize = 1000

	users, err := h.users.FindExpiredGuests(ctx, now, batchSize)
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to find expired guest users")
		return nil, fmt.Errorf("find expired guests: %w", err)
	}

	h.logger.Debug().
		Int("expired_guest_count", len(users)).
		Msg("found expired guest users")

	// Delete expired guests
	for _, user := range users {
		if err := h.users.Delete(ctx, user.ID()); err != nil {
			h.logger.Error().
				Err(err).
				Str("user_id", user.ID().String()).
				Msg("failed to delete expired guest user")
			errorCount++
			continue
		}

		h.logger.Debug().
			Str("user_id", user.ID().String()).
			Time("expired_at", *user.ExpiresAt()).
			Msg("deleted expired guest user")
		deletedCount++
	}

	duration := time.Since(startTime)

	h.logger.Info().
		Int("deleted_count", deletedCount).
		Int("error_count", errorCount).
		Dur("duration_ms", duration).
		Msg("expired guest cleanup job completed")

	return &CleanupResult{
		DeletedCount: deletedCount,
		ErrorCount:   errorCount,
		Duration:     duration,
	}, nil
}

// CleanupResult contains the results of the cleanup operation.
type CleanupResult struct {
	DeletedCount int           // Number of successfully deleted users
	ErrorCount   int           // Number of deletion failures
	Duration     time.Duration // Time taken to complete cleanup
}
