# goimg-datalayer API Documentation

Complete API reference for the goimg-datalayer image gallery backend. This self-hosted platform provides user management, image hosting with automatic variant generation, albums, social features, and content moderation.

**API Version**: 1.0.0
**OpenAPI Specification**: [api/openapi/openapi.yaml](../../api/openapi/openapi.yaml)

## Table of Contents

- [Getting Started](#getting-started)
- [Authentication](#authentication)
- [Rate Limiting](#rate-limiting)
- [Error Handling](#error-handling)
- [Endpoints](#endpoints)
  - [Authentication Endpoints](#authentication-endpoints)
  - [User Endpoints](#user-endpoints)
  - [Image Endpoints](#image-endpoints)
  - [Album Endpoints](#album-endpoints)
  - [Tag Endpoints](#tag-endpoints)
  - [Social Endpoints](#social-endpoints)
  - [Moderation Endpoints](#moderation-endpoints)
  - [Explore Endpoints](#explore-endpoints)
  - [Health & Monitoring](#health--monitoring)
- [Code Examples](#code-examples)

---

## Getting Started

### Base URLs

| Environment | Base URL |
|-------------|----------|
| Development | `http://localhost:8080/api/v1` |
| Production | `https://api.goimg.com/v1` (example - replace with your domain) |

All API requests use the `/api/v1` prefix.

### Quick Start

1. **Register** a new account
2. **Login** to receive JWT tokens
3. **Use access token** in the `Authorization` header for authenticated requests
4. **Refresh** when the access token expires (15 minutes)

---

## Authentication

The API uses **JWT (JSON Web Tokens)** for authentication with RS256 asymmetric signing.

### Token Types

| Token Type | Lifetime | Purpose |
|------------|----------|---------|
| **Access Token** | 15 minutes | Authenticate API requests |
| **Refresh Token** | 7 days | Obtain new access tokens |

### Authentication Flow

```
1. POST /auth/register  →  Create account
2. POST /auth/login     →  Get access_token + refresh_token
3. Use access_token in: Authorization: Bearer <token>
4. POST /auth/refresh   →  Get new tokens (when expired)
5. POST /auth/logout    →  Invalidate refresh token
```

### Security Features

- **RS256 signing**: Asymmetric key cryptography
- **Token rotation**: Each refresh invalidates the old refresh token
- **Replay detection**: Reusing an old refresh token revokes the entire token family
- **Account lockout**: 5 failed login attempts locks account for 15 minutes

---

## Rate Limiting

| Scope | Limit | Window | Applies To |
|-------|-------|--------|------------|
| **Login attempts** | 5 requests | 1 minute | Per IP address |
| **Global requests** | 100 requests | 1 minute | Per IP (unauthenticated) |
| **Authenticated requests** | 300 requests | 1 minute | Per user |
| **Image uploads** | 50 uploads | 1 hour | Per user |
| **Abuse reports** | 10 reports | 1 hour | Per user |

### Rate Limit Response Headers

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1733400000
```

### Rate Limit Exceeded (429)

```json
{
  "type": "https://api.goimg.com/problems/rate-limit-exceeded",
  "title": "Too Many Requests",
  "status": 429,
  "detail": "Rate limit exceeded. Retry after 60 seconds.",
  "traceId": "abc123-def456"
}
```

---

## Error Handling

All errors follow **RFC 7807 Problem Details** format.

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

### Common HTTP Status Codes

| Status | Title | When It Occurs |
|--------|-------|----------------|
| **200** | OK | Successful GET/PUT request |
| **201** | Created | Resource created successfully |
| **204** | No Content | Successful DELETE or action with no response body |
| **400** | Bad Request | Invalid request body or parameters |
| **401** | Unauthorized | Missing or invalid access token |
| **403** | Forbidden | Insufficient permissions |
| **404** | Not Found | Resource does not exist |
| **409** | Conflict | Resource already exists (e.g., duplicate email) |
| **413** | Payload Too Large | File exceeds size limit (10MB) |
| **415** | Unsupported Media Type | Invalid file type |
| **422** | Unprocessable Entity | Malware detected or processing failed |
| **429** | Too Many Requests | Rate limit exceeded |
| **500** | Internal Server Error | Server-side error |
| **503** | Service Unavailable | Dependencies unhealthy |

---

## Endpoints

### Authentication Endpoints

#### POST /auth/register

Register a new user account.

**Authentication**: None required

**Request Body**:
```json
{
  "email": "user@example.com",
  "username": "johndoe",
  "password": "MySecureP@ssw0rd123"
}
```

**Validation Rules**:
- Email: Must be unique, valid format, not from disposable provider
- Username: 3-32 chars, alphanumeric + underscore/hyphen, unique
- Password: 12-128 chars with complexity requirements

**Success Response (201 Created)**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "username": "johndoe",
  "created_at": "2025-12-05T10:30:00Z"
}
```

**Error Responses**:
- `400`: Validation error
- `409`: Email or username already exists

---

#### POST /auth/login

Authenticate with email and password to receive JWT tokens.

**Authentication**: None required

**Request Body**:
```json
{
  "email": "user@example.com",
  "password": "MySecureP@ssw0rd123"
}
```

**Success Response (200 OK)**:
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**Error Responses**:
- `401`: Invalid credentials (generic message to prevent enumeration)
- `429`: Rate limit exceeded (5 attempts per minute)

**Security Notes**:
- Account lockout after 5 failed attempts (15 min)
- Generic error messages prevent account enumeration

---

#### POST /auth/refresh

Exchange a refresh token for new access and refresh tokens.

**Authentication**: None required

**Request Body**:
```json
{
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Success Response (200 OK)**:
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**Error Responses**:
- `401`: Invalid or expired refresh token

**Security Notes**:
- Refresh token rotation: old token is invalidated
- Replay detection: reusing old token revokes entire family

---

#### POST /auth/logout

Invalidate the current refresh token.

**Authentication**: Required (Bearer token)

**Request Body**: None

**Success Response (204 No Content)**: Empty response

**Error Responses**:
- `401`: Unauthorized

---

### User Endpoints

#### GET /users/{id}

Retrieve public profile information for a user.

**Authentication**: Optional

**Path Parameters**:
- `id` (UUID): User ID

**Success Response (200 OK)**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "username": "johndoe",
  "display_name": "John Doe",
  "bio": "Photographer and nature enthusiast",
  "avatar_url": "https://cdn.goimg.com/avatars/550e8400.jpg",
  "image_count": 42,
  "created_at": "2025-12-05T10:30:00Z"
}
```

**Error Responses**:
- `404`: User not found

---

#### PUT /users/{id}

Update own user profile.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id` (UUID): User ID (must be own user ID)

**Request Body**:
```json
{
  "display_name": "John Doe",
  "bio": "Photographer and nature enthusiast"
}
```

**Success Response (200 OK)**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "username": "johndoe",
  "display_name": "John Doe",
  "bio": "Photographer and nature enthusiast",
  "avatar_url": null,
  "image_count": 42,
  "created_at": "2025-12-05T10:30:00Z"
}
```

**Error Responses**:
- `400`: Validation error
- `401`: Unauthorized
- `403`: Cannot update another user's profile
- `404`: User not found

---

#### DELETE /users/{id}

Delete own user account permanently.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id` (UUID): User ID (must be own user ID)

**Success Response (204 No Content)**: Empty response

**Error Responses**:
- `401`: Unauthorized
- `403`: Cannot delete another user's account
- `404`: User not found

**Warning**: This action cannot be undone. All user images and albums will be deleted.

---

### Image Endpoints

#### POST /images

Upload a new image with optional metadata.

**Authentication**: Required (Bearer token)

**Request**: `multipart/form-data`

**Form Fields**:
- `file` (binary, required): Image file (JPEG, PNG, GIF, WebP)
- `title` (string, optional): Max 255 chars
- `description` (string, optional): Max 2000 chars
- `visibility` (string, optional): `public`, `private`, or `unlisted` (default: `private`)
- `album_id` (UUID, optional): Add image to album
- `tags` (string, optional): Comma-separated tags (max 20)

**Success Response (201 Created)**:
```json
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "owner": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe",
    "display_name": "John Doe"
  },
  "title": "Sunset over mountains",
  "description": "Beautiful sunset captured in Rocky Mountains",
  "visibility": "public",
  "mime_type": "image/jpeg",
  "file_size": 2048576,
  "width": 4000,
  "height": 3000,
  "variants": {
    "thumbnail": {
      "url": "https://cdn.goimg.com/images/7c9e6679/thumbnail.jpg",
      "width": 150,
      "height": 113
    },
    "small": {
      "url": "https://cdn.goimg.com/images/7c9e6679/small.jpg",
      "width": 320,
      "height": 240
    },
    "medium": {
      "url": "https://cdn.goimg.com/images/7c9e6679/medium.jpg",
      "width": 800,
      "height": 600
    },
    "large": {
      "url": "https://cdn.goimg.com/images/7c9e6679/large.jpg",
      "width": 1600,
      "height": 1200
    },
    "original": {
      "url": "https://cdn.goimg.com/images/7c9e6679/original.jpg",
      "width": 4000,
      "height": 3000
    }
  },
  "tags": ["sunset", "mountains", "nature", "landscape"],
  "view_count": 0,
  "like_count": 0,
  "comment_count": 0,
  "created_at": "2025-12-05T10:45:00Z",
  "updated_at": "2025-12-05T10:45:00Z"
}
```

**Error Responses**:
- `400`: Validation error
- `401`: Unauthorized
- `413`: File too large (max 10MB)
- `415`: Unsupported media type
- `422`: Malware detected or processing failed
- `429`: Upload rate limit exceeded (50/hour)

**Processing Pipeline**:
1. Size check (max 10MB)
2. MIME validation (magic bytes)
3. Dimension check (max 8192x8192px)
4. ClamAV malware scan
5. EXIF metadata extraction
6. EXIF stripping (GPS removed)
7. Variant generation (4 sizes)
8. Re-encoding through libvips

**Available Variants**:
- `thumbnail`: 150x150px (JPEG, quality 80)
- `small`: 320x320px (JPEG, quality 85)
- `medium`: 800x800px (JPEG, quality 85)
- `large`: 1600x1600px (JPEG, quality 90)
- `original`: Unchanged (original format)

---

#### GET /images

List images with optional filtering and pagination.

**Authentication**: Optional

**Query Parameters**:
- `owner_id` (UUID, optional): Filter by image owner
- `album_id` (UUID, optional): Filter by album
- `visibility` (string, optional): `public`, `private`, or `unlisted` (only for own images)
- `tags` (string, optional): Comma-separated tags (AND logic)
- `q` (string, optional): Search query (title, description)
- `sort` (string, optional): `created_at`, `view_count`, `like_count` (default: `created_at`)
- `order` (string, optional): `asc` or `desc` (default: `desc`)
- `page` (integer, optional): Page number (default: 1)
- `per_page` (integer, optional): Items per page (default: 20, max: 100)

**Success Response (200 OK)**:
```json
{
  "items": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "owner_id": "550e8400-e29b-41d4-a716-446655440000",
      "owner": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "johndoe",
        "display_name": "John Doe"
      },
      "title": "Sunset over mountains",
      "visibility": "public",
      "mime_type": "image/jpeg",
      "width": 4000,
      "height": 3000,
      "variants": {
        "thumbnail": {
          "url": "https://cdn.goimg.com/images/7c9e6679/thumbnail.jpg",
          "width": 150,
          "height": 113
        }
      },
      "tags": ["sunset", "mountains"],
      "view_count": 150,
      "like_count": 42,
      "comment_count": 8,
      "created_at": "2025-12-05T10:45:00Z"
    }
  ],
  "pagination": {
    "total": 150,
    "page": 1,
    "per_page": 20,
    "total_pages": 8
  }
}
```

**Error Responses**:
- `400`: Invalid query parameters

**Access Control**:
- Returns only images the user has permission to view
- Own images (all visibility levels)
- Public images from other users

---

#### GET /images/{id}

Retrieve detailed information about a specific image.

**Authentication**: Optional

**Path Parameters**:
- `id` (UUID): Image ID

**Success Response (200 OK)**:
```json
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "owner": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe",
    "display_name": "John Doe"
  },
  "title": "Sunset over mountains",
  "description": "Beautiful sunset captured in Rocky Mountains",
  "visibility": "public",
  "mime_type": "image/jpeg",
  "file_size": 2048576,
  "width": 4000,
  "height": 3000,
  "variants": {
    "thumbnail": {
      "url": "https://cdn.goimg.com/images/7c9e6679/thumbnail.jpg",
      "width": 150,
      "height": 113
    },
    "small": {
      "url": "https://cdn.goimg.com/images/7c9e6679/small.jpg",
      "width": 320,
      "height": 240
    },
    "medium": {
      "url": "https://cdn.goimg.com/images/7c9e6679/medium.jpg",
      "width": 800,
      "height": 600
    },
    "large": {
      "url": "https://cdn.goimg.com/images/7c9e6679/large.jpg",
      "width": 1600,
      "height": 1200
    },
    "original": {
      "url": "https://cdn.goimg.com/images/7c9e6679/original.jpg",
      "width": 4000,
      "height": 3000
    }
  },
  "tags": ["sunset", "mountains", "nature", "landscape"],
  "view_count": 150,
  "like_count": 42,
  "comment_count": 8,
  "created_at": "2025-12-05T10:45:00Z",
  "updated_at": "2025-12-05T11:00:00Z"
}
```

**Error Responses**:
- `404`: Image not found or not accessible

---

#### PUT /images/{id}

Update image metadata.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id` (UUID): Image ID

**Request Body**:
```json
{
  "title": "Sunset over mountains",
  "description": "Beautiful sunset captured in Rocky Mountains",
  "visibility": "public",
  "tags": ["sunset", "mountains", "nature"]
}
```

**Success Response (200 OK)**:
```json
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Sunset over mountains",
  "description": "Beautiful sunset captured in Rocky Mountains",
  "visibility": "public",
  "tags": ["sunset", "mountains", "nature"],
  "updated_at": "2025-12-05T11:00:00Z"
}
```

**Error Responses**:
- `400`: Validation error
- `401`: Unauthorized
- `403`: Cannot update another user's image
- `404`: Image not found

**Access Control**:
- Only the image owner can update their images
- Admins can update any image

---

#### DELETE /images/{id}

Delete an image permanently.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id` (UUID): Image ID

**Success Response (204 No Content)**: Empty response

**Error Responses**:
- `401`: Unauthorized
- `403`: Cannot delete another user's image
- `404`: Image not found

**Access Control**:
- Only the image owner can delete their images
- Moderators and admins can delete any image

---

#### GET /images/{id}/variants/{size}

Retrieve a specific image variant (resized version).

**Authentication**: Optional

**Path Parameters**:
- `id` (UUID): Image ID
- `size` (string): Variant size (`thumbnail`, `small`, `medium`, `large`, `original`)

**Success Response (200 OK)**:
Binary image data (JPEG, PNG, GIF, or WebP)

**Response Headers**:
```
Content-Type: image/jpeg
Content-Length: 52341
Cache-Control: public, max-age=31536000
```

**Error Responses**:
- `404`: Image or variant not found

**Variant Sizes**:
- `thumbnail`: 150x150px (JPEG, quality 80)
- `small`: 320x320px (JPEG, quality 85)
- `medium`: 800x800px (JPEG, quality 85)
- `large`: 1600x1600px (JPEG, quality 90)
- `original`: Unchanged (original format and quality)

**Notes**:
- Aspect ratio is preserved (contain, not crop)
- All variants are cached on CDN

---

### Album Endpoints

#### POST /albums

Create a new album to organize images.

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "title": "Summer Vacation 2024",
  "description": "Photos from our trip to the mountains",
  "visibility": "private",
  "cover_image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7"
}
```

**Fields**:
- `title` (string, required): Max 255 chars
- `description` (string, optional): Max 2000 chars
- `visibility` (string, optional): `public`, `private`, or `unlisted` (default: `private`)
- `cover_image_id` (UUID, optional): Album cover image (defaults to first image)

**Success Response (201 Created)**:
```json
{
  "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Summer Vacation 2024",
  "description": "Photos from our trip to the mountains",
  "visibility": "private",
  "cover_image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "cover_image": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "title": "Sunset over mountains",
    "variants": {
      "thumbnail": {
        "url": "https://cdn.goimg.com/images/7c9e6679/thumbnail.jpg"
      }
    }
  },
  "image_count": 0,
  "created_at": "2025-12-05T11:00:00Z",
  "updated_at": "2025-12-05T11:00:00Z"
}
```

**Error Responses**:
- `400`: Validation error
- `401`: Unauthorized

---

#### GET /albums

List albums for the authenticated user or a specific user.

**Authentication**: Optional

**Query Parameters**:
- `owner_id` (UUID, optional): Filter by album owner
- `visibility` (string, optional): `public`, `private`, or `unlisted` (only for own albums)
- `page` (integer, optional): Page number (default: 1)
- `per_page` (integer, optional): Items per page (default: 20, max: 100)

**Success Response (200 OK)**:
```json
{
  "items": [
    {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "owner_id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Summer Vacation 2024",
      "description": "Photos from our trip to the mountains",
      "visibility": "public",
      "cover_image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "cover_image": {
        "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
        "variants": {
          "thumbnail": {
            "url": "https://cdn.goimg.com/images/7c9e6679/thumbnail.jpg"
          }
        }
      },
      "image_count": 15,
      "created_at": "2025-12-05T11:00:00Z",
      "updated_at": "2025-12-05T12:00:00Z"
    }
  ],
  "pagination": {
    "total": 5,
    "page": 1,
    "per_page": 20,
    "total_pages": 1
  }
}
```

---

#### GET /albums/{id}

Retrieve album details including all images in the album.

**Authentication**: Optional

**Path Parameters**:
- `id` (UUID): Album ID

**Query Parameters**:
- `page` (integer, optional): Page number for images (default: 1)
- `per_page` (integer, optional): Images per page (default: 20, max: 100)

**Success Response (200 OK)**:
```json
{
  "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Summer Vacation 2024",
  "description": "Photos from our trip to the mountains",
  "visibility": "public",
  "cover_image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "cover_image": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "title": "Sunset over mountains",
    "variants": {
      "thumbnail": {
        "url": "https://cdn.goimg.com/images/7c9e6679/thumbnail.jpg"
      }
    }
  },
  "image_count": 15,
  "created_at": "2025-12-05T11:00:00Z",
  "updated_at": "2025-12-05T12:00:00Z",
  "images": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "title": "Sunset over mountains",
      "variants": {
        "thumbnail": {
          "url": "https://cdn.goimg.com/images/7c9e6679/thumbnail.jpg"
        }
      }
    }
  ],
  "pagination": {
    "total": 15,
    "page": 1,
    "per_page": 20,
    "total_pages": 1
  }
}
```

**Error Responses**:
- `404`: Album not found or not accessible

---

#### PUT /albums/{id}

Update album metadata.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id` (UUID): Album ID

**Request Body**:
```json
{
  "title": "Summer Vacation 2024",
  "description": "Updated description",
  "visibility": "public",
  "cover_image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7"
}
```

**Success Response (200 OK)**:
```json
{
  "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Summer Vacation 2024",
  "description": "Updated description",
  "visibility": "public",
  "cover_image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "image_count": 15,
  "updated_at": "2025-12-05T12:30:00Z"
}
```

**Error Responses**:
- `400`: Validation error
- `401`: Unauthorized
- `403`: Cannot update another user's album
- `404`: Album not found

---

#### DELETE /albums/{id}

Delete an album permanently.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id` (UUID): Album ID

**Success Response (204 No Content)**: Empty response

**Error Responses**:
- `401`: Unauthorized
- `403`: Cannot delete another user's album
- `404`: Album not found

**Note**: Images in the album are not deleted, only the album itself.

---

#### POST /albums/{id}/images

Add one or more images to an album.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id` (UUID): Album ID

**Request Body**:
```json
{
  "image_ids": [
    "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "8d0e7789-8536-51ef-a845-f18gd2g01bf8"
  ]
}
```

**Fields**:
- `image_ids` (array of UUIDs, required): Max 100 images

**Success Response (200 OK)**:
```json
{
  "added_count": 2
}
```

**Error Responses**:
- `400`: Validation error
- `401`: Unauthorized
- `403`: Cannot modify another user's album
- `404`: Album not found

**Notes**:
- Images can belong to multiple albums
- Only the album owner can add images

---

#### DELETE /albums/{id}/images/{imageId}

Remove an image from an album.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id` (UUID): Album ID
- `imageId` (UUID): Image ID

**Success Response (204 No Content)**: Empty response

**Error Responses**:
- `401`: Unauthorized
- `403`: Cannot modify another user's album
- `404`: Album or image not found

**Note**: The image itself is not deleted, only removed from the album.

---

### Tag Endpoints

#### GET /tags

Retrieve a list of popular tags with usage counts.

**Authentication**: Optional

**Query Parameters**:
- `limit` (integer, optional): Number of tags to return (default: 50, max: 100)

**Success Response (200 OK)**:
```json
[
  {
    "name": "sunset",
    "slug": "sunset",
    "usage_count": 1234
  },
  {
    "name": "mountains",
    "slug": "mountains",
    "usage_count": 987
  },
  {
    "name": "nature",
    "slug": "nature",
    "usage_count": 856
  }
]
```

---

#### GET /tags/search

Search for tags by prefix (for autocomplete).

**Authentication**: Optional

**Query Parameters**:
- `q` (string, required): Search query (tag prefix), min 2 chars
- `limit` (integer, optional): Number of results (default: 10, max: 50)

**Success Response (200 OK)**:
```json
[
  {
    "name": "sunset",
    "slug": "sunset",
    "usage_count": 1234
  },
  {
    "name": "sunflower",
    "slug": "sunflower",
    "usage_count": 456
  }
]
```

---

#### GET /tags/{tag}/images

Retrieve all public images with a specific tag.

**Authentication**: Optional

**Path Parameters**:
- `tag` (string): Tag name

**Query Parameters**:
- `page` (integer, optional): Page number (default: 1)
- `per_page` (integer, optional): Items per page (default: 20, max: 100)

**Success Response (200 OK)**:
```json
{
  "items": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "title": "Sunset over mountains",
      "visibility": "public",
      "variants": {
        "thumbnail": {
          "url": "https://cdn.goimg.com/images/7c9e6679/thumbnail.jpg"
        }
      },
      "tags": ["sunset", "mountains"],
      "view_count": 150,
      "like_count": 42
    }
  ],
  "pagination": {
    "total": 1234,
    "page": 1,
    "per_page": 20,
    "total_pages": 62
  }
}
```

---

### Social Endpoints

#### POST /images/{id}/like

Like an image.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id` (UUID): Image ID

**Success Response (200 OK)**:
```json
{
  "liked": true,
  "like_count": 43
}
```

**Error Responses**:
- `401`: Unauthorized
- `404`: Image not found

**Note**: This operation is idempotent. Liking an already-liked image succeeds without error.

---

#### DELETE /images/{id}/like

Remove like from an image.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id` (UUID): Image ID

**Success Response (200 OK)**:
```json
{
  "liked": false,
  "like_count": 42
}
```

**Error Responses**:
- `401`: Unauthorized
- `404`: Image not found

---

#### GET /images/{id}/likes

Retrieve a list of users who liked this image.

**Authentication**: Optional

**Path Parameters**:
- `id` (UUID): Image ID

**Query Parameters**:
- `page` (integer, optional): Page number (default: 1)
- `per_page` (integer, optional): Items per page (default: 20, max: 100)

**Success Response (200 OK)**:
```json
{
  "items": [
    {
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "johndoe",
        "display_name": "John Doe"
      },
      "image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "created_at": "2025-12-05T10:30:00Z"
    }
  ],
  "pagination": {
    "total": 42,
    "page": 1,
    "per_page": 20,
    "total_pages": 3
  }
}
```

---

#### POST /images/{id}/comments

Add a comment to an image.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id` (UUID): Image ID

**Request Body**:
```json
{
  "content": "Beautiful photo! Love the colors."
}
```

**Fields**:
- `content` (string, required): 1-1000 chars

**Success Response (201 Created)**:
```json
{
  "id": "b89e3d8a-1234-5678-90ab-cdef12345678",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe",
    "display_name": "John Doe"
  },
  "image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "content": "Beautiful photo! Love the colors.",
  "created_at": "2025-12-05T11:00:00Z"
}
```

**Error Responses**:
- `400`: Validation error
- `401`: Unauthorized
- `404`: Image not found

**Security**: Comment content is sanitized using bluemonday's StrictPolicy to prevent XSS.

---

#### GET /images/{id}/comments

Retrieve all comments for an image.

**Authentication**: Optional

**Path Parameters**:
- `id` (UUID): Image ID

**Query Parameters**:
- `page` (integer, optional): Page number (default: 1)
- `per_page` (integer, optional): Items per page (default: 20, max: 100)

**Success Response (200 OK)**:
```json
{
  "items": [
    {
      "id": "b89e3d8a-1234-5678-90ab-cdef12345678",
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "johndoe",
        "display_name": "John Doe"
      },
      "image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "content": "Beautiful photo! Love the colors.",
      "created_at": "2025-12-05T11:00:00Z"
    }
  ],
  "pagination": {
    "total": 8,
    "page": 1,
    "per_page": 20,
    "total_pages": 1
  }
}
```

---

#### DELETE /comments/{id}

Delete a comment.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id` (UUID): Comment ID

**Success Response (204 No Content)**: Empty response

**Error Responses**:
- `401`: Unauthorized
- `403`: Cannot delete this comment
- `404`: Comment not found

**Access Control**:
- Comment author can delete their own comment
- Image owner can delete any comment on their image
- Moderator/Admin can delete any comment

---

### Moderation Endpoints

#### POST /reports

Report an image for abuse, spam, copyright violation, etc.

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "reason": "inappropriate",
  "description": "This image contains inappropriate content"
}
```

**Fields**:
- `image_id` (UUID, required): Image to report
- `reason` (string, required): `spam`, `inappropriate`, `copyright`, `harassment`, or `other`
- `description` (string, optional): Max 1000 chars

**Success Response (201 Created)**:
```json
{
  "id": "c90f4e9b-2345-6789-01bc-defg34567890",
  "reporter_id": "550e8400-e29b-41d4-a716-446655440000",
  "reporter": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe"
  },
  "image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "image": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "owner_id": "8e1f5g0c-3456-7890-12cd-efgh45678901",
    "title": "Inappropriate image",
    "thumbnail_url": "https://cdn.goimg.com/images/7c9e6679/thumbnail.jpg"
  },
  "reason": "inappropriate",
  "description": "This image contains inappropriate content",
  "status": "pending",
  "created_at": "2025-12-05T11:30:00Z"
}
```

**Error Responses**:
- `400`: Validation error
- `401`: Unauthorized
- `403`: Cannot report own images
- `429`: Rate limit exceeded (10 reports/hour)

---

#### GET /moderation/reports

List all reports (admin/moderator only).

**Authentication**: Required (Bearer token) - Admin or Moderator role

**Query Parameters**:
- `status` (string, optional): `pending`, `reviewing`, `resolved`, or `dismissed`
- `page` (integer, optional): Page number (default: 1)
- `per_page` (integer, optional): Items per page (default: 20, max: 100)

**Success Response (200 OK)**:
```json
{
  "items": [
    {
      "id": "c90f4e9b-2345-6789-01bc-defg34567890",
      "reporter_id": "550e8400-e29b-41d4-a716-446655440000",
      "reporter": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "johndoe"
      },
      "image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "image": {
        "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
        "owner_id": "8e1f5g0c-3456-7890-12cd-efgh45678901",
        "title": "Inappropriate image",
        "thumbnail_url": "https://cdn.goimg.com/images/7c9e6679/thumbnail.jpg"
      },
      "reason": "inappropriate",
      "description": "This image contains inappropriate content",
      "status": "pending",
      "created_at": "2025-12-05T11:30:00Z"
    }
  ],
  "pagination": {
    "total": 25,
    "page": 1,
    "per_page": 20,
    "total_pages": 2
  }
}
```

**Error Responses**:
- `401`: Unauthorized
- `403`: Requires admin or moderator role

---

#### GET /moderation/reports/{id}

Retrieve detailed information about a specific report (admin/moderator only).

**Authentication**: Required (Bearer token) - Admin or Moderator role

**Path Parameters**:
- `id` (UUID): Report ID

**Success Response (200 OK)**:
```json
{
  "id": "c90f4e9b-2345-6789-01bc-defg34567890",
  "reporter_id": "550e8400-e29b-41d4-a716-446655440000",
  "reporter": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe"
  },
  "image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "image": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "owner_id": "8e1f5g0c-3456-7890-12cd-efgh45678901",
    "title": "Inappropriate image",
    "thumbnail_url": "https://cdn.goimg.com/images/7c9e6679/thumbnail.jpg"
  },
  "reason": "inappropriate",
  "description": "This image contains inappropriate content",
  "status": "resolved",
  "resolved_by": {
    "id": "9f2g6h1d-4567-8901-23de-fghi56789012",
    "username": "moderator1"
  },
  "resolved_at": "2025-12-05T12:00:00Z",
  "resolution": "Image removed for violating content policy",
  "created_at": "2025-12-05T11:30:00Z"
}
```

**Error Responses**:
- `401`: Unauthorized
- `403`: Requires admin or moderator role
- `404`: Report not found

---

#### POST /moderation/reports/{id}/resolve

Resolve a report with an action (admin/moderator only).

**Authentication**: Required (Bearer token) - Admin or Moderator role

**Path Parameters**:
- `id` (UUID): Report ID

**Request Body**:
```json
{
  "action": "remove",
  "resolution": "Image removed for violating content policy"
}
```

**Fields**:
- `action` (string, required): `dismiss`, `warn`, `remove`, or `ban`
- `resolution` (string, optional): Max 1000 chars

**Success Response (200 OK)**:
```json
{
  "id": "c90f4e9b-2345-6789-01bc-defg34567890",
  "reporter_id": "550e8400-e29b-41d4-a716-446655440000",
  "image_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "reason": "inappropriate",
  "status": "resolved",
  "resolved_by": {
    "id": "9f2g6h1d-4567-8901-23de-fghi56789012",
    "username": "moderator1"
  },
  "resolved_at": "2025-12-05T12:00:00Z",
  "resolution": "Image removed for violating content policy"
}
```

**Error Responses**:
- `400`: Validation error
- `401`: Unauthorized
- `403`: Requires admin or moderator role
- `404`: Report not found

**Actions**:
- `dismiss`: Close report without action
- `warn`: Send warning to image owner
- `remove`: Delete the reported image
- `ban`: Ban the image owner

---

#### POST /users/{id}/ban

Ban a user from the platform (admin only).

**Authentication**: Required (Bearer token) - Admin role

**Path Parameters**:
- `id` (UUID): User ID

**Request Body**:
```json
{
  "reason": "Multiple content policy violations",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

**Fields**:
- `reason` (string, required): Max 1000 chars
- `expires_at` (datetime, optional): Ban expiration (null for permanent)

**Success Response (200 OK)**:
```json
{
  "user_id": "8e1f5g0c-3456-7890-12cd-efgh45678901",
  "banned": true,
  "reason": "Multiple content policy violations",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

**Error Responses**:
- `400`: Validation error
- `401`: Unauthorized
- `403`: Requires admin role
- `404`: User not found

---

#### DELETE /users/{id}/ban

Remove ban from a user (admin only).

**Authentication**: Required (Bearer token) - Admin role

**Path Parameters**:
- `id` (UUID): User ID

**Success Response (200 OK)**:
```json
{
  "user_id": "8e1f5g0c-3456-7890-12cd-efgh45678901",
  "banned": false
}
```

**Error Responses**:
- `401`: Unauthorized
- `403`: Requires admin role
- `404`: User not found

---

### Explore Endpoints

#### GET /explore/recent

Retrieve recently uploaded public images.

**Authentication**: Optional

**Query Parameters**:
- `page` (integer, optional): Page number (default: 1)
- `per_page` (integer, optional): Items per page (default: 20, max: 100)

**Success Response (200 OK)**:
```json
{
  "items": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "title": "Sunset over mountains",
      "visibility": "public",
      "variants": {
        "thumbnail": {
          "url": "https://cdn.goimg.com/images/7c9e6679/thumbnail.jpg"
        }
      },
      "tags": ["sunset", "mountains"],
      "view_count": 150,
      "like_count": 42,
      "created_at": "2025-12-05T10:45:00Z"
    }
  ],
  "pagination": {
    "total": 5000,
    "page": 1,
    "per_page": 20,
    "total_pages": 250
  }
}
```

---

#### GET /explore/popular

Retrieve popular images sorted by views or likes.

**Authentication**: Optional

**Query Parameters**:
- `period` (string, optional): `day`, `week`, `month`, or `all` (default: `week`)
- `page` (integer, optional): Page number (default: 1)
- `per_page` (integer, optional): Items per page (default: 20, max: 100)

**Success Response (200 OK)**:
```json
{
  "items": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "title": "Sunset over mountains",
      "visibility": "public",
      "variants": {
        "thumbnail": {
          "url": "https://cdn.goimg.com/images/7c9e6679/thumbnail.jpg"
        }
      },
      "tags": ["sunset", "mountains"],
      "view_count": 1500,
      "like_count": 420,
      "created_at": "2025-12-05T10:45:00Z"
    }
  ],
  "pagination": {
    "total": 500,
    "page": 1,
    "per_page": 20,
    "total_pages": 25
  }
}
```

---

### Health & Monitoring

#### GET /health

Liveness check - verify the service is running.

**Authentication**: None required

**Success Response (200 OK)**:
```json
{
  "status": "ok",
  "timestamp": "2025-12-05T10:30:00Z"
}
```

---

#### GET /health/ready

Readiness check - verify the service is ready to accept traffic.

**Authentication**: None required

**Success Response (200 OK)**:
```json
{
  "status": "ready",
  "timestamp": "2025-12-05T10:30:00Z",
  "checks": {
    "database": {
      "status": "up",
      "latency_ms": 5.2
    },
    "redis": {
      "status": "up",
      "latency_ms": 2.1
    }
  }
}
```

**Degraded Response (503 Service Unavailable)**:
```json
{
  "status": "not_ready",
  "timestamp": "2025-12-05T10:30:00Z",
  "checks": {
    "database": {
      "status": "down",
      "error": "connection refused"
    },
    "redis": {
      "status": "up",
      "latency_ms": 2.1
    }
  }
}
```

---

#### GET /metrics

Prometheus metrics in text format.

**Authentication**: None required

**Success Response (200 OK)**:
```
# HELP goimg_http_requests_total Total HTTP requests
# TYPE goimg_http_requests_total counter
goimg_http_requests_total{method="GET",path="/health",status="200"} 42
goimg_http_requests_total{method="POST",path="/images",status="201"} 15

