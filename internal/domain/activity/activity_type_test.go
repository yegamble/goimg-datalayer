package activity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
)

func TestActivityType_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		activityType activity.ActivityType
		valid        bool
	}{
		{
			name:         "image_uploaded is valid",
			activityType: activity.ActivityTypeImageUploaded,
			valid:        true,
		},
		{
			name:         "image_liked is valid",
			activityType: activity.ActivityTypeImageLiked,
			valid:        true,
		},
		{
			name:         "image_commented is valid",
			activityType: activity.ActivityTypeImageCommented,
			valid:        true,
		},
		{
			name:         "user_followed is valid",
			activityType: activity.ActivityTypeUserFollowed,
			valid:        true,
		},
		{
			name:         "album_created is valid",
			activityType: activity.ActivityTypeAlbumCreated,
			valid:        true,
		},
		{
			name:         "invalid type",
			activityType: activity.ActivityType("invalid"),
			valid:        false,
		},
		{
			name:         "empty type",
			activityType: activity.ActivityType(""),
			valid:        false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.activityType.Valid()
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestActivityType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		activityType activity.ActivityType
		expected     string
	}{
		{
			name:         "image_uploaded",
			activityType: activity.ActivityTypeImageUploaded,
			expected:     "image_uploaded",
		},
		{
			name:         "user_followed",
			activityType: activity.ActivityTypeUserFollowed,
			expected:     "user_followed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.activityType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseActivityType_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected activity.ActivityType
	}{
		{
			name:     "parse image_uploaded",
			input:    "image_uploaded",
			expected: activity.ActivityTypeImageUploaded,
		},
		{
			name:     "parse user_followed",
			input:    "user_followed",
			expected: activity.ActivityTypeUserFollowed,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := activity.ParseActivityType(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseActivityType_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid type",
			input: "invalid_type",
		},
		{
			name:  "empty string",
			input: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := activity.ParseActivityType(tt.input)
			require.Error(t, err)
			assert.Empty(t, result)
		})
	}
}
