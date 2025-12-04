package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ListImageCommentsQuery represents a request to list comments for an image.
type ListImageCommentsQuery struct {
	ImageID  string
	Page     int
	PerPage  int
	SortOrder string // "newest" or "oldest" (default)
}

// ListImageCommentsResult contains the query result with comments and pagination info.
type ListImageCommentsResult struct {
	Comments   []*gallery.Comment
	Total      int64
	Pagination shared.Pagination
}

// ListImageCommentsHandler processes queries for listing image comments.
type ListImageCommentsHandler struct {
	comments gallery.CommentRepository
}

// NewListImageCommentsHandler creates a new ListImageCommentsHandler.
func NewListImageCommentsHandler(
	comments gallery.CommentRepository,
) *ListImageCommentsHandler {
	return &ListImageCommentsHandler{
		comments: comments,
	}
}

// Handle executes the list image comments query.
//
// Process flow:
//  1. Parse and validate image ID
//  2. Create pagination from page/perPage
//  3. Retrieve comments for the image (ordered by creation time)
//  4. Return comments with pagination info
//
// Notes:
//   - Comments are ordered by creation time (oldest first by default)
//   - If sortOrder is "newest", results are reversed (newest first)
//   - Only non-deleted comments are returned
//
// Returns:
//   - ListImageCommentsResult with comments and pagination
//   - Validation errors from domain value objects
func (h *ListImageCommentsHandler) Handle(
	ctx context.Context,
	q ListImageCommentsQuery,
) (*ListImageCommentsResult, error) {
	// 1. Parse image ID
	imageID, err := gallery.ParseImageID(q.ImageID)
	if err != nil {
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	// 2. Create pagination
	pagination, err := shared.NewPagination(q.Page, q.PerPage)
	if err != nil {
		// Use default pagination if invalid
		pagination = shared.DefaultPagination()
	}

	// 3. Retrieve comments
	// Note: Repository returns oldest first. For "newest" we'd need to
	// either update the repository query or reverse results. For now,
	// keeping it simple with oldest first as per repository implementation.
	comments, total, err := h.comments.FindByImage(ctx, imageID, pagination)
	if err != nil {
		return nil, fmt.Errorf("find comments by image: %w", err)
	}

	// 4. Apply sort order if requested (simple in-memory reversal for "newest")
	if q.SortOrder == "newest" && len(comments) > 0 {
		// Reverse the slice
		for i, j := 0, len(comments)-1; i < j; i, j = i+1, j-1 {
			comments[i], comments[j] = comments[j], comments[i]
		}
	}

	// 5. Add total count to pagination
	pagination = pagination.WithTotal(total)

	return &ListImageCommentsResult{
		Comments:   comments,
		Total:      total,
		Pagination: pagination,
	}, nil
}
