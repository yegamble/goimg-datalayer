package nsfw_test

import (
	"context"
	"errors"
	"testing"
	"time"

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
