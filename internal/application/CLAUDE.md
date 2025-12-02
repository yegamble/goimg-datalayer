# Application Layer Guide

> Use cases, commands, queries, and application services. Orchestrates domain objects.

## Key Rules

1. **Import domain only** - Never import infrastructure or interfaces
2. **Commands for writes** - State-changing operations
3. **Queries for reads** - Read-only operations
4. **Create value objects** - Convert primitives to domain types
5. **Publish events after save** - Not before persistence succeeds

## Structure

```
internal/application/{context}/
├── commands/
│   ├── create_user.go
│   ├── update_user.go
│   └── delete_user.go
├── queries/
│   ├── get_user.go
│   └── list_users.go
└── services/
    └── auth_service.go
```

## Command Handler Pattern

```go
type CreateUserHandler struct {
    users     identity.UserRepository
    publisher EventPublisher
}

func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) (*identity.User, error) {
    // 1. Create value objects from primitives
    email, err := identity.NewEmail(cmd.Email)
    if err != nil {
        return nil, err
    }

    // 2. Check business rules
    existing, _ := h.users.FindByEmail(ctx, email)
    if existing != nil {
        return nil, identity.ErrUserAlreadyExists
    }

    // 3. Create aggregate via factory
    user, err := identity.NewUser(email, username, hash)
    if err != nil {
        return nil, err
    }

    // 4. Persist
    if err := h.users.Save(ctx, user); err != nil {
        return nil, fmt.Errorf("save user: %w", err)
    }

    // 5. Publish domain events
    for _, event := range user.Events() {
        h.publisher.Publish(ctx, event)
    }

    return user, nil
}
```

## Query Handler Pattern

```go
type GetUserHandler struct {
    users identity.UserRepository
}

func (h *GetUserHandler) Handle(ctx context.Context, q GetUserQuery) (*identity.User, error) {
    id, err := identity.ParseUserID(q.UserID)
    if err != nil {
        return nil, err
    }
    return h.users.FindByID(ctx, id)
}
```

## See Also

- Domain patterns: `claude/architecture.md`
- Handler integration: `claude/api_security.md`
