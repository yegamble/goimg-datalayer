# Phase 3 Sprint Plan

> **Status**: In Progress | **Version**: 1.1 | **Updated**: 2026-01-09
>
> Phase 3 focuses on advanced features, scalability improvements, and ecosystem expansion
> following the successful completion of Phase 2 (Sprints 10-15).
>
> **Current Sprint**: Sprint 16 - oEmbed + Social Media Cards 🚀

---

## Executive Summary

Phase 2 delivered all planned features including:
- Two-Factor Authentication (Sprint 11)
- OAuth Integration with Google/GitHub (Sprint 12)
- IPFS Decentralized Storage (Sprint 13)
- Content Moderation & Guest Uploads (Sprint 14)
- AI NSFW Detection & Advanced Search (Sprint 15)

Phase 3 continues the roadmap with features that enhance discoverability, social sharing,
and platform scalability.

---

## Phase 3 Sprint Overview

| Sprint | Focus | Duration | Priority | Status |
|--------|-------|----------|----------|--------|
| 16 | oEmbed + Social Media Cards | 1 week | P1 | 🚧 **IN PROGRESS** (oEmbed endpoint complete) |
| 17 | Nested Albums + Custom Variants | 2 weeks | P2 | Planned |
| 18 | Trending Tags + Featured Picks | 1 week | P2 | Planned |
| 19 | Groups/Communities | 2 weeks | P3 | Backlog |
| 20 | Video Support | 3 weeks | P3 | Backlog |
| 21 | Account Tiers/Subscriptions | 2 weeks | P3 | Backlog |

**Total Phase 3 Duration**: ~11 weeks (Sprints 16-21)

---

## Sprint 16: oEmbed + Social Media Cards

**Duration**: 1 week
**Priority**: P1 - HIGH
**Dependencies**: None
**Status**: 🚧 **IN PROGRESS**

### Objectives

Enable images to be embedded on external sites and generate rich social media previews.

### Progress

| Component | Status | Notes |
|-----------|--------|-------|
| oEmbed Endpoint | ✅ COMPLETE | `internal/interfaces/http/handlers/oembed_handler.go` |
| OpenAPI Spec | ✅ COMPLETE | oEmbed endpoint documented |
| Router Wiring | ✅ COMPLETE | Mounted at `/api/v1/oembed` |
| Open Graph Tags | ✅ COMPLETE | Meta tags in preview_handler.go |
| Twitter Cards | ✅ COMPLETE | Twitter card meta tags in preview_handler.go |
| Image Preview Page | ✅ COMPLETE | `GET /images/{id}/preview` |
| E2E Tests | ✅ COMPLETE | 8 Newman tests for oEmbed and preview |

### Deliverables

| Component | Description |
|-----------|-------------|
| oEmbed Endpoint | `GET /api/v1/oembed` - Standard oEmbed 1.0 response |
| Open Graph Tags | Meta tags for Facebook, LinkedIn sharing |
| Twitter Cards | Twitter card meta tags for image previews |
| WhatsApp Preview | Optimized preview for messaging apps |
| Image Page Route | Public image viewing page with meta tags |

### API Endpoints

```yaml
GET /api/v1/oembed?url={image_url}&format={json|xml}  # ✅ IMPLEMENTED
  - Returns oEmbed response for embedding
  - Supports maxwidth, maxheight parameters
  - JSON and XML response formats

GET /images/{id}/preview  # ✅ IMPLEMENTED
  - Public image page with social meta tags
  - SEO-friendly URL structure
```

### Technical Notes

- oEmbed responses cached in Redis (1 hour TTL)
- Meta tag generation at request time or pre-computed
- Support for private images (return generic placeholder)

---

## Sprint 17: Nested Albums + Custom Variants

**Duration**: 2 weeks
**Priority**: P2 - MEDIUM
**Dependencies**: Sprint 16

### Objectives

Support hierarchical album organization and user-configurable image variant sizes.

### Deliverables

| Component | Description |
|-----------|-------------|
| Album Hierarchy | Parent-child album relationships |
| Album Breadcrumbs | Navigation path for nested albums |
| Custom Variants | User-defined variant sizes (pro feature) |
| Variant Presets | Named presets (banner, avatar, etc.) |
| Database Migration | Album parent_id, variant_configs table |

### API Endpoints

```yaml
PATCH /api/v1/albums/{id}
  - Add parent_id field for nesting

GET /api/v1/albums/{id}/breadcrumb
  - Returns array of ancestor albums

POST /api/v1/images/{id}/variants
  - Generate custom variant with specified dimensions

GET /api/v1/variant-presets
  - List available variant presets
```

### Database Changes

```sql
-- Add parent_id to albums
ALTER TABLE albums ADD COLUMN parent_id UUID REFERENCES albums(id);
CREATE INDEX idx_albums_parent_id ON albums(parent_id);

-- Variant configurations table
CREATE TABLE variant_configs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    name VARCHAR(50) NOT NULL,
    max_width INTEGER,
    max_height INTEGER,
    format VARCHAR(10) DEFAULT 'jpeg',
    quality INTEGER DEFAULT 85,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## Sprint 18: Trending Tags + Featured Picks

**Duration**: 1 week
**Priority**: P2 - MEDIUM
**Dependencies**: None

### Objectives

Implement tag discovery features and curator-selected featured content.

### Deliverables

| Component | Description |
|-----------|-------------|
| Popular Tags API | `GET /api/v1/tags/popular` - Most used tags |
| Trending Tags | Time-weighted tag popularity |
| Tag Search | `GET /api/v1/tags/search?q=` |
| Featured Picks | Admin-curated featured images |
| Explore Enhancement | Add trending/featured sections |

### API Endpoints

```yaml
GET /api/v1/tags/popular
  - Returns top N tags by usage count
  - Query params: limit, period (day, week, month, all)

