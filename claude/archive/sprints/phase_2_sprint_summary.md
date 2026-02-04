# Phase 2 Sprint Planning Summary
> Quick reference for remaining Phase 2 sprint assignments
> **Last Updated**: 2026-01-07

---

## Direct Answers to Planning Questions

### 1. Essential Features for Competitive Parity (Must Have)

| Feature | Rationale | Sprint |
|---------|-----------|--------|
| **Content Moderation Suite** | Legal requirement (DSA compliance), platform safety, both Flickr and Chevereto have this | 14 |
| **Guest Uploads** | Chevereto core feature, essential for event photographers and communities | 14 |
| **AI NSFW Detection** | Industry standard in 2026, reduces manual moderation burden by 90% | 15 |
| **Advanced Search Filters** | Baseline expectation for all photo platforms (date, size, dimensions) | 15 |

**Justification**: These four features are table stakes in 2026. Without them, goimg cannot compete with Chevereto (guest uploads) or Flickr (AI moderation, advanced search). Content moderation is legally required for EU compliance (DSA).

---

### 2. Features Providing Strong Differentiation

| Feature | Differentiation | Status |
|---------|----------------|--------|
| **IPFS Storage** | UNIQUE IN MARKET - neither Flickr nor Chevereto offer decentralized storage | Sprint 13 (IN PROGRESS) |
| **Modern DDD Architecture** | UNIQUE - Chevereto is legacy PHP monolith, goimg is cloud-native Go | COMPLETED |
| **Security-First Design** | ADVANTAGE - 2FA, OAuth, audit logging, OWASP compliance exceeds competitors | COMPLETED (Sprints 10-12) |
| **API-First, 100% OpenAPI** | ADVANTAGE - mobile-ready, webhook support (future), comprehensive REST API | COMPLETED |

**Recommendation**: Market goimg as "The privacy-first, unstoppable image platform with Web3 storage" to emphasize IPFS differentiation.

---

### 3. Recommended Sprint Order for Remaining Phase 2

```
Sprint 13: IPFS Storage Integration (IN PROGRESS) ✅
  - Duration: 2 weeks
  - Status: Phase 1 COMPLETE (infrastructure), Phase 2 PENDING (application/HTTP)

Sprint 14: Safety & Growth (CRITICAL) ⭐
  - Duration: 2 weeks
  - Features: Content Moderation Suite + Guest Uploads
  - Priority: P0 (blocking competitive launch)
  - Dependencies: None

Sprint 15: AI & Discovery (IMPORTANT) ⭐
  - Duration: 2 weeks
  - Features: AI NSFW Detection + Advanced Search Filters
  - Priority: P1 (industry standard)
  - Dependencies: None (can run in parallel with Sprint 14)

Sprint 16: Sharing & Virality (QUICK WINS) ⭐
  - Duration: 1 week
  - Features: oEmbed Support + Social Media Preview Cards
  - Priority: P1 (low effort, high ROI)
  - Dependencies: None

Sprint 17: Organization (NICE-TO-HAVE)
  - Duration: 2 weeks
  - Features: Nested Albums + Custom Variant Sizes
  - Priority: P2 (power user features)
  - Dependencies: None

Sprint 18: Community Curation (NICE-TO-HAVE)
  - Duration: 1 week
  - Features: Trending Tags + Featured/Staff Picks
  - Priority: P2 (engagement drivers)
  - Dependencies: Sprint 14 (RBAC for Featured Picks)

Total Duration: 8 weeks (Sprint 14-18)
```

---

### 4. Technical Dependencies Between Features

**Dependency Tree**:

```
Sprint 13 (IPFS):
  └─ No blocking dependencies for Sprints 14-18 (can run in parallel)

Sprint 14 (Content Moderation + Guest Uploads):
  ├─ Content Moderation unlocks:
  │   ├─ Guest upload review (same sprint)
  │   └─ Featured/Staff Picks admin controls (Sprint 18)
  └─ No external dependencies (uses existing RBAC from Sprint 6)

Sprint 15 (AI NSFW + Advanced Search):
  ├─ AI NSFW Detection: No dependencies
  └─ Advanced Search: Extends existing search (Sprint 6)

Sprint 16 (oEmbed + Social Cards):
  ├─ oEmbed: No dependencies
  └─ Social Cards: No dependencies

Sprint 17 (Nested Albums + Custom Variants):
  ├─ Nested Albums: No dependencies
  └─ Custom Variants: Extends image processor (Sprint 5)

Sprint 18 (Trending Tags + Featured Picks):
  ├─ Trending Tags: No dependencies
  └─ Featured Picks: Requires Sprint 14 RBAC for admin controls
```

