# Setup Database Composite Action

This composite action handles common database setup tasks for CI jobs that require database connectivity.

## What It Does

1. **Waits for PostgreSQL to be ready** - Polls `pg_isready` with proper timeout and error handling
2. **Installs Goose migration tool** - Only if migrations directory exists
3. **Runs database migrations** - Executes `make migrate-up` if Makefile target exists

## Why This Exists

This action was created to eliminate code duplication between the `test-integration` and `e2e-tests` jobs. Both jobs require identical database setup logic, and maintaining it in one place improves:

- **Maintainability**: Single source of truth for database setup
- **Consistency**: Both jobs use identical setup procedures
- **Readability**: Reduces workflow file size and complexity

## Usage

```yaml
- name: Setup database and run migrations
  uses: ./.github/actions/setup-database
  with:
    database-url: ${{ env.DATABASE_URL }}
```

### With Skip Migrations

```yaml
- name: Setup database only (no migrations)
  uses: ./.github/actions/setup-database
  with:
    database-url: ${{ env.DATABASE_URL }}
    skip-migrations: 'true'
```

## Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `database-url` | Yes | - | PostgreSQL connection string |
| `skip-migrations` | No | `false` | Skip running migrations |

## Requirements

- PostgreSQL service container must be running
- `pg_isready` must be available in the runner
- Go must be installed (for Goose installation)
- Makefile with `migrate-up` target (optional, will skip if not present)

## Platform Limitation Note

This action addresses **code duplication**, but **cannot eliminate service container duplication** between jobs. GitHub Actions does not support sharing service containers between jobs due to job isolation. Each job that needs PostgreSQL/Redis must define its own service containers.
