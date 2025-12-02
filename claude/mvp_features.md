# MVP Feature Specification

> Feature requirements based on Flickr and Chevereto competitive analysis.
> Load this guide when implementing features to ensure competitive baseline.

## Overview

goimg-datalayer targets the self-hosted image gallery market (Chevereto) while providing features competitive with hosted platforms (Flickr). This document defines the minimum viable feature set.

---

## Feature Comparison: Flickr vs Chevereto vs goimg

| Feature | Flickr | Chevereto | goimg MVP | goimg Phase 2 |
|---------|--------|-----------|-----------|---------------|
| Email/password auth | ✅ | ✅ | ✅ | ✅ |
| OAuth (Google, etc.) | ✅ | ❌ | ❌ | ✅ |
| MFA/2FA | ✅ | ❌ | ❌ | ✅ |
| User profiles | ✅ | ✅ | ✅ | ✅ |
| Follow users | ✅ | ✅ | ❌ | ✅ |
| Activity feed | ✅ | ✅ | ❌ | ✅ |
| Image upload (web) | ✅ | ✅ | ✅ | ✅ |
| Bulk upload | ✅ | ✅ | ✅ | ✅ |
| EXIF preservation | ✅ | ✅ | ✅ | ✅ |
| EXIF stripping option | ✅ | ✅ | ✅ | ✅ |
| Auto-resize variants | ✅ (13 sizes) | ✅ (3 sizes) | ✅ (4 sizes) | ✅ |
| Albums | ✅ | ✅ | ✅ | ✅ |
| Nested albums | ✅ | ✅ | ❌ | ✅ |
| Tags | ✅ | ✅ | ✅ | ✅ |
| Search | ✅ | ✅ | ✅ (basic) | ✅ (advanced) |
| Likes/favorites | ✅ | ✅ | ✅ | ✅ |
| Comments | ✅ | ✅ | ✅ | ✅ |
| Privacy settings | ✅ | ✅ | ✅ | ✅ |
| Embed codes | ✅ | ✅ | ✅ | ✅ |
| Admin moderation | ✅ | ✅ | ✅ | ✅ |
| NSFW detection | ✅ | ✅ (plugin) | ❌ | ✅ |
| Local storage | ❌ | ✅ | ✅ | ✅ |
| S3 storage | ✅ | ✅ | ✅ | ✅ |
| IPFS storage | ❌ | ❌ | ❌ | ✅ |
| API access | ✅ | ✅ | ✅ | ✅ |
| Rate limiting | ✅ | ✅ | ✅ | ✅ |
| Watermarking | ❌ | ✅ | ❌ | ✅ |
| Video support | ✅ | ❌ | ❌ | ❌ |

**goimg differentiator**: IPFS decentralized storage (unique in market)

---

## User Management Features

### MVP (Must Have)

#### User Registration
```yaml
endpoint: POST /api/v1/auth/register
fields:
  - email: required, unique, max 255 chars
  - username: required, unique, 3-32 chars, alphanumeric
  - password: required, 12-128 chars, complexity requirements
validation:
  - Check disposable email providers (block)
  - Check username against reserved list (admin, root, system, etc.)
  - Check password against common password list (top 10k)
response:
  - 201: User created, verification email sent (future)
  - 400: Validation errors (RFC 7807)
  - 409: Email or username already exists
```

#### User Login
```yaml
endpoint: POST /api/v1/auth/login
fields:
  - email: required
  - password: required
security:
  - Rate limit: 5 attempts per minute per IP
  - Account lockout after 5 failed attempts (15 min)
  - Constant-time password comparison
  - Generic error message (no account enumeration)
response:
  - 200: { access_token, refresh_token, expires_in }
  - 401: Invalid credentials
  - 429: Rate limited
```

#### Token Refresh
```yaml
endpoint: POST /api/v1/auth/refresh
fields:
  - refresh_token: required
security:
  - Refresh token rotation (issue new token, invalidate old)
  - Replay attack detection (reuse = revoke entire family)
response:
  - 200: { access_token, refresh_token, expires_in }
  - 401: Invalid or expired token
```

#### User Profile
```yaml
endpoints:
  GET /api/v1/users/{id}: Get user profile
  PUT /api/v1/users/{id}: Update own profile
  DELETE /api/v1/users/{id}: Delete own account
fields:
  - id: UUID, read-only
  - email: read-only (except via special flow)
  - username: read-only (except via special flow)
  - display_name: optional, max 100 chars
  - bio: optional, max 500 chars
  - avatar_url: optional (future)
  - created_at: read-only
computed:
  - image_count: number of public images
  - follower_count: (Phase 2)
  - following_count: (Phase 2)
```

### Phase 2 (Should Have)

