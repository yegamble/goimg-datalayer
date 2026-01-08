# Phase 2 Sprint Plan (Sprints 13-18)
> Comprehensive roadmap for goimg-datalayer Phase 2 development
> **Last Updated**: 2026-01-07
> **Status**: Sprint 13 IN PROGRESS, Sprints 14-18 PLANNED

---

## Executive Summary

Phase 2 extends the production-ready MVP with advanced features for competitive parity with Flickr/Chevereto and unique differentiation through IPFS storage.

### Phase 2 Timeline

```
Sprint 13: IPFS Storage Integration     [IN PROGRESS] (2 weeks)
     ↓
Sprint 14: Content Moderation + Guest Uploads (2 weeks) ⭐ P0 CRITICAL
     ↓
Sprint 15: AI NSFW + Advanced Search    (2 weeks) ⭐ P1 IMPORTANT
     ↓
Sprint 16: Sharing & Virality           (1 week) ⭐ P1 QUICK WINS
     ↓
Sprint 17: Organization Features        (2 weeks) ⭐ P2 NICE-TO-HAVE
     ↓
Sprint 18: Community Curation           (1 week) ⭐ P2 NICE-TO-HAVE

Total Duration: 10 weeks
```

### Phase 2 Achievements (Already Complete)

| Sprint | Features | Status |
|--------|----------|--------|
| Sprint 10 | Random login delay, HIBP password check, Prometheus security metrics | ✅ COMPLETE |
| Sprint 11 | Two-factor authentication (TOTP), backup codes, 2FA rate limiting | ✅ COMPLETE |
| Sprint 12 | OAuth (Google/GitHub), follow users, activity feeds, email notifications | ✅ COMPLETE |

---

## Sprint 13: IPFS Storage Integration

**Status**: 🔄 IN PROGRESS (Phase 1 Complete)
**Duration**: 2 weeks (Weeks 25-26)
**Focus**: Decentralized storage with IPFS

### Phase 1: Infrastructure (COMPLETE ✅)

| Component | Status | Files |
|-----------|--------|-------|
| IPFS Client | ✅ COMPLETE | `internal/infrastructure/storage/ipfs/client.go` |
| Storage Orchestrator | ✅ COMPLETE | `internal/infrastructure/storage/orchestrator/orchestrator.go` |
| Database Migration | ✅ COMPLETE | `migrations/00011_add_ipfs_fields.sql` |
| Domain Layer | ✅ COMPLETE | `internal/domain/gallery/ipfs_metadata.go` |

**Key Implementations**:
- IPFS Client: Pure stdlib HTTP client (no external dependencies), CIDv0/CIDv1 validation, 40+ unit tests
- Storage Orchestrator: Three modes (`primary_only`, `dual_sync`, `dual_async`), fallback retrieval
- Domain: IPFSMetadata value object with URI/GatewayURL helpers

### Phase 2: Application & HTTP Layer (PENDING)

**Complexity**: Simple (4-5 days)

| Task | Priority | Effort | Status |
|------|----------|--------|--------|
| `UploadToIPFSCommand` + handler | P0 | 1 day | 📋 Pending |
| `PinImageCommand` + handler | P0 | 0.5 days | 📋 Pending |
| `UnpinImageCommand` + handler | P0 | 0.5 days | 📋 Pending |
| `GetIPFSMetadataQuery` + handler | P0 | 0.5 days | 📋 Pending |
| HTTP endpoints (3 endpoints) | P1 | 1 day | 📋 Pending |
| OpenAPI spec updates | P1 | 0.5 days | 📋 Pending |
| Integration tests | P1 | 1 day | 📋 Pending |
| E2E tests (5-8 requests) | P1 | 0.5 days | 📋 Pending |

### HTTP Endpoints

```yaml
POST /api/v1/images/{id}/ipfs:
  summary: Upload image to IPFS
  request: { pin: true }
  response: { cid, gateway_url, pinned, pinned_at }
  auth: required (owner only)

DELETE /api/v1/images/{id}/ipfs:
  summary: Unpin image from IPFS
  response: 204 No Content
  auth: required (owner only)

GET /api/v1/images/{id}/ipfs:
  summary: Get IPFS metadata for image
  response: { cid, gateway_url, pinned, pinned_at }
  auth: public (if image is public)
```

