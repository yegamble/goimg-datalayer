# IPFS Storage Integration

> Load this guide when implementing or maintaining IPFS storage functionality.

## Overview

IPFS (InterPlanetary File System) provides decentralized, content-addressed storage for images. In goimg-datalayer, IPFS operates **alongside** traditional storage providers (Local, S3), not as a replacement.

## Architecture

### Dual-Storage Model

```
┌─────────────────────────────────────────────────────────────────┐
│                     Application Layer                            │
│                  (Upload Image Command)                          │
├─────────────────────────────────────────────────────────────────┤
│                    Storage Orchestrator                          │
│              Coordinates multiple storage backends               │
├──────────────────────┬──────────────────────────────────────────┤
│   Primary Storage    │           IPFS Storage                    │
│   (S3/Local/Spaces)  │        (Optional, Parallel)               │
│                      │                                           │
│   - Fast retrieval   │   - Content-addressed (CID)               │
│   - CDN-friendly     │   - Decentralized                         │
│   - Cost-effective   │   - Permanent, immutable                  │
└──────────────────────┴──────────────────────────────────────────┘
```

### IPFS Integration Points

| Component | Location | Purpose |
|-----------|----------|---------|
| IPFS Client | `internal/infrastructure/storage/ipfs/` | HTTP API client for IPFS node |
| Storage Interface | `internal/infrastructure/storage/storage.go` | Common interface for all providers |
| Orchestrator | `internal/infrastructure/storage/orchestrator.go` | Coordinates primary + IPFS storage |
| Config | `internal/config/storage.go` | IPFS configuration options |

## IPFS Storage Interface

```go
// IPFSStorage extends the base Storage interface with IPFS-specific operations
type IPFSStorage interface {
    Storage

    // Pin ensures content remains available on IPFS network
    Pin(ctx context.Context, cid string) error

    // Unpin removes pinning (allows garbage collection)
    Unpin(ctx context.Context, cid string) error

    // IsPinned checks if content is pinned
    IsPinned(ctx context.Context, cid string) (bool, error)

    // GetCID returns the content identifier for stored data
    GetCID(ctx context.Context, key string) (string, error)

    // PinRemote pins to a remote pinning service (Pinata, Infura, etc.)
    PinRemote(ctx context.Context, cid string, service string) error
}

// Storage is the base interface all providers implement
type Storage interface {
    Put(ctx context.Context, key string, data []byte) error
    Get(ctx context.Context, key string) ([]byte, error)
    Delete(ctx context.Context, key string) error
    URL(key string) string
    Exists(ctx context.Context, key string) (bool, error)
}
```

## Implementation Guide

### 1. IPFS Client Implementation

```go
// internal/infrastructure/storage/ipfs/client.go

package ipfs

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

// Config holds IPFS connection settings
type Config struct {
    // APIEndpoint is the IPFS HTTP API address (e.g., "http://localhost:5001")
    APIEndpoint string

    // GatewayEndpoint for public URL generation (e.g., "https://ipfs.io")
    GatewayEndpoint string

    // Timeout for API operations
    Timeout time.Duration

    // PinByDefault automatically pins uploaded content
    PinByDefault bool

    // RemotePinningServices for redundant pinning
    RemotePinningServices []RemotePinConfig
}

type RemotePinConfig struct {
    Name     string // e.g., "pinata", "infura"
    Endpoint string
    Key      string
}

// Client implements IPFSStorage
type Client struct {
    config     Config
    httpClient *http.Client
}

// New creates a new IPFS client
func New(cfg Config) (*Client, error) {
    if cfg.APIEndpoint == "" {
        return nil, fmt.Errorf("ipfs: API endpoint required")
    }
    if cfg.Timeout == 0 {
        cfg.Timeout = 30 * time.Second
    }
    if cfg.GatewayEndpoint == "" {
        cfg.GatewayEndpoint = "https://ipfs.io"
    }

    return &Client{
        config: cfg,
        httpClient: &http.Client{
            Timeout: cfg.Timeout,
        },
    }, nil
}

// Put uploads data to IPFS and returns the CID as the key
func (c *Client) Put(ctx context.Context, key string, data []byte) error {
    cid, err := c.add(ctx, data)
    if err != nil {
        return fmt.Errorf("ipfs put %s: %w", key, err)
    }

    if c.config.PinByDefault {
        if err := c.Pin(ctx, cid); err != nil {
            return fmt.Errorf("ipfs pin %s: %w", cid, err)
        }
    }

    // Store key->CID mapping (implementation depends on your metadata store)
    // This allows retrieval by application key rather than CID
    return nil
}

// add uploads content to IPFS and returns the CID
func (c *Client) add(ctx context.Context, data []byte) (string, error) {
    url := fmt.Sprintf("%s/api/v0/add?pin=%t", c.config.APIEndpoint, c.config.PinByDefault)

    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
    if err != nil {
        return "", fmt.Errorf("create request: %w", err)
    }
    req.Header.Set("Content-Type", "application/octet-stream")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("ipfs add: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("ipfs add failed: %s", body)
    }

    var result struct {
        Hash string `json:"Hash"`
        Size string `json:"Size"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", fmt.Errorf("decode response: %w", err)
    }

    return result.Hash, nil
}

