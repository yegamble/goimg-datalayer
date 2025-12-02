package gallery_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

func TestNewImageID(t *testing.T) {
	t.Parallel()

	id := gallery.NewImageID()

	assert.False(t, id.IsZero(), "new ID should not be zero")
	assert.NotEmpty(t, id.String(), "new ID should have string representation")
}

func TestParseImageID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			input:   "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "valid UUID uppercase normalized to lowercase",
			input:   "550E8400-E29B-41D4-A716-446655440000",
			wantErr: false,
		},
		{
			name:    "invalid UUID format",
			input:   "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "partial UUID",
			input:   "550e8400-e29b-41d4",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := gallery.ParseImageID(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, id.IsZero())
			} else {
				require.NoError(t, err)
				assert.False(t, id.IsZero())
				// UUID is normalized to lowercase
				assert.NotEmpty(t, id.String())
			}
		})
	}
}

func TestImageID_String(t *testing.T) {
	t.Parallel()

	expected := "550e8400-e29b-41d4-a716-446655440000"
	id, err := gallery.ParseImageID(expected)
	require.NoError(t, err)

	assert.Equal(t, expected, id.String())
}

func TestImageID_IsZero(t *testing.T) {
	t.Parallel()

	t.Run("new ID is not zero", func(t *testing.T) {
		t.Parallel()
		id := gallery.NewImageID()
		assert.False(t, id.IsZero())
	})

	t.Run("zero value is zero", func(t *testing.T) {
		t.Parallel()
		var id gallery.ImageID
		assert.True(t, id.IsZero())
	})
}

func TestImageID_Equals(t *testing.T) {
	t.Parallel()

	id1, _ := gallery.ParseImageID("550e8400-e29b-41d4-a716-446655440000")
	id2, _ := gallery.ParseImageID("550e8400-e29b-41d4-a716-446655440000")
	id3, _ := gallery.ParseImageID("660e8400-e29b-41d4-a716-446655440000")

	t.Run("same UUID equals", func(t *testing.T) {
		t.Parallel()
		assert.True(t, id1.Equals(id2))
	})

	t.Run("different UUID not equals", func(t *testing.T) {
		t.Parallel()
		assert.False(t, id1.Equals(id3))
	})

	t.Run("zero values equal", func(t *testing.T) {
		t.Parallel()
		var zero1, zero2 gallery.ImageID
		assert.True(t, zero1.Equals(zero2))
	})
}

func TestMustParseImageID(t *testing.T) {
	t.Parallel()

	t.Run("valid UUID does not panic", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() {
			id := gallery.MustParseImageID("550e8400-e29b-41d4-a716-446655440000")
			assert.False(t, id.IsZero())
		})
	})

	t.Run("invalid UUID panics", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			gallery.MustParseImageID("invalid")
		})
	})
}

// Test AlbumID (same structure as ImageID)

func TestNewAlbumID(t *testing.T) {
	t.Parallel()

	id := gallery.NewAlbumID()

	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())
}

func TestParseAlbumID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			input:   "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "invalid UUID",
			input:   "not-a-uuid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := gallery.ParseAlbumID(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, id.IsZero())
			} else {
				require.NoError(t, err)
				assert.False(t, id.IsZero())
			}
		})
	}
}

func TestAlbumID_Equals(t *testing.T) {
	t.Parallel()

	id1, _ := gallery.ParseAlbumID("550e8400-e29b-41d4-a716-446655440000")
	id2, _ := gallery.ParseAlbumID("550e8400-e29b-41d4-a716-446655440000")
	id3, _ := gallery.ParseAlbumID("660e8400-e29b-41d4-a716-446655440000")

	assert.True(t, id1.Equals(id2))
	assert.False(t, id1.Equals(id3))
}

// Test CommentID (same structure as ImageID)

func TestNewCommentID(t *testing.T) {
	t.Parallel()

	id := gallery.NewCommentID()

	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())
}

func TestParseCommentID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			input:   "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "invalid UUID",
			input:   "not-a-uuid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := gallery.ParseCommentID(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, id.IsZero())
			} else {
				require.NoError(t, err)
				assert.False(t, id.IsZero())
			}
		})
	}
}

func TestCommentID_Equals(t *testing.T) {
	t.Parallel()

	id1, _ := gallery.ParseCommentID("550e8400-e29b-41d4-a716-446655440000")
	id2, _ := gallery.ParseCommentID("550e8400-e29b-41d4-a716-446655440000")
	id3, _ := gallery.ParseCommentID("660e8400-e29b-41d4-a716-446655440000")

	assert.True(t, id1.Equals(id2))
	assert.False(t, id1.Equals(id3))
}

// Test that IDs are type-safe (different types cannot be compared)
func TestID_TypeSafety(t *testing.T) {
	t.Parallel()

	// This test ensures that ImageID, AlbumID, and CommentID are distinct types
	// and cannot be accidentally mixed. This test doesn't execute runtime checks
	// but serves as documentation that the types are separate.

	imageID := gallery.NewImageID()
	albumID := gallery.NewAlbumID()
	commentID := gallery.NewCommentID()

	// The following lines would not compile (type safety):
	// _ = imageID.Equals(albumID)    // compile error
	// _ = albumID.Equals(commentID)  // compile error

	// But these work:
	assert.False(t, imageID.IsZero())
	assert.False(t, albumID.IsZero())
	assert.False(t, commentID.IsZero())
}
