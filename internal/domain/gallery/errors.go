package gallery

import "errors"

// Domain errors for the gallery bounded context.
// These errors represent business rule violations and validation failures.
// Use fmt.Errorf("operation: %w", err) to wrap with additional context.
var (
	// Entity not found errors
	ErrImageNotFound   = errors.New("image not found")
	ErrAlbumNotFound   = errors.New("album not found")
	ErrCommentNotFound = errors.New("comment not found")

	// Image lifecycle errors
	ErrImageDeleted    = errors.New("image has been deleted")
	ErrImageProcessing = errors.New("image is still processing")
	ErrImageFlagged    = errors.New("image has been flagged for moderation")

	// Validation errors - Metadata
	ErrInvalidMetadata    = errors.New("invalid image metadata")
	ErrTitleTooLong       = errors.New("title exceeds 255 characters")
	ErrDescriptionTooLong = errors.New("description exceeds 2000 characters")
	ErrInvalidMimeType    = errors.New("unsupported image format")
	ErrInvalidDimensions  = errors.New("invalid image dimensions")
	ErrFileTooLarge       = errors.New("file exceeds 10MB limit")
	ErrImageTooLarge      = errors.New("image dimensions exceed 8192x8192 limit")
	ErrImageTooManyPixels = errors.New("image exceeds 100 million pixel limit")
	ErrStorageKeyRequired = errors.New("storage key is required")
	ErrProviderRequired   = errors.New("storage provider is required")

	// Validation errors - Variant
	ErrVariantExists      = errors.New("variant already exists")
	ErrVariantNotFound    = errors.New("variant not found")
	ErrInvalidVariantType = errors.New("invalid variant type")
	ErrInvalidVariantData = errors.New("invalid variant data")

	// Validation errors - Tag
	ErrTagInvalid       = errors.New("invalid tag format")
	ErrTagTooShort      = errors.New("tag must be at least 2 characters")
	ErrTagTooLong       = errors.New("tag exceeds 50 characters")
	ErrTooManyTags      = errors.New("maximum 20 tags allowed")
	ErrTagAlreadyExists = errors.New("tag already exists on image")

	// Validation errors - Visibility and Status
	ErrInvalidVisibility  = errors.New("invalid visibility value")
	ErrInvalidImageStatus = errors.New("invalid image status")

	// Album validation errors
	ErrAlbumTitleRequired = errors.New("album title is required")
	ErrAlbumTitleTooLong  = errors.New("album title exceeds 255 characters")
	ErrAlbumDescTooLong   = errors.New("album description exceeds 2000 characters")

	// Comment validation errors
	ErrCommentRequired = errors.New("comment content is required")
	ErrCommentTooLong  = errors.New("comment exceeds 1000 characters")

	// Business rule violations
	ErrUnauthorizedAccess  = errors.New("unauthorized to access this resource")
	ErrCannotModifyDeleted = errors.New("cannot modify deleted image")
	ErrCannotDeleteFlagged = errors.New("cannot delete flagged image")
)
