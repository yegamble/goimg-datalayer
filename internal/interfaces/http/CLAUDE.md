# HTTP Interface Layer Guide

> Presentation layer: handlers, middleware, DTOs, routing with chi.

## Critical Rules

1. **No business logic** - Handlers only: parse → validate → delegate → respond
2. **Use DTOs** - Never serialize domain objects directly
3. **Problem Details (RFC 7807)** - All errors use standardized format
4. **Map domain errors** - Convert to appropriate HTTP status codes
5. **OpenAPI compliance** - All endpoints must match `api/openapi/openapi.yaml`
6. **Security headers** - Always set CORS, CSP, X-Frame-Options, etc.
7. **Test coverage: 75%** - Focus on handler logic and error mapping

## Structure

```
internal/interfaces/http/
├── server.go              # HTTP server setup with chi
├── router.go              # Route definitions
├── middleware/
│   ├── auth.go            # JWT validation (golang-jwt)
│   ├── rbac.go            # Permission checking
│   ├── rate_limit.go      # Rate limiting (go-redis)
│   ├── request_id.go      # X-Request-ID tracking
│   ├── logging.go         # Request/response logging (zerolog)
│   ├── cors.go            # CORS headers
│   └── recovery.go        # Panic recovery
├── handlers/
│   ├── health.go          # Health check endpoint
│   ├── auth_handler.go    # Login, register, logout
│   ├── user_handler.go    # User management
│   ├── image_handler.go   # Image upload, list, get
│   └── album_handler.go   # Album operations
├── dto/
│   ├── requests/
│   │   ├── user_request.go
│   │   └── image_request.go
│   └── responses/
│       ├── user_response.go
│       ├── image_response.go
│       └── error_response.go
└── openapi/
    └── generated.go       # oapi-codegen output
```

## Server Setup (chi router)

```go
package http

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/rs/zerolog"

    "goimg-datalayer/internal/interfaces/http/handlers"
    "goimg-datalayer/internal/interfaces/http/middleware"
)

type Server struct {
    router *chi.Mux
    server *http.Server
    logger *zerolog.Logger
}

func NewServer(
    logger *zerolog.Logger,
    handlers *handlers.Handlers,
    cfg Config,
) *Server {
    r := chi.NewRouter()

    // Global middleware (order matters!)
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger(logger))
    r.Use(middleware.Recoverer(logger))
    r.Use(middleware.CORS(cfg.CORS))
    r.Use(middleware.SecurityHeaders)
    r.Use(middleware.Timeout(30 * time.Second))

    // Health check (no auth required)
    r.Get("/health", handlers.Health.Check)
    r.Get("/ready", handlers.Health.Ready)

    // Public routes
    r.Group(func(r chi.Router) {
        r.Post("/api/v1/auth/register", handlers.Auth.Register)
        r.Post("/api/v1/auth/login", handlers.Auth.Login)
    })

    // Protected routes
    r.Group(func(r chi.Router) {
        // JWT authentication required
        r.Use(middleware.JWTAuth(cfg.JWTSecret))
        r.Use(middleware.RateLimitPerUser(cfg.Redis, 100, time.Minute))

        // User routes
        r.Route("/api/v1/users", func(r chi.Router) {
            r.Get("/me", handlers.User.GetCurrent)
            r.Patch("/me", handlers.User.UpdateProfile)
            r.Delete("/me", handlers.User.DeleteAccount)

            // Admin only
            r.Group(func(r chi.Router) {
                r.Use(middleware.RequireRole("admin"))
                r.Get("/", handlers.User.List)
                r.Get("/{userID}", handlers.User.GetByID)
                r.Delete("/{userID}", handlers.User.DeleteUser)
            })
        })

        // Image routes
        r.Route("/api/v1/images", func(r chi.Router) {
            r.Post("/", handlers.Image.Upload)
            r.Get("/", handlers.Image.List)
            r.Get("/{imageID}", handlers.Image.GetByID)
            r.Patch("/{imageID}", handlers.Image.Update)
            r.Delete("/{imageID}", handlers.Image.Delete)
        })
    })

    srv := &http.Server{
        Addr:         fmt.Sprintf(":%d", cfg.Port),
        Handler:      r,
        ReadTimeout:  cfg.ReadTimeout,
        WriteTimeout: cfg.WriteTimeout,
        IdleTimeout:  cfg.IdleTimeout,
    }

    return &Server{
        router: r,
        server: srv,
        logger: logger,
    }
}

func (s *Server) Start() error {
    s.logger.Info().
        Str("addr", s.server.Addr).
        Msg("starting http server")

    if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        return fmt.Errorf("http server error: %w", err)
    }

    return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
    s.logger.Info().Msg("shutting down http server")
    return s.server.Shutdown(ctx)
}
```