**Critical Path**:
- Sprint 14 must complete before Sprint 18 (Featured/Staff Picks depends on moderation RBAC)
- All other sprints can run in parallel or sequence

**Shared Infrastructure** (already complete):
- RBAC middleware (Sprint 6) ✅
- Rate limiting (Sprint 4) ✅
- Audit logging (Sprint 9) ✅
- Background jobs - Asynq (Sprint 6) ✅

---

### 5. Features to Push to Phase 3

| Feature | Reason for Deferral | Future Sprint |
|---------|---------------------|---------------|
| **Watermarking** | Niche feature (only for pro photographers selling prints), medium effort for low ROI | Phase 3, Sprint 19+ |
| **Account Tiers/Subscriptions** | High complexity (billing, Stripe, tax compliance), requires product-market fit validation first | Phase 3, Sprint 20+ |
| **Categories** | Tags already provide categorization, categories require editorial curation overhead | Phase 3 or NEVER |
| **QR Codes for Images** | Niche feature (event photography), low effort but low impact | Phase 3, Sprint 21+ |
| **Shortened URLs** | Low priority, external tools exist (bit.ly), URLs already reasonably short | Phase 3 (low priority) |

**Rationale**: Focus Phase 2 on competitive parity and viral growth. Defer monetization (tiers) until user base grows. Defer niche features (watermarking, QR codes) until core product achieves product-market fit.

---

## Priority Tier Breakdown

### P0 - Critical (MUST Complete Phase 2)

**Sprint 14: Content Moderation + Guest Uploads**
- Abuse reporting system
- Admin moderation queue
- User ban system
- Audit logging for moderation actions
- Guest upload endpoint (no auth required)
- Guest upload cleanup job (auto-delete after 30 days)

**Why P0?**
- Legal requirement: EU Digital Services Act (DSA) compliance
- Platform safety: Prevent spam, abuse, illegal content
- Competitive parity: Both Flickr and Chevereto have robust moderation
- Growth driver: Guest uploads enable viral sharing without registration friction

**Estimated Effort**: 2 weeks (13 working days)

---

### P1 - Important (High Impact, Should Complete Phase 2)

**Sprint 15: AI NSFW Detection + Advanced Search Filters**
- AI moderation API integration (SightEngine/ModerateContent)
- Async NSFW scoring job
- Admin override for false positives
- Date range, size, dimension search filters

**Why P1?**
- Industry standard: All photo platforms in 2026 use AI + human moderation
- Reduces manual load: AI flags 90%+ of NSFW content automatically
- User retention: Power users need advanced filtering

**Estimated Effort**: 2 weeks (11 working days)

**Sprint 16: oEmbed + Social Media Preview Cards**
- oEmbed JSON endpoint
- Open Graph meta tags
- Twitter Card meta tags

**Why P1?**
- Low effort, high ROI: Simple endpoints, massive virality impact
- Quality signal: Rich previews improve social CTR by 10%+

**Estimated Effort**: 1 week (4 working days)

---

### P2 - Nice-to-Have (Medium Impact, Optional for Phase 2)

**Sprint 17: Nested Albums + Custom Variant Sizes**
- Parent album references (max 3 levels deep)
- Recursive album tree queries
- Admin-configurable variant sizes

**Why P2?**
- Power user features: Only 5-10% of users need nested albums
- Not blocking: Most users use single-level albums (80/20 rule)

**Estimated Effort**: 2 weeks (9 working days)

**Sprint 18: Trending Tags + Featured/Staff Picks**
- Trending tags with Redis caching
- Featured image flag and admin controls

**Why P2?**
- Engagement drivers: Discovery features, but not core workflow
- Low effort: Simple COUNT queries and boolean flags

**Estimated Effort**: 1 week (5 working days)

---

### P3 - Future Phase (Phase 3)

**Deferred Features**:
- Watermarking (niche, medium effort)
- Account Tiers/Subscriptions (high complexity, requires PMF first)
- Categories (tags already provide this)
- QR Codes (low impact)
- Shortened URLs (external tools exist)

**Rationale**: Defer monetization and niche features until core product achieves product-market fit.

---

## Resource Allocation by Sprint

### Sprint 14: Content Moderation + Guest Uploads (2 weeks)

