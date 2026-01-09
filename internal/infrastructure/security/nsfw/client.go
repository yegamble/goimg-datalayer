// Package nsfw provides AI-powered NSFW content detection using external APIs.
// It supports multiple providers (SightEngine, ModerateContent) with automatic fallback.
package nsfw

import (
	"context"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// Default configuration values.
const (
	DefaultTimeout       = 10 * time.Second
	DefaultRetryAttempts = 2
	DefaultRetryDelay    = 500 * time.Millisecond
)

// ScanResult contains the result of an NSFW content scan.
type ScanResult struct {
	// Category is the primary NSFW category detected.
	Category moderation.NSFWCategory

	// Score is the overall NSFW confidence score (0.0 to 1.0).
	Score float64

	// Details contains provider-specific detection details.
	Details moderation.NSFWDetails

	// Provider identifies which API provider generated this result.
	Provider moderation.NSFWProvider

	// ScanDuration is how long the scan took.
	ScanDuration time.Duration
}

// Client defines the interface for NSFW content detection.
// Implementations should handle API rate limiting, retries, and timeout.
type Client interface {
	// Scan analyzes an image for NSFW content.
	// The imageURL must be publicly accessible or a base64-encoded data URI.
	// Returns the scan result or an error if the scan failed.
	Scan(ctx context.Context, imageURL string) (*ScanResult, error)

	// ScanBytes analyzes image data for NSFW content.
	// This is useful when the image is already in memory.
	// Returns the scan result or an error if the scan failed.
	ScanBytes(ctx context.Context, data []byte, contentType string) (*ScanResult, error)

	// Provider returns the provider identifier for this client.
	Provider() moderation.NSFWProvider

	// IsAvailable checks if the API is currently available.
	// This can be used for health checks and circuit breaker patterns.
	IsAvailable(ctx context.Context) bool
}

// Config holds common configuration for NSFW detection clients.
type Config struct {
	// Enabled determines whether NSFW detection is active.
	// When false, all scans return safe category with score 0.
	Enabled bool

	// Timeout is the maximum time to wait for API response.
	// Default: 10 seconds
	Timeout time.Duration

	// RetryAttempts is the number of retry attempts on transient failures.
	// Default: 2
	RetryAttempts int

	// RetryDelay is the delay between retry attempts.
	// Default: 500ms
	RetryDelay time.Duration

	// FailOpen determines behavior when API is unavailable.
	// - true: Return safe category (prioritizes availability)
	// - false: Return error (prioritizes security)
	FailOpen bool
}

// DefaultConfig returns the default NSFW configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:       true,
		Timeout:       DefaultTimeout,
		RetryAttempts: DefaultRetryAttempts,
		RetryDelay:    DefaultRetryDelay,
		FailOpen:      true,
	}
}

// ValidateConfig ensures configuration values are within acceptable ranges.
func ValidateConfig(cfg Config) Config {
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.RetryAttempts < 0 {
		cfg.RetryAttempts = 0
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = DefaultRetryDelay
	}
	return cfg
}
