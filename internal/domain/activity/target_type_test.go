package activity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
)

func TestTargetType_String(t *testing.T) {
	tests := []struct {
		name       string
		targetType activity.TargetType
		expected   string
	}{
		{"Image", activity.TargetTypeImage, "image"},
		{"User", activity.TargetTypeUser, "user"},
		{"Album", activity.TargetTypeAlbum, "album"},
		{"Comment", activity.TargetTypeComment, "comment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.targetType.String())
		})
	}
}

func TestParseTargetType(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    activity.TargetType
		expectError bool
	}{
		{"Valid Image", "image", activity.TargetTypeImage, false},
		{"Valid User", "user", activity.TargetTypeUser, false},
		{"Valid Album", "album", activity.TargetTypeAlbum, false},
		{"Valid Comment", "comment", activity.TargetTypeComment, false},
		{"Invalid Type", "invalid", "", true},
		{"Empty String", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := activity.ParseTargetType(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid target type")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
