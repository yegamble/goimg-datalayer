# API Contract & Security

> Load this guide when working on HTTP handlers, authentication, or API endpoints.

## OpenAPI as Source of Truth

Location: `api/openapi/`

```
api/openapi/
├── openapi.yaml      # Main specification
├── schemas/          # Reusable component schemas
│   ├── user.yaml
│   ├── image.yaml
│   └── error.yaml
└── paths/            # Endpoint definitions
    ├── auth.yaml
    ├── users.yaml
    └── images.yaml
```

### Validation Workflow

```bash
# Before any API changes
make validate-openapi

# Regenerate server code
make generate

# Verify no drift
git diff --exit-code
```

## RFC 7807 Problem Details

All error responses MUST use RFC 7807 format:

```go
type ProblemDetail struct {
    Type     string            `json:"type"`               // URI identifying problem type
    Title    string            `json:"title"`              // Short summary
    Status   int               `json:"status"`             // HTTP status code
    Detail   string            `json:"detail,omitempty"`   // Specific explanation
    Instance string            `json:"instance,omitempty"` // URI of occurrence
    Errors   []ValidationError `json:"errors,omitempty"`   // Field-level errors
    TraceID  string            `json:"traceId,omitempty"`  // Correlation ID
}
```

### Example Responses

```json
// 400 Bad Request - Validation
{
    "type": "https://api.goimg.com/problems/validation-error",
    "title": "Validation Error",
    "status": 400,
    "detail": "Request validation failed",
    "errors": [
        {"field": "email", "message": "must be a valid email address"},
        {"field": "password", "message": "must be at least 8 characters"}
    ],
    "traceId": "abc123-def456"
}

// 401 Unauthorized
{
    "type": "https://api.goimg.com/problems/unauthorized",
    "title": "Unauthorized",
    "status": 401,
    "detail": "Invalid or expired token"
}

// 404 Not Found
{
    "type": "https://api.goimg.com/problems/not-found",
    "title": "Not Found",
    "status": 404,
    "detail": "Image img_123 not found",
    "instance": "/api/v1/images/img_123"
}
```

## Authentication Flow

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  Login   │────▶│  Verify  │────▶│  Issue   │────▶│  Store   │
│ Request  │     │ Password │     │  Tokens  │     │ Session  │
└──────────┘     └──────────┘     └──────────┘     └──────────┘
                                        │
                      ┌─────────────────┼─────────────────┐
                      ▼                                   ▼
                ┌──────────┐                        ┌──────────┐
                │  Access  │                        │ Refresh  │
                │  Token   │                        │  Token   │
                │ (15 min) │                        │ (7 days) │
                └──────────┘                        └──────────┘
```

### JWT Configuration

| Setting | Value | Purpose |
|---------|-------|---------|
| Access Token TTL | 15 minutes | Short-lived for security |
| Refresh Token TTL | 7 days | Longer session persistence |
| Algorithm | RS256 or HS256 | Signing method |

### Token Claims

```go
type Claims struct {
    jwt.RegisteredClaims
    UserID      string   `json:"uid"`
    Role        string   `json:"role"`
    Permissions []string `json:"perms,omitempty"`
    SessionID   string   `json:"sid"`
}
```

## Role-Based Access Control

### Roles

| Role | Description |
|------|-------------|
| `admin` | Full system access |
| `moderator` | Content moderation, user warnings |
| `user` | Standard user operations |

### Permissions

```go
// Image permissions
PermImageUpload    = "image:upload"
PermImageDelete    = "image:delete"
PermImageDeleteAny = "image:delete:any"
PermImageModerate  = "image:moderate"

// User permissions
PermUserRead        = "user:read"
PermUserUpdate      = "user:update"
PermUserBan         = "user:ban"
PermUserManageRoles = "user:manage:roles"

// Moderation permissions
PermReportView    = "report:view"
PermReportResolve = "report:resolve"
```

### RBAC Middleware

```go
// Usage in routes
r.With(RequirePermission(PermImageModerate)).
    Post("/images/{id}/moderate", h.ModerateImage)

r.With(RequirePermission(PermUserBan)).
    Post("/users/{id}/ban", h.BanUser)
```

## Rate Limiting

| Scope | Limit | Window |
|-------|-------|--------|
| Global (per IP) | 100 requests | 1 minute |
| Authenticated | 300 requests | 1 minute |
| Uploads | 50 uploads | 1 hour |
| Login attempts | 5 attempts | 1 minute |

Redis-backed sliding window implementation.

## Image Security

### Upload Validation (Required Steps)

```go
func (p *ImageProcessor) ValidateUpload(data []byte) error {
    // 1. Size check
    if len(data) > p.maxFileSize {
        return ErrFileTooLarge
    }

    // 2. MIME sniffing (NOT extension!)
    mimeType := http.DetectContentType(data)
    if !isAllowedMIME(mimeType) {
        return ErrInvalidFileType
    }

    // 3. Decode validation
    _, err := bimg.NewImage(data).Size()
    if err != nil {
        return fmt.Errorf("invalid image: %w", err)
    }

    // 4. Malware scan
    result, err := p.clamav.Scan(data)
    if err != nil {
        return fmt.Errorf("scan failed: %w", err)
    }
    if result.Infected {
        return ErrMalwareDetected
    }

    return nil
}
```

### Allowed MIME Types

- `image/jpeg`
- `image/png`
- `image/gif`
- `image/webp`

## Security Headers

Applied by middleware to all responses:

```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("X-XSS-Protection", "1; mode=block")
w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
w.Header().Set("Content-Security-Policy", "default-src 'self'")
w.Header().Set("Permissions-Policy", "geolocation=(), microphone=()")
```

## Handler Expectations

Handlers MUST:
1. Parse and validate request DTOs
2. Delegate to application layer commands/queries
3. Map domain errors to Problem Detail responses
4. Never contain business logic
5. Never leak internal errors to clients

```go
// Map domain errors to HTTP responses
switch {
case errors.Is(err, identity.ErrUserNotFound):
    RespondProblem(w, r, ProblemNotFound("user not found"))
case errors.Is(err, identity.ErrInvalidCredentials):
    RespondProblem(w, r, ProblemUnauthorized())
case errors.Is(err, identity.ErrUserAlreadyExists):
    RespondProblem(w, r, ProblemConflict("user already exists"))
default:
    // Log internal error, return generic message
    log.Error().Err(err).Msg("internal error")
    RespondProblem(w, r, ProblemInternalError())
}
```

## Storage Abstraction

Storage providers implement a common interface:

```go
type Storage interface {
    Put(ctx context.Context, key string, data []byte) error
    Get(ctx context.Context, key string) ([]byte, error)
    Delete(ctx context.Context, key string) error
    URL(key string) string
}
```

Implementations in `internal/infrastructure/storage/`:
- `local/` - Filesystem storage
- `s3/` - S3-compatible (AWS, DO Spaces, Backblaze B2)
