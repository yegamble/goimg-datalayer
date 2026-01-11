package nsfw_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/nsfw"
)

// mockClient is a test helper implementing the Client interface.
type mockClient struct {
	provider      moderation.NSFWProvider
	available     bool
	scanResult    *nsfw.ScanResult
	scanError     error
	scanCallCount int
}

func (m *mockClient) Scan(ctx context.Context, imageURL string) (*nsfw.ScanResult, error) {
	m.scanCallCount++
	if m.scanError != nil {
		return nil, m.scanError
	}
	return m.scanResult, nil
}

func (m *mockClient) ScanBytes(ctx context.Context, data []byte, contentType string) (*nsfw.ScanResult, error) {
	return m.Scan(ctx, "data-uri")
}

func (m *mockClient) Provider() moderation.NSFWProvider {
	return m.provider
}

func (m *mockClient) IsAvailable(ctx context.Context) bool {
	return m.available
}

func TestOrchestrator_Scan_Disabled(t *testing.T) {
	primary := &mockClient{
		provider:  moderation.ProviderSightEngine,
		available: true,
	}

	config := nsfw.DefaultOrchestratorConfig()
	config.Enabled = false

	orch := nsfw.NewOrchestrator(primary, nil, config, nil)
	result, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

	require.NoError(t, err)
	assert.Equal(t, moderation.CategorySafe, result.Category)
	assert.Equal(t, 0, primary.scanCallCount) // Primary should not be called
}

func TestOrchestrator_Scan_PrimarySuccess(t *testing.T) {
	expectedResult := &nsfw.ScanResult{
		Category: moderation.CategoryNudity,
		Score:    0.85,
		Provider: moderation.ProviderSightEngine,
	}

	primary := &mockClient{
		provider:   moderation.ProviderSightEngine,
		available:  true,
		scanResult: expectedResult,
	}

	fallback := &mockClient{
		provider:  moderation.ProviderModerateContent,
		available: true,
	}

	config := nsfw.DefaultOrchestratorConfig()
	orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, nil)

	result, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

	require.NoError(t, err)
	assert.Equal(t, moderation.CategoryNudity, result.Category)
	assert.Equal(t, 0.85, result.Score)
	assert.Equal(t, moderation.ProviderSightEngine, result.Provider)
	assert.Equal(t, 1, primary.scanCallCount)
	assert.Equal(t, 0, fallback.scanCallCount) // Fallback should not be called
}

func TestOrchestrator_Scan_PrimaryFailsFallbackSucceeds(t *testing.T) {
	fallbackResult := &nsfw.ScanResult{
		Category: moderation.CategorySafe,
		Score:    0.95,
		Provider: moderation.ProviderModerateContent,
	}

	primary := &mockClient{
		provider:  moderation.ProviderSightEngine,
		available: true,
		scanError: errors.New("primary failed"),
	}

	fallback := &mockClient{
		provider:   moderation.ProviderModerateContent,
		available:  true,
		scanResult: fallbackResult,
	}

	config := nsfw.DefaultOrchestratorConfig()
	orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, nil)

	result, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

	require.NoError(t, err)
	assert.Equal(t, moderation.CategorySafe, result.Category)
	assert.Equal(t, moderation.ProviderModerateContent, result.Provider)
	assert.Equal(t, 1, primary.scanCallCount)
	assert.Equal(t, 1, fallback.scanCallCount)
}

func TestOrchestrator_Scan_AllProvidersFailFailOpen(t *testing.T) {
	primary := &mockClient{
		provider:  moderation.ProviderSightEngine,
		available: true,
		scanError: errors.New("primary failed"),
	}

	fallback := &mockClient{
		provider:  moderation.ProviderModerateContent,
		available: true,
		scanError: errors.New("fallback failed"),
	}

	config := nsfw.DefaultOrchestratorConfig()
	config.FailOpen = true

	orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, nil)
	result, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

	require.NoError(t, err)
	assert.Equal(t, moderation.CategoryUnknown, result.Category)
}