## Handler Pattern

### Standard Handler Structure

```go
package handlers

import (
    "encoding/json"
    "errors"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/rs/zerolog"

    "goimg-datalayer/internal/application/commands"
    "goimg-datalayer/internal/application/queries"
    "goimg-datalayer/internal/domain/identity"
    "goimg-datalayer/internal/interfaces/http/dto/requests"
    "goimg-datalayer/internal/interfaces/http/dto/responses"
)

type UserHandler struct {
    createUser  *commands.CreateUserHandler
    getUser     *queries.GetUserHandler
    updateUser  *commands.UpdateUserProfileHandler
    listUsers   *queries.ListUsersHandler
    logger      *zerolog.Logger
}

func NewUserHandler(
    createUser *commands.CreateUserHandler,
    getUser *queries.GetUserHandler,
    updateUser *commands.UpdateUserProfileHandler,
    listUsers *queries.ListUsersHandler,
    logger *zerolog.Logger,
) *UserHandler {
    return &UserHandler{
        createUser:  createUser,
        getUser:     getUser,
        updateUser:  updateUser,
        listUsers:   listUsers,
        logger:      logger,
    }
}

// Create handles POST /api/v1/users
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // 1. Parse request DTO
    var req requests.CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        RespondProblem(w, r, ProblemBadRequest("invalid request body", err.Error()))
        return
    }

    // 2. Validate DTO (go-playground/validator)
    if err := Validate(req); err != nil {
        RespondProblem(w, r, ProblemValidation(err))
        return
    }

    // 3. Delegate to application layer
    user, err := h.createUser.Handle(ctx, commands.CreateUserCommand{
        Email:    req.Email,
        Username: req.Username,
        Password: req.Password,
    })

    // 4. Map domain errors to HTTP status codes
    if err != nil {
        problem := h.mapError(err)
        RespondProblem(w, r, problem)
        return
    }

    // 5. Map domain entity to response DTO
    resp := responses.MapUserToResponse(user)

    // 6. Send response
    RespondJSON(w, http.StatusCreated, resp)
}

// GetByID handles GET /api/v1/users/{userID}
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // 1. Extract path parameter
    userID := chi.URLParam(r, "userID")
    if userID == "" {
        RespondProblem(w, r, ProblemBadRequest("missing user id", ""))
        return
    }

    // 2. Delegate to query handler
    user, err := h.getUser.Handle(ctx, queries.GetUserQuery{
        UserID: userID,
    })

    // 3. Handle errors
    if err != nil {
        problem := h.mapError(err)
        RespondProblem(w, r, problem)
        return
    }

    // 4. Map to response DTO
    resp := responses.MapUserToResponse(user)

    // 5. Send response
    RespondJSON(w, http.StatusOK, resp)
}

// List handles GET /api/v1/users?role=admin&status=active&search=john&offset=0&limit=20
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // 1. Parse query parameters
    query := r.URL.Query()

    var role *string
    if r := query.Get("role"); r != "" {
        role = &r
    }

    var status *string
    if s := query.Get("status"); s != "" {
        status = &s
    }

    search := query.Get("search")
    offset := ParseIntQueryParam(query, "offset", 0)
    limit := ParseIntQueryParam(query, "limit", 20)

    // 2. Delegate to query handler
    result, err := h.listUsers.Handle(ctx, queries.ListUsersQuery{
        Role:   role,
        Status: status,
        Search: search,
        Offset: offset,
        Limit:  limit,
    })

    if err != nil {
        problem := h.mapError(err)
        RespondProblem(w, r, problem)
        return
    }

    // 3. Map to paginated response
    resp := responses.PaginatedUsersResponse{
        Users:      responses.MapUsersToResponse(result.Users),
        TotalCount: result.TotalCount,
        Offset:     offset,
        Limit:      limit,
    }

    RespondJSON(w, http.StatusOK, resp)
}

// mapError converts domain errors to HTTP problem details
func (h *UserHandler) mapError(err error) ProblemDetail {
    switch {
    case errors.Is(err, identity.ErrUserNotFound):
        return ProblemNotFound("user not found")

    case errors.Is(err, identity.ErrUserAlreadyExists):
        return ProblemConflict("user already exists")

    case errors.Is(err, identity.ErrEmailInvalid):
        return ProblemBadRequest("invalid email", err.Error())

    case errors.Is(err, identity.ErrUsernameInvalid):
        return ProblemBadRequest("invalid username", err.Error())

    case errors.Is(err, identity.ErrInsufficientPermission):
        return ProblemForbidden("insufficient permission")

    default:
        h.logger.Error().Err(err).Msg("internal error in user handler")
        return ProblemInternalError()
    }
}
```

