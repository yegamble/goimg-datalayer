// Package gallery implements the Gallery bounded context for the goimg-datalayer application.
//
// The Gallery context is the largest and most central bounded context in the system,
// responsible for managing all aspects of image storage, organization, and presentation.
//
// # Aggregates
//
// Image is the primary aggregate root, managing:
//   - Image metadata (title, description, dimensions, file info)
//   - Multiple variants (thumbnail, small, medium, large, original)
//   - Tags for categorization and search
//   - Visibility settings (public, private, unlisted)
//   - Status tracking (processing, active, deleted, flagged)
//   - Engagement metrics (views, likes, comments)
//
// Album is an entity for organizing images into collections with:
//   - Title and description
//   - Visibility settings
//   - Optional cover image
//   - Image count tracking
//
// Comment is an entity for user discussions on images with:
//   - Immutable content (no editing after creation)
//   - Author and timestamp tracking
//
// # Value Objects
//
//   - ImageID, AlbumID, CommentID: Type-safe UUID-based identifiers
//   - ImageMetadata: Immutable metadata with strict validation
//   - ImageVariant: Represents processed image sizes
//   - Tag: Normalized keywords with slug generation
//   - Visibility: Access control (public, private, unlisted)
//   - ImageStatus: Lifecycle state (processing, active, deleted, flagged)
//   - VariantType: Predefined image sizes with max dimensions
//
// # Repository Interfaces
//
//   - ImageRepository: CRUD and queries for images
//   - AlbumRepository: CRUD and queries for albums
//   - CommentRepository: CRUD and queries for comments
//   - AlbumImageRepository: Many-to-many relationship management
//
// # Domain Events
//
// All state changes emit domain events for:
//   - Event sourcing and audit trails
//   - Cross-context integration (moderation, social)
//   - Asynchronous processing (notifications, indexing)
//
// # Business Rules
//
//   - Images start in processing status with private visibility
//   - Maximum 20 tags per image
//   - Tags are normalized to lowercase with URL-safe slugs
//   - Deleted images cannot be modified
//   - Flagged images cannot be deleted until reviewed
//   - File size limit: 10MB
//   - Dimension limit: 8192x8192 pixels
//   - Total pixel limit: 100 million
//   - Supported formats: JPEG, PNG, GIF, WebP
//
// # Design Principles
//
// This bounded context follows Domain-Driven Design principles:
//   - All modifications go through aggregate roots
//   - Value objects are immutable
//   - Entities validate invariants in constructors
//   - No infrastructure dependencies (pure domain logic)
//   - Repository interfaces defined here, implemented in infrastructure
//
// # Integration Points
//
// The Gallery context integrates with:
//   - Identity context: Owner relationships via UserID
//   - Moderation context: Flagged images, content review
//   - Social context: Likes and engagement metrics
//   - Storage infrastructure: File persistence (S3, IPFS, local)
//   - Image processing: Variant generation via bimg/libvips
package gallery
