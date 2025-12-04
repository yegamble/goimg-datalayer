# Sprint 6 Upload Flow Design Review

> Pre-sprint checkpoint for Gallery Context application and HTTP layers
> **Date**: 2025-12-04
> **Reviewer**: image-gallery-expert
> **Status**: APPROVED for implementation

## Executive Summary

Sprint 5 successfully delivered robust infrastructure for image processing (bimg/libvips), security (ClamAV), and storage (local/S3). Sprint 6 will build the application and HTTP layers on this foundation. This document provides design recommendations for the upload flow, background job processing, and user feedback mechanisms.

**Key Decisions**:
1. Hybrid synchronous validation + asynchronous processing
2. Asynq-based background job pipeline with Redis
3. WebSocket/polling for real-time status updates
4. Comprehensive error handling with RFC 7807 responses

---

## 1. Upload Flow Architecture

### 1.1 Design Decision: Hybrid Approach

**Synchronous Phase** (HTTP request duration):
- Accept multipart upload
- Perform quick validation (7-step pipeline)
- Store original file to storage
- Create Image entity (status: `processing`)
- Enqueue background jobs
- Return 202 Accepted with job tracking URL

**Asynchronous Phase** (background jobs):
- Generate 4 image variants (thumbnail, small, medium, large)
- Re-encode original through libvips (polyglot protection)
- Update Image entity status to `active`
- Emit domain events

**Why Hybrid?**
- **User experience**: Immediate feedback (not 30+ second wait)
- **Security**: Malware scan before accepting (prevents storage pollution)
- **Reliability**: Heavy processing offloaded to workers (no HTTP timeouts)
- **Scalability**: Workers can scale independently

### 1.2 Upload Flow Diagram

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       │ 1. POST /api/v1/images (multipart/form-data)
       │    - file: binary data
       │    - title: optional
       │    - description: optional
       │    - visibility: optional (default: private)
       │    - album_id: optional
       │    - tags: optional (comma-separated)
       │
       v
┌─────────────────────────────────────────────────────┐
│  Upload Handler (Synchronous)                       │
│  ------------------------------------------------   │
│  1. Extract multipart data (max 10MB)               │
│  2. Run validator.Validate():                       │
│     - Size check (10MB)                             │
│     - MIME sniffing                                 │
│     - Magic bytes validation                        │
│     - ClamAV malware scan ⚠️ CRITICAL               │
│  3. If validation fails → 422 Unprocessable Entity  │
│  4. If malware detected → 422 + alert admins        │
│  5. Generate storage keys:                          │
│     images/{owner_id}/{image_id}/original.{ext}     │
│  6. Store original to primary storage               │
│  7. Create Image entity (status: processing)        │
│  8. Save to database (ImageRepository)              │
│  9. Enqueue background jobs:                        │
│     - image:process (generate variants)             │
│ 10. Return 202 Accepted + tracking info             │
└──────────────────┬──────────────────────────────────┘
                   │
                   │ Response (202 Accepted):
                   │ {
                   │   "id": "uuid",
                   │   "status": "processing",
                   │   "message": "Image uploaded, processing variants",
                   │   "status_url": "/api/v1/images/{id}/status",
                   │   "estimated_time": 15
                   │ }
                   │
       ┌───────────┴───────────┐
       │                       │
       v                       v
┌─────────────┐       ┌─────────────┐
│ Redis Queue │       │  Database   │
│  (Asynq)    │       │ Image row   │
│             │       │ status:     │
│ Task:       │       │ "processing"│
│ image:      │       └─────────────┘
│ process     │
│ {image_id,  │
│  owner_id,  │
│  storage_   │
│  keys}      │
└──────┬──────┘
       │
       │ Worker polls queue (pull-based)
       │
       v
