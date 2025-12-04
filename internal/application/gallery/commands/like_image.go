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

// LikeImageCommand represents the intent to like an image.
type LikeImageCommand struct {
	UserID  string
	ImageID string
}

// LikeImageHandler processes image like commands.
// It validates that the image exists and is visible to the user,
// creates the like relationship, updates the denormalized like count,
// and publishes the ImageLiked domain event.
type LikeImageHandler struct {
	images    gallery.ImageRepository
	likes     gallery.LikeRepository
	users     identity.UserRepository
	publisher EventPublisher
	logger    *zerolog.Logger
}

// NewLikeImageHandler creates a new LikeImageHandler with the given dependencies.
func NewLikeImageHandler(
	images gallery.ImageRepository,
	likes gallery.LikeRepository,
	users identity.UserRepository,
	publisher EventPublisher,
	logger *zerolog.Logger,
) *LikeImageHandler {
	return &LikeImageHandler{
		images:    images,
		likes:     likes,
		users:     users,
		publisher: publisher,
		logger:    logger,
	}
}

// Handle executes the like image use case.
//
// Process flow:
//  1. Parse and validate user ID
//  2. Parse and validate image ID
//  3. Verify user exists
//  4. Verify image exists
//  5. Check image visibility (must be accessible to user)
//  6. Check if user has already liked the image
//  7. Create like relationship
//  8. Update denormalized like count on image
//  9. Publish ImageLiked domain event after successful save
//
// Returns:
//   - nil on successful like
//   - Validation errors from domain value objects
//   - ErrUserNotFound if user doesn't exist
//   - ErrImageNotFound if image doesn't exist
//   - ErrUnauthorizedAccess if image is not visible to user
func (h *LikeImageHandler) Handle(ctx context.Context, cmd LikeImageCommand) error {
	// 1. Parse user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id for like image")
		return fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Parse image ID
	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", cmd.ImageID).
			Msg("invalid image id for like")
		return fmt.Errorf("invalid image id: %w", err)
	}

	// 3. Verify user exists
	_, err = h.users.FindByID(ctx, userID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", userID.String()).
			Msg("user not found for like image")
		return fmt.Errorf("find user: %w", err)
	}

	// 4. Load image
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("image not found for like")
		return fmt.Errorf("find image: %w", err)
	}

	// 5. Check image visibility
	// User can like if:
	// - Image is public, OR
	// - User is the owner
	if image.Visibility() != gallery.VisibilityPublic && !image.IsOwnedBy(userID) {
		h.logger.Debug().
			Str("user_id", userID.String()).
			Str("image_id", imageID.String()).
			Str("visibility", image.Visibility().String()).
			Msg("unauthorized access to like image")
		return gallery.ErrUnauthorizedAccess
	}

	// 6. Check if already liked (idempotent)
	hasLiked, err := h.likes.HasLiked(ctx, userID, imageID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", userID.String()).
			Str("image_id", imageID.String()).
			Msg("failed to check if user has liked image")
		return fmt.Errorf("check has liked: %w", err)
	}

	if hasLiked {
		// Already liked - idempotent operation, return success
		h.logger.Debug().
			Str("user_id", userID.String()).
			Str("image_id", imageID.String()).
			Msg("user has already liked this image")
		return nil
	}

	// 7. Create like relationship
	if err := h.likes.Like(ctx, userID, imageID); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", userID.String()).
			Str("image_id", imageID.String()).
			Msg("failed to create like")
		return fmt.Errorf("create like: %w", err)
	}

	// 8. Update denormalized like count on image
	likeCount, err := h.likes.GetLikeCount(ctx, imageID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to get like count")
		return fmt.Errorf("get like count: %w", err)
	}

	image.SetLikeCount(likeCount)

	if err := h.images.Save(ctx, image); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to update image like count")
		return fmt.Errorf("update image like count: %w", err)
	}

	// 9. Publish ImageLiked domain event
	event := &gallery.ImageLiked{
		BaseEvent: shared.NewBaseEvent("gallery.image.liked", imageID.String()),
		ImageID:   imageID,
		UserID:    userID,
		LikedAt:   time.Now().UTC(),
	}

	if err := h.publisher.Publish(ctx, event); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Str("user_id", userID.String()).
			Msg("failed to publish ImageLiked event")
		// Don't fail the operation - the like was already created
	}

	h.logger.Info().
		Str("user_id", userID.String()).
		Str("image_id", imageID.String()).
		Int64("like_count", likeCount).
		Msg("image liked successfully")

	return nil
}
