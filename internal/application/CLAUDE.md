# Application Layer Guide

> Use cases, commands, queries, and application services. Orchestrates domain objects.

## Critical Rules

1. **Import domain only** - Never import infrastructure or interfaces layers
2. **Commands for writes** - State-changing operations return modified aggregates
3. **Queries for reads** - Read-only operations, no side effects
4. **Create value objects** - Convert primitives from DTOs to domain types
5. **Publish events after save** - Never before persistence succeeds
6. **Manage transactions** - This layer controls transaction boundaries
7. **Test coverage: 85%** - Application logic must be thoroughly tested

## Structure

```
internal/application/{context}/
├── commands/
│   ├── create_user.go        # Handler + command struct
│   ├── update_user.go
│   └── delete_user.go
├── queries/
│   ├── get_user.go           # Handler + query struct
│   └── list_users.go
├── services/
│   └── auth_service.go       # Application services
└── dto/
    └── user_dto.go           # Application-level DTOs (if needed)
```

## Command Pattern (CQRS-Lite)

### Command Structure

```go
package commands

import (
    "context"
    "fmt"

    "goimg-datalayer/internal/domain/identity"
)

// Command struct (input)
type CreateUserCommand struct {
    Email    string
    Username string
    Password string
}

// Handler dependencies via constructor injection
type CreateUserHandler struct {
    users     identity.UserRepository
    publisher EventPublisher
    logger    *zerolog.Logger
}

func NewCreateUserHandler(
    users identity.UserRepository,
    publisher EventPublisher,
    logger *zerolog.Logger,
) *CreateUserHandler {
    return &CreateUserHandler{
        users:     users,
        publisher: publisher,
        logger:    logger,
    }
}

// Handle executes the use case
func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) (*identity.User, error) {
    // 1. Convert primitives to value objects (validation happens here)
    email, err := identity.NewEmail(cmd.Email)
    if err != nil {
        return nil, fmt.Errorf("invalid email: %w", err)
    }

    username, err := identity.NewUsername(cmd.Username)
    if err != nil {
        return nil, fmt.Errorf("invalid username: %w", err)
    }

    // 2. Check business rules (uniqueness)
    existingByEmail, err := h.users.FindByEmail(ctx, email)
    if err != nil && !errors.Is(err, identity.ErrUserNotFound) {
        return nil, fmt.Errorf("check email uniqueness: %w", err)
    }
    if existingByEmail != nil {
        return nil, identity.ErrUserAlreadyExists
    }

    existingByUsername, err := h.users.FindByUsername(ctx, username)
    if err != nil && !errors.Is(err, identity.ErrUserNotFound) {
        return nil, fmt.Errorf("check username uniqueness: %w", err)
    }
    if existingByUsername != nil {
        return nil, identity.ErrUsernameAlreadyTaken
    }

    // 3. Hash password (application layer concern)
    passwordHash, err := identity.HashPassword(cmd.Password)
    if err != nil {
        return nil, fmt.Errorf("hash password: %w", err)
    }

    // 4. Create aggregate via domain factory
    user, err := identity.NewUser(email, username, passwordHash)
    if err != nil {
        return nil, fmt.Errorf("create user: %w", err)
    }

    // 5. Persist (transaction managed here if needed)
    if err := h.users.Save(ctx, user); err != nil {
        return nil, fmt.Errorf("save user: %w", err)
    }

    // 6. Publish domain events AFTER successful save
    for _, event := range user.Events() {
        if err := h.publisher.Publish(ctx, event); err != nil {
            // Log but don't fail the operation
            h.logger.Error().Err(err).
                Str("event_type", event.EventType()).
                Msg("failed to publish domain event")
        }
    }
    user.ClearEvents()

    h.logger.Info().
        Str("user_id", user.ID().String()).
        Str("email", user.Email().String()).
        Msg("user created successfully")

    return user, nil
}
```

### Command with Transaction