# HELP goimg_http_request_duration_seconds HTTP request latencies
# TYPE goimg_http_request_duration_seconds histogram
goimg_http_request_duration_seconds_bucket{method="GET",path="/images",le="0.1"} 50
goimg_http_request_duration_seconds_bucket{method="GET",path="/images",le="0.5"} 95
goimg_http_request_duration_seconds_bucket{method="GET",path="/images",le="+Inf"} 100

# HELP goimg_db_connections Database connection pool stats
# TYPE goimg_db_connections gauge
goimg_db_connections{state="idle"} 8
goimg_db_connections{state="in_use"} 2

# HELP goimg_redis_commands_total Redis commands executed
# TYPE goimg_redis_commands_total counter
goimg_redis_commands_total{command="get"} 1234
goimg_redis_commands_total{command="set"} 567

# HELP goimg_image_processing_duration_seconds Image processing latency
# TYPE goimg_image_processing_duration_seconds histogram
goimg_image_processing_duration_seconds_bucket{variant="thumbnail",le="0.1"} 10
goimg_image_processing_duration_seconds_bucket{variant="thumbnail",le="0.5"} 45
```

---

## Code Examples

### curl Examples

#### Register and Login

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "username": "johndoe",
    "password": "MySecureP@ssw0rd123"
  }'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "MySecureP@ssw0rd123"
  }'

# Save tokens from response
export ACCESS_TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
export REFRESH_TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
```

