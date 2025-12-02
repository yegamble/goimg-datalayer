# Domain Layer Guide

> Core business logic layer. **NO external dependencies allowed.**

## Critical Rules

1. **Only import stdlib** - Never import infrastructure packages (`database/sql`, `redis`, `github.com/lib/pq`, etc.)
2. **Entities via factories** - Always use `NewXxx()` functions that validate invariants
3. **Value objects are immutable** - Validate in constructor, no setters
4. **Aggregates own consistency** - All modifications through root methods
5. **Repository interfaces here** - Implementations go in `infrastructure/`
6. **Test coverage: 90%** - This is the most critical layer to test

## Structure Per Bounded Context

```
internal/domain/{context}/
├── {entity}.go           # Aggregate root or entity
├── {entity}_id.go        # ID value object (UUID-based)
├── {value_object}.go     # Value objects (Email, Username, etc.)
├── repository.go         # Repository interface
├── services.go           # Domain service interfaces
├── events.go             # Domain events
└── errors.go             # Domain-specific errors
```

## Bounded Contexts in This Project

- **identity**: Users, authentication, roles, permissions
- **media**: Images, albums, metadata, EXIF data
- **moderation**: Content review, flagging, approval workflows
- **social**: Follows, favorites, comments, likes

## Patterns

### 1. Entity Factory with Validation

```go
package identity

import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    id           UserID
    email        Email
    username     Username
    passwordHash PasswordHash
    role         Role
    status       UserStatus
    createdAt    time.Time
    updatedAt    time.Time
    events       []DomainEvent
}

// Factory enforces invariants at creation
func NewUser(email Email, username Username, passwordHash PasswordHash) (*User, error) {
    if email.IsEmpty() {
        return nil, ErrEmailRequired
    }
    if username.IsEmpty() {
        return nil, ErrUsernameRequired
    }
    if passwordHash.IsEmpty() {
        return nil, ErrPasswordRequired
    }

    user := &User{
        id:           NewUserID(),
        email:        email,
        username:     username,
        passwordHash: passwordHash,
        role:         RoleUser,
        status:       UserStatusActive,
        createdAt:    time.Now().UTC(),
        updatedAt:    time.Now().UTC(),
        events:       []DomainEvent{},
    }

    user.AddEvent(UserCreatedEvent{
        UserID:    user.id,
        Email:     user.email,
        Username:  user.username,
        CreatedAt: user.createdAt,
    })

    return user, nil
}

// Reconstitution from persistence (no validation)
func ReconstructUser(
    id UserID,
    email Email,
    username Username,
    passwordHash PasswordHash,
    role Role,
    status UserStatus,
    createdAt, updatedAt time.Time,
) *User {
    return &User{
        id:           id,
        email:        email,
        username:     username,
        passwordHash: passwordHash,
        role:         role,
        status:       status,
        createdAt:    createdAt,
        updatedAt:    updatedAt,
        events:       []DomainEvent{},
    }
}

// Business methods enforce consistency
func (u *User) ChangeEmail(email Email) error {
    if email.IsEmpty() {
        return ErrEmailRequired
    }
    if u.email.Equals(email) {
        return nil // No-op
    }

    u.email = email
    u.updatedAt = time.Now().UTC()
    u.AddEvent(UserEmailChangedEvent{
        UserID:   u.id,
        NewEmail: email,
        ChangedAt: u.updatedAt,
    })
    return nil
}

func (u *User) Suspend(reason string) error {
    if u.status == UserStatusSuspended {
        return ErrUserAlreadySuspended
    }
    u.status = UserStatusSuspended
    u.updatedAt = time.Now().UTC()
    u.AddEvent(UserSuspendedEvent{
        UserID:    u.id,
        Reason:    reason,
        SuspendedAt: u.updatedAt,
    })
    return nil
}
```

### 2. Value Objects (Immutable)

```go
package identity

import (
    "regexp"
    "strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type Email struct {
    value string
}

func NewEmail(v string) (Email, error) {
    v = strings.TrimSpace(strings.ToLower(v))
    if v == "" {
        return Email{}, ErrEmailEmpty
    }
    if len(v) > 255 {
        return Email{}, ErrEmailTooLong
    }
    if !emailRegex.MatchString(v) {
        return Email{}, ErrEmailInvalid
    }
    return Email{value: v}, nil
}

func (e Email) String() string {
    return e.value
}

func (e Email) IsEmpty() bool {
    return e.value == ""
}

func (e Email) Equals(other Email) bool {
    return e.value == other.value
}
```

### 3. ID Value Objects (UUID-based)

```go
package identity

import (
    "fmt"
    "github.com/google/uuid"
)

type UserID struct {
    value uuid.UUID
}

func NewUserID() UserID {
    return UserID{value: uuid.New()}
}

func ParseUserID(s string) (UserID, error) {
    id, err := uuid.Parse(s)
    if err != nil {
        return UserID{}, fmt.Errorf("invalid user id: %w", err)
    }
    return UserID{value: id}, nil
}

func (id UserID) String() string {
    return id.value.String()
}

func (id UserID) IsZero() bool {
    return id.value == uuid.Nil
}

func (id UserID) Equals(other UserID) bool {
    return id.value == other.value
}
```

### 4. Repository Interface (Context-Aware)

