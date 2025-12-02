# Infrastructure Layer Guide

> External service implementations: databases, storage, security, messaging.

## Key Rules

1. **Implement domain interfaces** - Repository interfaces defined in domain layer
2. **Map to/from domain types** - Use internal models for persistence
3. **Wrap all errors** - Add context: `fmt.Errorf("find user: %w", err)`
4. **Convert domain errors** - Map `sql.ErrNoRows` to `ErrUserNotFound`
5. **Never leak infrastructure types** - Return domain types only

## Structure

```
internal/infrastructure/
├── persistence/
│   ├── postgres/
│   │   ├── connection.go
│   │   ├── user_repository.go
│   │   ├── image_repository.go
│   │   └── migrations/
│   └── redis/
│       ├── connection.go
│       ├── session_store.go
│       └── cache.go
├── storage/
│   ├── storage.go         # Interface
│   ├── local/
│   └── s3/
├── security/
│   ├── jwt/
│   ├── oauth/
│   └── clamav/
└── messaging/
    └── event_publisher.go
```

## Repository Implementation Pattern

```go
// Internal persistence model
type userModel struct {
    ID        string    `db:"id"`
    Email     string    `db:"email"`
    Username  string    `db:"username"`
    CreatedAt time.Time `db:"created_at"`
}

func (m *userModel) toDomain() (*identity.User, error) {
    // Map to domain aggregate
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id identity.UserID) (*identity.User, error) {
    var model userModel
    err := r.db.GetContext(ctx, &model, "SELECT * FROM users WHERE id = $1", id.String())
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fmt.Errorf("user %s: %w", id, identity.ErrUserNotFound)
        }
        return nil, fmt.Errorf("find user by id: %w", err)
    }
    return model.toDomain()
}
```

## Migrations

Location: `persistence/postgres/migrations/`

```bash
# Create new migration
goose create add_users_table sql

# Run migrations
make migrate-up
```

## See Also

- Domain interfaces: `claude/architecture.md`
- Security details: `claude/api_security.md`