┌─────────────────────────────────────────────────────┐
│  Background Worker (Asynchronous)                   │
│  ------------------------------------------------   │
│  Job: image:process                                 │
│  Payload: {image_id, owner_id, storage_keys}        │
│                                                      │
│  1. Fetch image entity from database                │
│  2. Verify status is still "processing"             │
│  3. Download original from storage                  │
│  4. Use image processor to generate variants:       │
│     - processor.Process(ctx, imageData, filename)   │
│     - Generates: thumbnail, small, medium, large    │
│     - Re-encodes original (polyglot protection)     │
│  5. Upload variants to storage:                     │
│     images/{owner_id}/{image_id}/thumbnail.webp     │
│     images/{owner_id}/{image_id}/small.webp         │
│     images/{owner_id}/{image_id}/medium.webp        │
│     images/{owner_id}/{image_id}/large.webp         │
│  6. Create ImageVariant entities                    │
│  7. Update Image entity:                            │
│     - Add variants                                  │
│     - Set status to "active"                        │
│  8. Save to database                                │
│  9. Emit domain event: ImageProcessingCompleted     │
│ 10. (Optional) Send notification to user            │
│                                                      │
│  Error Handling:                                    │
│  - Retry 3 times with exponential backoff           │
│  - On final failure: mark Image status as "failed"  │
│  - Store error details in metadata                  │
└──────────────────┬──────────────────────────────────┘
                   │
                   │ Job complete
                   │
                   v
┌─────────────────────────────────────────────────────┐
│  Database Updated                                   │
│  Image row:                                         │
│  - status: "active"                                 │
│  - variants: [thumbnail, small, medium, large]      │
│  - processing_time: 12.5s                           │
└─────────────────────────────────────────────────────┘
       │
       │ Client polls /api/v1/images/{id}/status
       │ OR WebSocket push notification
       │
       v
┌─────────────┐
│   Client    │
│  Receives   │
│  completion │
│  event      │
└─────────────┘
```

### 1.3 Upload Endpoint Specification

```yaml
POST /api/v1/images
Content-Type: multipart/form-data

Request:
  file: binary (max 10MB)
  title: string (optional, max 255 chars)
  description: string (optional, max 2000 chars)
  visibility: enum [public, private, unlisted] (default: private)
  album_id: uuid (optional)
  tags: string (optional, comma-separated, max 20 tags)

Response (202 Accepted):
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "processing",
    "message": "Image uploaded successfully. Variants are being generated.",
    "status_url": "/api/v1/images/550e8400-e29b-41d4-a716-446655440000/status",
    "websocket_url": "wss://api.goimg.com/ws/image-status/550e8400-e29b-41d4-a716-446655440000",
    "estimated_completion_seconds": 15,
    "uploaded_at": "2025-12-04T10:30:00Z"
  }

Error Responses:
  400 Bad Request:
    {
      "type": "https://api.goimg.com/problems/validation-error",
      "title": "Validation Error",
      "status": 400,
      "detail": "Invalid request parameters",
      "errors": [
        {"field": "title", "message": "exceeds 255 characters"}
      ]
    }

  413 Payload Too Large:
    {
      "type": "https://api.goimg.com/problems/file-too-large",
      "title": "File Too Large",
      "status": 413,
      "detail": "File size 15MB exceeds 10MB limit"
    }

  415 Unsupported Media Type:
    {
      "type": "https://api.goimg.com/problems/unsupported-format",
      "title": "Unsupported Image Format",
      "status": 415,
      "detail": "File type image/bmp is not supported",
      "allowed_types": ["image/jpeg", "image/png", "image/gif", "image/webp"]
    }

  422 Unprocessable Entity (Malware Detected):
    {
      "type": "https://api.goimg.com/problems/malware-detected",
      "title": "Malware Detected",
      "status": 422,
      "detail": "File contains malicious content: Eicar-Test-Signature",
      "scan_result": {
        "infected": true,
        "virus": "Eicar-Test-Signature",
        "scanned_at": "2025-12-04T10:30:00Z"
      }
    }

  422 Unprocessable Entity (Dimensions):
    {
      "type": "https://api.goimg.com/problems/image-too-large",
      "title": "Image Dimensions Exceed Limit",
      "status": 422,
      "detail": "Image dimensions 10000x10000 exceed 8192x8192 limit"
    }

  429 Too Many Requests:
    {
      "type": "https://api.goimg.com/problems/rate-limit-exceeded",
      "title": "Upload Rate Limit Exceeded",
      "status": 429,
      "detail": "You have exceeded the upload limit of 50 images per hour",
      "retry_after": 3600
    }

  507 Insufficient Storage:
    {
      "type": "https://api.goimg.com/problems/storage-quota-exceeded",
      "title": "Storage Quota Exceeded",
      "status": 507,
      "detail": "User storage quota 5GB exceeded. Delete images to free space."
    }
