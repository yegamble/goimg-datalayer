# Phase 2 Feature Prioritization
> Strategic analysis based on Flickr/Chevereto competitive landscape (2026)
> **Last Updated**: 2026-01-07
> **Analyst**: image-gallery-expert

---

## Executive Summary

Based on competitive analysis and 2026 market trends, this document prioritizes remaining Phase 2 features into four tiers (P0-P3) with recommended sprint assignments and technical dependencies.

**Key Findings**:
1. **Content Moderation is P0 Critical** - Regulatory compliance (DSA) and market standards demand robust moderation
2. **Guest Uploads are P0** - Essential for growth and competitive with Chevereto
3. **Advanced Search is P1** - Expected baseline for photo platforms in 2026
4. **AI NSFW Detection is P1** - Industry standard, reduces manual moderation burden
5. **Account Tiers are P2** - Revenue generator, but not blocking MVP

---

## Prioritized Feature Matrix

| Feature | Priority | Sprint | Effort | Impact | Rationale |
|---------|----------|--------|--------|--------|-----------|
| **Content Moderation Suite** | **P0** | 14 | High | Critical | Regulatory compliance, platform safety |
| **Guest Uploads** | **P0** | 14 | Medium | High | Core Chevereto feature, growth driver |
| **AI NSFW Detection** | **P1** | 15 | Medium | High | Industry standard, reduces manual load |
| **Advanced Search Filters** | **P1** | 15 | Medium | High | Baseline expectation for 2026 platforms |
| **oEmbed Support** | **P1** | 16 | Low | Medium | Easy virality, low effort/high ROI |
| **Open Graph/Twitter Cards** | **P1** | 16 | Low | Medium | Social sharing quality signal |
| **Nested Albums** | **P2** | 17 | Medium | Medium | Nice-to-have organization feature |
| **Custom Variant Sizes** | **P2** | 17 | Low | Low | Power user feature, configurable |
| **Trending Tags** | **P2** | 18 | Low | Medium | Discovery feature, engagement driver |
| **Featured/Staff Picks** | **P2** | 18 | Low | Medium | Community curation, low effort |
| **Watermarking** | **P3** | Phase 3 | Medium | Low | Niche feature, low demand |
| **Account Tiers/Subscriptions** | **P3** | Phase 3 | High | Medium | Revenue model, requires billing |
| **Categories** | **P3** | Phase 3 | Medium | Low | Less effective than tags |
| **QR Codes** | **P3** | Phase 3 | Low | Low | Nice-to-have, minimal impact |
| **Shortened URLs** | **P3** | Phase 3 | Low | Low | Low priority, external tools exist |

---

## P0: Critical Features (Must Complete Phase 2)

### 1. Content Moderation Suite (Sprint 14)

**Rationale**:
- **Regulatory Compliance**: EU Digital Services Act (DSA) requires platforms to act quickly on harmful content
  - Fines up to 6% of global annual turnover for non-compliance
  - Must support user reports, transparency reports, and audits
- **Market Standard**: Content moderation market grew to $12.48B in 2025, expected to reach $42.36B by 2035
- **Platform Safety**: Required for public launch to prevent abuse, spam, and illegal content
- **Competitive Parity**: Both Flickr and Chevereto have robust moderation systems

**Scope**:
- Abuse reporting system (user-initiated reports)
- Admin moderation queue (pending reports with filters)
- User ban system (temporary and permanent bans)
- Audit logging for all moderation actions (GDPR/SOC2 compliance)
- Moderation workflows (warn, remove content, ban user)

**Implementation**:
```
Domain Layer:
  - Report entity (image_id, reporter_id, reason, status, resolved_by)
  - Ban entity (user_id, banned_by, reason, expires_at)
  - AuditLog entity (actor_id, action, entity_type, entity_id, metadata)

Infrastructure:
  - PostgreSQL repositories (ReportRepository, BanRepository, AuditLogRepository)
  - RBAC middleware (admin, moderator roles)

Application:
  - Commands: CreateReport, ResolveReport, BanUser, UnbanUser
  - Queries: ListReports (with filters), GetReport, ListBans

HTTP:
  - POST /api/v1/reports (any user)
  - GET /api/v1/moderation/reports (admin/moderator)
  - POST /api/v1/moderation/reports/{id}/resolve (admin/moderator)
  - POST /api/v1/users/{id}/ban (admin only)
  - DELETE /api/v1/users/{id}/ban (admin only)
```