| Task | Effort | Owner |
|------|--------|-------|
| Domain Layer (Report, Ban, GuestImage) | 2 days | senior-go-architect |
| Infrastructure (Repositories, RBAC) | 2 days | senior-go-architect |
| Application Layer (Commands, Queries) | 3 days | senior-go-architect |
| HTTP Layer (Endpoints, OpenAPI) | 2 days | senior-go-architect |
| Security Review (Audit logging, rate limiting) | 1 day | senior-secops-engineer |
| E2E Tests (20+ Newman tests) | 1 day | test-strategist |
| Documentation Updates | 1 day | senior-go-architect |

**Security Gate S14**: 6 controls (moderation audit, RBAC, guest upload rate limiting, ClamAV, auto-cleanup)

---

### Sprint 15: AI NSFW + Advanced Search (2 weeks)

| Task | Effort | Owner |
|------|--------|-------|
| AI API Integration (SightEngine/ModerateContent) | 3 days | senior-go-architect |
| Async Moderation Job (Asynq) | 1 day | senior-go-architect |
| NSFW Score Storage (migration, domain) | 1 day | senior-go-architect |
| Admin Override Controls | 1 day | senior-go-architect |
| Advanced Search Filters (date, size, dimensions) | 2 days | senior-go-architect |
| E2E Tests (15+ Newman tests) | 1 day | test-strategist |
| Documentation Updates | 1 day | senior-go-architect |

**Security Gate S15**: 4 controls (API key encryption, AI audit logging, manual override, SQL injection prevention)

---

### Sprint 16: oEmbed + Social Cards (1 week)

| Task | Effort | Owner |
|------|--------|-------|
| oEmbed Endpoint (JSON response) | 1 day | senior-go-architect |
| Open Graph Meta Tags | 0.5 days | senior-go-architect |
| Twitter Card Meta Tags | 0.5 days | senior-go-architect |
| E2E Tests (8+ Newman tests) | 0.5 days | test-strategist |
| Documentation Updates | 0.5 days | senior-go-architect |

**No Security Gate** (low-risk features, read-only endpoints)

---

### Sprint 17: Nested Albums + Custom Variants (2 weeks)

| Task | Effort | Owner |
|------|--------|-------|
| Database Migration (parent_album_id) | 1 day | senior-go-architect |
| Domain Layer (recursive methods) | 1 day | senior-go-architect |
| Application Layer (tree queries) | 2 days | senior-go-architect |
| HTTP Layer (breadcrumb navigation) | 1 day | senior-go-architect |
| Custom Variant Config (env vars) | 2 days | senior-go-architect |
| E2E Tests (12+ Newman tests) | 1 day | test-strategist |
| Documentation Updates | 1 day | senior-go-architect |

**No Security Gate** (internal features, low risk)

---

### Sprint 18: Trending Tags + Featured Picks (1 week)

| Task | Effort | Owner |
|------|--------|-------|
| Trending Tags (COUNT query + Redis cache) | 2 days | senior-go-architect |
| Featured Flag (migration, domain, endpoints) | 1 day | senior-go-architect |
| Admin Controls (feature/unfeature) | 0.5 days | senior-go-architect |
| E2E Tests (10+ Newman tests) | 0.5 days | test-strategist |
| Documentation Updates | 0.5 days | senior-go-architect |

**No Security Gate** (read-only features, low risk)

---

## Risk Mitigation Strategies

### High Risk: AI API Costs (Sprint 15)
- **Risk**: API costs scale with uploads ($0.001-$0.005/image)
- **Mitigation**:
  - Cache results in database (nsfw_score column)
  - Choose affordable provider (SightEngine ~$0.001/image)
  - Budget cap: $50/month for 10K-50K images
- **Fallback**: Disable AI moderation, fall back to manual review

### High Risk: Content Moderation Load (Sprint 14)
- **Risk**: Viral growth = manual moderation exceeds admin capacity
- **Mitigation**:
  - AI NSFW detection (Sprint 15) reduces 90% of manual review
  - Auto-ban repeat offenders (3 strikes)
  - Rate limit abuse reports (10/hour)
- **Escalation**: Hire contract moderators if needed

### Medium Risk: Guest Upload Abuse (Sprint 14)
- **Risk**: Spam, illegal content via guest uploads
- **Mitigation**:
  - Rate limiting (10 uploads/hour per IP)
  - ClamAV scanning (same as authenticated uploads)
  - Auto-delete after 30 days (storage cleanup)
  - Moderation queue review
- **Fallback**: Disable guest uploads if abuse rate > 10%

### Medium Risk: IPFS Storage Costs (Sprint 13)
- **Risk**: Dual storage (S3 + IPFS) doubles costs
- **Mitigation**:
  - Make IPFS optional (config flag: `ENABLE_IPFS=false`)
  - Store only originals on IPFS (variants on S3)
  - Free IPFS tier (Pinata: 1GB free, Infura: 5GB free)
