# HTTP Interface Layer Guide

> Presentation layer: handlers, middleware, DTOs, routing.

## Key Rules

1. **No business logic** - Handlers only: parse → validate → delegate → respond
2. **Use DTOs** - Never serialize domain objects directly
3. **Problem Details** - All errors use RFC 7807 format
4. **Map domain errors** - Convert to appropriate HTTP status codes
5. **OpenAPI compliance** - All endpoints must match spec

## Structure

```
internal/interfaces/http/
├── server.go              # HTTP server setup
├── router.go              # Route definitions
├── middleware/
│   ├── auth.go            # JWT validation
│   ├── rbac.go            # Permission checking
│   ├── rate_limit.go
│   ├── request_id.go
│   └── logging.go
├── handlers/
│   ├── health.go
│   ├── auth_handler.go
│   ├── user_handler.go
│   └── image_handler.go
├── dto/
│   ├── requests/
│   └── responses/
└── openapi/
    └── generated.go       # oapi-codegen output
```

## Handler Pattern

```go
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request DTO
    var req requests.CreateUserRequest
    if err := DecodeJSON(r, &req); err != nil {
        RespondProblem(w, r, ProblemBadRequest(err.Error()))
        return
    }

    // 2. Validate DTO
    if err := Validate(req); err != nil {
        RespondProblem(w, r, ProblemValidation(err))
        return
    }

    // 3. Delegate to application layer
    user, err := h.createUser.Handle(r.Context(), commands.CreateUserCommand{
        Email:    req.Email,
        Username: req.Username,
    })

    // 4. Map domain errors to HTTP
    if err != nil {
        switch {
        case errors.Is(err, identity.ErrUserAlreadyExists):
            RespondProblem(w, r, ProblemConflict("user exists"))
        case errors.Is(err, identity.ErrEmailInvalid):
            RespondProblem(w, r, ProblemBadRequest("invalid email"))
        default:
            log.Error().Err(err).Msg("internal error")
            RespondProblem(w, r, ProblemInternalError())
        }
        return
    }

    // 5. Map to response DTO
    RespondJSON(w, http.StatusCreated, responses.UserResponse{
        ID:       user.ID().String(),
        Email:    user.Email().String(),
        Username: user.Username().String(),
    })
}
```

## Error Response (RFC 7807)

```go
type ProblemDetail struct {
    Type     string `json:"type"`
    Title    string `json:"title"`
    Status   int    `json:"status"`
    Detail   string `json:"detail,omitempty"`
    TraceID  string `json:"traceId,omitempty"`
}
```

## See Also

- Full API guide: `claude/api_security.md`
- OpenAPI spec: `api/openapi/openapi.yaml`
