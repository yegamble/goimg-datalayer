package gallery_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

//nolint:funlen // Table-driven test with comprehensive test cases
func TestNewTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantName string
		wantSlug string
		wantErr  error
	}{
		{
			name:     "simple tag",
			input:    "landscape",
			wantName: "landscape",
			wantSlug: "landscape",
			wantErr:  nil,
		},
		{
			name:     "tag with spaces",
			input:    "nature photography",
			wantName: "nature photography",
			wantSlug: "nature-photography",
			wantErr:  nil,
		},
		{
			name:     "tag with multiple spaces",
			input:    "nature  and   wildlife",
			wantName: "nature  and   wildlife",
			wantSlug: "nature-and-wildlife",
			wantErr:  nil,
		},
		{
			name:     "tag with hyphen",
			input:    "street-photography",
			wantName: "street-photography",
			wantSlug: "street-photography",
			wantErr:  nil,
		},
		{
			name:     "tag with underscore",
			input:    "black_and_white",
			wantName: "black_and_white",
			wantSlug: "black_and_white",
			wantErr:  nil,
		},
		{
			name:     "tag with numbers",
			input:    "photo2024",
			wantName: "photo2024",
			wantSlug: "photo2024",
			wantErr:  nil,
		},
		{
			name:     "uppercase normalized",
			input:    "LANDSCAPE",
			wantName: "landscape",
			wantSlug: "landscape",
			wantErr:  nil,
		},
		{
			name:     "mixed case normalized",
			input:    "NaturE PhOtOGraPhy",
			wantName: "nature photography",
			wantSlug: "nature-photography",
			wantErr:  nil,
		},
		{
			name:     "tag with whitespace trimmed",
			input:    "  landscape  ",
			wantName: "landscape",
			wantSlug: "landscape",
			wantErr:  nil,
		},
		{
			name:     "minimum length",
			input:    "ab",
			wantName: "ab",
			wantSlug: "ab",
			wantErr:  nil,
		},
		{
			name:     "maximum length",
			input:    strings.Repeat("a", 50),
			wantName: strings.Repeat("a", 50),
			wantSlug: strings.Repeat("a", 50),
			wantErr:  nil,
		},
		{
			name:     "too short",
			input:    "a",
			wantName: "",
			wantSlug: "",
			wantErr:  gallery.ErrTagTooShort,
		},
		{
			name:     "too long",
			input:    strings.Repeat("a", 51),
			wantName: "",
			wantSlug: "",
			wantErr:  gallery.ErrTagTooLong,
		},
		{
			name:     "empty string",
			input:    "",
			wantName: "",
			wantSlug: "",
			wantErr:  gallery.ErrTagTooShort,
		},
		{
			name:     "only spaces",
			input:    "   ",
			wantName: "",
			wantSlug: "",
			wantErr:  gallery.ErrTagTooShort,
		},
		{
			name:     "special characters removed",
			input:    "nature@photo",
			wantName: "nature@photo",
			wantSlug: "naturephoto",
			wantErr:  nil,
		},
		{
			name:     "emoji removed",
			input:    "nature<?photo",
			wantName: "nature<?photo",
			wantSlug: "naturephoto",
			wantErr:  nil,
		},
		{
			name:     "leading/trailing hyphens removed",
			input:    "-landscape-",
			wantName: "-landscape-",
			wantSlug: "landscape",
			wantErr:  nil,
		},
		{
			name:     "multiple consecutive hyphens collapsed",
			input:    "nature---photo",
			wantName: "nature---photo",
			wantSlug: "nature-photo",
			wantErr:  nil,
		},
		{
			name:     "only special characters becomes invalid",
			input:    "@#$%",
			wantName: "",
			wantSlug: "",
			wantErr:  gallery.ErrTagInvalid,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tag, err := gallery.NewTag(tt.input)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantName, tag.Name())
				assert.Equal(t, tt.wantSlug, tag.Slug())
			}
		})
	}
}

func TestTag_String(t *testing.T) {
	t.Parallel()

	tag, err := gallery.NewTag("landscape")
	require.NoError(t, err)

	assert.Equal(t, "landscape", tag.String())
}

func TestTag_Equals(t *testing.T) {
	t.Parallel()

	tag1, _ := gallery.NewTag("landscape")
	tag2, _ := gallery.NewTag("landscape")
	tag3, _ := gallery.NewTag("portrait")
	tag4, _ := gallery.NewTag("LANDSCAPE") // normalized to lowercase

	t.Run("same tag equals", func(t *testing.T) {
		t.Parallel()
		assert.True(t, tag1.Equals(tag2))
	})

	t.Run("different tag not equals", func(t *testing.T) {
		t.Parallel()
		assert.False(t, tag1.Equals(tag3))
	})

	t.Run("normalized tags equal", func(t *testing.T) {
		t.Parallel()
		assert.True(t, tag1.Equals(tag4))
	})

	t.Run("tags with spaces equal by slug", func(t *testing.T) {
		t.Parallel()
		tagSpace, _ := gallery.NewTag("nature photography")
		tagHyphen, _ := gallery.NewTag("nature-photography")
		assert.True(t, tagSpace.Equals(tagHyphen))
	})
}

func TestMustNewTag(t *testing.T) {
	t.Parallel()

	t.Run("valid tag does not panic", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() {
			tag := gallery.MustNewTag("landscape")
			assert.Equal(t, "landscape", tag.Name())
		})
	})

	t.Run("invalid tag panics", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			gallery.MustNewTag("a") // too short
		})
	})
}

//nolint:funlen // Table-driven test with comprehensive test cases
func TestTag_SlugGeneration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantSlug string
	}{
		{
			name:     "alphanumeric only",
			input:    "photo123",
			wantSlug: "photo123",
		},
		{
			name:     "spaces to hyphens",
			input:    "my favorite photos",
			wantSlug: "my-favorite-photos",
		},
		{
			name:     "underscores preserved",
			input:    "black_and_white",
			wantSlug: "black_and_white",
		},
		{
			name:     "hyphens preserved",
			input:    "street-photography",
			wantSlug: "street-photography",
		},
		{
			name:     "mixed separators",
			input:    "nature_and-wildlife photos",
			wantSlug: "nature_and-wildlife-photos",
		},
		{
			name:     "special chars removed",
			input:    "photo!@#$%2024",
			wantSlug: "photo2024",
		},
		{
			name:     "dots removed",
			input:    "photo.2024",
			wantSlug: "photo2024",
		},
		{
			name:     "leading/trailing spaces",
			input:    "   photo   ",
			wantSlug: "photo",
		},
		{
			name:     "consecutive spaces",
			input:    "my    photo",
			wantSlug: "my-photo",
		},
		{
			name:     "parentheses removed",
			input:    "photo(2024)",
			wantSlug: "photo2024",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tag, err := gallery.NewTag(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSlug, tag.Slug())
		})
	}
}
