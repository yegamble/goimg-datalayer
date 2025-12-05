package handlers

import (
	"net/http"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// SocialHandler handles social interaction endpoints (likes, comments).
// It delegates to application layer command and query handlers for business logic.
type SocialHandler struct {
	likeImage          *commands.LikeImageHandler
	unlikeImage        *commands.UnlikeImageHandler
	addComment         *commands.AddCommentHandler
	deleteComment      *commands.DeleteCommentHandler
	listImageComments  *queries.ListImageCommentsHandler
	getUserLikedImages *queries.GetUserLikedImagesHandler
	logger             zerolog.Logger
}

// NewSocialHandler creates a new SocialHandler with the given dependencies.
func NewSocialHandler(
	likeImage *commands.LikeImageHandler,
	unlikeImage *commands.UnlikeImageHandler,
	addComment *commands.AddCommentHandler,
	deleteComment *commands.DeleteCommentHandler,
	listImageComments *queries.ListImageCommentsHandler,
	getUserLikedImages *queries.GetUserLikedImagesHandler,
	logger zerolog.Logger,
) *SocialHandler {
	return &SocialHandler{
		likeImage:          likeImage,
		unlikeImage:        unlikeImage,
		addComment:         addComment,
		deleteComment:      deleteComment,
		listImageComments:  listImageComments,
		getUserLikedImages: getUserLikedImages,
		logger:             logger,
	}
}

// LikeImage handles POST /api/v1/images/{imageID}/likes
// Likes an image.
//
// Path parameters:
//   - imageID: UUID of the image to like
//
// Response: 200 OK with success message
// Errors:
//   - 400: Invalid image ID format
//   - 401: Not authenticated
//   - 403: Image is not accessible to user
//   - 404: Image not found
//   - 409: User has already liked this image
//   - 500: Internal server error
func (h *SocialHandler) LikeImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in like handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract image ID from path
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	// 3. Build like command
	cmd := commands.LikeImageCommand{
		UserID:  userCtx.UserID.String(),
		ImageID: imageID,
	}

	// 4. Execute like command
	if err := h.likeImage.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "like image")
		return
	}

	// 5. Return success message
	h.logger.Info().
		Str("image_id", imageID).
		Str("user_id", userCtx.UserID.String()).
		Msg("image liked successfully")

	response := map[string]string{
		"message": "Image liked successfully",
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode like response")
	}
}

// UnlikeImage handles DELETE /api/v1/images/{imageID}/likes
// Removes a like from an image.
//
// Path parameters:
//   - imageID: UUID of the image to unlike
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid image ID format
//   - 401: Not authenticated
//   - 404: Image not found or not liked by user
//   - 500: Internal server error
func (h *SocialHandler) UnlikeImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in unlike handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract image ID from path
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	// 3. Build unlike command
	cmd := commands.UnlikeImageCommand{
		UserID:  userCtx.UserID.String(),
		ImageID: imageID,
	}

	// 4. Execute unlike command
	if err := h.unlikeImage.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "unlike image")
		return
	}

	// 5. Return 204 No Content
	h.logger.Info().
		Str("image_id", imageID).
		Str("user_id", userCtx.UserID.String()).
		Msg("image unliked successfully")

	w.WriteHeader(http.StatusNoContent)
}

// AddComment handles POST /api/v1/images/{imageID}/comments
// Adds a comment to an image.
//
// Path parameters:
//   - imageID: UUID of the image to comment on
//
// Request: AddCommentRequest JSON body
// Response: 201 Created with CommentResponse
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 403: Image is not accessible to user
//   - 404: Image not found
//   - 500: Internal server error
func (h *SocialHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in add comment handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract image ID from path
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	// 3. Decode request body
	var req AddCommentRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid add comment request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid comment data",
			validationErrors,
		)
		return
	}

	// 4. Build add comment command
	cmd := commands.AddCommentCommand{
		UserID:  userCtx.UserID.String(),
		ImageID: imageID,
		Content: req.Content,
	}

	// 5. Execute add comment command
	commentID, err := h.addComment.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "add comment")
		return
	}

	// 6. Return created comment
	h.logger.Info().
		Str("comment_id", commentID).
		Str("image_id", imageID).
		Str("user_id", userCtx.UserID.String()).
		Msg("comment added successfully")

	response := CommentResponse{
		ID:      commentID,
		ImageID: imageID,
		UserID:  userCtx.UserID.String(),
		Content: req.Content,
	}

	if err := EncodeJSON(w, http.StatusCreated, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode add comment response")
	}
}

