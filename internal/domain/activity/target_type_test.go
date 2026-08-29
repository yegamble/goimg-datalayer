package activity_test

import (
	"testing"

	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
)

func TestTargetType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected activity.TargetType
		valid    bool
	}{
		{"Image", "image", activity.TargetTypeImage, true},
		{"User", "user", activity.TargetTypeUser, true},
		{"Album", "album", activity.TargetTypeAlbum, true},
		{"Comment", "comment", activity.TargetTypeComment, true},
		{"Invalid", "invalid", activity.TargetType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := activity.ParseTargetType(tt.input)
			if tt.valid {
				if err != nil {
					t.Fatalf("expected valid, got error: %v", err)
				}
				if parsed != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, parsed)
				}
				if parsed.String() != tt.input {
					t.Errorf("expected string %v, got %v", tt.input, parsed.String())
				}
			} else {
				if err == nil {
					t.Fatalf("expected error, got valid")
				}
			}
		})
	}
}