### Agent Assignments

- **Lead**: senior-go-architect
- **Critical**: senior-secops-engineer (pinning credentials), backend-test-architect
- **Supporting**: cicd-guardian (IPFS testcontainers)

### Quality Gates

**Automated**:
- [ ] IPFS client unit tests passing (40+ tests)
- [ ] Orchestrator unit tests passing
- [ ] Application layer tests (85%+ coverage)
- [ ] Integration tests with IPFS testcontainer
- [ ] E2E tests (5-8 Newman requests)

**Manual**:
- [ ] Content retrieval via public gateway
- [ ] Pin persistence verification
- [ ] Dual-storage fallback testing

---

## Sprint 14: Content Moderation + Guest Uploads

**Status**: 📋 PLANNED
**Duration**: 2 weeks (Weeks 27-28)
**Priority**: **P0 CRITICAL**
**Focus**: Platform safety and growth

### Rationale

**Content Moderation**:
- Legal requirement: EU Digital Services Act (DSA) compliance (fines up to 6% of revenue)
- Market standard: Content moderation market grew to $12.48B in 2025
- Platform safety: Prevent spam, abuse, illegal content
- Competitive parity: Both Flickr and Chevereto have robust moderation

**Guest Uploads**:
- Chevereto core feature: Essential for event photographers, communities
- Growth driver: Viral sharing without registration friction
- Use cases: Wedding photographers, event organizers, forums

### Deliverables

#### Content Moderation System

**Database Migration** (`00012_create_moderation_tables.sql`):
```sql
CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id UUID NOT NULL REFERENCES users(id),
    image_id UUID NOT NULL REFERENCES images(id),
    reason VARCHAR(50) NOT NULL, -- spam, inappropriate, copyright, harassment, other
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, reviewing, resolved, dismissed
    resolved_by UUID REFERENCES users(id),
    resolved_at TIMESTAMPTZ,
    resolution TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_bans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    banned_by UUID NOT NULL REFERENCES users(id),
    reason TEXT NOT NULL,
    expires_at TIMESTAMPTZ, -- NULL = permanent
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id UUID,
    metadata JSONB,
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reports_status ON reports(status, created_at DESC);
CREATE INDEX idx_reports_image_id ON reports(image_id);
CREATE INDEX idx_bans_user_id ON user_bans(user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_audit_logs_action ON audit_logs(action, created_at DESC);
```

**Domain Layer** (extends existing `internal/domain/moderation/`):
- Report entity (already exists, implement repository)
- Ban entity (already exists, implement repository)
- AuditLog service (new)

**Infrastructure Layer**:
- `ReportRepository` (PostgreSQL) - ~300 lines
- `BanRepository` (PostgreSQL) - ~250 lines
- `AuditLogService` (PostgreSQL) - ~150 lines

**Application Layer**:
- Commands: `CreateReportCommand`, `ResolveReportCommand`, `BanUserCommand`, `UnbanUserCommand`
- Queries: `ListReportsQuery`, `GetReportQuery`, `ListBansQuery`, `GetUserBanStatusQuery`

**HTTP Layer**:
```yaml
POST /api/v1/reports:
  summary: Submit abuse report
  auth: any authenticated user
  rate_limit: 10/hour

GET /api/v1/moderation/reports:
  summary: List pending reports
  auth: moderator/admin only
  query: status, page, per_page

POST /api/v1/moderation/reports/{id}/resolve:
  summary: Resolve report
  auth: moderator/admin only
  request: { resolution: "dismiss|warn|remove|ban", reason: string }

POST /api/v1/users/{id}/ban:
  summary: Ban user
  auth: admin only
  request: { reason, expires_at? }

DELETE /api/v1/users/{id}/ban:
  summary: Unban user
  auth: admin only

GET /api/v1/moderation/audit-logs:
  summary: List audit logs
  auth: admin only
```

#### Guest Uploads

**Approach**: Pseudo-anonymous accounts (reuse User aggregate)

