# Identity Application Layer Guide

> Authentication, authorization, and user management use cases for goimg-datalayer.

## Overview

This directory implements the application layer for the Identity bounded context, orchestrating authentication workflows, user management, and session handling using CQRS patterns.

## Architecture

```
internal/application/identity/
├── types.go              # Base Command/Query interfaces, EventPublisher
├── commands/             # Write operations (create, update, delete)
│   ├── register_user.go
│   ├── login_user.go
│   ├── refresh_token.go
│   ├── logout_user.go
│   ├── change_password.go
│   └── update_profile.go
├── queries/              # Read operations (get, list)
│   ├── get_user.go
│   ├── list_users.go
│   └── get_active_sessions.go
├── dto/                  # Data Transfer Objects for HTTP layer
│   └── dto.go
└── services/             # Application services
    └── auth_service.go   # Authentication orchestration
```

## CQRS Pattern Usage

This application layer follows **CQRS-Lite** (Command Query Responsibility Segregation):

- **Commands**: State-changing operations that modify aggregates
- **Queries**: Read-only operations that retrieve data without side effects
- **Separation**: Clear boundary between writes and reads

### Why CQRS?

1. **Clear Intent**: Commands vs. Queries make operation semantics explicit
2. **Testability**: Easy to mock and test handlers independently
3. **Scalability**: Can optimize reads and writes differently
4. **Auditability**: Every state change goes through a command handler
5. **Event Sourcing Ready**: Natural fit if we need full audit trail later

## Command Pattern

### Command Structure

Commands represent **user intent** and contain all data needed to execute an operation.

```go
// Command struct (input parameters)
type RegisterUserCommand struct {
    Email    string
    Username string
    Password string
    IP       string
    UserAgent string
}

// Implement Command interface
func (RegisterUserCommand) isCommand() {}

// Handler with dependencies injected
type RegisterUserHandler struct {
    users          identity.UserRepository
    authService    services.AuthService
    eventPublisher EventPublisher
    logger         *zerolog.Logger
}

// Constructor with dependency injection
func NewRegisterUserHandler(
    users identity.UserRepository,
    authService services.AuthService,
    eventPublisher EventPublisher,
    logger *zerolog.Logger,
) *RegisterUserHandler {
    return &RegisterUserHandler{
        users:          users,
        authService:    authService,
        eventPublisher: eventPublisher,
        logger:         logger,
    }
}

// Handle executes the use case
func (h *RegisterUserHandler) Handle(ctx context.Context, cmd RegisterUserCommand) (dto.AuthResponseDTO, error) {
    // 1. Convert DTOs to domain value objects
    email, err := identity.NewEmail(cmd.Email)
    if err != nil {
        return dto.AuthResponseDTO{}, fmt.Errorf("invalid email: %w", err)
    }

    username, err := identity.NewUsername(cmd.Username)
    if err != nil {
        return dto.AuthResponseDTO{}, fmt.Errorf("invalid username: %w", err)
    }

    // 2. Check business rules (uniqueness)
    existingByEmail, err := h.users.FindByEmail(ctx, email)
    if err != nil && !errors.Is(err, identity.ErrUserNotFound) {
        return dto.AuthResponseDTO{}, fmt.Errorf("check email uniqueness: %w", err)
    }
    if existingByEmail != nil {
        return dto.AuthResponseDTO{}, identity.ErrEmailAlreadyExists
    }

    // 3. Hash password (application layer concern)
    passwordHash, err := identity.HashPassword(cmd.Password)
    if err != nil {
        return dto.AuthResponseDTO{}, fmt.Errorf("hash password: %w", err)
    }

    // 4. Create aggregate via domain factory
    user, err := identity.NewUser(email, username, passwordHash)
    if err != nil {
        return dto.AuthResponseDTO{}, fmt.Errorf("create user: %w", err)
    }

    // 5. Persist aggregate
    if err := h.users.Save(ctx, user); err != nil {
        return dto.AuthResponseDTO{}, fmt.Errorf("save user: %w", err)
    }

    // 6. Publish domain events AFTER successful save
    for _, event := range user.Events() {
        if err := h.eventPublisher.Publish(ctx, event); err != nil {
            h.logger.Error().Err(err).
                Str("event_type", event.EventType()).
                Msg("failed to publish domain event")
        }
    }
    user.ClearEvents()

    // 7. Delegate to AuthService for token generation
    loginDTO := dto.LoginDTO{
        Identifier: cmd.Email,
        Password:   cmd.Password,
        IP:         cmd.IP,
        UserAgent:  cmd.UserAgent,
    }
    authResponse, err := h.authService.Login(ctx, loginDTO)
    if err != nil {
        return dto.AuthResponseDTO{}, fmt.Errorf("auto-login after registration: %w", err)
    }

    h.logger.Info().
        Str("user_id", user.ID().String()).
        Str("email", user.Email().String()).
        Msg("user registered successfully")

    return authResponse, nil
}
```

