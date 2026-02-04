# Sprint 12: OAuth & Social Features

> **Status**: IN PROGRESS (~80%)
> **Start Date**: 2026-01-07
> **Duration**: 2 weeks
> **Sprint Goal**: Implement OAuth authentication (Google, GitHub), user follow system, activity feeds, email notifications, and session elevation after 2FA

## Current Progress (2026-01-07)

### Completed ✅
- **OAuth Domain Layer**: OAuthAccount entity, OAuthProvider value object, repository interface
- **OAuth Infrastructure**: Google/GitHub OAuth providers with golang.org/x/oauth2
- **OAuth Repository**: PostgreSQL implementation with encrypted token storage
- **OAuth Application**: Commands (AuthenticateOAuthCommand, LinkOAuthAccountCommand, UnlinkOAuthAccountCommand) + GetLinkedAccountsQuery
- **OAuth HTTP Layer**: OAuthHandler with 5 endpoints (/auth/oauth/{provider}, /callback, /link, /accounts)
- **Router Wiring**: OAuthHandler mounted at /api/v1/auth/oauth
- **OpenAPI Spec**: All OAuth endpoints documented
- **Database Migration 00007**: oauth_accounts table created
- **E2E Tests**: 9 Newman tests covering OAuth error scenarios

### Pending 📋
- Session elevation after 2FA (S11-2FA-004)
- Follow/unfollow users (social features)
- Activity feeds
- Email notifications (SMTP)
- Notification preferences

---

## Overview

Sprint 12 extends authentication options with OAuth providers and adds social networking features. Users can authenticate via Google/GitHub, follow other users, view activity feeds from followed users, and receive email notifications for key events.

This sprint also completes the deferred control S11-2FA-004 (session elevation after 2FA) from Sprint 11.

## Key Features

| Feature | Priority | Status | Description |
|---------|----------|--------|-------------|
| Google OAuth | P0 | ✅ COMPLETE | OAuth 2.0 integration with Google |
| GitHub OAuth | P0 | ✅ COMPLETE | OAuth 2.0 integration with GitHub |
| OAuth Router Wiring | P0 | ✅ COMPLETE | Endpoints mounted at /api/v1/auth/oauth |
| OAuth E2E Tests | P0 | ✅ COMPLETE | 9 Newman tests for error handling |
| Session Elevation after 2FA | P0 | 📋 PENDING | JWT token includes 2FA verification claim (S11-2FA-004) |
| Follow/Unfollow Users | P1 | 📋 PENDING | User-to-user follow relationships |
| Activity Feed | P1 | 📋 PENDING | Timeline of followed users' uploads |
| Email Notifications (SMTP) | P1 | 📋 PENDING | Email delivery for new followers, uploads, account events |
| Notification Preferences | P2 | 📋 PENDING | User control over email notification opt-in |

---

## Technical Implementation

### OAuth 2.0 Flow

**Provider Support**: Google and GitHub via OAuth 2.0 authorization code flow

**Libraries:**
- `golang.org/x/oauth2` (official Go OAuth2 client)
- Provider-specific packages:
  - `google.golang.org/api/oauth2/v2` (Google)
  - `github.com/google/go-github/v57/github` (GitHub API)

**Flow:**
```
1. User clicks "Sign in with Google/GitHub"
2. Redirect to provider authorization URL
3. User grants permission
4. Provider redirects to callback with authorization code
5. Exchange code for access token
6. Fetch user profile (email, name, avatar)
7. Create or link user account
8. Issue JWT tokens (same as email/password login)
```

**Database Migration 00007: OAuth Tables**

```sql
-- oauth_accounts: Link users to OAuth providers
CREATE TABLE oauth_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(20) NOT NULL,  -- 'google' or 'github'
    provider_user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    avatar_url VARCHAR(512),
    access_token_encrypted BYTEA,  -- Optional: store for API access
    refresh_token_encrypted BYTEA,
    token_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider, provider_user_id),
    INDEX idx_oauth_accounts_user_id (user_id),
    INDEX idx_oauth_accounts_provider_email (provider, email)
);
```

### Database Migration 00008: Social Features Tables

