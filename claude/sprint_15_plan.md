# Sprint 15 Implementation Plan: AI NSFW Detection + Advanced Search

> **Created**: 2026-01-08
> **Status**: IN PROGRESS
> **Duration**: 2 weeks
> **Priority**: P1 HIGH (User safety and experience)

## Executive Summary

Sprint 15 extends the moderation capabilities with AI-powered NSFW content detection and enhances the search functionality with advanced filters. This plan follows the established DDD architecture patterns.

---

## Part 1: AI NSFW Detection

### 1.1 Domain Layer Changes

#### New Files to Create

| File | Purpose |
|------|---------|
| `internal/domain/moderation/nsfw_scan.go` | NSFWScan entity |
| `internal/domain/moderation/nsfw_category.go` | Category value object |
| `internal/domain/moderation/nsfw_details.go` | Provider-specific details |
| `internal/domain/moderation/nsfw_scan_id.go` | ID value object |
| `internal/domain/moderation/nsfw_provider.go` | Provider type enum |
| `internal/domain/moderation/nsfw_scan_status.go` | Status enum |

#### NSFWScan Entity Structure

```go
type NSFWScan struct {
    id          NSFWScanID
    imageID     gallery.ImageID
    provider    NSFWProvider        // sightengine, moderatecontent
    rawScore    float64             // 0.0-1.0
    category    NSFWCategory        // safe, suggestive, explicit, etc.
    isNSFW      bool                // Final determination
    details     NSFWDetails         // Provider-specific details
    scanStatus  NSFWScanStatus      // pending, completed, failed
    scannedAt   *time.Time
    createdAt   time.Time
    events      []shared.DomainEvent
}
```

#### NSFWCategory Value Object

```go
type NSFWCategory string

const (
    NSFWCategorySafe       NSFWCategory = "safe"
    NSFWCategorySuggestive NSFWCategory = "suggestive"
    NSFWCategoryNudity     NSFWCategory = "nudity"
    NSFWCategoryExplicit   NSFWCategory = "explicit"
    NSFWCategoryViolence   NSFWCategory = "violence"
    NSFWCategoryUnknown    NSFWCategory = "unknown"
)
```

#### Repository Interface Extension

Add to `internal/domain/moderation/repository.go`:

```go
type NSFWScanRepository interface {
    NextID() NSFWScanID
    FindByID(ctx context.Context, id NSFWScanID) (*NSFWScan, error)
    FindByImageID(ctx context.Context, imageID gallery.ImageID) (*NSFWScan, error)
    FindPendingScans(ctx context.Context, pagination shared.Pagination) ([]*NSFWScan, int64, error)
    FindByStatus(ctx context.Context, status NSFWScanStatus, pagination shared.Pagination) ([]*NSFWScan, int64, error)
    Save(ctx context.Context, scan *NSFWScan) error
}
```

#### Domain Events

Add to `internal/domain/moderation/events.go`:

```go
type ImageNSFWScanRequested struct {
    shared.BaseEvent
    ImageID   gallery.ImageID
    Provider  NSFWProvider
}

type ImageNSFWScanCompleted struct {
    shared.BaseEvent
    ImageID   gallery.ImageID
    ScanID    NSFWScanID
    IsNSFW    bool
    Category  NSFWCategory
    Score     float64
}

type ImageFlaggedAsNSFW struct {
    shared.BaseEvent
    ImageID   gallery.ImageID
    Category  NSFWCategory
    Score     float64
}
```

### 1.2 Infrastructure Layer Changes

#### NSFW Detection API Client

Create `internal/infrastructure/security/nsfw/` directory:

| File | Purpose |
|------|---------|
| `config.go` | API configuration |
| `client.go` | NSFWClient implementation |
| `sightengine.go` | SightEngine-specific logic |
| `moderatecontent.go` | ModerateContent fallback |
| `metrics.go` | NSFWMetricsRecorder interface |

#### Config Structure

```go
type Config struct {
    Provider     string        // "sightengine" or "moderatecontent"
    APIKey       string
    APISecret    string        // SightEngine requires both
    BaseURL      string
    Timeout      time.Duration
    FailOpen     bool          // On API failure, treat as safe?
    Threshold    float64       // Score threshold for NSFW (e.g., 0.7)
}
```

