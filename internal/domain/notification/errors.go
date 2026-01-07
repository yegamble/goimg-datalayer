package notification

import "errors"

var (
	// ErrNotificationNotFound is returned when a notification cannot be found.
	ErrNotificationNotFound = errors.New("notification not found")

	// ErrRecipientRequired is returned when trying to create a notification without a recipient.
	ErrRecipientRequired = errors.New("recipient is required")

	// ErrTitleRequired is returned when trying to create a notification without a title.
	ErrTitleRequired = errors.New("title is required")

	// ErrInvalidNotificationType is returned when an invalid notification type is provided.
	ErrInvalidNotificationType = errors.New("invalid notification type")

	// ErrNotificationAlreadyRead is returned when trying to mark an already-read notification as read.
	ErrNotificationAlreadyRead = errors.New("notification already read")
)
