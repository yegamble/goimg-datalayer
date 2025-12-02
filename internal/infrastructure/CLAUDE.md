# Infrastructure Layer Guide

> External service implementations: databases, storage, security, messaging.

## Critical Rules

1. **Implement domain interfaces** - Repository interfaces defined in domain layer
2. **Map to/from domain types** - Use internal models for persistence
3. **Wrap all errors** - Add context: `fmt.Errorf("find user: %w", err)`
4. **Convert infrastructure errors** - Map `sql.ErrNoRows` to `domain.ErrNotFound`
5. **Never leak infrastructure types** - Return domain types only (no `sql.Row`, `redis.Conn`, etc.)
6. **Test coverage: 70%** - Focus on error handling and mapping logic

## Structure

```
internal/infrastructure/
├── persistence/
│   ├── postgres/
│   │   ├── connection.go         # sqlx DB pool setup
│   │   ├── user_repository.go
│   │   ├── image_repository.go
│   │   ├── transaction.go        # Transaction wrapper
│   │   └── migrations/           # goose migrations
│   │       ├── 00001_init.sql
│   │       └── 00002_add_users.sql
│   └── redis/
│       ├── connection.go         # go-redis client setup
│       ├── session_store.go
│       ├── cache.go
│       └── rate_limiter.go
├── storage/
│   ├── storage.go                # Storage interface
│   ├── local/
│   │   └── local_storage.go
│   ├── s3/
│   │   └── s3_storage.go
│   └── ipfs/
│       └── ipfs_storage.go
├── security/
│   ├── jwt/
│   │   └── jwt_service.go        # golang-jwt/jwt
│   ├── oauth/
│   │   └── oauth_provider.go
│   └── clamav/
│       └── scanner.go
├── imageprocessing/
│   └── bimg/
│       └── processor.go          # bimg (libvips)
└── messaging/
    ├── event_publisher.go        # asynq producer
    └── worker.go                 # asynq consumer
```

## PostgreSQL with sqlx

### Connection Setup

```go
package postgres

import (
    "context"
    "fmt"
    "time"

    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
)

type Config struct {
    Host            string
    Port            int
    Database        string
    User            string
    Password        string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
    SSLMode         string
}

func NewConnection(ctx context.Context, cfg Config) (*sqlx.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
    )

    db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("connect to postgres: %w", err)
    }

    db.SetMaxOpenConns(cfg.MaxOpenConns)
    db.SetMaxIdleConns(cfg.MaxIdleConns)
    db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

    // Verify connection
    if err := db.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("ping postgres: %w", err)
    }

    return db, nil
}
```

### Repository Implementation Pattern

