package nsfw_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/nsfw"
)

func TestDefaultConfig(t *testing.T) {
	cfg := nsfw.DefaultConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 10*time.Second, cfg.Timeout)
	assert.Equal(t, 2, cfg.RetryAttempts)
	assert.Equal(t, 500*time.Millisecond, cfg.RetryDelay)
	assert.True(t, cfg.FailOpen)
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    nsfw.Config
		expected nsfw.Config
	}{
		{
			name:  "valid config unchanged",
			input: nsfw.DefaultConfig(),
			expected: nsfw.Config{
				Enabled:       true,
				Timeout:       10 * time.Second,
				RetryAttempts: 2,
				RetryDelay:    500 * time.Millisecond,
				FailOpen:      true,
			},
		},
		{
			name: "zero timeout gets default",
			input: nsfw.Config{
				Enabled:       true,
				Timeout:       0,
				RetryAttempts: 1,
				RetryDelay:    time.Second,
				FailOpen:      true,
			},
			expected: nsfw.Config{
				Enabled:       true,
				Timeout:       10 * time.Second,
				RetryAttempts: 1,
				RetryDelay:    time.Second,
				FailOpen:      true,
			},
		},
		{
			name: "negative retry attempts set to 0",
			input: nsfw.Config{
				Enabled:       true,
				Timeout:       5 * time.Second,
				RetryAttempts: -1,
				RetryDelay:    time.Second,
				FailOpen:      true,
			},
			expected: nsfw.Config{
				Enabled:       true,
				Timeout:       5 * time.Second,
				RetryAttempts: 0,
				RetryDelay:    time.Second,
				FailOpen:      true,
			},
		},
		{
			name: "zero retry delay gets default",
			input: nsfw.Config{
				Enabled:       true,
				Timeout:       5 * time.Second,
				RetryAttempts: 1,
				RetryDelay:    0,
				FailOpen:      true,
			},
			expected: nsfw.Config{
				Enabled:       true,
				Timeout:       5 * time.Second,
				RetryAttempts: 1,
				RetryDelay:    500 * time.Millisecond,
				FailOpen:      true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nsfw.ValidateConfig(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
