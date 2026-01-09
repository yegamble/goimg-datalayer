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
	// SightEngine API endpoints.
	sightEngineAPIURL    = "https://api.sightengine.com/1.0/check.json"
	sightEngineUploadURL = "https://api.sightengine.com/1.0/check-workflow.json"

	// Default models for NSFW detection.
	// nudity-2.1: Advanced nudity detection.
	// wad: Weapon, Alcohol, Drug detection.
	// offensive: Offensive content detection.
	sightEngineDefaultModels = "nudity-2.1,wad,offensive"

	// Detection thresholds.
	sightEngineExplicitThreshold   = 0.7
	sightEngineNudityThreshold     = 0.5
	sightEngineSuggestiveThreshold = 0.3
	sightEngineWeaponThreshold     = 0.5
)

// SightEngineConfig holds configuration specific to SightEngine API.
type SightEngineConfig struct {
	Config

	// APIUser is the SightEngine API user ID.
	APIUser string

	// APISecret is the SightEngine API secret key.
	APISecret string

	// Models is a comma-separated list of models to use.
	// Default: "nudity-2.1,wad,offensive"
	Models string
}

// DefaultSightEngineConfig returns the default SightEngine configuration.
func DefaultSightEngineConfig() SightEngineConfig {
	return SightEngineConfig{
		Config: DefaultConfig(),
		Models: sightEngineDefaultModels,
	}
}

// SightEngineClient implements Client using the SightEngine API.
type SightEngineClient struct {
	httpClient *http.Client
	config     SightEngineConfig
	logger     *zerolog.Logger
}

// NewSightEngineClient creates a new SightEngine NSFW detection client.
func NewSightEngineClient(config SightEngineConfig, logger *zerolog.Logger) *SightEngineClient {
	cfg := ValidateConfig(config.Config)
	config.Config = cfg

	if config.Models == "" {
		config.Models = sightEngineDefaultModels
	}

	return &SightEngineClient{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		config: config,
		logger: logger,
	}
}

// Ensure SightEngineClient implements Client.
var _ Client = (*SightEngineClient)(nil)

// Scan analyzes an image URL for NSFW content.
func (c *SightEngineClient) Scan(ctx context.Context, imageURL string) (*ScanResult, error) {
	startTime := time.Now()

	// Check if enabled
	if !c.config.Enabled {
		c.logDebug("SightEngine scan disabled")
		return &ScanResult{
			Category:     moderation.CategorySafe,
			Score:        0,
			Details:      moderation.NSFWDetails{},
			Provider:     moderation.ProviderSightEngine,
			ScanDuration: time.Since(startTime),
		}, nil
	}

	// Validate credentials
	if c.config.APIUser == "" || c.config.APISecret == "" {
		return nil, fmt.Errorf("SightEngine API credentials not configured")
	}

	// Build request
	params := url.Values{}
	params.Set("url", imageURL)
	params.Set("models", c.config.Models)
	params.Set("api_user", c.config.APIUser)
	params.Set("api_secret", c.config.APISecret)

	// Execute with retry
	var lastErr error
	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			c.logDebug(fmt.Sprintf("Retry attempt %d after %v", attempt, c.config.RetryDelay))
			time.Sleep(c.config.RetryDelay)
		}

		result, err := c.executeRequest(ctx, sightEngineAPIURL, params)
		if err == nil {
			result.ScanDuration = time.Since(startTime)
			return result, nil
		}
		lastErr = err
	}

	// All retries failed
	c.logWarn("SightEngine scan failed after retries", lastErr)

	if c.config.FailOpen {
		return &ScanResult{
			Category:     moderation.CategoryUnknown,
			Score:        0,
			Details:      moderation.NSFWDetails{},
			Provider:     moderation.ProviderSightEngine,
			ScanDuration: time.Since(startTime),
		}, nil
	}

	return nil, fmt.Errorf("SightEngine scan failed: %w", lastErr)
}

// ScanBytes analyzes image data for NSFW content.
func (c *SightEngineClient) ScanBytes(ctx context.Context, data []byte, contentType string) (*ScanResult, error) {
	// Convert to base64 data URI
	encoded := base64.StdEncoding.EncodeToString(data)
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, encoded)
	return c.Scan(ctx, dataURI)
}

// Provider returns the provider identifier.
func (c *SightEngineClient) Provider() moderation.NSFWProvider {
	return moderation.ProviderSightEngine
}

// IsAvailable checks if the SightEngine API is available.
func (c *SightEngineClient) IsAvailable(_ context.Context) bool {
	if !c.config.Enabled {
		return false
	}
	if c.config.APIUser == "" || c.config.APISecret == "" {
		return false
	}
	// Could add a health check endpoint call here if needed.
	return true
}

