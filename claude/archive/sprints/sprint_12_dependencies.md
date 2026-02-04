# Sprint 12: Technical Dependencies & Blockers

> **Sprint**: Sprint 12 - OAuth & Social Features
> **Date**: 2026-01-07
> **Status**: Planning Complete - Ready for Implementation

---

## Technical Dependencies

### 1. External Service Dependencies

#### OAuth Providers

**Google OAuth 2.0**
- **Required**: Google Cloud Project with OAuth 2.0 credentials
- **Setup Steps**:
  1. Create Google Cloud Project at https://console.cloud.google.com
  2. Enable Google+ API or People API
  3. Configure OAuth consent screen (app name, logo, privacy policy URL)
  4. Create OAuth 2.0 Client ID (Web application)
  5. Add authorized redirect URIs: `https://api.goimg.com/auth/oauth/google/callback`
- **Environment Variables**:
  - `GOOGLE_OAUTH_CLIENT_ID`
  - `GOOGLE_OAUTH_CLIENT_SECRET`
  - `GOOGLE_OAUTH_REDIRECT_URL`
- **Blockers**: None (can use test credentials initially)
- **Documentation**: https://developers.google.com/identity/protocols/oauth2

**GitHub OAuth**
- **Required**: GitHub OAuth App registration
- **Setup Steps**:
  1. Navigate to GitHub Settings > Developer Settings > OAuth Apps
  2. Register new OAuth application
  3. Set Authorization callback URL: `https://api.goimg.com/auth/oauth/github/callback`
  4. Note Client ID and Client Secret
- **Environment Variables**:
  - `GITHUB_OAUTH_CLIENT_ID`
  - `GITHUB_OAUTH_CLIENT_SECRET`
  - `GITHUB_OAUTH_REDIRECT_URL`
- **Blockers**: None (can use test app initially)
- **Documentation**: https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps

#### SMTP Email Service

**Production Options**:
1. **AWS SES** (Recommended for production)
   - Requires AWS account and SES verification
   - Cost: $0.10 per 1,000 emails
   - Setup time: ~30 minutes (domain verification)

2. **SendGrid**
   - Free tier: 100 emails/day
   - Paid: Starting at $15/month
   - Setup time: ~15 minutes

3. **Mailgun**
   - Free tier: 5,000 emails/month
   - Setup time: ~15 minutes

4. **Self-hosted SMTP**
   - Requires mail server setup (Postfix, etc.)
   - Higher complexity, more control

**Development**: MailHog (already in docker-compose.yml)
- No external dependencies
- Web UI at http://localhost:8025
- SMTP server at localhost:1025

**Environment Variables**:
```bash
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USERNAME=apikey
SMTP_PASSWORD=<secret>
SMTP_FROM=noreply@goimg.com
SMTP_USE_TLS=true
```

**Blockers**:
- ❌ **BLOCKER** (production only): Domain verification for AWS SES
- ✅ **Not blocking**: Can use MailHog for development/testing

---

### 2. Infrastructure Dependencies

#### Database Migrations

**Migration 00007: OAuth Accounts**
- Dependencies: None (clean addition)
- Tables: `oauth_accounts`
- Indexes: 3 indexes (user_id, provider+provider_user_id, provider+email)
- Estimated migration time: <5 seconds

**Migration 00008: Social Features**
- Dependencies: Existing `users`, `images` tables
- Tables: `user_follows`, `notifications`, `notification_preferences`, `activity_feed`
- Indexes: 6 indexes across 4 tables
- Constraints: Foreign keys, check constraints
- Estimated migration time: <10 seconds

**Rollback Plan**: Down migrations provided for both 00007 and 00008

**Blockers**: None

---

#### Redis Requirements

**New Redis Key Patterns**:
```
goimg:oauth:state:{state_value}         # OAuth CSRF state (5-min TTL)
goimg:follow:ratelimit:{user_id}        # Follow rate limiting (20/hour)
goimg:email:ratelimit:{user_id}         # Email rate limiting (10/hour)
goimg:email:batch:{user_id}             # Email batching window (15-min)
```

**Memory Impact**: Minimal (<10 MB for 10,000 users)

**Blockers**: None (existing Redis instance sufficient)

---

#### Background Job Queue (Asynq)

**New Task Types**:
```go
email:send              // Individual email
email:send_batch        // Batch notification email
email:send_digest       // Daily digest email
feed:fanout             // Fan-out image upload to followers' feeds
```

**Queue Configuration**:
- `email` queue: Priority 5, concurrency 10
- `feed` queue: Priority 3, concurrency 20