func TestOrchestrator_Scan_AllProvidersFailFailClosed(t *testing.T) {
	primary := &mockClient{
		provider:  moderation.ProviderSightEngine,
		available: true,
		scanError: errors.New("primary failed"),
	}

	fallback := &mockClient{
		provider:  moderation.ProviderModerateContent,
		available: true,
		scanError: errors.New("fallback failed"),
	}

	config := nsfw.DefaultOrchestratorConfig()
	config.FailOpen = false

	orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, nil)
	_, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "all NSFW detection providers failed")
}

func TestOrchestrator_Scan_PrimaryUnavailable(t *testing.T) {
	fallbackResult := &nsfw.ScanResult{
		Category: moderation.CategorySafe,
		Score:    0.9,
		Provider: moderation.ProviderModerateContent,
	}

	primary := &mockClient{
		provider:  moderation.ProviderSightEngine,
		available: false, // Primary unavailable
	}

	fallback := &mockClient{
		provider:   moderation.ProviderModerateContent,
		available:  true,
		scanResult: fallbackResult,
	}

	config := nsfw.DefaultOrchestratorConfig()
	orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, nil)

	result, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

	require.NoError(t, err)
	assert.Equal(t, moderation.ProviderModerateContent, result.Provider)
	assert.Equal(t, 0, primary.scanCallCount) // Primary should not be called when unavailable
	assert.Equal(t, 1, fallback.scanCallCount)
}

func TestOrchestrator_IsAvailable(t *testing.T) {
	t.Run("unavailable when disabled", func(t *testing.T) {
		primary := &mockClient{available: true}
		config := nsfw.DefaultOrchestratorConfig()
		config.Enabled = false

		orch := nsfw.NewOrchestrator(primary, nil, config, nil)
		assert.False(t, orch.IsAvailable(context.Background()))
	})

	t.Run("available when primary is available", func(t *testing.T) {
		primary := &mockClient{available: true}
		config := nsfw.DefaultOrchestratorConfig()

		orch := nsfw.NewOrchestrator(primary, nil, config, nil)
		assert.True(t, orch.IsAvailable(context.Background()))
	})

	t.Run("available when fallback is available", func(t *testing.T) {
		primary := &mockClient{available: false}
		fallback := &mockClient{available: true}
		config := nsfw.DefaultOrchestratorConfig()

		orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, nil)
		assert.True(t, orch.IsAvailable(context.Background()))
	})

	t.Run("unavailable when all providers unavailable", func(t *testing.T) {
		primary := &mockClient{available: false}
		fallback := &mockClient{available: false}
		config := nsfw.DefaultOrchestratorConfig()

		orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, nil)
		assert.False(t, orch.IsAvailable(context.Background()))
	})
}

func TestOrchestrator_ScanBytes(t *testing.T) {
	expectedResult := &nsfw.ScanResult{
		Category:     moderation.CategorySafe,
		Score:        0.95,
		Provider:     moderation.ProviderSightEngine,
		ScanDuration: time.Millisecond,
	}

	primary := &mockClient{
		provider:   moderation.ProviderSightEngine,
		available:  true,
		scanResult: expectedResult,
	}

	config := nsfw.DefaultOrchestratorConfig()
	orch := nsfw.NewOrchestrator(primary, nil, config, nil)

	result, err := orch.ScanBytes(context.Background(), []byte("fake-image"), "image/jpeg")

	require.NoError(t, err)
	assert.Equal(t, moderation.CategorySafe, result.Category)
}

func TestOrchestrator_ScanBytes_Disabled(t *testing.T) {
	primary := &mockClient{
		provider:  moderation.ProviderSightEngine,
		available: true,
	}

	config := nsfw.DefaultOrchestratorConfig()
	config.Enabled = false

	orch := nsfw.NewOrchestrator(primary, nil, config, nil)
	result, err := orch.ScanBytes(context.Background(), []byte("fake-image"), "image/jpeg")

	require.NoError(t, err)
	assert.Equal(t, moderation.CategorySafe, result.Category)
	assert.Equal(t, 0, primary.scanCallCount)
}

