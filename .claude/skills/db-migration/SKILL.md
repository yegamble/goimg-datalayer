---
name: db-migration
version: 1.0.0
description: >
  Use this skill when creating, applying, or rolling back database migrations in
  goimg-datalayer. Triggers when:
  - User asks to "add column", "create table", "add index", "rename column", or "migrate"
  - A new domain entity or feature requires schema changes
  - User needs to modify existing table structure
  Uses Goose with sequential SQL files in migrations/.
---

# Skill: Database Migration

## Step 1 — Find the Next Migration Number

```bash
ls migrations/ | tail -1
# e.g., 00021_add_group_album_visibility.sql → next is 00022
```

## Step 2 — Create Migration File

```bash
make migrate-create NAME=add_foo_table
# Creates: migrations/00022_add_foo_table.sql
```

Or create manually: `migrations/00022_add_foo_table.sql`

## Step 3 — Write the SQL

```sql
-- +goose Up
CREATE TABLE foos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_foos_user_id ON foos(user_id);
CREATE INDEX idx_foos_status ON foos(status) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS foos;
```

**Column checklist:**
- [ ] UUID PK with `gen_random_uuid()`
- [ ] `TIMESTAMPTZ` (not `TIMESTAMP`) for all time fields
- [ ] `NOT NULL` with defaults where appropriate
- [ ] Foreign keys with `ON DELETE CASCADE` or `RESTRICT` as appropriate
- [ ] `deleted_at TIMESTAMPTZ` for soft-deleteable entities
- [ ] Indexes for every FK and commonly-filtered column
- [ ] Partial indexes `WHERE deleted_at IS NULL` for soft-delete tables

## Step 4 — Apply Migration

```bash
make migrate-up       # Apply pending migrations
make migrate-status   # Verify applied
```

## Step 5 — Update Repository

Add sqlx struct tags to the persistence model in `internal/infrastructure/persistence/postgres/`:

```go
type fooModel struct {
    ID        string         `db:"id"`
    UserID    string         `db:"user_id"`
    Name      string         `db:"name"`
    Status    string         `db:"status"`
    CreatedAt time.Time      `db:"created_at"`
    UpdatedAt time.Time      `db:"updated_at"`
    DeletedAt sql.NullTime   `db:"deleted_at"`
}
```

## Step 6 — Test

Integration tests use `testcontainers-go` which auto-runs migrations. Run:

```bash
make test-integration
```

To manually test rollback:
```bash
make migrate-down     # Rolls back last migration
make migrate-up       # Re-applies
```

## Common Patterns

### Add Column to Existing Table
```sql
-- +goose Up
ALTER TABLE users ADD COLUMN avatar_url VARCHAR(500);

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
```

### Rename Column (safe, additive pattern)
```sql
-- +goose Up
ALTER TABLE images ADD COLUMN file_size BIGINT NOT NULL DEFAULT 0;
UPDATE images SET file_size = size_bytes WHERE size_bytes IS NOT NULL;
ALTER TABLE images DROP COLUMN size_bytes;

-- +goose Down
ALTER TABLE images ADD COLUMN size_bytes BIGINT;
UPDATE images SET size_bytes = file_size;
ALTER TABLE images DROP COLUMN file_size;
```

### Add Index Only
```sql
-- +goose Up
CREATE INDEX CONCURRENTLY idx_images_created_at ON images(created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_images_created_at;
```

Use `CONCURRENTLY` for production tables to avoid locking.

### Enum-Like Status Column
Prefer `VARCHAR` + application-level validation over Postgres `ENUM` (easier to migrate):
```sql
status VARCHAR(20) NOT NULL DEFAULT 'pending'
-- Values: 'pending', 'active', 'suspended', 'deleted'
```

## Rules

- **Never edit an existing migration** — always create a new one
- **Always write a Down section** — even if just `DROP TABLE IF EXISTS`
- **Sequential numbering** — check `ls migrations/ | tail -1` before creating
- **One logical change per migration** — don't bundle unrelated changes
- **`TIMESTAMPTZ` always** — never plain `TIMESTAMP`
