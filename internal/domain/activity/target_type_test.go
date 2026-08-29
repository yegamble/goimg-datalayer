package activity

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTargetType(t *testing.T) {
	tests := []struct {
		name     string
		tt       TargetType
		expected string
	}{
		{"Image", TargetTypeImage, "image"},
		{"Album", TargetTypeAlbum, "album"},
		{"User", TargetTypeUser, "user"},
		{"Comment", TargetTypeComment, "comment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.tt.String())
			parsed, err := ParseTargetType(tt.expected)
			assert.NoError(t, err)
			assert.Equal(t, tt.tt, parsed)
		})
	}

	_, err := ParseTargetType("invalid")
	assert.Error(t, err)
}
