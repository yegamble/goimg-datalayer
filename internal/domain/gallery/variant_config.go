package gallery

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// VariantConfigID is the unique identifier for a variant configuration.
type VariantConfigID struct {
	value uuid.UUID
}

// NewVariantConfigID generates a new unique VariantConfigID.
func NewVariantConfigID() VariantConfigID {
	return VariantConfigID{value: uuid.New()}
}

// ParseVariantConfigID parses a string into a VariantConfigID.
func ParseVariantConfigID(s string) (VariantConfigID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return VariantConfigID{}, fmt.Errorf("invalid variant config ID: %w", err)
	}
	return VariantConfigID{value: id}, nil
}

// String returns the string representation of the ID.
func (id VariantConfigID) String() string {
	return id.value.String()
}

// IsZero returns true if the ID is empty.
func (id VariantConfigID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if two IDs are equal.
func (id VariantConfigID) Equals(other VariantConfigID) bool {
	return id.value == other.value
}

// UUID returns the underlying UUID value.
func (id VariantConfigID) UUID() uuid.UUID {
	return id.value
}

// CropMode defines how images are cropped when resizing.
type CropMode string

const (
	// CropModeFit scales image to fit within dimensions, preserving aspect ratio.
	CropModeFit CropMode = "fit"
	// CropModeFill scales image to fill dimensions, cropping excess.
	CropModeFill CropMode = "fill"
	// CropModeCrop crops the image to exact dimensions from center.
	CropModeCrop CropMode = "crop"
)

// IsValid returns true if the crop mode is valid.
func (m CropMode) IsValid() bool {
	switch m {
	case CropModeFit, CropModeFill, CropModeCrop:
		return true
	}
	return false
}

// OutputFormat defines the output image format.
type OutputFormat string

const (
	FormatJPEG OutputFormat = "jpeg"
	FormatPNG  OutputFormat = "png"
	FormatWebP OutputFormat = "webp"
	FormatAVIF OutputFormat = "avif"
)

// IsValid returns true if the format is valid.
func (f OutputFormat) IsValid() bool {
	switch f {
	case FormatJPEG, FormatPNG, FormatWebP, FormatAVIF:
		return true
	}
	return false
}

// Variant config constraints
const (
	MinVariantWidth  = 1
	MaxVariantWidth  = 8192
	MinVariantHeight = 1
	MaxVariantHeight = 8192
	MinQuality       = 1
	MaxQuality       = 100
	MaxConfigNameLen = 50
	MinConfigNameLen = 2
)

// Errors for variant config validation
var (
	ErrVariantConfigNameRequired    = fmt.Errorf("%w: variant config name is required", shared.ErrInvalidInput)
	ErrVariantConfigNameTooShort    = fmt.Errorf("%w: variant config name must be at least %d characters", shared.ErrInvalidInput, MinConfigNameLen)
	ErrVariantConfigNameTooLong     = fmt.Errorf("%w: variant config name must not exceed %d characters", shared.ErrInvalidInput, MaxConfigNameLen)
	ErrVariantConfigNameInvalid     = fmt.Errorf("%w: variant config name must contain only alphanumeric characters, hyphens, and underscores", shared.ErrInvalidInput)
	ErrVariantConfigWidthInvalid    = fmt.Errorf("%w: width must be between %d and %d", shared.ErrInvalidInput, MinVariantWidth, MaxVariantWidth)
	ErrVariantConfigHeightInvalid   = fmt.Errorf("%w: height must be between %d and %d", shared.ErrInvalidInput, MinVariantHeight, MaxVariantHeight)
	ErrVariantConfigQualityInvalid  = fmt.Errorf("%w: quality must be between %d and %d", shared.ErrInvalidInput, MinQuality, MaxQuality)
	ErrVariantConfigFormatInvalid   = fmt.Errorf("%w: invalid output format", shared.ErrInvalidInput)
	ErrVariantConfigCropModeInvalid = fmt.Errorf("%w: invalid crop mode", shared.ErrInvalidInput)
)

var configNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// VariantConfig represents a user-defined image variant configuration.
// This allows users to create custom-sized variants beyond the default ones.
type VariantConfig struct {
	id        VariantConfigID
	userID    identity.UserID
	name      string
	maxWidth  int
	maxHeight int
	format    OutputFormat
	quality   int
	cropMode  CropMode
	isPreset  bool // System-defined presets vs user-created
	createdAt time.Time
	updatedAt time.Time
	events    []shared.DomainEvent
}

// NewVariantConfig creates a new variant configuration.
func NewVariantConfig(
	userID identity.UserID,
	name string,
	maxWidth, maxHeight int,
	format OutputFormat,
	quality int,
	cropMode CropMode,
) (*VariantConfig, error) {
	if userID.IsZero() {
		return nil, fmt.Errorf("%w: user ID is required", shared.ErrInvalidInput)
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrVariantConfigNameRequired
	}
	if len(name) < MinConfigNameLen {
		return nil, ErrVariantConfigNameTooShort
	}
	if len(name) > MaxConfigNameLen {
		return nil, ErrVariantConfigNameTooLong
	}
	if !configNameRegex.MatchString(name) {
		return nil, ErrVariantConfigNameInvalid
	}

	if maxWidth < MinVariantWidth || maxWidth > MaxVariantWidth {
		return nil, ErrVariantConfigWidthInvalid
	}
	if maxHeight < MinVariantHeight || maxHeight > MaxVariantHeight {
		return nil, ErrVariantConfigHeightInvalid
	}
	if quality < MinQuality || quality > MaxQuality {
		return nil, ErrVariantConfigQualityInvalid
	}
	if !format.IsValid() {
		return nil, ErrVariantConfigFormatInvalid
	}
	if !cropMode.IsValid() {
		return nil, ErrVariantConfigCropModeInvalid
	}

	now := time.Now().UTC()
	config := &VariantConfig{
		id:        NewVariantConfigID(),
		userID:    userID,
		name:      strings.ToLower(name),
		maxWidth:  maxWidth,
		maxHeight: maxHeight,
		format:    format,
		quality:   quality,
		cropMode:  cropMode,
		isPreset:  false,
		createdAt: now,
		updatedAt: now,
		events:    []shared.DomainEvent{},
	}

	config.addEvent(&VariantConfigCreated{
		BaseEvent:       shared.NewBaseEvent("gallery.variant_config.created", config.id.String()),
		VariantConfigID: config.id,
		UserID:          config.userID,
		Name:            config.name,
	})

	return config, nil
}

// ReconstructVariantConfig reconstitutes a VariantConfig from persistence.
func ReconstructVariantConfig(
	id VariantConfigID,
	userID identity.UserID,
	name string,
	maxWidth, maxHeight int,
	format OutputFormat,
	quality int,
	cropMode CropMode,
	isPreset bool,
	createdAt, updatedAt time.Time,
) *VariantConfig {
	return &VariantConfig{
		id:        id,
		userID:    userID,
		name:      name,
		maxWidth:  maxWidth,
		maxHeight: maxHeight,
		format:    format,
		quality:   quality,
		cropMode:  cropMode,
		isPreset:  isPreset,
		createdAt: createdAt,
		updatedAt: updatedAt,
		events:    []shared.DomainEvent{},
	}
}

// Getters

func (c *VariantConfig) ID() VariantConfigID          { return c.id }
func (c *VariantConfig) UserID() identity.UserID      { return c.userID }
func (c *VariantConfig) Name() string                 { return c.name }
func (c *VariantConfig) MaxWidth() int                { return c.maxWidth }
func (c *VariantConfig) MaxHeight() int               { return c.maxHeight }
func (c *VariantConfig) Format() OutputFormat         { return c.format }
func (c *VariantConfig) Quality() int                 { return c.quality }
func (c *VariantConfig) CropMode() CropMode           { return c.cropMode }
func (c *VariantConfig) IsPreset() bool               { return c.isPreset }
func (c *VariantConfig) CreatedAt() time.Time         { return c.createdAt }
func (c *VariantConfig) UpdatedAt() time.Time         { return c.updatedAt }
func (c *VariantConfig) Events() []shared.DomainEvent { return c.events }

// ClearEvents clears all pending domain events.
func (c *VariantConfig) ClearEvents() {
	c.events = []shared.DomainEvent{}
}

// Behavior Methods

// Update modifies the variant configuration.
func (c *VariantConfig) Update(maxWidth, maxHeight int, format OutputFormat, quality int, cropMode CropMode) error {
	if maxWidth < MinVariantWidth || maxWidth > MaxVariantWidth {
		return ErrVariantConfigWidthInvalid
	}
	if maxHeight < MinVariantHeight || maxHeight > MaxVariantHeight {
		return ErrVariantConfigHeightInvalid
	}
	if quality < MinQuality || quality > MaxQuality {
		return ErrVariantConfigQualityInvalid
	}
	if !format.IsValid() {
		return ErrVariantConfigFormatInvalid
	}
	if !cropMode.IsValid() {
		return ErrVariantConfigCropModeInvalid
	}

	c.maxWidth = maxWidth
	c.maxHeight = maxHeight
	c.format = format
	c.quality = quality
	c.cropMode = cropMode
	c.updatedAt = time.Now().UTC()

	c.addEvent(&VariantConfigUpdated{
		BaseEvent:       shared.NewBaseEvent("gallery.variant_config.updated", c.id.String()),
		VariantConfigID: c.id,
	})

	return nil
}

// Helper Methods

// IsOwnedBy returns true if the config is owned by the given user.
func (c *VariantConfig) IsOwnedBy(userID identity.UserID) bool {
	return c.userID.Equals(userID)
}

func (c *VariantConfig) addEvent(event shared.DomainEvent) {
	c.events = append(c.events, event)
}

// VariantConfig Events

// VariantConfigCreated is emitted when a new variant config is created.
type VariantConfigCreated struct {
	shared.BaseEvent
	VariantConfigID VariantConfigID
	UserID          identity.UserID
	Name            string
}

func (e *VariantConfigCreated) EventType() string {
	return "gallery.variant_config.created"
}

// VariantConfigUpdated is emitted when a variant config is updated.
type VariantConfigUpdated struct {
	shared.BaseEvent
	VariantConfigID VariantConfigID
}

func (e *VariantConfigUpdated) EventType() string {
	return "gallery.variant_config.updated"
}

// VariantConfigDeleted is emitted when a variant config is deleted.
type VariantConfigDeleted struct {
	shared.BaseEvent
	VariantConfigID VariantConfigID
}

func (e *VariantConfigDeleted) EventType() string {
	return "gallery.variant_config.deleted"
}