### Command Responsibilities

1. **Input Validation**: Convert primitives to domain value objects
2. **Business Rule Enforcement**: Check uniqueness, permissions, invariants
3. **Aggregate Lifecycle**: Create, modify, or delete aggregates
4. **Persistence**: Save changes via repository
5. **Event Publishing**: Publish domain events after successful save
6. **Logging**: Log significant operations

### Command Naming Convention

- `{Verb}{Entity}Command`: RegisterUserCommand, UpdateProfileCommand
- Handler: `{Verb}{Entity}Handler`: RegisterUserHandler, UpdateProfileHandler

## Query Pattern

### Query Structure

Queries represent **data retrieval intent** and return read-only views.

```go
// Query struct (input parameters)
type GetUserQuery struct {
    UserID string
}

// Implement Query interface
func (GetUserQuery) isQuery() {}

// Handler with read-only dependencies
type GetUserHandler struct {
    users identity.UserRepository
}

func NewGetUserHandler(users identity.UserRepository) *GetUserHandler {
    return &GetUserHandler{users: users}
}

// Handle executes the query
func (h *GetUserHandler) Handle(ctx context.Context, q GetUserQuery) (dto.UserDTO, error) {
    // 1. Validate input
    userID, err := identity.ParseUserID(q.UserID)
    if err != nil {
        return dto.UserDTO{}, fmt.Errorf("invalid user id: %w", err)
    }

    // 2. Retrieve aggregate
    user, err := h.users.FindByID(ctx, userID)
    if err != nil {
        return dto.UserDTO{}, fmt.Errorf("find user: %w", err)
    }

    // 3. Convert to DTO
    return dto.FromDomain(user), nil
}
```

### Query Responsibilities

1. **Input Validation**: Parse and validate query parameters
2. **Data Retrieval**: Load aggregates or projections from repository
3. **DTO Conversion**: Convert domain objects to DTOs
4. **No Side Effects**: NEVER modify state in queries

### Query Naming Convention

- `{Verb}{Entity}Query`: GetUserQuery, ListUsersQuery
- Handler: `{Verb}{Entity}Handler`: GetUserHandler, ListUsersHandler

## Application Services

Application services orchestrate **complex workflows** that span multiple commands, queries, or aggregates.

### When to Use Application Services

Use application services when you need:
- Multi-step workflows (e.g., login = validate credentials + generate tokens + create session)
- Integration with multiple infrastructure services
- Transaction coordination across aggregates
- Complex business logic that doesn't fit in a single command

### AuthService Example