## DTOs (Data Transfer Objects)

### Request DTOs

```go
package requests

type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email,max=255"`
    Username string `json:"username" validate:"required,alphanum,min=3,max=50"`
    Password string `json:"password" validate:"required,min=8,max=128"`
}

type UpdateUserProfileRequest struct {
    Email    *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
    Username *string `json:"username,omitempty" validate:"omitempty,alphanum,min=3,max=50"`
    Bio      *string `json:"bio,omitempty" validate:"omitempty,max=500"`
}

type UploadImageRequest struct {
    Title       string   `json:"title" validate:"required,max=255"`
    Description string   `json:"description,omitempty" validate:"omitempty,max=2000"`
    Tags        []string `json:"tags,omitempty" validate:"omitempty,dive,max=50"`
    IsPublic    bool     `json:"isPublic"`
}
```

### Response DTOs

```go
package responses

import (
    "time"

    "goimg-datalayer/internal/domain/identity"
    "goimg-datalayer/internal/domain/media"
)

type UserResponse struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    Username  string    `json:"username"`
    Role      string    `json:"role"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"createdAt"`
}

func MapUserToResponse(user *identity.User) UserResponse {
    return UserResponse{
        ID:        user.ID().String(),
        Email:     user.Email().String(),
        Username:  user.Username().String(),
        Role:      user.Role().String(),
        Status:    user.Status().String(),
        CreatedAt: user.CreatedAt(),
    }
}

func MapUsersToResponse(users []*identity.User) []UserResponse {
    resp := make([]UserResponse, len(users))
    for i, user := range users {
        resp[i] = MapUserToResponse(user)
    }
    return resp
}

type ImageResponse struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description,omitempty"`
    URL         string    `json:"url"`
    ThumbnailURL string   `json:"thumbnailUrl"`
    UserID      string    `json:"userId"`
    Tags        []string  `json:"tags"`
    IsPublic    bool      `json:"isPublic"`
    CreatedAt   time.Time `json:"createdAt"`
}

func MapImageToResponse(image *media.Image) ImageResponse {
    return ImageResponse{
        ID:           image.ID().String(),
        Title:        image.Title().String(),
        Description:  image.Description().String(),
        URL:          image.URL(),
        ThumbnailURL: image.ThumbnailURL(),
        UserID:       image.UserID().String(),
        Tags:         image.Tags(),
        IsPublic:     image.IsPublic(),
        CreatedAt:    image.CreatedAt(),
    }
}

type PaginatedUsersResponse struct {
    Users      []UserResponse `json:"users"`
    TotalCount int            `json:"totalCount"`
    Offset     int            `json:"offset"`
    Limit      int            `json:"limit"`
}
```

## RFC 7807 Problem Details

```go
package http

