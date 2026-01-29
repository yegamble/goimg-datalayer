package security

import (
	"context"
	"crypto/sha1" // #nosec G505 // SHA-1 required by HIBP API specification
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

const (
	// hibpAPIURL is the Have I Been Pwned Passwords API endpoint.
	// Uses k-anonymity model: only hash prefix is sent, full hash never transmitted.
	hibpAPIURL = "https://api.pwnedpasswords.com/range/"

	// hibpHashPrefixLen is the number of characters from the SHA-1 hash to send to HIBP.
	// HIBP uses k-anonymity with 5-character prefixes for privacy.
	hibpHashPrefixLen = 5

	// hibpUserAgent is the User-Agent header sent to HIBP API.
	// HIBP requests identification via User-Agent for fair use policy.
	hibpUserAgent = "goimg-datalayer/1.0"

	// Default configuration values
	defaultTimeout  = 5 * time.Second
	defaultCacheTTL = 24 * time.Hour
)

// HIBPConfig holds configuration for HIBP password checking.
type HIBPConfig struct {
	// Enabled determines whether HIBP checks are performed.
	// When false, IsCompromised always returns (false, nil).
	Enabled bool

	// Timeout is the maximum time to wait for HIBP API response.
	// Default: 5 seconds
	Timeout time.Duration

	// CacheTTL is the time-to-live for cached negative results (password not pwned).
	// Positive results (pwned passwords) are cached indefinitely.
	// Default: 24 hours
	CacheTTL time.Duration

	// FailOpen determines behavior when HIBP API is unavailable.
	// - true: Allow registration/password change (default, prioritizes availability)
	// - false: Block registration/password change (prioritizes security)
	FailOpen bool
}

// DefaultHIBPConfig returns the default HIBP configuration.
func DefaultHIBPConfig() HIBPConfig {
	return HIBPConfig{
		Enabled:  true,
		Timeout:  defaultTimeout,
		CacheTTL: defaultCacheTTL,
		FailOpen: true,
	}
}

// HIBPClient implements PasswordSecurityChecker using the Have I Been Pwned API.
// It uses k-anonymity to check passwords without transmitting the full password or hash.
//
// How k-anonymity works:
//  1. Hash the password with SHA-1
//  2. Take the first 5 characters of the hash (prefix)
//  3. Send only the prefix to HIBP API
//  4. HIBP returns all hash suffixes that match the prefix
//  5. Check if our full hash suffix appears in the response
//
// This ensures the full password hash is never transmitted over the network.
//
// Security properties:
//   - Privacy: Full hash never sent to HIBP
//   - Fail-open: Configurable behavior when API is unavailable
//   - Caching: Optional caching reduces API calls
//   - No logging: Passwords never logged
//   - Timeout: Configurable timeout prevents hanging
//   - Metrics: Records check results and durations (Sprint 10)
type HIBPClient struct {
	httpClient *http.Client
	apiURL     string
	config     HIBPConfig
	cache      PasswordCache       // Optional: nil if no caching
	metrics    HIBPMetricsRecorder // Optional: nil if no metrics
	logger     *zerolog.Logger
}

// NewHIBPClient creates a new HIBP password checker with default configuration.
// For backward compatibility, uses default config and no caching.
func NewHIBPClient() *HIBPClient {
	config := DefaultHIBPConfig()
	return &HIBPClient{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		apiURL:  hibpAPIURL,
		config:  config,
		cache:   nil,
		metrics: nil,
		logger:  nil,
	}
}

// NewHIBPClientWithConfig creates a new HIBP password checker with custom configuration.
// The cache parameter is optional (can be nil).
// The metrics parameter is optional (can be nil).
// The logger parameter is optional (can be nil).
func NewHIBPClientWithConfig(
	config HIBPConfig,
	cache PasswordCache,
	metrics HIBPMetricsRecorder,
	logger *zerolog.Logger,
) *HIBPClient {
	// Apply defaults for zero values
	if config.Timeout == 0 {
		config.Timeout = defaultTimeout
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = defaultCacheTTL
	}

	// Use no-op metrics if nil provided
	if metrics == nil {
		metrics = &NoOpHIBPMetricsRecorder{}
	}

	return &HIBPClient{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		apiURL:  hibpAPIURL,
		config:  config,
		cache:   cache,
		metrics: metrics,
		logger:  logger,
	}
}

// Ensure HIBPClient implements the interface.
var _ identity.PasswordSecurityChecker = (*HIBPClient)(nil)

// IsCompromised checks if a password has been found in known data breaches using HIBP API.
//
// Algorithm:
//  1. Check if HIBP is enabled (return false if disabled)
//  2. Hash password with SHA-1 (HIBP requirement)
//  3. Check cache for previous result (if cache configured)
//  4. On cache miss, send hash prefix to HIBP API
//  5. Search response for matching suffix
//  6. Store result in cache
//  7. Return true if found, false otherwise
//
// Error handling:
//   - HIBP disabled: return false, nil
//   - Cache errors: log and continue to API
//   - Network/API errors: respect FailOpen config
//   - FailOpen=true: return false, nil (allow registration)
//   - FailOpen=false: return false, error (block registration)
//
// Caching strategy:
//   - Negative results (not pwned): cached for CacheTTL (default 24h)
//   - Positive results (pwned): cached indefinitely (once pwned, always pwned)
func (c *HIBPClient) IsCompromised(ctx context.Context, password string) (bool, error) {
	startTime := time.Now()

	// 0. Check if HIBP is enabled
	if !c.config.Enabled {
		if c.logger != nil {
			c.logger.Debug().Msg("HIBP check disabled by configuration")
		}
		c.recordMetric("skipped", false, time.Since(startTime))
		return false, nil
	}

	// 1. Hash the password with SHA-1
	// Note: SHA-1 is used because HIBP API requires it. This is NOT a security vulnerability
	// because the hash is used for lookup, not authentication. The k-anonymity model
	// protects user privacy even if SHA-1 has collision vulnerabilities.
	// #nosec G401 // Required by HIBP API
	hasher := sha1.New()
	hasher.Write([]byte(password))
	hashBytes := hasher.Sum(nil)
	hashStr := strings.ToUpper(hex.EncodeToString(hashBytes))

	// 2. Split into prefix (5 chars) and suffix
	if len(hashStr) < hibpHashPrefixLen {
		return false, fmt.Errorf("invalid hash length: %d", len(hashStr))
	}

	prefix := hashStr[:hibpHashPrefixLen]
	suffix := hashStr[hibpHashPrefixLen:]

	// 3. Check cache first (if configured)
	if c.cache != nil {
		if pwned, found := c.cache.Get(ctx, prefix, suffix); found {
			if c.logger != nil {
				c.logger.Debug().
					Bool("pwned", pwned).
					Msg("HIBP cache hit")
			}
			result := "clean"
			if pwned {
				result = "compromised"
			}
			c.recordMetric(result, true, time.Since(startTime))
			return pwned, nil
		}
	}

	// 4. Cache miss - call HIBP API
	pwned, err := c.checkHIBPAPI(ctx, prefix, suffix)
	if err != nil {
		// Log the error
		if c.logger != nil {
			c.logger.Warn().
				Err(err).
				Bool("fail_open", c.config.FailOpen).
				Msg("HIBP API check failed")
		}

		c.recordMetric("error", false, time.Since(startTime))

		// Respect fail-open configuration
		if c.config.FailOpen {
			// Fail open: allow registration despite API failure
			return false, nil
		}
		// Fail closed: block registration on API failure
		return false, fmt.Errorf("password breach check unavailable: %w", err)
	}

	// 5. Store result in cache (if configured)
	if c.cache != nil {
		// Determine TTL based on result
		ttl := c.config.CacheTTL
		if pwned {
			// Pwned passwords cached indefinitely (they'll never become un-pwned)
			ttl = 0
		}

		if err := c.cache.Set(ctx, prefix, suffix, pwned, ttl); err != nil {
			// Cache write failure is non-critical - log and continue
			if c.logger != nil {
				c.logger.Warn().
					Err(err).
					Msg("failed to cache HIBP result")
			}
		}
	}

	if c.logger != nil {
		c.logger.Debug().
			Bool("pwned", pwned).
			Msg("HIBP API check completed")
	}

	// Record metrics for successful check
	result := "clean"
	if pwned {
		result = "compromised"
	}
	c.recordMetric(result, false, time.Since(startTime))

	return pwned, nil
}

// recordMetric records HIBP check metrics if a metrics recorder is configured.
func (c *HIBPClient) recordMetric(result string, cacheHit bool, duration time.Duration) {
	if c.metrics == nil {
		return
	}
	c.metrics.RecordHIBPCheck(result)
	c.metrics.RecordHIBPCheckDuration(duration.Seconds(), cacheHit)
}

// checkHIBPAPI performs the actual HIBP API call and response parsing.
// Separated from IsCompromised for cleaner caching logic.
func (c *HIBPClient) checkHIBPAPI(ctx context.Context, prefix, suffix string) (bool, error) {
	// 1. Build API request
	url := c.apiURL + prefix

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("create HIBP request: %w", err)
	}

	// Set User-Agent header as requested by HIBP fair use policy
	req.Header.Set("User-Agent", hibpUserAgent)

	// 2. Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("HIBP API request failed: %w", err)
	}
	defer resp.Body.Close()

	// 3. Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("HIBP API returned status %d", resp.StatusCode)
	}

	// 4. Read response body
	// HIBP returns a list of hash suffixes with occurrence counts:
	// <suffix>:<count>
	// Example:
	// 0018A45C4D1DEF81644B54AB7F969B88D65:1
	// 00D4F6E8FA6EECAD2A3AA415EEC418D38EC:2
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("read HIBP response: %w", err)
	}

	// 5. Search for our suffix in the response
	// Each line is: <suffix>:<count>
	// We only care if our suffix exists, not the count
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split on colon to get suffix
		parts := strings.Split(line, ":")
		if len(parts) < 1 {
			continue
		}

		responseSuffix := strings.TrimSpace(parts[0])
		if responseSuffix == suffix {
			// Password found in breach database
			return true, nil
		}
	}

	// Password not found in breach database
	return false, nil
}
