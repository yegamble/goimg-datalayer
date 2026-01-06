package security

import (
	"context"
	"crypto/sha1" //nolint:gosec // SHA-1 required by HIBP API specification
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

const (
	// hibpAPIURL is the Have I Been Pwned Passwords API endpoint.
	// Uses k-anonymity model: only hash prefix is sent, full hash never transmitted.
	hibpAPIURL = "https://api.pwnedpasswords.com/range/"

	// hibpHashPrefixLen is the number of characters from the SHA-1 hash to send to HIBP.
	// HIBP uses k-anonymity with 5-character prefixes for privacy.
	hibpHashPrefixLen = 5

	// hibpTimeout is the maximum time to wait for HIBP API response.
	hibpTimeout = 5 * time.Second

	// hibpUserAgent is the User-Agent header sent to HIBP API.
	// HIBP requests identification via User-Agent for fair use policy.
	hibpUserAgent = "goimg-datalayer/1.0"
)

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
//   - Fail-open: API failures don't block user registration
//   - No logging: Passwords never logged
//   - Timeout: 5-second timeout prevents hanging
type HIBPClient struct {
	httpClient *http.Client
	apiURL     string
}

// NewHIBPClient creates a new HIBP password checker with default configuration.
func NewHIBPClient() *HIBPClient {
	return &HIBPClient{
		httpClient: &http.Client{
			Timeout: hibpTimeout,
		},
		apiURL: hibpAPIURL,
	}
}

// Ensure HIBPClient implements the interface.
var _ identity.PasswordSecurityChecker = (*HIBPClient)(nil)

// IsCompromised checks if a password has been found in known data breaches using HIBP API.
//
// Algorithm:
//  1. Hash password with SHA-1 (HIBP requirement)
//  2. Split hash into prefix (5 chars) and suffix
//  3. Send prefix to HIBP API
//  4. Search response for matching suffix
//  5. Return true if found, false otherwise
//
// Error handling:
//   - Network errors: return false (fail open)
//   - HTTP errors: return false (fail open)
//   - Parse errors: return false (fail open)
//   - Timeout: return false (fail open)
//
// This fail-open behavior ensures service availability is prioritized over
// perfect security. A degraded password check is better than blocking all registrations.
func (c *HIBPClient) IsCompromised(ctx context.Context, password string) (bool, error) {
	// 1. Hash the password with SHA-1
	// Note: SHA-1 is used because HIBP API requires it. This is NOT a security vulnerability
	// because the hash is used for lookup, not authentication. The k-anonymity model
	// protects user privacy even if SHA-1 has collision vulnerabilities.
	hasher := sha1.New() //nolint:gosec // Required by HIBP API
	hasher.Write([]byte(password))
	hashBytes := hasher.Sum(nil)
	hashStr := strings.ToUpper(hex.EncodeToString(hashBytes))

	// 2. Split into prefix (5 chars) and suffix
	if len(hashStr) < hibpHashPrefixLen {
		return false, fmt.Errorf("invalid hash length: %d", len(hashStr))
	}

	prefix := hashStr[:hibpHashPrefixLen]
	suffix := hashStr[hibpHashPrefixLen:]

	// 3. Send prefix to HIBP API
	url := c.apiURL + prefix

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("create HIBP request: %w", err)
	}

	// Set User-Agent header as requested by HIBP fair use policy
	req.Header.Set("User-Agent", hibpUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Network error - fail open
		return false, fmt.Errorf("HIBP API request failed: %w", err)
	}
	defer resp.Body.Close()

	// 4. Check HTTP status
	if resp.StatusCode != http.StatusOK {
		// API error - fail open
		return false, fmt.Errorf("HIBP API returned status %d", resp.StatusCode)
	}

	// 5. Read response body
	// HIBP returns a list of hash suffixes with occurrence counts:
	// <suffix>:<count>
	// Example:
	// 0018A45C4D1DEF81644B54AB7F969B88D65:1
	// 00D4F6E8FA6EECAD2A3AA415EEC418D38EC:2
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("read HIBP response: %w", err)
	}

	// 6. Search for our suffix in the response
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
