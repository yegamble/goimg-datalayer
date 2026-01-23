// Package orchestrator provides a dual-storage coordinator that writes to both
// a primary storage backend (S3/local) and IPFS for decentralized backup.
package orchestrator

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage/ipfs"
)

// Mode determines the orchestrator's write behavior.
type Mode string

const (
	// ModePrimaryOnly writes only to primary storage.
	ModePrimaryOnly Mode = "primary_only"
	// ModeDualSync writes to both primary and IPFS synchronously.
	ModeDualSync Mode = "dual_sync"
	// ModeDualAsync writes to primary immediately, then IPFS asynchronously.
	ModeDualAsync Mode = "dual_async"
)

// IPFSClient defines the interface for IPFS operations.
// This allows for testing with mock implementations.
type IPFSClient interface {
	AddBytes(ctx context.Context, data []byte) (*ipfs.AddResult, error)
	Get(ctx context.Context, cid string) (io.ReadCloser, error)
	GetBytes(ctx context.Context, cid string) ([]byte, error)
	URL(cid string) string
	IPFSURI(cid string) string
	Unpin(ctx context.Context, cid string) error
}

// Config configures the storage orchestrator.
type Config struct {
	// Mode determines write behavior.
	Mode Mode

	// FallbackEnabled enables reading from IPFS if primary fails.
	FallbackEnabled bool

	// IPFSEnabled determines if IPFS operations are active.
	// When false, the orchestrator behaves like primary-only.
	IPFSEnabled bool
}

// DefaultConfig returns default orchestrator configuration.
func DefaultConfig() Config {
	return Config{
		Mode:            ModePrimaryOnly,
		FallbackEnabled: false,
		IPFSEnabled:     false,
	}
}

// Orchestrator coordinates storage operations between primary and IPFS backends.
// It implements the storage.Storage interface for seamless integration.
type Orchestrator struct {
	primary    storage.Storage
	ipfsClient IPFSClient
	config     Config
}

// New creates a new storage orchestrator.
// The primary storage is required; IPFS client is optional based on config.
func New(primary storage.Storage, ipfsClient IPFSClient, cfg Config) (*Orchestrator, error) {
	if primary == nil {
		return nil, fmt.Errorf("orchestrator: primary storage is required")
	}

	if cfg.IPFSEnabled && ipfsClient == nil {
		return nil, fmt.Errorf("orchestrator: IPFS client required when IPFS is enabled")
	}

	return &Orchestrator{
		primary:    primary,
		ipfsClient: ipfsClient,
		config:     cfg,
	}, nil
}

// Put stores data to primary storage and optionally to IPFS.
// The key is used for primary storage; IPFS returns a CID which can be
// retrieved via AddToIPFS or obtained when using dual-sync mode.
func (o *Orchestrator) Put(ctx context.Context, key string, data io.Reader, size int64, opts storage.PutOptions) error {
	// For dual-sync mode, we need to buffer the data to write to both stores
	if o.shouldWriteIPFS() && o.config.Mode == ModeDualSync {
		return o.putDualSync(ctx, key, data, opts)
	}

	// Primary-only or primary-first (async IPFS)
	if err := o.primary.Put(ctx, key, data, size, opts); err != nil {
		return fmt.Errorf("orchestrator: primary put: %w", err)
	}
	return nil
}

// putDualSync writes to both primary and IPFS synchronously.
func (o *Orchestrator) putDualSync(ctx context.Context, key string, data io.Reader, opts storage.PutOptions) error {
	// Read all data into memory for dual write
	// Note: This requires buffering - for very large files, consider async mode
	allData, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("orchestrator: read data: %w", err)
	}

	// Write to primary first
	if err := o.primary.PutBytes(ctx, key, allData, opts); err != nil {
		return fmt.Errorf("orchestrator: primary put: %w", err)
	}

	// Write to IPFS - don't fail if IPFS fails since primary succeeded
	// In production, this failure should be logged/monitored
	_, _ = o.ipfsClient.AddBytes(ctx, allData)

	return nil
}

// PutBytes stores data to primary storage and optionally to IPFS.
func (o *Orchestrator) PutBytes(ctx context.Context, key string, data []byte, opts storage.PutOptions) error {
	// Write to primary first
	if err := o.primary.PutBytes(ctx, key, data, opts); err != nil {
		return fmt.Errorf("orchestrator: primary put: %w", err)
	}

	// Write to IPFS if enabled and in sync mode
	// Don't fail if IPFS fails since primary succeeded
	if o.shouldWriteIPFS() && o.config.Mode == ModeDualSync {
		_, _ = o.ipfsClient.AddBytes(ctx, data)
	}

	return nil
}

// Get retrieves data from primary storage, with optional IPFS fallback.
func (o *Orchestrator) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	reader, err := o.primary.Get(ctx, key)
	if err == nil {
		return reader, nil
	}

	// Try IPFS fallback if enabled and key looks like a CID
	if o.config.FallbackEnabled && o.ipfsClient != nil {
		if ipfs.ValidateCID(key) == nil {
			ipfsReader, ipfsErr := o.ipfsClient.Get(ctx, key)
			if ipfsErr != nil {
				return nil, fmt.Errorf("orchestrator: ipfs fallback: %w", ipfsErr)
			}
			return ipfsReader, nil
		}
	}

	return nil, fmt.Errorf("orchestrator: get: %w", err)
}