#### Background Job

Create `internal/infrastructure/jobs/tasks/nsfw_scan.go`:

```go
const TypeNSFWScan = "image:nsfw_scan"

type NSFWScanPayload struct {
    ImageID          string    `json:"image_id"`
    StorageKey       string    `json:"storage_key"`
    OwnerID          string    `json:"owner_id"`
    EnqueuedAt       time.Time `json:"enqueued_at"`
}
```

### 1.3 Application Layer Changes

#### Commands

| File | Purpose |
|------|---------|
| `commands/trigger_nsfw_scan.go` | Queue NSFW scan job |
| `commands/process_nsfw_result.go` | Process completed scan |
| `commands/admin_review_nsfw.go` | Admin review actions |

#### Queries

| File | Purpose |
|------|---------|
| `queries/get_nsfw_scan.go` | Get scan result for image |
| `queries/list_flagged_images.go` | List NSFW flagged images |

### 1.4 HTTP Layer Changes

Add to `internal/interfaces/http/handlers/moderation_handler.go`:

```go
// POST /api/v1/admin/images/{imageID}/nsfw-scan - Trigger manual scan
// GET /api/v1/admin/images/{imageID}/nsfw-status - Get NSFW scan result
// POST /api/v1/admin/images/{imageID}/nsfw-review - Admin review flagged content
// GET /api/v1/admin/images/flagged - List flagged images (with filters)
```

### 1.5 Database Migration

Create `migrations/00014_create_nsfw_scans.sql`:

```sql
-- +goose Up
CREATE TABLE nsfw_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    provider VARCHAR(30) NOT NULL,
    raw_score DECIMAL(4,3) NOT NULL,
    category VARCHAR(20) NOT NULL,
    is_nsfw BOOLEAN NOT NULL DEFAULT FALSE,
    nudity_score DECIMAL(4,3),
    weapon_score DECIMAL(4,3),
    violence_score DECIMAL(4,3),
    offensive_score DECIMAL(4,3),
    subcategories TEXT[],
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    scanned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT nsfw_scans_provider_check CHECK (provider IN ('sightengine', 'moderatecontent')),
    CONSTRAINT nsfw_scans_category_check CHECK (category IN ('safe', 'suggestive', 'nudity', 'explicit', 'violence', 'unknown')),
    CONSTRAINT nsfw_scans_status_check CHECK (status IN ('pending', 'scanning', 'completed', 'failed'))
);

-- Indexes
CREATE INDEX idx_nsfw_scans_image_id ON nsfw_scans(image_id);
CREATE INDEX idx_nsfw_scans_status ON nsfw_scans(status);
CREATE INDEX idx_nsfw_scans_is_nsfw ON nsfw_scans(is_nsfw) WHERE is_nsfw = TRUE;

-- Add nsfw_status to images
ALTER TABLE images ADD COLUMN nsfw_status VARCHAR(20) DEFAULT 'unknown';
CREATE INDEX idx_images_nsfw_status ON images(nsfw_status) WHERE nsfw_status != 'safe';

-- +goose Down
ALTER TABLE images DROP COLUMN IF EXISTS nsfw_status;
DROP TABLE IF EXISTS nsfw_scans;
```

---

## Part 2: Advanced Search Filters

### 2.1 Domain Layer Changes

Extend `SearchParams` in `internal/domain/gallery/repository.go`:

```go
type SearchParams struct {
    // Existing fields
    Query      string
    Tags       []Tag
    OwnerID    *identity.UserID
    Visibility *Visibility
    SortBy     SearchSortBy
    Pagination shared.Pagination

    // NEW: Date range filters
    CreatedAfter  *time.Time
    CreatedBefore *time.Time

    // NEW: File size filters (in bytes)
    MinFileSize *int64
    MaxFileSize *int64

    // NEW: Dimension filters (in pixels)
    MinWidth    *int
    MaxWidth    *int
    MinHeight   *int
    MaxHeight   *int

    // NEW: NSFW status filter
    NSFWStatus *NSFWStatus
}
```

### 2.2 Application Layer Changes

Extend `SearchImagesQuery` in `internal/application/gallery/queries/search_images.go`:

```go
type SearchImagesQuery struct {
    // Existing fields...

    // NEW: Advanced filters
    CreatedAfter  *time.Time
    CreatedBefore *time.Time
    MinFileSize   *int64
    MaxFileSize   *int64
    MinWidth      *int
    MaxWidth      *int
    MinHeight     *int
    MaxHeight     *int
    NSFWStatus    *string
}
```

### 2.3 Infrastructure Layer Changes

Update `internal/infrastructure/persistence/postgres/image_repository.go` to extend `buildSearchQuery` method.

### 2.4 Database Migration for Performance

Create `migrations/00015_add_search_indexes.sql`:

```sql
-- +goose Up
CREATE INDEX idx_images_created_range
    ON images(created_at DESC, status, visibility)
    WHERE deleted_at IS NULL AND status = 'active';

CREATE INDEX idx_images_file_size
    ON images(file_size)
    WHERE deleted_at IS NULL AND status = 'active';

CREATE INDEX idx_images_dimensions
    ON images(width, height)
    WHERE deleted_at IS NULL AND status = 'active';

-- +goose Down
DROP INDEX IF EXISTS idx_images_dimensions;
DROP INDEX IF EXISTS idx_images_file_size;
DROP INDEX IF EXISTS idx_images_created_range;
```

### 2.5 HTTP Layer Changes

Update `internal/interfaces/http/handlers/image_handler.go` to parse new query parameters.

---

## Part 3: OpenAPI Specification Changes

### New Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/images/{imageID}/nsfw-scan` | POST | Trigger NSFW scan |
| `/admin/images/{imageID}/nsfw-status` | GET | Get NSFW scan result |
| `/admin/images/{imageID}/nsfw-review` | POST | Admin review action |
| `/admin/images/flagged` | GET | List flagged images |

### New Search Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `created_after` | datetime | Filter by creation date (start) |
| `created_before` | datetime | Filter by creation date (end) |
| `min_size` | int64 | Minimum file size in bytes |
| `max_size` | int64 | Maximum file size in bytes |
| `min_width` | int | Minimum width in pixels |
| `max_width` | int | Maximum width in pixels |
| `min_height` | int | Minimum height in pixels |
| `max_height` | int | Maximum height in pixels |
| `nsfw_status` | string | Filter by NSFW status |

### New Schemas

- `NSFWScanResponse`
- `NSFWScanResult`
- `FlaggedImagesResponse`
- `FlaggedImageDTO`

---

## Part 4: Test Strategy

### Unit Tests

| Layer | Target Coverage | Focus Areas |
|-------|-----------------|-------------|
| Domain (moderation/nsfw*) | 90% | Value objects, entity behavior, events |
| Application (commands/queries) | 85% | Business logic, validation, error handling |
| Infrastructure (nsfw client) | 70% | API response parsing, error scenarios |

### E2E Tests (Newman/Postman)

1. **NSFW Scan Flow**: Trigger scan → Poll completion → Verify result
2. **Advanced Search Tests**: Date/size/dimension filtering
3. **Admin Review Flow**: List flagged → Review → Approve/Reject

---

## Part 5: Implementation Sequence

| Phase | Days | Tasks |
|-------|------|-------|
| 1. Foundation | 1-2 | Domain types, migration, repository |
| 2. Infrastructure | 3-4 | NSFW API client, background job, metrics |
| 3. Application | 5 | Commands and queries |
| 4. HTTP | 6 | Endpoints, OpenAPI spec |
| 5. Search | 7-8 | SearchParams extension, indexes |
| 6. Testing | 9-10 | Unit, integration, E2E tests |

---

## Security Considerations

| Concern | Mitigation |
|---------|------------|
| API key exposure | Store in encrypted env vars |
| False positives | Human review queue for flagged content |
| Rate limiting | Queue NSFW scans, don't block upload |
| PII in logs | Don't log image content or API responses |

---

## Critical Files

1. `internal/infrastructure/security/hibp_client.go` - Pattern for API client
2. `internal/infrastructure/jobs/tasks/image_scan.go` - Pattern for background job
3. `internal/infrastructure/persistence/postgres/image_repository.go` - Search query extension
4. `internal/domain/moderation/repository.go` - Interface definitions
5. `internal/application/gallery/queries/search_images.go` - Query handler extension
