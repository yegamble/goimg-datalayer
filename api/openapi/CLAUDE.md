# OpenAPI Specification Guide

> This directory contains the **single source of truth** for the HTTP API.

## Key Rules

1. **Spec first** - Define endpoints here before implementing
2. **Generate code** - Run `make generate` after changes
3. **Validate always** - Run `make validate-openapi` before commits
4. **No drift** - `git diff` must be clean after generation

## Structure

```
api/openapi/
├── openapi.yaml          # Main specification file
├── schemas/              # Reusable component schemas
│   ├── user.yaml
│   ├── image.yaml
│   └── error.yaml
└── paths/                # Endpoint definitions
    ├── auth.yaml
    ├── users.yaml
    └── images.yaml
```

## Workflow

```bash
# 1. Edit spec
vim api/openapi/openapi.yaml

# 2. Validate
make validate-openapi

# 3. Regenerate server code
make generate

# 4. Verify no unintended changes
git diff

# 5. Implement handler changes
```

## Error Schema (RFC 7807)

All errors must use this schema:

```yaml
ProblemDetail:
  type: object
  required: [type, title, status]
  properties:
    type:
      type: string
      format: uri
    title:
      type: string
    status:
      type: integer
    detail:
      type: string
    traceId:
      type: string
```

## Adding New Endpoint

1. Define path in `paths/{resource}.yaml`
2. Define schemas in `schemas/{resource}.yaml`
3. Reference from main `openapi.yaml`
4. Run `make validate-openapi`
5. Run `make generate`
6. Implement handler

## See Also

- API patterns: `claude/api_security.md`
- Handler guide: `internal/interfaces/http/CLAUDE.md`