```go
// Interface defined in services/auth_service.go
type AuthService interface {
    Register(ctx context.Context, req dto.CreateUserDTO) (dto.AuthResponseDTO, error)
    Login(ctx context.Context, req dto.LoginDTO) (dto.AuthResponseDTO, error)
    RefreshToken(ctx context.Context, req dto.RefreshTokenDTO) (dto.TokenPairDTO, error)
    Logout(ctx context.Context, sessionID, accessToken, refreshToken string) error
    // ... other methods
}

// Implementation in services/auth_service_impl.go (to be created)
type AuthServiceImpl struct {
    users          identity.UserRepository
    sessions       SessionStore
    jwtService     JWTService
    refreshService RefreshTokenService
    blacklist      TokenBlacklist
    logger         *zerolog.Logger
}

func (s *AuthServiceImpl) Login(ctx context.Context, req dto.LoginDTO) (dto.AuthResponseDTO, error) {
    // 1. Parse identifier (email or username)
    var user *identity.User
    var err error

    if email, emailErr := identity.NewEmail(req.Identifier); emailErr == nil {
        user, err = s.users.FindByEmail(ctx, email)
    } else if username, usernameErr := identity.NewUsername(req.Identifier); usernameErr == nil {
        user, err = s.users.FindByUsername(ctx, username)
    } else {
        return dto.AuthResponseDTO{}, fmt.Errorf("invalid identifier format")
    }

    if err != nil {
        return dto.AuthResponseDTO{}, identity.ErrInvalidCredentials
    }

    // 2. Verify password
    if err := user.VerifyPassword(req.Password); err != nil {
        return dto.AuthResponseDTO{}, identity.ErrInvalidCredentials
    }

    // 3. Check user status
    if !user.CanLogin() {
        return dto.AuthResponseDTO{}, identity.ErrUserSuspended
    }

    // 4. Generate session ID
    sessionID := uuid.New().String()

    // 5. Generate JWT tokens
    accessToken, err := s.jwtService.GenerateAccessToken(
        user.ID().String(),
        user.Email().String(),
        user.Role().String(),
        sessionID,
    )
    if err != nil {
        return dto.AuthResponseDTO{}, fmt.Errorf("generate access token: %w", err)
    }

    // 6. Generate refresh token family
    familyID := uuid.New().String()
    refreshToken, metadata, err := s.refreshService.GenerateToken(
        ctx,
        user.ID().String(),
        sessionID,
        familyID,
        "", // No parent for first token
        req.IP,
        req.UserAgent,
    )
    if err != nil {
        return dto.AuthResponseDTO{}, fmt.Errorf("generate refresh token: %w", err)
    }

    // 7. Create session in Redis
    session := Session{
        SessionID: sessionID,
        UserID:    user.ID().String(),
        Email:     user.Email().String(),
        Role:      user.Role().String(),
        IP:        req.IP,
        UserAgent: req.UserAgent,
        CreatedAt: time.Now().UTC(),
        ExpiresAt: metadata.ExpiresAt,
    }
    if err := s.sessions.Create(ctx, session); err != nil {
        return dto.AuthResponseDTO{}, fmt.Errorf("create session: %w", err)
    }

    // 8. Build response
    expiresAt, _ := s.jwtService.GetTokenExpiration(accessToken)
    tokens := dto.NewTokenPairDTO(accessToken, refreshToken, expiresAt)

    return dto.NewAuthResponseDTO(user, tokens), nil
}
```

## Dependency Injection

All handlers and services use **constructor injection** for dependencies.

### Why Constructor Injection?

1. **Explicit Dependencies**: Clear what each component needs
2. **Testability**: Easy to inject mocks/stubs
3. **Immutability**: Dependencies set once, never changed
4. **No Global State**: No singletons or service locators

### Dependency Guidelines

**Commands** typically need:
- Repository (for persistence)
- EventPublisher (for domain events)
- Logger (for observability)
- Application services (for orchestration)

**Queries** typically need:
- Repository (read-only)
- Logger (optional)

**Application Services** typically need:
- Multiple repositories
- Infrastructure services (JWT, Redis, etc.)
- Logger

## Error Handling

### Error Wrapping

Always wrap errors with context using `fmt.Errorf`:

```go
if err != nil {
    return fmt.Errorf("save user: %w", err)
}
```

### Domain Error Propagation

Let domain errors bubble up but add application context:

```go
user, err := h.users.FindByID(ctx, userID)
if err != nil {
    if errors.Is(err, identity.ErrUserNotFound) {
        return fmt.Errorf("user %s not found: %w", userID, err)
    }
    return fmt.Errorf("find user for profile update: %w", err)
}
```

### Error Handling in Event Publishing

Event publishing failures should **NOT** fail the operation:

```go
for _, event := range user.Events() {
    if err := h.eventPublisher.Publish(ctx, event); err != nil {
        // Log but don't fail
        h.logger.Error().Err(err).
            Str("event_type", event.EventType()).
            Msg("failed to publish domain event")
    }
}
```

## Transaction Boundaries

### When to Use Transactions

1. **Single Aggregate Save**: No transaction needed (atomic by nature)
2. **Multiple Operations**: Use transaction (e.g., update user + create audit log)
3. **Cross-Aggregate Consistency**: Use transaction (rare, prefer eventual consistency)

### Transaction Pattern

```go
// Start transaction
txRepo, err := h.users.WithTx(ctx)
if err != nil {
    return fmt.Errorf("begin transaction: %w", err)
}

// Defer rollback (no-op if committed)
defer func() {
    if err != nil {
        _ = txRepo.RollbackTx(ctx)
    }
}()

// Perform operations
if err := txRepo.Save(ctx, user); err != nil {
    return fmt.Errorf("save user: %w", err)
}

// Commit transaction
if err := txRepo.CommitTx(ctx); err != nil {
    return fmt.Errorf("commit transaction: %w", err)
}

// Publish events AFTER commit
for _, event := range user.Events() {
    h.eventPublisher.Publish(ctx, event)
}
```

## DTOs (Data Transfer Objects)

DTOs are used for **data transfer between layers** (application ↔ HTTP).

### DTO Guidelines

