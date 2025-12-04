# Security Test Suite

Comprehensive security tests for the goimg-datalayer image upload validation pipeline.

## Overview

This test suite validates the security controls implemented in Sprint 5 for image upload, storage key generation, and malware scanning. All tests are designed to verify specific security controls against OWASP Top 10 and common attack vectors.

## Test Structure

```
tests/security/
├── upload_test.go           # Image upload validation security tests
├── storage_test.go          # Storage key generation and validation tests
├── fixtures/                # Test files for security validation
│   ├── eicar.txt           # EICAR antivirus test file (safe)
│   ├── clean_image.jpg     # Minimal valid JPEG (68 bytes)
│   ├── oversized.bin       # 11MB file for size limit testing
│   ├── fake_jpeg.jpg       # Text file with .jpg extension
│   └── README.md           # Fixture documentation
└── mocks/
    └── clamav_mock.go      # Mock ClamAV scanner for unit tests
```

## Test Coverage

### Upload Security Tests (`upload_test.go`)

#### 1. TestUpload_RejectsOversizedFile
**Security Control**: File size validation prevents DoS via resource exhaustion
**Test Cases**:
- Files under limit (5MB) - PASS
- Files at exact limit (10MB) - PASS
- Files exceeding limit (11MB, 50MB) - REJECT

**Error**: `gallery.ErrFileTooLarge`

#### 2. TestUpload_ValidatesMIMEByContent
**Security Control**: MIME type detection by content, not extension (prevents file type confusion attacks)
**Test Cases**:
- Valid JPEG with .jpg extension - PASS
- Valid JPEG with wrong .png extension - PASS (content wins)
- Text file with .jpg extension - REJECT

**Error**: `gallery.ErrInvalidMimeType`

#### 3. TestUpload_RejectsMalware
**Security Control**: ClamAV malware scanning prevents upload of infected files
**Test Cases**:
- EICAR signature embedded in JPEG - REJECT
- Clean image - PASS

**Error**: `gallery.ErrMalwareDetected`

**Note**: Pure EICAR text file is rejected at MIME validation (defense in depth). This test uses EICAR embedded in valid JPEG to verify malware scanner invocation.

#### 4. TestUpload_RejectsPolyglotFile
**Security Control**: Polyglot file detection (files valid in multiple formats)
**Test Cases**:
- JPEG/HTML polyglot with embedded JavaScript - DETECT

**Defense**: Multi-layer validation (MIME, magic bytes, re-encoding via bimg)

#### 5. TestUpload_SanitizesFilename
**Security Control**: Filename sanitization prevents directory traversal attacks
**Test Cases**:
- Path traversal (`../../etc/passwd`) - Sanitized to `passwd`
- Absolute paths (`/etc/passwd`) - Sanitized to `passwd`
- Windows paths (`..\\..\\calc.exe`) - Sanitized
- Null bytes (`file\x00.jpg.exe`) - Replaced with `_`
- Dangerous characters (`<>:"/\|?*`) - Removed or replaced
- Long filenames (300 chars) - Truncated to 200 chars
- Unicode characters - Preserved (validator allows, storage removes)

**Function**: `validator.SanitizeFilename()`

#### 6. TestUpload_EnforcesDimensionLimits
**Security Control**: Image dimension limits prevent decompression bomb attacks
**Test Cases**:
- Dimensions within limits (1920x1080) - PASS
- Exactly at maximum (8192x8192) - PASS
- Width exceeds limit (8193x1080) - REJECT
- Height exceeds limit (1920x8193) - REJECT
- Invalid dimensions (0, negative) - REJECT

**Error**: `gallery.ErrImageTooLarge`, `gallery.ErrInvalidDimensions`

#### 7. TestUpload_EnforcesPixelCountLimit
**Security Control**: Total pixel count limit prevents memory exhaustion
**Test Cases**:
- Pixel count within limit (1920x1080 = 2M) - PASS
- Exactly at limit (10000x10000 = 100M) - PASS
- Exceeds limit (10001x10001 = 100M+) - REJECT
- Decompression bomb attempt (65535x65535 = 4.3B) - REJECT

**Error**: `gallery.ErrImageTooManyPixels`

**Configuration**: Default 100M pixels (10000x10000)

#### 8. TestUpload_ValidatesMagicBytes
**Security Control**: Magic byte validation provides defense-in-depth against file type confusion
**Test Cases**:
- Valid JPEG magic bytes (`0xFF 0xD8 0xFF`) - PASS
- Valid PNG magic bytes (`0x89 PNG...`) - PASS
- Valid GIF magic bytes (`GIF87a`, `GIF89a`) - PASS
- Valid WebP magic bytes (`RIFF...WEBP`) - PASS
- Invalid magic bytes (plain text, PDF) - REJECT
- File too small (<12 bytes) - REJECT

