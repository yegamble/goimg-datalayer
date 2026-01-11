package nsfw

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMaxFloats tests the maxFloats helper function.
func TestMaxFloats(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{
			name:     "empty slice",
			values:   []float64{},
			expected: 0,
		},
		{
			name:     "single value",
			values:   []float64{5.5},
			expected: 5.5,
		},
		{
			name:     "multiple values with max at start",
			values:   []float64{10.0, 5.0, 3.0, 2.0},
			expected: 10.0,
		},
		{
			name:     "multiple values with max at end",
			values:   []float64{2.0, 3.0, 5.0, 10.0},
			expected: 10.0,
		},
		{
			name:     "multiple values with max in middle",
			values:   []float64{2.0, 10.0, 5.0, 3.0},
			expected: 10.0,
		},
		{
			name:     "all same values",
			values:   []float64{5.0, 5.0, 5.0},
			expected: 5.0,
		},
		{
			name:     "with negative values",
			values:   []float64{-5.0, -2.0, -10.0, -1.0},
			expected: -1.0,
		},
		{
			name:     "with zero",
			values:   []float64{0.0, -5.0, -2.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maxFloats(tt.values...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSightEngineResponse_DetermineCategory tests category determination logic.
func TestSightEngineResponse_DetermineCategory(t *testing.T) {
	tests := []struct {
		name     string
		response sightEngineResponse
		wantCat  string // We'll check the category string representation
		wantMin  float64
		wantMax  float64
	}{
		{
			name: "explicit content - high sexual activity",
			response: sightEngineResponse{
				Nudity: struct {
					Sexual            float64 `json:"sexual_activity"`
					SexualDisplay     float64 `json:"sexual_display"`
					Erotica           float64 `json:"erotica"`
					Sextoy            float64 `json:"sextoy"`
					Suggestive        float64 `json:"suggestive"`
					SuggestiveClasses struct {
						Bikini          float64 `json:"bikini"`
						Lingerie        float64 `json:"lingerie"`
						Cleavage        float64 `json:"cleavage"`
						MiniskirtShorts float64 `json:"miniskirt_or_shorts"`
						MaleChest       float64 `json:"male_chest"`
						MaleUnderwear   float64 `json:"male_underwear"`
					} `json:"suggestive_classes"`
					Safe    float64 `json:"safe"`
					Partial float64 `json:"partial"`
					None    float64 `json:"none"`
				}{
					Sexual:        0.95,
					SexualDisplay: 0.20,
					Erotica:       0.10,
					Suggestive:    0.15,
					Safe:          0.05,
					Partial:       0.10,
				},
			},
			wantMin: 0.90,
			wantMax: 1.0,
		},
		{
			name: "explicit content - high sexual display",
			response: sightEngineResponse{
				Nudity: struct {
					Sexual            float64 `json:"sexual_activity"`
					SexualDisplay     float64 `json:"sexual_display"`
					Erotica           float64 `json:"erotica"`
					Sextoy            float64 `json:"sextoy"`
					Suggestive        float64 `json:"suggestive"`
					SuggestiveClasses struct {
						Bikini          float64 `json:"bikini"`
						Lingerie        float64 `json:"lingerie"`
						Cleavage        float64 `json:"cleavage"`
						MiniskirtShorts float64 `json:"miniskirt_or_shorts"`
						MaleChest       float64 `json:"male_chest"`
						MaleUnderwear   float64 `json:"male_underwear"`
					} `json:"suggestive_classes"`
					Safe    float64 `json:"safe"`
					Partial float64 `json:"partial"`
					None    float64 `json:"none"`
				}{
					Sexual:        0.20,
					SexualDisplay: 0.85,
					Erotica:       0.10,
					Suggestive:    0.15,
					Safe:          0.05,
					Partial:       0.10,
				},
			},
			wantMin: 0.80,
			wantMax: 1.0,
		},
		{
			name: "nudity content - high erotica",
			response: sightEngineResponse{
				Nudity: struct {
					Sexual            float64 `json:"sexual_activity"`
					SexualDisplay     float64 `json:"sexual_display"`
					Erotica           float64 `json:"erotica"`
					Sextoy            float64 `json:"sextoy"`
					Suggestive        float64 `json:"suggestive"`
					SuggestiveClasses struct {
						Bikini          float64 `json:"bikini"`
						Lingerie        float64 `json:"lingerie"`
						Cleavage        float64 `json:"cleavage"`
						MiniskirtShorts float64 `json:"miniskirt_or_shorts"`
						MaleChest       float64 `json:"male_chest"`
						MaleUnderwear   float64 `json:"male_underwear"`
					} `json:"suggestive_classes"`
					Safe    float64 `json:"safe"`
					Partial float64 `json:"partial"`
					None    float64 `json:"none"`
				}{
					Sexual:        0.20,
					SexualDisplay: 0.15,
					Erotica:       0.75,
					Suggestive:    0.25,
					Safe:          0.15,
					Partial:       0.30,
				},
			},
			wantMin: 0.70,
			wantMax: 0.80,
		},
		{
			name: "nudity content - high partial",
			response: sightEngineResponse{
				Nudity: struct {
					Sexual            float64 `json:"sexual_activity"`
					SexualDisplay     float64 `json:"sexual_display"`
					Erotica           float64 `json:"erotica"`
					Sextoy            float64 `json:"sextoy"`
					Suggestive        float64 `json:"suggestive"`
					SuggestiveClasses struct {
						Bikini          float64 `json:"bikini"`
						Lingerie        float64 `json:"lingerie"`
						Cleavage        float64 `json:"cleavage"`
						MiniskirtShorts float64 `json:"miniskirt_or_shorts"`
						MaleChest       float64 `json:"male_chest"`
						MaleUnderwear   float64 `json:"male_underwear"`
					} `json:"suggestive_classes"`
					Safe    float64 `json:"safe"`
					Partial float64 `json:"partial"`
					None    float64 `json:"none"`
				}{
					Sexual:        0.10,
					SexualDisplay: 0.05,
					Erotica:       0.30,
					Suggestive:    0.20,
					Safe:          0.20,
					Partial:       0.65,
				},
			},
			wantMin: 0.60,
			wantMax: 0.70,
		},
		{
			name: "violence content - high weapon",
			response: sightEngineResponse{
				Nudity: struct {
					Sexual            float64 `json:"sexual_activity"`
					SexualDisplay     float64 `json:"sexual_display"`
					Erotica           float64 `json:"erotica"`
					Sextoy            float64 `json:"sextoy"`
					Suggestive        float64 `json:"suggestive"`
					SuggestiveClasses struct {
						Bikini          float64 `json:"bikini"`
						Lingerie        float64 `json:"lingerie"`
						Cleavage        float64 `json:"cleavage"`
						MiniskirtShorts float64 `json:"miniskirt_or_shorts"`
						MaleChest       float64 `json:"male_chest"`
						MaleUnderwear   float64 `json:"male_underwear"`
					} `json:"suggestive_classes"`
					Safe    float64 `json:"safe"`
					Partial float64 `json:"partial"`
					None    float64 `json:"none"`
				}{
					Sexual:        0.05,
					SexualDisplay: 0.02,
					Erotica:       0.10,
					Suggestive:    0.15,
					Safe:          0.30,
					Partial:       0.05,
				},
				Weapon: 0.85,
			},
			wantMin: 0.80,
			wantMax: 0.90,
		},
		{
			name: "suggestive content",
			response: sightEngineResponse{
				Nudity: struct {
					Sexual            float64 `json:"sexual_activity"`
					SexualDisplay     float64 `json:"sexual_display"`
					Erotica           float64 `json:"erotica"`
					Sextoy            float64 `json:"sextoy"`
					Suggestive        float64 `json:"suggestive"`
					SuggestiveClasses struct {
						Bikini          float64 `json:"bikini"`
						Lingerie        float64 `json:"lingerie"`
						Cleavage        float64 `json:"cleavage"`
						MiniskirtShorts float64 `json:"miniskirt_or_shorts"`
						MaleChest       float64 `json:"male_chest"`
						MaleUnderwear   float64 `json:"male_underwear"`
					} `json:"suggestive_classes"`
					Safe    float64 `json:"safe"`
					Partial float64 `json:"partial"`
					None    float64 `json:"none"`
				}{
					Sexual:        0.10,
					SexualDisplay: 0.08,
					Erotica:       0.20,
					Suggestive:    0.65,
					Safe:          0.25,
					Partial:       0.15,
				},
			},
			wantMin: 0.60,
			wantMax: 0.70,
		},
		{
			name: "safe content",
			response: sightEngineResponse{
				Nudity: struct {
					Sexual            float64 `json:"sexual_activity"`
					SexualDisplay     float64 `json:"sexual_display"`
					Erotica           float64 `json:"erotica"`
					Sextoy            float64 `json:"sextoy"`
					Suggestive        float64 `json:"suggestive"`
					SuggestiveClasses struct {
						Bikini          float64 `json:"bikini"`
						Lingerie        float64 `json:"lingerie"`
						Cleavage        float64 `json:"cleavage"`
						MiniskirtShorts float64 `json:"miniskirt_or_shorts"`
						MaleChest       float64 `json:"male_chest"`
						MaleUnderwear   float64 `json:"male_underwear"`
					} `json:"suggestive_classes"`
					Safe    float64 `json:"safe"`
					Partial float64 `json:"partial"`
					None    float64 `json:"none"`
				}{
					Sexual:        0.01,
					SexualDisplay: 0.01,
					Erotica:       0.05,
					Suggestive:    0.10,
					Safe:          0.95,
					Partial:       0.02,
				},
			},
			wantMin: 0.90,
			wantMax: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &SightEngineClient{}
			category, score := client.determineCategory(tt.response)

			assert.NotEmpty(t, category)
			assert.GreaterOrEqual(t, score, tt.wantMin)
			assert.LessOrEqual(t, score, tt.wantMax)
		})
	}
}

// TestModerateContentResponse_DetermineCategory tests ModerateContent category logic.
func TestModerateContentResponse_DetermineCategory(t *testing.T) {
	tests := []struct {
		name     string
		response moderateContentResponse
		wantMin  float64
		wantMax  float64
	}{
		{
			name: "everyone rating",
			response: moderateContentResponse{
				RatingIndex: 1,
				Predictions: struct {
					Everyone float64 `json:"everyone"`
					Teen     float64 `json:"teen"`
					Adult    float64 `json:"adult"`
				}{
					Everyone: 0.95,
					Teen:     0.04,
					Adult:    0.01,
				},
			},
			wantMin: 0.90,
			wantMax: 1.0,
		},
		{
			name: "teen rating",
			response: moderateContentResponse{
				RatingIndex: 2,
				Predictions: struct {
					Everyone float64 `json:"everyone"`
					Teen     float64 `json:"teen"`
					Adult    float64 `json:"adult"`
				}{
					Everyone: 0.10,
					Teen:     0.80,
					Adult:    0.10,
				},
			},
			wantMin: 0.75,
			wantMax: 0.85,
		},
		{
			name: "adult rating - explicit (high score)",
			response: moderateContentResponse{
				RatingIndex: 3,
				Predictions: struct {
					Everyone float64 `json:"everyone"`
					Teen     float64 `json:"teen"`
					Adult    float64 `json:"adult"`
				}{
					Everyone: 0.01,
					Teen:     0.04,
					Adult:    0.95,
				},
			},
			wantMin: 0.90,
			wantMax: 1.0,
		},
		{
			name: "adult rating - nudity (medium score)",
			response: moderateContentResponse{
				RatingIndex: 3,
				Predictions: struct {
					Everyone float64 `json:"everyone"`
					Teen     float64 `json:"teen"`
					Adult    float64 `json:"adult"`
				}{
					Everyone: 0.10,
					Teen:     0.20,
					Adult:    0.70,
				},
			},
			wantMin: 0.65,
			wantMax: 0.75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &ModerateContentClient{}
			category, score := client.determineCategory(tt.response)

			assert.NotEmpty(t, category)
			assert.GreaterOrEqual(t, score, tt.wantMin)
			assert.LessOrEqual(t, score, tt.wantMax)
		})
	}
}
