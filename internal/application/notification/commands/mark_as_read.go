package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

// MarkNotificationsReadCommand represents a request to mark notifications as read.
type MarkNotificationsReadCommand struct {
	UserID          string   // Authenticated user ID (for authorization)
	NotificationIDs []string // Specific notification IDs to mark as read (optional)
	MarkAllAsRead   bool     // If true, mark all notifications for user as read
}

// MarkNotificationsReadHandler handles marking notifications as read.
type MarkNotificationsReadHandler struct {
	notifications notification.NotificationRepository
	logger        zerolog.Logger
}

// NewMarkNotificationsReadHandler creates a new MarkNotificationsReadHandler.
func NewMarkNotificationsReadHandler(
	notifications notification.NotificationRepository,
	logger zerolog.Logger,
) *MarkNotificationsReadHandler {
	return &MarkNotificationsReadHandler{
		notifications: notifications,
		logger:        logger,
	}
}

// Handle executes the mark notifications read command.
//
// Process flow:
// 1. Parse and validate user ID
// 2. If MarkAllAsRead is true, mark all user's notifications as read
// 3. Otherwise, mark specific notifications as read
// 4. Log successful operation
//
// Authorization:
// - Users can only mark their own notifications as read
// - Ownership is verified by the handler that calls this (HTTP layer)
//
// Returns:
// - nil on success
// - Error if operation fails
func (h *MarkNotificationsReadHandler) Handle(
	ctx context.Context,
	cmd MarkNotificationsReadCommand,
) error {
	// 1. Parse and validate user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	// 2. If MarkAllAsRead is true, mark all notifications
	if cmd.MarkAllAsRead {
		if err := h.notifications.MarkAllRead(ctx, userID); err != nil {
			return fmt.Errorf("mark all notifications as read: %w", err)
		}

		h.logger.Info().
			Str("user_id", userID.String()).
			Msg("all notifications marked as read")

		return nil
	}

	// 3. Otherwise, mark specific notifications as read
	if len(cmd.NotificationIDs) == 0 {
		return fmt.Errorf("no notification IDs provided")
	}

	ids := make([]notification.NotificationID, 0, len(cmd.NotificationIDs))
	for _, idStr := range cmd.NotificationIDs {
		notifID, err := notification.ParseNotificationID(idStr)
		if err != nil {
			return fmt.Errorf("invalid notification id %s: %w", idStr, err)
		}
		ids = append(ids, notifID)
	}

	if err := h.notifications.MarkManyAsRead(ctx, ids, userID); err != nil {
		return fmt.Errorf("mark notifications as read: %w", err)
	}

	h.logger.Info().
		Str("user_id", userID.String()).
		Int("count", len(cmd.NotificationIDs)).
		Msg("notifications marked as read")

	return nil
}
