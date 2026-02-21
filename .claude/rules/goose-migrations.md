# Goose Migration Conventions

Migrations live in `migrations/` and use sequential numeric prefixes.

## File Naming

```
{NNNNN}_{description}.sql
```

Examples:
- `00022_create_foos.sql`
- `00023_add_foo_index.sql`

**Rules:**
- Always 5-digit zero-padded sequential number (check last migration before creating)
- Snake_case description
- One migration per logical change (no bundling unrelated changes)
- **Never edit an existing migration** — create a new one instead

## File Structure

```sql
-- +goose Up
CREATE TABLE foos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_foos_name ON foos(name);

-- +goose Down
DROP TABLE IF EXISTS foos;
```

**Rules:**
- `-- +goose Up` and `-- +goose Down` are mandatory markers
- Always write a rollback (`Down`) — even if just `DROP TABLE IF EXISTS`
- Use `TIMESTAMPTZ` not `TIMESTAMP` (time zone aware)
- Use `gen_random_uuid()` for UUID primary keys
- Partial indexes: `WHERE deleted_at IS NULL` for soft-delete tables
- Prefer additive migrations; avoid renaming columns (use add+backfill+drop pattern)

## Common Patterns

### Add Column (with default for existing rows)
```sql
-- +goose Up
ALTER TABLE foos ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'active';
CREATE INDEX idx_foos_status ON foos(status);

-- +goose Down
DROP INDEX IF EXISTS idx_foos_status;
ALTER TABLE foos DROP COLUMN IF EXISTS status;
```

### Soft Delete Support
```sql
deleted_at TIMESTAMPTZ,
-- then add partial index:
CREATE INDEX idx_foos_name ON foos(name) WHERE deleted_at IS NULL;
```

### Foreign Key
```sql
user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
```

## Commands

```bash
make migrate-up              # Apply all pending migrations
make migrate-down            # Rollback last migration
make migrate-status          # Show migration status
make migrate-create NAME=create_foos   # Create new migration file
```

## Integration Test Pattern

Integration tests use `testcontainers-go` which runs migrations automatically via the `containers.NewPostgresContainer()` helper. No manual migration needed in tests.
