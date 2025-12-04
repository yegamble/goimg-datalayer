package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// GetUserLikedImagesQuery represents a request to get images a user has liked (favorites).
type GetUserLikedImagesQuery struct {
	UserID  string
	Page    int
	PerPage int
}

// GetUserLikedImagesResult contains the query result with liked images and pagination info.
type GetUserLikedImagesResult struct {
	Images     []*gallery.Image
	Total      int64
	Pagination shared.Pagination
}

// GetUserLikedImagesHandler processes queries for user's liked images.
type GetUserLikedImagesHandler struct {
	likes  gallery.LikeRepository
	images gallery.ImageRepository
}

// NewGetUserLikedImagesHandler creates a new GetUserLikedImagesHandler.
func NewGetUserLikedImagesHandler(
	likes gallery.LikeRepository,
	images gallery.ImageRepository,
) *GetUserLikedImagesHandler {
	return &GetUserLikedImagesHandler{
		likes:  likes,
		images: images,
	}
}

// Handle executes the get user liked images query.
//
// Process flow:
//  1. Parse and validate user ID
//  2. Create pagination from page/perPage
//  3. Get liked image IDs for the user (ordered by most recent first)
//  4. Load full image entities for each ID
//  5. Return images with pagination info
//
// Returns:
//   - GetUserLikedImagesResult with images and pagination
//   - Validation errors from domain value objects
func (h *GetUserLikedImagesHandler) Handle(
	ctx context.Context,
	q GetUserLikedImagesQuery,
) (*GetUserLikedImagesResult, error) {
	// 1. Parse user ID
	userID, err := identity.ParseUserID(q.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Create pagination
	pagination, err := shared.NewPagination(q.Page, q.PerPage)
	if err != nil {
		// Use default pagination if invalid
		pagination = shared.DefaultPagination()
	}

	// 3. Get liked image IDs
	imageIDs, err := h.likes.GetLikedImageIDs(ctx, userID, pagination)
	if err != nil {
		return nil, fmt.Errorf("get liked image ids: %w", err)
	}

	// 4. Get total count
	total, err := h.likes.CountLikedImagesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count liked images: %w", err)
	}

	// 5. Load full image entities
	images := make([]*gallery.Image, 0, len(imageIDs))
	for _, id := range imageIDs {
		imageID, err := gallery.ParseImageID(id.String())
		if err != nil {
			// Skip invalid IDs (shouldn't happen in practice)
			continue
		}

		image, err := h.images.FindByID(ctx, imageID)
		if err != nil {
			// Skip if image not found (may have been deleted)
			continue
		}

		images = append(images, image)
	}

	// 6. Add total count to pagination
	pagination = pagination.WithTotal(total)

	return &GetUserLikedImagesResult{
		Images:     images,
		Total:      total,
		Pagination: pagination,
	}, nil
}