// DeleteComment handles DELETE /api/v1/comments/{commentID}
// Deletes a comment.
//
// Path parameters:
//   - commentID: UUID of the comment to delete
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid comment ID format
//   - 401: Not authenticated
//   - 403: User is not the comment author (or admin/moderator)
//   - 404: Comment not found
//   - 500: Internal server error
func (h *SocialHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in delete comment handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract comment ID from path
	commentID := GetPathParam(r, "commentID")
	if commentID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing comment ID",
		)
		return
	}

	// 3. Build delete command
	cmd := commands.DeleteCommentCommand{
		UserID:    userCtx.UserID.String(),
		CommentID: commentID,
	}

	// 4. Execute delete command
	if err := h.deleteComment.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "delete comment")
		return
	}

	// 5. Return 204 No Content
	h.logger.Info().
		Str("comment_id", commentID).
		Str("user_id", userCtx.UserID.String()).
		Msg("comment deleted successfully")

	w.WriteHeader(http.StatusNoContent)
}

// ListImageComments handles GET /api/v1/images/{imageID}/comments
// Lists all comments for an image with pagination.
//
// Path parameters:
//   - imageID: UUID of the image
//
// Query parameters:
//   - page: Page number, 1-indexed (default: 1)
//   - per_page: Items per page, max 100 (default: 20)
//   - sort_order: Sort order: newest, oldest (default: oldest)
//
// Response: 200 OK with PaginatedCommentsResponse
// Errors:
//   - 400: Invalid parameters
//   - 404: Image not found
//   - 500: Internal server error
func (h *SocialHandler) ListImageComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract image ID from path
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	// 2. Parse query parameters
	queryParams := r.URL.Query()

	page, err := parseIntParam(queryParams.Get("page"), 1)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(queryParams.Get("per_page"), 20)
	if err != nil || perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	sortOrder := queryParams.Get("sort_order")
	if sortOrder == "" {
		sortOrder = "oldest"
	}

	// 3. Build query
	query := queries.ListImageCommentsQuery{
		ImageID:   imageID,
		Page:      page,
		PerPage:   perPage,
		SortOrder: sortOrder,
	}

	// 4. Execute query
	result, err := h.listImageComments.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list image comments")
		return
	}

	// 5. Convert comments to DTOs
	commentDTOs := make([]CommentDTO, 0, len(result.Comments))
	for _, comment := range result.Comments {
		commentDTOs = append(commentDTOs, CommentDTO{
			ID:        comment.ID().String(),
			ImageID:   comment.ImageID().String(),
			UserID:    comment.UserID().String(),
			Content:   comment.Content(),
			CreatedAt: comment.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	// 6. Return paginated results
	h.logger.Debug().
		Str("image_id", imageID).
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(commentDTOs)).
		Int64("total", result.Total).
		Msg("image comments listed successfully")

	response := PaginatedCommentsResponse{
		Comments: commentDTOs,
		Total:    result.Total,
		Page:     page,
		PerPage:  perPage,
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list comments response")
	}
}

// GetUserLikedImages handles GET /api/v1/users/{userID}/likes
// Retrieves all images liked by a user.
//
// Path parameters:
//   - userID: UUID of the user
//
// Query parameters:
//   - page: Page number, 1-indexed (default: 1)
//   - per_page: Items per page, max 100 (default: 20)
//
// Response: 200 OK with PaginatedImagesResponse
// Errors:
//   - 400: Invalid parameters
//   - 404: User not found
//   - 500: Internal server error
func (h *SocialHandler) GetUserLikedImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user ID from path
	userID := GetPathParam(r, "userID")
	if userID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing user ID",
		)
		return
	}

	// 2. Parse query parameters
	queryParams := r.URL.Query()

	page, err := parseIntParam(queryParams.Get("page"), 1)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(queryParams.Get("per_page"), 20)
	if err != nil || perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	// 3. Build query
	query := queries.GetUserLikedImagesQuery{
		UserID:  userID,
		Page:    page,
		PerPage: perPage,
	}

	// 4. Execute query
	result, err := h.getUserLikedImages.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get user liked images")
		return
	}

	// 5. Convert domain entities to DTOs
	imageDTOs := make([]ImageDTO, 0, len(result.Images))
	for _, img := range result.Images {
		imageDTOs = append(imageDTOs, *queries.ImageToDTO(img))
	}

	// 6. Convert to pagination result format
	offset := (page - 1) * perPage
	hasMore := int64(offset+perPage) < result.Total

	// 7. Return paginated results
	h.logger.Debug().
		Str("user_id", userID).
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(imageDTOs)).
		Int64("total", result.Total).
		Msg("user liked images retrieved successfully")

	response := PaginatedImagesResponse{
		Images:     imageDTOs,
		TotalCount: result.Total,
		Offset:     offset,
		Limit:      perPage,
		HasMore:    hasMore,
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode user liked images response")
	}
}

// mapErrorAndRespond maps application/domain errors to HTTP responses.
func (h *SocialHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("social operation failed")

	// Simplified error mapping - expand based on actual domain errors
	middleware.WriteError(w, r,
		http.StatusInternalServerError,
		"Internal Server Error",
		"An unexpected error occurred",
	)
}