**Dependencies**: None (can implement immediately)

**Estimated Effort**: 1.5 weeks (8-10 working days)

---

### 2. Guest Uploads (Sprint 14)

**Rationale**:
- **Chevereto Core Feature**: Essential for self-hosted platforms (wedding photographers, event organizers)
- **Growth Driver**: Enables viral sharing and network effects without registration friction
- **Use Cases**:
  - Event attendees upload photos without accounts
  - Quick image hosting for forums/communities
  - Temporary file sharing (with expiration)
- **Competitive Advantage**: Chevereto's strength over Flickr (Flickr requires accounts)

**Scope**:
- Upload endpoint without authentication
- Temporary image storage (auto-delete after 30 days unless claimed)
- Guest upload rate limiting (10/hour per IP)
- Optional account claiming (convert guest upload to owned image)
- Moderation queue integration (flag guest uploads for review)

**Implementation**:
```
Domain Layer:
  - GuestImage entity (extends Image with guest_ip, expires_at, claim_token)
  - Visibility: guest uploads default to "unlisted" (not public)

Infrastructure:
  - IP-based rate limiting (Redis)
  - Guest upload cleanup job (Asynq background task)

Application:
  - Commands: UploadGuestImage, ClaimGuestImage, DeleteExpiredGuestImages
  - Queries: GetGuestImage, ListGuestImages (by IP)

HTTP:
  - POST /api/v1/guest/upload (no auth required)
  - GET /api/v1/guest/images/{id}?token={claim_token}
  - POST /api/v1/guest/images/{id}/claim (requires auth)
```

**Security Considerations**:
- Rate limiting: 10 uploads/hour per IP (prevent abuse)
- ClamAV scanning (same as authenticated uploads)
- Auto-delete after 30 days (prevent storage bloat)
- Claim token validation (prevent hijacking)
- Moderation queue review (flag suspicious uploads)

**Dependencies**: Sprint 14 Content Moderation (for guest upload review)

**Estimated Effort**: 1 week (5-6 working days)

---

## P1: Important Features (High Impact)

### 3. AI NSFW Detection (Sprint 15)

**Rationale**:
- **Industry Standard**: Flickr, Chevereto (via plugins), and all major platforms use AI moderation
- **Hybrid Moderation**: 2026 best practice is AI + human review (not manual-only)
- **Reduces Moderator Burden**: Flags 90%+ of NSFW content automatically
- **Legal Risk Mitigation**: Protects platform from hosting illegal content

**Recommended Solution**: Use third-party API (SightEngine, ModerateContent, or Azure Content Moderator)
- Avoid building in-house models (expensive, requires ML expertise)
- APIs provide: adult content detection, violence, gore, hate symbols, text extraction (OCR)
- Cost: ~$0.001-$0.005 per image (affordable for 10K images/month = $10-$50/month)

**Scope**:
- Integration with AI moderation API (async processing)
- NSFW confidence score (0-1) stored in database
- Auto-flagging for review (threshold: confidence > 0.85)
- Admin override (false positives)
- Audit trail for AI decisions

**Implementation**:
```
Infrastructure:
  - AIModeratorClient (HTTP client for SightEngine/ModerateContent API)
  - Async job: ModerateImageJob (Asynq task after upload)

Domain Layer:
  - ModerationScore value object (nsfw_score, violence_score, source)

Application:
  - Commands: ModerateImage, OverrideAIModeration
  - Queries: GetModerationScores

Database:
  - Add columns: nsfw_score, violence_score, ai_moderated_at
  - Index: nsfw_score (for filtering)
```

**Dependencies**: None (can run in parallel with Sprint 14)

**Estimated Effort**: 1 week (5-6 working days)

---

### 4. Advanced Search Filters (Sprint 15)