GET /api/v1/tags/trending
  - Returns tags with recent usage growth
  - Time-weighted popularity algorithm

GET /api/v1/tags/search?q={query}
  - Search tags by prefix/partial match

GET /api/v1/explore/featured
  - Admin-curated featured images
  - Rotates daily/weekly

POST /api/v1/moderation/featured (admin)
  - Add/remove images from featured list
```

### Technical Notes

- Tag popularity cached in Redis (recalculated hourly)
- Trending algorithm: (recent_count / total_count) * time_decay
- Featured picks stored in dedicated table with scheduling

---

## Sprint 19: Groups/Communities (Backlog)

**Duration**: 2 weeks
**Priority**: P3 - LOW
**Dependencies**: Sprint 17

### Objectives

Enable users to create and join interest-based groups with shared albums.

### Deliverables

| Component | Description |
|-----------|-------------|
| Group Entity | Name, description, privacy, member count |
| Group Membership | Join/leave, roles (owner, admin, member) |
| Group Albums | Shared albums within groups |
| Group Activity | Activity feed scoped to group |
| Group Discovery | Search and browse public groups |

---

## Sprint 20: Video Support (Backlog)

**Duration**: 3 weeks
**Priority**: P3 - LOW
**Dependencies**: Sprint 16

### Objectives

Extend the platform to support video uploads with streaming and thumbnails.

### Deliverables

| Component | Description |
|-----------|-------------|
| Video Upload | MP4, WebM, MOV support |
| Video Processing | FFmpeg transcoding to HLS/DASH |
| Video Thumbnails | Auto-generated thumbnail variants |
| Video Player | Embedded player endpoint |
| Storage Strategy | Video-specific storage tiers |

### Technical Notes

- Video processing via separate worker pool
- HLS segments stored in S3 with CDN
- Thumbnail extraction at upload time
- Separate rate limits for video uploads

---

## Sprint 21: Account Tiers/Subscriptions (Backlog)

**Duration**: 2 weeks
**Priority**: P3 - LOW
**Dependencies**: Sprint 17

### Objectives

Implement tiered account levels with different storage limits and features.

### Deliverables

| Component | Description |
|-----------|-------------|
| Account Tiers | Free, Pro, Business levels |
| Storage Quotas | Tier-based storage limits |
| Feature Gates | Custom variants, priority processing |
| Usage Tracking | Storage and bandwidth metering |
| Upgrade Flow | In-app upgrade prompts |

---

## Phase 3 Security Gates

Each sprint requires security review before merge:

| Sprint | Gate ID | Key Controls |
|--------|---------|--------------|
| 16 | S16-EMBED | oEmbed URL validation, CSRF protection on previews |
| 17 | S17-ALBUM | Authorization on nested album access |
| 18 | S18-TAG | Tag injection prevention, featured image validation |
| 19 | S19-GROUP | Group privacy enforcement, role-based access |
| 20 | S20-VIDEO | Video file validation, transcoding security |
| 21 | S21-TIER | Payment security, quota enforcement |

---

## Technical Debt Items

The following items should be addressed during Phase 3:

| Item | Priority | Sprint |
|------|----------|--------|
| Extract inline OpenAPI schemas | Medium | 16 |
| Remove unused OpenAPI schema definitions | Low | 16 |
| Add RBAC scopes to OpenAPI security | Medium | 17 |
| Document rate limits in OpenAPI spec | Medium | 16 |
| Improve handler test coverage | Medium | Any |
| Add middleware unit tests | Low | Any |

---

## Agent Assignments

| Sprint | Lead Agent | Critical Agents |
|--------|------------|-----------------|
| 16 | senior-go-architect | image-gallery-expert, senior-secops-engineer |
| 17 | senior-go-architect | backend-test-architect |
| 18 | image-gallery-expert | senior-go-architect |
| 19 | senior-go-architect | senior-secops-engineer |
| 20 | senior-go-architect | image-gallery-expert, cicd-guardian |
| 21 | senior-secops-engineer | senior-go-architect |

---

## Success Metrics

### Technical

- API response time: P95 < 200ms (unchanged)
- Video processing: < 5 minutes for 100MB video
- Availability: 99.9% uptime (unchanged)

### Feature Adoption

- oEmbed usage: 1000+ embeds/month within 3 months
- Group creation: 100+ active groups within 6 months
- Video uploads: 10% of total uploads within 6 months

---

## References

- [Sprint Plan (Sprints 1-15)](sprint_plan.md)
- [Phase 2 Feature Prioritization](phase_2_feature_prioritization.md)
- [MVP Features Specification](mvp_features.md)
- [Security Gates Documentation](security_gates.md)
