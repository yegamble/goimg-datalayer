# goimg-datalayer API Documentation

Welcome to the goimg-datalayer API documentation. This guide helps you get started with the RESTful API for image hosting, user management, and content moderation.

## Overview

The goimg-datalayer API is a self-hosted image gallery backend inspired by Flickr and Chevereto. It provides:

- User registration and JWT-based authentication
- Image upload with automatic variant generation
- Album organization and tagging
- Social features (likes, comments)
- Content moderation and abuse reporting
- Full-text search across images and albums

**API Version**: 1.0.0
**OpenAPI Specification**: [api/openapi/openapi.yaml](../../api/openapi/openapi.yaml)

## Base URL

| Environment | Base URL |
|-------------|----------|
| Development | `http://localhost:8080/api/v1` |
| Production | `https://api.goimg.com/v1` (example - replace with your domain) |

All API requests use the `/api/v1` prefix. Future versions will use `/api/v2`, `/api/v3`, etc.

## Authentication

The API uses **JWT (JSON Web Tokens)** for authentication with RS256 asymmetric signing.

### Token Types

| Token Type | Lifetime | Purpose |
|------------|----------|---------|
| **Access Token** | 15 minutes | Authenticate API requests |
| **Refresh Token** | 7 days | Obtain new access tokens |

### Authentication Flow

1. **Register** a new account: `POST /auth/register`
2. **Login** to receive tokens: `POST /auth/login`
3. **Use access token** in requests: `Authorization: Bearer <access_token>`
4. **Refresh** when access token expires: `POST /auth/refresh`
5. **Logout** to invalidate refresh token: `POST /auth/logout`

### Example: Registration and Login

```bash
# 1. Register a new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "username": "johndoe",
    "password": "MySecureP@ssw0rd123"
  }'

# Response (201 Created):
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "username": "johndoe",
  "created_at": "2025-12-05T10:30:00Z"
}

# 2. Login to get tokens
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "MySecureP@ssw0rd123"
  }'

# Response (200 OK):
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### Using Access Tokens

Include the access token in the `Authorization` header for all protected endpoints:

```bash
curl -X GET http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Refreshing Tokens

When your access token expires (15 minutes), use the refresh token to get a new pair:

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

**Important**: Refresh token rotation is enabled. Each refresh invalidates the old refresh token and issues a new one. Reusing an old refresh token triggers replay detection and revokes all tokens in the family.

## Rate Limiting

The API enforces rate limits to prevent abuse:

| Scope | Limit | Window | Applies To |
|-------|-------|--------|------------|
| **Login attempts** | 5 requests | 1 minute | Per IP address |
| **Global requests** | 100 requests | 1 minute | Per IP address (unauthenticated) |
| **Authenticated requests** | 300 requests | 1 minute | Per user (authenticated) |
| **Image uploads** | 50 uploads | 1 hour | Per user |

### Rate Limit Headers

All responses include rate limit information:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1638360000
```

### Rate Limit Exceeded (429)

When you exceed a rate limit, you receive a `429 Too Many Requests` response:

```json
{
  "type": "https://api.goimg.com/problems/rate-limit-exceeded",
  "title": "Too Many Requests",
  "status": 429,
  "detail": "Rate limit exceeded. Retry after 60 seconds.",
  "traceId": "abc123-def456"
}
```

Wait for the time specified in the `Retry-After` header before making additional requests.

## Error Handling

All errors follow **RFC 7807 Problem Details** format for consistency and machine-readability.

### Error Response Structure

```json
{
  "type": "https://api.goimg.com/problems/validation-error",
  "title": "Validation Error",
  "status": 400,
  "detail": "Request validation failed",
  "traceId": "abc123-def456",
  "errors": [
    {
      "field": "password",
      "message": "password must be at least 12 characters"
    }
  ]
}
```

### Error Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | URI | Machine-readable error type identifier |
| `title` | string | Human-readable error category |
| `status` | integer | HTTP status code (400, 401, 403, 404, etc.) |
| `detail` | string | Specific error message for this occurrence |
| `traceId` | string | Correlation ID for debugging (use this in support requests) |
| `errors` | array | Field-level validation errors (optional) |

### Common Error Codes

| Status | Title | When It Occurs |
|--------|-------|----------------|
| **400** | Bad Request | Invalid request body or parameters |
| **401** | Unauthorized | Missing or invalid access token |
| **403** | Forbidden | Insufficient permissions for resource |
| **404** | Not Found | Resource does not exist |
| **409** | Conflict | Resource already exists (e.g., duplicate email) |
| **422** | Unprocessable Entity | Validation failed (business logic) |
| **429** | Too Many Requests | Rate limit exceeded |
| **500** | Internal Server Error | Server-side error (contact support with traceId) |

### Security Error Messages

To prevent account enumeration, authentication errors use generic messages:

```json
{
  "type": "https://api.goimg.com/problems/unauthorized",
  "title": "Unauthorized",
  "status": 401,
  "detail": "Invalid email or password",
  "traceId": "abc123-def456"
}
```

You will not receive different messages for "user not found" vs "wrong password."

## Common Operations

### Upload an Image

```bash
curl -X POST http://localhost:8080/api/v1/images \
  -H "Authorization: Bearer <access_token>" \
  -F "file=@/path/to/image.jpg" \
  -F "title=Sunset over mountains" \
  -F "description=Beautiful sunset captured in Rocky Mountains" \
  -F "visibility=public"

