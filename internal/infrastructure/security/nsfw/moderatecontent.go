package nsfw

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

const (
	// ModerateContent API endpoint.
	moderateContentAPIURL = "https://api.moderatecontent.com/moderate/"

	// ModerateContent rating indices.
	mcRatingEveryone = 1
	mcRatingTeen     = 2
	mcRatingAdult    = 3

	// ModerateContent thresholds.
	mcExplicitThreshold    = 0.8
	mcSubCategoryThreshold = 0.5
	mcOffensiveScoreFactor = 0.5
)

// ModerateContentConfig holds configuration for ModerateContent API.
type ModerateContentConfig struct {
	Config

	// APIKey is the ModerateContent API key.
	APIKey string
}

// DefaultModerateContentConfig returns the default ModerateContent configuration.
func DefaultModerateContentConfig() ModerateContentConfig {
	return ModerateContentConfig{
		Config: DefaultConfig(),
	}
}

// ModerateContentClient implements Client using the ModerateContent API.
// ModerateContent is simpler than SightEngine but provides a good fallback.
type ModerateContentClient struct {
	httpClient *http.Client
	config     ModerateContentConfig
	logger     *zerolog.Logger
}

// NewModerateContentClient creates a new ModerateContent NSFW detection client.
func NewModerateContentClient(config ModerateContentConfig, logger *zerolog.Logger) *ModerateContentClient {
	cfg := ValidateConfig(config.Config)
	config.Config = cfg

	return &ModerateContentClient{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		config: config,
		logger: logger,
	}
}

// Ensure ModerateContentClient implements Client.
var _ Client = (*ModerateContentClient)(nil)

// Scan analyzes an image URL for NSFW content.
func (c *ModerateContentClient) Scan(ctx context.Context, imageURL string) (*ScanResult, error) {
	startTime := time.Now()

	// Check if enabled
	if !c.config.Enabled {
		c.logDebug("ModerateContent scan disabled")
		return &ScanResult{
			Category:     moderation.CategorySafe,
			Score:        0,
			Details:      moderation.NSFWDetails{},
			Provider:     moderation.ProviderModerateContent,
			ScanDuration: time.Since(startTime),
		}, nil
	}

	// Validate credentials
	if c.config.APIKey == "" {
		return nil, fmt.Errorf("ModerateContent API key not configured")
	}

	// Build request URL
	apiURL := fmt.Sprintf(
		"%s?key=%s&url=%s",
		moderateContentAPIURL,
		url.QueryEscape(c.config.APIKey),
		url.QueryEscape(imageURL),
	)

	// Execute with retry
	var lastErr error
	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			c.logDebug(fmt.Sprintf("Retry attempt %d after %v", attempt, c.config.RetryDelay))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.config.RetryDelay):
			}
		}

		result, err := c.executeRequest(ctx, apiURL)
		if err == nil {
			result.ScanDuration = time.Since(startTime)
			return result, nil
		}
		lastErr = err
	}

	// All retries failed
	c.logWarn("ModerateContent scan failed after retries", lastErr)

	if c.config.FailOpen {
		return &ScanResult{
			Category:     moderation.CategoryUnknown,
			Score:        0,
			Details:      moderation.NSFWDetails{},
			Provider:     moderation.ProviderModerateContent,
			ScanDuration: time.Since(startTime),
		}, nil
	}

	return nil, fmt.Errorf("ModerateContent scan failed: %w", lastErr)
}

// ScanBytes analyzes image data for NSFW content.
func (c *ModerateContentClient) ScanBytes(ctx context.Context, data []byte, contentType string) (*ScanResult, error) {
	// Convert to base64 data URI
	encoded := base64.StdEncoding.EncodeToString(data)
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, encoded)
	return c.Scan(ctx, dataURI)
}

// Provider returns the provider identifier.
func (c *ModerateContentClient) Provider() moderation.NSFWProvider {
	return moderation.ProviderModerateContent
}

