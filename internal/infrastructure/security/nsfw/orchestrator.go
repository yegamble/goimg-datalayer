package nsfw

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// Orchestrator manages multiple NSFW detection providers with automatic fallback.
// It tries the primary provider first, falling back to secondary providers if needed.
type Orchestrator struct {
	primary   Client
	fallbacks []Client
	config    OrchestratorConfig
	logger    *zerolog.Logger
}

// OrchestratorConfig holds configuration for the orchestrator.
type OrchestratorConfig struct {
	// Enabled determines whether NSFW detection is active.
	// When false, all scans return safe category.
	Enabled bool

	// FailOpen determines behavior when all providers fail.
	// - true: Return unknown category (prioritizes availability)
	// - false: Return error (prioritizes security)
	FailOpen bool

	// MaxAttempts is the total number of provider attempts before giving up.
	// This includes the primary provider and all fallbacks.
	// Default: uses all configured providers
	MaxAttempts int
}

// DefaultOrchestratorConfig returns the default orchestrator configuration.
func DefaultOrchestratorConfig() OrchestratorConfig {
	return OrchestratorConfig{
		Enabled:     true,
		FailOpen:    true,
		MaxAttempts: 0, // Use all providers
	}
}

// NewOrchestrator creates a new NSFW detection orchestrator.
// The primary provider is tried first, with fallbacks tried in order if primary fails.
func NewOrchestrator(
	primary Client,
	fallbacks []Client,
	config OrchestratorConfig,
	logger *zerolog.Logger,
) *Orchestrator {
	return &Orchestrator{
		primary:   primary,
		fallbacks: fallbacks,
		config:    config,
		logger:    logger,
	}
}

// Scan analyzes an image for NSFW content using available providers.
// It tries the primary provider first, then fallbacks in order.
func (o *Orchestrator) Scan(ctx context.Context, imageURL string) (*ScanResult, error) {
	startTime := time.Now()

	// Check if enabled.
	if !o.config.Enabled {
		o.logDebug("NSFW orchestrator disabled")
		return o.disabledResult(startTime), nil
	}

	// Try primary provider.
	if result := o.tryPrimaryProvider(ctx, imageURL); result != nil {
		return result, nil
	}

	// Try fallback providers.
	if result := o.tryFallbackProviders(ctx, imageURL); result != nil {
		return result, nil
	}

	// All providers failed.
	return o.handleAllProvidersFailed(startTime)
}

// disabledResult returns a safe result when orchestrator is disabled.
func (o *Orchestrator) disabledResult(startTime time.Time) *ScanResult {
	return &ScanResult{
		Category:     moderation.CategorySafe,
		Score:        0,
		Details:      moderation.NSFWDetails{},
		Provider:     moderation.NSFWProvider("disabled"),
		ScanDuration: time.Since(startTime),
	}
}

// tryPrimaryProvider attempts to scan using the primary provider.
func (o *Orchestrator) tryPrimaryProvider(ctx context.Context, imageURL string) *ScanResult {
	if o.primary == nil {
		return nil
	}

	if !o.primary.IsAvailable(ctx) {
		o.logDebug(fmt.Sprintf("Primary provider (%s) unavailable", o.primary.Provider()))
		return nil
	}

	result, err := o.primary.Scan(ctx, imageURL)
	if err != nil {
		o.logWarn(fmt.Sprintf("Primary provider (%s) failed", o.primary.Provider()), err)
		return nil
	}

	o.logDebug(fmt.Sprintf("Primary provider (%s) succeeded", o.primary.Provider()))
	return result
}