- OAuth providers (Google, GitHub)
- Email verification
- Password reset flow
- MFA/TOTP support
- Follow/unfollow users
- Activity notifications
- Email notifications (SMTP)

---

## Image Management Features

### MVP (Must Have)

#### Image Upload
```yaml
endpoint: POST /api/v1/images
method: multipart/form-data
fields:
  - file: required, binary
  - title: optional, max 255 chars
  - description: optional, max 2000 chars
  - visibility: optional, enum [public, private, unlisted], default private
  - album_id: optional, UUID
  - tags: optional, comma-separated, max 20 tags
validation:
  - Max file size: 10MB (configurable)
  - Allowed MIME types: image/jpeg, image/png, image/gif, image/webp
  - MIME detection via magic bytes (not extension)
  - Max dimensions: 8192x8192
  - Max pixels: 100 million (prevent decompression bombs)
processing:
  - ClamAV malware scan (required)
  - EXIF metadata extraction
  - EXIF stripping (optional, default: strip GPS)
  - Generate 4 variants: thumbnail (150px), small (320px), medium (800px), large (1600px)
  - Store original
  - Re-encode through libvips (prevent polyglot exploits)
response:
  - 201: Image created (processing may be async)
  - 400: Validation error
  - 413: File too large
  - 415: Unsupported media type
  - 422: Malware detected
  - 429: Upload rate limit exceeded
rate_limit: 50 uploads per hour per user
```

#### Image Retrieval
```yaml
endpoints:
  GET /api/v1/images/{id}: Get image metadata
  GET /api/v1/images/{id}/download: Download original
  GET /api/v1/images/{id}/variants/{size}: Get variant (thumb, small, medium, large)
response_fields:
  - id: UUID
  - owner_id: UUID
  - owner: { id, username, display_name }
  - title: string
  - description: string
  - visibility: enum
  - mime_type: string
  - file_size: bytes
  - width: pixels
  - height: pixels
  - variants:
    - thumbnail: { url, width, height }
    - small: { url, width, height }
    - medium: { url, width, height }
    - large: { url, width, height }
  - tags: string[]
  - view_count: number
  - like_count: number
  - comment_count: number
  - created_at: ISO 8601
  - updated_at: ISO 8601
```

#### Image Listing
```yaml
endpoint: GET /api/v1/images
query_params:
  - owner_id: filter by user
  - album_id: filter by album
  - visibility: filter by visibility (own images only)
  - tags: filter by tags (comma-separated, AND logic)
  - q: search query (title, description)
  - sort: created_at, view_count, like_count (default: created_at)
  - order: asc, desc (default: desc)
  - page: page number (default: 1)
  - per_page: items per page (default: 20, max: 100)
response:
  - items: Image[]
  - total: number
  - page: number
  - per_page: number
  - total_pages: number
```

#### Image Update/Delete
```yaml
endpoints:
  PUT /api/v1/images/{id}: Update image metadata
  DELETE /api/v1/images/{id}: Delete image
authorization:
  - Owner can update/delete own images
  - Moderator can delete any image
  - Admin can update/delete any image
updatable_fields:
  - title
  - description
  - visibility
  - tags
```

### Image Variants Specification

| Variant | Max Width | Max Height | Format | Quality |
|---------|-----------|------------|--------|---------|
| thumbnail | 150 | 150 | JPEG | 80 |
| small | 320 | 320 | JPEG | 85 |
| medium | 800 | 800 | JPEG | 85 |
| large | 1600 | 1600 | JPEG | 90 |
| original | unchanged | unchanged | original | original |

Aspect ratio preserved. Fit mode: contain (not crop).

### Phase 2 (Should Have)

- IPFS storage integration
- Custom variant sizes
- Watermarking
- Bulk operations (delete, move to album, change visibility)
- EXIF viewer in UI
- Advanced search (date range, size, dimensions)
- View statistics per image

---

## Album Features

### MVP (Must Have)

#### Album CRUD
```yaml
endpoints:
  POST /api/v1/albums: Create album
  GET /api/v1/albums: List user's albums
  GET /api/v1/albums/{id}: Get album with images
  PUT /api/v1/albums/{id}: Update album
  DELETE /api/v1/albums/{id}: Delete album (images not deleted)
fields:
  - id: UUID
  - owner_id: UUID
  - title: required, max 255 chars
  - description: optional, max 2000 chars
  - visibility: enum [public, private, unlisted]
  - cover_image_id: optional, UUID (auto-set to first image if not specified)
  - image_count: computed
  - created_at: ISO 8601
  - updated_at: ISO 8601
```