// IsAvailable checks if the ModerateContent API is available.
func (c *ModerateContentClient) IsAvailable(_ context.Context) bool {
	if !c.config.Enabled {
		return false
	}
	if c.config.APIKey == "" {
		return false
	}
	return true
}

// executeRequest performs the actual API call.
func (c *ModerateContentClient) executeRequest(ctx context.Context, apiURL string) (*ScanResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // Error on close is not actionable

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return c.parseResponse(body)
}

// moderateContentResponse represents the ModerateContent API response.
type moderateContentResponse struct {
	// Error code (0 = success)
	ErrorCode int    `json:"error_code"`
	Error     string `json:"error"`

	// URL of the analyzed image
	URL string `json:"url"`

	// Rating category: 1=everyone, 2=teen, 3=adult
	RatingIndex int `json:"rating_index"`

	// Rating label
	RatingLabel string `json:"rating_label"`

	// Detailed predictions
	Predictions struct {
		Everyone float64 `json:"everyone"`
		Teen     float64 `json:"teen"`
		Adult    float64 `json:"adult"`
	} `json:"predictions"`
}

// parseResponse converts the ModerateContent API response to our domain types.
func (c *ModerateContentClient) parseResponse(body []byte) (*ScanResult, error) {
	var resp moderateContentResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Check for API errors
	if resp.ErrorCode != 0 {
		return nil, fmt.Errorf("ModerateContent error: %s (code %d)", resp.Error, resp.ErrorCode)
	}

	// Map ModerateContent scores to our details.
	// ModerateContent uses: everyone (safe), teen (suggestive), adult (NSFW).
	nudityScore := resp.Predictions.Adult
	offensiveScore := resp.Predictions.Adult * mcOffensiveScoreFactor // Estimate
	suggestiveScore := resp.Predictions.Teen

	details := moderation.NewNSFWDetails(nudityScore, 0, 0, offensiveScore, 0)

	// Add subcategories based on predictions.
	var subCategories []string
	if suggestiveScore > mcSubCategoryThreshold {
		subCategories = append(subCategories, "teen_rated")
	}
	if nudityScore > mcSubCategoryThreshold {
		subCategories = append(subCategories, "adult_rated")
	}
	details = details.WithSubCategories(subCategories)

	// Store raw response
	var rawResp map[string]interface{}
	_ = json.Unmarshal(body, &rawResp)
	details = details.WithRawResponse(rawResp)

	// Determine category
	category, score := c.determineCategory(resp)

	return &ScanResult{
		Category: category,
		Score:    score,
		Details:  details,
		Provider: moderation.ProviderModerateContent,
	}, nil
}

// determineCategory maps ModerateContent rating to our categories.
func (c *ModerateContentClient) determineCategory(resp moderateContentResponse) (moderation.NSFWCategory, float64) {
	// ModerateContent uses rating_index:
	// 1 = everyone (safe)
	// 2 = teen (suggestive)
	// 3 = adult (NSFW)
	switch resp.RatingIndex {
	case mcRatingAdult:
		// Adult content - determine if explicit or nudity.
		if resp.Predictions.Adult > mcExplicitThreshold {
			return moderation.CategoryExplicit, resp.Predictions.Adult
		}
		return moderation.CategoryNudity, resp.Predictions.Adult
	case mcRatingTeen:
		// Teen content - suggestive.
		return moderation.CategorySuggestive, resp.Predictions.Teen
	default:
		// Everyone - safe (includes mcRatingEveryone).
		return moderation.CategorySafe, resp.Predictions.Everyone
	}
}

// logDebug logs a debug message if logger is configured.
func (c *ModerateContentClient) logDebug(msg string) {
	if c.logger != nil {
		c.logger.Debug().Msg(msg)
	}
}

// logWarn logs a warning message if logger is configured.
func (c *ModerateContentClient) logWarn(msg string, err error) {
	if c.logger != nil {
		c.logger.Warn().Err(err).Msg(msg)
	}
}