**Worker Changes**: Update `cmd/worker/main.go` to register new task handlers

**Blockers**: None (existing Asynq infrastructure in place)

---

### 3. Code Dependencies

#### Existing Sprint 11 Code

**Required for Session Elevation (S11-2FA-004)**:
- ✅ `internal/infrastructure/security/totp_service.go` - TOTP verification
- ✅ `internal/domain/identity/user.go` - User.Has2FAEnabled()
- ✅ `internal/infrastructure/persistence/postgres/totp_repository.go` - Check 2FA status

**Changes Required**:
- JWT claims structure update (add `2fa`, `2fa_verified`, `verified_at` fields)
- Middleware: `RequireTwoFactorElevation` (new)
- Login handler: Set `2fa_verified` claim after TOTP verification

**Blockers**: None (all Sprint 11 code complete)

---

#### Domain Layer Dependencies

**Existing Entities Required**:
- ✅ `User` (identity context) - For follow relationships, notifications
- ✅ `Image` (gallery context) - For activity feed generation

**New Entities Required**:
- `OAuthAccount` (identity context) - NEW
- `UserFollow` (social context) - NEW
- `Notification` (social context) - NEW
- `ActivityFeedItem` (social context) - NEW

**Domain Events Required**:
- `UserFollowed` - Triggers notification creation
- `ImageUploaded` - Already exists, extend for feed fan-out
- `UserBanned` - Already exists (moderation context)

**Blockers**: None

---

### 4. Third-Party Library Dependencies

#### New Go Modules

```go
require (
    golang.org/x/oauth2 v0.15.0                        // OAuth 2.0 client (Google-maintained)
    google.golang.org/api v0.154.0                     // Google APIs client
    github.com/google/go-github/v57 v57.0.0           // GitHub API v3 client
    github.com/jordan-wright/email v4.0.1-0.20210109023952-943e75fe5223+incompatible  // SMTP email
)
```

**Security Considerations**:
- ✅ All libraries actively maintained
- ✅ `golang.org/x/oauth2` - Official Go OAuth2 library (zero CVEs)
- ✅ `google.golang.org/api` - Google official SDK
- ✅ `github.com/google/go-github` - Google-maintained GitHub client
- ⚠️ `github.com/jordan-wright/email` - Last updated 2021 (consider alternatives: `gomail.v2`)

**License Compatibility**:
- ✅ All libraries use BSD-3-Clause or Apache-2.0 (MIT-compatible)

**Blockers**: None (all libraries production-ready)

---

## Potential Blockers & Mitigations

### High-Priority Blockers

**None identified**

---

### Medium-Priority Risks

#### 1. OAuth Provider Rate Limits

**Risk**: Google/GitHub API rate limits during development/testing

**Impact**:
- Google: 10 requests/second per user
- GitHub: 5,000 requests/hour (authenticated)

**Mitigation**:
- Use OAuth token caching (store access tokens)
- Implement exponential backoff on API failures
- Mock OAuth providers in E2E tests

**Status**: ✅ Mitigated

---

#### 2. Email Deliverability

**Risk**: Emails marked as spam, low delivery rate

**Impact**: Users don't receive notifications

**Mitigation**:
- Configure SPF, DKIM, DMARC records for sending domain
- Use reputable SMTP provider (AWS SES, SendGrid)
- Include unsubscribe link in all emails
- Implement email warm-up for new domains
- Monitor bounce rates and spam complaints

**Status**: ✅ Mitigated (use established provider)

---

#### 3. Activity Feed Performance

**Risk**: Slow feed queries for users following 1,000+ users

**Impact**: Timeline loads slowly (>500ms)

**Mitigation**:
- Denormalized `activity_feed` table (fan-out on write)
- Index on `(user_id, created_at DESC)`
- Pagination with cursor-based offset
- Cache recent feed in Redis (future optimization)

**Status**: ✅ Mitigated (design accounts for scale)

---

#### 4. Email Queue Delays

**Risk**: Email delivery delayed during high upload volume

**Impact**: Notification emails arrive 5-10 minutes late

**Mitigation**:
- Dedicated `email` queue with higher priority
- Increase Asynq worker concurrency for email tasks
- Batch notifications within 15-minute windows
- Monitor queue depth and latency

**Status**: ✅ Mitigated (configurable worker pool)

---

### Low-Priority Risks

#### 1. OAuth State Parameter Collision

**Risk**: Two users use same OAuth state value simultaneously

**Impact**: CSRF vulnerability

**Mitigation**:
- Use `crypto/rand` for state generation (128-bit entropy)
- Store state in Redis with 5-minute TTL
- Delete state after use (one-time token)

