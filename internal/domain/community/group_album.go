package community

import (
	"fmt"
	"strings"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

const (
	// MaxAlbumTitleLength is the maximum length for group album titles.
	MaxAlbumTitleLength = 255

	// MaxAlbumDescriptionLength is the maximum length for group album descriptions.
	MaxAlbumDescriptionLength = 2000
)

// GroupAlbum is an entity representing an album within a group.
// Albums organize images into curated collections within a group's image pool.
// This entity is owned by the Group aggregate but can be queried independently.
type GroupAlbum struct {
	id           GroupAlbumID
	groupID      GroupID
	createdBy    identity.UserID
	title        string
	description  string
	coverImageID *gallery.ImageID
	imageCount   int
	isPublic     bool
	createdAt    time.Time
	updatedAt    time.Time
	events       []shared.DomainEvent
}

// NewGroupAlbum creates a new GroupAlbum with the given group, creator, and title.
// Returns an error if validation fails.
func NewGroupAlbum(
	groupID GroupID,
	createdBy identity.UserID,
	title string,
	isPublic bool,
) (*GroupAlbum, error) {
	if groupID.IsZero() {
		return nil, fmt.Errorf("%w: group ID is required", shared.ErrInvalidInput)
	}

	if createdBy.IsZero() {
		return nil, fmt.Errorf("%w: creator ID is required", shared.ErrInvalidInput)
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrGroupAlbumTitleRequired
	}

	if len(title) > MaxAlbumTitleLength {
		return nil, fmt.Errorf("%w: got %d characters", ErrGroupAlbumTitleTooLong, len(title))
	}

	now := time.Now().UTC()
	album := &GroupAlbum{
		id:           NewGroupAlbumID(),
		groupID:      groupID,
		createdBy:    createdBy,
		title:        title,
		description:  "",
		coverImageID: nil,
		imageCount:   0,
		isPublic:     isPublic,
		createdAt:    now,
		updatedAt:    now,
		events:       []shared.DomainEvent{},
	}

	album.addEvent(&GroupAlbumCreated{
		BaseEvent:    shared.NewBaseEvent("community.group_album.created", album.groupID.String()),
		GroupAlbumID: album.id,
		GroupID:      album.groupID,
		CreatedBy:    album.createdBy,
		Title:        album.title,
	})

	return album, nil
}

// ReconstructGroupAlbum reconstitutes a GroupAlbum from persistence without validation or events.
// Use this only when loading from the database.
func ReconstructGroupAlbum(
	id GroupAlbumID,
	groupID GroupID,
	createdBy identity.UserID,
	title string,
	description string,
	coverImageID *gallery.ImageID,
	imageCount int,
	isPublic bool,
	createdAt, updatedAt time.Time,
) *GroupAlbum {
	return &GroupAlbum{
		id:           id,
		groupID:      groupID,
		createdBy:    createdBy,
		title:        title,
		description:  description,
		coverImageID: coverImageID,
		imageCount:   imageCount,
		isPublic:     isPublic,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		events:       []shared.DomainEvent{},
	}
}

// Getters

// ID returns the unique identifier of the album.
func (a *GroupAlbum) ID() GroupAlbumID {
	return a.id
}

// GroupID returns the ID of the group this album belongs to.
func (a *GroupAlbum) GroupID() GroupID {
	return a.groupID
}

// CreatedBy returns the ID of the user who created the album.
func (a *GroupAlbum) CreatedBy() identity.UserID {
	return a.createdBy
}

// Title returns the album title.
func (a *GroupAlbum) Title() string {
	return a.title
}

// Description returns the album description.
func (a *GroupAlbum) Description() string {
	return a.description
}

// CoverImageID returns the ID of the cover image, or nil if not set.
func (a *GroupAlbum) CoverImageID() *gallery.ImageID {
	return a.coverImageID
}

// ImageCount returns the number of images in the album.
func (a *GroupAlbum) ImageCount() int {
	return a.imageCount
}

// IsPublic returns true if the album is publicly visible.
func (a *GroupAlbum) IsPublic() bool {
	return a.isPublic
}

// CreatedAt returns when the album was created.
func (a *GroupAlbum) CreatedAt() time.Time {
	return a.createdAt
}

// UpdatedAt returns when the album was last modified.
func (a *GroupAlbum) UpdatedAt() time.Time {
	return a.updatedAt
}

// Events returns the domain events that have occurred.
func (a *GroupAlbum) Events() []shared.DomainEvent {
	return a.events
}

// ClearEvents clears all pending domain events.
func (a *GroupAlbum) ClearEvents() {
	a.events = []shared.DomainEvent{}
}

// Behavior Methods

// UpdateTitle changes the album title.
func (a *GroupAlbum) UpdateTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrGroupAlbumTitleRequired
	}

	if len(title) > MaxAlbumTitleLength {
		return fmt.Errorf("%w: got %d characters", ErrGroupAlbumTitleTooLong, len(title))
	}

	if a.title == title {
		return nil // No change
	}

	oldTitle := a.title
	a.title = title
	a.updatedAt = time.Now().UTC()

	a.addEvent(&GroupAlbumUpdated{
		BaseEvent:    shared.NewBaseEvent("community.group_album.updated", a.groupID.String()),
		GroupAlbumID: a.id,
		GroupID:      a.groupID,
		OldTitle:     oldTitle,
		NewTitle:     title,
	})

	return nil
}