**Domain Layer**:
- Add `UserType` enum: `registered`, `guest`
- Add `GuestUser` factory: `NewGuestUser(ipAddress string) (*User, error)`
- Guest accounts auto-expire after 30 days

**Database Migration** (extend users table):
```sql
ALTER TABLE users ADD COLUMN user_type VARCHAR(20) NOT NULL DEFAULT 'registered';
ALTER TABLE users ADD COLUMN ip_address INET;
ALTER TABLE users ADD COLUMN expires_at TIMESTAMPTZ;

CREATE INDEX idx_users_guest_expiry ON users(user_type, expires_at)
    WHERE user_type = 'guest';
```

**Application Layer**:
- `CreateGuestSessionCommand` - Issue JWT without registration
- Modify `UploadImageCommand` - Accept guest accounts
- `CleanupExpiredGuestsCommand` - Asynq job (daily)

**HTTP Layer**:
```yaml
POST /api/v1/auth/guest:
  summary: Create guest session
  auth: none
  rate_limit: 10/hour per IP
  response: { access_token, expires_in, is_guest: true }

POST /api/v1/guest/images/{id}/claim:
  summary: Claim guest upload to account
  auth: required (registered user)
```

**Security Controls**:
- Rate limiting: 5 uploads/hour per IP (stricter than registered)
- ClamAV required: All guest uploads must pass malware scan
- Auto-moderation: Guest uploads start as `status: pending_review`
- IP banning: Ban IP after 3 malware attempts
- Expiry: Guest accounts auto-delete after 30 days

### Task Breakdown

| Task | Effort | Owner |
|------|--------|-------|
| Domain Layer (Report, Ban, GuestUser) | 2 days | senior-go-architect |
| Database Migration (moderation + guest) | 0.5 days | senior-go-architect |
| Infrastructure (Repositories, AuditLogService) | 2 days | senior-go-architect |
| Application Layer (Commands, Queries) | 3 days | senior-go-architect |
| HTTP Layer (ModerationHandler, RBAC middleware) | 2 days | senior-go-architect |
| Guest Upload Security Controls | 1 day | senior-secops-engineer |
| OpenAPI Spec | 0.5 days | senior-go-architect |
| E2E Tests (20+ Newman tests) | 1 day | test-strategist |
| Security Gate Review | 1 day | senior-secops-engineer |

**Total**: 13 days

### Security Gate S14

| Control ID | Requirement | Owner |
|------------|-------------|-------|
| S14-MOD-001 | Audit logging for all moderation actions | senior-secops-engineer |
| S14-MOD-002 | RBAC enforcement on moderation endpoints | senior-secops-engineer |
| S14-MOD-003 | Rate limiting on abuse reports (10/hour) | senior-secops-engineer |
| S14-MOD-004 | Banned users cannot login | senior-secops-engineer |
| S14-GUEST-001 | Guest upload rate limiting (5/hour per IP) | senior-secops-engineer |
| S14-GUEST-002 | ClamAV scanning on all guest uploads | senior-secops-engineer |
| S14-GUEST-003 | Auto-cleanup of expired guest accounts | senior-secops-engineer |
| S14-GUEST-004 | IP banning after 3 malware attempts | senior-secops-engineer |

### Agent Checkpoints

**Pre-Sprint**:
- [ ] senior-go-architect: Review moderation domain model (already exists)
- [ ] senior-secops-engineer: Review RBAC and audit logging architecture
- [ ] backend-test-architect: Plan integration test strategy

**Mid-Sprint (Day 7)**:
- [ ] senior-secops-engineer: Review guest upload security controls
- [ ] senior-go-architect: Review RBAC middleware implementation
- [ ] backend-test-architect: Coverage check (85%+ target)

**Pre-Merge**:
- [ ] senior-go-architect: Code review approval
- [ ] senior-secops-engineer: Security Gate S14 (8/8 controls)
- [ ] test-strategist: E2E tests passing (20+ requests)

---

## Sprint 15: AI NSFW Detection + Advanced Search

**Status**: 📋 PLANNED
**Duration**: 2 weeks (Weeks 29-30)
**Priority**: **P1 IMPORTANT**
**Focus**: Industry-standard moderation and discovery

