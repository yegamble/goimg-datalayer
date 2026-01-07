package notification

import (
	"context"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// NotificationRepository defines the persistence interface for notifications.
// Implementations live in the infrastructure layer.
type NotificationRepository interface {
	// Save persists a notification to storage.
	Save(ctx context.Context, n *Notification) error

	// FindByID retrieves a notification by its ID.
	// Returns ErrNotificationNotFound if not found.
	FindByID(ctx context.Context, id NotificationID) (*Notification, error)

	// FindByRecipient retrieves notifications for a specific user with pagination.
	// Results are ordered by created_at DESC (newest first).
	FindByRecipient(ctx context.Context, recipientID identity.UserID, limit, offset int) ([]*Notification, error)

	// FindUnreadByRecipient retrieves all unread notifications for a user.
	// Results are ordered by created_at DESC (newest first).
	FindUnreadByRecipient(ctx context.Context, recipientID identity.UserID) ([]*Notification, error)

	// CountUnread returns the number of unread notifications for a user.
	CountUnread(ctx context.Context, recipientID identity.UserID) (int, error)

	// MarkAsRead marks a specific notification as read.
	MarkAsRead(ctx context.Context, id NotificationID) error

	// MarkAllRead marks all notifications for a user as read.
	MarkAllRead(ctx context.Context, recipientID identity.UserID) error

	// Delete removes a notification from storage.
	Delete(ctx context.Context, id NotificationID) error

	// DeleteOlderThan removes notifications older than the specified time.
	// This is used for cleanup/retention policies.
	DeleteOlderThan(ctx context.Context, before time.Time) error
}
