package gallery

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// Image is the aggregate root for the gallery bounded context.
// It manages all aspects of an uploaded image including metadata, variants, tags, and visibility.
// All modifications must go through the aggregate root to maintain invariants.
type Image struct {
	id           ImageID
	ownerID      identity.UserID
	metadata     ImageMetadata
	visibility   Visibility
	status       ImageStatus
	variants     []ImageVariant
	tags         []Tag
	viewCount    int64
	likeCount    int64
	commentCount int64
	createdAt    time.Time
	updatedAt    time.Time
	events       []shared.DomainEvent
}

// NewImage creates a new Image aggregate with the given owner and metadata.
// The image starts in processing status with private visibility.
// An ImageUploaded event is emitted.
func NewImage(ownerID identity.UserID, metadata ImageMetadata) (*Image, error) {
	if ownerID.IsZero() {
		return nil, fmt.Errorf("%w: owner ID is required", shared.ErrInvalidInput)
	}

	now := time.Now().UTC()
	img := &Image{
		id:           NewImageID(),
		ownerID:      ownerID,
		metadata:     metadata,
		visibility:   VisibilityPrivate, // Start private until processing completes
		status:       StatusProcessing,
		variants:     []ImageVariant{},
		tags:         []Tag{},
		viewCount:    0,
		likeCount:    0,
		commentCount: 0,
		createdAt:    now,
		updatedAt:    now,
		events:       []shared.DomainEvent{},
	}

	// Emit domain event
	img.addEvent(&ImageUploaded{
		BaseEvent: shared.NewBaseEvent("gallery.image.uploaded", img.id.String()),
		ImageID:   img.id,
		OwnerID:   img.ownerID,
		Metadata:  img.metadata,
	})

	return img, nil
}

// ReconstructImage reconstitutes an Image from persistence without validation or events.
// Use this only when loading from the database.
func ReconstructImage(
	id ImageID,
	ownerID identity.UserID,
	metadata ImageMetadata,
	visibility Visibility,
	status ImageStatus,
	variants []ImageVariant,
	tags []Tag,
	viewCount, likeCount, commentCount int64,
	createdAt, updatedAt time.Time,
) *Image {
	return &Image{
		id:           id,
		ownerID:      ownerID,
		metadata:     metadata,
		visibility:   visibility,
		status:       status,
		variants:     variants,
		tags:         tags,
		viewCount:    viewCount,
		likeCount:    likeCount,
		commentCount: commentCount,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		events:       []shared.DomainEvent{},
	}
}

// Getters

// ID returns the unique identifier of the image.
func (i *Image) ID() ImageID {
	return i.id
}

// OwnerID returns the ID of the user who owns this image.
func (i *Image) OwnerID() identity.UserID {
	return i.ownerID
}

// Metadata returns the image metadata.
func (i *Image) Metadata() ImageMetadata {
	return i.metadata
}

// Visibility returns the current visibility setting.
func (i *Image) Visibility() Visibility {
	return i.visibility
}

// Status returns the current status.
func (i *Image) Status() ImageStatus {
	return i.status
}

// Variants returns a copy of the variants slice.
func (i *Image) Variants() []ImageVariant {
	result := make([]ImageVariant, len(i.variants))
	copy(result, i.variants)
	return result
}

// Tags returns a copy of the tags slice.
func (i *Image) Tags() []Tag {
	result := make([]Tag, len(i.tags))
	copy(result, i.tags)
	return result
}

// ViewCount returns the number of views.
func (i *Image) ViewCount() int64 {
	return i.viewCount
}

// LikeCount returns the number of likes.
func (i *Image) LikeCount() int64 {
	return i.likeCount
}

// CommentCount returns the number of comments.
func (i *Image) CommentCount() int64 {
	return i.commentCount
}

// CreatedAt returns when the image was created.
func (i *Image) CreatedAt() time.Time {
	return i.createdAt
}

// UpdatedAt returns when the image was last modified.
func (i *Image) UpdatedAt() time.Time {
	return i.updatedAt
}

// Events returns the domain events that have occurred.
func (i *Image) Events() []shared.DomainEvent {
	return i.events
}

// ClearEvents clears all pending domain events.
func (i *Image) ClearEvents() {
	i.events = []shared.DomainEvent{}
}

// Getters.

// ID returns the unique identifier of the image.
func (i *Image) ID() ImageID {
	return i.id
}

// OwnerID returns the ID of the user who owns this image.
func (i *Image) OwnerID() identity.UserID {
	return i.ownerID
}

// Metadata returns the image metadata.
func (i *Image) Metadata() ImageMetadata {
	return i.metadata
}

// Visibility returns the current visibility setting.
func (i *Image) Visibility() Visibility {
	return i.visibility
}

