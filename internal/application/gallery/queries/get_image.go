package queries

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// GetImageQuery retrieves a single image by its unique ID.
// This is a read-only operation with visibility checks and optional view counting.
type GetImageQuery struct {
	ImageID           string
	RequestingUserID  string // Optional: ID of the user requesting (for authorization)
	IncrementViewOnly bool   // If true, only increments view count, no authorization check
}

// Implement Query interface
func (GetImageQuery) isQuery() {}

// ImageDTO represents the data transfer object for an image.
// It contains all public information about an image.
type ImageDTO struct {
	ID               string        `json:"id"`
	OwnerID          string        `json:"owner_id"`
	Title            string        `json:"title"`
	Description      string        `json:"description"`
	OriginalFilename string        `json:"original_filename"`
	MimeType         string        `json:"mime_type"`
	Width            int           `json:"width"`
	Height           int           `json:"height"`
	FileSize         int64         `json:"file_size"`
	StorageKey       string        `json:"storage_key"`
	StorageProvider  string        `json:"storage_provider"`
	Visibility       string        `json:"visibility"`
	Status           string        `json:"status"`
	Variants         []VariantDTO  `json:"variants"`
	Tags             []TagDTO      `json:"tags"`
	ViewCount        int64         `json:"view_count"`
	LikeCount        int64         `json:"like_count"`
	CommentCount     int64         `json:"comment_count"`
	CreatedAt        string        `json:"created_at"`
	UpdatedAt        string        `json:"updated_at"`
}

// VariantDTO represents an image variant (thumbnail, optimized, etc.).
type VariantDTO struct {
	Type       string `json:"type"`
	StorageKey string `json:"storage_key"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	FileSize   int64  `json:"file_size"`
	Format     string `json:"format"`
}

// TagDTO represents a tag attached to an image.
type TagDTO struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// GetImageHandler processes GetImageQuery requests.
// It retrieves an image, enforces visibility rules, and optionally increments view count.
type GetImageHandler struct {
	images gallery.ImageRepository
	logger *zerolog.Logger
}

// NewGetImageHandler creates a new GetImageHandler with the given dependencies.
func NewGetImageHandler(
	images gallery.ImageRepository,
	logger *zerolog.Logger,
) *GetImageHandler {
	return &GetImageHandler{
		images: images,
		logger: logger,
	}
}

// Handle executes the GetImageQuery and returns the image data.
//
// Process flow:
//  1. Parse and validate image ID
//  2. Load image from repository
//  3. Check visibility and authorization
//  4. Increment view count if appropriate
//  5. Convert to DTO and return
//
// Returns:
//   - *ImageDTO: The image data with variants and tags
//   - ErrImageNotFound: If the image doesn't exist
//   - ErrUnauthorizedAccess: If user lacks permission to view
func (h *GetImageHandler) Handle(ctx context.Context, q GetImageQuery) (*ImageDTO, error) {
	// 1. Parse and validate image ID
	imageID, err := gallery.ParseImageID(q.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", q.ImageID).
			Msg("invalid image id during get")
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	// Parse requesting user ID (optional)
	var requestingUserID identity.UserID
	if q.RequestingUserID != "" {
		requestingUserID, err = identity.ParseUserID(q.RequestingUserID)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("requesting_user_id", q.RequestingUserID).
				Msg("invalid requesting user id")
			return nil, fmt.Errorf("invalid requesting user id: %w", err)
		}
	}

	// 2. Load image from repository
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		if errors.Is(err, gallery.ErrImageNotFound) {
			h.logger.Debug().
				Str("image_id", imageID.String()).
				Msg("image not found")
			return nil, fmt.Errorf("find image: %w", err)
		}
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to load image")
		return nil, fmt.Errorf("find image: %w", err)
	}

	// 3. Check visibility and authorization
	if !q.IncrementViewOnly {
		// Check if image is viewable based on status
		if !image.IsViewable() {
			h.logger.Debug().
				Str("image_id", imageID.String()).
				Str("status", image.Status().String()).
				Msg("image not viewable")
			return nil, gallery.ErrImageNotFound // Don't leak that it exists
		}

		// Check visibility rules
		visibility := image.Visibility()
		isOwner := !requestingUserID.IsZero() && image.IsOwnedBy(requestingUserID)

		// Private images can only be viewed by owner
		if visibility.IsPrivate() && !isOwner {
			h.logger.Debug().
				Str("image_id", imageID.String()).
				Str("visibility", visibility.String()).
				Bool("is_owner", isOwner).
				Msg("unauthorized access to private image")
			return nil, gallery.ErrUnauthorizedAccess
		}
	}

	// 4. Increment view count if appropriate
	// Only increment for public/unlisted images, and only if not the owner viewing
	shouldIncrementViews := q.IncrementViewOnly ||
		(!requestingUserID.IsZero() && !image.IsOwnedBy(requestingUserID) &&
			(image.Visibility().IsPublic() || image.Visibility() == gallery.VisibilityUnlisted))

	if shouldIncrementViews {
		image.IncrementViews()
		// Save updated view count asynchronously (don't block on this)
		if err := h.images.Save(ctx, image); err != nil {
			h.logger.Error().
				Err(err).
				Str("image_id", imageID.String()).
				Msg("failed to save view count increment")
			// Don't fail the request if view count save fails
		}
	}

	// 5. Convert to DTO and return
	dto := imageToDTO(image)

	h.logger.Debug().
		Str("image_id", imageID.String()).
		Str("requesting_user_id", q.RequestingUserID).
		Bool("incremented_views", shouldIncrementViews).
		Msg("image retrieved successfully")

	return dto, nil
}

// imageToDTO converts a domain Image to an ImageDTO.
func imageToDTO(image *gallery.Image) *ImageDTO {
	metadata := image.Metadata()

	// Convert variants
	variants := make([]VariantDTO, 0, len(image.Variants()))
	for _, v := range image.Variants() {
		variants = append(variants, VariantDTO{
			Type:       v.VariantType().String(),
			StorageKey: v.StorageKey(),
			Width:      v.Width(),
			Height:     v.Height(),
			FileSize:   v.FileSize(),
			Format:     v.Format(),
		})
	}

	// Convert tags
	tags := make([]TagDTO, 0, len(image.Tags()))
	for _, t := range image.Tags() {
		tags = append(tags, TagDTO{
			Name: t.Name(),
			Slug: t.Slug(),
		})
	}

	return &ImageDTO{
		ID:               image.ID().String(),
		OwnerID:          image.OwnerID().String(),
		Title:            metadata.Title(),
		Description:      metadata.Description(),
		OriginalFilename: metadata.OriginalFilename(),
		MimeType:         metadata.MimeType(),
		Width:            metadata.Width(),
		Height:           metadata.Height(),
		FileSize:         metadata.FileSize(),
		StorageKey:       metadata.StorageKey(),
		StorageProvider:  metadata.StorageProvider(),
		Visibility:       image.Visibility().String(),
		Status:           image.Status().String(),
		Variants:         variants,
		Tags:             tags,
		ViewCount:        image.ViewCount(),
		LikeCount:        image.LikeCount(),
		CommentCount:     image.CommentCount(),
		CreatedAt:        image.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        image.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}
