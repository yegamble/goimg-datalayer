package gallery

import (
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// Image Events

// ImageUploaded is emitted when a new image is uploaded.
type ImageUploaded struct {
	shared.BaseEvent
	ImageID  ImageID
	OwnerID  identity.UserID
	Metadata ImageMetadata
}

// EventType returns the event type identifier.
func (e *ImageUploaded) EventType() string {
	return "gallery.image.uploaded"
}

// ImageProcessingCompleted is emitted when image processing (variant generation) completes.
type ImageProcessingCompleted struct {
	shared.BaseEvent
	ImageID ImageID
}

// EventType returns the event type identifier.
func (e *ImageProcessingCompleted) EventType() string {
	return "gallery.image.processing_completed"
}

// ImageDeleted is emitted when an image is soft-deleted.
type ImageDeleted struct {
	shared.BaseEvent
	ImageID ImageID
	OwnerID identity.UserID
}

// EventType returns the event type identifier.
func (e *ImageDeleted) EventType() string {
	return "gallery.image.deleted"
}

// ImageFlagged is emitted when an image is flagged for moderation.
type ImageFlagged struct {
	shared.BaseEvent
	ImageID ImageID
}

// EventType returns the event type identifier.
func (e *ImageFlagged) EventType() string {
	return "gallery.image.flagged"
}

// ImageVisibilityChanged is emitted when an image's visibility changes.
type ImageVisibilityChanged struct {
	shared.BaseEvent
	ImageID       ImageID
	OldVisibility Visibility
	NewVisibility Visibility
}

// EventType returns the event type identifier.
func (e *ImageVisibilityChanged) EventType() string {
	return "gallery.image.visibility_changed"
}

// ImageMetadataUpdated is emitted when an image's metadata (title/description) is updated.
type ImageMetadataUpdated struct {
	shared.BaseEvent
	ImageID  ImageID
	Metadata ImageMetadata
}

// EventType returns the event type identifier.
func (e *ImageMetadataUpdated) EventType() string {
	return "gallery.image.metadata_updated"
}

// ImageVariantAdded is emitted when a new variant is added to an image.
type ImageVariantAdded struct {
	shared.BaseEvent
	ImageID     ImageID
	VariantType VariantType
}

// EventType returns the event type identifier.
func (e *ImageVariantAdded) EventType() string {
	return "gallery.image.variant_added"
}

// ImageTagAdded is emitted when a tag is added to an image.
type ImageTagAdded struct {
	shared.BaseEvent
	ImageID ImageID
	Tag     Tag
}

// EventType returns the event type identifier.
func (e *ImageTagAdded) EventType() string {
	return "gallery.image.tag_added"
}

// ImageTagRemoved is emitted when a tag is removed from an image.
type ImageTagRemoved struct {
	shared.BaseEvent
	ImageID ImageID
	Tag     Tag
}

// EventType returns the event type identifier.
func (e *ImageTagRemoved) EventType() string {
	return "gallery.image.tag_removed"
}

// Album Events

// AlbumCreated is emitted when a new album is created.
type AlbumCreated struct {
	shared.BaseEvent
	AlbumID AlbumID
	OwnerID identity.UserID
	Title   string
}

// EventType returns the event type identifier.
func (e *AlbumCreated) EventType() string {
	return "gallery.album.created"
}

// AlbumTitleUpdated is emitted when an album's title is updated.
type AlbumTitleUpdated struct {
	shared.BaseEvent
	AlbumID  AlbumID
	NewTitle string
}

// EventType returns the event type identifier.
func (e *AlbumTitleUpdated) EventType() string {
	return "gallery.album.title_updated"
}

// AlbumDescriptionUpdated is emitted when an album's description is updated.
type AlbumDescriptionUpdated struct {
	shared.BaseEvent
	AlbumID        AlbumID
	NewDescription string
}

// EventType returns the event type identifier.
func (e *AlbumDescriptionUpdated) EventType() string {
	return "gallery.album.description_updated"
}

// AlbumVisibilityChanged is emitted when an album's visibility changes.
type AlbumVisibilityChanged struct {
	shared.BaseEvent
	AlbumID       AlbumID
	OldVisibility Visibility
	NewVisibility Visibility
}

// EventType returns the event type identifier.
func (e *AlbumVisibilityChanged) EventType() string {
	return "gallery.album.visibility_changed"
}

// AlbumCoverImageChanged is emitted when an album's cover image changes.
type AlbumCoverImageChanged struct {
	shared.BaseEvent
	AlbumID      AlbumID
	CoverImageID ImageID
}

// EventType returns the event type identifier.
func (e *AlbumCoverImageChanged) EventType() string {
	return "gallery.album.cover_image_changed"
}

// AlbumImageAdded is emitted when an image is added to an album.
type AlbumImageAdded struct {
	shared.BaseEvent
	AlbumID AlbumID
	ImageID ImageID
}

// EventType returns the event type identifier.
func (e *AlbumImageAdded) EventType() string {
	return "gallery.album.image_added"
}

// AlbumImageRemoved is emitted when an image is removed from an album.
type AlbumImageRemoved struct {
	shared.BaseEvent
	AlbumID AlbumID
	ImageID ImageID
}

// EventType returns the event type identifier.
func (e *AlbumImageRemoved) EventType() string {
	return "gallery.album.image_removed"
}

// Comment Events

// CommentAdded is emitted when a comment is added to an image.
type CommentAdded struct {
	shared.BaseEvent
	CommentID CommentID
	ImageID   ImageID
	UserID    identity.UserID
	Content   string
}

// EventType returns the event type identifier.
func (e *CommentAdded) EventType() string {
	return "gallery.comment.added"
}

// CommentDeleted is emitted when a comment is deleted.
type CommentDeleted struct {
	shared.BaseEvent
	CommentID CommentID
	ImageID   ImageID
	UserID    identity.UserID
	DeletedAt time.Time
}

// EventType returns the event type identifier.
func (e *CommentDeleted) EventType() string {
	return "gallery.comment.deleted"
}

// Like Events

// ImageLiked is emitted when a user likes an image.
type ImageLiked struct {
	shared.BaseEvent
	ImageID   ImageID
	UserID    identity.UserID
	LikedAt   time.Time
}

// EventType returns the event type identifier.
func (e *ImageLiked) EventType() string {
	return "gallery.image.liked"
}

// ImageUnliked is emitted when a user unlikes an image.
type ImageUnliked struct {
	shared.BaseEvent
	ImageID    ImageID
	UserID     identity.UserID
	UnlikedAt  time.Time
}

// EventType returns the event type identifier.
func (e *ImageUnliked) EventType() string {
	return "gallery.image.unliked"
}
