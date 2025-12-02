package gallery

import (
	"fmt"
	"strings"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// Album is an entity representing a collection of images.
// Albums organize images and can have their own visibility settings.
type Album struct {
	id           AlbumID
	ownerID      identity.UserID
	title        string
	description  string
	visibility   Visibility
	coverImageID *ImageID // Optional cover image
	imageCount   int
	createdAt    time.Time
	updatedAt    time.Time
	events       []shared.DomainEvent
}

// NewAlbum creates a new Album with the given owner and title.
// Returns an error if the title is invalid.
func NewAlbum(ownerID identity.UserID, title string) (*Album, error) {
	if ownerID.IsZero() {
		return nil, fmt.Errorf("%w: owner ID is required", shared.ErrInvalidInput)
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrAlbumTitleRequired
	}
	if len(title) > MaxTitleLength {
		return nil, fmt.Errorf("%w: got %d characters", ErrAlbumTitleTooLong, len(title))
	}

	now := time.Now().UTC()
	album := &Album{
		id:           NewAlbumID(),
		ownerID:      ownerID,
		title:        title,
		description:  "",
		visibility:   VisibilityPrivate, // Start private by default
		coverImageID: nil,
		imageCount:   0,
		createdAt:    now,
		updatedAt:    now,
		events:       []shared.DomainEvent{},
	}

	album.addEvent(&AlbumCreated{
		BaseEvent: shared.NewBaseEvent("gallery.album.created", album.id.String()),
		AlbumID:   album.id,
		OwnerID:   album.ownerID,
		Title:     album.title,
	})

	return album, nil
}

// ReconstructAlbum reconstitutes an Album from persistence without validation or events.
// Use this only when loading from the database.
func ReconstructAlbum(
	id AlbumID,
	ownerID identity.UserID,
	title, description string,
	visibility Visibility,
	coverImageID *ImageID,
	imageCount int,
	createdAt, updatedAt time.Time,
) *Album {
	return &Album{
		id:           id,
		ownerID:      ownerID,
		title:        title,
		description:  description,
		visibility:   visibility,
		coverImageID: coverImageID,
		imageCount:   imageCount,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		events:       []shared.DomainEvent{},
	}
}

// Getters

// ID returns the unique identifier of the album.
func (a *Album) ID() AlbumID {
	return a.id
}

// OwnerID returns the ID of the user who owns this album.
func (a *Album) OwnerID() identity.UserID {
	return a.ownerID
}

// Title returns the album title.
func (a *Album) Title() string {
	return a.title
}

// Description returns the album description.
func (a *Album) Description() string {
	return a.description
}

// Visibility returns the album visibility.
func (a *Album) Visibility() Visibility {
	return a.visibility
}

// CoverImageID returns the ID of the cover image, or nil if not set.
func (a *Album) CoverImageID() *ImageID {
	return a.coverImageID
}

// ImageCount returns the number of images in the album.
func (a *Album) ImageCount() int {
	return a.imageCount
}

// CreatedAt returns when the album was created.
func (a *Album) CreatedAt() time.Time {
	return a.createdAt
}

// UpdatedAt returns when the album was last modified.
func (a *Album) UpdatedAt() time.Time {
	return a.updatedAt
}

// Events returns the domain events that have occurred.
func (a *Album) Events() []shared.DomainEvent {
	return a.events
}

// ClearEvents clears all pending domain events.
func (a *Album) ClearEvents() {
	a.events = []shared.DomainEvent{}
}

// Behavior Methods

// UpdateTitle changes the album title.
func (a *Album) UpdateTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrAlbumTitleRequired
	}
	if len(title) > MaxTitleLength {
		return fmt.Errorf("%w: got %d characters", ErrAlbumTitleTooLong, len(title))
	}

	if a.title == title {
		return nil // No change
	}

	a.title = title
	a.updatedAt = time.Now().UTC()

	a.addEvent(&AlbumTitleUpdated{
		BaseEvent: shared.NewBaseEvent("gallery.album.title_updated", a.id.String()),
		AlbumID:   a.id,
		NewTitle:  title,
	})

	return nil
}

// UpdateDescription changes the album description.
func (a *Album) UpdateDescription(description string) error {
	description = strings.TrimSpace(description)
	if len(description) > MaxDescriptionLength {
		return fmt.Errorf("%w: got %d characters", ErrAlbumDescTooLong, len(description))
	}

	if a.description == description {
		return nil // No change
	}

	a.description = description
	a.updatedAt = time.Now().UTC()

	a.addEvent(&AlbumDescriptionUpdated{
		BaseEvent:      shared.NewBaseEvent("gallery.album.description_updated", a.id.String()),
		AlbumID:        a.id,
		NewDescription: description,
	})

	return nil
}

// UpdateVisibility changes the album visibility.
func (a *Album) UpdateVisibility(visibility Visibility) error {
	if !visibility.IsValid() {
		return ErrInvalidVisibility
	}

	if a.visibility == visibility {
		return nil // No change
	}

	oldVisibility := a.visibility
	a.visibility = visibility
	a.updatedAt = time.Now().UTC()

	a.addEvent(&AlbumVisibilityChanged{
		BaseEvent:     shared.NewBaseEvent("gallery.album.visibility_changed", a.id.String()),
		AlbumID:       a.id,
		OldVisibility: oldVisibility,
		NewVisibility: visibility,
	})

	return nil
}

// SetCoverImage sets the cover image for the album.
// Pass nil to remove the cover image.
func (a *Album) SetCoverImage(imageID *ImageID) {
	// Check if actually changing
	if a.coverImageID == nil && imageID == nil {
		return // Both nil
	}
	if a.coverImageID != nil && imageID != nil && a.coverImageID.Equals(*imageID) {
		return // Same image
	}

	a.coverImageID = imageID
	a.updatedAt = time.Now().UTC()

	var eventImageID ImageID
	if imageID != nil {
		eventImageID = *imageID
	}

	a.addEvent(&AlbumCoverImageChanged{
		BaseEvent:    shared.NewBaseEvent("gallery.album.cover_image_changed", a.id.String()),
		AlbumID:      a.id,
		CoverImageID: eventImageID,
	})
}

// IncrementImageCount increments the count of images in the album.
// This is typically called when an image is added to the album.
func (a *Album) IncrementImageCount() {
	a.imageCount++
	a.updatedAt = time.Now().UTC()
}

// DecrementImageCount decrements the count of images in the album.
// This is typically called when an image is removed from the album.
func (a *Album) DecrementImageCount() {
	if a.imageCount > 0 {
		a.imageCount--
		a.updatedAt = time.Now().UTC()
	}
}

// Helper Methods

// IsOwnedBy returns true if the album is owned by the given user.
func (a *Album) IsOwnedBy(userID identity.UserID) bool {
	return a.ownerID.Equals(userID)
}

// IsEmpty returns true if the album has no images.
func (a *Album) IsEmpty() bool {
	return a.imageCount == 0
}

// addEvent appends a domain event to the events slice.
func (a *Album) addEvent(event shared.DomainEvent) {
	a.events = append(a.events, event)
}