# Response (201 Created):
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Sunset over mountains",
  "description": "Beautiful sunset captured in Rocky Mountains",
  "visibility": "public",
  "status": "processing",
  "original_filename": "image.jpg",
  "mime_type": "image/jpeg",
  "file_size": 2048576,
  "width": 3840,
  "height": 2160,
  "created_at": "2025-12-05T10:45:00Z",
  "variants": []
}
```

**Image Processing**: Images are processed asynchronously. Check the `status` field:
- `processing`: Image is being scanned and variants are being generated
- `active`: Image is ready, variants are available
- `rejected`: Image failed malware scan or validation

### Create an Album

```bash
curl -X POST http://localhost:8080/api/v1/albums \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Vacation 2025",
    "description": "Photos from summer vacation",
    "visibility": "private"
  }'

# Response (201 Created):
{
  "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Vacation 2025",
  "description": "Photos from summer vacation",
  "visibility": "private",
  "image_count": 0,
  "created_at": "2025-12-05T10:50:00Z"
}
```

### Add Image to Album

```bash
curl -X POST http://localhost:8080/api/v1/albums/f47ac10b-58cc-4372-a567-0e02b2c3d479/images \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7"
  }'

# Response (204 No Content)
```

### Like an Image

```bash
curl -X POST http://localhost:8080/api/v1/images/7c9e6679-7425-40de-944b-e07fc1f90ae7/like \
  -H "Authorization: Bearer <access_token>"

# Response (204 No Content)
```

Likes are idempotent: liking an already-liked image succeeds without error.

### Add a Comment

```bash
curl -X POST http://localhost:8080/api/v1/images/7c9e6679-7425-40de-944b-e07fc1f90ae7/comments \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Amazing photo! Great composition."
  }'

# Response (201 Created):
{
  "id": "b89e3d8a-1234-5678-90ab-cdef12345678",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "content": "Amazing photo! Great composition.",
  "created_at": "2025-12-05T11:00:00Z"
}
```

**HTML Sanitization**: Comment content is sanitized using bluemonday's StrictPolicy to prevent XSS attacks.

### Search Images

```bash
curl -X GET "http://localhost:8080/api/v1/images?q=mountains&visibility=public&limit=20" \
  -H "Authorization: Bearer <access_token>"