### Rationale

**AI NSFW Detection**:
- Industry standard: All major platforms use AI + human moderation in 2026
- Hybrid approach: AI flags 90%+ of NSFW content automatically
- Cost-effective: $0.001-$0.005 per image (~$10-50/month for 10K images)
- Legal protection: Reduces risk of hosting illegal content

**Advanced Search**:
- Baseline expectation: All photo platforms offer date/size/dimension filters
- User retention: Power users need advanced filtering
- Discovery: Improves explore experience

### Deliverables

#### AI NSFW Detection

**Recommended Provider**: SightEngine or ModerateContent API
- Cost: ~$0.001-$0.005/image
- Features: Adult, violence, gore, hate symbols, text extraction
- Response time: <500ms async

**Infrastructure Layer** (`internal/infrastructure/security/ai_moderator.go`):
```go
type AIModerator interface {
    ModerateImage(ctx context.Context, imageURL string) (*ModerationResult, error)
}

type ModerationResult struct {
    NSFWScore     float64 // 0.0-1.0
    ViolenceScore float64 // 0.0-1.0
    Source        string  // "sightengine" | "moderate_content"
    RawResponse   json.RawMessage
}
```

**Database Migration** (`00013_add_moderation_scores.sql`):
```sql
ALTER TABLE images ADD COLUMN nsfw_score DECIMAL(4,3);
ALTER TABLE images ADD COLUMN violence_score DECIMAL(4,3);
ALTER TABLE images ADD COLUMN ai_moderated_at TIMESTAMPTZ;
ALTER TABLE images ADD COLUMN ai_moderation_source VARCHAR(50);

CREATE INDEX idx_images_nsfw ON images(nsfw_score) WHERE nsfw_score > 0.5;
```

**Application Layer**:
- `ModerateImageCommand` - Async job (Asynq) after upload
- `OverrideAIModerationCommand` - Admin override for false positives

**Background Job**:
```go
// After image upload, queue moderation job
asynq.NewTask("image:moderate", payload)
```

#### Advanced Search Filters

**Approach**: PostgreSQL full-text search extensions (no Elasticsearch needed)

**Database Migration** (`00014_add_search_indexes.sql`):
```sql
-- Full-text search vector
ALTER TABLE images ADD COLUMN search_vector tsvector;

CREATE INDEX idx_images_search ON images USING GIN(search_vector);

-- Trigger for auto-update
CREATE TRIGGER images_search_update
    BEFORE INSERT OR UPDATE ON images
    FOR EACH ROW EXECUTE FUNCTION
    tsvector_update_trigger(search_vector, 'pg_catalog.english', title, description);

-- Fuzzy matching
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_images_title_trigram ON images USING GIN(title gin_trgm_ops);
```

**Updated Search Query**:
```yaml
GET /api/v1/search/images:
  query_params:
    q: search query (full-text)
    uploaded_after: ISO 8601 date
    uploaded_before: ISO 8601 date
    min_width: pixels
    max_width: pixels
    min_height: pixels
    max_height: pixels
    min_size: bytes
    max_size: bytes
    aspect_ratio: portrait|landscape|square
```

### Task Breakdown

| Task | Effort | Owner |
|------|--------|-------|
| AI Moderator API Integration | 3 days | senior-go-architect |
| Async Moderation Job (Asynq) | 1 day | senior-go-architect |
| NSFW Score Storage (migration, domain) | 1 day | senior-go-architect |
| Admin Override Controls | 1 day | senior-go-architect |
| Full-text Search Migration | 1 day | senior-go-architect |
| Advanced Search Filters (date, size, dimensions) | 2 days | senior-go-architect |
| E2E Tests (15+ Newman tests) | 1 day | test-strategist |

**Total**: 11 days

### Security Gate S15

| Control ID | Requirement | Owner |
|------------|-------------|-------|
| S15-AI-001 | API keys encrypted at rest | senior-secops-engineer |
| S15-AI-002 | AI decisions logged for audit | senior-secops-engineer |
| S15-AI-003 | Manual override capability | senior-secops-engineer |
| S15-SEARCH-001 | Query parameter validation (SQL injection prevention) | senior-secops-engineer |