**Status**: ✅ Mitigated (design includes this)

---

#### 2. Email Template Rendering Errors

**Risk**: Invalid template syntax breaks email sending

**Impact**: Emails fail to send, users not notified

**Mitigation**:
- Template validation on startup (fail-fast)
- Unit tests for all email templates
- Fallback to plain-text email if HTML rendering fails
- Monitor email send failures (alert on >5% failure rate)

**Status**: ✅ Mitigated (test coverage + monitoring)

---

## External Dependencies Summary

| Dependency | Type | Required For | Blockers | Setup Time |
|------------|------|--------------|----------|------------|
| Google OAuth Credentials | External Service | Google OAuth login | None | 15 min |
| GitHub OAuth App | External Service | GitHub OAuth login | None | 10 min |
| SMTP Provider (AWS SES) | External Service | Email notifications (prod) | Domain verification | 30 min |
| MailHog (Docker) | Development Tool | Email notifications (dev) | None | 0 min (in docker-compose) |
| OAuth2 Library | Go Package | OAuth flows | None | 0 min (`go mod tidy`) |
| GitHub API Library | Go Package | GitHub OAuth | None | 0 min (`go mod tidy`) |
| Google API Library | Go Package | Google OAuth | None | 0 min (`go mod tidy`) |
| Email Library | Go Package | SMTP sending | None | 0 min (`go mod tidy`) |

---

## Pre-Sprint Setup Checklist

### For Development (Local)

- [x] Docker Compose includes MailHog service ✅ (already configured)
- [ ] Create Google OAuth test app (15 min)
- [ ] Create GitHub OAuth test app (10 min)
- [ ] Add OAuth credentials to `.env.local` (2 min)
- [ ] Run migrations 00007 and 00008 (1 min)
- [ ] Install new Go dependencies: `go mod tidy` (1 min)

**Total Setup Time**: ~30 minutes

---

### For Production Deployment

- [ ] Register Google OAuth production app (15 min)
- [ ] Register GitHub OAuth production app (10 min)
- [ ] Set up AWS SES (or alternative SMTP) (30 min)
- [ ] Configure SPF/DKIM/DMARC DNS records (20 min)
- [ ] Add OAuth credentials to secrets manager (5 min)
- [ ] Add SMTP credentials to secrets manager (5 min)
- [ ] Update production environment variables (5 min)
- [ ] Run database migrations (5 min)

**Total Setup Time**: ~2 hours

---

## Critical Path Analysis

### Week 1 Dependencies

**Day 1-2**: Database migrations
- No blockers
- Can proceed immediately

**Day 2-5**: OAuth implementation
- ⚠️ Requires test OAuth credentials (30-min setup)
- Can develop with mocked providers initially

**Day 5-7**: Session elevation
- ✅ Sprint 11 code complete
- No blockers

### Week 2 Dependencies

**Day 8-10**: Social features
- Depends on: Database migrations (00008)
- Can proceed in parallel with OAuth

**Day 10-12**: Email service
- ⚠️ Requires MailHog for testing (already available)
- Production SMTP not blocking for development

**Day 12-14**: E2E tests and security review
- Depends on: All features complete
- No external blockers

---

## Recommended Sprint Kickoff Actions

1. **Immediate** (before Sprint 12 starts):
   - [ ] Create Google OAuth test app
   - [ ] Create GitHub OAuth test app
   - [ ] Add credentials to developer documentation

2. **Day 1 of Sprint**:
   - [ ] Run database migrations in development
   - [ ] Confirm MailHog is running in Docker Compose
   - [ ] Review OAuth flow diagrams with team

3. **Production (can be done in parallel)**:
   - [ ] Set up AWS SES account
   - [ ] Configure sending domain DNS records
   - [ ] Request Google OAuth production credentials

---

## Conclusion

**Overall Sprint 12 Readiness**: ✅ **GREEN** - No critical blockers identified

**Key Strengths**:
- All infrastructure dependencies already in place (PostgreSQL, Redis, Asynq)
- Sprint 11 code provides solid foundation for session elevation
- Development environment fully configured (MailHog for email testing)
- Well-defined database schema and API contracts

**Recommended Actions**:
1. Set up OAuth test apps (30 minutes total)
2. Add OAuth credentials to `.env.local`
3. Begin implementation on Day 1 without delays

**Production Considerations**:
- SMTP provider setup can happen in parallel (not blocking development)
- OAuth production apps can be created closer to deployment
- No architectural decisions pending

---

**Sprint 12 is ready to begin implementation.**