// Status returns the current status.
func (i *Image) Status() ImageStatus {
	return i.status
}

// Variants returns a copy of the variants slice.
func (i *Image) Variants() []ImageVariant {
	result := make([]ImageVariant, len(i.variants))
	copy(result, i.variants)
	return result
}

// Tags returns a copy of the tags slice.
func (i *Image) Tags() []Tag {
	result := make([]Tag, len(i.tags))
	copy(result, i.tags)
	return result
}

// ViewCount returns the number of views.
func (i *Image) ViewCount() int64 {
	return i.viewCount
}

// LikeCount returns the number of likes.
func (i *Image) LikeCount() int64 {
	return i.likeCount
}

// CommentCount returns the number of comments.
func (i *Image) CommentCount() int64 {
	return i.commentCount
}

// CreatedAt returns when the image was created.
func (i *Image) CreatedAt() time.Time {
	return i.createdAt
}

// UpdatedAt returns when the image was last modified.
func (i *Image) UpdatedAt() time.Time {
	return i.updatedAt
}

// Events returns the domain events that have occurred.
func (i *Image) Events() []shared.DomainEvent {
	return i.events
}

// ClearEvents clears all pending domain events.
func (i *Image) ClearEvents() {
	i.events = []shared.DomainEvent{}
}

// Behavior Methods - Variant Management.

// AddVariant adds a new variant to the image.
// Returns an error if a variant of the same type already exists.
func (i *Image) AddVariant(variant ImageVariant) error {
	// Check if variant already exists
	for _, v := range i.variants {
		if v.VariantType() == variant.VariantType() {
			return fmt.Errorf("%w: %s variant already exists", ErrVariantExists, variant.VariantType())
		}
	}

	i.variants = append(i.variants, variant)
	i.updatedAt = time.Now().UTC()

	i.addEvent(&ImageVariantAdded{
		BaseEvent:   shared.NewBaseEvent("gallery.image.variant_added", i.id.String()),
		ImageID:     i.id,
		VariantType: variant.VariantType(),
	})

	return nil
}

// GetVariant retrieves a variant by type.
// Returns an error if the variant doesn't exist.
func (i *Image) GetVariant(variantType VariantType) (ImageVariant, error) {
	for _, v := range i.variants {
		if v.VariantType() == variantType {
			return v, nil
		}
	}
	return ImageVariant{}, fmt.Errorf("%w: %s variant not found", ErrVariantNotFound, variantType)
}

// HasVariant returns true if the image has a variant of the given type.
func (i *Image) HasVariant(variantType VariantType) bool {
	_, err := i.GetVariant(variantType)
	return err == nil
}

// Behavior Methods - Tag Management.

// AddTag adds a tag to the image.
// Returns an error if the tag already exists or the maximum number of tags is reached.
func (i *Image) AddTag(tag Tag) error {
	// Check if image is deleted
	if i.status == StatusDeleted {
		return ErrCannotModifyDeleted
	}

	// Check maximum tags
	if len(i.tags) >= MaxTagsPerImage {
		return fmt.Errorf("%w: image has %d tags", ErrTooManyTags, len(i.tags))
	}

	// Check if tag already exists
	if i.HasTag(tag) {
		return ErrTagAlreadyExists
	}

	i.tags = append(i.tags, tag)
	i.updatedAt = time.Now().UTC()

	i.addEvent(&ImageTagAdded{
		BaseEvent: shared.NewBaseEvent("gallery.image.tag_added", i.id.String()),
		ImageID:   i.id,
		Tag:       tag,
	})

	return nil
}

// RemoveTag removes a tag from the image.
// No error if the tag doesn't exist (idempotent).
func (i *Image) RemoveTag(tag Tag) error {
	// Check if image is deleted
	if i.status == StatusDeleted {
		return ErrCannotModifyDeleted
	}

	// Find and remove the tag
	for idx, t := range i.tags {
		if t.Equals(tag) {
			i.tags = append(i.tags[:idx], i.tags[idx+1:]...)
			i.updatedAt = time.Now().UTC()

			i.addEvent(&ImageTagRemoved{
				BaseEvent: shared.NewBaseEvent("gallery.image.tag_removed", i.id.String()),
				ImageID:   i.id,
				Tag:       tag,
			})

			return nil
		}
	}

	// Tag not found, but this is idempotent so no error
	return nil
}

// HasTag returns true if the image has the given tag.
func (i *Image) HasTag(tag Tag) bool {
	for _, t := range i.tags {
		if t.Equals(tag) {
			return true
		}
	}
	return false
}

// Behavior Methods - Visibility and Status.

