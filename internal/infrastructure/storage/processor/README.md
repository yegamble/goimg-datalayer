# Image Processor

The image processor handles all image processing operations using **bimg** (libvips bindings for Go).

## Features

- **Variant Generation**: Creates multiple size variants (thumbnail, small, medium, large, original)
- **EXIF Stripping**: Removes metadata for privacy and security
- **Format Conversion**: Converts to WebP for optimized variants
- **Polyglot Protection**: Re-encodes originals through libvips to prevent exploits
- **Worker Pool**: Limits concurrent operations to prevent resource exhaustion
- **Memory Management**: Configurable memory limits for libvips cache

## Variants

| Variant | Max Width | Format | Quality | Use Case |
|---------|-----------|--------|---------|----------|
| Thumbnail | 160px | WebP | 82 | Image previews, galleries |
| Small | 320px | WebP | 85 | Mobile devices |
| Medium | 800px | WebP | 85 | Tablets, web previews |
| Large | 1600px | WebP | 88 | Desktop displays |
| Original | unchanged | original | 90 | Full-size download |

## Usage

```go
import "github.com/yegamble/goimg-datalayer/internal/infrastructure/storage/processor"

// Create processor with default config
cfg := processor.DefaultConfig()
proc, err := processor.New(cfg)
if err != nil {
    return err
}
defer proc.Shutdown()

// Process an image
ctx := context.Background()
imageData, _ := os.ReadFile("photo.jpg")

result, err := proc.Process(ctx, imageData, "photo.jpg")
if err != nil {
    return err
}

// Access variants
fmt.Printf("Thumbnail: %dx%d, %d bytes\n",
    result.Thumbnail.Width,
    result.Thumbnail.Height,
    result.Thumbnail.FileSize)

// Generate a single variant
variant, err := proc.GenerateVariant(ctx, imageData, processor.VariantMedium)
if err != nil {
    return err
}
```

## Configuration

```go
cfg := processor.Config{
    MemoryLimitMB:    256,  // libvips cache limit
    MaxConcurrentOps: 32,   // Worker pool size
    StripMetadata:    true, // Always strip EXIF
    ThumbnailQuality: 82,
    SmallQuality:     85,
    MediumQuality:    85,
    LargeQuality:     88,
    OriginalQuality:  90,
}

proc, err := processor.New(cfg)
```

## Processing Pipeline

1. **Decode & Validate**: Verify image format and dimensions
2. **Strip Metadata**: Remove EXIF data (GPS, camera info, etc.)
3. **Generate Variants**: Create resized WebP versions
4. **Re-encode Original**: Process through libvips for security

## Security Features

- **EXIF Stripping**: Removes GPS coordinates and camera metadata
- **Format Validation**: Only processes JPEG, PNG, GIF, WebP
- **Polyglot Prevention**: Re-encodes originals to prevent exploit files
- **Resource Limits**: Memory caps and concurrent operation limits
- **Dimension Validation**: Rejects invalid or malicious dimensions

## Performance

- **Target**: Process 10MB image in <30 seconds
- **Memory**: 256MB cache limit (configurable)
- **Concurrency**: 32 simultaneous operations (configurable)
- **Optimization**: libvips uses SIMD and multi-threading

## Error Handling

```go
result, err := proc.Process(ctx, imageData, "photo.jpg")
if err != nil {
    switch {
    case errors.Is(err, processor.ErrUnsupportedFormat):
        // Handle unsupported format
    case errors.Is(err, processor.ErrInvalidDimensions):
        // Handle invalid dimensions
    case errors.Is(err, processor.ErrProcessingFailed):
        // Handle processing failure
    default:
        // Handle other errors
    }
}
```

## Testing

Run unit tests:
```bash
go test ./internal/infrastructure/storage/processor/...
```

Run integration tests (requires libvips):
```bash
go test -v ./internal/infrastructure/storage/processor/...
```

Skip integration tests:
```bash
go test -short ./internal/infrastructure/storage/processor/...
```

## Dependencies

- **bimg**: https://github.com/h2non/bimg
- **libvips**: Requires libvips 8.0+ to be installed

### Installing libvips

**macOS**:
```bash
brew install vips
```

**Ubuntu/Debian**:
```bash
apt-get install libvips-dev
```

**Docker**:
```dockerfile
FROM golang:1.24-alpine
RUN apk add --no-cache vips-dev gcc musl-dev
```

## Architecture Notes

- **Infrastructure Layer**: This processor is in the infrastructure layer
- **No Domain Imports**: Does not import domain packages (DDD layering)
- **Standalone**: Can be used independently of domain logic
- **Stateless**: All methods are context-aware and cancellable

## Related Files

- `config.go`: Configuration and variant specifications
- `variants.go`: Variant types and result structures
- `processor.go`: Main processing implementation
- `processor_test.go`: Unit and integration tests
