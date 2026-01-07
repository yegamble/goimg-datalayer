package ipfs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CID length constants for validation.
const (
	cidV0Length    = 46 // CIDv0: Qm... (base58btc, 46 characters)
	cidV1MinLength = 50 // CIDv1: bafy... (base32, minimum 50 characters)
	cidOtherMinLen = 10 // Other CIDv1 bases: b... (minimum length)
)

// Client implements IPFS storage operations using the Kubo HTTP API.
// It does not depend on any external IPFS libraries, using only net/http.
type Client struct {
	config     Config
	httpClient *http.Client
}

// New creates a new IPFS client with the given configuration.
func New(cfg Config) (*Client, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

// AddResult contains the response from an IPFS add operation.
type AddResult struct {
	// Hash is the CID (Content Identifier) of the uploaded content.
	Hash string `json:"Hash"`
	// Name is the filename (if provided).
	Name string `json:"Name"`
	// Size is the size in bytes as a string.
	Size string `json:"Size"`
}

// PutOptions configures storage upload behavior (for compatibility with Storage interface).
type PutOptions struct {
	ContentType  string
	CacheControl string
	Metadata     map[string]string
}

// ObjectInfo contains metadata about a stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
}

// Put stores data in IPFS and returns the CID.
// The key parameter is ignored for IPFS as content is addressed by CID.
func (c *Client) Put(ctx context.Context, _ string, data io.Reader, _ int64, _ PutOptions) error {
	_, err := c.Add(ctx, data)
	return err
}

// PutBytes is a convenience method for storing small in-memory data.
func (c *Client) PutBytes(ctx context.Context, key string, data []byte, opts PutOptions) error {
	return c.Put(ctx, key, bytes.NewReader(data), int64(len(data)), opts)
}

// Add uploads content to IPFS and returns the CID.
// This is the primary method for adding content to IPFS.
func (c *Client) Add(ctx context.Context, data io.Reader) (*AddResult, error) {
	// Build the multipart request
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "data")
	if err != nil {
		return nil, fmt.Errorf("ipfs create form: %w", err)
	}

	if _, err := io.Copy(part, data); err != nil {
		return nil, fmt.Errorf("ipfs copy data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("ipfs close writer: %w", err)
	}

	// Build request URL
	apiURL := fmt.Sprintf("%s/api/v0/add?pin=%t", c.config.APIEndpoint, c.config.PinByDefault)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, &body)
	if err != nil {
		return nil, fmt.Errorf("ipfs create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.wrapHTTPError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", ErrUploadFailed, resp.StatusCode, string(respBody))
	}

	var result AddResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ipfs decode response: %w", err)
	}

	return &result, nil
}

// AddBytes is a convenience method for uploading byte slices.
func (c *Client) AddBytes(ctx context.Context, data []byte) (*AddResult, error) {
	return c.Add(ctx, bytes.NewReader(data))
}

// Get retrieves content from IPFS by CID.
func (c *Client) Get(ctx context.Context, cid string) (io.ReadCloser, error) {
	if err := ValidateCID(cid); err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/api/v0/cat?arg=%s", c.config.APIEndpoint, url.QueryEscape(cid))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ipfs create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.wrapHTTPError(err)
	}

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusInternalServerError {
		_ = resp.Body.Close()
		return nil, ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("ipfs cat failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	return resp.Body, nil
}

// GetBytes retrieves content fully into memory.
func (c *Client) GetBytes(ctx context.Context, cid string) ([]byte, error) {
	reader, err := c.Get(ctx, cid)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("ipfs read content: %w", err)
	}
	return data, nil
}

// Pin pins content to the local IPFS node.
// Pinned content is protected from garbage collection.
func (c *Client) Pin(ctx context.Context, cid string) error {
	return c.pinOperation(ctx, cid, "/api/v0/pin/add", ErrPinFailed)
}

// Unpin removes a pin from content.
// Unpinned content may be garbage collected.
func (c *Client) Unpin(ctx context.Context, cid string) error {
	return c.pinOperation(ctx, cid, "/api/v0/pin/rm", ErrUnpinFailed)
}

// pinOperation executes a pin add/remove operation.
func (c *Client) pinOperation(ctx context.Context, cid, apiPath string, failErr error) error {
	if err := ValidateCID(cid); err != nil {
		return err
	}

	apiURL := fmt.Sprintf("%s%s?arg=%s", c.config.APIEndpoint, apiPath, url.QueryEscape(cid))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return fmt.Errorf("ipfs create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.wrapHTTPError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s", failErr, resp.StatusCode, string(respBody))
	}

	return nil
}

