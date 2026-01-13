# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Improved logging for invalid integer environment variables in database configuration
- Enhanced configuration validation with new `ValidateConfig()` function
- Updated documentation for SSL mode options

## [1.0.0] - 2026-01-13

### Added

- Go 1.25+ requirement with runtime version checking
- `ValidateConfig()` function for explicit database configuration validation
- Warning logs when required database credentials are not set
- Comprehensive SSL mode documentation

### Changed

- **BREAKING**: Minimum Go version increased to 1.25 (from 1.24)
- **BREAKING**: Default `SSLMode` changed from `"disable"` to `"require"` for security
- **BREAKING**: Default database credentials removed (no longer defaults to `postgres/postgres`)

### Migration Guide

#### Go Version Requirement

This release requires Go 1.25 or later. Update your Go installation before building:

```bash
# Verify Go version
go version  # Must show go1.25.x or higher
```

#### SSL Mode Breaking Change

The default SSL mode for database connections has changed from `"disable"` to `"require"`.

**Impact**: Applications connecting to databases that do not support SSL will fail to connect.

**Migration Options**:

1. **Recommended**: Enable SSL on your PostgreSQL server
2. **Alternative**: Set `DB_SSL_MODE=disable` environment variable for development/testing
3. **Production**: Use `DB_SSL_MODE=verify-full` for full certificate validation

```bash
# For development without SSL (not recommended for production)
export DB_SSL_MODE=disable

# For production with full certificate validation (recommended)
export DB_SSL_MODE=verify-full
```

#### Database Credentials Breaking Change

Default credentials (`postgres/postgres`) have been removed for security.

**Impact**: Code that previously relied on default credentials will fail to connect.

**Migration**:

1. Always set `DB_USER` and `DB_PASSWORD` environment variables
2. Use a secrets manager in production
3. Update any scripts or configurations that relied on defaults

```bash
# Required environment variables
export DB_USER=your_username
export DB_PASSWORD=your_secure_password
```

#### Configuration Validation

A new `ValidateConfig()` function is available to check configuration before connecting:

```go
import "github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"

cfg := postgres.ConfigFromEnv()
if err := postgres.ValidateConfig(cfg); err != nil {
    log.Fatal().Err(err).Msg("invalid database configuration")
}

db, err := postgres.NewDB(cfg)
if err != nil {
    log.Fatal().Err(err).Msg("failed to connect to database")
}
```

### Security

- Database connections now require SSL by default
- Credentials must be explicitly provided (no insecure defaults)
- Warning logs for missing required configuration help identify issues early
