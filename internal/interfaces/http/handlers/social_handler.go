package handlers

import (
	"net/http"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

type SocialHandler struct {
	likeImage          *commands.LikeImageHandler
	unlikeImage        *commands.UnlikeImageHandler
	addComment         *commands.AddCommentHandler
	deleteComment      *commands.DeleteCommentHandler
	listImageComments  *queries.ListImageCommentsHandler
	getUserLikedImages *queries.GetUserLikedImagesHandler
	logger             zerolog.Logger
}

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

//nolint:dupl // Standard authenticated handler pattern - duplication is intentional for clarity
func (h *SocialHandler) LikeImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in like handler")
		middleware.WriteError(
			w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	cmd := commands.LikeImageCommand{
		UserID:  userCtx.UserID.String(),
		ImageID: imageID,
	}

	result, err := h.likeImage.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "like image")
		return
	}

	h.logger.Info().
		Str("image_id", imageID).
		Str("user_id", userCtx.UserID.String()).
		Int64("like_count", result.LikeCount).
		Msg("image liked successfully")

	response := LikeResponse{
		Liked:     result.Liked,
		LikeCount: result.LikeCount,
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode like response")
	}
}

//nolint:dupl // Standard authenticated handler pattern - duplication is intentional for clarity
func (h *SocialHandler) UnlikeImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in unlike handler")
		middleware.WriteError(
			w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	cmd := commands.UnlikeImageCommand{
		UserID:  userCtx.UserID.String(),
		ImageID: imageID,
	}

	result, err := h.unlikeImage.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "unlike image")
		return
	}

	h.logger.Info().
		Str("image_id", imageID).
		Str("user_id", userCtx.UserID.String()).
		Int64("like_count", result.LikeCount).
		Msg("image unliked successfully")

	response := LikeResponse{
		Liked:     result.Liked,
		LikeCount: result.LikeCount,
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode unlike response")
	}
}

//nolint:funlen // HTTP handler with validation and response.
func (h *SocialHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in add comment handler")
		middleware.WriteError(
			w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	if !userCtx.EmailVerified {
		middleware.WriteError(
			w, r,
			http.StatusForbidden,
			"Forbidden",
			"Email verification required to post comments",
		)
		return
	}

	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	var req AddCommentRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid add comment request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(
			w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid comment data",
			validationErrors,
		)
		return
	}

	cmd := commands.AddCommentCommand{
		UserID:  userCtx.UserID.String(),
		ImageID: imageID,
		Content: req.Content,
	}

	commentID, err := h.addComment.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "add comment")
		return
	}

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

//nolint:dupl // Standard authenticated handler pattern - duplication is intentional for clarity
func (h *SocialHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in delete comment handler")
		middleware.WriteError(
			w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	commentID := GetPathParam(r, "commentID")
	if commentID == "" {
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing comment ID",
		)
		return
	}

	cmd := commands.DeleteCommentCommand{
		UserID:    userCtx.UserID.String(),
		CommentID: commentID,
	}

	if err := h.deleteComment.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "delete comment")
		return
	}

	h.logger.Info().
		Str("comment_id", commentID).
		Str("user_id", userCtx.UserID.String()).
		Msg("comment deleted successfully")

	w.WriteHeader(http.StatusNoContent)
}

//nolint:funlen,cyclop // HTTP handler with pagination, validation, and response mapping.
func (h *SocialHandler) ListImageComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	queryParams := r.URL.Query()

	page, err := parseIntParam(queryParams.Get("page"), 1)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(queryParams.Get("per_page"), defaultPerPage)
	if err != nil || perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	sortOrder := queryParams.Get("sort_order")
	if sortOrder == "" {
		sortOrder = "oldest"
	}

	query := queries.ListImageCommentsQuery{
		ImageID:   imageID,
		Page:      page,
		PerPage:   perPage,
		SortOrder: sortOrder,
	}

	result, err := h.listImageComments.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list image comments")
		return
	}

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

//nolint:funlen // HTTP handler with validation and response.
func (h *SocialHandler) GetUserLikedImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := GetPathParam(r, "userID")
	if userID == "" {
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing user ID",
		)
		return
	}

	queryParams := r.URL.Query()

	page, err := parseIntParam(queryParams.Get("page"), 1)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(queryParams.Get("per_page"), defaultPerPage)
	if err != nil || perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	query := queries.GetUserLikedImagesQuery{
		UserID:  userID,
		Page:    page,
		PerPage: perPage,
	}

	result, err := h.getUserLikedImages.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get user liked images")
		return
	}

	imageDTOs := make([]ImageDTO, 0, len(result.Images))
	for _, img := range result.Images {
		imageDTOs = append(imageDTOs, *queries.ImageToDTO(img))
	}

	offset := (page - 1) * perPage
	hasMore := int64(offset+perPage) < result.Total

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

func (h *SocialHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("social operation failed")

	middleware.WriteError(
		w, r,
		http.StatusInternalServerError,
		"Internal Server Error",
		"An unexpected error occurred",
	)
}