#### Upload Image

```bash
curl -X POST http://localhost:8080/api/v1/images \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "file=@/path/to/image.jpg" \
  -F "title=Sunset over mountains" \
  -F "description=Beautiful sunset" \
  -F "visibility=public" \
  -F "tags=sunset,mountains,nature"
```

#### List Images

```bash
curl -X GET "http://localhost:8080/api/v1/images?q=mountains&sort=like_count&order=desc&page=1&per_page=20" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

#### Create Album

```bash
curl -X POST http://localhost:8080/api/v1/albums \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Vacation 2024",
    "description": "Summer vacation photos",
    "visibility": "public"
  }'
```

#### Like Image

```bash
curl -X POST http://localhost:8080/api/v1/images/7c9e6679-7425-40de-944b-e07fc1f90ae7/like \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

#### Add Comment

```bash
curl -X POST http://localhost:8080/api/v1/images/7c9e6679-7425-40de-944b-e07fc1f90ae7/comments \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Amazing photo!"
  }'
```

#### Refresh Token

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "'$REFRESH_TOKEN'"
  }'
```

---

### JavaScript (fetch) Examples

#### Register and Login

```javascript
// Register
async function register() {
  const response = await fetch('http://localhost:8080/api/v1/auth/register', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      email: 'user@example.com',
      username: 'johndoe',
      password: 'MySecureP@ssw0rd123',
    }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.detail);
  }

  const user = await response.json();
  console.log('User created:', user);
  return user;
}