```go
package postgres

import (
    "context"
    "database/sql"
    "errors"
    "fmt"

    "github.com/jmoiron/sqlx"
    "goimg-datalayer/internal/domain/identity"
)

// Internal persistence model (maps to DB schema)
type userModel struct {
    ID           string         `db:"id"`
    Email        string         `db:"email"`
    Username     string         `db:"username"`
    PasswordHash string         `db:"password_hash"`
    Role         string         `db:"role"`
    Status       string         `db:"status"`
    CreatedAt    time.Time      `db:"created_at"`
    UpdatedAt    time.Time      `db:"updated_at"`
    Version      int            `db:"version"` // Optimistic locking
}

// Map to domain aggregate
func (m *userModel) toDomain() (*identity.User, error) {
    userID, err := identity.ParseUserID(m.ID)
    if err != nil {
        return nil, fmt.Errorf("parse user id: %w", err)
    }

    email, err := identity.NewEmail(m.Email)
    if err != nil {
        return nil, fmt.Errorf("parse email: %w", err)
    }

    username, err := identity.NewUsername(m.Username)
    if err != nil {
        return nil, fmt.Errorf("parse username: %w", err)
    }

    role, err := identity.ParseRole(m.Role)
    if err != nil {
        return nil, fmt.Errorf("parse role: %w", err)
    }

    status, err := identity.ParseUserStatus(m.Status)
    if err != nil {
        return nil, fmt.Errorf("parse status: %w", err)
    }

    passwordHash := identity.PasswordHashFromString(m.PasswordHash)

    return identity.ReconstructUser(
        userID,
        email,
        username,
        passwordHash,
        role,
        status,
        m.CreatedAt,
        m.UpdatedAt,
    ), nil
}

// Map from domain aggregate
func fromDomain(user *identity.User) *userModel {
    return &userModel{
        ID:           user.ID().String(),
        Email:        user.Email().String(),
        Username:     user.Username().String(),
        PasswordHash: user.PasswordHash().String(),
        Role:         user.Role().String(),
        Status:       user.Status().String(),
        CreatedAt:    user.CreatedAt(),
        UpdatedAt:    user.UpdatedAt(),
    }
}

// Repository implementation
type PostgresUserRepository struct {
    db *sqlx.DB
    tx *sqlx.Tx // nil unless in transaction
}

func NewPostgresUserRepository(db *sqlx.DB) *PostgresUserRepository {
    return &PostgresUserRepository{db: db}
}

// Get executor (db or tx)
func (r *PostgresUserRepository) executor() sqlx.ExtContext {
    if r.tx != nil {
        return r.tx
    }
    return r.db
}

func (r *PostgresUserRepository) FindByID(
    ctx context.Context,
    id identity.UserID,
) (*identity.User, error) {
    query := `SELECT * FROM users WHERE id = $1`

    var model userModel
    err := sqlx.GetContext(ctx, r.executor(), &model, query, id.String())
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fmt.Errorf("user %s: %w", id, identity.ErrUserNotFound)
        }
        return nil, fmt.Errorf("find user by id: %w", err)
    }

    user, err := model.toDomain()
    if err != nil {
        return nil, fmt.Errorf("map to domain: %w", err)
    }

    return user, nil
}

func (r *PostgresUserRepository) FindByEmail(
    ctx context.Context,
    email identity.Email,
) (*identity.User, error) {
    query := `SELECT * FROM users WHERE email = $1`

    var model userModel
    err := sqlx.GetContext(ctx, r.executor(), &model, query, email.String())
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, identity.ErrUserNotFound
        }
        return nil, fmt.Errorf("find user by email: %w", err)
    }

    return model.toDomain()
}

func (r *PostgresUserRepository) Save(
    ctx context.Context,
    user *identity.User,
) error {
    model := fromDomain(user)

    // Upsert with optimistic locking
    query := `
        INSERT INTO users (
            id, email, username, password_hash, role, status,
            created_at, updated_at, version
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, 1
        )
        ON CONFLICT (id) DO UPDATE SET
            email = EXCLUDED.email,
            username = EXCLUDED.username,
            password_hash = EXCLUDED.password_hash,
            role = EXCLUDED.role,
            status = EXCLUDED.status,
            updated_at = EXCLUDED.updated_at,
            version = users.version + 1
        WHERE users.version = $9
    `

    result, err := r.executor().ExecContext(
        ctx, query,
        model.ID, model.Email, model.Username, model.PasswordHash,
        model.Role, model.Status, model.CreatedAt, model.UpdatedAt,
        model.Version, // for WHERE clause
    )
    if err != nil {
        return fmt.Errorf("save user: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("check rows affected: %w", err)
    }
    if rows == 0 {
        return fmt.Errorf("optimistic lock failed: %w", identity.ErrConcurrentModification)
    }

    return nil
}

func (r *PostgresUserRepository) List(
    ctx context.Context,
    filter identity.UserFilter,
    pagination identity.Pagination,
) ([]*identity.User, int, error) {
    // Build dynamic query
    query, args := r.buildListQuery(filter, pagination)

    var models []userModel
    err := sqlx.SelectContext(ctx, r.executor(), &models, query, args...)
    if err != nil {
        return nil, 0, fmt.Errorf("list users: %w", err)
    }

    // Map to domain
    users := make([]*identity.User, 0, len(models))
    for _, model := range models {
        user, err := model.toDomain()
        if err != nil {
            return nil, 0, fmt.Errorf("map user %s: %w", model.ID, err)
        }
        users = append(users, user)
    }

    // Get total count
    countQuery := `SELECT COUNT(*) FROM users WHERE 1=1` // Add filters
    var total int
    err = r.executor().GetContext(ctx, &total, countQuery)
    if err != nil {
        return nil, 0, fmt.Errorf("count users: %w", err)
    }

    return users, total, nil
}

// Transaction methods
func (r *PostgresUserRepository) WithTx(ctx context.Context) (identity.UserRepository, error) {
    tx, err := r.db.BeginTxx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("begin transaction: %w", err)
    }

    return &PostgresUserRepository{
        db: r.db,
        tx: tx,
    }, nil
}

func (r *PostgresUserRepository) CommitTx(ctx context.Context) error {
    if r.tx == nil {
        return fmt.Errorf("no transaction to commit")
    }
    if err := r.tx.Commit(); err != nil {
        return fmt.Errorf("commit transaction: %w", err)
    }
    return nil
}

func (r *PostgresUserRepository) RollbackTx(ctx context.Context) error {
    if r.tx == nil {
        return nil // No-op
    }
    if err := r.tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
        return fmt.Errorf("rollback transaction: %w", err)
    }
    return nil
}
```

