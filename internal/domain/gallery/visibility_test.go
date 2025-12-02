package gallery_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

func TestParseVisibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    gallery.Visibility
		wantErr bool
	}{
		{
			name:    "public",
			input:   "public",
			want:    gallery.VisibilityPublic,
			wantErr: false,
		},
		{
			name:    "private",
			input:   "private",
			want:    gallery.VisibilityPrivate,
			wantErr: false,
		},
		{
			name:    "unlisted",
			input:   "unlisted",
			want:    gallery.VisibilityUnlisted,
			wantErr: false,
		},
		{
			name:    "invalid visibility",
			input:   "invalid",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "uppercase not normalized",
			input:   "PUBLIC",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := gallery.ParseVisibility(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrInvalidVisibility)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestVisibility_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		visibility gallery.Visibility
		want       bool
	}{
		{
			name:       "public is valid",
			visibility: gallery.VisibilityPublic,
			want:       true,
		},
		{
			name:       "private is valid",
			visibility: gallery.VisibilityPrivate,
			want:       true,
		},
		{
			name:       "unlisted is valid",
			visibility: gallery.VisibilityUnlisted,
			want:       true,
		},
		{
			name:       "empty is invalid",
			visibility: "",
			want:       false,
		},
		{
			name:       "random string is invalid",
			visibility: "invalid",
			want:       false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.visibility.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVisibility_IsPublic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		visibility gallery.Visibility
		want       bool
	}{
		{
			name:       "public is public",
			visibility: gallery.VisibilityPublic,
			want:       true,
		},
		{
			name:       "private is not public",
			visibility: gallery.VisibilityPrivate,
			want:       false,
		},
		{
			name:       "unlisted is not public",
			visibility: gallery.VisibilityUnlisted,
			want:       false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.visibility.IsPublic()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVisibility_IsPrivate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		visibility gallery.Visibility
		want       bool
	}{
		{
			name:       "private is private",
			visibility: gallery.VisibilityPrivate,
			want:       true,
		},
		{
			name:       "public is not private",
			visibility: gallery.VisibilityPublic,
			want:       false,
		},
		{
			name:       "unlisted is not private",
			visibility: gallery.VisibilityUnlisted,
			want:       false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.visibility.IsPrivate()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVisibility_IsSearchable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		visibility gallery.Visibility
		want       bool
	}{
		{
			name:       "public is searchable",
			visibility: gallery.VisibilityPublic,
			want:       true,
		},
		{
			name:       "private is not searchable",
			visibility: gallery.VisibilityPrivate,
			want:       false,
		},
		{
			name:       "unlisted is not searchable",
			visibility: gallery.VisibilityUnlisted,
			want:       false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.visibility.IsSearchable()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVisibility_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		visibility gallery.Visibility
		want       string
	}{
		{
			name:       "public",
			visibility: gallery.VisibilityPublic,
			want:       "public",
		},
		{
			name:       "private",
			visibility: gallery.VisibilityPrivate,
			want:       "private",
		},
		{
			name:       "unlisted",
			visibility: gallery.VisibilityUnlisted,
			want:       "unlisted",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.visibility.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAllVisibilities(t *testing.T) {
	t.Parallel()

	visibilities := gallery.AllVisibilities()

	assert.Len(t, visibilities, 3)
	assert.Contains(t, visibilities, gallery.VisibilityPublic)
	assert.Contains(t, visibilities, gallery.VisibilityPrivate)
	assert.Contains(t, visibilities, gallery.VisibilityUnlisted)
}
