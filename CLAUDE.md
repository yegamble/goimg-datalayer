# Claude Agent Guide

This repository uses a foldered guide so Claude agents can stay within scope and keep context size low. Load only what you need for the area you are working in.

- Start here for orientation, then jump into the topic-specific files under `claude/`.
- When working in a single folder or feature, load only the matching guide (for example, tests ➜ `claude/testing_ci.md`; HTTP handlers ➜ `claude/api_security.md`).
- Prefer placing short, folder-local `CLAUDE.md` files next to the code they describe (see `claude/placement.md`). This follows [Anthropic's scoped-instruction guidance](https://www.anthropic.com/engineering/claude-code-best-practices).

## Quick Navigation

| Topic | File |
| --- | --- |
| Architecture & domain model | `claude/architecture.md` |
| Coding standards, linting, and formatting | `claude/coding.md` |
| API contract, security, and HTTP guidance | `claude/api_security.md` |
| Testing strategy and CI/CD expectations | `claude/testing_ci.md` |
| Mandatory agent checklist | `claude/agent_checklist.md` |
| Where to place scoped guides | `claude/placement.md` |

## Core Expectations

- Follow Domain-Driven Design (DDD) layering and keep domain logic free of infrastructure dependencies.
- Treat the OpenAPI spec as the single source of truth for the HTTP API.
- Maintain the testing and security posture documented in the topic guides.
- Keep instructions scoped: prefer localized guides over monolithic documents to prevent unnecessary memory use.