// Get retrieves data from IPFS by CID
func (c *Client) Get(ctx context.Context, cid string) ([]byte, error) {
    url := fmt.Sprintf("%s/api/v0/cat?arg=%s", c.config.APIEndpoint, cid)

    req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("ipfs cat: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("ipfs cat failed: status %d", resp.StatusCode)
    }

    return io.ReadAll(resp.Body)
}

// Pin pins content to the local IPFS node
func (c *Client) Pin(ctx context.Context, cid string) error {
    url := fmt.Sprintf("%s/api/v0/pin/add?arg=%s", c.config.APIEndpoint, cid)

    req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("ipfs pin: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("ipfs pin failed: %s", body)
    }

    return nil
}

// Unpin removes a pin from content
func (c *Client) Unpin(ctx context.Context, cid string) error {
    url := fmt.Sprintf("%s/api/v0/pin/rm?arg=%s", c.config.APIEndpoint, cid)

    req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("ipfs unpin: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("ipfs unpin failed: %s", body)
    }

    return nil
}

// URL returns the public gateway URL for a CID
func (c *Client) URL(cid string) string {
    return fmt.Sprintf("%s/ipfs/%s", c.config.GatewayEndpoint, cid)
}

// IsPinned checks if content is pinned locally
func (c *Client) IsPinned(ctx context.Context, cid string) (bool, error) {
    url := fmt.Sprintf("%s/api/v0/pin/ls?arg=%s", c.config.APIEndpoint, cid)

    req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
    if err != nil {
        return false, fmt.Errorf("create request: %w", err)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return false, fmt.Errorf("ipfs pin ls: %w", err)
    }
    defer resp.Body.Close()

    return resp.StatusCode == http.StatusOK, nil
}

// Delete removes content from IPFS (unpin + garbage collect)
func (c *Client) Delete(ctx context.Context, cid string) error {
    if err := c.Unpin(ctx, cid); err != nil {
        return err
    }
    // Note: Actual deletion requires garbage collection
    // which is typically done periodically, not per-request
    return nil
}

// Exists checks if content is available locally
func (c *Client) Exists(ctx context.Context, cid string) (bool, error) {
    url := fmt.Sprintf("%s/api/v0/block/stat?arg=%s", c.config.APIEndpoint, cid)

    req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
    if err != nil {
        return false, fmt.Errorf("create request: %w", err)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return false, nil // Network error = not available
    }
    defer resp.Body.Close()

    return resp.StatusCode == http.StatusOK, nil
}
```

### 2. Storage Orchestrator

The orchestrator manages dual storage (primary + IPFS):

```go
// internal/infrastructure/storage/orchestrator.go

package storage

import (
    "context"
    "fmt"
    "sync"
)

// Orchestrator coordinates multiple storage backends
type Orchestrator struct {
    primary Storage       // Required: S3, Local, etc.
    ipfs    IPFSStorage   // Optional: IPFS for decentralized backup
    cidMap  CIDMapper     // Maps application keys to IPFS CIDs
}

// CIDMapper stores key->CID mappings (implement with Redis/Postgres)
type CIDMapper interface {
    Set(ctx context.Context, key, cid string) error
    Get(ctx context.Context, key string) (string, error)
    Delete(ctx context.Context, key string) error
}

// OrchestratorConfig configures the storage orchestrator
type OrchestratorConfig struct {
    EnableIPFS        bool
    IPFSAsync         bool // Upload to IPFS in background
    RequireIPFSPin    bool // Fail if IPFS pinning fails
}