```sql
-- user_follows: User-to-user follow relationships
CREATE TABLE user_follows (
    follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    followed_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, followed_id),
    CHECK (follower_id != followed_id),  -- Cannot follow self
    INDEX idx_user_follows_follower (follower_id),
    INDEX idx_user_follows_followed (followed_id)
);

-- notifications: Internal notification storage
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    metadata JSONB,  -- Flexible payload (image IDs, user IDs, etc.)
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    INDEX idx_notifications_recipient (recipient_id, created_at DESC),
    INDEX idx_notifications_unread (recipient_id, read_at) WHERE read_at IS NULL
);

-- notification_preferences: User email notification settings
CREATE TABLE notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    email_new_follower BOOLEAN NOT NULL DEFAULT true,
    email_new_uploads BOOLEAN NOT NULL DEFAULT true,
    email_account_events BOOLEAN NOT NULL DEFAULT true,  -- Always true for bans/suspensions
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- activity_feed: Denormalized feed for performance
CREATE TABLE activity_feed (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL,  -- 'image_uploaded', 'user_followed'
    entity_id UUID,  -- Image ID or other entity
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    INDEX idx_activity_feed_user_created (user_id, created_at DESC),
    INDEX idx_activity_feed_actor (actor_id)
);
```

### API Endpoints