- **Fallback**: S3-only mode for cost-conscious deployments

---

## Success Criteria (Phase 2 Completion)

### Sprint 14 Success
- ✅ Abuse reporting works (> 0 reports created)
- ✅ Moderation queue functional (> 0 moderation actions)
- ✅ User bans enforced (banned users cannot login)
- ✅ Guest uploads: 10% of total uploads
- ✅ Guest conversion rate: 5% claim to account
- ✅ Security Gate S14: 6/6 controls passed

### Sprint 15 Success
- ✅ AI NSFW detection accuracy: > 90% (vs manual review)
- ✅ False positive rate: < 5%
- ✅ Advanced search usage: 20% of searches use filters
- ✅ Security Gate S15: 4/4 controls passed

### Sprint 16 Success
- ✅ oEmbed embeds: > 0 (virality metric)
- ✅ Social preview CTR: +10% vs plain links
- ✅ All major platforms render previews (Twitter, Facebook, LinkedIn, Slack)

### Sprint 17 Success
- ✅ Nested albums: 5% of users create sub-albums
- ✅ Custom variants: 10% of admins configure custom sizes
- ✅ No performance degradation (recursive queries optimized)

### Sprint 18 Success
- ✅ Trending tags page views: 10% of explore traffic
- ✅ Featured images: 1-5 new features per week (curation consistency)
- ✅ Cache hit rate: > 90% (Redis caching effective)

---

## Phase 2 Completion Checklist

### Must Complete (P0-P1)
- [ ] Sprint 13: IPFS Storage Integration (Phase 2 - application/HTTP layer)
- [ ] Sprint 14: Content Moderation Suite
- [ ] Sprint 14: Guest Uploads
- [ ] Sprint 15: AI NSFW Detection
- [ ] Sprint 15: Advanced Search Filters
- [ ] Sprint 16: oEmbed Support
- [ ] Sprint 16: Social Media Preview Cards

### Optional (P2)
- [ ] Sprint 17: Nested Albums
- [ ] Sprint 17: Custom Variant Sizes
- [ ] Sprint 18: Trending Tags
- [ ] Sprint 18: Featured/Staff Picks

### Deferred (P3)
- Watermarking → Phase 3
- Account Tiers/Subscriptions → Phase 3
- Categories → Phase 3
- QR Codes → Phase 3
- Shortened URLs → Phase 3

---

## Competitive Scorecard (After Phase 2)

| Feature Category | Flickr | Chevereto | goimg (Phase 2) | Winner |
|------------------|--------|-----------|-----------------|--------|
| **Core Image Hosting** | ✅ Excellent | ✅ Excellent | ✅ Excellent | TIE |
| **Social Features** | ✅ Excellent | ✅ Good | ✅ Excellent | TIE |
| **Content Moderation** | ✅ AI + Human | ✅ Basic | ✅ AI + Human (after S15) | TIE |
| **Guest Uploads** | ❌ None | ✅ Yes | ✅ Yes (after S14) | Chevereto/goimg |
| **Self-Hosted** | ❌ No | ✅ Yes | ✅ Yes | Chevereto/goimg |
| **Decentralized Storage (IPFS)** | ❌ No | ❌ No | ✅ YES | **goimg** 🏆 |
| **Modern API** | ✅ Good | ✅ Basic | ✅ Excellent (100% OpenAPI) | **goimg** 🏆 |
| **Security & Privacy** | ✅ Good | ⚠️ Basic | ✅ Excellent (2FA, OAuth, audit) | **goimg** 🏆 |
| **Advanced Search** | ✅ Excellent | ✅ Good | ✅ Excellent (after S15) | TIE |

**Verdict**: After Phase 2, goimg achieves competitive parity with Flickr and Chevereto while offering unique differentiation (IPFS, modern API, superior security).

---

## Next Steps

1. **Complete Sprint 13 Phase 2** (IPFS application/HTTP layer)
2. **Kick off Sprint 14** (Content Moderation + Guest Uploads) - CRITICAL
3. **Security Gate S14 Review** (6 controls)
4. **Proceed to Sprint 15** (AI NSFW + Advanced Search)
5. **Evaluate P2 features** (Sprints 17-18) based on user feedback after Sprint 16

**Recommendation**: Focus on completing P0-P1 features (Sprints 14-16) before considering P2 features (Sprints 17-18). User feedback from Sprints 14-16 will inform whether nested albums and trending tags are truly needed.
