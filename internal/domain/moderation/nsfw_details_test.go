package moderation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestNewNSFWDetails(t *testing.T) {
	details := moderation.NewNSFWDetails(0.5, 0.3, 0.2, 0.1, 0.05)

	assert.Equal(t, 0.5, details.NudityScore)
	assert.Equal(t, 0.3, details.WeaponScore)
	assert.Equal(t, 0.2, details.ViolenceScore)
	assert.Equal(t, 0.1, details.OffensiveScore)
	assert.Equal(t, 0.05, details.DrugScore)
	assert.Empty(t, details.SubCategories)
	assert.Nil(t, details.RawResponse)
}

func TestNewNSFWDetails_Clamping(t *testing.T) {
	t.Run("clamps negative values to 0", func(t *testing.T) {
		details := moderation.NewNSFWDetails(-0.5, -1.0, -0.1, -10, -100)

		assert.Equal(t, 0.0, details.NudityScore)
		assert.Equal(t, 0.0, details.WeaponScore)
		assert.Equal(t, 0.0, details.ViolenceScore)
		assert.Equal(t, 0.0, details.OffensiveScore)
		assert.Equal(t, 0.0, details.DrugScore)
	})

	t.Run("clamps values above 1 to 1", func(t *testing.T) {
		details := moderation.NewNSFWDetails(1.5, 2.0, 100, 10, 1.1)

		assert.Equal(t, 1.0, details.NudityScore)
		assert.Equal(t, 1.0, details.WeaponScore)
		assert.Equal(t, 1.0, details.ViolenceScore)
		assert.Equal(t, 1.0, details.OffensiveScore)
		assert.Equal(t, 1.0, details.DrugScore)
	})
}

func TestNSFWDetails_WithSubCategories(t *testing.T) {
	details := moderation.NewNSFWDetails(0.5, 0, 0, 0, 0)
	categories := []string{"partial_nudity", "bikini"}

	updated := details.WithSubCategories(categories)

	assert.Equal(t, categories, updated.SubCategories)
	assert.Empty(t, details.SubCategories) // Original is unchanged
}

func TestNSFWDetails_WithRawResponse(t *testing.T) {
	details := moderation.NewNSFWDetails(0.5, 0, 0, 0, 0)
	raw := map[string]interface{}{
		"nudity": 0.5,
		"raw":    true,
	}

	updated := details.WithRawResponse(raw)

	assert.Equal(t, raw, updated.RawResponse)
	assert.Nil(t, details.RawResponse) // Original is unchanged
}

func TestNSFWDetails_MaxScore(t *testing.T) {
	tests := []struct {
		name     string
		details  moderation.NSFWDetails
		expected float64
	}{
		{
			name:     "nudity highest",
			details:  moderation.NewNSFWDetails(0.8, 0.3, 0.2, 0.1, 0.05),
			expected: 0.8,
		},
		{
			name:     "weapon highest",
			details:  moderation.NewNSFWDetails(0.3, 0.9, 0.2, 0.1, 0.05),
			expected: 0.9,
		},
		{
			name:     "violence highest",
			details:  moderation.NewNSFWDetails(0.3, 0.2, 0.95, 0.1, 0.05),
			expected: 0.95,
		},
		{
			name:     "offensive highest",
			details:  moderation.NewNSFWDetails(0.3, 0.2, 0.1, 0.85, 0.05),
			expected: 0.85,
		},
		{
			name:     "drug highest",
			details:  moderation.NewNSFWDetails(0.1, 0.2, 0.1, 0.15, 0.75),
			expected: 0.75,
		},
		{
			name:     "all zeros",
			details:  moderation.NewNSFWDetails(0, 0, 0, 0, 0),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, tt.details.MaxScore(), 0.001)
		})
	}
}

func TestNSFWDetails_DominantCategory(t *testing.T) {
	tests := []struct {
		name     string
		details  moderation.NSFWDetails
		expected moderation.NSFWCategory
	}{
		{
			name:     "nudity dominant",
			details:  moderation.NewNSFWDetails(0.8, 0.3, 0.2, 0.1, 0.05),
			expected: moderation.CategoryNudity,
		},
		{
			name:     "violence dominant (weapon high)",
			details:  moderation.NewNSFWDetails(0.3, 0.9, 0.2, 0.1, 0.05),
			expected: moderation.CategoryViolence,
		},
		{
			name:     "violence dominant (violence high)",
			details:  moderation.NewNSFWDetails(0.3, 0.2, 0.85, 0.1, 0.05),
			expected: moderation.CategoryViolence,
		},
		{
			name:     "explicit dominant",
			details:  moderation.NewNSFWDetails(0.3, 0.2, 0.1, 0.85, 0.05),
			expected: moderation.CategoryExplicit,
		},
		{
			name:     "safe when all scores below threshold",
			details:  moderation.NewNSFWDetails(0.2, 0.1, 0.1, 0.1, 0.05),
			expected: moderation.CategorySafe,
		},
		{
			name:     "safe when all zeros",
			details:  moderation.NewNSFWDetails(0, 0, 0, 0, 0),
			expected: moderation.CategorySafe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.details.DominantCategory())
		})
	}
}

func TestNSFWDetails_HasSubCategory(t *testing.T) {
	details := moderation.NewNSFWDetails(0.5, 0, 0, 0, 0).
		WithSubCategories([]string{"partial_nudity", "bikini"})

	t.Run("returns true for existing subcategory", func(t *testing.T) {
		assert.True(t, details.HasSubCategory("partial_nudity"))
		assert.True(t, details.HasSubCategory("bikini"))
	})

	t.Run("returns false for non-existing subcategory", func(t *testing.T) {
		assert.False(t, details.HasSubCategory("explicit"))
		assert.False(t, details.HasSubCategory(""))
	})
}

func TestNSFWDetails_IsEmpty(t *testing.T) {
	t.Run("returns true when all zeros", func(t *testing.T) {
		details := moderation.NewNSFWDetails(0, 0, 0, 0, 0)
		assert.True(t, details.IsEmpty())
	})

	t.Run("returns false when any score is non-zero", func(t *testing.T) {
		details := moderation.NewNSFWDetails(0.1, 0, 0, 0, 0)
		assert.False(t, details.IsEmpty())
	})

	t.Run("returns true for zero value struct", func(t *testing.T) {
		var details moderation.NSFWDetails
		assert.True(t, details.IsEmpty())
	})
}