// UpdateVisibility changes the visibility of the image.
// Cannot change visibility of deleted or processing images.
func (i *Image) UpdateVisibility(visibility Visibility) error {
	// Check if image is deleted
	if i.status == StatusDeleted {
		return ErrCannotModifyDeleted
	}

	// Check if image is still processing
	if i.status == StatusProcessing {
		return ErrImageProcessing
	}

	// No-op if same visibility
	if i.visibility == visibility {
		return nil
	}

	oldVisibility := i.visibility
	i.visibility = visibility
	i.updatedAt = time.Now().UTC()

	i.addEvent(&ImageVisibilityChanged{
		BaseEvent:     shared.NewBaseEvent("gallery.image.visibility_changed", i.id.String()),
		ImageID:       i.id,
		OldVisibility: oldVisibility,
		NewVisibility: visibility,
	})

	return nil
}

// MarkAsActive marks the image as active after processing completes.
// This makes the image viewable according to its visibility settings.
func (i *Image) MarkAsActive() error {
	if i.status == StatusDeleted {
		return ErrCannotModifyDeleted
	}

	if i.status == StatusActive {
		return nil // Already active
	}

	i.status = StatusActive
	i.updatedAt = time.Now().UTC()

	i.addEvent(&ImageProcessingCompleted{
		BaseEvent: shared.NewBaseEvent("gallery.image.processing_completed", i.id.String()),
		ImageID:   i.id,
	})

	return nil
}

// MarkAsDeleted soft-deletes the image.
// Flagged images cannot be deleted until unflagged.
func (i *Image) MarkAsDeleted() error {
	if i.status == StatusDeleted {
		return nil // Already deleted
	}

	if i.status == StatusFlagged {
		return ErrCannotDeleteFlagged
	}

	i.status = StatusDeleted
	i.updatedAt = time.Now().UTC()

	i.addEvent(&ImageDeleted{
		BaseEvent: shared.NewBaseEvent("gallery.image.deleted", i.id.String()),
		ImageID:   i.id,
		OwnerID:   i.ownerID,
	})

	return nil
}

// Flag marks the image as flagged for moderation review.
func (i *Image) Flag() error {
	if i.status == StatusDeleted {
		return ErrCannotModifyDeleted
	}

	if i.status == StatusFlagged {
		return nil // Already flagged
	}

	i.status = StatusFlagged
	i.updatedAt = time.Now().UTC()

	i.addEvent(&ImageFlagged{
		BaseEvent: shared.NewBaseEvent("gallery.image.flagged", i.id.String()),
		ImageID:   i.id,
	})

	return nil
}

// Behavior Methods - Metadata Updates.

// UpdateMetadata updates the title and description of the image.
// Cannot modify deleted images.
func (i *Image) UpdateMetadata(title, description string) error {
	if i.status == StatusDeleted {
		return ErrCannotModifyDeleted
	}

	// Update metadata immutably
	newMetadata, err := i.metadata.WithTitle(title)
	if err != nil {
		return err
	}

	newMetadata, err = newMetadata.WithDescription(description)
	if err != nil {
		return err
	}

	// Only update if something changed
	if newMetadata.Title() == i.metadata.Title() && newMetadata.Description() == i.metadata.Description() {
		return nil
	}

	i.metadata = newMetadata
	i.updatedAt = time.Now().UTC()

	i.addEvent(&ImageMetadataUpdated{
		BaseEvent: shared.NewBaseEvent("gallery.image.metadata_updated", i.id.String()),
		ImageID:   i.id,
		Metadata:  i.metadata,
	})

	return nil
}

// Behavior Methods - Engagement Metrics.

// IncrementViews increments the view count.
func (i *Image) IncrementViews() {
	i.viewCount++
	// Note: We don't update updatedAt for view increments as they're frequent
	// and not considered meaningful changes to the image entity
}

// SetLikeCount sets the like count.
// This is typically called by the social context via events.
func (i *Image) SetLikeCount(count int64) {
	if count < 0 {
		count = 0
	}
	i.likeCount = count
}

// SetCommentCount sets the comment count.
// This is typically called when comments are added or removed.
func (i *Image) SetCommentCount(count int64) {
	if count < 0 {
		count = 0
	}
	i.commentCount = count
}

// Helper Methods.

// IsOwnedBy returns true if the image is owned by the given user.
func (i *Image) IsOwnedBy(userID identity.UserID) bool {
	return i.ownerID.Equals(userID)
}

// IsViewable returns true if the image can be viewed based on its status.
func (i *Image) IsViewable() bool {
	return i.status.IsViewable()
}

// IsDeleted returns true if the image has been deleted.
func (i *Image) IsDeleted() bool {
	return i.status.IsDeleted()
}

// IsFlagged returns true if the image has been flagged.
func (i *Image) IsFlagged() bool {
	return i.status.IsFlagged()
}

// addEvent appends a domain event to the events slice.
func (i *Image) addEvent(event shared.DomainEvent) {
	i.events = append(i.events, event)
}