**Rationale**:
- **Baseline Expectation**: All photo platforms in 2026 offer advanced filtering
- **User Retention**: Power users need date range, size, dimension filters
- **Discovery**: Improves explore/discovery experience

**Scope**:
- Date range filters (uploaded_after, uploaded_before)
- File size filters (min_size, max_size in MB)
- Dimension filters (min_width, max_width, min_height, max_height)
- Aspect ratio filters (portrait, landscape, square)
- EXIF filters (camera, lens) - Phase 3 (requires EXIF storage)

**Implementation**:
```
Application Layer:
  - Update SearchImagesQuery with new filters
  - PostgreSQL WHERE clauses for filters

HTTP Layer:
  - GET /api/v1/search/images?q=sunset&uploaded_after=2026-01-01&min_width=1920

OpenAPI:
  - Add query parameters to /search/images endpoint
```

**Dependencies**: None (extends existing search)

**Estimated Effort**: 3-4 days (simple query parameter addition)

---

### 5. oEmbed Support (Sprint 16)

**Rationale**:
- **Virality**: Enables rich embeds in WordPress, Medium, Slack, Discord, etc.
- **Low Effort, High ROI**: Simple JSON endpoint, massive distribution impact
- **Industry Standard**: All major platforms (Flickr, Instagram, YouTube) support oEmbed

**Scope**:
- oEmbed discovery endpoint: `GET /api/v1/oembed?url={image_url}&format=json`
- Returns: `{ type: "photo", url, width, height, title, author_name, provider_name }`
- HTML meta tags for oEmbed discovery: `<link rel="alternate" type="application/json+oembed" href="...">`

**Implementation**:
```
HTTP Layer:
  - OEmbedHandler with single GET endpoint
  - Parse URL, extract image_id, return JSON response

OpenAPI:
  - Add /oembed endpoint
```

**Dependencies**: None

**Estimated Effort**: 1-2 days (trivial endpoint)

---

### 6. Social Media Preview Cards (Sprint 16)

**Rationale**:
- **Quality Signal**: Rich previews on Twitter, Facebook, LinkedIn improve click-through rates
- **Branding**: Consistent previews across social platforms
- **Low Effort**: Just HTML meta tags

**Scope**:
- Open Graph tags (og:image, og:title, og:description, og:url)
- Twitter Card tags (twitter:card, twitter:image, twitter:title)
- Applied to image detail pages (`/images/{id}`)

**Implementation**:
```
HTTP Layer:
  - Update image detail page HTML template with meta tags
  - Use medium variant as og:image (optimal size for previews)

Example:
<meta property="og:image" content="https://goimg.com/i/{id}/medium">
<meta property="og:title" content="{image.title}">
<meta property="og:description" content="{image.description}">
<meta name="twitter:card" content="summary_large_image">
```

**Dependencies**: None

**Estimated Effort**: 1 day (template changes only)

---

## P2: Nice-to-Have Features (Medium Impact)

### 7. Nested Albums (Sprint 17)

**Rationale**:
- **Power User Feature**: Flickr and Chevereto support sub-albums for complex organization
- **Not Blocking**: Most users use single-level albums (80/20 rule)
- **Medium Effort**: Requires database schema changes and UI updates

**Scope**:
- Allow albums to have a parent_album_id (self-referencing foreign key)
- Max nesting depth: 3 levels (prevent infinite recursion)
- Breadcrumb navigation (Parent > Child > Grandchild)
- Recursive queries for album tree

**Implementation**:
```
Database Migration:
  - ALTER TABLE albums ADD COLUMN parent_album_id UUID REFERENCES albums(id)
  - CHECK constraint: prevent circular references

Domain Layer:
  - Album.ParentAlbumID (optional)
  - Album.GetAncestors() (recursive query)

Application:
  - Commands: CreateAlbum (with parent_id), MoveAlbum (change parent)
  - Queries: GetAlbumTree (recursive CTE)
```

**Dependencies**: None

**Estimated Effort**: 1 week (6-8 working days)

---

### 8. Custom Variant Sizes (Sprint 17)

