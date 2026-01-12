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
	// SearchSortByRelevance sorts by search relevance score.
	SearchSortByRelevance SearchSortBy = "relevance"
	// SearchSortByCreatedAt sorts by creation timestamp.
	SearchSortByCreatedAt SearchSortBy = "created_at"
	// SearchSortByViewCount sorts by view count.
	SearchSortByViewCount SearchSortBy = "view_count"
	// SearchSortByLikeCount sorts by like count.
	SearchSortByLikeCount SearchSortBy = "like_count"
)

// NSFWFilter defines how to filter search results based on NSFW content.
type NSFWFilter string

const (
	// NSFWFilterExcludeAll excludes all NSFW content (default for public search).
	NSFWFilterExcludeAll NSFWFilter = "exclude_all"
	// NSFWFilterSafeOnly shows only content classified as safe.
	NSFWFilterSafeOnly NSFWFilter = "safe_only"
	// NSFWFilterIncludeAll includes all content regardless of NSFW status.
	NSFWFilterIncludeAll NSFWFilter = "include_all"
	// NSFWFilterSuggestiveOK allows safe and suggestive content (excludes explicit).
	NSFWFilterSuggestiveOK NSFWFilter = "suggestive_ok"
)

// IsValid returns true if the NSFWFilter is a valid value.
func (f NSFWFilter) IsValid() bool {
	switch f {
	case NSFWFilterExcludeAll, NSFWFilterSafeOnly, NSFWFilterIncludeAll, NSFWFilterSuggestiveOK:
		return true
	default:
		return false
	}
}

// SearchParams encapsulates all search criteria for image queries.
type SearchParams struct {
	Query      string            // Full-text search query (searches title and description)
	Tags       []Tag             // Filter by tags (AND logic for multiple tags)
	OwnerID    *identity.UserID  // Optional: filter by owner
	Visibility *Visibility       // Optional: filter by visibility (defaults to public only)
	NSFWFilter *NSFWFilter       // Optional: NSFW content filter (defaults to ExcludeAll for public)
	SortBy     SearchSortBy      // Sort order (defaults to relevance)
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

	// FindChildren retrieves all direct child albums of a parent album.
	// Used for hierarchical album navigation.
	FindChildren(ctx context.Context, parentID AlbumID) ([]*Album, error)

	// FindRootAlbumsByOwner retrieves all root albums (no parent) for a user.
	// Used for displaying top-level album structure.
	FindRootAlbumsByOwner(ctx context.Context, ownerID identity.UserID) ([]*Album, error)

	// FindAncestors retrieves the breadcrumb path from root to the given album.
	// Returns albums ordered from root to the target album (inclusive).
	// Returns ErrAlbumNotFound if the album doesn't exist.
	FindAncestors(ctx context.Context, albumID AlbumID) ([]*Album, error)

	// Save persists an album (insert or update).
	Save(ctx context.Context, album *Album) error

	// Delete permanently removes an album.
	Delete(ctx context.Context, id AlbumID) error

	// ExistsByID checks if an album exists.
	ExistsByID(ctx context.Context, id AlbumID) (bool, error)
}

