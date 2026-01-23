# Sprint Revive Issues

## Issue 1: Wire Application Entry Points (P0)
**Assignee:** Builder 🛠️
**Labels:** backend, infra, critical

**Description:**
The application entry points (`cmd/api/main.go` and `cmd/worker/main.go`) are currently empty stubs. They need to be implemented to allow the application to run.

**Requirements:**
1.  **`cmd/api/main.go`**:
    *   Load configuration (env vars / .env).
    *   Initialize Logger (zerolog).
    *   Connect to PostgreSQL (wait for connection).
    *   Connect to Redis.
    *   Initialize all Repositories:
        *   Identity: `UserRepository`, `SessionRepository`
        *   Gallery: `ImageRepository`, `AlbumRepository`, `TagRepository`, `VariantConfigRepository`
        *   Social: `LikeRepository`, `CommentRepository`, `FollowRepository`
        *   Community: `GroupRepository`, `GroupMembershipRepository`, `GroupInvitationRepository`
        *   Moderation: `ReportRepository`, `BanRepository`, `NSFWScanRepository`, `FeaturedPickRepository`
    *   Initialize Infrastructure Services:
        *   `JWTService`
        *   `StorageOrchestrator` (Local/S3 + IPFS)
        *   `ClamAVScanner`
        *   `NSFWDetector` (SightEngine/ModerateContent)
        *   `EmailService` (SMTP)
        *   `EventPublisher` (Redis PubSub or internal)
    *   Initialize Application Handlers (Commands/Queries):
        *   Wire all repositories and services into command/query handlers.
    *   Initialize HTTP Handlers:
        *   Instantiate all handlers in `internal/interfaces/http/handlers/`.
        *   Pass dependencies via constructors.
    *   Router Setup:
        *   Call `handlers.NewRouter(...)`.
    *   Server Start:
        *   Run HTTP server on configured port with graceful shutdown.

2.  **`cmd/worker/main.go`**:
    *   Initialize `asynq.Server`.
    *   Register task handlers for background jobs (Image Processing, Email, etc.).
    *   Start server.

**Acceptance Criteria:**
*   `make run` starts the server without panic.
*   `curl http://localhost:8080/health` returns 200 OK.
*   Logs show successful connection to DB and Redis.

---

## Issue 2: Implement Group Invitation HTTP Layer (P1)
**Assignee:** Builder 🛠️
**Labels:** backend, api

**Description:**
Complete the Group Invitation system implementation. The domain and application logic exists, but the HTTP interface is missing.

**Requirements:**
1.  **Create Query Handler:**
    *   Create `internal/application/community/queries/list_group_invitations.go`.
    *   Implement logic to fetch pending invitations for a group (paginated).
2.  **Update `GroupHandler`:**
    *   Add fields for `InviteToGroupHandler`, `AcceptInvitationHandler`, `DeclineInvitationHandler`, `ListGroupInvitationsHandler`.
    *   Update `NewGroupHandler` constructor.
    *   Implement HTTP methods:
        *   `CreateInvitation(w, r)`
        *   `AcceptInvitation(w, r)`
        *   `DeclineInvitation(w, r)`
        *   `ListInvitations(w, r)`
    *   Implement `mapInvitationToResponse` DTO mapper.
3.  **Update OpenAPI:**
    *   Add paths and schemas to `api/openapi/openapi.yaml`.

**Acceptance Criteria:**
*   All new methods implemented.
*   `GroupHandler` compiles with new dependencies.
*   `make validate-openapi` passes.

---

## Issue 3: Verify Invitation System (P1)
**Assignee:** QA Guardian 🧪
**Labels:** tests, e2e

**Description:**
Add tests to verify the new Invitation System features.

**Requirements:**
1.  **Unit Tests:**
    *   Update `internal/interfaces/http/handlers/group_handler_test.go`.
    *   Test `CreateInvitation`: Valid request, unauthorized, invalid payload.
    *   Test `AcceptInvitation`: Valid token, expired token, already member.
    *   Test `DeclineInvitation`.
    *   Test `ListInvitations`.
2.  **E2E Tests:**
    *   Add `Group Invitations` folder to Postman collection.
    *   Scenario: Admin invites user -> User accepts -> User is member.
    *   Scenario: User declines invitation.

**Acceptance Criteria:**
*   Unit tests pass with 80%+ coverage.
*   E2E tests pass (once server is running).

---

## Issue 4: Video Support Design (P2)
**Assignee:** Product Architect 🧩
**Labels:** design, documentation

**Description:**
Design the architecture for handling video uploads, transcoding, and playback.

**Requirements:**
1.  Create `claude/features/video_support.md`.
2.  Define:
    *   Supported input formats.
    *   Transcoding strategy (HLS vs DASH, resolutions).
    *   Storage structure for video segments.
    *   Thumbnail generation.
    *   Database schema updates (`videos` table vs `images` table expansion).
    *   API endpoints.

**Acceptance Criteria:**
*   Design document created and reviewed.