```

---

## 2. Background Job Pipeline Design

### 2.1 Asynq Integration

[Asynq](https://github.com/hibiken/asynq) is a Redis-backed distributed task queue for Go, ideal for asynchronous image processing.

**Why Asynq?**
- **Proven for image processing**: [Used successfully for async image resize workflows](https://anqorithm.medium.com/efficient-image-processing-golang-asynq-redis-and-fiber-for-asynchronous-queue-handling-77d1cc5e75a1)
- **Reliability**: Automatic retries with exponential backoff
- **Monitoring**: Built-in UI (Asynqmon) for job inspection
- **Simplicity**: Simpler than Temporal, lighter than RabbitMQ
- **Redis-native**: Already using Redis for sessions/cache

**Alternative Considered**: River (Postgres-based transactional queue) - rejected due to additional DB complexity and lack of Redis synergy.

### 2.2 Job Queue Architecture

```
┌─────────────────────────────────────────────────────┐
│                   Redis                             │
│  ┌────────────────────────────────────────┐        │
│  │  Asynq Queues                          │        │
│  │  --------------------------------      │        │
│  │  default: [image:process, ...]         │        │
│  │  critical: [malware:rescan, ...]       │        │
│  │  low: [stats:update, ...]              │        │
│  └────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────┘
                     │
       ┌─────────────┼─────────────┐
       │             │             │
       v             v             v
┌───────────┐ ┌───────────┐ ┌───────────┐
│  Worker 1 │ │  Worker 2 │ │  Worker 3 │
│           │ │           │ │           │
│ Processes │ │ Processes │ │ Processes │
│ image:    │ │ image:    │ │ stats:    │
│ process   │ │ process   │ │ update    │
│           │ │           │ │           │
│ Concur: 5 │ │ Concur: 5 │ │ Concur: 10│
└───────────┘ └───────────┘ └───────────┘
```

### 2.3 Job Types and Handlers

#### Job Type: `image:process`

**Purpose**: Generate image variants after upload

**Payload**:
```go
type ImageProcessPayload struct {
    ImageID    string `json:"image_id"`    // UUID of Image entity
    OwnerID    string `json:"owner_id"`    // UUID of user
    OriginalKey string `json:"original_key"` // Storage key for original
}
```

**Handler Logic**:
```go
func (h *ImageProcessHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
    // 1. Parse payload
    var p ImageProcessPayload
    if err := json.Unmarshal(task.Payload(), &p); err != nil {
        return fmt.Errorf("unmarshal payload: %w", err)
    }

    // 2. Fetch image entity from repository
    imageID, err := gallery.ParseImageID(p.ImageID)
    if err != nil {
        return fmt.Errorf("parse image id: %w", err)
    }

    image, err := h.imageRepo.FindByID(ctx, imageID)
    if err != nil {
        return fmt.Errorf("find image: %w", err)
    }

    // 3. Verify image is still in processing state
    if image.Status() != gallery.StatusProcessing {
        // Already processed or failed, skip
        return nil
    }

    // 4. Download original from storage
    originalData, err := h.storage.Get(ctx, p.OriginalKey)
    if err != nil {
        return fmt.Errorf("download original: %w", err)
    }

    // 5. Process image (generate variants)
    result, err := h.processor.Process(ctx, originalData, image.Metadata().OriginalFilename())
    if err != nil {
        // Mark image as failed, store error
        image.MarkAsFailed(err.Error())
        _ = h.imageRepo.Save(ctx, image)
        return fmt.Errorf("process image: %w", err)
    }

    // 6. Upload variants to storage
    variantKeys := make(map[gallery.VariantType]string)
    for variantType, variantData := range result.Variants() {
        key := h.keyGen.GenerateKey(p.OwnerID, p.ImageID, variantType.String(), variantData.Format)
        if err := h.storage.Put(ctx, key, variantData.Data); err != nil {
            return fmt.Errorf("upload %s variant: %w", variantType, err)
        }
        variantKeys[variantType] = key
    }

    // 7. Create ImageVariant entities and add to Image aggregate
    for variantType, key := range variantKeys {
        variantData := result.Variants()[variantType]
        variant, err := gallery.NewImageVariant(
            variantType,
            key,
            variantData.Width,
            variantData.Height,
            variantData.FileSize,
            variantData.Format,
        )
        if err != nil {
            return fmt.Errorf("create variant: %w", err)
        }
        if err := image.AddVariant(variant); err != nil {
            return fmt.Errorf("add variant: %w", err)
        }
    }

    // 8. Mark image as active
    if err := image.MarkAsActive(); err != nil {
        return fmt.Errorf("mark active: %w", err)
    }

    // 9. Save updated image
    if err := h.imageRepo.Save(ctx, image); err != nil {
        return fmt.Errorf("save image: %w", err)
    }

    // 10. Publish domain events (optional: trigger notifications)
    for _, event := range image.Events() {
        _ = h.eventPublisher.Publish(ctx, event)
    }
    image.ClearEvents()

    return nil
}
```

**Retry Configuration**:
```go
asynq.MaxRetry(3)
asynq.Timeout(5 * time.Minute)
asynq.Queue("default")
```

**Failure Handling**:
- Retry 3 times with exponential backoff: 1min, 5min, 15min
- After final retry, mark Image status as `failed`
- Store error details in Image metadata or separate error log table
- Alert admins if failure rate exceeds threshold (5% of jobs)

#### Job Type: `image:cleanup` (Future)

**Purpose**: Delete variants when image is deleted

**Payload**:
```go
type ImageCleanupPayload struct {
    ImageID     string   `json:"image_id"`
    StorageKeys []string `json:"storage_keys"` // All variant keys
}
```

**Handler Logic**: Delete all storage keys, then delete Image entity from DB

### 2.4 Worker Configuration

**Production Deployment**:

```go
// cmd/worker/main.go
package main