**Rationale**:
- **Power User Feature**: Photographers want specific dimensions (e.g., 2560x1440 for wallpapers)
- **Low Effort**: Configuration-driven variant generation
- **Differentiation**: Chevereto supports custom sizes, Flickr has fixed sizes

**Scope**:
- Admin-configurable variant sizes (via environment variables)
- Default variants: thumbnail (150px), small (320px), medium (800px), large (1600px)
- Custom variants: `CUSTOM_VARIANTS=wallpaper:2560x1440,print:3000x3000`

**Implementation**:
```
Infrastructure:
  - Update ImageProcessor to read CUSTOM_VARIANTS env var
  - Generate variants dynamically based on config

Configuration:
  CUSTOM_VARIANTS=wallpaper:2560x1440,print:3000x3000,banner:1920x400
```

**Dependencies**: None

**Estimated Effort**: 2-3 days (configuration changes)

---

### 9. Trending Tags (Sprint 18)

**Rationale**:
- **Discovery Feature**: Highlights popular content, drives engagement
- **Low Effort**: Simple COUNT query over time window
- **Flickr Core Feature**: "Explore" page shows trending tags

**Scope**:
- Endpoint: `GET /api/v1/tags/trending?period=week`
- Periods: day, week, month
- Redis caching (refresh every 15 minutes)
- Returns: tag name, usage count, change % from previous period

**Implementation**:
```
Application Layer:
  - Query: GetTrendingTags (period parameter)
  - PostgreSQL query: COUNT tags over time window

Caching:
  - Redis key: trending:tags:week
  - TTL: 15 minutes
```

**Dependencies**: None

**Estimated Effort**: 2-3 days

---

### 10. Featured/Staff Picks (Sprint 18)

**Rationale**:
- **Community Curation**: Highlights quality content, encourages creators
- **Low Effort**: Admin-only flag on images
- **Flickr Core Feature**: "Explore" curated section

**Scope**:
- Add `featured` boolean column to images table
- Admin endpoint: `POST /api/v1/moderation/images/{id}/feature`
- Public endpoint: `GET /api/v1/explore/featured`

**Implementation**:
```
Database Migration:
  - ALTER TABLE images ADD COLUMN featured BOOLEAN DEFAULT FALSE
  - CREATE INDEX idx_images_featured ON images(featured, created_at)

Application:
  - Command: FeatureImage (admin only)
  - Query: ListFeaturedImages (public)
```

**Dependencies**: Sprint 14 Content Moderation (admin RBAC)

**Estimated Effort**: 1-2 days

---

## P3: Future Phase Features (Phase 3)

### 11. Watermarking

**Rationale**:
- **Niche Feature**: Only relevant for professional photographers selling prints
- **Chevereto Feature**: Supported via plugins, not core
- **Medium Effort**: Requires image composition with text/logo overlay

**Deferred Because**:
- Low demand in MVP user base
- Can be added as plugin/extension later
- Medium implementation complexity for low ROI

**Future Sprint**: Phase 3, Sprint 19+

---

### 12. Account Tiers/Subscriptions

**Rationale**:
- **Revenue Model**: Freemium (free tier + paid Pro tier)
- **High Effort**: Requires billing integration (Stripe), tier management, quota enforcement
- **Not Blocking MVP**: Can launch with single free tier

**Deferred Because**:
- High complexity (billing, invoicing, tax compliance)
- Requires product-market fit validation first
- Can be added after user base grows

**Future Sprint**: Phase 3, Sprint 20+

---

### 13. Categories

**Rationale**:
- **Less Effective Than Tags**: Tags are more flexible, user-driven
- **Flickr Uses Tags**: No rigid category structure
- **Chevereto Has Categories**: But tags are primary organization

**Deferred Because**:
- Tags already provide categorization
- Categories require editorial curation (overhead)
- Low user demand vs effort

**Future Sprint**: Phase 3 (or never)

---

### 14. QR Codes for Images

**Rationale**:
- **Niche Feature**: Useful for event photography (print QR codes)
- **Low Effort**: Simple QR code generation library
- **Low Impact**: Minimal user demand