import (
    "encoding/json"
    "net/http"

    "github.com/go-playground/validator/v10"
)

type ProblemDetail struct {
    Type     string                 `json:"type"`
    Title    string                 `json:"title"`
    Status   int                    `json:"status"`
    Detail   string                 `json:"detail,omitempty"`
    Instance string                 `json:"instance,omitempty"`
    TraceID  string                 `json:"traceId,omitempty"`
    Errors   map[string]interface{} `json:"errors,omitempty"`
}

func ProblemBadRequest(title, detail string) ProblemDetail {
    return ProblemDetail{
        Type:   "https://api.goimg.dev/problems/bad-request",
        Title:  title,
        Status: http.StatusBadRequest,
        Detail: detail,
    }
}

func ProblemValidation(err error) ProblemDetail {
    validationErrors := make(map[string]interface{})

    if ve, ok := err.(validator.ValidationErrors); ok {
        for _, fe := range ve {
            validationErrors[fe.Field()] = map[string]string{
                "tag":   fe.Tag(),
                "value": fe.Param(),
            }
        }
    }

    return ProblemDetail{
        Type:   "https://api.goimg.dev/problems/validation-error",
        Title:  "Validation failed",
        Status: http.StatusBadRequest,
        Detail: "One or more fields are invalid",
        Errors: validationErrors,
    }
}

func ProblemNotFound(title string) ProblemDetail {
    return ProblemDetail{
        Type:   "https://api.goimg.dev/problems/not-found",
        Title:  title,
        Status: http.StatusNotFound,
    }
}

func ProblemConflict(title string) ProblemDetail {
    return ProblemDetail{
        Type:   "https://api.goimg.dev/problems/conflict",
        Title:  title,
        Status: http.StatusConflict,
    }
}

func ProblemUnauthorized() ProblemDetail {
    return ProblemDetail{
        Type:   "https://api.goimg.dev/problems/unauthorized",
        Title:  "Authentication required",
        Status: http.StatusUnauthorized,
    }
}

func ProblemForbidden(title string) ProblemDetail {
    return ProblemDetail{
        Type:   "https://api.goimg.dev/problems/forbidden",
        Title:  title,
        Status: http.StatusForbidden,
    }
}

func ProblemInternalError() ProblemDetail {
    return ProblemDetail{
        Type:   "https://api.goimg.dev/problems/internal-error",
        Title:  "Internal server error",
        Status: http.StatusInternalServerError,
        Detail: "An unexpected error occurred. Please try again later.",
    }
}

func RespondProblem(w http.ResponseWriter, r *http.Request, problem ProblemDetail) {
    // Add request ID and instance
    problem.TraceID = GetRequestID(r.Context())
    problem.Instance = r.URL.Path

    w.Header().Set("Content-Type", "application/problem+json")
    w.WriteHeader(problem.Status)
    _ = json.NewEncoder(w).Encode(problem)
}
```

## Middleware Patterns

### JWT Authentication

```go
package middleware

import (
    "context"
    "fmt"
    "net/http"
    "strings"

    "github.com/golang-jwt/jwt/v5"

    httputil "goimg-datalayer/internal/interfaces/http"
)

type contextKey string

const UserIDKey contextKey = "userID"
const UserRoleKey contextKey = "userRole"

func JWTAuth(secret string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Extract token from Authorization header
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                httputil.RespondProblem(w, r, httputil.ProblemUnauthorized())
                return
            }

            parts := strings.Split(authHeader, " ")
            if len(parts) != 2 || parts[0] != "Bearer" {
                httputil.RespondProblem(w, r, httputil.ProblemUnauthorized())
                return
            }

            tokenString := parts[1]

            // 2. Parse and validate JWT
            token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
                if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                    return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
                }
                return []byte(secret), nil
            })

            if err != nil || !token.Valid {
                httputil.RespondProblem(w, r, httputil.ProblemUnauthorized())
                return
            }

            // 3. Extract claims
            claims, ok := token.Claims.(jwt.MapClaims)
            if !ok {
                httputil.RespondProblem(w, r, httputil.ProblemUnauthorized())
                return
            }

            userID, _ := claims["sub"].(string)
            role, _ := claims["role"].(string)

            // 4. Add to context
            ctx := context.WithValue(r.Context(), UserIDKey, userID)
            ctx = context.WithValue(ctx, UserRoleKey, role)

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func GetUserID(ctx context.Context) string {
    if userID, ok := ctx.Value(UserIDKey).(string); ok {
        return userID
    }
    return ""
}