// VariantConfigRepository defines the interface for persisting variant configurations.
// Implementations reside in the infrastructure layer.
type VariantConfigRepository interface {
	// NextID generates a new unique VariantConfigID.
	NextID() VariantConfigID

	// FindByID retrieves a variant config by its ID.
	// Returns ErrVariantConfigNotFound if the config doesn't exist.
	FindByID(ctx context.Context, id VariantConfigID) (*VariantConfig, error)

	// FindByUser retrieves all variant configs for a user.
	FindByUser(ctx context.Context, userID identity.UserID) ([]*VariantConfig, error)

	// FindByUserAndName retrieves a specific variant config by user and name.
	// Returns ErrVariantConfigNotFound if not found.
	FindByUserAndName(ctx context.Context, userID identity.UserID, name string) (*VariantConfig, error)

	// FindPresets retrieves all system-defined variant presets.
	FindPresets(ctx context.Context) ([]*VariantConfig, error)

	// Save persists a variant config (insert or update).
	Save(ctx context.Context, config *VariantConfig) error

	// Delete permanently removes a variant config.
	Delete(ctx context.Context, id VariantConfigID) error

	// ExistsByID checks if a variant config exists.
	ExistsByID(ctx context.Context, id VariantConfigID) (bool, error)
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

// TagPeriod defines the time period for tag popularity calculations.
type TagPeriod string

const (
	// TagPeriodDay calculates popularity over the last 24 hours.
	TagPeriodDay TagPeriod = "day"
	// TagPeriodWeek calculates popularity over the last 7 days.
	TagPeriodWeek TagPeriod = "week"
	// TagPeriodMonth calculates popularity over the last 30 days.
	TagPeriodMonth TagPeriod = "month"
	// TagPeriodAll calculates popularity over all time.
	TagPeriodAll TagPeriod = "all"
)

// IsValid returns true if the TagPeriod is a valid value.
func (p TagPeriod) IsValid() bool {
	switch p {
	case TagPeriodDay, TagPeriodWeek, TagPeriodMonth, TagPeriodAll:
		return true
	default:
		return false
	}
}

// TagWithUsage represents a tag with its usage statistics.
type TagWithUsage struct {
	Tag
	UsageCount int64   // Total usage count
	TrendScore float64 // Time-weighted trending score (optional)
}

// TagRepository defines the interface for persisting and retrieving tags.
// Implementations reside in the infrastructure layer.
type TagRepository interface {
	// FindBySlug retrieves a tag by its URL-friendly slug.
	// Returns ErrTagNotFound if the tag doesn't exist.
	FindBySlug(ctx context.Context, slug string) (*Tag, error)

	// FindByName retrieves a tag by its display name.
	// Returns ErrTagNotFound if the tag doesn't exist.
	FindByName(ctx context.Context, name string) (*Tag, error)

	// FindPopular retrieves the most popular tags by usage count.
	// limit: maximum number of tags to return (1-100)
	// period: time period for counting (all = total usage count)
	FindPopular(ctx context.Context, limit int, period TagPeriod) ([]TagWithUsage, error)

	// FindTrending retrieves trending tags with time-weighted popularity.
	// Uses algorithm: (recent_count / total_count) * time_decay_factor
	// limit: maximum number of tags to return (1-100)
	// period: time window for trending calculation
	FindTrending(ctx context.Context, limit int, period TagPeriod) ([]TagWithUsage, error)

	// SearchByPrefix finds tags matching a name prefix for autocomplete.
	// query: prefix to search for (min 2 chars)
	// limit: maximum number of tags to return (1-50)
	SearchByPrefix(ctx context.Context, query string, limit int) ([]TagWithUsage, error)

	// Save persists a tag (insert or update).
	// If the tag already exists (by slug), updates the usage count.
	Save(ctx context.Context, tag Tag) error

	// GetOrCreate retrieves a tag by name or creates it if it doesn't exist.
	// Returns the tag (possibly newly created) and any error.
	GetOrCreate(ctx context.Context, name string) (*Tag, error)

	// IncrementUsage increments the usage count for a tag by 1.
	// Used when a tag is added to an image.
	IncrementUsage(ctx context.Context, tagSlug string) error

	// DecrementUsage decrements the usage count for a tag by 1.
	// Used when a tag is removed from an image.
	// Usage count cannot go below 0.
	DecrementUsage(ctx context.Context, tagSlug string) error

	// ExistsBySlug checks if a tag exists by slug.
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
}

// FeaturedPickRepository defines the interface for managing featured image picks.
// Featured picks are admin-curated images displayed on the explore page.
type FeaturedPickRepository interface {
	// FindByID retrieves a featured pick by its ID.
	// Returns ErrFeaturedPickNotFound if the pick doesn't exist.
	FindByID(ctx context.Context, id FeaturedPickID) (*FeaturedPick, error)

	// FindByImageID retrieves the active featured pick for an image.
	// Returns ErrFeaturedPickNotFound if the image is not currently featured.
	FindByImageID(ctx context.Context, imageID ImageID) (*FeaturedPick, error)

	// ListActive retrieves currently active featured picks.
	// Active means: featured_from <= NOW() AND (featured_until IS NULL OR featured_until > NOW())
	// Results are ordered by display_order ASC, featured_from DESC.
	// limit: maximum number of picks to return (1-100)
	ListActive(ctx context.Context, limit int) ([]*FeaturedPick, error)

	// ListAll retrieves all featured picks (active and expired) for admin management.
	// includeExpired: if true, includes picks with featured_until < NOW()
	// offset/limit: pagination parameters
	ListAll(ctx context.Context, includeExpired bool, offset, limit int) ([]*FeaturedPick, int, error)

	// Save persists a featured pick (insert or update).
	Save(ctx context.Context, pick *FeaturedPick) error

	// Delete removes a featured pick.
	Delete(ctx context.Context, id FeaturedPickID) error

	// ExistsByImageID checks if an image is currently featured.
	ExistsByImageID(ctx context.Context, imageID ImageID) (bool, error)
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