**Deferred Because**:
- Not core to image hosting workflow
- Can be implemented as simple utility later

**Future Sprint**: Phase 3, Sprint 21+

---

### 15. Shortened URLs

**Rationale**:
- **Low Priority**: External tools exist (bit.ly, TinyURL)
- **Low Effort**: Simple hash mapping
- **Low Impact**: URLs already reasonably short

**Deferred Because**:
- Not differentiating feature
- External services work fine
- Low ROI

**Future Sprint**: Phase 3 (low priority)

---

## Technical Dependencies

### Dependency Graph

```
Sprint 14:
  - Content Moderation (no dependencies) ─┐
  - Guest Uploads (depends on Content Moderation for review) ─┘

Sprint 15:
  - AI NSFW Detection (no dependencies)
  - Advanced Search Filters (no dependencies)

Sprint 16:
  - oEmbed Support (no dependencies)
  - Social Media Preview Cards (no dependencies)

Sprint 17:
  - Nested Albums (no dependencies)
  - Custom Variant Sizes (no dependencies)

Sprint 18:
  - Trending Tags (no dependencies)
  - Featured/Staff Picks (depends on Sprint 14 RBAC)
```

### Key Dependencies

1. **Sprint 14 Content Moderation** unlocks:
   - Guest upload review (Sprint 14)
   - Featured/Staff Picks admin controls (Sprint 18)

2. **Sprint 13 IPFS** (already in progress):
   - No blocking dependencies for Phase 2 features
   - IPFS Phase 2 can run in parallel with Sprints 14-18

3. **Shared Infrastructure** (already complete):
   - RBAC middleware (Sprint 6)
   - Rate limiting (Sprint 4)
   - Audit logging (Sprint 9)
   - Background jobs (Sprint 6 - Asynq)

---

## Recommended Sprint Assignments

### Sprint 14: Safety & Growth (2 weeks)
**Focus**: Content moderation + guest uploads

| Feature | Effort | Owner |
|---------|--------|-------|
| Content Moderation Suite | 8 days | senior-go-architect, senior-secops-engineer |
| Guest Uploads | 5 days | senior-go-architect |
| E2E Tests | 1 day | test-strategist |

**Deliverables**:
- Abuse reporting system
- Admin moderation queue
- User ban system
- Audit logging
- Guest upload endpoint
- Guest upload cleanup job
- 20+ Newman E2E tests

**Security Gate S14**:
- S14-MOD-001: Audit logging for all moderation actions
- S14-MOD-002: RBAC enforcement on moderation endpoints
- S14-MOD-003: Rate limiting on abuse reports (10/hour)
- S14-GUEST-001: Guest upload rate limiting (10/hour per IP)
- S14-GUEST-002: ClamAV scanning on guest uploads
- S14-GUEST-003: Auto-cleanup of expired guest uploads

---

### Sprint 15: AI & Discovery (2 weeks)
**Focus**: AI moderation + search enhancements

| Feature | Effort | Owner |
|---------|--------|-------|
| AI NSFW Detection | 6 days | senior-go-architect, senior-secops-engineer |
| Advanced Search Filters | 4 days | senior-go-architect |
| E2E Tests | 1 day | test-strategist |

**Deliverables**:
- AI moderation API integration (SightEngine/ModerateContent)
- Async moderation job (Asynq)
- NSFW score storage and indexing
- Admin override for false positives
- Advanced search filters (date, size, dimensions)
- 15+ Newman E2E tests

**Security Gate S15**:
- S15-AI-001: API keys encrypted at rest
- S15-AI-002: AI decisions logged for audit
- S15-AI-003: Manual override capability
- S15-SEARCH-001: Query parameter validation (SQL injection prevention)

---

### Sprint 16: Sharing & Virality (1 week)
**Focus**: Low-effort, high-impact sharing features

| Feature | Effort | Owner |
|---------|--------|-------|
| oEmbed Support | 2 days | senior-go-architect |
| Social Media Preview Cards | 1 day | senior-go-architect |
| E2E Tests | 1 day | test-strategist |

