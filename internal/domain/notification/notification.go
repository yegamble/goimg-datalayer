package notification

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// Notification represents an in-app notification for a user.
// It is an entity in the Notification bounded context.
//
// Business Rules:
// - Every notification must have a recipient and title
// - Notifications can only be marked read, never unread
// - Metadata is flexible (JSON) to support different notification types
// - Notifications are immutable after creation (except read status)
type Notification struct {
	id          NotificationID
	recipientID identity.UserID
	notifType   NotificationType
	title       string
	body        string
	metadata    map[string]string // Flexible payload (user IDs, image IDs, etc.)
	readAt      *time.Time
	createdAt   time.Time
	events      []shared.DomainEvent
}

// NewNotification creates a new notification with validation.
// Emits no domain events (notifications are passive records).
func NewNotification(
	recipientID identity.UserID,
	notifType NotificationType,
	title, body string,
	metadata map[string]string,
) (*Notification, error) {
	if recipientID.IsZero() {
		return nil, ErrRecipientRequired
	}

	if title == "" {
		return nil, ErrTitleRequired
	}

	if !notifType.IsValid() {
		return nil, ErrInvalidNotificationType
	}

	// Initialize metadata if nil
	if metadata == nil {
		metadata = make(map[string]string)
	}

	now := time.Now().UTC()
	return &Notification{
		id:          NewNotificationID(),
		recipientID: recipientID,
		notifType:   notifType,
		title:       title,
		body:        body,
		metadata:    metadata,
		readAt:      nil,
		createdAt:   now,
		events:      []shared.DomainEvent{},
	}, nil
}

// ReconstructNotification reconstitutes a Notification from persistence without validation or events.
// This should only be used by the repository layer when loading from storage.
func ReconstructNotification(
	id NotificationID,
	recipientID identity.UserID,
	notifType NotificationType,
	title, body string,
	metadata map[string]string,
	readAt *time.Time,
	createdAt time.Time,
) *Notification {
	if metadata == nil {
		metadata = make(map[string]string)
	}

	return &Notification{
		id:          id,
		recipientID: recipientID,
		notifType:   notifType,
		title:       title,
		body:        body,
		metadata:    metadata,
		readAt:      readAt,
		createdAt:   createdAt,
		events:      []shared.DomainEvent{},
	}
}

// ID returns the notification's unique identifier.
func (n *Notification) ID() NotificationID {
	return n.id
}

// RecipientID returns the ID of the user who receives this notification.
func (n *Notification) RecipientID() identity.UserID {
	return n.recipientID
}

// Type returns the notification type.
func (n *Notification) Type() NotificationType {
	return n.notifType
}

// Title returns the notification title.
func (n *Notification) Title() string {
	return n.title
}

// Body returns the notification body text.
func (n *Notification) Body() string {
	return n.body
}

// Metadata returns the notification metadata (flexible key-value pairs).
func (n *Notification) Metadata() map[string]string {
	return n.metadata
}

// ReadAt returns when the notification was read (nil if unread).
func (n *Notification) ReadAt() *time.Time {
	return n.readAt
}

// CreatedAt returns when the notification was created.
func (n *Notification) CreatedAt() time.Time {
	return n.createdAt
}

// IsRead returns true if the notification has been read.
func (n *Notification) IsRead() bool {
	return n.readAt != nil
}

// Events returns the domain events that have occurred on this entity.
func (n *Notification) Events() []shared.DomainEvent {
	return n.events
}

// ClearEvents clears all domain events from this entity.
func (n *Notification) ClearEvents() {
	n.events = []shared.DomainEvent{}
}

// MarkRead marks the notification as read with the current timestamp.
// This operation is idempotent - marking an already-read notification does nothing.
func (n *Notification) MarkRead() error {
	if n.readAt != nil {
		return nil // Already read, no-op
	}

	now := time.Now().UTC()
	n.readAt = &now
	return nil
}

// GetMetadata retrieves a metadata value by key.
// Returns empty string if key doesn't exist.
func (n *Notification) GetMetadata(key string) string {
	if n.metadata == nil {
		return ""
	}
	return n.metadata[key]
}

// addEvent adds a domain event to the entity's event list.
func (n *Notification) addEvent(event shared.DomainEvent) {
	n.events = append(n.events, event)
}

// Validate ensures the notification is in a valid state.
// This is primarily for testing and defensive programming.
func (n *Notification) Validate() error {
	if n.recipientID.IsZero() {
		return fmt.Errorf("validation: %w", ErrRecipientRequired)
	}

	if n.title == "" {
		return fmt.Errorf("validation: %w", ErrTitleRequired)
	}

	if !n.notifType.IsValid() {
		return fmt.Errorf("validation: %w", ErrInvalidNotificationType)
	}

	return nil
}