## Redis with go-redis

### Connection Setup

```go
package redis

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

type Config struct {
    Host         string
    Port         int
    Password     string
    DB           int
    PoolSize     int
    MinIdleConns int
    MaxRetries   int
}

func NewConnection(ctx context.Context, cfg Config) (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
        Password:     cfg.Password,
        DB:           cfg.DB,
        PoolSize:     cfg.PoolSize,
        MinIdleConns: cfg.MinIdleConns,
        MaxRetries:   cfg.MaxRetries,
    })

    // Verify connection
    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("ping redis: %w", err)
    }

    return client, nil
}
```

### Cache Pattern

```go
package redis

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

type Cache struct {
    client *redis.Client
    prefix string
    ttl    time.Duration
}

func NewCache(client *redis.Client, prefix string, ttl time.Duration) *Cache {
    return &Cache{
        client: client,
        prefix: prefix,
        ttl:    ttl,
    }
}

func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
    fullKey := c.prefix + ":" + key

    data, err := c.client.Get(ctx, fullKey).Bytes()
    if err != nil {
        if errors.Is(err, redis.Nil) {
            return ErrCacheMiss
        }
        return fmt.Errorf("redis get: %w", err)
    }

    if err := json.Unmarshal(data, dest); err != nil {
        return fmt.Errorf("unmarshal cache data: %w", err)
    }

    return nil
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}) error {
    fullKey := c.prefix + ":" + key

    data, err := json.Marshal(value)
    if err != nil {
        return fmt.Errorf("marshal cache data: %w", err)
    }

    err = c.client.Set(ctx, fullKey, data, c.ttl).Err()
    if err != nil {
        return fmt.Errorf("redis set: %w", err)
    }

    return nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
    fullKey := c.prefix + ":" + key
    return c.client.Del(ctx, fullKey).Err()
}
```

## Database Migrations (goose)

### Migration Files

Location: `internal/infrastructure/persistence/postgres/migrations/`

```sql
-- 00001_init.sql
-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_status ON users(status);

-- +goose Down
DROP TABLE IF EXISTS users;
```

### Running Migrations

```bash
# Create new migration
goose -dir internal/infrastructure/persistence/postgres/migrations create add_images_table sql

# Run migrations
make migrate-up

# Rollback
make migrate-down

# Status
goose -dir internal/infrastructure/persistence/postgres/migrations postgres "CONNECTION_STRING" status
```

## Storage Abstraction

### Interface

```go
package storage

import (
    "context"
    "io"
)

type Storage interface {
    // Store uploads a file and returns its key/path
    Store(ctx context.Context, key string, reader io.Reader, metadata Metadata) error

    // Retrieve downloads a file
    Retrieve(ctx context.Context, key string) (io.ReadCloser, error)

    // Delete removes a file
    Delete(ctx context.Context, key string) error

    // Exists checks if a file exists
    Exists(ctx context.Context, key string) (bool, error)

    // GetURL returns a public or signed URL
    GetURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

type Metadata struct {
    ContentType string
    Size        int64
    Custom      map[string]string
}
```

### S3 Implementation

```go
package s3

import (
    "context"
    "fmt"
    "io"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"

    "goimg-datalayer/internal/infrastructure/storage"
)

type S3Storage struct {
    client *s3.Client
    bucket string
    region string
}

func NewS3Storage(ctx context.Context, bucket, region string) (*S3Storage, error) {
    cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
    if err != nil {
        return nil, fmt.Errorf("load aws config: %w", err)
    }

    return &S3Storage{
        client: s3.NewFromConfig(cfg),
        bucket: bucket,
        region: region,
    }, nil
}

func (s *S3Storage) Store(
    ctx context.Context,
    key string,
    reader io.Reader,
    metadata storage.Metadata,
) error {
    _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket:      aws.String(s.bucket),
        Key:         aws.String(key),
        Body:        reader,
        ContentType: aws.String(metadata.ContentType),
    })
    if err != nil {
        return fmt.Errorf("s3 put object: %w", err)
    }

    return nil
}

// ... other methods
```

