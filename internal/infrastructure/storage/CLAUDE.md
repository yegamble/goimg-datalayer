# Storage Layer Guide

> Scoped guide for implementing and maintaining storage providers.

## Overview

This package provides multi-provider object storage for images. Storage providers can work **independently** or **together** (dual-storage with IPFS).

## Supported Providers

| Provider | Package | Use Case |
|----------|---------|----------|
| Local | `local/` | Development, small deployments |
| S3 | `s3/` | AWS S3, DO Spaces, Backblaze B2 |
| IPFS | `ipfs/` | Decentralized, content-addressed |

## Architecture

```
storage/
├── storage.go          # Common Storage interface
├── orchestrator.go     # Multi-provider coordination
├── errors.go           # Shared error types
├── local/
│   ├── local.go        # Filesystem storage
│   └── local_test.go
├── s3/
│   ├── s3.go           # S3-compatible storage
│   ├── config.go       # S3 configuration
│   └── s3_test.go
└── ipfs/
    ├── client.go       # IPFS HTTP API client
    ├── config.go       # IPFS configuration
    ├── remote_pin.go   # Remote pinning services
    ├── errors.go       # IPFS-specific errors
    └── client_test.go
```

## Storage Interface

All providers must implement this interface:

```go
type Storage interface {
    // Put stores data with the given key
    Put(ctx context.Context, key string, data []byte) error

    // Get retrieves data by key
    Get(ctx context.Context, key string) ([]byte, error)

    // Delete removes data by key
    Delete(ctx context.Context, key string) error

    // URL returns a publicly accessible URL for the key
    URL(key string) string

    // Exists checks if key exists in storage
    Exists(ctx context.Context, key string) (bool, error)
}
```

## IPFS Extended Interface

IPFS provider implements additional methods:

```go
type IPFSStorage interface {
    Storage

    // Pin ensures content remains on IPFS network
    Pin(ctx context.Context, cid string) error

    // Unpin removes pinning (allows garbage collection)
    Unpin(ctx context.Context, cid string) error

    // IsPinned checks pinning status
    IsPinned(ctx context.Context, cid string) (bool, error)

    // GetCID returns content identifier for key
    GetCID(ctx context.Context, key string) (string, error)

    // PinRemote pins to external service (Pinata, Infura)
    PinRemote(ctx context.Context, cid, service string) error
}
```

## Implementation Rules

### Error Handling

Always wrap errors with context:

```go
// Good
return fmt.Errorf("ipfs put %s: %w", key, err)

// Bad
return err
```

### Context Usage

All operations must respect context cancellation:

```go
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
    req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
    // ...
}
```

### No Domain Imports

Storage packages must NOT import domain packages:

```go
// Forbidden
import "goimg-datalayer/internal/domain/gallery"

// Allowed
import "goimg-datalayer/internal/infrastructure/storage"
```

### Return Only stdlib/Infrastructure Types

Never expose provider-specific types to callers:

```go
// Good: returns standard types
func (c *Client) Get(ctx context.Context, key string) ([]byte, error)

// Bad: exposes IPFS-specific types
func (c *Client) Get(ctx context.Context, key string) (*ipfs.Object, error)
```

## Dual-Storage Pattern

When IPFS is enabled alongside primary storage:

```go
// Upload to both storages
func (o *Orchestrator) Put(ctx context.Context, key string, data []byte) error {
    // 1. Always store in primary (fast, reliable)
    if err := o.primary.Put(ctx, key, data); err != nil {
        return fmt.Errorf("primary storage: %w", err)
    }

    // 2. Store in IPFS if enabled (can be async)
    if o.ipfs != nil {
        go func() {
            if err := o.ipfs.Put(context.Background(), key, data); err != nil {
                // Log error, don't fail upload
                o.logger.Error("ipfs backup failed", "key", key, "error", err)
            }
        }()
    }

    return nil
}
```

## Testing Requirements

### Unit Tests
- Mock HTTP responses for IPFS API
- Test all error paths
- Verify context cancellation

### Integration Tests
- Use testcontainers for real IPFS node
- Test full pin/unpin cycle
- Verify content retrieval

### Coverage Targets
- Minimum 70% for storage implementations
- 90% for orchestrator logic

## Configuration

Environment variables for each provider:

```bash
# Primary Storage
STORAGE_PROVIDER=local|s3|spaces|b2

# Local Storage
LOCAL_STORAGE_PATH=/var/lib/goimg/images

# S3-Compatible
S3_ENDPOINT=https://s3.amazonaws.com
S3_BUCKET=goimg-images
S3_ACCESS_KEY=...
S3_SECRET_KEY=...
S3_REGION=us-east-1

# IPFS (can be enabled alongside any primary)
IPFS_ENABLED=true
IPFS_API_ENDPOINT=http://localhost:5001
IPFS_GATEWAY_ENDPOINT=https://ipfs.io
IPFS_PIN_BY_DEFAULT=true
IPFS_ASYNC_UPLOAD=true
```

## Quick Reference

### IPFS API Endpoints

| Operation | Endpoint | Method |
|-----------|----------|--------|
| Add | `/api/v0/add` | POST |
| Cat | `/api/v0/cat?arg={cid}` | POST |
| Pin Add | `/api/v0/pin/add?arg={cid}` | POST |
| Pin Remove | `/api/v0/pin/rm?arg={cid}` | POST |
| Pin List | `/api/v0/pin/ls` | POST |

### Error Types

```go
var (
    ErrNotFound       = errors.New("storage: not found")
    ErrAlreadyExists  = errors.New("storage: already exists")
    ErrInvalidKey     = errors.New("storage: invalid key")
    ErrProviderError  = errors.New("storage: provider error")
)
```

## Related Guides

- [IPFS Storage Details](../../../claude/ipfs_storage.md)
- [Architecture](../../../claude/architecture.md)
- [Testing](../../../claude/testing_ci.md)
