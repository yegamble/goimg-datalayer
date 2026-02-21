# Storage Orchestrator Pattern

Multi-backend storage for images: primary (local/S3/DO Spaces/B2) + optional IPFS.

## Architecture

```
infrastructure/storage/
├── storage.go       # Storage interface + PutOptions + ErrNotFound
├── keys.go          # KeyGenerator: generates/validates keys (path traversal safe)
├── errors.go        # ErrNotFound, ErrInvalidKey
├── local/           # Filesystem storage (dev)
├── s3/              # S3-compatible (AWS, DO Spaces, B2)
├── ipfs/            # IPFS Kubo client + remote pinning (Pinata/Infura)
├── validator/       # 7-step image validation pipeline
├── processor/       # bimg/libvips image variant generation
└── orchestrator/    # Coordinates primary + IPFS
```

## Orchestrator Modes

| Mode | Behavior |
|------|----------|
| `primary_only` | Write/read only from primary (default) |
| `dual_sync` | Write to primary + IPFS synchronously |
| `dual_async` | Write to primary immediately, IPFS in background |

```go
// Create orchestrator
orch, err := orchestrator.New(primaryStorage, ipfsClient, orchestrator.Config{
    Mode:            orchestrator.ModeDualAsync,
    FallbackEnabled: true,  // Read from IPFS if primary fails
    IPFSEnabled:     true,
})
```

## Storage Key Format

```
images/{owner_id}/{image_id}/{variant}.{ext}
```

Always use `KeyGenerator` — never hand-construct keys:
```go
gen := storage.NewKeyGenerator()
key := gen.GenerateKey(ownerID, imageID, "thumbnail", "jpg")
// → "images/550e.../7c9e.../thumbnail.jpg"

if err := gen.ValidateKey(key); err != nil { ... }  // Path traversal check
```

## Image Validation Pipeline (validator/)

7 steps executed in order before storage:
1. Size ≤ 10MB
2. MIME type detection (by content, not extension)
3. Magic bytes validation
4. Dimensions ≤ 8192×8192
5. Pixel count ≤ 100M (decompression bomb)
6. ClamAV malware scan
7. Filename sanitization

```go
v := validator.New(validator.DefaultConfig(), clamavClient)
result, err := v.Validate(ctx, imageData, filename)
if err != nil || !result.Valid { ... }
```

## Adding a New Storage Provider

1. Create package under `internal/infrastructure/storage/{provider}/`
2. Implement the `storage.Storage` interface
3. Add `Put`, `PutBytes`, `Get`, `GetBytes`, `Delete`, `Stat`, `URL` methods
4. Never import domain packages
5. Return only stdlib types (no provider-specific structs)
6. Wire in `cmd/api/main.go` based on `STORAGE_PROVIDER` env var

## Configuration (env vars)

```bash
STORAGE_PROVIDER=local|s3|spaces|b2

# S3-compatible
S3_ENDPOINT=https://s3.amazonaws.com
S3_BUCKET=goimg-images
S3_ACCESS_KEY=...
S3_SECRET_KEY=...
S3_REGION=us-east-1

# IPFS (add-on to primary)
IPFS_ENABLED=true
IPFS_API_ENDPOINT=http://localhost:5001
IPFS_GATEWAY_ENDPOINT=https://ipfs.io
IPFS_ASYNC_UPLOAD=true
```

## Coverage Targets

- Storage implementations: 70%
- Orchestrator logic: 90%
- Unit tests: mock `storage.Storage` interface
- Integration tests: testcontainers or temp directory for local
