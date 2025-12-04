package gallery

import (
	"context"

	"github.com/google/uuid"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SearchSortBy defines the sort order for search results.
type SearchSortBy string

const (
	SearchSortByRelevance  SearchSortBy = "relevance"
	SearchSortByCreatedAt  SearchSortBy = "created_at"
	SearchSortByViewCount  SearchSortBy = "view_count"
	SearchSortByLikeCount  SearchSortBy = "like_count"
)

// SearchParams encapsulates all search criteria for image queries.
type SearchParams struct {
	Query      string          // Full-text search query (searches title and description)
	Tags       []Tag           // Filter by tags (AND logic for multiple tags)
	OwnerID    *identity.UserID // Optional: filter by owner
	Visibility *Visibility     // Optional: filter by visibility (defaults to public only)
	SortBy     SearchSortBy    // Sort order (defaults to relevance)
	Pagination shared.Pagination // Pagination parameters
}

// ImageRepository defines the interface for persisting and retrieving images.
// Implementations reside in the infrastructure layer.
type ImageRepository interface {
	// NextID generates a new unique ImageID.
	// This is used by the application layer to create new images.
	NextID() ImageID

	// FindByID retrieves an image by its ID.
	// Returns ErrImageNotFound if the image doesn't exist.
	FindByID(ctx context.Context, id ImageID) (*Image, error)

	// FindByOwner retrieves all images owned by a user with pagination.
	// Returns the images, total count, and any error.
	FindByOwner(ctx context.Context, ownerID identity.UserID, pagination shared.Pagination) ([]*Image, int64, error)

	// FindPublic retrieves all public images with pagination.
	// Only returns images with VisibilityPublic and StatusActive.
	FindPublic(ctx context.Context, pagination shared.Pagination) ([]*Image, int64, error)

	// FindByTag retrieves all public images with a specific tag, with pagination.
	// Only returns images with VisibilityPublic and StatusActive.
	FindByTag(ctx context.Context, tag Tag, pagination shared.Pagination) ([]*Image, int64, error)

	// FindByStatus retrieves images by status with pagination.
	// Used for administrative tasks like finding processing or flagged images.
	FindByStatus(ctx context.Context, status ImageStatus, pagination shared.Pagination) ([]*Image, int64, error)

	// Search performs a full-text search on images with filters and sorting.
	// Returns images matching the search criteria, total count, and any error.
	Search(ctx context.Context, params SearchParams) ([]*Image, int64, error)

	// Save persists an image (insert or update).
	// The repository is responsible for detecting whether to insert or update.
	Save(ctx context.Context, image *Image) error

	// Delete permanently removes an image.
	// This is different from MarkAsDeleted, which is a soft delete.
	Delete(ctx context.Context, id ImageID) error

	// ExistsByID checks if an image exists.
	ExistsByID(ctx context.Context, id ImageID) (bool, error)
}

// AlbumRepository defines the interface for persisting and retrieving albums.
// Implementations reside in the infrastructure layer.
type AlbumRepository interface {
	// NextID generates a new unique AlbumID.
	NextID() AlbumID

	// FindByID retrieves an album by its ID.
	// Returns ErrAlbumNotFound if the album doesn't exist.
	FindByID(ctx context.Context, id AlbumID) (*Album, error)

	// FindByOwner retrieves all albums owned by a user.
	// Albums are typically not paginated as users don't usually have many.
	FindByOwner(ctx context.Context, ownerID identity.UserID) ([]*Album, error)

	// FindPublic retrieves all public albums with pagination.
	// Only returns albums with VisibilityPublic.
	FindPublic(ctx context.Context, pagination shared.Pagination) ([]*Album, int64, error)

	// Save persists an album (insert or update).
	Save(ctx context.Context, album *Album) error

	// Delete permanently removes an album.
	Delete(ctx context.Context, id AlbumID) error

	// ExistsByID checks if an album exists.
	ExistsByID(ctx context.Context, id AlbumID) (bool, error)
}

// CommentRepository defines the interface for persisting and retrieving comments.
// Implementations reside in the infrastructure layer.
type CommentRepository interface {
	// NextID generates a new unique CommentID.
	NextID() CommentID

	// FindByID retrieves a comment by its ID.
	// Returns ErrCommentNotFound if the comment doesn't exist.
	FindByID(ctx context.Context, id CommentID) (*Comment, error)

	// FindByImage retrieves all comments for an image with pagination.
	// Returns the comments, total count, and any error.
	// Comments are ordered by creation time (oldest first).
	FindByImage(ctx context.Context, imageID ImageID, pagination shared.Pagination) ([]*Comment, int64, error)

	// FindByUser retrieves all comments by a user with pagination.
	// Used for user profile pages.
	FindByUser(ctx context.Context, userID identity.UserID, pagination shared.Pagination) ([]*Comment, int64, error)

	// CountByImage returns the number of comments on an image.
	// Used to update the comment count on the image entity.
	CountByImage(ctx context.Context, imageID ImageID) (int64, error)

	// Save persists a comment (insert only - comments are immutable).
	Save(ctx context.Context, comment *Comment) error

	// Delete permanently removes a comment.
	Delete(ctx context.Context, id CommentID) error

	// ExistsByID checks if a comment exists.
	ExistsByID(ctx context.Context, id CommentID) (bool, error)
}

// AlbumImageRepository defines the interface for the many-to-many relationship
// between albums and images. This is a separate repository because it represents
// a relationship, not an aggregate.
type AlbumImageRepository interface {
	// AddImageToAlbum adds an image to an album.
	// Returns an error if the image is already in the album.
	AddImageToAlbum(ctx context.Context, albumID AlbumID, imageID ImageID) error

	// RemoveImageFromAlbum removes an image from an album.
	// Returns no error if the image wasn't in the album (idempotent).
	RemoveImageFromAlbum(ctx context.Context, albumID AlbumID, imageID ImageID) error

	// FindImagesInAlbum retrieves all images in an album with pagination.
	// Returns the images, total count, and any error.
	FindImagesInAlbum(ctx context.Context, albumID AlbumID, pagination shared.Pagination) ([]*Image, int64, error)

	// FindAlbumsForImage retrieves all albums containing an image.
	// Used to show which albums an image is in.
	FindAlbumsForImage(ctx context.Context, imageID ImageID) ([]*Album, error)

	// IsImageInAlbum checks if an image is in an album.
	IsImageInAlbum(ctx context.Context, albumID AlbumID, imageID ImageID) (bool, error)

	// CountImagesInAlbum returns the number of images in an album.
	// Used to update the image count on the album entity.
	CountImagesInAlbum(ctx context.Context, albumID AlbumID) (int, error)
}

// LikeRepository defines the interface for managing image likes.
// Likes represent a many-to-many relationship between users and images.
type LikeRepository interface {
	// Like creates a like relationship between a user and an image.
	// Returns nil if the like already exists (idempotent).
	Like(ctx context.Context, userID identity.UserID, imageID ImageID) error

	// Unlike removes a like relationship between a user and an image.
	// Returns nil if the like doesn't exist (idempotent).
	Unlike(ctx context.Context, userID identity.UserID, imageID ImageID) error

	// HasLiked checks if a user has liked an image.
	HasLiked(ctx context.Context, userID identity.UserID, imageID ImageID) (bool, error)

	// GetLikeCount returns the total number of likes for an image.
	GetLikeCount(ctx context.Context, imageID ImageID) (int64, error)

	// GetLikedImageIDs returns a paginated list of image IDs that a user has liked.
	// Results are ordered by most recently liked first.
	GetLikedImageIDs(ctx context.Context, userID identity.UserID, pagination shared.Pagination) ([]uuid.UUID, error)

	// CountLikedImagesByUser returns the total number of images a user has liked.
	CountLikedImagesByUser(ctx context.Context, userID identity.UserID) (int64, error)
}
