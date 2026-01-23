# Sprint Plan: Revive & Complete (Sprint 24)

**Date:** 2026-01-20
**Sprint Goal:** Make the application executable (fix entry points) and finalize the Group Invitation system.
**Status:** PLANNED

## Executive Summary
A critical audit of the codebase revealed that the application entry points (`cmd/api/main.go` and `cmd/worker/main.go`) are empty stubs. While the internal architecture (Domain, Application, Infrastructure) is well-developed, the application cannot run in its current state.

This sprint focuses on "wiring up" the application to make it functional, while simultaneously completing the unfinished HTTP layer for the Group Invitation system.

## Sprint Backlog

### 1. 🛠️ [Builder] Wire Application Entry Points (P0 - CRITICAL)
**Context:**
The `cmd/api/main.go` file currently only prints a log message. It contains no server initialization code. The same applies to `cmd/worker/main.go`.

**Objectives:**
*   Implement a functional dependency injection graph in `cmd/api/main.go`.
*   Initialize and start the HTTP server.
*   Initialize and start the background worker.

**Definition of Done:**
*   `go run ./cmd/api` starts the server on port 8080 (or configured port).
*   Health check endpoints (`/health`, `/health/ready`) return 200 OK.
*   Database and Redis connections are established.

### 2. 🛠️ [Builder] Implement Group Invitation HTTP Layer (P1)
**Context:**
The Domain and Application layers for Group Invitations are complete, but the HTTP handlers and OpenAPI definitions are missing (as noted in `INVITATION_SYSTEM_IMPLEMENTATION.md`).

**Objectives:**
*   Implement `InviteToGroup`, `AcceptInvitation`, `DeclineInvitation`, and `ListInvitations` handlers in `GroupHandler`.
*   Implement the missing `ListGroupInvitations` query handler.
*   Update OpenAPI specification.

**Definition of Done:**
*   Handlers are implemented and wired in `GroupHandler`.
*   OpenAPI spec includes the 4 new endpoints.
*   `make validate-openapi` passes.

### 3. 🧪 [QA Guardian] Invitation System Tests (P1)
**Context:**
New features require verification.

**Objectives:**
*   Add unit tests for the new `GroupHandler` methods.
*   Add E2E tests (Newman) for the invitation flow.

**Definition of Done:**
*   Unit tests pass with >80% coverage for new code.
*   New E2E test file/folder added to Postman collection covering the invitation lifecycle.

### 4. 🧩 [Product Architect] Video Support Design (P2)
**Context:**
Preparing for the next major feature (Video Support) to align with the "PeerTube-like" vision.

**Objectives:**
*   Create a detailed technical specification for video uploads and playback.

**Definition of Done:**
*   `claude/features/video_support.md` created and reviewed.

## Coordination Notes
*   **Builder** must finish Task 1 (Entry Points) before **QA Guardian** can run E2E tests against a local server.
*   **Builder** should prioritize the HTTP Handlers (Task 2) implementation details while working on wiring (Task 1) to ensure the `GroupHandler` constructor is ready for injection.

## Risks
*   **Dependency Hell:** Wiring up the entire application might reveal circular dependencies or missing configuration options.
*   **Environment:** running `cmd/api` requires a working Postgres and Redis instance (Docker).