**OAuth Endpoints:**

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/auth/oauth/google` | Initiate Google OAuth flow |
| GET | `/auth/oauth/google/callback` | Google OAuth callback |
| GET | `/auth/oauth/github` | Initiate GitHub OAuth flow |
| GET | `/auth/oauth/github/callback` | GitHub OAuth callback |
| POST | `/auth/oauth/link` | Link OAuth account to existing user (requires auth) |
| DELETE | `/auth/oauth/{provider}` | Unlink OAuth account (requires auth) |

**Social Endpoints:**

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/users/{id}/follow` | Follow user |
| DELETE | `/users/{id}/follow` | Unfollow user |
| GET | `/users/{id}/followers` | List followers |
| GET | `/users/{id}/following` | List following |
| GET | `/feed` | Get activity feed (followed users' uploads) |
| GET | `/notifications` | List notifications (paginated) |
| PATCH | `/notifications/{id}/read` | Mark notification as read |
| PATCH | `/notifications/read-all` | Mark all notifications as read |
| GET | `/notifications/preferences` | Get notification preferences |
| PATCH | `/notifications/preferences` | Update notification preferences |

### Application Layer

**Commands:**
- `AuthenticateWithOAuthCommand` - OAuth login/registration
- `LinkOAuthAccountCommand` - Link OAuth to existing account
- `UnlinkOAuthAccountCommand` - Remove OAuth link
- `FollowUserCommand` - Create follow relationship
- `UnfollowUserCommand` - Remove follow relationship
- `MarkNotificationReadCommand` - Mark notification as read
- `MarkAllNotificationsReadCommand` - Bulk mark as read
- `UpdateNotificationPreferencesCommand` - Change email settings

**Queries:**
- `GetOAuthAccountQuery` - Retrieve OAuth account by provider
- `GetUserFollowersQuery` - List followers with pagination
- `GetUserFollowingQuery` - List following with pagination
- `GetActivityFeedQuery` - Feed with pagination (images from followed users)
- `ListNotificationsQuery` - Notifications with pagination
- `GetNotificationPreferencesQuery` - User's email preferences

**Event Handlers:**
- `UserFollowedEventHandler` - Create notification, send email if opted-in
- `ImageUploadedEventHandler` - Fan-out to followers' feeds, send batch emails
- `UserBannedEventHandler` - Create notification, always send email

### Session Elevation (S11-2FA-004)

**JWT Token Changes:**

Add claim to JWT access token after 2FA verification:

```go
type JWTClaims struct {
    UserID         string    `json:"sub"`
    Email          string    `json:"email"`
    Role           string    `json:"role"`
    TwoFactorAuth  bool      `json:"2fa"`           // NEW: 2FA enabled for user
    TwoFactorValid bool      `json:"2fa_verified"`  // NEW: 2FA verified this session
    VerifiedAt     time.Time `json:"verified_at"`   // NEW: Timestamp of 2FA verification
    IssuedAt       time.Time `json:"iat"`
    ExpiresAt      time.Time `json:"exp"`
}
```

**Partial Session Token (from Sprint 11):**

When user with 2FA enabled logs in with password only, issue a partial session token:

```go
type PartialSessionClaims struct {
    UserID         string    `json:"sub"`
    Email          string    `json:"email"`
    SessionType    string    `json:"session_type"`  // "partial"
    IssuedAt       time.Time `json:"iat"`
    ExpiresAt      time.Time `json:"exp"`  // 5-minute TTL
}
```

**Middleware Update:**

Add middleware to check `2fa_verified` claim for sensitive operations:

```go
// RequireTwoFactorElevation checks if user has completed 2FA verification
func RequireTwoFactorElevation(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := GetJWTClaims(r.Context())

        // If user has 2FA enabled but not verified this session
        if claims.TwoFactorAuth && !claims.TwoFactorValid {
            RespondProblem(w, r, ProblemForbidden("2FA verification required"))
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

**Sensitive Operations Requiring 2FA Elevation:**
- Change password
- Disable 2FA
- Link/unlink OAuth accounts
- Delete account
- Change email address

### Email Notification System

**SMTP Configuration:**

```go
type SMTPConfig struct {
    Host          string
    Port          int
    Username      string
    Password      string  // From secrets manager
    From          string
    UseTLS        bool
    UseStartTLS   bool
}
```

**Email Templates:**

Use Go's `html/template` for email rendering:

```
internal/infrastructure/messaging/email/
├── templates/
│   ├── new_follower.html
│   ├── new_uploads.html
│   ├── account_banned.html
│   └── layout.html
├── sender.go
└── sender_test.go
```

**Rate Limiting on Emails:**

Prevent email spam:
- Max 10 emails per user per hour (Redis-backed)
- Batch notifications (e.g., "5 new uploads from users you follow")
- Digest mode: daily summary instead of real-time

**Email Queue (Asynq):**

Background job types:
```go
const (
    TaskSendEmail         = "email:send"
    TaskSendBatchEmail    = "email:send_batch"
    TaskSendDigestEmail   = "email:send_digest"
)
```

---

## Security Requirements (Gate S12)

| Control ID | Requirement | Status |
|------------|-------------|--------|
| S12-OAUTH-001 | OAuth state parameter prevents CSRF | Pending |
| S12-OAUTH-002 | OAuth tokens encrypted at rest (AES-256-GCM) | Pending |
| S12-OAUTH-003 | OAuth callback URL validated against whitelist | Pending |
| S12-OAUTH-004 | OAuth provider user ID stored, not email only | Pending |
| S12-OAUTH-005 | Account linking requires active session | Pending |
| S12-2FA-004 | Session elevation after 2FA (deferred from S11) | Pending |
| S12-2FA-006 | JWT includes 2FA verification claim | Pending |
| S12-2FA-007 | Sensitive operations require 2FA elevation | Pending |
| S12-SOCIAL-001 | Cannot follow/unfollow self | Pending |
| S12-SOCIAL-002 | Follow rate limiting (20/hour per user) | Pending |
| S12-EMAIL-001 | Email addresses validated before sending | Pending |
| S12-EMAIL-002 | Email rate limiting (10/hour per user) | Pending |
| S12-EMAIL-003 | Unsubscribe link in all marketing emails | Pending |
| S12-EMAIL-004 | No PII in email logs | Pending |
| S12-NOTIF-001 | Notification content sanitized (XSS prevention) | Pending |

---

## Test Strategy

### Coverage Targets
- **Overall**: 85%+ (matching Sprints 10-11)
- **OAuth Core**: 90% (security-critical)
- **Social Features**: 85%
- **Email Service**: 80%
- **E2E**: 100% endpoint coverage

### Test Categories

**Unit Tests:**
- OAuth flow state generation and validation
- JWT claim encoding/decoding with 2FA claims
- Follow relationship validation (cannot follow self)
- Email template rendering
- Notification preference logic

**Integration Tests:**
- OAuth account creation and linking
- Follow/unfollow operations with database
- Activity feed generation
- Email sending with mock SMTP server
- Redis rate limiting on emails and follows

**E2E Tests (Newman/Postman):**
- OAuth flow simulation (mocked provider responses)
- Follow → see uploads in feed
- Notification creation and marking as read
- Email preference updates
- 2FA elevation enforcement on sensitive endpoints

**Security Tests:**
- OAuth CSRF prevention (state parameter)
- Account enumeration via OAuth callback
- Email injection attempts
- Follow spam prevention (rate limiting)
- XSS in notification content

---

## Implementation Timeline

### Week 1 (Days 1-7)

| Day | Task | Owner |
|-----|------|-------|
| 1-2 | Database migrations 00007 (OAuth) and 00008 (Social) | cicd-guardian |
| 2-3 | OAuth domain layer: OAuthAccount entity, provider enums | senior-go-architect |
| 3-4 | OAuth infrastructure: GoogleOAuthService, GitHubOAuthService | senior-go-architect |
| 4-5 | OAuth application layer: AuthenticateWithOAuth, LinkOAuthAccount commands | senior-go-architect |
| 5-6 | Social domain layer: UserFollow entity, Notification entity | senior-go-architect |
| 6-7 | Session elevation: JWT claims update, middleware implementation | senior-secops-engineer |

### Week 2 (Days 8-14)

| Day | Task | Owner |
|-----|------|-------|
| 8-9 | Social application layer: FollowUser, GetActivityFeed queries | senior-go-architect |
| 9-10 | HTTP handlers: OAuth callback handlers, social endpoints | senior-go-architect |
| 10-11 | Email infrastructure: SMTP sender, template rendering | senior-go-architect |
| 11-12 | Notification event handlers: UserFollowed, ImageUploaded | senior-go-architect |
| 12-13 | E2E tests (Newman/Postman): OAuth, social, email flows | test-strategist |
| 13-14 | Security Gate S12 review | senior-secops-engineer |

---

## Agent Assignments

| Agent | Responsibilities |
|-------|------------------|
| **senior-go-architect** | OAuth implementation, social features, email service |
| **senior-secops-engineer** | OAuth security review, session elevation, S12-2FA-004 completion |
| **backend-test-architect** | Test strategy, coverage targets |
| **cicd-guardian** | Database migrations, SMTP configuration in CI |
| **test-strategist** | E2E tests, OAuth flow testing, email delivery validation |
| **scrum-master** | Sprint coordination, progress tracking |

---

## Key Architecture Decisions

### 1. OAuth Token Storage

**Decision**: Store OAuth access tokens encrypted at rest (optional feature)

**Rationale**: Enables future API integrations (e.g., import photos from Google Photos). Encrypted with AES-256-GCM using same `SecretEncryptor` as TOTP secrets.

**Implementation**:
```go
type OAuthAccount struct {
    id                   uuid.UUID
    userID               uuid.UUID
    provider             OAuthProvider
    providerUserID       string
    email                string
    encryptedAccessToken []byte  // Optional
    tokenExpiresAt       *time.Time
}
```

### 2. Activity Feed Denormalization

**Decision**: Use denormalized `activity_feed` table for performance

**Rationale**: Fan-out on write is acceptable for follow operations (infrequent). Avoids complex joins on timeline reads (frequent).

**Trade-offs**:
- ✅ Fast feed reads (single query)
- ✅ Simple pagination
- ❌ Higher write cost when user uploads
- ❌ Storage overhead

### 3. Email Batching Strategy

**Decision**: Batch notifications within 15-minute windows

**Rationale**: Reduces email spam if followed user uploads 10 images in 5 minutes. Send single email: "User X uploaded 10 new photos".

**Implementation**: Asynq scheduled task runs every 15 minutes, aggregates pending notifications.

### 4. Session Elevation Middleware

**Decision**: Apply `RequireTwoFactorElevation` middleware at route level, not globally

**Rationale**: Most operations (viewing images, liking) don't require 2FA re-verification. Only sensitive account changes need elevation.

**Protected Operations**:
- `PATCH /users/me/password`
- `DELETE /auth/2fa`
- `POST /auth/oauth/link`
- `DELETE /users/me`

---

## Dependencies

### New Go Packages

```go
require (
    golang.org/x/oauth2 v0.15.0                        // OAuth 2.0 client
    google.golang.org/api v0.154.0                     // Google APIs
    github.com/google/go-github/v57 v57.0.0           // GitHub API
    github.com/jordan-wright/email v4.0.1-0.20210109023952-943e75fe5223+incompatible  // SMTP email sending
)
```

### SMTP Configuration

**Development**: Use MailHog (Docker Compose)
```yaml
mailhog:
  image: mailhog/mailhog:latest
  ports:
    - "1025:1025"  # SMTP
    - "8025:8025"  # Web UI
```

**Production**: Support multiple providers
- AWS SES
- SendGrid
- Mailgun
- Self-hosted SMTP

---

## Success Criteria

- [ ] All 15 Security Gate S12 controls pass
- [ ] Test coverage 85%+
- [ ] E2E tests passing (OAuth flows, social features, email delivery)
- [ ] No critical/high vulnerabilities
- [ ] OAuth flow completes in <3 seconds p95
- [ ] Activity feed query <200ms p95
- [ ] Email delivery <5 seconds p95 (background job)
- [ ] Documentation complete (OAuth setup guide, SMTP configuration)
- [ ] S11-2FA-004 control completed (session elevation)

---

## References

- [OAuth 2.0 RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749)
- [OAuth 2.0 Security Best Practices](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-security-topics)
- [Google OAuth 2.0 Documentation](https://developers.google.com/identity/protocols/oauth2)
- [GitHub OAuth Documentation](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps)
- [OWASP OAuth Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/OAuth2_Cheat_Sheet.html)
- [SMTP Email Security Best Practices](https://cheatsheetseries.owasp.org/cheatsheets/Email_Security_Cheat_Sheet.html)
- [Activity Streams 2.0](https://www.w3.org/TR/activitystreams-core/) (feed design patterns)
- [CAN-SPAM Act Compliance](https://www.ftc.gov/business-guidance/resources/can-spam-act-compliance-guide-business) (email marketing)