**Deliverables**:
- oEmbed JSON endpoint
- Open Graph meta tags
- Twitter Card meta tags
- oEmbed discovery meta tags
- 8+ Newman E2E tests

**No Security Gate** (low-risk features)

---

### Sprint 17: Organization (2 weeks)
**Focus**: Power user organization features

| Feature | Effort | Owner |
|---------|--------|-------|
| Nested Albums | 6 days | senior-go-architect |
| Custom Variant Sizes | 3 days | senior-go-architect |
| E2E Tests | 1 day | test-strategist |

**Deliverables**:
- Nested album schema migration
- Recursive album tree queries
- Breadcrumb navigation
- Custom variant configuration
- 12+ Newman E2E tests

**No Security Gate** (internal features, low risk)

---

### Sprint 18: Community Curation (1 week)
**Focus**: Discovery and engagement features

| Feature | Effort | Owner |
|---------|--------|-------|
| Trending Tags | 3 days | senior-go-architect |
| Featured/Staff Picks | 2 days | senior-go-architect |
| E2E Tests | 1 day | test-strategist |

**Deliverables**:
- Trending tags endpoint with Redis caching
- Featured image flag and endpoints
- Admin feature/unfeature controls
- 10+ Newman E2E tests

**No Security Gate** (read-only features, low risk)

---

## Competitive Differentiation Strategy

### Where goimg Wins (Unique Value)

1. **IPFS Storage** (Sprint 13) - UNIQUE IN MARKET
   - Neither Flickr nor Chevereto offer decentralized storage
   - Differentiator for privacy-conscious users and Web3 community

2. **Modern DDD Architecture** - UNIQUE IN MARKET
   - Chevereto is PHP monolith (legacy)
   - goimg is Go microservices-ready, cloud-native

3. **Security-First** - COMPETITIVE ADVANTAGE
   - OWASP Top 10 compliance, ClamAV, 2FA, OAuth, rate limiting
   - Audit logging, penetration testing, security gates
   - Exceeds Chevereto security posture

4. **API-First Design** - COMPETITIVE ADVANTAGE
   - 100% OpenAPI compliance, comprehensive REST API
   - Mobile app ready from day 1
   - Webhook support (future)

### Where goimg Matches (Competitive Parity)

1. **Self-Hosted Option** - PARITY WITH CHEVERETO
   - Full control over data and platform rules
   - Docker-based deployment

2. **Multi-Storage Support** - PARITY WITH CHEVERETO
   - Local, S3, IPFS (Chevereto has S3, Google Cloud, Alibaba)

3. **Social Features** - PARITY WITH FLICKR
   - Likes, comments, follows, activity feeds (Sprints 6, 12)

4. **Moderation Tools** - PARITY WITH BOTH (after Sprint 14)
   - Abuse reporting, admin queue, bans
   - AI NSFW detection (Sprint 15)

### Where goimg Needs to Catch Up (Phase 2 Focus)

1. **Guest Uploads** - PARITY WITH CHEVERETO (Sprint 14)
   - Essential for event photographers, communities

2. **Advanced Search** - PARITY WITH BOTH (Sprint 15)
   - Date range, size, dimension filters

3. **Nested Albums** - PARITY WITH BOTH (Sprint 17)
   - Power user organization feature

### Where goimg Can Skip (Low Priority)

1. **Video Support** - OUT OF SCOPE
   - Different product category (hosting + transcoding costs)
   - Focus on photo excellence first

2. **Account Tiers** - PHASE 3
   - Revenue model, but not blocking user adoption
   - Launch with single free tier, add paid later

---

## Market Positioning

### Target Segments

1. **Privacy-Conscious Users** (IPFS differentiator)
   - Web3 enthusiasts, crypto community
   - Journalists, activists (censorship resistance)

2. **Self-Hosters** (Chevereto replacement)
   - Photographers running own servers
   - Communities with custom branding needs

3. **Event Photographers** (guest uploads)
   - Wedding, sports, conference photographers
   - Need guest uploads without registration friction

4. **Pro Photographers** (future paid tier)
   - Need advanced organization (nested albums)
   - Custom watermarking, client proofing (Phase 3)