// Login
async function login() {
  const response = await fetch('http://localhost:8080/api/v1/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      email: 'user@example.com',
      password: 'MySecureP@ssw0rd123',
    }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.detail);
  }

  const tokens = await response.json();
  localStorage.setItem('access_token', tokens.access_token);
  localStorage.setItem('refresh_token', tokens.refresh_token);
  return tokens;
}
```

#### Upload Image

```javascript
async function uploadImage(file, metadata) {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('title', metadata.title);
  formData.append('description', metadata.description);
  formData.append('visibility', metadata.visibility);
  formData.append('tags', metadata.tags.join(','));

  const accessToken = localStorage.getItem('access_token');

  const response = await fetch('http://localhost:8080/api/v1/images', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${accessToken}`,
    },
    body: formData,
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.detail);
  }

  const image = await response.json();
  console.log('Image uploaded:', image);
  return image;
}

// Usage
const fileInput = document.querySelector('#file-input');
const file = fileInput.files[0];

uploadImage(file, {
  title: 'Sunset over mountains',
  description: 'Beautiful sunset',
  visibility: 'public',
  tags: ['sunset', 'mountains', 'nature'],
});
```

#### List Images

```javascript
async function listImages(filters = {}) {
  const accessToken = localStorage.getItem('access_token');

  const params = new URLSearchParams({
    page: filters.page || 1,
    per_page: filters.per_page || 20,
    ...(filters.q && { q: filters.q }),
    ...(filters.tags && { tags: filters.tags }),
    ...(filters.sort && { sort: filters.sort }),
    ...(filters.order && { order: filters.order }),
  });

  const response = await fetch(`http://localhost:8080/api/v1/images?${params}`, {
    headers: {
      'Authorization': `Bearer ${accessToken}`,
    },
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.detail);
  }

  const data = await response.json();
  return data;
}

