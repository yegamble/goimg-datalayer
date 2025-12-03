# PostgreSQL Persistence Layer

This directory implements the persistence layer for the Identity bounded context using PostgreSQL with sqlx.

## Architecture Principles

1. **Repository Pattern**: Implements domain repository interfaces from `internal/domain/identity/repository.go`
2. **No Business Logic**: Pure data access layer - all business rules live in domain layer
3. **Error Mapping**: Translates database errors to domain errors
4. **Prepared Statements**: All queries use parameterized statements for security
5. **Soft Deletes**: Users are soft-deleted by setting `deleted_at` timestamp

## Files

### db.go
Database connection pool management with sqlx:
- `Config` struct with sensible defaults
- `NewDB()` creates connection pool with configurable parameters
- `HealthCheck()` verifies database connectivity
- Connection pool settings:
  - Max open connections: 25
  - Max idle connections: 5
  - Connection max lifetime: 30 minutes
  - Connection max idle time: 10 minutes

### user_repository.go
Implements `identity.UserRepository` interface:

**Methods:**
- `NextID()` - Generates new UserID using UUID v4
- `FindByID()` - Retrieves user by UUID
- `FindByEmail()` - Retrieves user by email (unique constraint)
- `FindByUsername()` - Retrieves user by username (unique constraint)
- `Save()` - Upserts user (checks existence, then insert or update)
- `Delete()` - Soft deletes user by setting deleted_at

**Error Handling:**
- Maps `sql.ErrNoRows` to `identity.ErrUserNotFound`
- Maps PostgreSQL unique constraint violations to:
  - `identity.ErrEmailExists` for duplicate emails
  - `identity.ErrUsernameExists` for duplicate usernames
- Wraps all other errors with context using `fmt.Errorf()`

**Security:**
- All queries use parameterized statements (no SQL injection)
- Password hashes stored using Argon2id (from domain layer)
- Email addresses normalized to lowercase (handled by domain Email value object)

### session_repository.go
Manages authentication sessions:

**Session Struct:**
```go
type Session struct {
    ID               uuid.UUID       // Session identifier
    UserID           identity.UserID // User who owns this session
    RefreshTokenHash string          // Hashed refresh token
    IPAddress        string          // Client IP (optional)
    UserAgent        string          // Client user agent (optional)
    ExpiresAt        time.Time       // Expiration timestamp
    CreatedAt        time.Time       // Creation timestamp
    RevokedAt        *time.Time      // Revocation timestamp (NULL if active)
}
```

**Methods:**
- `Create()` - Creates new session
- `GetByID()` - Retrieves session by ID
- `GetByUserID()` - Retrieves all active sessions for a user
- `Revoke()` - Revokes session by setting revoked_at
- `DeleteExpired()` - Cleanup job - deletes expired sessions and old revoked sessions (30+ days)

**Indexes:**
- `idx_sessions_user_id` - Fast lookups by user
- `idx_sessions_refresh_token_hash` - Fast token validation
- `idx_sessions_expires_at` - Efficient cleanup queries (partial index, excludes revoked)

## Database Schema

### users table
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    display_name VARCHAR(100) NOT NULL DEFAULT '',
    bio VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
```

**Indexes:**
- `idx_users_email` - Unique email lookups
- `idx_users_username` - Unique username lookups
- `idx_users_status` - Filter by status (partial index, excludes soft-deleted)
- `idx_users_role` - Filter by role

### sessions table
```sql
CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);
```

## Usage Example

```go
package main

import (
    "context"
    "log"

    "github.com/yegamble/goimg-datalayer/internal/domain/identity"
    "github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
)

func main() {
    // Create database connection
    cfg := postgres.DefaultConfig()
    cfg.Host = "localhost"
    cfg.Database = "goimg"

    db, err := postgres.NewDB(cfg)
    if err != nil {
        log.Fatalf("failed to connect to database: %v", err)
    }
    defer postgres.Close(db)

    // Create repository
    userRepo := postgres.NewUserRepository(db)

    // Create user
    email, _ := identity.NewEmail("user@example.com")
    username, _ := identity.NewUsername("johndoe")
    password, _ := identity.NewPasswordHash("securePassword123")

    user, err := identity.NewUser(email, username, password)
    if err != nil {
        log.Fatalf("failed to create user: %v", err)
    }

    // Save to database
    ctx := context.Background()
    if err := userRepo.Save(ctx, user); err != nil {
        log.Fatalf("failed to save user: %v", err)
    }

    // Retrieve by email
    foundUser, err := userRepo.FindByEmail(ctx, email)
    if err != nil {
        log.Fatalf("failed to find user: %v", err)
    }

    log.Printf("Found user: %s (%s)", foundUser.Username(), foundUser.Email())
}
```

## Migrations

Migrations are managed using Goose in `/home/user/goimg-datalayer/migrations/`:

```bash
# Run pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Check migration status
make migrate-status

# Create new migration
make migrate-create NAME=add_user_profile_fields
```

## Environment Variables

Configure database connection via environment variables:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=goimg
export DB_SSL_MODE=disable
```

## Testing Strategy

### Unit Tests
- Test error mapping (SQL errors → domain errors)
- Test row conversion functions
- Mock sqlx.DB using interfaces

### Integration Tests
- Use testcontainers with real PostgreSQL
- Test full CRUD operations
- Test concurrent operations and unique constraints
- Test soft delete behavior

**Tag integration tests:**
```go
//go:build integration
```

Run with: `make test-integration`

## Performance Considerations

1. **Connection Pooling**: Configured for 25 max open connections
2. **Prepared Statements**: sqlx uses prepared statements automatically
3. **Indexes**: All lookup patterns have covering indexes
4. **Partial Indexes**: Status and expiration indexes exclude irrelevant rows
5. **Soft Deletes**: `deleted_at IS NULL` in WHERE clauses and indexes

## Security

1. **No SQL Injection**: All queries use parameterized statements
2. **Password Storage**: Argon2id hashing in domain layer (OWASP 2024 recommended params)
3. **Token Storage**: Refresh tokens stored as hashes, never plaintext
4. **Context Timeouts**: All queries use context for cancellation
5. **Unique Constraints**: Database enforces email and username uniqueness

## Common Issues

### Unique Constraint Violations
```go
err := userRepo.Save(ctx, user)
if errors.Is(err, identity.ErrEmailExists) {
    // Handle duplicate email
}
if errors.Is(err, identity.ErrUsernameExists) {
    // Handle duplicate username
}
```

### User Not Found
```go
user, err := userRepo.FindByEmail(ctx, email)
if errors.Is(err, identity.ErrUserNotFound) {
    // User doesn't exist
}
```

### Connection Issues
```go
err := postgres.HealthCheck(ctx, db)
if err != nil {
    // Database is unreachable
}
```

## Future Enhancements

- [ ] Add transaction support for multi-repository operations
- [ ] Implement read replicas for scaling reads
- [ ] Add query result caching layer (Redis)
- [ ] Implement user search with full-text search
- [ ] Add pagination support for list queries
- [ ] Implement audit logging for user changes