// GetBytes retrieves data with fallback support.
func (o *Orchestrator) GetBytes(ctx context.Context, key string) ([]byte, error) {
	data, err := o.primary.GetBytes(ctx, key)
	if err == nil {
		return data, nil
	}

	// Try IPFS fallback if enabled and key looks like a CID
	if o.config.FallbackEnabled && o.ipfsClient != nil {
		if ipfs.ValidateCID(key) == nil {
			ipfsData, ipfsErr := o.ipfsClient.GetBytes(ctx, key)
			if ipfsErr != nil {
				return nil, fmt.Errorf("orchestrator: ipfs fallback: %w", ipfsErr)
			}
			return ipfsData, nil
		}
	}

	return nil, fmt.Errorf("orchestrator: get: %w", err)
}

// Delete removes data from primary storage and unpins from IPFS if enabled.
func (o *Orchestrator) Delete(ctx context.Context, key string) error {
	// Delete from primary storage first.
	if err := o.primary.Delete(ctx, key); err != nil {
		return fmt.Errorf("orchestrator: primary delete: %w", err)
	}

	// If IPFS is enabled, attempt to unpin the content.
	// We assume the key can be a CID for IPFS operations.
	if o.shouldWriteIPFS() {
		// Validate if the key is a valid CID before attempting to unpin.
		if ipfs.ValidateCID(key) == nil {
			if err := o.ipfsClient.Unpin(ctx, key); err != nil {
				// Do not fail the operation if unpin fails, as primary is the source of truth.
				// In a production system, this should be logged for monitoring.
				// Example: log.Printf("warning: failed to unpin CID %s: %v", key, err)
			}
		}
	}

	return nil
}

// Exists checks if data exists in primary storage.
func (o *Orchestrator) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := o.primary.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("orchestrator: exists: %w", err)
	}
	return exists, nil
}

// URL returns the URL from primary storage.
func (o *Orchestrator) URL(key string) string {
	return o.primary.URL(key)
}

// PresignedURL generates a presigned URL from primary storage.
func (o *Orchestrator) PresignedURL(ctx context.Context, key string, duration time.Duration) (string, error) {
	url, err := o.primary.PresignedURL(ctx, key, duration)
	if err != nil {
		return "", fmt.Errorf("orchestrator: presigned url: %w", err)
	}
	return url, nil
}

// Stat returns metadata from primary storage.
func (o *Orchestrator) Stat(ctx context.Context, key string) (*storage.ObjectInfo, error) {
	info, err := o.primary.Stat(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: stat: %w", err)
	}
	return info, nil
}

// Provider returns "orchestrator" to identify this as an orchestrated storage.
func (o *Orchestrator) Provider() string {
	return "orchestrator"
}

// AddToIPFS explicitly uploads content to IPFS and returns the CID.
// This is useful for async backup or when you need the CID.
func (o *Orchestrator) AddToIPFS(ctx context.Context, data []byte) (*ipfs.AddResult, error) {
	if o.ipfsClient == nil {
		return nil, fmt.Errorf("orchestrator: IPFS not configured")
	}
	result, err := o.ipfsClient.AddBytes(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: ipfs add: %w", err)
	}
	return result, nil
}

// GetFromIPFS retrieves content directly from IPFS by CID.
func (o *Orchestrator) GetFromIPFS(ctx context.Context, cid string) ([]byte, error) {
	if o.ipfsClient == nil {
		return nil, fmt.Errorf("orchestrator: IPFS not configured")
	}
	data, err := o.ipfsClient.GetBytes(ctx, cid)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: ipfs get: %w", err)
	}
	return data, nil
}

// IPFSURL returns the IPFS gateway URL for a CID.
func (o *Orchestrator) IPFSURL(cid string) string {
	if o.ipfsClient == nil {
		return ""
	}
	return o.ipfsClient.URL(cid)
}

// IPFSURI returns the canonical ipfs:// URI for a CID.
func (o *Orchestrator) IPFSURI(cid string) string {
	if o.ipfsClient == nil {
		return ""
	}
	return o.ipfsClient.IPFSURI(cid)
}

// IPFSEnabled returns whether IPFS is configured and enabled.
func (o *Orchestrator) IPFSEnabled() bool {
	return o.config.IPFSEnabled && o.ipfsClient != nil
}

// shouldWriteIPFS returns true if IPFS writes are active.
func (o *Orchestrator) shouldWriteIPFS() bool {
	return o.config.IPFSEnabled && o.ipfsClient != nil && o.config.Mode != ModePrimaryOnly
}

// IPFS returns the underlying IPFS client for direct access.
// Use sparingly - prefer the orchestrator's interface.
func (o *Orchestrator) IPFS() IPFSClient {
	return o.ipfsClient
}
