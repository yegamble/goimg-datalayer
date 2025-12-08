package handlers

import (
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
)

// HTTP-specific request DTOs for the handlers layer.
// These DTOs are separate from application-layer DTOs and represent the HTTP contract.
// They include JSON tags and validation rules using go-playground/validator.

// RegisterRequest represents the HTTP request body for user registration.
// POST /api/v1/auth/register.
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Password string `json:"password" validate:"required,min=12,max=128"`
}

// LoginRequest represents the HTTP request body for user login.
// POST /api/v1/auth/login
//
// Identifier can be either email or username.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest represents the HTTP request body for token refresh.
// POST /api/v1/auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest represents the HTTP request body for logout.
// POST /api/v1/auth/logout
//
// Both fields are optional. If neither is provided, logout uses the session from JWT context.
// If logout_all is true, all sessions for the user are revoked.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
	LogoutAll    bool   `json:"logout_all,omitempty"`
}

// UpdateUserRequest represents the HTTP request body for updating user profile.
// PUT /api/v1/users/{id}
//
// All fields are optional (use pointers to indicate "no change").
// Only provided fields will be updated.
type UpdateUserRequest struct {
	DisplayName *string `json:"display_name,omitempty" validate:"omitempty,max=100"`
	Bio         *string `json:"bio,omitempty" validate:"omitempty,max=500"`
}

// DeleteUserRequest represents the HTTP request body for deleting a user account.
// DELETE /api/v1/users/{id}
//
// Requires password confirmation to prevent accidental deletion.
type DeleteUserRequest struct {
	Password string `json:"password" validate:"required"`
}

// ============================================================================
// Gallery DTOs - Image Management
// ============================================================================

// UploadImageRequest represents the HTTP request for image upload.
// POST /api/v1/images
//
// Note: This is used with multipart/form-data. Fields are extracted from the form.
// Validation is performed at the application layer.
type UploadImageRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Visibility  string   `json:"visibility,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// UploadImageResponse represents the HTTP response after image upload.
type UploadImageResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// UpdateImageRequest represents the HTTP request body for updating image metadata.
// PUT /api/v1/images/{id}
//
// All fields are optional. Only provided fields will be updated.
type UpdateImageRequest struct {
	Title       *string  `json:"title,omitempty" validate:"omitempty,max=255"`
	Description *string  `json:"description,omitempty" validate:"omitempty,max=2000"`
	Visibility  *string  `json:"visibility,omitempty" validate:"omitempty,oneof=public private unlisted"`
	Tags        []string `json:"tags,omitempty" validate:"omitempty,dive,max=50"`
}

// PaginatedImagesResponse represents a paginated list of images.
type PaginatedImagesResponse struct {
	Images     []ImageDTO `json:"images"`
	TotalCount int64      `json:"total_count"`
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	HasMore    bool       `json:"has_more"`
}

// Type aliases for application layer DTOs to avoid duplication.
type (
	ImageDTO   = queries.ImageDTO
	AlbumDTO   = queries.AlbumDTO
	VariantDTO = queries.VariantDTO
	TagDTO     = queries.TagDTO
)

// ============================================================================
// Gallery DTOs - Album Management
// ============================================================================

// CreateAlbumRequest represents the HTTP request body for creating an album.
// POST /api/v1/albums.
type CreateAlbumRequest struct {
	Title       string `json:"title" validate:"required,max=255"`
	Description string `json:"description,omitempty" validate:"omitempty,max=2000"`
	Visibility  string `json:"visibility,omitempty" validate:"omitempty,oneof=public private unlisted"`
}

// UpdateAlbumRequest represents the HTTP request body for updating album metadata.
// PUT /api/v1/albums/{id}
//
// All fields are optional. Only provided fields will be updated.
type UpdateAlbumRequest struct {
	Title        *string `json:"title,omitempty" validate:"omitempty,max=255"`
	Description  *string `json:"description,omitempty" validate:"omitempty,max=2000"`
	Visibility   *string `json:"visibility,omitempty" validate:"omitempty,oneof=public private unlisted"`
	CoverImageID *string `json:"cover_image_id,omitempty" validate:"omitempty,uuid"`
}

// AddImageToAlbumRequest represents the HTTP request body for adding an image to an album.
// POST /api/v1/albums/{albumID}/images.
type AddImageToAlbumRequest struct {
	ImageID string `json:"image_id" validate:"required,uuid"`
}

// PaginatedAlbumsResponse represents a paginated list of albums.
type PaginatedAlbumsResponse struct {
	Albums     []AlbumDTO `json:"albums"`
	TotalCount int64      `json:"total_count"`
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	HasMore    bool       `json:"has_more"`
}

// AlbumDTO is imported from application/gallery/queries package.
// It's already defined there as queries.AlbumDTO.

// ============================================================================
// Gallery DTOs - Social Interactions
// ============================================================================

// LikeResponse represents the HTTP response for like/unlike operations.
// POST /api/v1/images/{imageID}/like
// DELETE /api/v1/images/{imageID}/like.
type LikeResponse struct {
	Liked     bool  `json:"liked"`
	LikeCount int64 `json:"like_count"`
}

// AddCommentRequest represents the HTTP request body for adding a comment.
// POST /api/v1/images/{imageID}/comments
type AddCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

// CommentResponse represents the HTTP response after adding a comment.
type CommentResponse struct {
	ID        string `json:"id"`
	ImageID   string `json:"image_id"`
	UserID    string `json:"user_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at,omitempty"`
}

// CommentDTO represents a comment in list responses.
type CommentDTO struct {
	ID        string `json:"id"`
	ImageID   string `json:"image_id"`
	UserID    string `json:"user_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// PaginatedCommentsResponse represents a paginated list of comments.
type PaginatedCommentsResponse struct {
	Comments []CommentDTO `json:"comments"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PerPage  int          `json:"per_page"`
}
