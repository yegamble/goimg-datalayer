# Sprint 5 Image Processing - Quick Reference

> **TL;DR**: Use bimg (libvips), WebP variants, 82-88% quality, 256MB memory limit, strip all EXIF. Approved to proceed.

---

## Recommended Variant Configuration

```go
// Update internal/domain/gallery/variant_type.go

const (
    VariantThumbnail VariantType = "thumbnail"  // 160px (CHANGED from 150px)
    VariantSmall     VariantType = "small"      // 320px
    VariantMedium    VariantType = "medium"     // 800px
    VariantLarge     VariantType = "large"      // 1600px
    // Phase 2: VariantXLarge  VariantType = "xlarge"  // 2048px (4K)
    VariantOriginal  VariantType = "original"   // Unchanged
)

func (v VariantType) MaxWidth() int {
    switch v {
    case VariantThumbnail:
        return 160  // CHANGED from 150
    case VariantSmall:
        return 320
    case VariantMedium:
        return 800
    case VariantLarge:
        return 1600
    case VariantOriginal:
        return 0
    default:
        return 0
    }
}
```

---

## Format & Quality Settings

| Variant | Format | Quality | Rationale |
|---------|--------|---------|-----------|
| thumbnail | **WebP** | 82 | Higher quality for small sizes |
| small | **WebP** | 85 | Mobile viewing |
| medium | **WebP** | 85 | Web display sweet spot |
| large | **WebP** | 88 | Desktop viewing |
| original | original | original | Never re-encode |

**Why WebP?** 25-34% smaller than JPEG at equivalent quality, 94% browser support.

---

## bimg Configuration (Copy-Paste Ready)

```go
package processing

import (
    "context"
    "fmt"
    "time"

    "github.com/h2non/bimg"
)

// Constants
const (
    MaxImageWidth      = 8192
    MaxImageHeight     = 8192
    MaxPixels          = 100_000_000  // 100 megapixels
    MaxFileSizeBytes   = 10 * 1024 * 1024  // 10MB
    ProcessingTimeout  = 30 * time.Second
    MaxConcurrent      = 32  // Tune based on CPU/memory
)

// Initialize libvips (call once at startup)
func Initialize() {
    bimg.Initialize()
    bimg.VipsCacheSetMax(100)
    bimg.VipsCacheSetMaxMem(256 * 1024 * 1024) // 256MB
}

// Shutdown cleans up (call on graceful shutdown)
func Shutdown() {
    bimg.Shutdown()
}

// ProcessVariant generates a single variant
func ProcessVariant(imageData []byte, variant VariantType) ([]byte, error) {
    options := bimg.Options{
        Width:         variant.MaxWidth(),
        Height:        0,  // Preserve aspect ratio
        Quality:       getQuality(variant),
        Type:          bimg.WEBP,  // Use WebP for variants
        Enlarge:       false,
        StripMetadata: true,  // Remove ALL EXIF
        Interlace:     true,  // Progressive loading
    }

    img := bimg.NewImage(imageData)

    // Validate dimensions (decompression bomb protection)
    size, err := img.Size()
    if err != nil {
        return nil, fmt.Errorf("failed to get image size: %w", err)
    }
    if size.Width > MaxImageWidth || size.Height > MaxImageHeight {
        return nil, fmt.Errorf("dimensions exceed max (%dx%d)", MaxImageWidth, MaxImageHeight)
    }
    if size.Width*size.Height > MaxPixels {
        return nil, fmt.Errorf("too many pixels (max %d)", MaxPixels)
    }

    return img.Process(options)
}

// Quality mapping
func getQuality(variant VariantType) int {
    switch variant {
    case VariantThumbnail:
        return 82
    case VariantSmall:
        return 85
    case VariantMedium:
        return 85
    case VariantLarge:
        return 88
    default:
        return 85
    }
}

// ValidateImage checks before processing
func ValidateImage(data []byte) error {
    if len(data) == 0 {
        return fmt.Errorf("empty image data")
    }
    if len(data) > MaxFileSizeBytes {
        return fmt.Errorf("exceeds 10MB limit")
    }

    img := bimg.NewImage(data)
    metadata, err := img.Metadata()
    if err != nil {
        return fmt.Errorf("invalid image: %w", err)
    }

    // Validate format
    switch metadata.Type {
    case "jpeg", "png", "webp", "gif":
        return nil
    default:
        return fmt.Errorf("unsupported format: %s", metadata.Type)
    }
}
```

---

## Worker Pool Pattern (Prevent Memory Exhaustion)

