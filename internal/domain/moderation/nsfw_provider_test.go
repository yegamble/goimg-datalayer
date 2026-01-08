package moderation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestParseNSFWProvider(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected moderation.NSFWProvider
		wantErr  bool
	}{
		{
			name:     "sightengine provider",
			input:    "sightengine",
			expected: moderation.ProviderSightEngine,
			wantErr:  false,
		},
		{
			name:     "moderatecontent provider",
			input:    "moderatecontent",
			expected: moderation.ProviderModerateContent,
			wantErr:  false,
		},
		{
			name:     "invalid provider",
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
			provider, err := moderation.ParseNSFWProvider(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, moderation.ErrInvalidNSFWProvider)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, provider)
			}
		})
	}
}

func TestNSFWProvider_String(t *testing.T) {
	tests := []struct {
		provider moderation.NSFWProvider
		expected string
	}{
		{moderation.ProviderSightEngine, "sightengine"},
		{moderation.ProviderModerateContent, "moderatecontent"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.provider.String())
		})
	}
}

func TestNSFWProvider_IsValid(t *testing.T) {
	tests := []struct {
		provider moderation.NSFWProvider
		valid    bool
	}{
		{moderation.ProviderSightEngine, true},
		{moderation.ProviderModerateContent, true},
		{moderation.NSFWProvider("invalid"), false},
		{moderation.NSFWProvider(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.provider.IsValid())
		})
	}
}