func TestOrchestrator_ScanBytes_PrimaryFailsFallbackSucceeds(t *testing.T) {
	fallbackResult := &nsfw.ScanResult{
		Category: moderation.CategorySafe,
		Score:    0.90,
		Provider: moderation.ProviderModerateContent,
	}

	primary := &mockClient{
		provider:  moderation.ProviderSightEngine,
		available: true,
		scanError: errors.New("primary scan bytes failed"),
	}

	fallback := &mockClient{
		provider:   moderation.ProviderModerateContent,
		available:  true,
		scanResult: fallbackResult,
	}

	config := nsfw.DefaultOrchestratorConfig()
	orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, nil)

	result, err := orch.ScanBytes(context.Background(), []byte("fake-image"), "image/jpeg")

	require.NoError(t, err)
	assert.Equal(t, moderation.ProviderModerateContent, result.Provider)
	assert.Equal(t, 1, primary.scanCallCount)
	assert.Equal(t, 1, fallback.scanCallCount)
}

func TestOrchestrator_ScanBytes_AllProvidersFail_FailOpen(t *testing.T) {
	primary := &mockClient{
		provider:  moderation.ProviderSightEngine,
		available: true,
		scanError: errors.New("primary failed"),
	}

	fallback := &mockClient{
		provider:  moderation.ProviderModerateContent,
		available: true,
		scanError: errors.New("fallback failed"),
	}

	config := nsfw.DefaultOrchestratorConfig()
	config.FailOpen = true

	orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, nil)
	result, err := orch.ScanBytes(context.Background(), []byte("fake-image"), "image/jpeg")

	require.NoError(t, err)
	assert.Equal(t, moderation.CategoryUnknown, result.Category)
}

func TestOrchestrator_ScanBytes_AllProvidersFail_FailClosed(t *testing.T) {
	primary := &mockClient{
		provider:  moderation.ProviderSightEngine,
		available: true,
		scanError: errors.New("primary failed"),
	}

	fallback := &mockClient{
		provider:  moderation.ProviderModerateContent,
		available: true,
		scanError: errors.New("fallback failed"),
	}

	config := nsfw.DefaultOrchestratorConfig()
	config.FailOpen = false

	orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, nil)
	_, err := orch.ScanBytes(context.Background(), []byte("fake-image"), "image/jpeg")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "all NSFW detection providers failed")
}

func TestOrchestrator_ScanBytes_PrimaryUnavailable(t *testing.T) {
	fallbackResult := &nsfw.ScanResult{
		Category: moderation.CategorySafe,
		Score:    0.92,
		Provider: moderation.ProviderModerateContent,
	}

	primary := &mockClient{
		provider:  moderation.ProviderSightEngine,
		available: false,
	}

	fallback := &mockClient{
		provider:   moderation.ProviderModerateContent,
		available:  true,
		scanResult: fallbackResult,
	}

	config := nsfw.DefaultOrchestratorConfig()
	orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, nil)

	result, err := orch.ScanBytes(context.Background(), []byte("fake-image"), "image/jpeg")

	require.NoError(t, err)
	assert.Equal(t, moderation.ProviderModerateContent, result.Provider)
	assert.Equal(t, 0, primary.scanCallCount)
	assert.Equal(t, 1, fallback.scanCallCount)
}