// executeRequest performs the actual API call.
func (c *SightEngineClient) executeRequest(ctx context.Context, apiURL string, params url.Values) (*ScanResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL+"?"+params.Encode(), nil)
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

// sightEngineResponse represents the SightEngine API response structure.
type sightEngineResponse struct {
	Status string `json:"status"`
	Error  struct {
		Type    int    `json:"type"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`

	// Nudity detection results
	Nudity struct {
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
	} `json:"nudity"`

	// Weapon/Alcohol/Drug detection
	Weapon  float64 `json:"weapon"`
	Alcohol float64 `json:"alcohol"`
	Drugs   float64 `json:"drugs"`

	// Offensive content
	Offensive struct {
		Prob float64 `json:"prob"`
	} `json:"offensive"`

	// Raw media info
	Media struct {
		ID  string `json:"id"`
		URI string `json:"uri"`
	} `json:"media"`

	RequestID string `json:"request_id"`
}

// parseResponse converts the SightEngine API response to our domain types.
func (c *SightEngineClient) parseResponse(body []byte) (*ScanResult, error) {
	var resp sightEngineResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Check for API errors
	if resp.Status == "failure" {
		return nil, fmt.Errorf("SightEngine error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	// Calculate scores
	nudityScore := maxFloats(resp.Nudity.Sexual, resp.Nudity.SexualDisplay, resp.Nudity.Erotica, resp.Nudity.Partial)
	weaponScore := resp.Weapon
	violenceScore := resp.Weapon // Using weapon as proxy for violence
	offensiveScore := resp.Offensive.Prob
	drugScore := maxFloats(resp.Alcohol, resp.Drugs)

	// Build details
	details := moderation.NewNSFWDetails(nudityScore, weaponScore, violenceScore, offensiveScore, drugScore)

	// Collect subcategories
	var subCategories []string
	if resp.Nudity.Suggestive > sightEngineSuggestiveThreshold {
		subCategories = append(subCategories, "suggestive")
	}
	if resp.Nudity.SuggestiveClasses.Bikini > sightEngineNudityThreshold {
		subCategories = append(subCategories, "bikini")
	}
	if resp.Nudity.SuggestiveClasses.Lingerie > sightEngineNudityThreshold {
		subCategories = append(subCategories, "lingerie")
	}
	if resp.Nudity.Partial > sightEngineNudityThreshold {
		subCategories = append(subCategories, "partial_nudity")
	}

	details = details.WithSubCategories(subCategories)

	// Store raw response for debugging
	var rawResp map[string]interface{}
	_ = json.Unmarshal(body, &rawResp)
	details = details.WithRawResponse(rawResp)

	// Determine category and overall score
	category, score := c.determineCategory(resp)

	return &ScanResult{
		Category: category,
		Score:    score,
		Details:  details,
		Provider: moderation.ProviderSightEngine,
	}, nil
}

// determineCategory determines the NSFW category based on response scores.
func (c *SightEngineClient) determineCategory(resp sightEngineResponse) (moderation.NSFWCategory, float64) {
	// Thresholds for classification
	const (
		explicitThreshold   = 0.7
		nudityThreshold     = 0.5
		suggestiveThreshold = 0.3
		weaponThreshold     = 0.5
	)

	// Check for explicit content first (highest severity)
	if resp.Nudity.Sexual > explicitThreshold || resp.Nudity.SexualDisplay > explicitThreshold {
		return moderation.CategoryExplicit, maxFloats(resp.Nudity.Sexual, resp.Nudity.SexualDisplay)
	}

	// Check for nudity
	if resp.Nudity.Erotica > nudityThreshold || resp.Nudity.Partial > nudityThreshold {
		return moderation.CategoryNudity, maxFloats(resp.Nudity.Erotica, resp.Nudity.Partial)
	}

	// Check for violence (weapons)
	if resp.Weapon > weaponThreshold {
		return moderation.CategoryViolence, resp.Weapon
	}

	// Check for suggestive content
	if resp.Nudity.Suggestive > suggestiveThreshold {
		return moderation.CategorySuggestive, resp.Nudity.Suggestive
	}

	// Safe content
	return moderation.CategorySafe, resp.Nudity.Safe
}

// logDebug logs a debug message if logger is configured.
func (c *SightEngineClient) logDebug(msg string) {
	if c.logger != nil {
		c.logger.Debug().Msg(msg)
	}
}

// logWarn logs a warning message if logger is configured.
func (c *SightEngineClient) logWarn(msg string, err error) {
	if c.logger != nil {
		c.logger.Warn().Err(err).Msg(msg)
	}
}

// maxFloats returns the maximum of the provided float64 values.
func maxFloats(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}