```go
package identity

import "context"

type UserRepository interface {
    // Queries
    FindByID(ctx context.Context, id UserID) (*User, error)
    FindByEmail(ctx context.Context, email Email) (*User, error)
    FindByUsername(ctx context.Context, username Username) (*User, error)
    List(ctx context.Context, filter UserFilter, pagination Pagination) ([]*User, int, error)

    // Commands
    Save(ctx context.Context, user *User) error
    Delete(ctx context.Context, id UserID) error

    // Transactions (used by application layer)
    WithTx(ctx context.Context) (UserRepository, error)
    CommitTx(ctx context.Context) error
    RollbackTx(ctx context.Context) error
}

type UserFilter struct {
    Role   *Role
    Status *UserStatus
    Search string // Email or username prefix
}

type Pagination struct {
    Offset int
    Limit  int
}
```

### 5. Domain Events

```go
package identity

import (
    "time"
)

type DomainEvent interface {
    EventType() string
    OccurredAt() time.Time
}

type UserCreatedEvent struct {
    UserID    UserID
    Email     Email
    Username  Username
    CreatedAt time.Time
}

func (e UserCreatedEvent) EventType() string {
    return "identity.user.created"
}

func (e UserCreatedEvent) OccurredAt() time.Time {
    return e.CreatedAt
}

// Entity methods for event management
func (u *User) AddEvent(event DomainEvent) {
    u.events = append(u.events, event)
}

func (u *User) Events() []DomainEvent {
    return u.events
}

func (u *User) ClearEvents() {
    u.events = []DomainEvent{}
}
```

### 6. Domain Services (Stateless)

Use domain services when business logic:
- Spans multiple aggregates
- Doesn't naturally belong to any single entity
- Requires external data (via repository)

```go
package identity

type UniqueEmailChecker interface {
    IsEmailUnique(ctx context.Context, email Email, excludeUserID *UserID) (bool, error)
}

// Implementation lives in application layer or infrastructure
```

### 7. Domain Errors

```go
package identity

import "errors"

var (
    // Validation errors
    ErrEmailEmpty        = errors.New("email cannot be empty")
    ErrEmailInvalid      = errors.New("email format is invalid")
    ErrEmailTooLong      = errors.New("email exceeds 255 characters")
    ErrUsernameRequired  = errors.New("username is required")
    ErrPasswordRequired  = errors.New("password is required")

    // Business rule violations
    ErrUserNotFound           = errors.New("user not found")
    ErrUserAlreadyExists      = errors.New("user already exists")
    ErrUserAlreadySuspended   = errors.New("user is already suspended")
    ErrInsufficientPermission = errors.New("insufficient permission")
)
```

## Testing Requirements

### Coverage Target: 90%

Domain logic is the most critical layer. Test:

1. **Entity factories** - Valid and invalid inputs
2. **Value object validation** - Boundary conditions
3. **Business methods** - State transitions, invariants
4. **Domain events** - Correct events emitted
5. **Edge cases** - Concurrent modifications, boundary values

### Test Patterns

```go
package identity_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewEmail(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        input   string
        wantErr error
    }{
        {"valid email", "user@example.com", nil},
        {"empty", "", identity.ErrEmailEmpty},
        {"no @", "notanemail", identity.ErrEmailInvalid},
        {"no domain", "user@", identity.ErrEmailInvalid},
        {"whitespace trimmed", "  user@example.com  ", nil},
        {"uppercase normalized", "User@Example.COM", nil},
        {"too long", strings.Repeat("a", 250) + "@test.com", identity.ErrEmailTooLong},
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            email, err := identity.NewEmail(tt.input)

            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
                assert.True(t, email.IsEmpty())
            } else {
                require.NoError(t, err)
                assert.False(t, email.IsEmpty())
            }
        })
    }
}

func TestUser_ChangeEmail(t *testing.T) {
    t.Parallel()

    // Arrange
    oldEmail, _ := identity.NewEmail("old@example.com")
    newEmail, _ := identity.NewEmail("new@example.com")
    username, _ := identity.NewUsername("testuser")
    password, _ := identity.HashPassword("password123")

    user, err := identity.NewUser(oldEmail, username, password)
    require.NoError(t, err)
    user.ClearEvents() // Clear creation event

    // Act
    err = user.ChangeEmail(newEmail)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, newEmail, user.Email())
    assert.Len(t, user.Events(), 1)

    event := user.Events()[0]
    assert.Equal(t, "identity.user.email_changed", event.EventType())
}
```

## Common Anti-Patterns to Avoid

1. **Anemic domain model** - Don't create entities with only getters/setters. Encapsulate behavior.
2. **Infrastructure leakage** - Never import `database/sql`, `redis`, HTTP libraries
3. **Primitive obsession** - Use value objects instead of raw strings/ints
4. **Missing validation** - Always validate in constructors
5. **Public setters** - Expose business methods, not raw setters

## Agent Responsibilities

- **senior-go-architect**: Reviews domain modeling decisions, aggregate boundaries
- **test-strategist**: Ensures 90% coverage, reviews test quality
- **All agents**: Must never violate the "no infrastructure imports" rule

## See Also

- Full DDD patterns: `claude/architecture.md`
- Coding standards: `claude/coding.md`
- Application layer integration: `internal/application/CLAUDE.md`