---

## Sprint 16: Sharing & Virality

**Status**: 📋 PLANNED
**Duration**: 1 week (Week 31)
**Priority**: **P1 QUICK WINS**
**Focus**: Low-effort, high-impact sharing features

### Rationale

- Low effort, high ROI: Simple endpoints with massive virality impact
- Industry standard: All major platforms support oEmbed
- Quality signal: Rich previews improve social CTR by 10%+

### Deliverables

#### oEmbed Support

**HTTP Endpoint**:
```yaml
GET /api/v1/oembed:
  query_params:
    url: https://goimg.com/i/{id}
    format: json
  response:
    type: photo
    version: "1.0"
    title: "Image Title"
    author_name: "Username"
    author_url: "https://goimg.com/u/{username}"
    provider_name: "goimg"
    provider_url: "https://goimg.com"
    url: "https://goimg.com/i/{id}/medium"
    width: 800
    height: 600
```

**HTML Discovery** (add to image pages):
```html
<link rel="alternate" type="application/json+oembed"
      href="https://goimg.com/api/v1/oembed?url=https://goimg.com/i/{id}&format=json">
```

#### Social Media Preview Cards

**Open Graph** (add to image pages):
```html
<meta property="og:type" content="website">
<meta property="og:url" content="https://goimg.com/i/{id}">
<meta property="og:title" content="{image.title}">
<meta property="og:description" content="{image.description}">
<meta property="og:image" content="https://goimg.com/i/{id}/medium">
<meta property="og:image:width" content="800">
<meta property="og:image:height" content="600">
```

**Twitter Cards**:
```html
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="{image.title}">
<meta name="twitter:description" content="{image.description}">
<meta name="twitter:image" content="https://goimg.com/i/{id}/medium">
```

### Task Breakdown

| Task | Effort | Owner |
|------|--------|-------|
| oEmbed Endpoint (JSON response) | 1 day | senior-go-architect |
| oEmbed Discovery Meta Tags | 0.5 days | senior-go-architect |
| Open Graph Meta Tags | 0.5 days | senior-go-architect |
| Twitter Card Meta Tags | 0.5 days | senior-go-architect |
| E2E Tests (8+ Newman tests) | 0.5 days | test-strategist |

**Total**: 4 days

**No Security Gate** (low-risk, read-only features)

---

## Sprint 17: Organization Features

**Status**: 📋 PLANNED
**Duration**: 2 weeks (Weeks 32-33)
**Priority**: **P2 NICE-TO-HAVE**
**Focus**: Power user organization features

### Deliverables

#### Nested Albums

**Database Migration** (`00015_add_nested_albums.sql`):
```sql
ALTER TABLE albums ADD COLUMN parent_album_id UUID REFERENCES albums(id);

-- Prevent circular references
ALTER TABLE albums ADD CONSTRAINT no_circular_parent
    CHECK (id != parent_album_id);

-- Index for tree queries
CREATE INDEX idx_albums_parent ON albums(parent_album_id);
```

**Domain Layer**:
- Add `ParentAlbumID` to Album entity
- Add `GetAncestors()` method (recursive CTE)
- Limit nesting depth to 3 levels

**Application Layer**:
- Update `CreateAlbumCommand` with optional `parent_album_id`
- Add `MoveAlbumCommand` to change parent
- Add `GetAlbumTreeQuery` (recursive CTE)

#### Custom Variant Sizes

**Configuration** (environment variables):
```bash
# Default variants (always generated)
IMAGE_VARIANTS=thumbnail:150,small:320,medium:800,large:1600

# Custom variants (optional, admin-configurable)
CUSTOM_VARIANTS=wallpaper:2560,print:3000,banner:1920x400
```

**Infrastructure Layer**:
- Update `ImageProcessor` to read custom variants from config
- Support both square (`:size`) and fixed (`widthxheight`) formats

### Task Breakdown