func TestOrchestrator_Scan_MaxAttempts(t *testing.T) {
	t.Run("max attempts limits fallback providers", func(t *testing.T) {
		primary := &mockClient{
			provider:  moderation.ProviderSightEngine,
			available: true,
			scanError: errors.New("primary failed"),
		}

		fallback1 := &mockClient{
			provider:  moderation.ProviderModerateContent,
			available: true,
			scanError: errors.New("fallback1 failed"),
		}

		fallback2 := &mockClient{
			provider:  moderation.ProviderModerateContent,
			available: true,
			scanError: errors.New("fallback2 failed"),
		}

		config := nsfw.DefaultOrchestratorConfig()
		config.MaxAttempts = 2 // Primary + 1 fallback only
		config.FailOpen = true

		orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback1, fallback2}, config, nil)
		result, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

		require.NoError(t, err)
		assert.Equal(t, moderation.CategoryUnknown, result.Category)
		assert.Equal(t, 1, primary.scanCallCount)
		assert.Equal(t, 1, fallback1.scanCallCount)
		assert.Equal(t, 0, fallback2.scanCallCount) // Should not be called due to MaxAttempts
	})

	t.Run("max attempts 0 uses all providers", func(t *testing.T) {
		fallbackResult := &nsfw.ScanResult{
			Category: moderation.CategorySafe,
			Score:    0.9,
			Provider: moderation.ProviderModerateContent,
		}

		primary := &mockClient{
			provider:  moderation.ProviderSightEngine,
			available: true,
			scanError: errors.New("primary failed"),
		}

		fallback1 := &mockClient{
			provider:  moderation.ProviderModerateContent,
			available: true,
			scanError: errors.New("fallback1 failed"),
		}

		fallback2 := &mockClient{
			provider:   moderation.ProviderModerateContent,
			available:  true,
			scanResult: fallbackResult,
		}

		config := nsfw.DefaultOrchestratorConfig()
		config.MaxAttempts = 0 // Use all providers

		orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback1, fallback2}, config, nil)
		result, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

		require.NoError(t, err)
		assert.Equal(t, moderation.CategorySafe, result.Category)
		assert.Equal(t, 1, primary.scanCallCount)
		assert.Equal(t, 1, fallback1.scanCallCount)
		assert.Equal(t, 1, fallback2.scanCallCount) // All fallbacks tried
	})

	t.Run("max attempts 1 only tries primary", func(t *testing.T) {
		primary := &mockClient{
			provider:  moderation.ProviderSightEngine,
			available: true,
			scanError: errors.New("primary failed"),
		}

		fallback := &mockClient{
			provider:  moderation.ProviderModerateContent,
			available: true,
		}

		config := nsfw.DefaultOrchestratorConfig()
		config.MaxAttempts = 1 // Only primary, no fallbacks
		config.FailOpen = true

		orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, nil)
		result, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

		require.NoError(t, err)
		assert.Equal(t, moderation.CategoryUnknown, result.Category)
		assert.Equal(t, 1, primary.scanCallCount)
		assert.Equal(t, 0, fallback.scanCallCount) // Fallback should not be tried
	})
}

func TestOrchestrator_Scan_NilPrimary(t *testing.T) {
	fallbackResult := &nsfw.ScanResult{
		Category: moderation.CategorySafe,
		Score:    0.9,
		Provider: moderation.ProviderModerateContent,
	}

	fallback := &mockClient{
		provider:   moderation.ProviderModerateContent,
		available:  true,
		scanResult: fallbackResult,
	}

	config := nsfw.DefaultOrchestratorConfig()
	orch := nsfw.NewOrchestrator(nil, []nsfw.Client{fallback}, config, nil)

	result, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

	require.NoError(t, err)
	assert.Equal(t, moderation.CategorySafe, result.Category)
	assert.Equal(t, moderation.ProviderModerateContent, result.Provider)
	assert.Equal(t, 1, fallback.scanCallCount)
}

func TestOrchestrator_Scan_MultipleFallbacksSkipUnavailable(t *testing.T) {
	thirdFallbackResult := &nsfw.ScanResult{
		Category: moderation.CategorySafe,
		Score:    0.88,
		Provider: moderation.ProviderModerateContent,
	}

	primary := &mockClient{
		provider:  moderation.ProviderSightEngine,
		available: true,
		scanError: errors.New("primary failed"),
	}

	fallback1 := &mockClient{
		provider:  moderation.ProviderModerateContent,
		available: false, // Unavailable
	}

	fallback2 := &mockClient{
		provider:  moderation.ProviderModerateContent,
		available: true,
		scanError: errors.New("fallback2 failed"),
	}

	fallback3 := &mockClient{
		provider:   moderation.ProviderModerateContent,
		available:  true,
		scanResult: thirdFallbackResult,
	}

	config := nsfw.DefaultOrchestratorConfig()
	orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback1, fallback2, fallback3}, config, nil)

	result, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

	require.NoError(t, err)
	assert.Equal(t, moderation.CategorySafe, result.Category)
	assert.Equal(t, 1, primary.scanCallCount)
	assert.Equal(t, 0, fallback1.scanCallCount) // Skipped due to unavailable
	assert.Equal(t, 1, fallback2.scanCallCount)
	assert.Equal(t, 1, fallback3.scanCallCount)
}