// UpdateDescription changes the album description.
func (a *GroupAlbum) UpdateDescription(description string) error {
	description = strings.TrimSpace(description)
	if len(description) > MaxAlbumDescriptionLength {
		return fmt.Errorf("%w: got %d characters", ErrGroupAlbumDescTooLong, len(description))
	}

	if a.description == description {
		return nil // No change
	}

	a.description = description
	a.updatedAt = time.Now().UTC()

	a.addEvent(&GroupAlbumUpdated{
		BaseEvent:      shared.NewBaseEvent("community.group_album.updated", a.groupID.String()),
		GroupAlbumID:   a.id,
		GroupID:        a.groupID,
		NewDescription: &description,
	})

	return nil
}

// SetCoverImage sets the cover image for the album.
// Pass nil to remove the cover image.
func (a *GroupAlbum) SetCoverImage(imageID *gallery.ImageID) {
	// Check if actually changing
	if a.coverImageID == nil && imageID == nil {
		return // Both nil
	}
	if a.coverImageID != nil && imageID != nil && a.coverImageID.Equals(*imageID) {
		return // Same image
	}

	a.coverImageID = imageID
	a.updatedAt = time.Now().UTC()

	a.addEvent(&GroupAlbumUpdated{
		BaseEvent:       shared.NewBaseEvent("community.group_album.updated", a.groupID.String()),
		GroupAlbumID:    a.id,
		GroupID:         a.groupID,
		NewCoverImageID: imageID,
	})
}

// SetVisibility changes the album's public visibility.
func (a *GroupAlbum) SetVisibility(isPublic bool) {
	if a.isPublic == isPublic {
		return // No change
	}

	a.isPublic = isPublic
	a.updatedAt = time.Now().UTC()

	a.addEvent(&GroupAlbumUpdated{
		BaseEvent:    shared.NewBaseEvent("community.group_album.updated", a.groupID.String()),
		GroupAlbumID: a.id,
		GroupID:      a.groupID,
		NewIsPublic:  &isPublic,
	})
}

// IncrementImageCount increments the image count.
// This is typically called when an image is added to the album.
func (a *GroupAlbum) IncrementImageCount() {
	a.imageCount++
	a.updatedAt = time.Now().UTC()
}

// DecrementImageCount decrements the image count.
// This is typically called when an image is removed from the album.
func (a *GroupAlbum) DecrementImageCount() {
	if a.imageCount > 0 {
		a.imageCount--
		a.updatedAt = time.Now().UTC()
	}
}

// IsOwnedBy returns true if the album was created by the given user.
func (a *GroupAlbum) IsOwnedBy(userID identity.UserID) bool {
	return a.createdBy.Equals(userID)
}

// IsEmpty returns true if the album has no images.
func (a *GroupAlbum) IsEmpty() bool {
	return a.imageCount == 0
}

// Helper Methods

// addEvent appends a domain event to the events slice.
func (a *GroupAlbum) addEvent(event shared.DomainEvent) {
	a.events = append(a.events, event)
}