// Usage
const images = await listImages({
  q: 'mountains',
  sort: 'like_count',
  order: 'desc',
  page: 1,
  per_page: 20,
});
```

#### Like Image

```javascript
async function likeImage(imageId) {
  const accessToken = localStorage.getItem('access_token');

  const response = await fetch(`http://localhost:8080/api/v1/images/${imageId}/like`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${accessToken}`,
    },
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.detail);
  }

  const result = await response.json();
  return result;
}
```

#### Refresh Token

```javascript
async function refreshToken() {
  const refreshToken = localStorage.getItem('refresh_token');

  const response = await fetch('http://localhost:8080/api/v1/auth/refresh', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      refresh_token: refreshToken,
    }),
  });

  if (!response.ok) {
    // Refresh token expired, redirect to login
    window.location.href = '/login';
    return;
  }

  const tokens = await response.json();
  localStorage.setItem('access_token', tokens.access_token);
  localStorage.setItem('refresh_token', tokens.refresh_token);
  return tokens;
}

// Automatic token refresh
async function fetchWithAuth(url, options = {}) {
  const accessToken = localStorage.getItem('access_token');

  let response = await fetch(url, {
    ...options,
    headers: {
      ...options.headers,
      'Authorization': `Bearer ${accessToken}`,
    },
  });

  // If 401, try refreshing token
  if (response.status === 401) {
    await refreshToken();
    const newAccessToken = localStorage.getItem('access_token');

    response = await fetch(url, {
      ...options,
      headers: {
        ...options.headers,
        'Authorization': `Bearer ${newAccessToken}`,
      },
    });
  }

  return response;
}
```

---

### Python (requests) Examples

#### Register and Login

```python
import requests

BASE_URL = "http://localhost:8080/api/v1"

# Register
def register():
    response = requests.post(
        f"{BASE_URL}/auth/register",
        json={
            "email": "user@example.com",
            "username": "johndoe",
            "password": "MySecureP@ssw0rd123"
        }
    )

    if not response.ok:
        error = response.json()
        raise Exception(error['detail'])

    user = response.json()
    print(f"User created: {user}")
    return user

# Login
def login():
    response = requests.post(
        f"{BASE_URL}/auth/login",
        json={
            "email": "user@example.com",
            "password": "MySecureP@ssw0rd123"
        }
    )

    if not response.ok:
        error = response.json()
        raise Exception(error['detail'])

    tokens = response.json()
    return tokens

# Get tokens
tokens = login()
access_token = tokens['access_token']
refresh_token = tokens['refresh_token']
```

#### Upload Image

```python
def upload_image(file_path, metadata, access_token):
    with open(file_path, 'rb') as f:
        files = {'file': f}
        data = {
            'title': metadata['title'],
            'description': metadata['description'],
            'visibility': metadata['visibility'],
            'tags': ','.join(metadata['tags'])
        }

        response = requests.post(
            f"{BASE_URL}/images",
            headers={'Authorization': f'Bearer {access_token}'},
            files=files,
            data=data
        )

    if not response.ok:
        error = response.json()
        raise Exception(error['detail'])

    image = response.json()
    print(f"Image uploaded: {image}")
    return image

# Usage
image = upload_image(
    '/path/to/image.jpg',
    {
        'title': 'Sunset over mountains',
        'description': 'Beautiful sunset',
        'visibility': 'public',
        'tags': ['sunset', 'mountains', 'nature']
    },
    access_token
)
```

#### List Images

```python
def list_images(access_token, filters=None):
    if filters is None:
        filters = {}

    params = {
        'page': filters.get('page', 1),
        'per_page': filters.get('per_page', 20)
    }

    if 'q' in filters:
        params['q'] = filters['q']
    if 'tags' in filters:
        params['tags'] = filters['tags']
    if 'sort' in filters:
        params['sort'] = filters['sort']
    if 'order' in filters:
        params['order'] = filters['order']

    response = requests.get(
        f"{BASE_URL}/images",
        headers={'Authorization': f'Bearer {access_token}'},
        params=params
    )

    if not response.ok:
        error = response.json()
        raise Exception(error['detail'])

    data = response.json()
    return data

# Usage
images = list_images(
    access_token,
    {
        'q': 'mountains',
        'sort': 'like_count',
        'order': 'desc',
        'page': 1,
        'per_page': 20
    }
)
```

#### Create Album

```python
def create_album(metadata, access_token):
    response = requests.post(
        f"{BASE_URL}/albums",
        headers={
            'Authorization': f'Bearer {access_token}',
            'Content-Type': 'application/json'
        },
        json=metadata
    )

    if not response.ok:
        error = response.json()
        raise Exception(error['detail'])

    album = response.json()
    return album

# Usage
album = create_album(
    {
        'title': 'Vacation 2024',
        'description': 'Summer vacation photos',
        'visibility': 'public'
    },
    access_token
)
```

#### Like Image

```python
def like_image(image_id, access_token):
    response = requests.post(
        f"{BASE_URL}/images/{image_id}/like",
        headers={'Authorization': f'Bearer {access_token}'}
    )

    if not response.ok:
        error = response.json()
        raise Exception(error['detail'])

    result = response.json()
    return result

# Usage
result = like_image('7c9e6679-7425-40de-944b-e07fc1f90ae7', access_token)
print(f"Liked: {result['liked']}, Like count: {result['like_count']}")
```

#### Add Comment

```python
def add_comment(image_id, content, access_token):
    response = requests.post(
        f"{BASE_URL}/images/{image_id}/comments",
        headers={
            'Authorization': f'Bearer {access_token}',
            'Content-Type': 'application/json'
        },
        json={'content': content}
    )

    if not response.ok:
        error = response.json()
        raise Exception(error['detail'])

    comment = response.json()
    return comment

# Usage
comment = add_comment(
    '7c9e6679-7425-40de-944b-e07fc1f90ae7',
    'Amazing photo!',
    access_token
)
```

#### Refresh Token

```python
def refresh_access_token(refresh_token):
    response = requests.post(
        f"{BASE_URL}/auth/refresh",
        json={'refresh_token': refresh_token}
    )

    if not response.ok:
        error = response.json()
        raise Exception(error['detail'])

    tokens = response.json()
    return tokens

# Automatic token refresh wrapper
class APIClient:
    def __init__(self, access_token, refresh_token):
        self.access_token = access_token
        self.refresh_token = refresh_token

    def request(self, method, endpoint, **kwargs):
        headers = kwargs.get('headers', {})
        headers['Authorization'] = f'Bearer {self.access_token}'
        kwargs['headers'] = headers

        response = requests.request(method, f"{BASE_URL}{endpoint}", **kwargs)

        # If 401, try refreshing token
        if response.status_code == 401:
            tokens = refresh_access_token(self.refresh_token)
            self.access_token = tokens['access_token']
            self.refresh_token = tokens['refresh_token']

            # Retry request
            headers['Authorization'] = f'Bearer {self.access_token}'
            response = requests.request(method, f"{BASE_URL}{endpoint}", **kwargs)

        return response

# Usage
client = APIClient(access_token, refresh_token)
response = client.request('GET', '/images', params={'page': 1})
images = response.json()
```

---

## Additional Resources

- **OpenAPI Specification**: [api/openapi/openapi.yaml](../../api/openapi/openapi.yaml)
- **Architecture Documentation**: [claude/architecture.md](../../claude/architecture.md)
- **API Security Guide**: [claude/api_security.md](../../claude/api_security.md)
- **Testing Guide**: [claude/testing_ci.md](../../claude/testing_ci.md)
- **GitHub Repository**: [yegamble/goimg-datalayer](https://github.com/yegamble/goimg-datalayer)

---

**Last Updated**: 2025-12-05 (Sprint 9)
**API Version**: 1.0.0