| Task | Effort | Owner |
|------|--------|-------|
| Nested Albums Migration | 1 day | senior-go-architect |
| Domain Layer (recursive methods) | 1 day | senior-go-architect |
| Application Layer (tree queries) | 2 days | senior-go-architect |
| HTTP Layer (breadcrumb support) | 1 day | senior-go-architect |
| Custom Variant Configuration | 2 days | senior-go-architect |
| E2E Tests (12+ Newman tests) | 1 day | test-strategist |

**Total**: 9 days

**No Security Gate** (internal features, low risk)

---

## Sprint 18: Community Curation

**Status**: 📋 PLANNED
**Duration**: 1 week (Week 34)
**Priority**: **P2 NICE-TO-HAVE**
**Focus**: Discovery and engagement features

### Deliverables

#### Trending Tags

**Database Query**:
```sql
SELECT t.name, COUNT(*) as usage_count
FROM image_tags it
JOIN tags t ON it.tag_id = t.id
JOIN images i ON it.image_id = i.id
WHERE i.created_at > NOW() - INTERVAL '7 days'
  AND i.visibility = 'public'
GROUP BY t.id
ORDER BY usage_count DESC
LIMIT 20;
```

**Redis Caching**:
- Key: `trending:tags:{period}` (day, week, month)
- TTL: 15 minutes
- Refresh on cache miss

**HTTP Endpoint**:
```yaml
GET /api/v1/tags/trending:
  query_params:
    period: day|week|month (default: week)
    limit: 10-50 (default: 20)
  response:
    - name: sunset
      usage_count: 1234
      change_percent: +15.2
```

#### Featured/Staff Picks

**Database Migration** (`00016_add_featured_images.sql`):
```sql
ALTER TABLE images ADD COLUMN featured BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE images ADD COLUMN featured_at TIMESTAMPTZ;
ALTER TABLE images ADD COLUMN featured_by UUID REFERENCES users(id);

CREATE INDEX idx_images_featured ON images(featured, featured_at DESC)
    WHERE featured = TRUE;
```

**HTTP Endpoints**:
```yaml
POST /api/v1/moderation/images/{id}/feature:
  summary: Feature image (staff pick)
  auth: admin only
  response: 204 No Content

DELETE /api/v1/moderation/images/{id}/feature:
  summary: Unfeature image
  auth: admin only
  response: 204 No Content

GET /api/v1/explore/featured:
  summary: List featured images
  auth: public
```

### Task Breakdown

| Task | Effort | Owner |
|------|--------|-------|
| Trending Tags Query + Redis Cache | 2 days | senior-go-architect |
| Featured Flag Migration | 0.5 days | senior-go-architect |
| Featured Endpoints | 1 day | senior-go-architect |
| Admin Controls | 0.5 days | senior-go-architect |
| E2E Tests (10+ Newman tests) | 0.5 days | test-strategist |

**Total**: 5 days

**No Security Gate** (read-only features, admin-only mutations with existing RBAC)

---

## Phase 3 Backlog (Deferred Features)

These features are intentionally deferred to Phase 3:

| Feature | Reason | Estimated Effort |
|---------|--------|------------------|
| **Watermarking** | Niche feature (pro photographers only), medium effort for low ROI | 1 week |
| **Account Tiers/Subscriptions** | High complexity (billing, Stripe), requires product-market fit validation | 3-4 weeks |
| **Remote IPFS Pinning (Pinata/Infura)** | Nice-to-have redundancy, not critical for launch | 1 week |
| **Categories** | Tags already provide categorization, editorial overhead | 1 week |
| **QR Codes for Images** | Low impact, external tools exist | 2-3 days |
| **Shortened URLs** | Low priority, external services work | 2-3 days |
| **Elasticsearch Integration** | Overkill for current scale, PostgreSQL full-text sufficient | 3 weeks |
| **Video Support** | Different product category, requires transcoding infrastructure | 6+ weeks |

---

## Technical Dependencies

### Dependency Graph

