# Sprint 5 Pre-Sprint Checkpoint 1: Image Processing Pipeline Design Review

**Date**: 2025-12-03
**Reviewer**: image-gallery-expert
**Sprint**: Sprint 5 - Domain & Infrastructure - Gallery Context
**Status**: APPROVED WITH RECOMMENDATIONS

---

## Executive Summary

This checkpoint reviews the image processing pipeline design before implementation begins. The current approach using **bimg (libvips wrapper)** is sound and aligns with industry best practices. Key recommendations include:

1. **Adopt WebP format for variants** (25-34% smaller than JPEG at equivalent quality)
2. **Increase thumbnail size to 160px** (match Flickr/Chevereto standards)
3. **Add 4K variant (2048px)** for future-proofing
4. **Use bimg's built-in EXIF stripping** with selective preservation
5. **Set libvips memory limit to 256MB** per operation

**Overall Assessment**: Architecture is solid. Proceed with implementation using recommendations below.

---

## 1. Image Processing Architecture Decision

### Decision: Use bimg (libvips) for Image Processing

**Status**: ✅ APPROVED

### Rationale

| Factor | bimg/libvips | ImageMagick | Go native (image pkg) |
|--------|--------------|-------------|----------------------|
| Performance | 4-8x faster | Baseline | 2-4x slower |
| Memory footprint | Low (streaming) | High | Medium |
| Thread safety | Excellent | Good | Excellent |
| Format support | JPEG, PNG, WebP, GIF, AVIF | Excellent | Limited |
| Go integration | Native CGO bindings | Exec overhead | Native |
| Production maturity | Battle-tested | Battle-tested | Immature for advanced use |

**Winner**: bimg (libvips) offers the best balance of performance, memory efficiency, and Go integration.

### Competitive Analysis

**Flickr**:
- Uses custom image processing pipeline (not publicly documented)
- Supports up to **6K display (6144px)** for Pro members
- Generates 13+ size variants
- Quality preservation is noted as superior to competitors like 500px

**Chevereto**:
- Uses **Imagick (ImageMagick)** with GD fallback
- Generates **3 sizes**: thumbnail, medium, original
- Uses **90% quality** for all variants (industry standard)
- Does NOT compress originals (quality preservation)
- Maximum upload size configurable (default varies by hosting)

**goimg Position**:
Our approach with bimg/libvips positions us **between Flickr (enterprise) and Chevereto (self-hosted)** in terms of performance and features.

---

## 2. Variant Generation Strategy

### Current Plan (Sprint Plan)

| Variant | Max Width | Format | Quality |
|---------|-----------|--------|---------|
| thumbnail | 150px | JPEG | 80 |
| small | 320px | JPEG | 85 |
| medium | 800px | JPEG | 85 |
| large | 1600px | JPEG | 90 |
| original | unchanged | original | original |

### Recommended Changes

#### 2.1 Thumbnail Size: Increase to 160px

**Current**: 150px
**Recommended**: **160px**

**Rationale**:
- 160px aligns with mobile device pixel densities (80px * 2x DPI)
- Better for 2x/3x retina displays
- Chevereto uses configurable thumbs (typically 150-200px)
- Minimal storage difference: ~5-10KB per thumbnail

#### 2.2 Add 4K Variant (2048px)

**Recommended**: Add `VariantXLarge` at **2048px**

**Rationale**:
- Flickr added 1600px and **2048px** sizes as standard options
- Future-proofs for 4K displays (increasingly common)
- Pro/premium tier differentiator opportunity
- Photography portfolios benefit from higher resolution

**Updated Variant Table**:

| Variant | Max Width | Format | Quality | Use Case |
|---------|-----------|--------|---------|----------|
| thumbnail | 160px | WebP | 82 | Grid views, thumbnails |
| small | 320px | WebP | 85 | Mobile devices |
| medium | 800px | WebP | 85 | Tablets, web previews |
| large | 1600px | WebP | 88 | Desktop displays |
| xlarge | 2048px | WebP | 90 | 4K displays, portfolios (Phase 2) |
| original | unchanged | original | 100 | Downloads, archival (maximum quality, near-lossless) |