func TestOrchestrator_WithLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	t.Run("logs when disabled", func(t *testing.T) {
		buf.Reset()
		primary := &mockClient{
			provider:  moderation.ProviderSightEngine,
			available: true,
		}

		config := nsfw.DefaultOrchestratorConfig()
		config.Enabled = false

		orch := nsfw.NewOrchestrator(primary, nil, config, &logger)
		_, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

		require.NoError(t, err)
		assert.Contains(t, buf.String(), "disabled")
	})

	t.Run("logs primary provider failure", func(t *testing.T) {
		buf.Reset()
		primary := &mockClient{
			provider:  moderation.ProviderSightEngine,
			available: true,
			scanError: errors.New("primary scan failed"),
		}

		fallback := &mockClient{
			provider:  moderation.ProviderModerateContent,
			available: true,
			scanResult: &nsfw.ScanResult{
				Category: moderation.CategorySafe,
				Score:    0.9,
				Provider: moderation.ProviderModerateContent,
			},
		}

		config := nsfw.DefaultOrchestratorConfig()
		orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, &logger)
		_, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

		require.NoError(t, err)
		// Logger should have logged the warning about primary failure
		output := buf.String()
		assert.True(t, len(output) > 0)
	})

	t.Run("logs when all providers fail", func(t *testing.T) {
		buf.Reset()
		primary := &mockClient{
			provider:  moderation.ProviderSightEngine,
			available: true,
			scanError: errors.New("primary failed"),
		}

		fallback := &mockClient{
			provider:  moderation.ProviderModerateContent,
			available: true,
			scanError: errors.New("fallback failed"),
		}

		config := nsfw.DefaultOrchestratorConfig()
		config.FailOpen = true

		orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, &logger)
		_, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

		require.NoError(t, err)
		output := buf.String()
		assert.True(t, len(output) > 0)
	})

	t.Run("logs primary unavailable", func(t *testing.T) {
		buf.Reset()
		primary := &mockClient{
			provider:  moderation.ProviderSightEngine,
			available: false,
		}

		fallback := &mockClient{
			provider:  moderation.ProviderModerateContent,
			available: true,
			scanResult: &nsfw.ScanResult{
				Category: moderation.CategorySafe,
				Score:    0.9,
				Provider: moderation.ProviderModerateContent,
			},
		}

		config := nsfw.DefaultOrchestratorConfig()
		orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback}, config, &logger)
		_, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

		require.NoError(t, err)
		output := buf.String()
		assert.True(t, len(output) > 0)
	})

	t.Run("logs fallback provider failures", func(t *testing.T) {
		buf.Reset()
		primary := &mockClient{
			provider:  moderation.ProviderSightEngine,
			available: true,
			scanError: errors.New("primary failed"),
		}

		fallback1 := &mockClient{
			provider:  moderation.ProviderModerateContent,
			available: true,
			scanError: errors.New("fallback1 failed"),
		}

		fallback2 := &mockClient{
			provider:  moderation.ProviderModerateContent,
			available: true,
			scanResult: &nsfw.ScanResult{
				Category: moderation.CategorySafe,
				Score:    0.85,
				Provider: moderation.ProviderModerateContent,
			},
		}

		config := nsfw.DefaultOrchestratorConfig()
		orch := nsfw.NewOrchestrator(primary, []nsfw.Client{fallback1, fallback2}, config, &logger)
		_, err := orch.Scan(context.Background(), "http://example.com/image.jpg")

		require.NoError(t, err)
		output := buf.String()
		// Should log warnings for both primary and fallback1 failures
		assert.True(t, len(output) > 0)
	})
}