// tryFallbackProviders attempts to scan using fallback providers.
func (o *Orchestrator) tryFallbackProviders(ctx context.Context, imageURL string) *ScanResult {
	maxAttempts := o.calculateMaxFallbackAttempts()

	for i := 0; i < maxAttempts; i++ {
		fallback := o.fallbacks[i]
		if !fallback.IsAvailable(ctx) {
			o.logDebug(fmt.Sprintf("Fallback provider (%s) unavailable", fallback.Provider()))
			continue
		}

		result, err := fallback.Scan(ctx, imageURL)
		if err != nil {
			o.logWarn(fmt.Sprintf("Fallback provider (%s) failed", fallback.Provider()), err)
			continue
		}

		o.logDebug(fmt.Sprintf("Fallback provider (%s) succeeded", fallback.Provider()))
		return result
	}

	return nil
}

// calculateMaxFallbackAttempts determines how many fallback providers to try.
func (o *Orchestrator) calculateMaxFallbackAttempts() int {
	maxAttempts := len(o.fallbacks)
	if o.config.MaxAttempts > 0 && o.config.MaxAttempts-1 < maxAttempts {
		maxAttempts = o.config.MaxAttempts - 1
	}
	return maxAttempts
}

// handleAllProvidersFailed handles the case when all providers have failed.
func (o *Orchestrator) handleAllProvidersFailed(startTime time.Time) (*ScanResult, error) {
	o.logWarn("All NSFW providers failed", nil)

	if o.config.FailOpen {
		return &ScanResult{
			Category:     moderation.CategoryUnknown,
			Score:        0,
			Details:      moderation.NSFWDetails{},
			Provider:     moderation.NSFWProvider("unknown"),
			ScanDuration: time.Since(startTime),
		}, nil
	}

	return nil, fmt.Errorf("all NSFW detection providers failed")
}

// ScanBytes analyzes image data for NSFW content.
func (o *Orchestrator) ScanBytes(ctx context.Context, data []byte, contentType string) (*ScanResult, error) {
	startTime := time.Now()

	// Check if enabled
	if !o.config.Enabled {
		return &ScanResult{
			Category:     moderation.CategorySafe,
			Score:        0,
			Details:      moderation.NSFWDetails{},
			Provider:     moderation.NSFWProvider("disabled"),
			ScanDuration: time.Since(startTime),
		}, nil
	}

	// Try primary provider
	if o.primary != nil && o.primary.IsAvailable(ctx) {
		result, err := o.primary.ScanBytes(ctx, data, contentType)
		if err == nil {
			return result, nil
		}
		o.logWarn(fmt.Sprintf("Primary provider (%s) failed for bytes", o.primary.Provider()), err)
	}

	// Try fallback providers
	for _, fallback := range o.fallbacks {
		if !fallback.IsAvailable(ctx) {
			continue
		}

		result, err := fallback.ScanBytes(ctx, data, contentType)
		if err == nil {
			return result, nil
		}
		o.logWarn(fmt.Sprintf("Fallback provider (%s) failed for bytes", fallback.Provider()), err)
	}

	// All providers failed
	if o.config.FailOpen {
		return &ScanResult{
			Category:     moderation.CategoryUnknown,
			Score:        0,
			Details:      moderation.NSFWDetails{},
			Provider:     moderation.NSFWProvider("unknown"),
			ScanDuration: time.Since(startTime),
		}, nil
	}

	return nil, fmt.Errorf("all NSFW detection providers failed")
}

// IsAvailable returns true if at least one provider is available.
func (o *Orchestrator) IsAvailable(ctx context.Context) bool {
	if !o.config.Enabled {
		return false
	}

	if o.primary != nil && o.primary.IsAvailable(ctx) {
		return true
	}

	for _, fallback := range o.fallbacks {
		if fallback.IsAvailable(ctx) {
			return true
		}
	}

	return false
}

// logDebug logs a debug message if logger is configured.
func (o *Orchestrator) logDebug(msg string) {
	if o.logger != nil {
		o.logger.Debug().Msg(msg)
	}
}

// logWarn logs a warning message if logger is configured.
func (o *Orchestrator) logWarn(msg string, err error) {
	if o.logger != nil {
		if err != nil {
			o.logger.Warn().Err(err).Msg(msg)
		} else {
			o.logger.Warn().Msg(msg)
		}
	}
}