#### 2.3 Format: Use WebP Instead of JPEG

**Current**: JPEG for all variants
**Recommended**: **WebP for variants, preserve original format for original**

**Rationale**:

**File Size Savings**:
- WebP is **25-34% smaller** than JPEG at equivalent SSIM quality (Google study)
- Independent testing shows **18% median size reduction** vs JPEG
- Bandwidth savings compound across millions of images

**Quality**:
- Lossy WebP superior to JPEG below quality 70
- JPEG slightly better at quality 90+ (edge case for our use)
- WebP quality 80-88 range is ideal for variants

**Browser Support**:
- **94% browser support** (Chrome, Firefox, Edge, Safari 14+)
- Graceful fallback: Serve JPEG if browser doesn't support WebP (check Accept header)

**Performance**:
- Faster decoding than JPEG on modern devices
- Better compression for photographs and graphics

**Implementation**:
```go
// bimg supports WebP natively
options := bimg.Options{
    Width:   width,
    Height:  height,
    Quality: quality,
    Type:    bimg.WEBP,  // Use WebP for variants
    Crop:    false,      // Preserve aspect ratio
}
```

**Future**: Consider AVIF support (31% smaller than JPEG, 10% better than WebP) when browser support exceeds 90%.

#### 2.4 Quality Settings: Fine-Tuned Recommendations

| Variant | Format | Quality | Rationale |
|---------|--------|---------|-----------|
| thumbnail | WebP | 82 | Higher quality for small sizes (more visible artifacts) |
| small | WebP | 85 | Balanced for mobile viewing |
| medium | WebP | 85 | Sweet spot for web display |
| large | WebP | 88 | Higher quality for desktop viewing |
| xlarge | WebP | 90 | Portfolio quality (Phase 2) |
| original | original | 100 | Re-encode at maximum quality for security (prevents polyglot exploits) |

**Quality Rationale**:
- Chevereto uses **90% for all sizes** - we differentiate by use case
- Google recommends **75-85** for WebP; we're slightly higher for quality preservation
- Smaller sizes (thumbnail) need higher quality to avoid visible artifacts
- **Original variant uses quality 100** (maximum, near-lossless) with security re-encoding through libvips to prevent polyglot file exploits while preserving image quality

---

## 3. EXIF Stripping Approach

### Decision: Use bimg's Built-in EXIF Stripping with Selective Preservation

**Status**: ✅ APPROVED

### Privacy-First EXIF Strategy

**Always Strip**:
- GPS coordinates (latitude, longitude, altitude)
- GPS timestamp
- Device serial numbers
- Thumbnail embedded in EXIF
- MakerNote (camera-specific binary data)

**Selectively Preserve** (User-Controlled):
- Copyright information
- Artist/Creator name
- Camera make/model (anonymized)
- Focal length, aperture, ISO (camera settings)
- Creation timestamp (date only, not time)

**Completely Remove from Variants**:
- All variants should have **zero EXIF data** (privacy by default)
- Original preserves user-selected metadata only

### Implementation with bimg

```go
// For variants: Strip ALL EXIF
options := bimg.Options{
    Width:        width,
    Height:       height,
    Quality:      quality,
    Type:         bimg.WEBP,
    StripMetadata: true,  // Remove all EXIF/metadata
}

// For original: Selective stripping using libvips directly
// 1. Extract EXIF to parse
// 2. Remove GPS tags
// 3. Optionally preserve copyright/artist (user setting)
// 4. Re-encode with cleaned EXIF
```

### Alternative Libraries Considered

**Rejected**: goexif, exiftool-wrapper
**Reason**:
- bimg/libvips already handles stripping efficiently
- Additional libraries add complexity
- goexif is read-only (requires separate write library)
- exiftool adds exec overhead

**Recommendation**: Use bimg's `StripMetadata: true` for variants, implement selective stripping for originals using libvips EXIF APIs if needed in Phase 2.

### Privacy Best Practices Alignment

Based on research from EXIF privacy guides:

1. **Default to maximum privacy**: Strip GPS by default (most critical)
2. **User control**: Let users opt-in to preserve copyright/artist metadata
3. **Re-encoding protection**: Re-encoding through libvips prevents polyglot file exploits
4. **Social media alignment**: Match behavior of Facebook/Instagram (strip public-facing EXIF)

Sources: [EXIF Data Privacy Guide](https://exifdata.org/blog/exif-data-privacy-the-ultimate-guide-to-protecting-your-image-metadata), [GPS Metadata Privacy](https://exifdata.org/blog/photo-gps-data-privacy-guide-to-exif-location-removal)

---

## 4. Memory Limits for libvips

### Recommended Configuration

```go
package processing

import (
    "github.com/h2non/bimg"
)

func init() {
    // Set libvips cache limits
    bimg.VipsCacheSetMax(100)              // Max 100 operations in cache
    bimg.VipsCacheSetMaxMem(256 * 1024 * 1024) // Max 256MB memory for cache

    // Optional: Enable libvips memory tracking for debugging
    // bimg.VipsDebugInfo() // Outputs to stdout
}

// Per-operation limits
const (
    MaxImageWidth       = 8192   // 8K width
    MaxImageHeight      = 8192   // 8K height
    MaxPixels           = 100_000_000 // 100 megapixels (prevents decompression bombs)
    MaxFileSizeBytes    = 10 * 1024 * 1024 // 10MB upload limit
    ProcessingTimeout   = 30 * time.Second  // Max 30 seconds per image
)
```

### Memory Management Strategy

**Per-Request Limits**:
- **256MB cache limit**: Prevents runaway memory growth
- **100 operations max**: Reasonable for concurrent processing
- **30 second timeout**: Prevents hung operations

**Decompression Bomb Protection**:
- Max dimensions: **8192x8192 pixels**
- Max pixels: **100 million** (e.g., 10000x10000)
- Reject images exceeding limits before libvips processing

**Cleanup Strategy**:
```go
// Periodic cache cleanup (run in background goroutine)
func periodicCleanup(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            bimg.VipsCacheDropAll() // Drop vips cache
            debug.FreeOSMemory()    // Return memory to OS
        case <-ctx.Done():
            return
        }
    }
}

// On graceful shutdown
func Shutdown() {
    bimg.Shutdown() // Thread-safe libvips shutdown
}
```

### Concurrency Considerations

**Worker Pool Pattern** (recommended):
```go
// Limit concurrent image processing
type ProcessingPool struct {
    sem chan struct{}
}

func NewProcessingPool(maxConcurrent int) *ProcessingPool {
    return &ProcessingPool{
        sem: make(chan struct{}, maxConcurrent),
    }
}

func (p *ProcessingPool) Process(ctx context.Context, img []byte) error {
    select {
    case p.sem <- struct{}{}:
        defer func() { <-p.sem }()
        return processImage(ctx, img)
    case <-ctx.Done():
        return ctx.Err()
    }
}

// Recommended: 4-8 concurrent operations per CPU core
const MaxConcurrentProcessing = 32 // Tune based on CPU/memory
```

**Rationale**:
- libvips is **thread-safe** but CPU/memory bound
- Too many concurrent operations cause memory pressure
- Worker pool prevents resource exhaustion

### Memory Leak Detection

From Stack Overflow research on bimg memory behavior:

```go
// Monitor memory usage (debugging)
func logMemoryStats() {
    info := bimg.VipsMemoryInfo()
    log.Printf("Vips Memory: %d bytes, Highwater: %d, Allocations: %d",
        info.Memory, info.MemoryHighwater, info.Allocations)
}

// Set leak detection (development only)
// os.Setenv("VIPS_LEAK", "1") // Enables libvips leak tracking
```

**Note**: CGO memory behavior means Go's GC doesn't directly manage libvips memory. Use `VipsCacheDropAll()` and `Shutdown()` to free memory explicitly.

Sources: [bimg GitHub](https://github.com/h2non/bimg), [Stack Overflow: Memory not released](https://stackoverflow.com/questions/54844282/memory-not-being-released-back-to-os)

---

## 5. Image Format Support

### MVP Formats (Sprint 5)

| Format | MIME Type | Priority | Notes |
|--------|-----------|----------|-------|
| JPEG | image/jpeg | P0 | Universal support, photography |
| PNG | image/png | P0 | Lossless, transparency, graphics |
| GIF | image/gif | P0 | Animated GIF support (Phase 2) |
| WebP | image/webp | P0 | Modern format, better compression |

### Phase 2 Formats

| Format | MIME Type | Priority | Browser Support | Notes |
|--------|-----------|----------|-----------------|-------|
| AVIF | image/avif | P2 | ~80% (2024) | 10% better than WebP, future default |
| HEIC | image/heic | P3 | Safari only | iOS default, requires conversion |

### Format Detection

**Use Magic Bytes, NOT Extensions**:
```go
// CRITICAL: Detect MIME type by magic bytes (prevent MIME type confusion attacks)
func detectMIME(data []byte) (string, error) {
    // bimg can infer type
    metadata, err := bimg.NewImage(data).Metadata()
    if err != nil {
        return "", err
    }

    switch metadata.Type {
    case "jpeg":
        return "image/jpeg", nil
    case "png":
        return "image/png", nil
    case "webp":
        return "image/webp", nil
    case "gif":
        return "image/gif", nil
    default:
        return "", ErrUnsupportedFormat
    }
}
```

### Animated GIF Handling

**MVP**: Accept GIF uploads, **process first frame only** for variants
**Phase 2**: Preserve animation in original, generate animated WebP variants

**Rationale**:
- Animated GIF processing is complex (frame extraction, optimization)
- First-frame thumbnails are industry standard
- Future: Use `libvips` animated WebP support

---

## 6. Recommended bimg Configuration

### Complete Configuration Code Snippet

```go
package processing

import (
    "context"
    "fmt"
    "time"

    "github.com/h2non/bimg"
)

// Initialize libvips (call once at startup)
func Initialize() error {
    bimg.Initialize()

    // Set cache limits
    bimg.VipsCacheSetMax(100)
    bimg.VipsCacheSetMaxMem(256 * 1024 * 1024) // 256MB

    return nil
}

// Shutdown cleans up libvips resources
func Shutdown() {
    bimg.Shutdown()
}

// ImageProcessor handles image transformations
type ImageProcessor struct {
    maxConcurrent int
    sem           chan struct{}
}

func NewImageProcessor(maxConcurrent int) *ImageProcessor {
    return &ImageProcessor{
        maxConcurrent: maxConcurrent,
        sem:           make(chan struct{}, maxConcurrent),
    }
}

// ProcessVariant generates a single image variant
func (p *ImageProcessor) ProcessVariant(
    ctx context.Context,
    imageData []byte,
    variant VariantType,
) ([]byte, error) {
    // Acquire semaphore
    select {
    case p.sem <- struct{}{}:
        defer func() { <-p.sem }()
    case <-ctx.Done():
        return nil, ctx.Err()
    }

    // Set processing timeout
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    // Configure options based on variant
    options := bimg.Options{
        Width:         variant.MaxWidth(),
        Height:        0, // Preserve aspect ratio
        Quality:       getQualityForVariant(variant),
        Type:          bimg.WEBP,
        Enlarge:       false, // Don't upscale small images
        StripMetadata: true,  // Remove ALL EXIF for variants
        Interlace:     true,  // Progressive loading
    }

    // Process with bimg
    img := bimg.NewImage(imageData)

    // Validate dimensions BEFORE processing (decompression bomb protection)
    size, err := img.Size()
    if err != nil {
        return nil, fmt.Errorf("failed to get image size: %w", err)
    }

    if size.Width > 8192 || size.Height > 8192 {
        return nil, fmt.Errorf("image dimensions exceed maximum (8192x8192): %dx%d",
            size.Width, size.Height)
    }

    pixels := size.Width * size.Height
    if pixels > 100_000_000 {
        return nil, fmt.Errorf("image has too many pixels (%d > 100M)", pixels)
    }

    // Transform
    processed, err := img.Process(options)
    if err != nil {
        return nil, fmt.Errorf("failed to process image: %w", err)
    }

    return processed, nil
}

// Quality mapping
func getQualityForVariant(variant VariantType) int {
    switch variant {
    case VariantThumbnail:
        return 82 // Higher quality for small sizes
    case VariantSmall:
        return 85
    case VariantMedium:
        return 85
    case VariantLarge:
        return 88
    case VariantXLarge: // Phase 2
        return 90
    default:
        return 85
    }
}

// ValidateImage checks image before processing
func ValidateImage(data []byte) error {
    if len(data) == 0 {
        return fmt.Errorf("empty image data")
    }

    if len(data) > 10*1024*1024 { // 10MB limit
        return fmt.Errorf("image exceeds 10MB limit")
    }

    // Check if bimg can parse it
    img := bimg.NewImage(data)
    metadata, err := img.Metadata()
    if err != nil {
        return fmt.Errorf("invalid image format: %w", err)
    }

    // Validate format
    switch metadata.Type {
    case "jpeg", "png", "webp", "gif":
        // OK
    default:
        return fmt.Errorf("unsupported format: %s", metadata.Type)
    }

    return nil
}
```

---

## 7. Concerns and Risks

### 7.1 libvips Platform Availability

**Risk**: libvips requires native C library installation

**Impact**: Medium
**Probability**: High (developer onboarding)

**Mitigation**:
- **Docker-first development**: All developers use Docker Compose (libvips pre-installed)
- Clear setup docs with platform-specific instructions
- CI/CD runs in Docker (no platform issues)
- Production deployment via Docker/K8s

**Platform Status**:
- Linux: Excellent support (apt, yum packages)
- macOS: Homebrew (`brew install vips`)
- Windows: More complex (vcpkg or pre-built binaries)

### 7.2 WebP Browser Support Gap

**Risk**: 6% of browsers don't support WebP (older Safari, IE11)

**Impact**: Low
**Probability**: Low (declining over time)

**Mitigation**:
- Check `Accept: image/webp` header
- Fallback to JPEG for unsupported browsers
- Future: Content negotiation strategy

**Implementation**:
```go
func chooseFormat(acceptHeader string) bimg.ImageType {
    if strings.Contains(acceptHeader, "image/webp") {
        return bimg.WEBP
    }
    return bimg.JPEG // Fallback
}
```

### 7.3 Animated GIF Complexity

**Risk**: Animated GIFs require special handling (frame extraction, optimization)

**Impact**: Medium
**Probability**: Medium (users upload animated GIFs)

**Mitigation**:
- MVP: Process first frame only for variants (industry standard)
- Preserve original GIF with animation intact
- Phase 2: Implement animated WebP variants (better compression)

### 7.4 Memory Pressure Under Load

**Risk**: High concurrent image processing can exhaust memory

**Impact**: High (OOM crashes)
**Probability**: Medium (under heavy load)

**Mitigation**:
- Worker pool pattern (max 32 concurrent operations)
- libvips cache limits (256MB)
- Kubernetes resource limits (memory requests/limits)
- Monitoring: Track libvips memory usage with Prometheus
- Horizontal scaling: Run multiple worker pods

### 7.5 Processing Time SLA

**Risk**: Large images (10MB, 8000x8000px) may take >10 seconds to process

**Impact**: Medium (user experience)
**Probability**: Medium

**Mitigation**:
- Asynchronous processing with job queue (asynq)
- Immediate response: Return 202 Accepted with processing status
- Polling endpoint: Check processing status
- Websocket/SSE: Real-time progress updates (Phase 2)

**Expected Performance** (based on libvips benchmarks):
- 1MB JPEG (3000x2000): ~500ms for all variants
- 5MB JPEG (5000x4000): ~2-3 seconds
- 10MB JPEG (8000x6000): ~5-8 seconds

### 7.6 AVIF Adoption Timeline

**Risk**: AVIF (next-gen format) has 80% browser support but not universal

**Impact**: Low (nice-to-have)
**Probability**: High (will improve)

**Mitigation**:
- MVP: Skip AVIF (too early)
- Monitor browser support (wait for 90%+)
- Phase 3: Add AVIF as optional format
- Use content negotiation: `Accept: image/avif,image/webp,image/*`

---

## 8. Comparison with Competitors

### Feature Matrix

| Feature | Flickr | Chevereto | goimg (MVP) | goimg (Phase 2) |
|---------|--------|-----------|-------------|-----------------|
| Max variants | 13+ | 3 | 5 | 6 |
| Max display size | 6144px (6K) | Configurable | 1600px | 2048px (4K) |
| Format support | JPEG, PNG, GIF | JPEG, PNG, GIF, WebP, BMP | JPEG, PNG, GIF, WebP | + AVIF |
| Processing lib | Custom | ImageMagick | libvips | libvips |
| Quality | High | 90% JPEG | 82-88% WebP (variants), 100 (original) | Adaptive |
| EXIF stripping | Yes | Limited | Privacy-first | Selective |
| Original compression | Yes | No | Yes (Q100 security re-encode) | Yes (Q100) |
| Animated GIF | Full support | Full support | First frame | Animated WebP |
| Performance | Enterprise | Good | Excellent | Excellent |

### Competitive Positioning

**goimg vs Flickr**:
- Flickr: Enterprise-scale, custom infrastructure, 13+ variants, 6K support
- goimg: Self-hosted, open-source, 5 variants, 1600px (Phase 2: 4K)
- **Advantage**: Privacy-first EXIF stripping, modern WebP format, faster processing (libvips)

**goimg vs Chevereto**:
- Chevereto: PHP, ImageMagick, 3 variants, 90% JPEG quality
- goimg: Go, libvips, 5 variants, WebP format, 82-88% quality (variants), 100% quality (original)
- **Advantage**: 4-8x faster processing, smaller file sizes (WebP), better memory efficiency, maximum quality original preservation

**Unique Differentiators**:
1. **WebP-first strategy**: 25-34% smaller files vs competitors using JPEG
2. **Privacy by default**: Aggressive EXIF stripping (GPS always removed)
3. **Performance**: libvips 4-8x faster than ImageMagick
4. **Future-ready**: AVIF support planned (Phase 3)

---

## 9. Implementation Recommendations

### Priority Order for Sprint 5

1. **Week 1 (Days 1-5)**:
   - [ ] Implement `ImageProcessor` service with bimg
   - [ ] Variant generation (thumbnail, small, medium, large)
   - [ ] MIME type validation (magic bytes)
   - [ ] Dimension validation (8192x8192, 100M pixels)
   - [ ] Unit tests for image processing logic

2. **Week 2 (Days 6-10)**:
   - [ ] EXIF stripping implementation (StripMetadata: true)
   - [ ] ClamAV integration (malware scanning)
   - [ ] Storage provider interface + local implementation
   - [ ] Integration tests with sample images (JPEG, PNG, GIF, WebP)
   - [ ] Performance benchmarks

### Testing Requirements

**Test Images** (add to `tests/fixtures/`):
```
tests/fixtures/images/
├── valid_jpeg_1mb.jpg
├── valid_png_transparency.png
├── valid_webp_lossy.webp
├── valid_gif_static.gif
├── large_8k_image.jpg (8000x6000, ~10MB)
├── tiny_50x50.png
├── malware_eicar.jpg (EICAR test file)
├── invalid_corrupted.jpg
└── exif_gps_data.jpg (with GPS tags)
```

**Coverage Targets**:
- Image processing service: 85%+
- Variant generation: 90%+
- Validation logic: 95%+

**Performance Tests**:
- Benchmark variant generation time
- Measure memory usage (libvips cache)
- Concurrent processing stress test (32 simultaneous ops)

### Configuration (Environment Variables)

```bash
# Image Processing
IMAGE_MAX_FILE_SIZE=10485760          # 10MB in bytes
IMAGE_MAX_WIDTH=8192
IMAGE_MAX_HEIGHT=8192
IMAGE_MAX_PIXELS=100000000            # 100 megapixels
IMAGE_PROCESSING_TIMEOUT=30s
IMAGE_PROCESSING_CONCURRENCY=32       # Max concurrent operations

# libvips
VIPS_CACHE_MAX_OPERATIONS=100
VIPS_CACHE_MAX_MEMORY=268435456       # 256MB in bytes

# Storage
STORAGE_PROVIDER=local                # local, s3, ipfs
STORAGE_LOCAL_PATH=/var/goimg/uploads
```

### Phase 2 Enhancements (Sprint 6+)

- [ ] Add VariantXLarge (2048px, 4K)
- [ ] AVIF format support (when browser support >90%)
- [ ] Animated GIF support (extract frames, optimize)
- [ ] Animated WebP variants
- [ ] Selective EXIF preservation (copyright, artist)
- [ ] Smart crop for thumbnails (face detection)
- [ ] Watermarking support
- [ ] Custom variant sizes (user-defined)

---

## 10. Final Recommendations

### ✅ APPROVED TO PROCEED

The image processing architecture is **solid and well-designed**. Proceed with implementation in Sprint 5 using the following adjustments:

1. **Adopt WebP format** for all variants (25-34% size reduction)
2. **Increase thumbnail to 160px** (better for retina displays)
3. **Use quality range 82-88** (variant-specific, not one-size-fits-all)
4. **Set libvips memory limit to 256MB** with 100 operation cache
5. **Plan for 2048px variant** in Phase 2 (match Flickr 4K support)
6. **Use bimg's StripMetadata: true** for variants (simple, effective)
7. **Implement worker pool pattern** (max 32 concurrent operations)

### Architecture Decision Record

**ADR-005: Image Processing with bimg/libvips**

**Status**: Accepted
**Date**: 2025-12-03
**Context**: Need fast, memory-efficient image processing for MVP
**Decision**: Use bimg (libvips wrapper) with WebP variants
**Consequences**:
- ✅ 4-8x faster than ImageMagick
- ✅ 25-34% smaller files (WebP vs JPEG)
- ✅ Low memory footprint (streaming processing)
- ⚠️ Requires libvips C library (Docker-first development)
- ⚠️ 6% browsers need JPEG fallback (declining issue)

**Alternatives Considered**:
- ImageMagick: Slower, higher memory usage
- Go native image package: Limited format support, immature for advanced use

**Approval**: image-gallery-expert ✓

---

## Sources and References

### Research Sources

**Flickr**:
- [Flickr 6K Display Support for Pros](https://www.dpreview.com/news/6868253357/flickr-upgrades-maximum-display-resolution-to-6k-for-pro-members)
- [Flickr Photo Size Options](https://www.flickr.com/help/forum/en-us/72157629593354820/)

**Chevereto**:
- [Chevereto Image Upload Settings](https://v3-docs.chevereto.com/settings/image-upload.html)
- [Chevereto Image Compression Discussion](https://chevereto.com/community/threads/image-compression.4511/)

**bimg/libvips**:
- [bimg GitHub Repository](https://github.com/h2non/bimg)
- [bimg Go Package Documentation](https://pkg.go.dev/gopkg.in/h2non/bimg.v0)

**WebP vs JPEG**:
- [Google WebP Compression Study](https://developers.google.com/speed/webp/docs/webp_study)
- [Cloudinary WebP Format Guide](https://cloudinary.com/guides/front-end-development/webp-format-technology-pros-cons-and-alternatives)
- [Photutorial Image Format Comparison](https://photutorial.com/image-format-comparison-statistics/)

**EXIF Privacy**:
- [EXIF Data Privacy Guide](https://exifdata.org/blog/exif-data-privacy-the-ultimate-guide-to-protecting-your-image-metadata)
- [Photo GPS Data Privacy](https://exifdata.org/blog/photo-gps-data-privacy-guide-to-exif-location-removal)
- [How to Remove EXIF Metadata](https://exif.pro/blog/privacy-metadata-removal/)

---

**Reviewer**: image-gallery-expert
**Approval Date**: 2025-12-03
**Next Checkpoint**: Mid-Sprint (Day 7) - Verify variant generation quality and performance
