package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

// GetNotificationsQuery represents a request to retrieve notifications for a user.
type GetNotificationsQuery struct {
	UserID     string // User whose notifications to retrieve
	UnreadOnly bool   // If true, only return unread notifications
	Limit      int    // Maximum number of notifications to return
	Offset     int    // Number of notifications to skip for pagination
}

// GetNotificationsResult represents the result of a get notifications query.
type GetNotificationsResult struct {
	Notifications []*notification.Notification
	TotalUnread   int
}

// GetNotificationsHandler handles queries for retrieving user notifications.
type GetNotificationsHandler struct {
	notifications notification.NotificationRepository
}

// NewGetNotificationsHandler creates a new GetNotificationsHandler.
func NewGetNotificationsHandler(
	notifications notification.NotificationRepository,
) *GetNotificationsHandler {
	return &GetNotificationsHandler{
		notifications: notifications,
	}
}

// Handle executes the get notifications query.
//
// Process flow:
// 1. Parse and validate user ID
// 2. Retrieve notifications based on query parameters
// 3. Get unread count for badge display
// 4. Return results
//
// Returns:
// - GetNotificationsResult with notifications and unread count
// - Error if user ID is invalid or query fails
func (h *GetNotificationsHandler) Handle(
	ctx context.Context,
	q GetNotificationsQuery,
) (*GetNotificationsResult, error) {
	// 1. Parse and validate user ID
	userID, err := identity.ParseUserID(q.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Retrieve notifications based on query parameters
	var notifications []*notification.Notification

	if q.UnreadOnly {
		// Get only unread notifications
		notifications, err = h.notifications.FindUnreadByRecipient(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("find unread notifications: %w", err)
		}
	} else {
		// Get all notifications with pagination
		notifications, err = h.notifications.FindByRecipient(ctx, userID, q.Limit, q.Offset)
		if err != nil {
			return nil, fmt.Errorf("find notifications: %w", err)
		}
	}

	// 3. Get unread count for badge display
	unreadCount, err := h.notifications.CountUnread(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count unread notifications: %w", err)
	}

	return &GetNotificationsResult{
		Notifications: notifications,
		TotalUnread:   unreadCount,
	}, nil
}
