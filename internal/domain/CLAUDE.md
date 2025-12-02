# Domain Layer Guide

> This is the core business logic layer. **NO external dependencies allowed.**

## Key Rules

1. **Only import stdlib** - Never import infrastructure packages (`database/sql`, `redis`, etc.)
2. **Entities via factories** - Always use `NewXxx()` functions that validate invariants
3. **Value objects are immutable** - Validate in constructor, no setters
4. **Aggregates own consistency** - All modifications through root methods
5. **Repository interfaces here** - Implementations go in `infrastructure/`

## Structure Per Bounded Context

```
internal/domain/{context}/
├── {entity}.go           # Aggregate root or entity
├── {entity}_id.go        # ID value object
├── {value_object}.go     # Value objects
├── repository.go         # Repository interface
├── services.go           # Domain service interfaces
├── events.go             # Domain events
└── errors.go             # Domain-specific errors
```

## Patterns

### Entity Factory

```go
func NewUser(email Email, username Username) (*User, error) {
    if email.IsEmpty() {
        return nil, ErrEmailRequired
    }
    return &User{
        id:        NewUserID(),
        email:     email,
        username:  username,
        createdAt: time.Now().UTC(),
    }, nil
}
```

### Value Object

```go
type Email struct{ value string }

func NewEmail(v string) (Email, error) {
    v = strings.TrimSpace(strings.ToLower(v))
    if v == "" { return Email{}, ErrEmailEmpty }
    if !emailRegex.MatchString(v) { return Email{}, ErrEmailInvalid }
    return Email{value: v}, nil
}
```

### Repository Interface

```go
type UserRepository interface {
    FindByID(ctx context.Context, id UserID) (*User, error)
    Save(ctx context.Context, user *User) error
}
```

## See Also

- Full patterns: `claude/architecture.md`
- Coding standards: `claude/coding.md`
