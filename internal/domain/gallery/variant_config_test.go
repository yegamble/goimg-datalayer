package gallery

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// Test Helpers

func createTestUserID(t *testing.T) identity.UserID {
	t.Helper()
	id, err := identity.ParseUserID(uuid.New().String())
	require.NoError(t, err)
	return id
}

// VariantConfigID Tests

func TestVariantConfigID_NewVariantConfigID(t *testing.T) {
	id := NewVariantConfigID()
	assert.False(t, id.IsZero(), "new ID should not be zero")
	assert.NotEmpty(t, id.String(), "string representation should not be empty")
}

func TestVariantConfigID_ParseVariantConfigID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			input:   uuid.New().String(),
			wantErr: false,
		},
		{
			name:    "invalid UUID",
			input:   "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "nil UUID",
			input:   uuid.Nil.String(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParseVariantConfigID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.input, id.String())
			}
		})
	}
}

func TestVariantConfigID_Equals(t *testing.T) {
	id1 := NewVariantConfigID()
	id2 := NewVariantConfigID()

	assert.True(t, id1.Equals(id1), "ID should equal itself")
	assert.False(t, id1.Equals(id2), "different IDs should not be equal")

	parsed, _ := ParseVariantConfigID(id1.String())
	assert.True(t, id1.Equals(parsed), "parsed ID should equal original")
}

func TestVariantConfigID_IsZero(t *testing.T) {
	zeroID := VariantConfigID{}
	newID := NewVariantConfigID()

	assert.True(t, zeroID.IsZero(), "zero value ID should be zero")
	assert.False(t, newID.IsZero(), "new ID should not be zero")
}

func TestVariantConfigID_UUID(t *testing.T) {
	id := NewVariantConfigID()
	assert.NotEqual(t, uuid.Nil, id.UUID(), "UUID should not be nil")
}

// CropMode Tests

func TestCropMode_IsValid(t *testing.T) {
	tests := []struct {
		mode  CropMode
		valid bool
	}{
		{CropModeFit, true},
		{CropModeFill, true},
		{CropModeCrop, true},
		{CropMode("invalid"), false},
		{CropMode(""), false},
		{CropMode("FIT"), false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.mode.IsValid())
		})
	}
}

// OutputFormat Tests

func TestOutputFormat_IsValid(t *testing.T) {
	tests := []struct {
		format OutputFormat
		valid  bool
	}{
		{FormatJPEG, true},
		{FormatPNG, true},
		{FormatWebP, true},
		{FormatAVIF, true},
		{OutputFormat("gif"), false},
		{OutputFormat(""), false},
		{OutputFormat("JPEG"), false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.format.IsValid())
		})
	}
}

// NewVariantConfig Tests

func TestNewVariantConfig_Success(t *testing.T) {
	userID := createTestUserID(t)

	config, err := NewVariantConfig(
		userID,
		"my-variant",
		800,
		600,
		FormatWebP,
		85,
		CropModeFit,
	)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.False(t, config.ID().IsZero())
	assert.Equal(t, userID, config.UserID())
	assert.Equal(t, "my-variant", config.Name())
	assert.Equal(t, 800, config.MaxWidth())
	assert.Equal(t, 600, config.MaxHeight())
	assert.Equal(t, FormatWebP, config.Format())
	assert.Equal(t, 85, config.Quality())
	assert.Equal(t, CropModeFit, config.CropMode())
	assert.False(t, config.IsPreset())
	assert.False(t, config.CreatedAt().IsZero())
	assert.False(t, config.UpdatedAt().IsZero())

	// Should emit created event
	assert.Len(t, config.Events(), 1)
	assert.IsType(t, &VariantConfigCreated{}, config.Events()[0])
}