```go
package commands

import (
    "context"
    "fmt"
)

type UpdateUserProfileCommand struct {
    UserID   string
    Email    *string // Optional updates
    Username *string
    Bio      *string
}

type UpdateUserProfileHandler struct {
    users     identity.UserRepository
    publisher EventPublisher
    logger    *zerolog.Logger
}

func (h *UpdateUserProfileHandler) Handle(
    ctx context.Context,
    cmd UpdateUserProfileCommand,
) (*identity.User, error) {
    // 1. Parse ID
    userID, err := identity.ParseUserID(cmd.UserID)
    if err != nil {
        return nil, fmt.Errorf("invalid user id: %w", err)
    }

    // 2. Start transaction
    txRepo, err := h.users.WithTx(ctx)
    if err != nil {
        return nil, fmt.Errorf("begin transaction: %w", err)
    }
    defer func() {
        if err != nil {
            _ = txRepo.RollbackTx(ctx)
        }
    }()

    // 3. Load aggregate
    user, err := txRepo.FindByID(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("find user: %w", err)
    }

    // 4. Apply changes via domain methods
    if cmd.Email != nil {
        email, err := identity.NewEmail(*cmd.Email)
        if err != nil {
            return nil, fmt.Errorf("invalid email: %w", err)
        }
        if err := user.ChangeEmail(email); err != nil {
            return nil, err
        }
    }

    if cmd.Username != nil {
        username, err := identity.NewUsername(*cmd.Username)
        if err != nil {
            return nil, fmt.Errorf("invalid username: %w", err)
        }
        if err := user.ChangeUsername(username); err != nil {
            return nil, err
        }
    }

    // 5. Save changes
    if err := txRepo.Save(ctx, user); err != nil {
        return nil, fmt.Errorf("save user: %w", err)
    }

    // 6. Commit transaction
    if err := txRepo.CommitTx(ctx); err != nil {
        return nil, fmt.Errorf("commit transaction: %w", err)
    }

    // 7. Publish events after commit
    for _, event := range user.Events() {
        if err := h.publisher.Publish(ctx, event); err != nil {
            h.logger.Error().Err(err).Msg("failed to publish event")
        }
    }
    user.ClearEvents()

    return user, nil
}
```

## Query Pattern (Read-Only)

### Simple Query

```go
package queries

import (
    "context"
    "fmt"

    "goimg-datalayer/internal/domain/identity"
)

type GetUserQuery struct {
    UserID string
}

type GetUserHandler struct {
    users identity.UserRepository
}

func NewGetUserHandler(users identity.UserRepository) *GetUserHandler {
    return &GetUserHandler{users: users}
}

func (h *GetUserHandler) Handle(ctx context.Context, q GetUserQuery) (*identity.User, error) {
    userID, err := identity.ParseUserID(q.UserID)
    if err != nil {
        return nil, fmt.Errorf("invalid user id: %w", err)
    }

    user, err := h.users.FindByID(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("find user: %w", err)
    }

    return user, nil
}
```

### List Query with Pagination

```go
package queries

import (
    "context"
    "fmt"

    "goimg-datalayer/internal/domain/identity"
)

type ListUsersQuery struct {
    Role   *string
    Status *string
    Search string
    Offset int
    Limit  int
}

type ListUsersResult struct {
    Users      []*identity.User
    TotalCount int
}

type ListUsersHandler struct {
    users identity.UserRepository
}

func (h *ListUsersHandler) Handle(ctx context.Context, q ListUsersQuery) (*ListUsersResult, error) {
    // Convert to domain filter
    filter := identity.UserFilter{
        Search: q.Search,
    }

    if q.Role != nil {
        role, err := identity.ParseRole(*q.Role)
        if err != nil {
            return nil, fmt.Errorf("invalid role: %w", err)
        }
        filter.Role = &role
    }

    if q.Status != nil {
        status, err := identity.ParseUserStatus(*q.Status)
        if err != nil {
            return nil, fmt.Errorf("invalid status: %w", err)
        }
        filter.Status = &status
    }

    pagination := identity.Pagination{
        Offset: q.Offset,
        Limit:  q.Limit,
    }

    users, total, err := h.users.List(ctx, filter, pagination)
    if err != nil {
        return nil, fmt.Errorf("list users: %w", err)
    }

    return &ListUsersResult{
        Users:      users,
        TotalCount: total,
    }, nil
}
```