// NewOrchestrator creates a storage orchestrator
func NewOrchestrator(primary Storage, ipfs IPFSStorage, cidMap CIDMapper, cfg OrchestratorConfig) *Orchestrator {
    return &Orchestrator{
        primary: primary,
        ipfs:    ipfs,
        cidMap:  cidMap,
    }
}

// Put stores data in primary storage and optionally IPFS
func (o *Orchestrator) Put(ctx context.Context, key string, data []byte) error {
    // Always store in primary
    if err := o.primary.Put(ctx, key, data); err != nil {
        return fmt.Errorf("primary storage: %w", err)
    }

    // Store in IPFS if enabled
    if o.ipfs != nil {
        if err := o.putIPFS(ctx, key, data); err != nil {
            // Log but don't fail if IPFS is optional
            // return err if IPFS is required
        }
    }

    return nil
}

func (o *Orchestrator) putIPFS(ctx context.Context, key string, data []byte) error {
    if err := o.ipfs.Put(ctx, key, data); err != nil {
        return fmt.Errorf("ipfs storage: %w", err)
    }

    cid, err := o.ipfs.GetCID(ctx, key)
    if err != nil {
        return fmt.Errorf("get cid: %w", err)
    }

    if err := o.cidMap.Set(ctx, key, cid); err != nil {
        return fmt.Errorf("store cid mapping: %w", err)
    }

    return nil
}

// Get retrieves data, falling back to IPFS if primary fails
func (o *Orchestrator) Get(ctx context.Context, key string) ([]byte, error) {
    // Try primary first
    data, err := o.primary.Get(ctx, key)
    if err == nil {
        return data, nil
    }

    // Fallback to IPFS
    if o.ipfs != nil {
        cid, cidErr := o.cidMap.Get(ctx, key)
        if cidErr == nil {
            return o.ipfs.Get(ctx, cid)
        }
    }

    return nil, fmt.Errorf("get %s: %w", key, err)
}
```

### 3. Domain Layer (No IPFS Knowledge)

The domain layer remains pure - it only knows about storage concepts:

```go
// internal/domain/gallery/image.go

package gallery

// Image aggregate - NO knowledge of IPFS/S3/etc.
type Image struct {
    id         ImageID
    ownerID    UserID
    storageKey string          // Opaque key, could map to any storage
    ipfsCID    *string         // Optional: stored if IPFS is enabled
    metadata   ImageMetadata
    // ...
}

// ImageRepository interface - implementation handles storage details
type ImageRepository interface {
    Save(ctx context.Context, image *Image) error
    FindByID(ctx context.Context, id ImageID) (*Image, error)
    // ...
}
```

## Docker Configuration

### IPFS Node Container

```yaml
# docker/docker-compose.yml

services:
  ipfs:
    image: ipfs/kubo:latest
    container_name: goimg-ipfs
    environment:
      - IPFS_PROFILE=server
    ports:
      - "4001:4001"     # P2P swarm
      - "5001:5001"     # API
      - "8080:8080"     # Gateway
    volumes:
      - ipfs_data:/data/ipfs
      - ipfs_staging:/export
    command: ["daemon", "--migrate=true", "--enable-gc"]
    healthcheck:
      test: ["CMD", "ipfs", "id"]
      interval: 30s
      timeout: 10s
      retries: 3
    restart: unless-stopped

volumes:
  ipfs_data:
  ipfs_staging:
```

### IPFS Configuration Options

For production, consider these IPFS daemon configurations:

```bash
# Limit connections for resource management
ipfs config --json Swarm.ConnMgr.LowWater 50
ipfs config --json Swarm.ConnMgr.HighWater 200

# Enable AutoNAT for better connectivity
ipfs config --json AutoNAT.ServiceMode '"enabled"'

# Configure gateway
ipfs config --json Gateway.HTTPHeaders.Access-Control-Allow-Origin '["*"]'

# Enable garbage collection
ipfs config --json Datastore.GCPeriod '"1h"'
```

## Configuration

### Environment Variables

```bash
# IPFS Configuration
IPFS_ENABLED=true                           # Enable IPFS storage
IPFS_API_ENDPOINT=http://localhost:5001     # IPFS node API
IPFS_GATEWAY_ENDPOINT=https://ipfs.io       # Public gateway for URLs
IPFS_PIN_BY_DEFAULT=true                    # Auto-pin uploaded content
IPFS_ASYNC_UPLOAD=true                      # Non-blocking IPFS uploads
IPFS_REQUIRE_PIN=false                      # Fail upload if pinning fails