```go
type ProcessingPool struct {
    sem chan struct{}
}

func NewProcessingPool(maxConcurrent int) *ProcessingPool {
    return &ProcessingPool{
        sem: make(chan struct{}, maxConcurrent),
    }
}

func (p *ProcessingPool) Process(ctx context.Context, fn func() error) error {
    select {
    case p.sem <- struct{}{}:
        defer func() { <-p.sem }()
        return fn()
    case <-ctx.Done():
        return ctx.Err()
    }
}

// Usage
pool := NewProcessingPool(32)
err := pool.Process(ctx, func() error {
    return processImage(imageData)
})
```

---

## Memory Cleanup (Background Goroutine)

```go
func StartMemoryCleanup(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            bimg.VipsCacheDropAll()
            debug.FreeOSMemory()
        case <-ctx.Done():
            return
        }
    }
}
```

---

## Environment Variables

```bash
# .env or docker-compose.yml
IMAGE_MAX_FILE_SIZE=10485760          # 10MB
IMAGE_MAX_WIDTH=8192
IMAGE_MAX_HEIGHT=8192
IMAGE_MAX_PIXELS=100000000
IMAGE_PROCESSING_TIMEOUT=30s
IMAGE_PROCESSING_CONCURRENCY=32
VIPS_CACHE_MAX_OPERATIONS=100
VIPS_CACHE_MAX_MEMORY=268435456       # 256MB
```

---

## Test Fixtures Required

```bash
tests/fixtures/images/
├── valid_jpeg_1mb.jpg
├── valid_png_transparency.png
├── valid_webp_lossy.webp
├── valid_gif_static.gif
├── large_8k_image.jpg (8000x6000, ~10MB)
├── tiny_50x50.png
├── malware_eicar.jpg (EICAR test file)
├── invalid_corrupted.jpg
└── exif_gps_data.jpg (with GPS coordinates)
```

---

## Key Changes from Original Plan

| Item | Original | Recommended | Reason |
|------|----------|-------------|--------|
| Thumbnail size | 150px | **160px** | Better for retina displays |
| Format | JPEG | **WebP** | 25-34% smaller files |
| Quality (variants) | 80-90 | **82-88** | Variant-specific tuning |
| Quality (original) | N/A | **100** | Maximum quality with security re-encoding |
| EXIF handling | TBD | **StripMetadata: true** | Simple, privacy-first |
| Memory limit | TBD | **256MB cache** | Prevent OOM |
| Concurrency | TBD | **32 max workers** | Balance CPU/memory |

---

## Performance Expectations

| Image Size | Resolution | Processing Time (All Variants) |
|------------|------------|-------------------------------|
| 1MB JPEG | 3000x2000 | ~500ms |
| 5MB JPEG | 5000x4000 | ~2-3 seconds |
| 10MB JPEG | 8000x6000 | ~5-8 seconds |

**Note**: Processing is asynchronous via job queue (asynq). Return `202 Accepted` immediately.

---

## Browser Fallback (Phase 2)

```go
// Check Accept header for WebP support
func chooseFormat(acceptHeader string) bimg.ImageType {
    if strings.Contains(acceptHeader, "image/webp") {
        return bimg.WEBP
    }
    return bimg.JPEG  // Fallback for 6% of browsers
}
```

---

## Sprint 5 Implementation Checklist

### Week 1 (Days 1-5)
- [ ] Update `variant_type.go` (thumbnail 160px)
- [ ] Implement `ImageProcessor` service with bimg
- [ ] Add `ProcessVariant` function (WebP, quality settings)
- [ ] Add `ValidateImage` function (MIME, dimensions, size)
- [ ] Unit tests (85%+ coverage)

### Week 2 (Days 6-10)
- [ ] Implement worker pool pattern (concurrency control)
- [ ] Add memory cleanup goroutine
- [ ] ClamAV integration (malware scan before processing)
- [ ] Integration tests with test fixtures
- [ ] Performance benchmarks

---

## Gotchas to Avoid

1. **Original variant quality is 100** - Re-encode at maximum quality for security (prevents polyglot exploits) while preserving image quality
2. **Use magic bytes for MIME detection** - Not file extensions (security)
3. **Validate dimensions BEFORE processing** - Prevent decompression bombs
4. **Call `bimg.Shutdown()` on exit** - Free libvips memory
5. **Use worker pool** - Don't spawn unlimited goroutines
6. **Set timeouts** - Prevent hung operations (30s max)
7. **Animated GIFs**: Process first frame only in MVP

---

## Questions? Escalation Path

- **Technical issues**: senior-go-architect
- **Security concerns**: senior-secops-engineer
- **Testing strategy**: backend-test-architect
- **Feature questions**: image-gallery-expert

---

**Full Analysis**: See `/home/user/goimg-datalayer/docs/sprint5-checkpoint-1-image-processing.md`
