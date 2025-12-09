package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// UnlikeImageCommand represents the intent to unlike an image.
type UnlikeImageCommand struct {
	UserID  string
	ImageID string
}

// UnlikeImageResult contains the result of an unlike operation.
type UnlikeImageResult struct {
	Liked     bool  // Always false for an unlike operation
	LikeCount int64 // Current total like count for the image
}

// UnlikeImageHandler processes image unlike commands.
// It removes the like relationship, updates the denormalized like count,
// and publishes the ImageUnliked domain event.
type UnlikeImageHandler struct {
	images    gallery.ImageRepository
	likes     gallery.LikeRepository
	users     identity.UserRepository
	publisher EventPublisher
	logger    *zerolog.Logger
}

// NewUnlikeImageHandler creates a new UnlikeImageHandler with the given dependencies.
func NewUnlikeImageHandler(
	images gallery.ImageRepository,
	likes gallery.LikeRepository,
	users identity.UserRepository,
	publisher EventPublisher,
	logger *zerolog.Logger,
) *UnlikeImageHandler {
	return &UnlikeImageHandler{
		images:    images,
		likes:     likes,
		users:     users,
		publisher: publisher,
		logger:    logger,
	}
}

// Handle executes the unlike image use case.
//
// Process flow:
//  1. Parse and validate user ID
//  2. Parse and validate image ID
//  3. Verify user exists
//  4. Verify image exists
//  5. Check if like exists (idempotent if doesn't exist)
//  6. Remove like relationship
//  7. Update denormalized like count on image
//  8. Publish ImageUnliked domain event after successful save
//
// Returns:
//   - UnlikeImageResult with liked=false and like_count on success
//   - Validation errors from domain value objects
//   - ErrUserNotFound if user doesn't exist
//   - ErrImageNotFound if image doesn't exist
//
//nolint:cyclop // Command handler requires sequential validation: user ID, image ID, permissions, like status, and persistence
func (h *UnlikeImageHandler) Handle(ctx context.Context, cmd UnlikeImageCommand) (*UnlikeImageResult, error) {
	// 1. Parse user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id for unlike image")
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Parse image ID
	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", cmd.ImageID).
			Msg("invalid image id for unlike")
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	// 3. Verify user exists
	_, err = h.users.FindByID(ctx, userID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", userID.String()).
			Msg("user not found for unlike image")
		return nil, fmt.Errorf("find user: %w", err)
	}

	// 4. Load image
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("image not found for unlike")
		return nil, fmt.Errorf("find image: %w", err)
	}

	// 5. Check if like exists (idempotent)
	hasLiked, err := h.likes.HasLiked(ctx, userID, imageID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", userID.String()).
			Str("image_id", imageID.String()).
			Msg("failed to check if user has liked image")
		return nil, fmt.Errorf("check has liked: %w", err)
	}

	// Get current like count (needed for response even if not liked)
	likeCount, err := h.likes.GetLikeCount(ctx, imageID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to get like count")
		return nil, fmt.Errorf("get like count: %w", err)
	}

	if !hasLiked {
		// Not liked - idempotent operation, return success with current count
		h.logger.Debug().
			Str("user_id", userID.String()).
			Str("image_id", imageID.String()).
			Msg("user has not liked this image")
		return &UnlikeImageResult{Liked: false, LikeCount: likeCount}, nil
	}

	// 6. Remove like relationship
	if err := h.likes.Unlike(ctx, userID, imageID); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", userID.String()).
			Str("image_id", imageID.String()).
			Msg("failed to remove like")
		return nil, fmt.Errorf("remove like: %w", err)
	}

	// 7. Update denormalized like count on image (re-fetch after unlike)
	likeCount, err = h.likes.GetLikeCount(ctx, imageID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to get like count")
		return nil, fmt.Errorf("get like count: %w", err)
	}

	image.SetLikeCount(likeCount)

	if err := h.images.Save(ctx, image); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to update image like count")
		return nil, fmt.Errorf("update image like count: %w", err)
	}

	// 8. Publish ImageUnliked domain event
	event := &gallery.ImageUnliked{
		BaseEvent: shared.NewBaseEvent("gallery.image.unliked", imageID.String()),
		ImageID:   imageID,
		UserID:    userID,
		UnlikedAt: time.Now().UTC(),
	}

	if err := h.publisher.Publish(ctx, event); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Str("user_id", userID.String()).
			Msg("failed to publish ImageUnliked event")
		// Don't fail the operation - the unlike was already processed
	}

	h.logger.Info().
		Str("user_id", userID.String()).
		Str("image_id", imageID.String()).
		Int64("like_count", likeCount).
		Msg("image unliked successfully")

	return &UnlikeImageResult{Liked: false, LikeCount: likeCount}, nil
}