# Remote Pinning Services (optional, for redundancy)
IPFS_PINATA_ENABLED=true
IPFS_PINATA_JWT=your-pinata-jwt
IPFS_INFURA_ENABLED=false
IPFS_INFURA_PROJECT_ID=your-project-id
IPFS_INFURA_PROJECT_SECRET=your-secret
```

### Config Struct

```go
// internal/config/ipfs.go

package config

type IPFSConfig struct {
    Enabled         bool          `env:"IPFS_ENABLED" envDefault:"false"`
    APIEndpoint     string        `env:"IPFS_API_ENDPOINT" envDefault:"http://localhost:5001"`
    GatewayEndpoint string        `env:"IPFS_GATEWAY_ENDPOINT" envDefault:"https://ipfs.io"`
    PinByDefault    bool          `env:"IPFS_PIN_BY_DEFAULT" envDefault:"true"`
    AsyncUpload     bool          `env:"IPFS_ASYNC_UPLOAD" envDefault:"true"`
    RequirePin      bool          `env:"IPFS_REQUIRE_PIN" envDefault:"false"`
    Timeout         time.Duration `env:"IPFS_TIMEOUT" envDefault:"30s"`

    // Remote pinning services
    Pinata PinataConfig
    Infura InfuraConfig
}

type PinataConfig struct {
    Enabled bool   `env:"IPFS_PINATA_ENABLED" envDefault:"false"`
    JWT     string `env:"IPFS_PINATA_JWT"`
}

type InfuraConfig struct {
    Enabled       bool   `env:"IPFS_INFURA_ENABLED" envDefault:"false"`
    ProjectID     string `env:"IPFS_INFURA_PROJECT_ID"`
    ProjectSecret string `env:"IPFS_INFURA_PROJECT_SECRET"`
}
```

## Testing Strategy

### Unit Tests

```go
// internal/infrastructure/storage/ipfs/client_test.go

package ipfs_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "goimg-datalayer/internal/infrastructure/storage/ipfs"
)

func TestClient_Put(t *testing.T) {
    // Mock IPFS API
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/api/v0/add" {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte(`{"Hash":"QmTest123","Size":"1234"}`))
            return
        }
        if r.URL.Path == "/api/v0/pin/add" {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte(`{"Pins":["QmTest123"]}`))
            return
        }
        w.WriteHeader(http.StatusNotFound)
    }))
    defer server.Close()

    client, err := ipfs.New(ipfs.Config{
        APIEndpoint:  server.URL,
        PinByDefault: true,
    })
    require.NoError(t, err)

    ctx := context.Background()
    err = client.Put(ctx, "test-key", []byte("test image data"))
    assert.NoError(t, err)
}

func TestClient_Get(t *testing.T) {
    expectedData := []byte("test image content")

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/api/v0/cat" {
            w.WriteHeader(http.StatusOK)
            w.Write(expectedData)
            return
        }
        w.WriteHeader(http.StatusNotFound)
    }))
    defer server.Close()

    client, err := ipfs.New(ipfs.Config{APIEndpoint: server.URL})
    require.NoError(t, err)

    data, err := client.Get(context.Background(), "QmTest123")
    require.NoError(t, err)
    assert.Equal(t, expectedData, data)
}

func TestClient_Pin(t *testing.T) {
    pinned := false

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/api/v0/pin/add" {
            pinned = true
            w.WriteHeader(http.StatusOK)
            w.Write([]byte(`{"Pins":["QmTest123"]}`))
            return
        }
        w.WriteHeader(http.StatusNotFound)
    }))
    defer server.Close()

    client, err := ipfs.New(ipfs.Config{APIEndpoint: server.URL})
    require.NoError(t, err)

    err = client.Pin(context.Background(), "QmTest123")
    require.NoError(t, err)
    assert.True(t, pinned)
}

func TestClient_IsPinned(t *testing.T) {
    tests := []struct {
        name       string
        statusCode int
        expected   bool
    }{
        {"pinned", http.StatusOK, true},
        {"not pinned", http.StatusInternalServerError, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(tt.statusCode)
            }))
            defer server.Close()

            client, _ := ipfs.New(ipfs.Config{APIEndpoint: server.URL})
            isPinned, _ := client.IsPinned(context.Background(), "QmTest123")
            assert.Equal(t, tt.expected, isPinned)
        })
    }
}
```

### Integration Tests

```go
// tests/integration/storage/ipfs_test.go

