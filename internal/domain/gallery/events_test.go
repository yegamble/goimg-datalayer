package gallery_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestImageEvents_EventType(t *testing.T) {
	t.Parallel()

	metadata, _ := gallery.NewImageMetadata("Title", "", "file.jpg", "image/jpeg", 100, 100, 1000, "key", "s3")
	imageID := gallery.NewImageID()
	ownerID := identity.NewUserID()

	tests := []struct {
		name      string
		event     shared.DomainEvent
		eventType string
	}{
		{
			name: "ImageUploaded",
			event: &gallery.ImageUploaded{
				BaseEvent: shared.NewBaseEvent("gallery.image.uploaded", imageID.String()),
				ImageID:   imageID,
				OwnerID:   ownerID,
				Metadata:  metadata,
			},
			eventType: "gallery.image.uploaded",
		},
		{
			name: "ImageProcessingCompleted",
			event: &gallery.ImageProcessingCompleted{
				BaseEvent: shared.NewBaseEvent("gallery.image.processing_completed", imageID.String()),
				ImageID:   imageID,
			},
			eventType: "gallery.image.processing_completed",
		},
		{
			name: "ImageDeleted",
			event: &gallery.ImageDeleted{
				BaseEvent: shared.NewBaseEvent("gallery.image.deleted", imageID.String()),
				ImageID:   imageID,
				OwnerID:   ownerID,
			},
			eventType: "gallery.image.deleted",
		},
		{
			name: "ImageFlagged",
			event: &gallery.ImageFlagged{
				BaseEvent: shared.NewBaseEvent("gallery.image.flagged", imageID.String()),
				ImageID:   imageID,
			},
			eventType: "gallery.image.flagged",
		},
		{
			name: "ImageVisibilityChanged",
			event: &gallery.ImageVisibilityChanged{
				BaseEvent:     shared.NewBaseEvent("gallery.image.visibility_changed", imageID.String()),
				ImageID:       imageID,
				OldVisibility: gallery.VisibilityPrivate,
				NewVisibility: gallery.VisibilityPublic,
			},
			eventType: "gallery.image.visibility_changed",
		},
		{
			name: "ImageMetadataUpdated",
			event: &gallery.ImageMetadataUpdated{
				BaseEvent: shared.NewBaseEvent("gallery.image.metadata_updated", imageID.String()),
				ImageID:   imageID,
				Metadata:  metadata,
			},
			eventType: "gallery.image.metadata_updated",
		},
		{
			name: "ImageVariantAdded",
			event: &gallery.ImageVariantAdded{
				BaseEvent:   shared.NewBaseEvent("gallery.image.variant_added", imageID.String()),
				ImageID:     imageID,
				VariantType: gallery.VariantThumbnail,
			},
			eventType: "gallery.image.variant_added",
		},
		{
			name: "ImageTagAdded",
			event: &gallery.ImageTagAdded{
				BaseEvent: shared.NewBaseEvent("gallery.image.tag_added", imageID.String()),
				ImageID:   imageID,
				Tag:       gallery.MustNewTag("landscape"),
			},
			eventType: "gallery.image.tag_added",
		},
		{
			name: "ImageTagRemoved",
			event: &gallery.ImageTagRemoved{
				BaseEvent: shared.NewBaseEvent("gallery.image.tag_removed", imageID.String()),
				ImageID:   imageID,
				Tag:       gallery.MustNewTag("landscape"),
			},
			eventType: "gallery.image.tag_removed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.eventType, tt.event.EventType())
			assert.False(t, tt.event.OccurredAt().IsZero())
			assert.NotEmpty(t, tt.event.EventID())
		})
	}
}

func TestAlbumEvents_EventType(t *testing.T) {
	t.Parallel()

	albumID := gallery.NewAlbumID()
	imageID := gallery.NewImageID()
	ownerID := identity.NewUserID()

	tests := []struct {
		name      string
		event     shared.DomainEvent
		eventType string
	}{
		{
			name: "AlbumCreated",
			event: &gallery.AlbumCreated{
				BaseEvent: shared.NewBaseEvent("gallery.album.created", albumID.String()),
				AlbumID:   albumID,
				OwnerID:   ownerID,
				Title:     "Album",
			},
			eventType: "gallery.album.created",
		},
		{
			name: "AlbumTitleUpdated",
			event: &gallery.AlbumTitleUpdated{
				BaseEvent: shared.NewBaseEvent("gallery.album.title_updated", albumID.String()),
				AlbumID:   albumID,
				NewTitle:  "New Title",
			},
			eventType: "gallery.album.title_updated",
		},
		{
			name: "AlbumDescriptionUpdated",
			event: &gallery.AlbumDescriptionUpdated{
				BaseEvent:      shared.NewBaseEvent("gallery.album.description_updated", albumID.String()),
				AlbumID:        albumID,
				NewDescription: "New Description",
			},
			eventType: "gallery.album.description_updated",
		},
		{
			name: "AlbumVisibilityChanged",
			event: &gallery.AlbumVisibilityChanged{
				BaseEvent:     shared.NewBaseEvent("gallery.album.visibility_changed", albumID.String()),
				AlbumID:       albumID,
				OldVisibility: gallery.VisibilityPrivate,
				NewVisibility: gallery.VisibilityPublic,
			},
			eventType: "gallery.album.visibility_changed",
		},
		{
			name: "AlbumCoverImageChanged",
			event: &gallery.AlbumCoverImageChanged{
				BaseEvent:    shared.NewBaseEvent("gallery.album.cover_image_changed", albumID.String()),
				AlbumID:      albumID,
				CoverImageID: imageID,
			},
			eventType: "gallery.album.cover_image_changed",
		},
		{
			name: "AlbumImageAdded",
			event: &gallery.AlbumImageAdded{
				BaseEvent: shared.NewBaseEvent("gallery.album.image_added", albumID.String()),
				AlbumID:   albumID,
				ImageID:   imageID,
			},
			eventType: "gallery.album.image_added",
		},
		{
			name: "AlbumImageRemoved",
			event: &gallery.AlbumImageRemoved{
				BaseEvent: shared.NewBaseEvent("gallery.album.image_removed", albumID.String()),
				AlbumID:   albumID,
				ImageID:   imageID,
			},
			eventType: "gallery.album.image_removed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.eventType, tt.event.EventType())
		})
	}
}

func TestCommentEvents_EventType(t *testing.T) {
	t.Parallel()

	commentID := gallery.NewCommentID()
	imageID := gallery.NewImageID()
	userID := identity.NewUserID()

	tests := []struct {
		name      string
		event     shared.DomainEvent
		eventType string
	}{
		{
			name: "CommentAdded",
			event: &gallery.CommentAdded{
				BaseEvent: shared.NewBaseEvent("gallery.comment.added", commentID.String()),
				CommentID: commentID,
				ImageID:   imageID,
				UserID:    userID,
				Content:   "Comment",
			},
			eventType: "gallery.comment.added",
		},
		{
			name: "CommentDeleted",
			event: &gallery.CommentDeleted{
				BaseEvent: shared.NewBaseEvent("gallery.comment.deleted", commentID.String()),
				CommentID: commentID,
				ImageID:   imageID,
				UserID:    userID,
			},
			eventType: "gallery.comment.deleted",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.eventType, tt.event.EventType())
		})
	}
}