**Error**: `gallery.ErrInvalidMimeType`

#### 9. TestUpload_MalwareScanDisabled
**Security Control**: Graceful degradation when ClamAV unavailable
**Test Cases**:
- Valid image with malware scanning disabled - PASS (no scan result)

### Storage Security Tests (`storage_test.go`)

#### 1. TestStorage_PreventsPathTraversal
**Security Control**: Path traversal prevention protects against directory escape attacks
**Test Cases**:
- Valid image key - PASS
- Parent directory (`../`) - REJECT
- Multiple levels (`../../`) - REJECT
- Path traversal in middle - REJECT
- Encoded traversal (`..%2F..%2F`) - REJECT
- Absolute paths (`/`, `\`) - REJECT
- Null byte injection - REJECT
- Non-canonical paths (extra slashes) - REJECT

**Error**: `storage.ErrPathTraversal`, `storage.ErrInvalidKey`

#### 2. TestStorage_GeneratesNonGuessableKeys
**Security Control**: UUID-based keys prevent enumeration attacks
**Verification**:
- Keys contain valid UUIDs (not sequential integers)
- Same owner/image, different variant produces different keys
- Generated keys follow format: `images/{owner_uuid}/{image_uuid}/{variant}.{ext}`

#### 3. TestStorage_ValidatesKeyFormat
**Security Control**: Strict key format validation prevents injection attacks
**Test Cases**:
- Valid keys (thumbnail, original, variants) - PASS
- Malformed UUIDs (owner or image) - REJECT
- Unsupported file extensions (.exe) - REJECT
- Missing variant - REJECT
- Wrong path component count - REJECT
- Uppercase in variant (must be lowercase) - REJECT

**Error**: `storage.ErrInvalidKey`

**Format**: `images/{owner-uuid}/{image-uuid}/{variant}.{ext}`

#### 4. TestStorage_ParseKey
**Security Control**: Key component extraction validates before use
**Test Cases**:
- Valid keys parsed correctly
- Invalid keys rejected with clear errors

#### 5. TestStorage_SanitizeFilename
**Security Control**: Stored filename sanitization prevents executable/special characters
**Test Cases**:
- Path components removed
- Spaces replaced with underscores
- Special characters removed (keeps only alphanumeric, `.`, `-`, `_`)
- Unicode characters removed
- Long filenames truncated to 200 chars
- Empty/invalid filenames get default "unnamed.jpg"

**Function**: `storage.SanitizeFilename()` - more restrictive than validator version

#### 6. TestStorage_KeyGeneration_Uniqueness
**Security Control**: Ensures no key collisions
**Verification**: Generates 1000 keys, verifies all unique

#### 7. TestStorage_KeyGeneration_FormatNormalization
**Security Control**: Consistent file extensions prevent MIME confusion
**Test Cases**:
- MIME types normalized to extensions (`image/jpeg` → `jpg`)
- Extensions normalized (`jpeg` → `jpg`, `PNG` → `png`)
- Unsupported formats default to `jpg`

## Test Fixtures

### eicar.txt
Standard EICAR antivirus test file. **NOT actual malware** - safe to commit.

```
X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*
```

**Purpose**: Test ClamAV malware detection
**Source**: https://www.eicar.org/download-anti-malware-testfile/

### clean_image.jpg
Minimal valid JPEG file (68 bytes) for positive tests.

**Structure**:
- SOI marker (0xFF 0xD8)
- JFIF header (1x1 pixel, grayscale)
- Minimal Huffman table
- EOI marker (0xFF 0xD9)

### oversized.bin
11MB binary file (exceeds 10MB limit).

**Creation**: `dd if=/dev/zero of=oversized.bin bs=1M count=11`

### fake_jpeg.jpg
Plain text file with `.jpg` extension.

**Purpose**: Test MIME type validation by content, not extension

## Mock Implementations

### MockClamAVScanner (`mocks/clamav_mock.go`)

**Features**:
- Customizable scan behavior via function injection
- Call tracking for verification
- Default: all scans return clean
- `NewMalwareDetectingScanner()`: Detects EICAR signature

**Methods**:
- `Scan(ctx, data)` - Scan byte array
- `ScanReader(ctx, reader, size)` - Scan stream
- `Ping(ctx)` - Health check
- `Version(ctx)` - Version string
- `Stats(ctx)` - Statistics
- `Reset()` - Clear call counters

## Running Tests

### All Security Tests
```bash
go test -v ./tests/security/... -short
```

### Specific Test
```bash
go test -v ./tests/security/... -short -run TestUpload_RejectsMalware
```

### With Coverage
```bash
go test -coverprofile=coverage.out ./tests/security/... -short
go tool cover -html=coverage.out
```

### Integration Tests (requires ClamAV)
```bash
# Start ClamAV via Docker
docker-compose -f docker/docker-compose.yml up -d clamav

# Run without -short flag
go test -v ./tests/security/...
```

## Security Testing Principles

1. **Defense in Depth**: Multiple validation layers (MIME, magic bytes, malware scan, re-encoding)
2. **Fail Secure**: Default to rejection on ambiguous input
3. **Input Validation**: Validate all user-controlled input
4. **Output Encoding**: Sanitize filenames before storage
5. **Resource Limits**: Prevent DoS via size/dimension/pixel limits
6. **Non-Guessable IDs**: Use UUIDs, not sequential integers
7. **Path Traversal Prevention**: Validate all file paths and storage keys

## Threat Model

### Prevented Attacks

| Attack Type | Prevention | Test Coverage |
|-------------|-----------|---------------|
| Malware Upload | ClamAV scanning | `TestUpload_RejectsMalware` |
| DoS (Large Files) | Size limits (10MB) | `TestUpload_RejectsOversizedFile` |
| DoS (Decompression Bomb) | Dimension/pixel limits | `TestUpload_EnforcesDimensionLimits`, `TestUpload_EnforcesPixelCountLimit` |
| Path Traversal | Key validation, filename sanitization | `TestStorage_PreventsPathTraversal`, `TestUpload_SanitizesFilename` |
| File Type Confusion | MIME by content, magic bytes | `TestUpload_ValidatesMIMEByContent`, `TestUpload_ValidatesMagicBytes` |
| Polyglot Files | Multi-layer validation | `TestUpload_RejectsPolyglotFile` |
| Enumeration | UUID-based keys | `TestStorage_GeneratesNonGuessableKeys` |
| Injection | Key format validation | `TestStorage_ValidatesKeyFormat` |

## Validation Pipeline

The 7-step validation pipeline (implemented in `validator.Validate()`):

1. **Size Check** → `ErrFileTooLarge`
2. **MIME Sniffing** → `ErrInvalidMimeType`
3. **Magic Byte Validation** → `ErrInvalidMimeType`
4. **Dimension Check** (post-decode) → `ErrImageTooLarge`
5. **Pixel Count Check** (post-decode) → `ErrImageTooManyPixels`
6. **ClamAV Malware Scan** → `ErrMalwareDetected`
7. **Filename Sanitization** → Safe filename

## Configuration

### Default Limits (validator.DefaultConfig())

```go
MaxFileSize:      10 * 1024 * 1024  // 10MB
MaxWidth:         8192                // pixels
MaxHeight:        8192                // pixels
MaxPixels:        100_000_000         // 100M pixels
AllowedMIMETypes: ["image/jpeg", "image/png", "image/gif", "image/webp"]
EnableMalwareScan: true
```

### Key Format

```
images/{owner-uuid}/{image-uuid}/{variant}.{ext}
```

Example: `images/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/thumbnail.jpg`

## Related Documentation

- Validator Implementation: `/internal/infrastructure/storage/validator/validator.go`
- Key Generation: `/internal/infrastructure/storage/keys.go`
- ClamAV Client: `/internal/infrastructure/security/clamav/scanner.go`
- Domain Errors: `/internal/domain/gallery/errors.go`
- Storage Errors: `/internal/infrastructure/storage/errors.go`

## Continuous Integration

These tests run automatically in GitHub Actions:

```yaml
- name: Security Tests
  run: go test -v ./tests/security/... -short
```

**Note**: Full integration tests (with real ClamAV) run in dedicated CI job with Docker services.

## Future Enhancements

1. **ML-based Content Moderation**: Detect NSFW/inappropriate content
2. **Advanced Polyglot Detection**: AI-based multi-format detection
3. **Rate Limiting Tests**: Upload rate limit validation
4. **Fuzzing**: Automated fuzz testing for validator
5. **Performance Tests**: Benchmark validation pipeline throughput
6. **Security Benchmarks**: OWASP dependency check, gosec integration

## Security Reporting

For security vulnerabilities, contact: security@goimg.example.com

**DO NOT** open public GitHub issues for security bugs.