//go:build integration

package storage_test

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"

    "goimg-datalayer/internal/infrastructure/storage/ipfs"
)

func TestIPFSIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()

    // Start IPFS container
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "ipfs/kubo:latest",
            ExposedPorts: []string{"5001/tcp", "8080/tcp"},
            WaitingFor:   wait.ForHTTP("/api/v0/id").WithPort("5001/tcp"),
            Cmd:          []string{"daemon", "--offline"},
        },
        Started: true,
    })
    require.NoError(t, err)
    defer container.Terminate(ctx)

    apiEndpoint, err := container.Endpoint(ctx, "5001/tcp")
    require.NoError(t, err)

    client, err := ipfs.New(ipfs.Config{
        APIEndpoint:  "http://" + apiEndpoint,
        PinByDefault: true,
        Timeout:      30 * time.Second,
    })
    require.NoError(t, err)

    t.Run("upload and retrieve image", func(t *testing.T) {
        testImage := []byte("fake image data for testing")

        // Upload
        err := client.Put(ctx, "test-image", testImage)
        require.NoError(t, err)

        // Retrieve by CID (in real impl, you'd get CID from Put)
        // For this test, we use the known CID pattern
        cid, err := client.GetCID(ctx, "test-image")
        require.NoError(t, err)

        data, err := client.Get(ctx, cid)
        require.NoError(t, err)
        assert.Equal(t, testImage, data)
    })

    t.Run("pin and verify", func(t *testing.T) {
        testData := []byte("content to pin")

        err := client.Put(ctx, "pinned-content", testData)
        require.NoError(t, err)

        cid, _ := client.GetCID(ctx, "pinned-content")

        isPinned, err := client.IsPinned(ctx, cid)
        require.NoError(t, err)
        assert.True(t, isPinned)
    })

    t.Run("unpin content", func(t *testing.T) {
        testData := []byte("content to unpin")

        err := client.Put(ctx, "unpin-test", testData)
        require.NoError(t, err)

        cid, _ := client.GetCID(ctx, "unpin-test")

        err = client.Unpin(ctx, cid)
        require.NoError(t, err)

        isPinned, _ := client.IsPinned(ctx, cid)
        assert.False(t, isPinned)
    })
}

func TestIPFSImageRoundTrip(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()

    // Use testcontainers to start IPFS
    container := setupIPFSContainer(t, ctx)
    defer container.Terminate(ctx)

    apiEndpoint, _ := container.Endpoint(ctx, "5001/tcp")

    client, _ := ipfs.New(ipfs.Config{
        APIEndpoint:  "http://" + apiEndpoint,
        PinByDefault: true,
    })

    // Test with actual image file
    testCases := []struct {
        name     string
        mimeType string
        data     []byte
    }{
        {"jpeg", "image/jpeg", generateTestJPEG()},
        {"png", "image/png", generateTestPNG()},
        {"webp", "image/webp", generateTestWebP()},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            key := "test-" + tc.name

            err := client.Put(ctx, key, tc.data)
            require.NoError(t, err)

            cid, err := client.GetCID(ctx, key)
            require.NoError(t, err)
            require.NotEmpty(t, cid)

            retrieved, err := client.Get(ctx, cid)
            require.NoError(t, err)
            assert.Equal(t, tc.data, retrieved)

            // Verify pinned
            isPinned, err := client.IsPinned(ctx, cid)
            require.NoError(t, err)
            assert.True(t, isPinned)
        })
    }
}
```

### E2E Tests

```go
// tests/e2e/upload_ipfs_test.go

//go:build e2e

package e2e_test