1. **JSON Tags**: Always include JSON tags for serialization
2. **Validation Tags**: Use `validate` tags for input validation
3. **Pointer Fields**: Use pointers for optional fields in requests
4. **No Logic**: DTOs should be pure data structures
5. **Conversion Functions**: Provide `FromDomain()` and `ToDomain()` helpers

### DTO Conversion

```go
// Domain to DTO (for responses)
func FromDomain(user *identity.User) UserDTO {
    return UserDTO{
        ID:       user.ID().String(),
        Email:    user.Email().String(),
        Username: user.Username().String(),
        // ... other fields
    }
}

// DTO to Domain (in handler)
email, err := identity.NewEmail(dto.Email)
username, err := identity.NewUsername(dto.Username)
```

## Testing Strategy

### Coverage Target: 85%

The application layer is critical and must be thoroughly tested.

### Command Handler Testing

```go
func TestRegisterUserHandler_Handle_Success(t *testing.T) {
    // Arrange
    mockRepo := new(MockUserRepository)
    mockAuthService := new(MockAuthService)
    mockPublisher := new(MockEventPublisher)
    logger := zerolog.Nop()

    handler := NewRegisterUserHandler(mockRepo, mockAuthService, mockPublisher, &logger)

    cmd := RegisterUserCommand{
        Email:    "test@example.com",
        Username: "testuser",
        Password: "Password123!",
    }

    // Mock expectations
    email, _ := identity.NewEmail(cmd.Email)
    mockRepo.On("FindByEmail", mock.Anything, email).
        Return(nil, identity.ErrUserNotFound)
    mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*identity.User")).
        Return(nil)
    mockPublisher.On("Publish", mock.Anything, mock.Anything).
        Return(nil)
    mockAuthService.On("Login", mock.Anything, mock.Anything).
        Return(dto.AuthResponseDTO{}, nil)

    // Act
    result, err := handler.Handle(context.Background(), cmd)

    // Assert
    require.NoError(t, err)
    assert.NotEmpty(t, result.User.ID)
    mockRepo.AssertExpectations(t)
    mockPublisher.AssertExpectations(t)
}
```

### Query Handler Testing

```go
func TestGetUserHandler_Handle_Success(t *testing.T) {
    // Arrange
    mockRepo := new(MockUserRepository)
    handler := NewGetUserHandler(mockRepo)

    userID := uuid.New()
    user := createTestUser(t, userID)

    mockRepo.On("FindByID", mock.Anything, mock.Anything).
        Return(user, nil)

    query := GetUserQuery{UserID: userID.String()}

    // Act
    result, err := handler.Handle(context.Background(), query)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, userID.String(), result.ID)
    mockRepo.AssertExpectations(t)
}
```

### Application Service Testing

```go
func TestAuthService_Login_Success(t *testing.T) {
    // Arrange
    mockUsers := new(MockUserRepository)
    mockSessions := new(MockSessionStore)
    mockJWT := new(MockJWTService)
    mockRefresh := new(MockRefreshTokenService)
    mockBlacklist := new(MockTokenBlacklist)
    logger := zerolog.Nop()

    service := NewAuthServiceImpl(
        mockUsers, mockSessions, mockJWT,
        mockRefresh, mockBlacklist, &logger,
    )

    // Create test user
    user := createTestUser(t, uuid.New())
    email, _ := identity.NewEmail("test@example.com")

    // Mock expectations
    mockUsers.On("FindByEmail", mock.Anything, email).Return(user, nil)
    mockJWT.On("GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
        Return("access-token", nil)
    mockRefresh.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
        Return("refresh-token", &RefreshTokenMetadata{}, nil)
    mockSessions.On("Create", mock.Anything, mock.Anything).Return(nil)

    req := dto.LoginDTO{
        Identifier: "test@example.com",
        Password:   "Password123!",
        IP:         "127.0.0.1",
        UserAgent:  "test-agent",
    }

    // Act
    result, err := service.Login(context.Background(), req)

    // Assert
    require.NoError(t, err)
    assert.NotEmpty(t, result.Tokens.AccessToken)
    assert.NotEmpty(t, result.Tokens.RefreshToken)
}
```

## Best Practices

### 1. Keep Handlers Thin

Handlers should orchestrate, not implement business logic:

```go
// Bad - business logic in handler
func (h *RegisterUserHandler) Handle(ctx context.Context, cmd RegisterUserCommand) error {
    user := &identity.User{}
    user.Email = cmd.Email
    user.Status = "active"
    // ...
}

// Good - delegate to domain
func (h *RegisterUserHandler) Handle(ctx context.Context, cmd RegisterUserCommand) error {
    email, _ := identity.NewEmail(cmd.Email)
    username, _ := identity.NewUsername(cmd.Username)
    passwordHash, _ := identity.HashPassword(cmd.Password)

    user, err := identity.NewUser(email, username, passwordHash)
    // ...
}
```

