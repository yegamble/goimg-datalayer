package community

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroupSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
		wantVal string
	}{
		{
			name:    "valid slug",
			input:   "landscape-photography",
			wantErr: nil,
			wantVal: "landscape-photography",
		},
		{
			name:    "minimum length",
			input:   "art",
			wantErr: nil,
			wantVal: "art",
		},
		{
			name:    "maximum length",
			input:   strings.Repeat("a", 110),
			wantErr: nil,
			wantVal: strings.Repeat("a", 110),
		},
		{
			name:    "with numbers",
			input:   "photo-2024",
			wantErr: nil,
			wantVal: "photo-2024",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: ErrGroupSlugRequired,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: ErrGroupSlugRequired,
		},
		{
			name:    "too short",
			input:   "ab",
			wantErr: ErrGroupSlugTooShort,
		},
		{
			name:    "too long",
			input:   strings.Repeat("a", 111),
			wantErr: ErrGroupSlugTooLong,
		},
		{
			name:    "uppercase normalized",
			input:   "Photography",
			wantErr: nil,
			wantVal: "photography",
		},
		{
			name:    "mixed case normalized",
			input:   "LanDSCape-PhotoGRAPHY",
			wantErr: nil,
			wantVal: "landscape-photography",
		},
		{
			name:    "invalid characters - spaces",
			input:   "my group",
			wantErr: ErrGroupSlugInvalid,
		},
		{
			name:    "invalid characters - underscores",
			input:   "my_group",
			wantErr: ErrGroupSlugInvalid,
		},
		{
			name:    "invalid characters - special",
			input:   "my@group",
			wantErr: ErrGroupSlugInvalid,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			slug, err := NewGroupSlug(tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.True(t, slug.IsEmpty())
			} else {
				require.NoError(t, err)
				assert.False(t, slug.IsEmpty())
				assert.Equal(t, tt.wantVal, slug.String())
			}
		})
	}
}

func TestGenerateSlugFromName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "Photography",
			expected: "photography",
		},
		{
			name:     "name with spaces",
			input:    "Landscape Photography",
			expected: "landscape-photography",
		},
		{
			name:     "name with multiple spaces",
			input:    "Nature   and   Wildlife",
			expected: "nature-and-wildlife",
		},
		{
			name:     "name with special characters",
			input:    "Art & Photography!",
			expected: "art-photography",
		},
		{
			name:     "name with numbers",
			input:    "Photography 2024",
			expected: "photography-2024",
		},
		{
			name:     "name with underscores",
			input:    "My_Group_Name",
			expected: "mygroupname",
		},
		{
			name:     "name with multiple hyphens",
			input:    "My---Group",
			expected: "my-group",
		},
		{
			name:     "name starting with hyphen",
			input:    "-Photography",
			expected: "photography",
		},
		{
			name:     "name ending with hyphen",
			input:    "Photography-",
			expected: "photography",
		},
		{
			name:     "very long name truncated",
			input:    strings.Repeat("Photography ", 20),
			expected: strings.TrimRight(strings.Repeat("photography-", 9)+"photography", "-")[:110],
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			slug, err := GenerateSlugFromName(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, slug.String())
		})
	}
}

func TestGroupSlug_Equals(t *testing.T) {
	t.Parallel()

	slug1, _ := NewGroupSlug("photography")
	slug2, _ := NewGroupSlug("photography")
	slug3, _ := NewGroupSlug("art")

	assert.True(t, slug1.Equals(slug2))
	assert.False(t, slug1.Equals(slug3))
}
