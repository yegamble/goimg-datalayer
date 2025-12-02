# Scoped Guide Placement

Keep instructions discoverable and lightweight by placing short `CLAUDE.md` files in the directory that contains the code they govern. Claude will read only the files in the relevant path, which minimizes context usage.

## Placement Rules
- **Put guidance where work happens.** Add a small `CLAUDE.md` or `AGENTS.md` inside the directory that needs special rules.
- **Keep files concise and link out.** Point to the topic guides in `claude/` when more detail is needed.
- **Avoid one giant file.** Split rules by subsystem to prevent unnecessary context loading.

## Suggested Locations for Go Projects
- Root repo: brief orientation (this file set) plus links to scoped guides.
- `cmd/`: runtime flags, startup wiring expectations, logging defaults.
- `api/openapi/`: OpenAPI contract rules and generation steps.
- `internal/domain/`: DDD invariants, entity/value-object patterns, and domain error conventions.
- `internal/application/`: command/query handler expectations and validation patterns.
- `internal/infrastructure/`: persistence/storage/security specifics; remind not to leak into domain.
- `internal/interfaces/http/`: handler/middleware DTO expectations and Problem Details responses.
- `tests/`: test data, integration/e2e setup, and coverage targets.
- `scripts/` and `docker/`: local tooling, migrations, and container requirements.

## How to Add a Scoped File
1. Create `CLAUDE.md` (or `AGENTS.md` if you prefer) inside the target folder.
2. Include only the rules unique to that folder; link back to `../claude/*.md` for shared standards.
3. Keep the file under ~150 lines; shorter is better for context efficiency.
4. When moving or adding folders, move the corresponding scoped file with it.

Following this layout keeps Claude aligned with [Anthropic's scoped-instruction best practices](https://www.anthropic.com/engineering/claude-code-best-practices) while keeping memory use low.
