package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

// GetUnreadCountQuery represents a request to count unread notifications for a user.
type GetUnreadCountQuery struct {
	UserID string // User whose unread count to retrieve
}

// GetUnreadCountResult represents the result of a get unread count query.
type GetUnreadCountResult struct {
	Count int
}

// GetUnreadCountHandler handles queries for counting unread notifications.
type GetUnreadCountHandler struct {
	notifications notification.NotificationRepository
}

// NewGetUnreadCountHandler creates a new GetUnreadCountHandler.
func NewGetUnreadCountHandler(
	notifications notification.NotificationRepository,
) *GetUnreadCountHandler {
	return &GetUnreadCountHandler{
		notifications: notifications,
	}
}

// Handle executes the get unread count query.
//
// Process flow:
// 1. Parse and validate user ID
// 2. Count unread notifications
// 3. Return count
//
// Returns:
// - GetUnreadCountResult with the count
// - Error if user ID is invalid or query fails
func (h *GetUnreadCountHandler) Handle(
	ctx context.Context,
	q GetUnreadCountQuery,
) (*GetUnreadCountResult, error) {
	// 1. Parse and validate user ID
	userID, err := identity.ParseUserID(q.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Count unread notifications
	count, err := h.notifications.CountUnread(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count unread notifications: %w", err)
	}

	return &GetUnreadCountResult{
		Count: count,
	}, nil
}