### 2. Idempotent Commands

Commands should be idempotent when possible:

```go
func (h *DeleteUserHandler) Handle(ctx context.Context, cmd DeleteUserCommand) error {
    user, err := h.users.FindByID(ctx, userID)
    if err != nil {
        if errors.Is(err, identity.ErrUserNotFound) {
            return nil // Already deleted - idempotent
        }
        return err
    }

    return h.users.Delete(ctx, userID)
}
```

### 3. Fail Fast

Validate early and fail fast:

```go
func (h *UpdateProfileHandler) Handle(ctx context.Context, cmd UpdateProfileCommand) error {
    // Validate all inputs first
    userID, err := identity.ParseUserID(cmd.UserID)
    if err != nil {
        return fmt.Errorf("invalid user id: %w", err)
    }

    if cmd.DisplayName != nil && len(*cmd.DisplayName) > 100 {
        return fmt.Errorf("display name too long")
    }

    // Then load aggregate
    user, err := h.users.FindByID(ctx, userID)
    // ...
}
```

### 4. Log Significant Operations

Always log command execution:

```go
h.logger.Info().
    Str("user_id", user.ID().String()).
    Str("command", "RegisterUser").
    Msg("user registered successfully")
```

### 5. Never Return Domain Objects

Always convert to DTOs before returning:

```go
// Bad - leaks domain object
func (h *GetUserHandler) Handle(ctx context.Context, q GetUserQuery) (*identity.User, error) {
    return h.users.FindByID(ctx, userID)
}

// Good - returns DTO
func (h *GetUserHandler) Handle(ctx context.Context, q GetUserQuery) (dto.UserDTO, error) {
    user, err := h.users.FindByID(ctx, userID)
    if err != nil {
        return dto.UserDTO{}, err
    }
    return dto.FromDomain(user), nil
}
```

## Sprint 4 Implementation Plan

### Phase 1: Commands (Write Operations)
1. RegisterUserCommand + Handler
2. LoginUserCommand + Handler (delegates to AuthService)
3. RefreshTokenCommand + Handler (delegates to AuthService)
4. LogoutUserCommand + Handler (delegates to AuthService)
5. ChangePasswordCommand + Handler
6. UpdateProfileCommand + Handler

### Phase 2: Queries (Read Operations)
1. GetUserQuery + Handler
2. ListUsersQuery + Handler
3. GetActiveSessionsQuery + Handler

### Phase 3: Application Services
1. AuthServiceImpl (implements AuthService interface)
2. Integration tests with mocked infrastructure

### Phase 4: HTTP Layer
1. HTTP handlers that delegate to commands/queries
2. JWT middleware using AuthService.ValidateToken
3. E2E tests with Postman/Newman

## Common Pitfalls

### 1. Business Logic in Application Layer

**Wrong**: Implementing domain logic in handlers
```go
// Bad - domain logic in handler
if user.Role == "admin" && user.Status == "active" {
    user.CanModerate = true
}
```

**Right**: Use domain methods
```go
// Good - domain method encapsulates logic
if user.CanModerate() {
    // ...
}
```

### 2. Tight Coupling to Infrastructure

**Wrong**: Direct infrastructure dependencies
```go
// Bad - imports infrastructure
import "github.com/redis/go-redis/v9"

func (h *Handler) Handle(ctx context.Context, cmd Command) error {
    redisClient.Set(ctx, "key", "value", 0)
}
```

**Right**: Use interfaces
```go
// Good - depends on interface
type SessionStore interface {
    Create(ctx context.Context, session Session) error
}

func (h *Handler) Handle(ctx context.Context, cmd Command) error {
    h.sessions.Create(ctx, session)
}
```

### 3. Not Publishing Events

**Wrong**: Forgetting to publish domain events
```go
user.Suspend("policy violation")
h.users.Save(ctx, user)
// Events never published!
```

**Right**: Always publish after save
```go
user.Suspend("policy violation")
h.users.Save(ctx, user)

for _, event := range user.Events() {
    h.eventPublisher.Publish(ctx, event)
}
user.ClearEvents()
```

## See Also

- Domain layer: `internal/domain/identity/`
- Infrastructure implementations: `internal/infrastructure/`
- HTTP handlers: `internal/interfaces/http/`
- Application layer guide: `internal/application/CLAUDE.md`
- API security: `claude/api_security.md`