// IsPinned checks if content is pinned to the local node.
func (c *Client) IsPinned(ctx context.Context, cid string) (bool, error) {
	if err := ValidateCID(cid); err != nil {
		return false, err
	}

	apiURL := fmt.Sprintf("%s/api/v0/pin/ls?arg=%s", c.config.APIEndpoint, url.QueryEscape(cid))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return false, fmt.Errorf("ipfs create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, c.wrapHTTPError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	// A 200 response with pin info means it's pinned
	// A 500 error typically means it's not pinned
	return resp.StatusCode == http.StatusOK, nil
}

// Delete removes content from IPFS (unpin + trigger GC).
// Note: Actual deletion requires garbage collection which runs periodically.
func (c *Client) Delete(ctx context.Context, cid string) error {
	return c.Unpin(ctx, cid)
}

// Exists checks if content is available locally.
func (c *Client) Exists(ctx context.Context, cid string) (bool, error) {
	if err := ValidateCID(cid); err != nil {
		return false, err
	}

	apiURL := fmt.Sprintf("%s/api/v0/block/stat?arg=%s", c.config.APIEndpoint, url.QueryEscape(cid))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return false, fmt.Errorf("ipfs create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Network error = assume not available locally (intentional design choice)
		return false, nil
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK, nil
}

// URL returns the public gateway URL for a CID.
func (c *Client) URL(cid string) string {
	if c.config.GatewayEndpoint == "" {
		return ""
	}
	return fmt.Sprintf("%s/ipfs/%s", c.config.GatewayEndpoint, cid)
}

// IPFSURI returns the canonical IPFS URI for a CID.
// This is the preferred format for NFT metadata and long-term storage.
func (c *Client) IPFSURI(cid string) string {
	return fmt.Sprintf("ipfs://%s", cid)
}

// PresignedURL is not supported for IPFS (content is public by CID).
func (c *Client) PresignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", ErrNotSupported
}

// Stat returns metadata about stored content.
// Note: IPFS doesn't track modification time, so LastModified is zero.
func (c *Client) Stat(ctx context.Context, cid string) (*ObjectInfo, error) {
	if err := ValidateCID(cid); err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/api/v0/block/stat?arg=%s", c.config.APIEndpoint, url.QueryEscape(cid))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ipfs create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.wrapHTTPError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrNotFound
	}

	var result struct {
		Key  string `json:"Key"`
		Size int    `json:"Size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ipfs decode response: %w", err)
	}

	return &ObjectInfo{
		Key:         cid,
		Size:        int64(result.Size),
		ContentType: "application/octet-stream", // IPFS doesn't track content type
		ETag:        fmt.Sprintf(`"%s"`, cid),   // CID is the content hash
	}, nil
}

// Provider returns the provider type name.
func (c *Client) Provider() string {
	return "ipfs"
}

// NodeID returns the IPFS node's peer ID.
func (c *Client) NodeID(ctx context.Context) (string, error) {
	apiURL := fmt.Sprintf("%s/api/v0/id", c.config.APIEndpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("ipfs create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrNodeUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", ErrNodeUnavailable, resp.StatusCode)
	}

	var result struct {
		ID string `json:"ID"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("ipfs decode response: %w", err)
	}

	return result.ID, nil
}

// wrapHTTPError wraps HTTP client errors with appropriate IPFS errors.
func (c *Client) wrapHTTPError(err error) error {
	if isTimeoutError(err) {
		return fmt.Errorf("%w: %w", ErrTimeout, err)
	}
	return fmt.Errorf("%w: %w", ErrNodeUnavailable, err)
}

// ValidateCID checks if a string is a valid IPFS CID.
// Supports both CIDv0 (Qm...) and CIDv1 (bafy...).
func ValidateCID(cid string) error {
	if cid == "" {
		return fmt.Errorf("%w: empty CID", ErrInvalidCID)
	}

	// CIDv0: starts with "Qm", base58btc encoded, 46 characters
	if strings.HasPrefix(cid, "Qm") {
		if len(cid) != cidV0Length {
			return fmt.Errorf("%w: invalid CIDv0 length", ErrInvalidCID)
		}
		return nil
	}

	// CIDv1: starts with "bafy" (base32) or "b" (other bases)
	if strings.HasPrefix(cid, "bafy") || strings.HasPrefix(cid, "bafk") {
		if len(cid) < cidV1MinLength {
			return fmt.Errorf("%w: invalid CIDv1 length", ErrInvalidCID)
		}
		return nil
	}

	// Allow other CIDv1 formats starting with 'b'
	if strings.HasPrefix(cid, "b") && len(cid) > cidOtherMinLen {
		return nil
	}

	return fmt.Errorf("%w: unrecognized format", ErrInvalidCID)
}

// isTimeoutError checks if an error is a timeout error.
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	// Check for context deadline exceeded
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	// Check for url.Error timeout
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Timeout()
	}
	return false
}
