package gallery

import (
	"fmt"
	"strings"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

const (
	// MaxCommentLength is the maximum allowed length for a comment.
	MaxCommentLength = 1000
)

// Comment is an entity representing a user's comment on an image.
// Comments are immutable once created (no editing, only deletion).
type Comment struct {
	id        CommentID
	imageID   ImageID
	userID    identity.UserID
	content   string
	createdAt time.Time
	events    []shared.DomainEvent
}

// NewComment creates a new Comment with validation.
// Returns an error if the content is invalid.
func NewComment(imageID ImageID, userID identity.UserID, content string) (*Comment, error) {
	if imageID.IsZero() {
		return nil, fmt.Errorf("%w: image ID is required", shared.ErrInvalidInput)
	}
	if userID.IsZero() {
		return nil, fmt.Errorf("%w: user ID is required", shared.ErrInvalidInput)
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrCommentRequired
	}
	if len(content) > MaxCommentLength {
		return nil, fmt.Errorf("%w: got %d characters", ErrCommentTooLong, len(content))
	}

	now := time.Now().UTC()
	comment := &Comment{
		id:        NewCommentID(),
		imageID:   imageID,
		userID:    userID,
		content:   content,
		createdAt: now,
		events:    []shared.DomainEvent{},
	}

	comment.addEvent(&CommentAdded{
		BaseEvent: shared.NewBaseEvent("gallery.comment.added", comment.id.String()),
		CommentID: comment.id,
		ImageID:   comment.imageID,
		UserID:    comment.userID,
		Content:   comment.content,
	})

	return comment, nil
}

// ReconstructComment reconstitutes a Comment from persistence without validation or events.
// Use this only when loading from the database.
func ReconstructComment(
	id CommentID,
	imageID ImageID,
	userID identity.UserID,
	content string,
	createdAt time.Time,
) *Comment {
	return &Comment{
		id:        id,
		imageID:   imageID,
		userID:    userID,
		content:   content,
		createdAt: createdAt,
		events:    []shared.DomainEvent{},
	}
}

// Getters.

// ID returns the unique identifier of the comment.
func (c *Comment) ID() CommentID {
	return c.id
}

// ImageID returns the ID of the image this comment is on.
func (c *Comment) ImageID() ImageID {
	return c.imageID
}

// UserID returns the ID of the user who created the comment.
func (c *Comment) UserID() identity.UserID {
	return c.userID
}

// Content returns the comment content.
func (c *Comment) Content() string {
	return c.content
}

// CreatedAt returns when the comment was created.
func (c *Comment) CreatedAt() time.Time {
	return c.createdAt
}

// Events returns the domain events that have occurred.
func (c *Comment) Events() []shared.DomainEvent {
	return c.events
}

// ClearEvents clears all pending domain events.
func (c *Comment) ClearEvents() {
	c.events = []shared.DomainEvent{}
}

// Helper Methods.

// IsAuthoredBy returns true if the comment was authored by the given user.
func (c *Comment) IsAuthoredBy(userID identity.UserID) bool {
	return c.userID.Equals(userID)
}

// addEvent appends a domain event to the events slice.
func (c *Comment) addEvent(event shared.DomainEvent) {
	c.events = append(c.events, event)
}
