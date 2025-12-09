package gallery_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

//nolint:funlen // Table-driven test with comprehensive test cases
func TestNewImageMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		title            string
		description      string
		originalFilename string
		mimeType         string
		width            int
		height           int
		fileSize         int64
		storageKey       string
		storageProvider  string
		wantErr          error
	}{
		{
			name:             "valid metadata with title",
			title:            "My Photo",
			description:      "A beautiful landscape",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            1920,
			height:           1080,
			fileSize:         1024000,
			storageKey:       "images/photo.jpg",
			storageProvider:  "s3",
			wantErr:          nil,
		},
		{
			name:             "valid metadata without title uses filename",
			title:            "",
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            1920,
			height:           1080,
			fileSize:         1024000,
			storageKey:       "images/photo.jpg",
			storageProvider:  "s3",
			wantErr:          nil,
		},
		{
			name:             "title too long",
			title:            strings.Repeat("a", 256),
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            1920,
			height:           1080,
			fileSize:         1024000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          gallery.ErrTitleTooLong,
		},
		{
			name:             "description too long",
			title:            "Title",
			description:      strings.Repeat("a", 2001),
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            1920,
			height:           1080,
			fileSize:         1024000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          gallery.ErrDescriptionTooLong,
		},
		{
			name:             "unsupported mime type",
			title:            "Title",
			description:      "",
			originalFilename: "photo.bmp",
			mimeType:         "image/bmp",
			width:            1920,
			height:           1080,
			fileSize:         1024000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          gallery.ErrInvalidMimeType,
		},
		{
			name:             "zero width",
			title:            "Title",
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            0,
			height:           1080,
			fileSize:         1024000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          gallery.ErrInvalidDimensions,
		},
		{
			name:             "zero height",
			title:            "Title",
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            1920,
			height:           0,
			fileSize:         1024000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          gallery.ErrInvalidDimensions,
		},
		{
			name:             "negative width",
			title:            "Title",
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            -1,
			height:           1080,
			fileSize:         1024000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          gallery.ErrInvalidDimensions,
		},
		{
			name:             "dimensions too large",
			title:            "Title",
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            9000,
			height:           9000,
			fileSize:         1024000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          gallery.ErrImageTooLarge,
		},
		{
			name:             "file size zero",
			title:            "Title",
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            1920,
			height:           1080,
			fileSize:         0,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          gallery.ErrInvalidMetadata,
		},
		{
			name:             "file size too large",
			title:            "Title",
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            1920,
			height:           1080,
			fileSize:         10485761, // 10MB + 1 byte
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          gallery.ErrFileTooLarge,
		},
		{
			name:             "storage key empty",
			title:            "Title",
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            1920,
			height:           1080,
			fileSize:         1024000,
			storageKey:       "",
			storageProvider:  "s3",
			wantErr:          gallery.ErrStorageKeyRequired,
		},
		{
			name:             "storage provider empty",
			title:            "Title",
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            1920,
			height:           1080,
			fileSize:         1024000,
			storageKey:       "key",
			storageProvider:  "",
			wantErr:          gallery.ErrProviderRequired,
		},
		{
			name:             "filename empty",
			title:            "Title",
			description:      "",
			originalFilename: "",
			mimeType:         "image/jpeg",
			width:            1920,
			height:           1080,
			fileSize:         1024000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          gallery.ErrInvalidMetadata,
		},
		{
			name:             "all supported mime types - jpeg",
			title:            "Title",
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            100,
			height:           100,
			fileSize:         1000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          nil,
		},
		{
			name:             "all supported mime types - jpg",
			title:            "Title",
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpg",
			width:            100,
			height:           100,
			fileSize:         1000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          nil,
		},
		{
			name:             "all supported mime types - png",
			title:            "Title",
			description:      "",
			originalFilename: "photo.png",
			mimeType:         "image/png",
			width:            100,
			height:           100,
			fileSize:         1000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          nil,
		},
		{
			name:             "all supported mime types - gif",
			title:            "Title",
			description:      "",
			originalFilename: "photo.gif",
			mimeType:         "image/gif",
			width:            100,
			height:           100,
			fileSize:         1000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          nil,
		},
		{
			name:             "all supported mime types - webp",
			title:            "Title",
			description:      "",
			originalFilename: "photo.webp",
			mimeType:         "image/webp",
			width:            100,
			height:           100,
			fileSize:         1000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          nil,
		},
		{
			name:             "max allowed dimensions",
			title:            "Title",
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            8192,
			height:           8192,
			fileSize:         1024000,
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          nil,
		},
		{
			name:             "max allowed file size",
			title:            "Title",
			description:      "",
			originalFilename: "photo.jpg",
			mimeType:         "image/jpeg",
			width:            1920,
			height:           1080,
			fileSize:         10485760, // exactly 10MB
			storageKey:       "key",
			storageProvider:  "s3",
			wantErr:          nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			metadata, err := gallery.NewImageMetadata(
				tt.title,
				tt.description,
				tt.originalFilename,
				tt.mimeType,
				tt.width,
				tt.height,
				tt.fileSize,
				tt.storageKey,
				tt.storageProvider,
			)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				// Verify metadata was created correctly
				if tt.title == "" {
					assert.Equal(t, tt.originalFilename, metadata.Title())
				} else {
					assert.Equal(t, tt.title, metadata.Title())
				}
				assert.Equal(t, strings.TrimSpace(tt.description), metadata.Description())
				assert.Equal(t, tt.width, metadata.Width())
				assert.Equal(t, tt.height, metadata.Height())
			}
		})
	}
}

