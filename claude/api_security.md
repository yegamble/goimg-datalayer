# API Contract & Security

## OpenAPI as Source of Truth
- Spec lives in `api/openapi/`; treat it as authoritative for all HTTP behavior.
- Generate server code with `make generate` and ensure `git diff` is clean afterwards.
- Run `make validate-openapi` before submitting API changes.

## HTTP Layer Expectations
- Handlers should only translate HTTP ↔ DTO ↔ application commands/queries.
- Use DTO validation for request shape; domain constructors validate invariants.
- Error responses must follow RFC 7807 Problem Details.

## Authentication & Authorization
- JWT access/refresh tokens with configurable TTLs; sessions stored in Redis.
- Role-based permissions (admin, moderator, user) backed by permission constants.
- Middleware should short-circuit unauthorized/forbidden requests with Problem responses.

## Rate Limiting & Abuse Controls
- Per-IP and per-user throttles (RPM/RPH) with Redis-backed limiters.
- Endpoint-specific limits for uploads and login attempts.

## Image Safety
- Enforce maximum upload size and MIME sniffing; validate images can be decoded.
- Scan uploads with ClamAV; reject infected files.
- Generate and validate image variants through aggregate methods.

## Security Headers
- Apply headers such as `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, CSP, Referrer-Policy, and Permissions-Policy in middleware.

## Storage & External Services
- Storage implementations (local FS, S3/Spaces/B2) hide behind interfaces in `internal/infrastructure/storage`.
- Keep domain logic independent of storage or transport concerns.