## Application Services

Use application services for:
- Complex workflows spanning multiple aggregates
- Integration with external services
- Coordination logic that doesn't fit in a single command/query

```go
package services

import (
    "context"
    "fmt"
    "time"

    "goimg-datalayer/internal/domain/identity"
)

type AuthService struct {
    users        identity.UserRepository
    sessions     SessionStore
    jwtSecret    string
    tokenExpiry  time.Duration
}

func NewAuthService(
    users identity.UserRepository,
    sessions SessionStore,
    jwtSecret string,
    tokenExpiry time.Duration,
) *AuthService {
    return &AuthService{
        users:       users,
        sessions:    sessions,
        jwtSecret:   jwtSecret,
        tokenExpiry: tokenExpiry,
    }
}

func (s *AuthService) Authenticate(
    ctx context.Context,
    email, password string,
) (string, error) {
    // 1. Convert to value object
    emailVO, err := identity.NewEmail(email)
    if err != nil {
        return "", fmt.Errorf("invalid email: %w", err)
    }

    // 2. Find user
    user, err := s.users.FindByEmail(ctx, emailVO)
    if err != nil {
        if errors.Is(err, identity.ErrUserNotFound) {
            return "", identity.ErrInvalidCredentials
        }
        return "", fmt.Errorf("find user: %w", err)
    }

    // 3. Verify password (domain method)
    if !user.VerifyPassword(password) {
        return "", identity.ErrInvalidCredentials
    }

    // 4. Check if user can login
    if !user.CanLogin() {
        return "", identity.ErrUserSuspended
    }

    // 5. Generate JWT (infrastructure concern, but called from application)
    token, err := GenerateJWT(user.ID(), user.Role(), s.jwtSecret, s.tokenExpiry)
    if err != nil {
        return "", fmt.Errorf("generate token: %w", err)
    }

    // 6. Store session
    session := identity.NewSession(user.ID(), token, s.tokenExpiry)
    if err := s.sessions.Save(ctx, session); err != nil {
        // Non-critical, log but continue
        log.Warn().Err(err).Msg("failed to store session")
    }

    return token, nil
}
```

## Event Publisher Interface

```go
package application

import (
    "context"
    "goimg-datalayer/internal/domain/identity"
)

// EventPublisher is defined in application layer
// Implemented in infrastructure layer (using asynq, Redis, etc.)
type EventPublisher interface {
    Publish(ctx context.Context, event interface{}) error
}
```

## Error Handling

### Wrap All Errors

```go
if err != nil {
    return fmt.Errorf("descriptive context: %w", err)
}
```

### Convert Domain Errors

Don't let domain errors leak unchanged - add application context:

```go
user, err := h.users.FindByID(ctx, userID)
if err != nil {
    if errors.Is(err, identity.ErrUserNotFound) {
        return fmt.Errorf("user %s: %w", userID, err)
    }
    return fmt.Errorf("find user for profile update: %w", err)
}
```

## Transaction Boundaries

Guidelines for when to use transactions:

1. **Single aggregate save**: No transaction needed (atomic by nature)
2. **Multiple saves in sequence**: Use transaction
3. **Cross-aggregate consistency**: Use transaction
4. **Eventual consistency OK**: No transaction, use domain events

```go
// Need transaction: updating user and creating audit log
txRepo, _ := h.users.WithTx(ctx)
defer txRepo.RollbackTx(ctx)

// Multiple operations
txRepo.Save(ctx, user)
txRepo.SaveAuditLog(ctx, log)

txRepo.CommitTx(ctx)
```

## Testing Requirements

### Coverage Target: 85%

Test command and query handlers thoroughly:

```go
package commands_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "goimg-datalayer/internal/application/commands"
    "goimg-datalayer/internal/domain/identity"
)

// Mock repository (using testify/mock)
type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email identity.Email) (*identity.User, error) {
    args := m.Called(ctx, email)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) Save(ctx context.Context, user *identity.User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}

// Mock event publisher
type MockEventPublisher struct {
    mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, event interface{}) error {
    args := m.Called(ctx, event)
    return args.Error(0)
}

func TestCreateUserHandler_Handle_Success(t *testing.T) {
    // Arrange
    mockRepo := new(MockUserRepository)
    mockPublisher := new(MockEventPublisher)
    logger := zerolog.Nop()

    handler := commands.NewCreateUserHandler(mockRepo, mockPublisher, &logger)

    cmd := commands.CreateUserCommand{
        Email:    "test@example.com",
        Username: "testuser",
        Password: "Password123!",
    }

    // Mock expectations
    email, _ := identity.NewEmail(cmd.Email)
    username, _ := identity.NewUsername(cmd.Username)

    mockRepo.On("FindByEmail", mock.Anything, email).
        Return(nil, identity.ErrUserNotFound)
    mockRepo.On("FindByUsername", mock.Anything, username).
        Return(nil, identity.ErrUserNotFound)
    mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*identity.User")).
        Return(nil)
    mockPublisher.On("Publish", mock.Anything, mock.Anything).
        Return(nil)

    // Act
    user, err := handler.Handle(context.Background(), cmd)

    // Assert
    require.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, email, user.Email())
    assert.Equal(t, username, user.Username())

    mockRepo.AssertExpectations(t)
    mockPublisher.AssertExpectations(t)
}

func TestCreateUserHandler_Handle_DuplicateEmail(t *testing.T) {
    // Arrange
    mockRepo := new(MockUserRepository)
    mockPublisher := new(MockEventPublisher)
    logger := zerolog.Nop()

    handler := commands.NewCreateUserHandler(mockRepo, mockPublisher, &logger)

    cmd := commands.CreateUserCommand{
        Email:    "existing@example.com",
        Username: "testuser",
        Password: "Password123!",
    }

    email, _ := identity.NewEmail(cmd.Email)
    existingUser, _ := identity.NewUser(email, identity.MustUsername("existing"), identity.PasswordHash{})

    mockRepo.On("FindByEmail", mock.Anything, email).
        Return(existingUser, nil)

    // Act
    user, err := handler.Handle(context.Background(), cmd)

    // Assert
    assert.Nil(t, user)
    require.ErrorIs(t, err, identity.ErrUserAlreadyExists)

    mockRepo.AssertExpectations(t)
    mockPublisher.AssertNotCalled(t, "Publish")
}
```

## Common Patterns

### 1. Idempotent Commands

```go
func (h *DeleteUserHandler) Handle(ctx context.Context, cmd DeleteUserCommand) error {
    userID, err := identity.ParseUserID(cmd.UserID)
    if err != nil {
        return fmt.Errorf("invalid user id: %w", err)
    }

    user, err := h.users.FindByID(ctx, userID)
    if err != nil {
        if errors.Is(err, identity.ErrUserNotFound) {
            // Already deleted - idempotent
            return nil
        }
        return fmt.Errorf("find user: %w", err)
    }

    return h.users.Delete(ctx, userID)
}
```

### 2. Optimistic Locking

```go
type UpdateImageCommand struct {
    ImageID string
    Title   string
    Version int // Optimistic lock version
}

func (h *UpdateImageHandler) Handle(ctx context.Context, cmd UpdateImageCommand) error {
    image, err := h.images.FindByID(ctx, imageID)
    if err != nil {
        return err
    }

    // Check version
    if image.Version() != cmd.Version {
        return media.ErrOptimisticLockFailed
    }

    // Update and increment version
    image.UpdateTitle(cmd.Title)
    return h.images.Save(ctx, image)
}
```

## Agent Responsibilities

- **senior-go-architect**: Reviews transaction boundaries, handler structure
- **backend-developer**: Implements command/query handlers
- **test-strategist**: Ensures 85% coverage with proper mocking

## See Also

- Domain layer: `internal/domain/CLAUDE.md`
- Infrastructure implementations: `internal/infrastructure/CLAUDE.md`
- HTTP integration: `internal/interfaces/http/CLAUDE.md`
- Architecture guide: `claude/architecture.md`
