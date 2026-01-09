package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// GenerateCustomVariantCommand represents the intent to generate a custom image variant.
type GenerateCustomVariantCommand struct {
	ImageID   string
	UserID    string
	ConfigID  string // Optional: Use a saved variant config
	Name      string // Custom variant name (for storage key)
	MaxWidth  int    // Direct parameters if ConfigID not provided
	MaxHeight int
	Format    string
	Quality   int
	CropMode  string
}

// CustomVariantResult contains the result of custom variant generation.
type CustomVariantResult struct {
	VariantKey  string `json:"variant_key"`
	VariantURL  string `json:"variant_url"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Format      string `json:"format"`
	FileSize    int64  `json:"file_size"`
	ContentType string `json:"content_type"`
}

// ImageProcessor defines the interface for image processing operations.
type ImageProcessor interface {
	// GenerateCustomVariant generates a single custom variant from image data.
	GenerateCustomVariant(
		ctx context.Context,
		input []byte,
		maxWidth, maxHeight int,
		format string,
		quality int,
		cropMode string,
	) (*CustomVariantData, error)
}

// CustomVariantData represents the output of custom variant generation.
type CustomVariantData struct {
	Data        []byte
	Width       int
	Height      int
	Format      string
	ContentType string
	FileSize    int64
}

// StorageProvider defines the interface for storage operations.
type StorageProvider interface {
	GetBytes(ctx context.Context, key string) ([]byte, error)
	PutBytes(ctx context.Context, key string, data []byte, contentType string) error
	URL(key string) string
}

// GenerateCustomVariantHandler processes custom variant generation commands.
type GenerateCustomVariantHandler struct {
	images        gallery.ImageRepository
	variantConfigs gallery.VariantConfigRepository
	storage       StorageProvider
	processor     ImageProcessor
	publisher     EventPublisher
	logger        *zerolog.Logger
}

// NewGenerateCustomVariantHandler creates a new GenerateCustomVariantHandler.
func NewGenerateCustomVariantHandler(
	images gallery.ImageRepository,
	variantConfigs gallery.VariantConfigRepository,
	storage StorageProvider,
	processor ImageProcessor,
	publisher EventPublisher,
	logger *zerolog.Logger,
) *GenerateCustomVariantHandler {
	return &GenerateCustomVariantHandler{
		images:        images,
		variantConfigs: variantConfigs,
		storage:       storage,
		processor:     processor,
		publisher:     publisher,
		logger:        logger,
	}
}

// Handle executes the custom variant generation use case.
//
// Process flow:
//  1. Parse and validate IDs
//  2. Retrieve image from repository
//  3. Verify user ownership or public access
//  4. If ConfigID provided, load and validate variant config
//  5. Retrieve original image from storage
//  6. Generate custom variant using processor
//  7. Store custom variant
//  8. Return result with URL
//
// Returns:
//   - CustomVariantResult on success
//   - ErrImageNotFound if image doesn't exist
//   - Authorization error if user cannot access the image
//   - Processing error if variant generation fails
func (h *GenerateCustomVariantHandler) Handle(ctx context.Context, cmd GenerateCustomVariantCommand) (*CustomVariantResult, error) {
	// 1. Parse image ID
	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", cmd.ImageID).
			Msg("invalid image id for custom variant generation")
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	// 2. Parse user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id for custom variant generation")
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 3. Retrieve image
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("image not found for custom variant generation")
		return nil, fmt.Errorf("find image: %w", err)
	}

	// 4. Verify access (owner only for now - could add public access later)
	if !image.IsOwnedBy(userID) {
		h.logger.Warn().
			Str("image_id", imageID.String()).
			Str("owner_id", image.OwnerID().String()).
			Str("user_id", userID.String()).
			Msg("unauthorized custom variant generation attempt")
		return nil, fmt.Errorf("unauthorized: user does not own this image")
	}

	// 5. Resolve variant parameters
	var maxWidth, maxHeight, quality int
	var format, cropMode, variantName string

	if cmd.ConfigID != "" {
		// Use saved variant config
		configID, err := gallery.ParseVariantConfigID(cmd.ConfigID)
		if err != nil {
			return nil, fmt.Errorf("invalid config id: %w", err)
		}

		config, err := h.variantConfigs.FindByID(ctx, configID)
		if err != nil {
			return nil, fmt.Errorf("find variant config: %w", err)
		}

		// Verify config ownership (user must own config or it must be a preset)
		if !config.IsPreset() && !config.IsOwnedBy(userID) {
			return nil, fmt.Errorf("unauthorized: cannot use this variant config")
		}

		maxWidth = config.MaxWidth()
		maxHeight = config.MaxHeight()
		format = string(config.Format())
		quality = config.Quality()
		cropMode = string(config.CropMode())
		variantName = config.Name()
	} else {
		// Use direct parameters
		maxWidth = cmd.MaxWidth
		maxHeight = cmd.MaxHeight
		format = cmd.Format
		quality = cmd.Quality
		cropMode = cmd.CropMode
		variantName = cmd.Name

		// Validate direct parameters
		if maxWidth < 1 || maxWidth > 8192 {
			return nil, fmt.Errorf("max_width must be between 1 and 8192")
		}
		if maxHeight < 1 || maxHeight > 8192 {
			return nil, fmt.Errorf("max_height must be between 1 and 8192")
		}
		if quality < 1 || quality > 100 {
			return nil, fmt.Errorf("quality must be between 1 and 100")
		}
		if variantName == "" {
			variantName = fmt.Sprintf("custom_%dx%d", maxWidth, maxHeight)
		}
	}

	// 6. Retrieve original image from storage
	originalVariant, err := image.GetVariant(gallery.VariantOriginal)
	if err != nil {
		return nil, fmt.Errorf("image has no original variant: %w", err)
	}
	originalKey := originalVariant.StorageKey()
	if originalKey == "" {
		return nil, fmt.Errorf("image original variant has no storage key")
	}

	originalData, err := h.storage.GetBytes(ctx, originalKey)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Str("storage_key", originalKey).
			Msg("failed to retrieve original image from storage")
		return nil, fmt.Errorf("retrieve original image: %w", err)
	}

	// 7. Generate custom variant
	variantData, err := h.processor.GenerateCustomVariant(
		ctx,
		originalData,
		maxWidth,
		maxHeight,
		format,
		quality,
		cropMode,
	)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Int("max_width", maxWidth).
			Int("max_height", maxHeight).
			Str("format", format).
			Msg("failed to generate custom variant")
		return nil, fmt.Errorf("generate custom variant: %w", err)
	}

	// 8. Store custom variant
	variantKey := fmt.Sprintf("images/%s/custom/%s.%s", imageID.String(), variantName, format)
	if err := h.storage.PutBytes(ctx, variantKey, variantData.Data, variantData.ContentType); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Str("variant_key", variantKey).
			Msg("failed to store custom variant")
		return nil, fmt.Errorf("store custom variant: %w", err)
	}

	// 9. Return result
	result := &CustomVariantResult{
		VariantKey:  variantKey,
		VariantURL:  h.storage.URL(variantKey),
		Width:       variantData.Width,
		Height:      variantData.Height,
		Format:      variantData.Format,
		FileSize:    variantData.FileSize,
		ContentType: variantData.ContentType,
	}

	h.logger.Info().
		Str("image_id", imageID.String()).
		Str("user_id", userID.String()).
		Str("variant_name", variantName).
		Int("width", result.Width).
		Int("height", result.Height).
		Msg("custom variant generated successfully")

	return result, nil
}