#### Album Image Management
```yaml
endpoints:
  POST /api/v1/albums/{id}/images: Add images to album
  DELETE /api/v1/albums/{id}/images/{image_id}: Remove image from album
  PUT /api/v1/albums/{id}/images/reorder: Reorder images
request_body (add):
  - image_ids: UUID[] (max 100 per request)
notes:
  - Images can belong to multiple albums
  - Removing from album doesn't delete image
  - Album visibility doesn't override image visibility
```

### Phase 2 (Should Have)

- Nested albums (sub-albums)
- Album sharing with direct link
- Collaborative albums
- Album templates

---

## Tag Features

### MVP (Must Have)

```yaml
constraints:
  - Max 20 tags per image
  - Tag name: 2-50 chars, alphanumeric + hyphen + underscore
  - Tags normalized to lowercase
  - Auto-complete suggestions (popular tags)

endpoints:
  GET /api/v1/tags: List popular tags
  GET /api/v1/tags/search?q={query}: Search tags
  GET /api/v1/tags/{tag}/images: List images with tag

response (popular tags):
  - name: string
  - slug: string
  - usage_count: number
```

---

## Social Features

### MVP (Must Have)

#### Likes/Favorites
```yaml
endpoints:
  POST /api/v1/images/{id}/like: Like image
  DELETE /api/v1/images/{id}/like: Unlike image
  GET /api/v1/images/{id}/likes: List users who liked
  GET /api/v1/users/{id}/likes: List images user liked
behavior:
  - Idempotent (liking twice = same as once)
  - like_count incremented/decremented atomically
```

#### Comments
```yaml
endpoints:
  POST /api/v1/images/{id}/comments: Add comment
  GET /api/v1/images/{id}/comments: List comments (paginated)
  DELETE /api/v1/comments/{id}: Delete comment
fields:
  - id: UUID
  - user_id: UUID
  - user: { id, username, display_name }
  - image_id: UUID
  - content: required, 1-1000 chars
  - created_at: ISO 8601
authorization:
  - Owner of comment can delete
  - Image owner can delete any comment on their image
  - Moderator/Admin can delete any comment
```

### Phase 2 (Should Have)

- Follow users
- Activity feed (images from followed users)
- Notifications (new follower, new like, new comment)
- @mentions in comments
- Comment editing (5 min window)

---

## Content Moderation Features

### MVP (Must Have)

#### Content Flags
```yaml
visibility_levels:
  - public: Visible to everyone
  - private: Only visible to owner
  - unlisted: Accessible via direct link, not in search/explore

content_rating:
  - safe: General audience (default)
  - nsfw: Adult content (must be flagged by uploader or moderator)
```

#### Abuse Reporting
```yaml
endpoint: POST /api/v1/reports
fields:
  - image_id: required, UUID
  - reason: required, enum [spam, inappropriate, copyright, harassment, other]
  - description: optional, max 1000 chars
authorization:
  - Any authenticated user can report
  - Cannot report own images
rate_limit: 10 reports per hour per user
```

#### Moderation Queue (Admin/Moderator)
```yaml
endpoints:
  GET /api/v1/moderation/reports: List pending reports
  GET /api/v1/moderation/reports/{id}: Get report details
  POST /api/v1/moderation/reports/{id}/resolve: Resolve report
resolve_actions:
  - dismiss: Report unfounded
  - warn: Warning to image owner
  - remove: Remove image
  - ban: Ban image owner
fields:
  - id: UUID
  - reporter: { id, username }
  - image: { id, owner_id, title, thumbnail_url }
  - reason: enum
  - description: string
  - status: pending, reviewing, resolved, dismissed
  - resolved_by: { id, username }
  - resolved_at: ISO 8601
  - resolution: string
  - created_at: ISO 8601
```

#### User Bans
```yaml
endpoints:
  POST /api/v1/users/{id}/ban: Ban user
  DELETE /api/v1/users/{id}/ban: Unban user
  GET /api/v1/moderation/bans: List bans (admin)
fields:
  - user_id: UUID
  - banned_by: UUID
  - reason: required
  - expires_at: optional (null = permanent)
behavior:
  - Banned users cannot login
  - Banned users' images hidden from public
  - Audit log entry created
```

### Phase 2 (Should Have)

- AI-based NSFW detection (ModerateContent.com or similar)
- Automated spam detection
- User warnings (non-ban)
- Appeal process
- Content removal DMCA workflow

---

## API Features

### MVP (Must Have)

#### Authentication
- JWT Bearer tokens
- Access token: 15 minute TTL
- Refresh token: 7 day TTL
- Token refresh with rotation

#### Rate Limiting
| Scope | Limit | Window |
|-------|-------|--------|
| Global (per IP) | 100 | 1 minute |
| Authenticated | 300 | 1 minute |
| Uploads | 50 | 1 hour |
| Login attempts | 5 | 1 minute |
| Reports | 10 | 1 hour |