func TestImageMetadata_Getters(t *testing.T) {
	t.Parallel()

	metadata, err := gallery.NewImageMetadata(
		"My Photo",
		"A description",
		"original.jpg",
		"image/jpeg",
		1920,
		1080,
		1024000,
		"images/photo.jpg",
		"s3",
	)
	require.NoError(t, err)

	assert.Equal(t, "My Photo", metadata.Title())
	assert.Equal(t, "A description", metadata.Description())
	assert.Equal(t, "original.jpg", metadata.OriginalFilename())
	assert.Equal(t, "image/jpeg", metadata.MimeType())
	assert.Equal(t, 1920, metadata.Width())
	assert.Equal(t, 1080, metadata.Height())
	assert.Equal(t, int64(1024000), metadata.FileSize())
	assert.Equal(t, "images/photo.jpg", metadata.StorageKey())
	assert.Equal(t, "s3", metadata.StorageProvider())
}

func TestImageMetadata_AspectRatio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		width  int
		height int
		want   float64
	}{
		{
			name:   "16:9 landscape",
			width:  1920,
			height: 1080,
			want:   1.7777777777777777,
		},
		{
			name:   "4:3 landscape",
			width:  1024,
			height: 768,
			want:   1.3333333333333333,
		},
		{
			name:   "1:1 square",
			width:  1000,
			height: 1000,
			want:   1.0,
		},
		{
			name:   "9:16 portrait",
			width:  1080,
			height: 1920,
			want:   0.5625,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			metadata, err := gallery.NewImageMetadata(
				"Title",
				"",
				"file.jpg",
				"image/jpeg",
				tt.width,
				tt.height,
				1000,
				"key",
				"s3",
			)
			require.NoError(t, err)

			assert.InDelta(t, tt.want, metadata.AspectRatio(), 0.0001)
		})
	}
}

func TestImageMetadata_Orientation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		width         int
		height        int
		wantLandscape bool
		wantPortrait  bool
		wantSquare    bool
	}{
		{
			name:          "landscape",
			width:         1920,
			height:        1080,
			wantLandscape: true,
			wantPortrait:  false,
			wantSquare:    false,
		},
		{
			name:          "portrait",
			width:         1080,
			height:        1920,
			wantLandscape: false,
			wantPortrait:  true,
			wantSquare:    false,
		},
		{
			name:          "square",
			width:         1000,
			height:        1000,
			wantLandscape: false,
			wantPortrait:  false,
			wantSquare:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			metadata, err := gallery.NewImageMetadata(
				"Title",
				"",
				"file.jpg",
				"image/jpeg",
				tt.width,
				tt.height,
				1000,
				"key",
				"s3",
			)
			require.NoError(t, err)

			assert.Equal(t, tt.wantLandscape, metadata.IsLandscape())
			assert.Equal(t, tt.wantPortrait, metadata.IsPortrait())
			assert.Equal(t, tt.wantSquare, metadata.IsSquare())
		})
	}
}

func TestImageMetadata_WithTitle(t *testing.T) {
	t.Parallel()

	original, err := gallery.NewImageMetadata(
		"Original Title",
		"Description",
		"file.jpg",
		"image/jpeg",
		1920,
		1080,
		1000,
		"key",
		"s3",
	)
	require.NoError(t, err)

	t.Run("valid title update", func(t *testing.T) {
		t.Parallel()

		updated, err := original.WithTitle("New Title")
		require.NoError(t, err)

		assert.Equal(t, "New Title", updated.Title())
		assert.Equal(t, "Description", updated.Description()) // unchanged
		assert.Equal(t, "Original Title", original.Title())   // original unchanged
	})

	t.Run("empty title uses filename", func(t *testing.T) {
		t.Parallel()

		updated, err := original.WithTitle("")
		require.NoError(t, err)

		assert.Equal(t, "file.jpg", updated.Title())
	})

	t.Run("title too long", func(t *testing.T) {
		t.Parallel()

		_, err := original.WithTitle(strings.Repeat("a", 256))
		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrTitleTooLong)
	})
}

func TestImageMetadata_WithDescription(t *testing.T) {
	t.Parallel()

	original, err := gallery.NewImageMetadata(
		"Title",
		"Original Description",
		"file.jpg",
		"image/jpeg",
		1920,
		1080,
		1000,
		"key",
		"s3",
	)
	require.NoError(t, err)

	t.Run("valid description update", func(t *testing.T) {
		t.Parallel()

		updated, err := original.WithDescription("New Description")
		require.NoError(t, err)

		assert.Equal(t, "New Description", updated.Description())
		assert.Equal(t, "Title", updated.Title())                       // unchanged
		assert.Equal(t, "Original Description", original.Description()) // original unchanged
	})

	t.Run("empty description", func(t *testing.T) {
		t.Parallel()

		updated, err := original.WithDescription("")
		require.NoError(t, err)

		assert.Empty(t, updated.Description())
	})

	t.Run("description too long", func(t *testing.T) {
		t.Parallel()

		_, err := original.WithDescription(strings.Repeat("a", 2001))
		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrDescriptionTooLong)
	})
}
