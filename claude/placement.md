# Scoped Guide Placement

> Keep instructions discoverable and lightweight by placing `CLAUDE.md` files next to the code they govern.

## Why Scoped Guides?

- **Reduced context** - Claude loads only relevant instructions
- **Discoverability** - Guidance lives where work happens
- **Maintainability** - Rules stay with the code they describe

## Active Scoped Guides

| Location | Purpose |
|----------|---------|
| `internal/domain/CLAUDE.md` | DDD patterns, no external deps |
| `internal/application/CLAUDE.md` | Commands, queries, orchestration |
| `internal/infrastructure/CLAUDE.md` | DB, storage, external services |
| `internal/interfaces/http/CLAUDE.md` | Handlers, middleware, DTOs |
| `api/openapi/CLAUDE.md` | API spec workflow |
| `tests/CLAUDE.md` | Testing patterns and fixtures |
| `cmd/CLAUDE.md` | Entry points and startup |

## Placement Rules

1. **Put guidance where work happens** - Add `CLAUDE.md` in the directory that needs rules
2. **Keep files concise** - Under ~100 lines; link to `claude/` for details
3. **Avoid duplication** - Reference shared standards, don't repeat them
4. **Move with code** - When refactoring, move the guide too

## When to Add a Scoped Guide

Add a `CLAUDE.md` when a directory has:
- Special patterns not covered elsewhere
- Subsystem-specific rules
- Common mistakes to document
- Unique testing or validation requirements

## Template

```markdown
# {Directory} Guide

> One-line purpose of this directory.

## Key Rules

1. Rule one
2. Rule two
3. Rule three

## Structure

\`\`\`
directory/
├── file.go
└── other.go
\`\`\`

## Pattern Example

\`\`\`go
// Code example
\`\`\`

## See Also

- Related guide: `claude/topic.md`
```

## Hierarchy

```
CLAUDE.md (root)           ← Entry point, navigation
├── claude/                ← Topic guides (detailed)
│   ├── architecture.md
│   ├── coding.md
│   └── ...
├── internal/
│   ├── domain/CLAUDE.md   ← Scoped (concise)
│   ├── application/CLAUDE.md
│   └── ...
└── api/openapi/CLAUDE.md
```

## Reference

- [Anthropic's Claude Code Best Practices](https://www.anthropic.com/engineering/claude-code-best-practices)