## Image Processing with bimg (libvips)

```go
package bimg

import (
    "fmt"

    "gopkg.in/h2non/bimg.v1"
)

type ImageProcessor struct {
    maxWidth  int
    maxHeight int
    quality   int
}

func NewImageProcessor(maxWidth, maxHeight, quality int) *ImageProcessor {
    return &ImageProcessor{
        maxWidth:  maxWidth,
        maxHeight: maxHeight,
        quality:   quality,
    }
}

func (p *ImageProcessor) ResizeImage(data []byte, width, height int) ([]byte, error) {
    image := bimg.NewImage(data)

    options := bimg.Options{
        Width:   width,
        Height:  height,
        Quality: p.quality,
        Crop:    false,
        Enlarge: false,
    }

    resized, err := image.Process(options)
    if err != nil {
        return nil, fmt.Errorf("resize image: %w", err)
    }

    return resized, nil
}

func (p *ImageProcessor) GenerateThumbnail(data []byte) ([]byte, error) {
    return p.ResizeImage(data, 200, 200)
}
```

## Event Publishing with asynq

```go
package messaging

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/hibiken/asynq"
)

type AsynqEventPublisher struct {
    client *asynq.Client
}

func NewAsynqEventPublisher(redisAddr string) *AsynqEventPublisher {
    client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
    return &AsynqEventPublisher{client: client}
}

func (p *AsynqEventPublisher) Publish(ctx context.Context, event interface{}) error {
    // Get event type (assuming domain events implement EventType() method)
    eventTyper, ok := event.(interface{ EventType() string })
    if !ok {
        return fmt.Errorf("event does not implement EventType()")
    }

    payload, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("marshal event: %w", err)
    }

    task := asynq.NewTask(eventTyper.EventType(), payload)
    _, err = p.client.EnqueueContext(ctx, task)
    if err != nil {
        return fmt.Errorf("enqueue task: %w", err)
    }

    return nil
}

func (p *AsynqEventPublisher) Close() error {
    return p.client.Close()
}
```

## Testing Requirements

### Coverage Target: 70%

Focus on:
1. **Error handling** - Infrastructure failures
2. **Mapping logic** - Domain ↔ persistence models
3. **Transaction behavior** - Commit/rollback scenarios

### Integration Tests with testcontainers

```go
package postgres_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"

    "goimg-datalayer/internal/infrastructure/persistence/postgres"
)

func setupTestDB(t *testing.T) *sqlx.DB {
    t.Helper()

    ctx := context.Background()

    // Start Postgres container
    postgresContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    require.NoError(t, err)

    t.Cleanup(func() {
        _ = postgresContainer.Terminate(ctx)
    })

    connStr, err := postgresContainer.ConnectionString(ctx)
    require.NoError(t, err)

    db, err := postgres.NewConnection(ctx, postgres.Config{
        // Parse connStr...
    })
    require.NoError(t, err)

    // Run migrations
    runMigrations(t, db)

    return db
}

func TestPostgresUserRepository_FindByID(t *testing.T) {
    db := setupTestDB(t)
    repo := postgres.NewPostgresUserRepository(db)

    // Test implementation...
}
```

## Error Handling Patterns

### Convert Infrastructure Errors

```go
// Bad - leaks sql.ErrNoRows
return nil, err

// Good - converts to domain error
if errors.Is(err, sql.ErrNoRows) {
    return nil, identity.ErrUserNotFound
}
```

### Wrap with Context

```go
if err != nil {
    return nil, fmt.Errorf("save user to postgres: %w", err)
}
```

## Agent Responsibilities

- **senior-go-architect**: Reviews repository patterns, transaction handling
- **backend-developer**: Implements repositories and infrastructure services
- **image-gallery-expert**: Reviews storage and image processing implementations
- **test-strategist**: Ensures testcontainers setup and 70% coverage

## See Also

- Domain interfaces: `internal/domain/CLAUDE.md`
- Application layer usage: `internal/application/CLAUDE.md`
- Storage-specific guide: `internal/infrastructure/storage/CLAUDE.md`
- Security patterns: `claude/api_security.md`