### Messaging

**Tagline**: "Your photos, your rules, forever."

**Value Props**:
- Own your data (self-hosted)
- Unstoppable storage (IPFS)
- Built for privacy (encryption, security-first)
- Modern API (integrate anywhere)
- Open source (transparency, community)

---

## Risk Assessment

### High Risk Items

1. **AI NSFW API Costs** (Sprint 15)
   - Risk: API costs scale with uploads ($0.001-$0.005/image)
   - Mitigation: Cache results, batch processing, choose affordable provider
   - Budget: $10-$50/month for 10K images (acceptable)

2. **Content Moderation Load** (Sprint 14)
   - Risk: Viral growth = moderation burden exceeds admin capacity
   - Mitigation: AI NSFW detection (Sprint 15) reduces 90% of manual review
   - Escalation: Hire contract moderators if needed

3. **Guest Upload Abuse** (Sprint 14)
   - Risk: Spam, illegal content via guest uploads
   - Mitigation: Rate limiting (10/hour), ClamAV, auto-delete after 30 days
   - Fallback: Disable guest uploads if abuse detected

### Medium Risk Items

1. **Storage Costs** (IPFS + S3 dual storage)
   - Risk: Storing on both IPFS and S3 doubles storage costs
   - Mitigation: Make IPFS optional (config flag), use IPFS for original only

2. **Search Performance** (Sprint 15 advanced filters)
   - Risk: Complex filters slow down searches
   - Mitigation: Database indexes, pagination, Redis caching

### Low Risk Items

1. **Nested Albums Complexity** (Sprint 17)
   - Risk: Recursive queries cause performance issues
   - Mitigation: Limit nesting depth to 3 levels, use CTEs

2. **Feature Creep** (P2/P3 features)
   - Risk: Too many features delay core product
   - Mitigation: Strict prioritization, defer P3 to Phase 3

---

## Success Metrics (Phase 2)

### Sprint 14 KPIs
- Abuse reports created: > 0 (feature usage)
- Moderation actions: > 0 (admin engagement)
- Guest uploads: 10% of total uploads (adoption)
- Guest upload conversion rate: 5% claim to account (growth)

### Sprint 15 KPIs
- AI NSFW detection accuracy: > 90% (vs manual review)
- False positive rate: < 5% (quality)
- Advanced search usage: 20% of searches use filters (adoption)

### Sprint 16 KPIs
- oEmbed embeds: > 0 (virality)
- Social preview CTR: +10% vs plain links (quality signal)

### Sprint 17 KPIs
- Nested albums: 5% of users create sub-albums (power user adoption)
- Custom variants: 10% of admins configure custom sizes (flexibility)

### Sprint 18 KPIs
- Trending tags page views: 10% of explore traffic (discovery)
- Featured images: 1-5 new features per week (curation consistency)

---

## Conclusion

**Recommended Phase 2 Sprint Plan**:

- **Sprint 13**: IPFS Storage Integration (IN PROGRESS) ✅
- **Sprint 14**: Content Moderation + Guest Uploads (2 weeks) ⭐ P0 CRITICAL
- **Sprint 15**: AI NSFW Detection + Advanced Search (2 weeks) ⭐ P1 IMPORTANT
- **Sprint 16**: oEmbed + Social Preview Cards (1 week) ⭐ P1 IMPORTANT
- **Sprint 17**: Nested Albums + Custom Variants (2 weeks) ⭐ P2 NICE-TO-HAVE
- **Sprint 18**: Trending Tags + Featured Picks (1 week) ⭐ P2 NICE-TO-HAVE

**Total Duration**: 8 weeks (Sprint 14-18)

**Push to Phase 3**:
- Watermarking
- Account Tiers/Subscriptions
- Categories
- QR Codes
- Shortened URLs

**Rationale**: Focus Phase 2 on competitive parity (content moderation, guest uploads, AI NSFW, advanced search) and low-effort virality features (oEmbed, social cards). Defer revenue model (tiers) and niche features (watermarking, QR codes) to Phase 3 after achieving product-market fit.