```
Sprint 13 (IPFS):
  └─ No blocking dependencies for Sprints 14-18

Sprint 14 (Content Moderation + Guest Uploads):
  ├─ Content Moderation unlocks:
  │   └─ Featured/Staff Picks admin controls (Sprint 18)
  └─ No external dependencies

Sprint 15 (AI NSFW + Advanced Search):
  ├─ AI NSFW Detection: No dependencies
  └─ Advanced Search: Extends existing search (Sprint 6)

Sprint 16 (oEmbed + Social Cards):
  └─ No dependencies

Sprint 17 (Nested Albums + Custom Variants):
  └─ No dependencies

Sprint 18 (Trending Tags + Featured Picks):
  └─ Featured Picks depends on Sprint 14 RBAC
```

### Shared Infrastructure (Already Complete)

| Component | Sprint | Status |
|-----------|--------|--------|
| RBAC middleware | Sprint 6 | ✅ COMPLETE |
| Rate limiting (Redis) | Sprint 4 | ✅ COMPLETE |
| Audit logging | Sprint 9 | ✅ COMPLETE |
| Background jobs (Asynq) | Sprint 6 | ✅ COMPLETE |
| ClamAV integration | Sprint 5 | ✅ COMPLETE |
| Image processing (bimg) | Sprint 5 | ✅ COMPLETE |

---

## Risk Mitigation

### High Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| AI API costs scale with uploads | Medium | Cache results, batch processing, budget cap $50/month |
| Content moderation load exceeds capacity | High | AI detection reduces 90% manual review, auto-ban repeat offenders |
| Guest upload abuse (spam, illegal content) | High | Rate limiting (5/hour), ClamAV, auto-delete 30 days, IP banning |

### Medium Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| IPFS + S3 doubles storage costs | Medium | Make IPFS optional (config flag), store originals only on IPFS |
| Advanced search performance | Medium | PostgreSQL indexes, pagination, Redis caching |

### Low Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| Nested albums recursive query performance | Low | Limit nesting to 3 levels, use CTEs with indexes |
| Feature creep (too many P2/P3 features) | Low | Strict prioritization, defer P3 to Phase 3 |

---

## Success Metrics

### Sprint 14
- Abuse reports created: > 0 (feature usage)
- Moderation actions: > 0 (admin engagement)
- Guest uploads: 10% of total uploads
- Guest conversion rate: 5% claim to account
- Security Gate S14: 8/8 controls passed

### Sprint 15
- AI NSFW detection accuracy: > 90%
- False positive rate: < 5%
- Advanced search usage: 20% of searches use filters
- Security Gate S15: 4/4 controls passed

### Sprint 16
- oEmbed embeds: > 0 (virality metric)
- Social preview CTR: +10% vs plain links

### Sprint 17
- Nested albums: 5% of users create sub-albums
- Custom variants: 10% of admins configure custom sizes

### Sprint 18
- Trending tags page views: 10% of explore traffic
- Featured images: 1-5 new features per week

---

## Competitive Scorecard (After Phase 2)

| Feature Category | Flickr | Chevereto | goimg (Phase 2) | Winner |
|------------------|--------|-----------|-----------------|--------|
| Core Image Hosting | ✅ | ✅ | ✅ | TIE |
| Social Features | ✅ | ✅ | ✅ | TIE |
| Content Moderation | ✅ AI+Human | ✅ Basic | ✅ AI+Human | TIE |
| Guest Uploads | ❌ | ✅ | ✅ | Chevereto/goimg |
| Self-Hosted | ❌ | ✅ | ✅ | Chevereto/goimg |
| **IPFS Storage** | ❌ | ❌ | ✅ | **goimg** 🏆 |
| **Modern API** | ✅ Good | ✅ Basic | ✅ Excellent | **goimg** 🏆 |
| **Security** | ✅ Good | ⚠️ Basic | ✅ Excellent | **goimg** 🏆 |

---

## Next Steps

1. **Complete Sprint 13 Phase 2** - IPFS application/HTTP layer (4-5 days)
2. **Kick off Sprint 14** - Content Moderation + Guest Uploads (CRITICAL)
3. **Security Gate S14 Review** - 8 controls
4. **Proceed to Sprint 15** - AI NSFW + Advanced Search
5. **Evaluate P2 features** (Sprints 17-18) based on user feedback after Sprint 16

---

**Document Maintainer**: scrum-master
**Review Cycle**: End of each sprint
