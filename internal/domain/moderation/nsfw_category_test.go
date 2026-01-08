package moderation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestParseNSFWCategory(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected moderation.NSFWCategory
		wantErr  bool
	}{
		{
			name:     "safe category",
			input:    "safe",
			expected: moderation.CategorySafe,
			wantErr:  false,
		},
		{
			name:     "suggestive category",
			input:    "suggestive",
			expected: moderation.CategorySuggestive,
			wantErr:  false,
		},
		{
			name:     "nudity category",
			input:    "nudity",
			expected: moderation.CategoryNudity,
			wantErr:  false,
		},
		{
			name:     "explicit category",
			input:    "explicit",
			expected: moderation.CategoryExplicit,
			wantErr:  false,
		},
		{
			name:     "violence category",
			input:    "violence",
			expected: moderation.CategoryViolence,
			wantErr:  false,
		},
		{
			name:     "unknown category",
			input:    "unknown",
			expected: moderation.CategoryUnknown,
			wantErr:  false,
		},
		{
			name:     "invalid category",
			input:    "invalid",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category, err := moderation.ParseNSFWCategory(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, moderation.ErrInvalidNSFWCategory)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, category)
			}
		})
	}
}

func TestNSFWCategory_String(t *testing.T) {
	tests := []struct {
		category moderation.NSFWCategory
		expected string
	}{
		{moderation.CategorySafe, "safe"},
		{moderation.CategorySuggestive, "suggestive"},
		{moderation.CategoryNudity, "nudity"},
		{moderation.CategoryExplicit, "explicit"},
		{moderation.CategoryViolence, "violence"},
		{moderation.CategoryUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.category.String())
		})
	}
}

func TestNSFWCategory_IsValid(t *testing.T) {
	tests := []struct {
		category moderation.NSFWCategory
		valid    bool
	}{
		{moderation.CategorySafe, true},
		{moderation.CategorySuggestive, true},
		{moderation.CategoryNudity, true},
		{moderation.CategoryExplicit, true},
		{moderation.CategoryViolence, true},
		{moderation.CategoryUnknown, true},
		{moderation.NSFWCategory("invalid"), false},
		{moderation.NSFWCategory(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.category.IsValid())
		})
	}
}

func TestNSFWCategory_IsNSFW(t *testing.T) {
	tests := []struct {
		category moderation.NSFWCategory
		isNSFW   bool
	}{
		{moderation.CategorySafe, false},
		{moderation.CategorySuggestive, false},
		{moderation.CategoryNudity, true},
		{moderation.CategoryExplicit, true},
		{moderation.CategoryViolence, true},
		{moderation.CategoryUnknown, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			assert.Equal(t, tt.isNSFW, tt.category.IsNSFW())
		})
	}
}

func TestNSFWCategory_IsSuggestive(t *testing.T) {
	tests := []struct {
		category     moderation.NSFWCategory
		isSuggestive bool
	}{
		{moderation.CategorySafe, false},
		{moderation.CategorySuggestive, true},
		{moderation.CategoryNudity, false},
		{moderation.CategoryExplicit, false},
		{moderation.CategoryViolence, false},
		{moderation.CategoryUnknown, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			assert.Equal(t, tt.isSuggestive, tt.category.IsSuggestive())
		})
	}
}

func TestNSFWCategory_RequiresReview(t *testing.T) {
	tests := []struct {
		category       moderation.NSFWCategory
		requiresReview bool
	}{
		{moderation.CategorySafe, false},
		{moderation.CategorySuggestive, true},
		{moderation.CategoryNudity, true},
		{moderation.CategoryExplicit, true},
		{moderation.CategoryViolence, true},
		{moderation.CategoryUnknown, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			assert.Equal(t, tt.requiresReview, tt.category.RequiresReview())
		})
	}
}

func TestNSFWCategory_Severity(t *testing.T) {
	tests := []struct {
		category moderation.NSFWCategory
		severity int
	}{
		{moderation.CategorySafe, 0},
		{moderation.CategorySuggestive, 1},
		{moderation.CategoryNudity, 2},
		{moderation.CategoryViolence, 3},
		{moderation.CategoryExplicit, 4},
		{moderation.CategoryUnknown, -1},
		{moderation.NSFWCategory("invalid"), -1},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			assert.Equal(t, tt.severity, tt.category.Severity())
		})
	}
}