Headers returned:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`

#### Error Responses (RFC 7807)
```json
{
  "type": "https://api.goimg.com/problems/validation-error",
  "title": "Validation Error",
  "status": 400,
  "detail": "Request validation failed",
  "instance": "/api/v1/images",
  "traceId": "abc123",
  "errors": [
    { "field": "title", "message": "must not exceed 255 characters" }
  ]
}
```

#### Pagination
```json
{
  "items": [...],
  "pagination": {
    "total": 150,
    "page": 1,
    "per_page": 20,
    "total_pages": 8
  }
}
```

### Phase 2 (Should Have)

- API keys for external apps
- Webhook support
- Batch operations
- GraphQL endpoint (optional)

---

## Sharing & Embedding

### MVP (Must Have)

#### Direct Links
```
https://goimg.com/i/{image_id}           # Image page
https://goimg.com/i/{image_id}/thumb     # Thumbnail
https://goimg.com/i/{image_id}/medium    # Medium variant
```

#### Embed Codes
```html
<!-- HTML -->
<a href="https://goimg.com/i/{id}">
  <img src="https://goimg.com/i/{id}/medium" alt="{title}">
</a>

<!-- BBCode -->
[url=https://goimg.com/i/{id}][img]https://goimg.com/i/{id}/medium[/img][/url]

<!-- Markdown -->
[![{title}](https://goimg.com/i/{id}/medium)](https://goimg.com/i/{id})
```

### Phase 2 (Should Have)

- oEmbed support
- Social media preview cards (Open Graph, Twitter Cards)
- QR codes for images
- Shortened URLs

---

## Storage Architecture

### MVP (Must Have)

#### Primary Storage (choose one)
1. **Local Filesystem** - Development, small deployments
2. **S3-Compatible** - AWS S3, DigitalOcean Spaces, Backblaze B2

#### Storage Interface
```go
type Storage interface {
    Put(ctx context.Context, key string, data []byte, contentType string) error
    Get(ctx context.Context, key string) ([]byte, error)
    Delete(ctx context.Context, key string) error
    URL(key string) string
    Exists(ctx context.Context, key string) (bool, error)
}
```

#### Key Format
```
images/{user_id}/{image_id}/original.{ext}
images/{user_id}/{image_id}/thumbnail.jpg
images/{user_id}/{image_id}/small.jpg
images/{user_id}/{image_id}/medium.jpg
images/{user_id}/{image_id}/large.jpg
```

### Phase 2 (Should Have)

- IPFS integration (content-addressed, decentralized)
- Multi-provider support (primary + backup)
- CDN integration
- Signed URLs for private images

---

## Explore/Discovery

### MVP (Must Have)

```yaml
endpoints:
  GET /api/v1/explore/recent: Recently uploaded public images
  GET /api/v1/explore/popular: Popular images (by views/likes)
  GET /api/v1/explore/tags/{tag}: Images with specific tag
query_params:
  - page: page number
  - per_page: items per page (max 100)
  - period: day, week, month, all (for popular)
```

### Phase 2 (Should Have)

- Personalized feed based on follows
- Trending tags
- Featured/staff picks
- Categories

---

## Health & Monitoring

### MVP (Must Have)

```yaml
endpoints:
  GET /health: Liveness check (is service running?)
  GET /health/ready: Readiness check (can accept traffic?)
response:
  - status: "ok" | "degraded" | "down"
  - checks:
      database: "up" | "down"
      redis: "up" | "down"
      storage: "up" | "down"
  - timestamp: ISO 8601
```

### Metrics (Prometheus)
```
goimg_http_requests_total{method, path, status}
goimg_http_request_duration_seconds{method, path}
goimg_image_uploads_total{status, format}
goimg_image_processing_duration_seconds{operation}
goimg_storage_operations_total{provider, operation, status}
```

---

## Implementation Priority Matrix

### P0 - Critical (Sprint 2-4)
- User registration/login
- JWT authentication
- Image upload with processing
- Basic image retrieval
- Basic search

### P1 - Important (Sprint 5-6)
- Albums
- Tags
- Likes
- Comments
- Admin moderation

### P2 - Nice to Have (Sprint 7+)
- OAuth providers
- Follow users
- Activity feed
- Email notifications
- IPFS storage

### P3 - Future
- MFA
- AI moderation
- Video support
- Groups/communities

---

## Competitive Advantages

1. **Self-hosted**: Full control over data (like Chevereto)
2. **Modern architecture**: DDD, clean code, well-tested
3. **IPFS support**: Decentralized storage (unique differentiator)
4. **Open source**: Community contributions, transparency
5. **API-first**: Extensible, mobile-ready from day 1
6. **Security-focused**: OWASP compliance, ClamAV, rate limiting