import (
    "bytes"
    "encoding/json"
    "io"
    "mime/multipart"
    "net/http"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUploadImageWithIPFS(t *testing.T) {
    baseURL := getTestServerURL(t)
    token := getTestAuthToken(t)

    // Create multipart form
    var buf bytes.Buffer
    writer := multipart.NewWriter(&buf)

    part, _ := writer.CreateFormFile("image", "test.jpg")
    part.Write(loadTestImage(t, "fixtures/test-image.jpg"))
    writer.Close()

    // Upload image
    req, _ := http.NewRequest("POST", baseURL+"/api/v1/images", &buf)
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", writer.FormDataContentType())

    resp, err := http.DefaultClient.Do(req)
    require.NoError(t, err)
    defer resp.Body.Close()

    require.Equal(t, http.StatusCreated, resp.StatusCode)

    var result struct {
        ID       string  `json:"id"`
        URL      string  `json:"url"`
        IPFSCID  *string `json:"ipfs_cid,omitempty"`
        IPFSURL  *string `json:"ipfs_url,omitempty"`
    }

    body, _ := io.ReadAll(resp.Body)
    err = json.Unmarshal(body, &result)
    require.NoError(t, err)

    // Verify IPFS fields are present (when IPFS is enabled)
    if isIPFSEnabled() {
        require.NotNil(t, result.IPFSCID)
        require.NotNil(t, result.IPFSURL)
        assert.Contains(t, *result.IPFSURL, "/ipfs/")

        // Verify image is retrievable from IPFS
        ipfsResp, err := http.Get(*result.IPFSURL)
        require.NoError(t, err)
        defer ipfsResp.Body.Close()

        assert.Equal(t, http.StatusOK, ipfsResp.StatusCode)
    }
}
```

## Error Handling

```go
// internal/infrastructure/storage/ipfs/errors.go

package ipfs

import "errors"

var (
    // ErrNotFound indicates content not found on IPFS
    ErrNotFound = errors.New("ipfs: content not found")

    // ErrPinFailed indicates pinning operation failed
    ErrPinFailed = errors.New("ipfs: pin failed")

    // ErrNodeUnavailable indicates IPFS node is not reachable
    ErrNodeUnavailable = errors.New("ipfs: node unavailable")

    // ErrInvalidCID indicates an invalid content identifier
    ErrInvalidCID = errors.New("ipfs: invalid CID")

    // ErrTimeout indicates operation timed out
    ErrTimeout = errors.New("ipfs: operation timed out")
)

// IsRetryable returns true if the error can be retried
func IsRetryable(err error) bool {
    return errors.Is(err, ErrNodeUnavailable) || errors.Is(err, ErrTimeout)
}
```

## Security Considerations

1. **Private IPFS Network**: For sensitive content, consider running a private IPFS network
2. **Gateway Authentication**: Protect your IPFS gateway if exposing publicly
3. **Content Validation**: Always validate content before uploading to IPFS
4. **CID Integrity**: Store CIDs in the database to verify content integrity
5. **Rate Limiting**: Implement rate limiting for IPFS operations

## Monitoring

```go
// Prometheus metrics for IPFS operations
var (
    ipfsUploadDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "ipfs_upload_duration_seconds",
            Help: "Duration of IPFS upload operations",
        },
        []string{"status"},
    )

    ipfsPinTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ipfs_pin_total",
            Help: "Total number of IPFS pin operations",
        },
        []string{"status"},
    )

    ipfsStorageBytes = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "ipfs_storage_bytes",
            Help: "Total bytes stored in IPFS",
        },
    )
)
```

## Remote Pinning Services

For production reliability, pin to multiple services:

```go
// internal/infrastructure/storage/ipfs/remote_pin.go

package ipfs

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

// PinataClient handles Pinata remote pinning
type PinataClient struct {
    jwt        string
    httpClient *http.Client
}

func (p *PinataClient) Pin(ctx context.Context, cid string, name string) error {
    payload := map[string]interface{}{
        "hashToPin": cid,
        "pinataMetadata": map[string]string{
            "name": name,
        },
    }

    body, _ := json.Marshal(payload)
    req, _ := http.NewRequestWithContext(ctx, "POST",
        "https://api.pinata.cloud/pinning/pinByHash",
        bytes.NewReader(body))

    req.Header.Set("Authorization", "Bearer "+p.jwt)
    req.Header.Set("Content-Type", "application/json")

    resp, err := p.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("pinata pin: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("pinata pin failed: status %d", resp.StatusCode)
    }

    return nil
}
```

## Quick Reference

| Operation | Endpoint | Method |
|-----------|----------|--------|
| Add content | `/api/v0/add` | POST |
| Get content | `/api/v0/cat?arg={cid}` | POST |
| Pin content | `/api/v0/pin/add?arg={cid}` | POST |
| Unpin content | `/api/v0/pin/rm?arg={cid}` | POST |
| List pins | `/api/v0/pin/ls` | POST |
| Node info | `/api/v0/id` | POST |
| Repo stats | `/api/v0/repo/stat` | POST |