func TestNewVariantConfig_NameNormalization(t *testing.T) {
	userID := createTestUserID(t)

	// Test whitespace trimming
	config, err := NewVariantConfig(userID, "  my-variant  ", 800, 600, FormatWebP, 85, CropModeFit)
	require.NoError(t, err)
	assert.Equal(t, "my-variant", config.Name())

	// Test lowercase conversion
	config2, err := NewVariantConfig(userID, "MY-VARIANT", 800, 600, FormatWebP, 85, CropModeFit)
	require.NoError(t, err)
	assert.Equal(t, "my-variant", config2.Name())
}

func TestNewVariantConfig_ValidationErrors(t *testing.T) {
	userID := createTestUserID(t)
	zeroUserID := identity.UserID{}

	tests := []struct {
		name       string
		userID     identity.UserID
		configName string
		width      int
		height     int
		format     OutputFormat
		quality    int
		cropMode   CropMode
		wantErr    error
	}{
		{
			name:       "zero user ID",
			userID:     zeroUserID,
			configName: "test",
			width:      800,
			height:     600,
			format:     FormatWebP,
			quality:    85,
			cropMode:   CropModeFit,
			wantErr:    nil, // Just check for any error
		},
		{
			name:       "empty name",
			userID:     userID,
			configName: "",
			width:      800,
			height:     600,
			format:     FormatWebP,
			quality:    85,
			cropMode:   CropModeFit,
			wantErr:    ErrVariantConfigNameRequired,
		},
		{
			name:       "name too short",
			userID:     userID,
			configName: "a",
			width:      800,
			height:     600,
			format:     FormatWebP,
			quality:    85,
			cropMode:   CropModeFit,
			wantErr:    ErrVariantConfigNameTooShort,
		},
		{
			name:       "name too long",
			userID:     userID,
			configName: strings.Repeat("a", 51),
			width:      800,
			height:     600,
			format:     FormatWebP,
			quality:    85,
			cropMode:   CropModeFit,
			wantErr:    ErrVariantConfigNameTooLong,
		},
		{
			name:       "name with invalid characters",
			userID:     userID,
			configName: "my variant!",
			width:      800,
			height:     600,
			format:     FormatWebP,
			quality:    85,
			cropMode:   CropModeFit,
			wantErr:    ErrVariantConfigNameInvalid,
		},
		{
			name:       "width too small",
			userID:     userID,
			configName: "test",
			width:      0,
			height:     600,
			format:     FormatWebP,
			quality:    85,
			cropMode:   CropModeFit,
			wantErr:    ErrVariantConfigWidthInvalid,
		},
		{
			name:       "width too large",
			userID:     userID,
			configName: "test",
			width:      9000,
			height:     600,
			format:     FormatWebP,
			quality:    85,
			cropMode:   CropModeFit,
			wantErr:    ErrVariantConfigWidthInvalid,
		},
		{
			name:       "height too small",
			userID:     userID,
			configName: "test",
			width:      800,
			height:     0,
			format:     FormatWebP,
			quality:    85,
			cropMode:   CropModeFit,
			wantErr:    ErrVariantConfigHeightInvalid,
		},
		{
			name:       "height too large",
			userID:     userID,
			configName: "test",
			width:      800,
			height:     9000,
			format:     FormatWebP,
			quality:    85,
			cropMode:   CropModeFit,
			wantErr:    ErrVariantConfigHeightInvalid,
		},
		{
			name:       "quality too low",
			userID:     userID,
			configName: "test",
			width:      800,
			height:     600,
			format:     FormatWebP,
			quality:    0,
			cropMode:   CropModeFit,
			wantErr:    ErrVariantConfigQualityInvalid,
		},
		{
			name:       "quality too high",
			userID:     userID,
			configName: "test",
			width:      800,
			height:     600,
			format:     FormatWebP,
			quality:    101,
			cropMode:   CropModeFit,
			wantErr:    ErrVariantConfigQualityInvalid,
		},
		{
			name:       "invalid format",
			userID:     userID,
			configName: "test",
			width:      800,
			height:     600,
			format:     OutputFormat("gif"),
			quality:    85,
			cropMode:   CropModeFit,
			wantErr:    ErrVariantConfigFormatInvalid,
		},
		{
			name:       "invalid crop mode",
			userID:     userID,
			configName: "test",
			width:      800,
			height:     600,
			format:     FormatWebP,
			quality:    85,
			cropMode:   CropMode("invalid"),
			wantErr:    ErrVariantConfigCropModeInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewVariantConfig(
				tt.userID,
				tt.configName,
				tt.width,
				tt.height,
				tt.format,
				tt.quality,
				tt.cropMode,
			)

			assert.Nil(t, config)
			assert.Error(t, err)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestNewVariantConfig_BoundaryValues(t *testing.T) {
	userID := createTestUserID(t)

	// Test minimum valid values
	config, err := NewVariantConfig(userID, "ab", MinVariantWidth, MinVariantHeight, FormatJPEG, MinQuality, CropModeFit)
	require.NoError(t, err)
	assert.Equal(t, MinVariantWidth, config.MaxWidth())
	assert.Equal(t, MinVariantHeight, config.MaxHeight())
	assert.Equal(t, MinQuality, config.Quality())

	// Test maximum valid values
	longName := strings.Repeat("a", MaxConfigNameLen)
	config2, err := NewVariantConfig(userID, longName, MaxVariantWidth, MaxVariantHeight, FormatAVIF, MaxQuality, CropModeCrop)
	require.NoError(t, err)
	assert.Equal(t, MaxVariantWidth, config2.MaxWidth())
	assert.Equal(t, MaxVariantHeight, config2.MaxHeight())
	assert.Equal(t, MaxQuality, config2.Quality())
}

// ReconstructVariantConfig Tests

func TestReconstructVariantConfig(t *testing.T) {
	id := NewVariantConfigID()
	userID := createTestUserID(t)
	now := time.Now().UTC()

	config := ReconstructVariantConfig(
		id,
		userID,
		"my-preset",
		1920,
		1080,
		FormatWebP,
		90,
		CropModeFill,
		true,
		now,
		now,
	)

	assert.Equal(t, id, config.ID())
	assert.Equal(t, userID, config.UserID())
	assert.Equal(t, "my-preset", config.Name())
	assert.Equal(t, 1920, config.MaxWidth())
	assert.Equal(t, 1080, config.MaxHeight())
	assert.Equal(t, FormatWebP, config.Format())
	assert.Equal(t, 90, config.Quality())
	assert.Equal(t, CropModeFill, config.CropMode())
	assert.True(t, config.IsPreset())
	assert.Equal(t, now, config.CreatedAt())
	assert.Equal(t, now, config.UpdatedAt())
	// Reconstructed configs should have no events
	assert.Empty(t, config.Events())
}

// Update Tests

func TestVariantConfig_Update_Success(t *testing.T) {
	userID := createTestUserID(t)
	config, _ := NewVariantConfig(userID, "test", 800, 600, FormatJPEG, 80, CropModeFit)
	config.ClearEvents() // Clear creation event

	originalUpdatedAt := config.UpdatedAt()
	time.Sleep(time.Millisecond) // Ensure time difference

	err := config.Update(1200, 900, FormatWebP, 95, CropModeFill)
	require.NoError(t, err)

	assert.Equal(t, 1200, config.MaxWidth())
	assert.Equal(t, 900, config.MaxHeight())
	assert.Equal(t, FormatWebP, config.Format())
	assert.Equal(t, 95, config.Quality())
	assert.Equal(t, CropModeFill, config.CropMode())
	assert.True(t, config.UpdatedAt().After(originalUpdatedAt))

	// Should emit updated event
	require.Len(t, config.Events(), 1)
	assert.IsType(t, &VariantConfigUpdated{}, config.Events()[0])
}

func TestVariantConfig_Update_ValidationErrors(t *testing.T) {
	userID := createTestUserID(t)
	config, _ := NewVariantConfig(userID, "test", 800, 600, FormatJPEG, 80, CropModeFit)
	config.ClearEvents()

	tests := []struct {
		name     string
		width    int
		height   int
		format   OutputFormat
		quality  int
		cropMode CropMode
		wantErr  error
	}{
		{"width too small", 0, 600, FormatWebP, 85, CropModeFit, ErrVariantConfigWidthInvalid},
		{"width too large", 9000, 600, FormatWebP, 85, CropModeFit, ErrVariantConfigWidthInvalid},
		{"height too small", 800, 0, FormatWebP, 85, CropModeFit, ErrVariantConfigHeightInvalid},
		{"height too large", 800, 9000, FormatWebP, 85, CropModeFit, ErrVariantConfigHeightInvalid},
		{"quality too low", 800, 600, FormatWebP, 0, CropModeFit, ErrVariantConfigQualityInvalid},
		{"quality too high", 800, 600, FormatWebP, 101, CropModeFit, ErrVariantConfigQualityInvalid},
		{"invalid format", 800, 600, OutputFormat("gif"), 85, CropModeFit, ErrVariantConfigFormatInvalid},
		{"invalid crop mode", 800, 600, FormatWebP, 85, CropMode("invalid"), ErrVariantConfigCropModeInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh config for each test
			cfg, _ := NewVariantConfig(userID, "test", 800, 600, FormatJPEG, 80, CropModeFit)
			cfg.ClearEvents()

			err := cfg.Update(tt.width, tt.height, tt.format, tt.quality, tt.cropMode)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, cfg.Events()) // No event on error
		})
	}
}

// IsOwnedBy Tests

func TestVariantConfig_IsOwnedBy(t *testing.T) {
	owner := createTestUserID(t)
	other := createTestUserID(t)

	config, _ := NewVariantConfig(owner, "test", 800, 600, FormatWebP, 85, CropModeFit)

	assert.True(t, config.IsOwnedBy(owner))
	assert.False(t, config.IsOwnedBy(other))
}

// Event Tests

func TestVariantConfig_ClearEvents(t *testing.T) {
	userID := createTestUserID(t)
	config, _ := NewVariantConfig(userID, "test", 800, 600, FormatWebP, 85, CropModeFit)

	assert.NotEmpty(t, config.Events())
	config.ClearEvents()
	assert.Empty(t, config.Events())
}

func TestVariantConfigCreated_EventType(t *testing.T) {
	event := &VariantConfigCreated{}
	assert.Equal(t, "gallery.variant_config.created", event.EventType())
}

func TestVariantConfigUpdated_EventType(t *testing.T) {
	event := &VariantConfigUpdated{}
	assert.Equal(t, "gallery.variant_config.updated", event.EventType())
}

func TestVariantConfigDeleted_EventType(t *testing.T) {
	event := &VariantConfigDeleted{}
	assert.Equal(t, "gallery.variant_config.deleted", event.EventType())
}

// Valid Config Name Tests

func TestVariantConfig_ValidNames(t *testing.T) {
	userID := createTestUserID(t)

	validNames := []string{
		"thumbnail",
		"my-variant",
		"variant_1",
		"HD-1080p",
		"a1",
		"test-variant-name",
		"UPPERCASE",
		"MixedCase123",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			config, err := NewVariantConfig(userID, name, 800, 600, FormatWebP, 85, CropModeFit)
			require.NoError(t, err, "name should be valid: %s", name)
			assert.NotNil(t, config)
		})
	}
}

func TestVariantConfig_InvalidNames(t *testing.T) {
	userID := createTestUserID(t)

	invalidNames := []string{
		"my variant",    // space
		"variant.name",  // dot
		"variant@name",  // special char
		"variant/name",  // slash
		"variant\\name", // backslash
		"<script>",      // XSS attempt
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			config, err := NewVariantConfig(userID, name, 800, 600, FormatWebP, 85, CropModeFit)
			assert.Error(t, err, "name should be invalid: %s", name)
			assert.Nil(t, config)
		})
	}
}