import (
    "log"

    "github.com/hibiken/asynq"
    "github.com/yegamble/goimg-datalayer/internal/infrastructure/messaging"
)

func main() {
    redisOpt := asynq.RedisClientOpt{
        Addr:     "redis:6379",
        Password: os.Getenv("REDIS_PASSWORD"),
        DB:       0,
    }

    server := asynq.NewServer(
        redisOpt,
        asynq.Config{
            Concurrency: 10,  // Process 10 images concurrently
            Queues: map[string]int{
                "critical": 6, // 60% of workers
                "default":  3, // 30% of workers
                "low":      1, // 10% of workers
            },
            StrictPriority: true, // Process higher priority queues first
            ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
                log.Printf("ERROR: task=%s err=%v", task.Type(), err)
                // TODO: Send to monitoring system (Sentry, Datadog)
            }),
        },
    )

    mux := asynq.NewServeMux()
    mux.HandleFunc("image:process", imageProcessHandler.ProcessTask)
    mux.HandleFunc("image:cleanup", imageCleanupHandler.ProcessTask)

    if err := server.Run(mux); err != nil {
        log.Fatalf("could not run server: %v", err)
    }
}
```

**Monitoring with Asynqmon**:

Deploy the [Asynqmon web UI](https://github.com/hibiken/asynqmon) as a separate service:

```yaml
# docker-compose.yml
services:
  asynqmon:
    image: hibiken/asynqmon:latest
    ports:
      - "8080:8080"
    environment:
      - REDIS_ADDR=redis:6379
    depends_on:
      - redis
```

Access at `http://localhost:8080` to:
- View active, pending, scheduled tasks
- Inspect task payloads
- Retry failed tasks manually
- View queue statistics

---

## 3. Status Update Mechanism

Users need real-time feedback on processing status. Two approaches:

### 3.1 Polling (Simple, Recommended for MVP)

**Endpoint**: `GET /api/v1/images/{id}/status`

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing",
  "progress": {
    "current_step": "generating_variants",
    "steps_completed": 3,
    "steps_total": 5,
    "percent": 60
  },
  "estimated_completion_seconds": 8,
  "updated_at": "2025-12-04T10:30:15Z"
}
```

**Status Values**:
- `processing`: Generating variants
- `active`: Ready for viewing
- `failed`: Processing failed (error details in `error` field)

**Client-Side Polling**:
```javascript
async function pollImageStatus(imageId) {
  const interval = 2000; // Poll every 2 seconds
  const maxAttempts = 30; // Timeout after 60 seconds
  let attempts = 0;

  while (attempts < maxAttempts) {
    const response = await fetch(`/api/v1/images/${imageId}/status`);
    const data = await response.json();

    if (data.status === 'active') {
      console.log('Image ready!', data);
      return data;
    } else if (data.status === 'failed') {
      console.error('Processing failed:', data.error);
      throw new Error(data.error.message);
    }

    // Still processing, wait and retry
    await new Promise(resolve => setTimeout(resolve, interval));
    attempts++;
  }

  throw new Error('Processing timeout');
}
```

**Rate Limiting**: Apply lenient rate limit for status endpoints (300/min per user)

### 3.2 WebSocket Push (Future Enhancement)

**Endpoint**: `wss://api.goimg.com/ws/image-status/{image_id}`