# Response (200 OK):
{
  "images": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "title": "Sunset over mountains",
      "description": "Beautiful sunset captured in Rocky Mountains",
      "visibility": "public",
      "status": "active",
      "width": 3840,
      "height": 2160,
      "view_count": 45,
      "like_count": 12,
      "comment_count": 3,
      "created_at": "2025-12-05T10:45:00Z",
      "variants": [
        {
          "variant_type": "thumbnail",
          "url": "https://cdn.example.com/images/thumb_abc123.jpg",
          "width": 150,
          "height": 84
        },
        {
          "variant_type": "medium",
          "url": "https://cdn.example.com/images/medium_abc123.jpg",
          "width": 800,
          "height": 450
        }
      ]
    }
  ],
  "pagination": {
    "total": 42,
    "limit": 20,
    "offset": 0,
    "has_more": true
  }
}
```

## Pagination

List endpoints support pagination using offset-based or cursor-based approaches:

### Offset-Based Pagination

```bash
curl -X GET "http://localhost:8080/api/v1/images?limit=20&offset=40"
```

**Parameters**:
- `limit`: Number of items per page (default: 20, max: 100)
- `offset`: Number of items to skip (default: 0)

### Pagination Response

```json
{
  "images": [...],
  "pagination": {
    "total": 150,
    "limit": 20,
    "offset": 40,
    "has_more": true
  }
}
```

## Image Variants

Uploaded images are automatically processed into multiple sizes:

| Variant | Max Width | Use Case |
|---------|-----------|----------|
| `thumbnail` | 150px | List views, thumbnails |
| `small` | 320px | Mobile devices |
| `medium` | 800px | Desktop previews |
| `large` | 1600px | High-resolution viewing |
| `original` | Unchanged | Full-size download (private visibility only) |

Each variant includes:
- `variant_type`: Size identifier
- `url`: CDN or storage URL
- `width`, `height`: Actual dimensions
- `file_size`: Size in bytes
- `format`: Image format (JPEG, PNG, WebP)

## Security Features

### Image Upload Security

All uploaded images undergo a 7-step validation pipeline:

1. **Size check**: Maximum 10MB
2. **MIME validation**: Magic bytes verification (not extension-based)
3. **Dimension check**: Maximum 8192x8192 pixels
4. **Pixel count check**: Maximum 100 million pixels
5. **Malware scan**: ClamAV signature verification
6. **EXIF stripping**: GPS and sensitive metadata removed
7. **Re-encoding**: Images re-encoded through libvips to prevent polyglot exploits

### Authorization

The API implements role-based access control (RBAC):

| Role | Permissions |
|------|-------------|
| **user** | Upload images, create albums, like/comment, manage own content |
| **moderator** | + Moderate content, delete any image, ban users |
| **admin** | + Manage roles, access admin panel, view audit logs |

**Ownership Validation**: Users can only modify resources they own. IDOR (Insecure Direct Object Reference) attacks are prevented through ownership middleware.

### Security Headers

All responses include security headers:

```
Content-Security-Policy: default-src 'self'
Strict-Transport-Security: max-age=31536000; includeSubDomains
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
```

## API Reference

For complete endpoint documentation, see the OpenAPI specification:

- **OpenAPI 3.0 Spec**: [api/openapi/openapi.yaml](../../api/openapi/openapi.yaml)
- **Interactive Docs**: Use [Swagger UI](https://swagger.io/tools/swagger-ui/) or [Redoc](https://redocly.com/redoc/) to explore the API

### Endpoint Categories

| Category | Endpoints | Description |
|----------|-----------|-------------|
| **Authentication** | `/auth/*` | Register, login, refresh, logout |
| **Users** | `/users/*` | User profiles, sessions |
| **Images** | `/images/*` | Upload, retrieve, update, delete images |
| **Albums** | `/albums/*` | Create and manage albums |
| **Social** | `/images/{id}/like`, `/images/{id}/comments` | Likes, comments |
| **Search** | `/images?q=...` | Full-text search |
| **Moderation** | `/moderation/*` | Report abuse, moderation queue (admin) |
| **Health** | `/health`, `/health/ready` | Service health checks |

## Client Libraries

Currently, client libraries are not available. We recommend using standard HTTP clients:

- **curl**: Command-line testing
- **Postman**: Interactive API exploration
- **Go**: `net/http` package
- **JavaScript**: `fetch` or `axios`
- **Python**: `requests` library

## Support and Feedback

- **Issues**: [GitHub Issues](https://github.com/yegamble/goimg-datalayer/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yegamble/goimg-datalayer/discussions)
- **Security**: See [SECURITY.md](../../SECURITY.md) for vulnerability disclosure

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-12-05 | Initial MVP release |

## Additional Resources

- [Architecture Documentation](../../claude/architecture.md) - System design and DDD patterns
- [API Security Guide](../../claude/api_security.md) - Security controls and best practices
- [Testing Guide](../../claude/testing_ci.md) - API testing strategies
- [Deployment Guide](../../docs/deployment.md) - Production deployment instructions (Sprint 9)

---

**Last Updated**: 2025-12-05 (Sprint 9)

**API Version**: 1.0.0