func GetUserRole(ctx context.Context) string {
    if role, ok := ctx.Value(UserRoleKey).(string); ok {
        return role
    }
    return ""
}
```

### RBAC Middleware

```go
package middleware

import (
    "net/http"

    httputil "goimg-datalayer/internal/interfaces/http"
)

func RequireRole(requiredRole string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            role := GetUserRole(r.Context())

            if role != requiredRole {
                httputil.RespondProblem(w, r, httputil.ProblemForbidden("insufficient permissions"))
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### Security Headers

```go
package middleware

import "net/http"

func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

        next.ServeHTTP(w, r)
    })
}
```

## Testing Requirements

### Coverage Target: 75%

Test handler logic, error mapping, and middleware:

```go
package handlers_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/go-chi/chi/v5"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "goimg-datalayer/internal/interfaces/http/handlers"
    "goimg-datalayer/internal/interfaces/http/dto/requests"
    "goimg-datalayer/internal/interfaces/http/dto/responses"
)

func TestUserHandler_Create_Success(t *testing.T) {
    // Arrange
    mockCreateUser := new(MockCreateUserHandler)
    logger := zerolog.Nop()

    handler := handlers.NewUserHandler(mockCreateUser, nil, nil, nil, &logger)

    reqBody := requests.CreateUserRequest{
        Email:    "test@example.com",
        Username: "testuser",
        Password: "Password123!",
    }

    body, _ := json.Marshal(reqBody)
    req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
    rec := httptest.NewRecorder()

    expectedUser := createMockUser()
    mockCreateUser.On("Handle", mock.Anything, mock.Anything).
        Return(expectedUser, nil)

    // Act
    handler.Create(rec, req)

    // Assert
    assert.Equal(t, http.StatusCreated, rec.Code)

    var resp responses.UserResponse
    err := json.NewDecoder(rec.Body).Decode(&resp)
    require.NoError(t, err)

    assert.Equal(t, expectedUser.ID().String(), resp.ID)
    assert.Equal(t, expectedUser.Email().String(), resp.Email)

    mockCreateUser.AssertExpectations(t)
}

func TestUserHandler_Create_ValidationError(t *testing.T) {
    handler := handlers.NewUserHandler(nil, nil, nil, nil, &zerolog.Nop())

    reqBody := requests.CreateUserRequest{
        Email:    "invalid-email",
        Username: "",
        Password: "123", // Too short
    }

    body, _ := json.Marshal(reqBody)
    req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
    rec := httptest.NewRecorder()

    handler.Create(rec, req)

    assert.Equal(t, http.StatusBadRequest, rec.Code)

    var problem httputil.ProblemDetail
    err := json.NewDecoder(rec.Body).Decode(&problem)
    require.NoError(t, err)

    assert.Equal(t, "Validation failed", problem.Title)
    assert.NotEmpty(t, problem.Errors)
}
```

## Validation with go-playground/validator

```go
package http

import (
    "github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
    validate = validator.New()
}

func Validate(req interface{}) error {
    return validate.Struct(req)
}
```

## Agent Responsibilities

- **senior-go-architect**: Reviews handler structure, middleware patterns
- **backend-developer**: Implements handlers and DTOs
- **senior-secops-engineer**: Reviews security headers, JWT implementation, RBAC
- **test-strategist**: Ensures 75% coverage with proper HTTP testing patterns

## See Also

- Application layer integration: `internal/application/CLAUDE.md`
- OpenAPI spec: `api/openapi/openapi.yaml`
- API & security guide: `claude/api_security.md`
- Middleware guide: `claude/api_security.md`