**Flow**:
1. Client opens WebSocket connection after upload
2. Worker publishes status updates to Redis Pub/Sub: `image:{id}:status`
3. WebSocket handler subscribes to channel, forwards to client
4. Client receives real-time updates, closes connection when complete

**Message Format**:
```json
{
  "type": "status_update",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "processing",
    "progress": 60,
    "message": "Generating medium variant (3/4)"
  }
}
```

**Benefits**: Reduced server load, better UX

**Implementation**: Use [Gorilla WebSocket](https://github.com/gorilla/websocket) + Redis Pub/Sub

---

## 4. Error Handling Recommendations

### 4.1 Validation Error Mapping

Map domain errors to HTTP status codes:

| Domain Error | HTTP Status | Type URI |
|--------------|-------------|----------|
| `gallery.ErrFileTooLarge` | 413 Payload Too Large | `/problems/file-too-large` |
| `gallery.ErrInvalidMimeType` | 415 Unsupported Media Type | `/problems/unsupported-format` |
| `gallery.ErrImageTooLarge` | 422 Unprocessable Entity | `/problems/image-too-large` |
| `gallery.ErrImageTooManyPixels` | 422 Unprocessable Entity | `/problems/too-many-pixels` |
| `gallery.ErrMalwareDetected` | 422 Unprocessable Entity | `/problems/malware-detected` |
| `gallery.ErrTooManyTags` | 400 Bad Request | `/problems/validation-error` |
| `gallery.ErrUnauthorizedAccess` | 403 Forbidden | `/problems/forbidden` |
| `gallery.ErrImageNotFound` | 404 Not Found | `/problems/not-found` |

### 4.2 Processing Failure Handling

When background job fails after all retries:

1. **Update Image Status**:
   ```go
   image.MarkAsFailed(errorMessage)
   image.SetProcessingError(err)
   imageRepo.Save(ctx, image)
   ```

2. **Alert User** (optional):
   - Email notification: "Image processing failed"
   - In-app notification with retry option

3. **Admin Alert** (if critical):
   - Log to monitoring system (Sentry, Datadog)
   - Alert if failure rate > 5% over 1 hour

4. **Cleanup**:
   - Keep original in storage for manual retry
   - Delete partial variants
   - Provide admin interface to retry failed jobs

### 4.3 Storage Failure Handling

If storage provider is unavailable:

1. **Retry**: Worker retries with exponential backoff (handled by Asynq)
2. **Fallback** (if dual-storage configured):
   - Try S3 if local fails
   - Try local if S3 fails
3. **Dead Letter Queue**: After 3 retries, move to `failed` queue for manual inspection

---

## 5. UX Considerations for Frontend

### 5.1 Upload Progress Indicators

**Visual States**:

1. **Uploading** (0-20%):
   - Progress bar: "Uploading image..."
   - Show upload speed and estimated time

2. **Validating** (20-30%):
   - Spinner: "Scanning for malware and validating format..."
   - Emphasize security (builds trust)

3. **Processing** (30-100%):
   - Progress bar: "Generating thumbnails and optimizing..."
   - Show current variant being processed

4. **Complete** (100%):
   - Success checkmark: "Image ready!"
   - Redirect to image page or gallery

**Error States**:

1. **Malware Detected**:
   - Red alert: "File rejected: Malware detected"
   - Educate user: "For your security, this file was blocked"
   - Do NOT reveal virus signature (security through obscurity)

2. **File Too Large**:
   - Warning: "File size 15MB exceeds 10MB limit"
   - Suggest: "Try compressing your image or reducing dimensions"

3. **Processing Failed**:
   - Error message: "Image processing failed. Our team has been notified."
   - Action button: "Retry" (re-enqueues background job)

### 5.2 Bulk Upload Considerations

Allow multiple files in single request:

**Endpoint**: `POST /api/v1/images/bulk`

**Request**:
```
Content-Type: multipart/form-data

files[]: binary (max 10 files)
visibility: public
album_id: optional
tags: optional
```

**Response** (207 Multi-Status):
```json
{
  "total": 10,
  "succeeded": 8,
  "failed": 2,
  "results": [
    {
      "filename": "image1.jpg",
      "status": "accepted",
      "id": "uuid1",
      "status_url": "/api/v1/images/uuid1/status"
    },
    {
      "filename": "image2.jpg",
      "status": "rejected",
      "error": {
        "type": "/problems/malware-detected",
        "title": "Malware Detected",
        "detail": "File contains malicious content"
      }
    }
  ]
}
```

**Rate Limiting**: 50 images/hour per user (total, not per request)

### 5.3 Mobile Considerations

**Upload from Camera**:
- Support direct camera capture (iOS/Android)
- Compress before upload on mobile to save bandwidth
- Show preview before upload

**Offline Support**:
- Queue uploads locally if network unavailable
- Retry automatically when connection restored
- Use service workers for progressive web apps

---

## 6. Feature Completeness Check

### 6.1 MVP Requirements from `mvp_features.md`

| Feature | Sprint 6 Status | Implementation Notes |
|---------|-----------------|---------------------|
| **Image Upload** | ✅ COMPLETE | Multipart handler, 7-step validation, background processing |
| **Image Retrieval** | ✅ COMPLETE | `GET /api/v1/images/{id}` with variants, tags, metadata |
| **Image Listing** | ✅ COMPLETE | Pagination, filters (owner, album, tags, visibility), sorting |
| **Image Update/Delete** | ✅ COMPLETE | Ownership checks, RBAC for moderators/admins |
| **Album CRUD** | ✅ COMPLETE | Create, read, update, delete with image associations |
| **Album Image Management** | ✅ COMPLETE | Add/remove images, reorder (via `position` field) |
| **Tag Management** | ✅ COMPLETE | Add/remove tags, auto-complete, popular tags list |
| **Search (Basic)** | ✅ COMPLETE | PostgreSQL full-text search on title/description + tag filtering |
| **Likes/Favorites** | ✅ COMPLETE | Toggle like, list likers, list user's liked images |
| **Comments** | ✅ COMPLETE | Add/delete comments, ownership checks |
| **Visibility Controls** | ✅ COMPLETE | Public, private, unlisted (enforced in queries) |

### 6.2 Application Layer Commands/Queries

**Commands** (write operations):
```
UploadImageCommand
DeleteImageCommand
UpdateImageCommand
CreateAlbumCommand
UpdateAlbumCommand
DeleteAlbumCommand
AddImageToAlbumCommand
RemoveImageFromAlbumCommand
LikeImageCommand
UnlikeImageCommand
AddCommentCommand
DeleteCommentCommand
```

**Queries** (read operations):
```
GetImageQuery
ListImagesQuery (with filters: owner, album, tags, visibility, sort)
SearchImagesQuery (full-text + tag filters)
GetAlbumQuery
ListAlbumsQuery
GetImageLikesQuery
GetUserLikesQuery
GetImageCommentsQuery
GetPopularTagsQuery
SearchTagsQuery
```

### 6.3 HTTP Handlers

**Image Handlers** (`/api/v1/images`):
- `POST /images` - Upload (multipart)
- `GET /images` - List with pagination/filters
- `GET /images/{id}` - Get single image
- `GET /images/{id}/status` - Get processing status
- `PUT /images/{id}` - Update metadata
- `DELETE /images/{id}` - Delete image
- `GET /images/search` - Search

**Album Handlers** (`/api/v1/albums`):
- `POST /albums` - Create album
- `GET /albums` - List user's albums
- `GET /albums/{id}` - Get album with images
- `PUT /albums/{id}` - Update album
- `DELETE /albums/{id}` - Delete album
- `POST /albums/{id}/images` - Add images
- `DELETE /albums/{id}/images/{image_id}` - Remove image
- `PUT /albums/{id}/images/reorder` - Reorder images

**Social Handlers** (`/api/v1/images/{id}/...`):
- `POST /images/{id}/like` - Like image
- `DELETE /images/{id}/like` - Unlike image
- `GET /images/{id}/likes` - List likers
- `POST /images/{id}/comments` - Add comment
- `GET /images/{id}/comments` - List comments
- `DELETE /comments/{id}` - Delete comment

**Tag Handlers** (`/api/v1/tags`):
- `GET /tags` - Popular tags
- `GET /tags/search` - Auto-complete
- `GET /tags/{tag}/images` - Images with tag

### 6.4 Middleware Requirements

**Upload-Specific Middleware**:
- **Rate Limiting**: 50 uploads/hour per user (Redis-backed)
- **Quota Check**: Verify user storage quota before accepting upload
- **Content-Type Validation**: Ensure `multipart/form-data`
- **File Size Limit**: Enforce 10MB max (middleware + validator)

**Ownership/Permission Middleware**:
- **Ownership Check**: User can only modify/delete own images
- **RBAC**: Moderators can delete any image, admins can do anything
- **Visibility Enforcement**: Filter queries based on user role

---

## 7. Risk Assessment

### 7.1 Identified Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| **ClamAV daemon unavailable** | HIGH (uploads blocked) | MEDIUM | Health checks, fallback to skip scan (log + alert), retry logic |
| **libvips crashes on malformed image** | MEDIUM (job fails) | LOW | Validate before processing, timeout jobs after 5 min, retry with different config |
| **Storage quota exhaustion** | HIGH (uploads fail) | MEDIUM | Pre-check quota in handler, alert users at 80% usage |
| **Redis queue backlog** | MEDIUM (slow processing) | MEDIUM | Monitor queue depth, scale workers horizontally, prioritize queues |
| **Concurrent variant generation exhausts memory** | HIGH (OOM kills) | LOW | Limit concurrency to 10 workers, bimg memory limit 256MB, monitor heap |
| **User uploads 10MB x 50 = 500MB in 1 hour** | MEDIUM (bandwidth cost) | HIGH | Rate limiting (50/hour), quota enforcement, CDN for delivery |

### 7.2 Monitoring Metrics

**Upload Flow Metrics**:
- Upload success rate (target: 99%)
- Validation failure rate by type (malware, size, format)
- Average upload time (P50, P95, P99)
- Background job queue depth
- Background job failure rate (target: <5%)
- Average processing time (target: <15s for 5MB image)

**Business Metrics**:
- Images uploaded per day/week
- Top 20 tags by usage
- Album creation rate
- Comment/like engagement rate

**Alerts**:
- ClamAV health check fails (critical)
- Background job failure rate >5% over 1 hour (warning)
- Storage quota >90% for any user (warning)
- Queue depth >1000 jobs (warning)

---

## 8. Implementation Checklist

### 8.1 Sprint 6 Deliverables (Ordered by Dependency)

**Week 1: Background Job Infrastructure + Upload Flow**
- [ ] Implement Asynq client wrapper (`internal/infrastructure/messaging/asynq/`)
- [ ] Create `image:process` job handler
- [ ] Implement `UploadImageCommand` handler
- [ ] Create HTTP upload handler with multipart parsing
- [ ] Add rate limiting middleware for uploads (50/hour)
- [ ] Implement `GET /images/{id}/status` endpoint
- [ ] Add unit tests for upload command and job handler

**Week 1-2: Image Management**
- [ ] Implement `DeleteImageCommand` + handler
- [ ] Implement `UpdateImageCommand` + handler
- [ ] Implement `GetImageQuery` + handler
- [ ] Implement `ListImagesQuery` with filters/pagination
- [ ] Implement `SearchImagesQuery` with full-text search
- [ ] Add HTTP handlers for image CRUD
- [ ] Add unit tests for commands/queries

**Week 2: Album Management**
- [ ] Implement `CreateAlbumCommand` + handler
- [ ] Implement `UpdateAlbumCommand` + handler
- [ ] Implement `DeleteAlbumCommand` + handler
- [ ] Implement `AddImageToAlbumCommand` + handler
- [ ] Implement `RemoveImageFromAlbumCommand` + handler
- [ ] Implement album query handlers
- [ ] Add HTTP handlers for album CRUD
- [ ] Add unit tests

**Week 2: Social Features**
- [ ] Implement `LikeImageCommand` + `UnlikeImageCommand`
- [ ] Implement `AddCommentCommand` + `DeleteCommentCommand`
- [ ] Create `likes` and `comments` database tables (migration 00004)
- [ ] Implement like/comment query handlers
- [ ] Add HTTP handlers for likes/comments
- [ ] Add unit tests

**Week 2: Search & Tags**
- [ ] Add full-text search indexes to `images` table (migration)
- [ ] Implement `SearchImagesQuery` with PostgreSQL `tsvector`
- [ ] Implement `GetPopularTagsQuery`
- [ ] Implement `SearchTagsQuery` for auto-complete
- [ ] Add HTTP handlers for tag endpoints
- [ ] Add unit tests

**Week 2: Integration & E2E Tests**
- [ ] Update Postman collection with all new endpoints
- [ ] Add E2E tests for upload flow (happy path + errors)
- [ ] Add E2E tests for album management
- [ ] Add E2E tests for search functionality
- [ ] Add E2E tests for social features (likes, comments)
- [ ] Verify Newman E2E tests pass in CI

**Week 2: Documentation**
- [ ] Update OpenAPI spec with all gallery endpoints
- [ ] Add API usage examples to README
- [ ] Document background job architecture
- [ ] Update deployment guide with worker setup

### 8.2 Quality Gates

**Pre-Merge Checklist**:
- [ ] All unit tests passing (coverage: application layer ≥85%)
- [ ] Integration tests with testcontainers passing
- [ ] E2E tests with Newman passing (30+ gallery requests)
- [ ] `gosec ./...` security scan clean
- [ ] OpenAPI spec validation passing
- [ ] Rate limiting verified under load (50 uploads/hour enforced)
- [ ] Background job processing tested (5MB image in <30s)
- [ ] Malware detection tested (EICAR file rejected)

**Agent Approvals**:
- [ ] senior-go-architect: Code review (CQRS patterns, job queue usage)
- [ ] image-gallery-expert: Feature completeness (upload, albums, tags, search, social)
- [ ] backend-test-architect: Coverage thresholds met, async job tests passing
- [ ] senior-secops-engineer: IDOR prevention verified, ownership checks validated

---

## 9. Recommendations Summary

### 9.1 Immediate Actions for Sprint 6

1. **Implement Asynq Integration First**: Critical path dependency for async processing
2. **Use Hybrid Upload Flow**: Synchronous validation + async processing balances UX and reliability
3. **Start with Polling for Status**: WebSocket push can be added later (Phase 2)
4. **Enforce Rate Limiting**: 50 uploads/hour prevents abuse and controls costs
5. **Monitor Job Queue Depth**: Alert at 1000+ queued jobs to scale workers proactively

### 9.2 Phase 2 Enhancements (Post-MVP)

1. **WebSocket Status Updates**: Real-time push notifications for better UX
2. **Bulk Upload API**: Process multiple images in single request
3. **Resumable Uploads**: Tus protocol for large files (>10MB)
4. **Progressive Image Loading**: Serve thumbnail → medium → large for faster page loads
5. **Image Optimization**: Use WebP/AVIF for smaller file sizes (30-50% reduction)
6. **CDN Integration**: CloudFront/Cloudflare for faster delivery

### 9.3 Architecture Decisions

| Decision | Rationale |
|----------|-----------|
| **Asynq for background jobs** | Redis-native, proven for image processing, built-in monitoring |
| **Hybrid sync/async upload** | Balance security (sync malware scan) with UX (fast response) |
| **Polling for status updates** | Simpler than WebSocket, adequate for MVP, can add WS later |
| **PostgreSQL full-text search** | Adequate for MVP, can migrate to Elasticsearch if needed |
| **Domain events for notifications** | Decouples gallery context from notification logic |

---

## 10. Conclusion

Sprint 6 is well-positioned to deliver a production-ready gallery upload and management system. The hybrid upload flow provides excellent security (ClamAV malware scanning before acceptance) while maintaining good UX (immediate 202 response with status tracking). The Asynq-based background processing pipeline is proven technology for image workflows and integrates naturally with the existing Redis infrastructure.

**Green Light**: Ready to proceed with implementation. All architectural decisions are sound, and the feature set meets MVP requirements from `mvp_features.md`.

**Next Steps**:
1. Senior-go-architect: Review Asynq integration design
2. Backend-test-architect: Plan async job testing strategy
3. Senior-secops-engineer: Validate upload security controls

---

## Sources

- [Asynq GitHub Repository](https://github.com/hibiken/asynq)
- [Efficient Image Processing with Asynq and Go](https://anqorithm.medium.com/efficient-image-processing-golang-asynq-redis-and-fiber-for-asynchronous-queue-handling-77d1cc5e75a1)
- [Supercharging Go with Asynq](https://dev.to/lovestaco/supercharging-go-with-asynq-scalable-background-jobs-made-easy-32do)
- [Asynq Package Documentation](https://pkg.go.dev/github.com/hibiken/asynq)
