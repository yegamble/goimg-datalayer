# Sprint 20: Groups/Communities

> **Status**: IN PROGRESS 🚧
> **Duration**: 2 weeks
> **Priority**: P3 - LOW
> **Dependencies**: Sprint 17 (Nested Albums)
> **Last Updated**: 2026-01-12

---

## Executive Summary

Sprint 20 implements Groups/Communities functionality inspired by Flickr groups and addressing Chevereto's community requests for shared/group albums. This feature enables users to create interest-based communities with shared image pools, discussions, and collaborative album management.

### Research Insights

**Flickr Groups** (as of 2025-2026):
- Groups have **photo pools** where members share images
- **Discussion boards** with sticky topics for announcements
- **Pin groups** to get notifications in activity feed
- **Moderation roles**: Group admin can set rules and moderate content
- Official Flickr groups like "Flickr Social" curated by Flickr team
- Monthly challenges and photographer showcases

**Chevereto Current State**:
- No native group/shared album support (most requested feature)
- Community workarounds: Master accounts with guest uploads + moderation
- User requests emphasize: "multiple users with access (upload/delete) to shared albums"
- Use cases: Teams, families, commercial collaborations (e.g., soccer team parents)

**Industry Best Practices**:
- **RBAC (Role-Based Access Control)** for granular permissions
- Permission levels: Owner, Admin/Moderator, Member, Viewer
- Moderation tools: Content approval, member removal, ban lists
- Privacy options: Public, Private, Invite-Only, Password-Protected
- Regular permission audits and clear access guidelines

### Goals

1. Enable users to create and join interest-based groups
2. Implement shared image pools with group-level permissions
3. Support group albums with collaborative management
4. Provide moderation tools for group owners/admins
5. Build group discovery and browsing features

---

## Feature Comparison: Flickr vs goimg Groups

| Feature | Flickr Implementation | goimg Sprint 20 (MVP) | Priority |
|---------|----------------------|----------------------|----------|
| **Group Creation** | Any user can create groups | ✅ Any user can create groups | P0 |
| **Group Types** | Public, Private, Invite-Only | ✅ Public, Private, Invite-Only | P0 |
| **Photo Pool** | Members add photos to shared pool | ✅ Members share images to group | P0 |
| **Group Albums** | Not supported | ✅ Shared albums within groups | P0 |
| **Member Roles** | Owner, Admin, Member | ✅ Owner, Admin, Member | P0 |
| **Join Workflow** | Public (auto-join), Invite-Only (request/invite) | ✅ Same | P0 |
| **Discovery** | Browse/search public groups | ✅ Search groups by name/description | P0 |
| **Discussions** | Full discussion boards with threads | ⏳ **Phase 2** (Deferred) | P2 |
| **Pin Groups** | Pin to activity feed | ⏳ **Phase 2** (Deferred) | P2 |
| **Group Rules** | Custom rules per group | ✅ Description field (manual) | P1 |
| **Moderation Queue** | Admin approves images before visible | ✅ Group-level moderation setting | P1 |
| **Member Limits** | Configurable per group | ⏳ **Phase 2** (Fixed at 1000) | P3 |
| **Activity Feed** | Group-specific activity | ✅ Group activity timeline | P1 |
| **Notifications** | New posts, discussions, admin actions | ⏳ **Phase 2** (Basic only) | P2 |

### Differentiation: Group Albums

**Key Innovation**: Unlike Flickr (which lacks group albums), goimg groups will support **shared albums** where:
- Multiple members can add/remove images
- Album ownership belongs to the group, not an individual
- Admins can assign member roles per album (Contributor, Viewer)
- Addresses Chevereto community's #1 requested feature

---

## Domain Model (DDD)

### Bounded Context: `community`

New bounded context for groups/communities, separate from `gallery` and `identity`.

```
internal/domain/community/
├── group.go                    # Group aggregate root
├── group_id.go                 # Group ID value object
├── group_type.go               # GroupType enum (Public, Private, InviteOnly)
├── group_membership.go         # Membership entity (owned by Group)
├── group_role.go               # GroupRole enum (Owner, Admin, Member)
├── group_album.go              # GroupAlbum entity (group-owned album)
├── group_invitation.go         # Invitation entity
├── group_settings.go           # GroupSettings value object
├── repository.go               # Repository interfaces
├── events.go                   # Domain events
├── errors.go                   # Domain errors
└── doc.go                      # Package documentation
```

### Entities & Value Objects

#### 1. Group (Aggregate Root)

```go
type Group struct {
    id              GroupID
    name            GroupName       // Value object (3-100 chars, unique)
    slug            GroupSlug       // URL-safe slug
    description     string          // 0-1000 chars
    groupType       GroupType       // Public, Private, InviteOnly
    ownerID         identity.UserID
    settings        GroupSettings   // Value object
    memberCount     int
    imageCount      int
    albumCount      int
    coverImageID    *gallery.ImageID
    createdAt       time.Time
    updatedAt       time.Time
    events          []shared.DomainEvent
}

// Factory
func NewGroup(ownerID identity.UserID, name GroupName, groupType GroupType) (*Group, error)

// Business methods
func (g *Group) UpdateDescription(desc string) error
func (g *Group) UpdateSettings(settings GroupSettings) error
func (g *Group) UpdateCoverImage(imageID gallery.ImageID) error
func (g *Group) IncrementMemberCount()
func (g *Group) DecrementMemberCount()
```

#### 2. GroupMembership (Entity, Owned by Group)

```go
type GroupMembership struct {
    id          MembershipID
    groupID     GroupID
    userID      identity.UserID
    role        GroupRole      // Owner, Admin, Member
    status      MemberStatus   // Active, Invited, Requested, Banned
    invitedBy   *identity.UserID
    joinedAt    time.Time
    updatedAt   time.Time
}

// Factory
func NewGroupMembership(groupID GroupID, userID identity.UserID, role GroupRole) *GroupMembership

// Business methods
func (m *GroupMembership) PromoteToAdmin() error
func (m *GroupMembership) DemoteToMember() error
func (m *GroupMembership) Ban() error
func (m *GroupMembership) Activate() error
```

#### 3. GroupAlbum (Entity)

Group-owned album, distinct from user-owned `gallery.Album`.

```go
type GroupAlbum struct {
    id              GroupAlbumID
    groupID         GroupID
    createdByUserID identity.UserID
    title           string
    description     string
    coverImageID    *gallery.ImageID
    imageCount      int
    visibility      AlbumVisibility // Inherited from group or custom
    createdAt       time.Time
    updatedAt       time.Time
}

// Factory
func NewGroupAlbum(groupID GroupID, createdBy identity.UserID, title string) (*GroupAlbum, error)
```

#### 4. GroupInvitation (Entity)

```go
type GroupInvitation struct {
    id          InvitationID
    groupID     GroupID
    inviterID   identity.UserID
    inviteeEmail Email           // Can invite non-users
    inviteeUserID *identity.UserID // If registered user
    status      InvitationStatus // Pending, Accepted, Declined, Expired
    token       InvitationToken  // For email acceptance
    expiresAt   time.Time        // 7 days
    createdAt   time.Time
}
```

#### 5. Value Objects

```go
// GroupName - 3-100 chars, alphanumeric + spaces
type GroupName struct {
    value string
}

// GroupSlug - URL-safe, unique, generated from name
type GroupSlug struct {
    value string
}

// GroupSettings - Configuration
type GroupSettings struct {
    requireApproval    bool   // Moderation queue for new images
    allowMemberInvites bool   // Members can invite others
    allowMemberAlbums  bool   // Members can create group albums
    maxMembers         int    // 0 = unlimited
}

// GroupType enum
type GroupType int
const (
    GroupTypePublic GroupType = iota    // Anyone can join
    GroupTypePrivate                    // Invite-only, not discoverable
    GroupTypeInviteOnly                 // Discoverable but invite-only
)

// GroupRole enum
type GroupRole int
const (
    GroupRoleMember GroupRole = iota
    GroupRoleAdmin
    GroupRoleOwner
)

// MemberStatus enum
type MemberStatus int
const (
    MemberStatusInvited MemberStatus = iota
    MemberStatusRequested  // User requested to join
    MemberStatusActive
    MemberStatusBanned
)
```

### Repository Interfaces

```go
// GroupRepository - Aggregate root repository
type GroupRepository interface {
    NextID() GroupID
    FindByID(ctx context.Context, id GroupID) (*Group, error)
    FindBySlug(ctx context.Context, slug GroupSlug) (*Group, error)
    FindPublicGroups(ctx context.Context, filter GroupFilter, page Pagination) ([]*Group, int, error)
    SearchGroups(ctx context.Context, query string, page Pagination) ([]*Group, int, error)
    Save(ctx context.Context, group *Group) error
    Delete(ctx context.Context, id GroupID) error
}

// GroupMembershipRepository
type GroupMembershipRepository interface {
    NextID() MembershipID
    FindByID(ctx context.Context, id MembershipID) (*GroupMembership, error)
    FindByGroupAndUser(ctx context.Context, groupID GroupID, userID identity.UserID) (*GroupMembership, error)
    FindByGroup(ctx context.Context, groupID GroupID, filter MemberFilter, page Pagination) ([]*GroupMembership, int, error)
    FindByUser(ctx context.Context, userID identity.UserID, page Pagination) ([]*GroupMembership, int, error)
    Save(ctx context.Context, membership *GroupMembership) error
    Delete(ctx context.Context, id MembershipID) error
}

// GroupAlbumRepository
type GroupAlbumRepository interface {
    NextID() GroupAlbumID
    FindByID(ctx context.Context, id GroupAlbumID) (*GroupAlbum, error)
    FindByGroup(ctx context.Context, groupID GroupID, page Pagination) ([]*GroupAlbum, int, error)
    Save(ctx context.Context, album *GroupAlbum) error
    Delete(ctx context.Context, id GroupAlbumID) error
}

// GroupInvitationRepository
type GroupInvitationRepository interface {
    NextID() InvitationID
    FindByID(ctx context.Context, id InvitationID) (*GroupInvitation, error)
    FindByToken(ctx context.Context, token InvitationToken) (*GroupInvitation, error)
    FindPendingByGroup(ctx context.Context, groupID GroupID) ([]*GroupInvitation, error)
    FindExpired(ctx context.Context) ([]*GroupInvitation, error)
    Save(ctx context.Context, invitation *GroupInvitation) error
    Delete(ctx context.Context, id InvitationID) error
}
```

### Domain Events

```go
// Group lifecycle events
type GroupCreated struct {
    GroupID     GroupID
    OwnerID     identity.UserID
    Name        GroupName
    GroupType   GroupType
    CreatedAt   time.Time
}

type GroupDeleted struct {
    GroupID     GroupID
    DeletedBy   identity.UserID
    DeletedAt   time.Time
}

// Membership events
type UserJoinedGroup struct {
    GroupID     GroupID
    UserID      identity.UserID
    Role        GroupRole
    JoinedAt    time.Time
}

type UserLeftGroup struct {
    GroupID     GroupID
    UserID      identity.UserID
    LeftAt      time.Time
}

type MemberPromoted struct {
    GroupID     GroupID
    UserID      identity.UserID
    NewRole     GroupRole
    PromotedBy  identity.UserID
    PromotedAt  time.Time
}

type MemberBanned struct {
    GroupID     GroupID
    UserID      identity.UserID
    BannedBy    identity.UserID
    Reason      string
    BannedAt    time.Time
}

// Group activity events
type ImageSharedToGroup struct {
    GroupID     GroupID
    ImageID     gallery.ImageID
    UserID      identity.UserID
    SharedAt    time.Time
}

type GroupAlbumCreated struct {
    GroupAlbumID GroupAlbumID
    GroupID      GroupID
    CreatedBy    identity.UserID
    Title        string
    CreatedAt    time.Time
}
```

### Domain Errors

```go
var (
    // Group errors
    ErrGroupNotFound          = errors.New("group not found")
    ErrGroupNameRequired      = errors.New("group name is required")
    ErrGroupNameTooShort      = errors.New("group name must be at least 3 characters")
    ErrGroupNameTooLong       = errors.New("group name exceeds 100 characters")
    ErrGroupSlugTaken         = errors.New("group slug is already taken")
    ErrGroupDeleted           = errors.New("group has been deleted")

    // Membership errors
    ErrNotGroupMember         = errors.New("user is not a member of this group")
    ErrAlreadyGroupMember     = errors.New("user is already a member")
    ErrInsufficientGroupRole  = errors.New("insufficient permissions for this action")
    ErrCannotLeaveAsOwner     = errors.New("group owner cannot leave (transfer ownership first)")
    ErrMemberBanned           = errors.New("user is banned from this group")
    ErrMemberLimitReached     = errors.New("group has reached maximum member capacity")

    // Invitation errors
    ErrInvitationExpired      = errors.New("invitation has expired")
    ErrInvitationAlreadyUsed  = errors.New("invitation has already been used")

    // Album errors
    ErrGroupAlbumNotFound     = errors.New("group album not found")
    ErrGroupAlbumTitleRequired = errors.New("group album title is required")
)
```

---

## Database Schema

### Migration: `00017_create_groups.sql`

```sql
-- =====================================================
-- Groups table
-- =====================================================
CREATE TABLE groups (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    slug            VARCHAR(110) NOT NULL UNIQUE, -- URL-safe, indexed
    description     TEXT DEFAULT '',
    group_type      VARCHAR(20) NOT NULL DEFAULT 'public', -- public, private, invite_only
    owner_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Settings (denormalized for query performance)
    require_approval    BOOLEAN NOT NULL DEFAULT false,
    allow_member_invites BOOLEAN NOT NULL DEFAULT true,
    allow_member_albums BOOLEAN NOT NULL DEFAULT true,
    max_members         INTEGER NOT NULL DEFAULT 0, -- 0 = unlimited

    -- Counts (updated by triggers or application)
    member_count    INTEGER NOT NULL DEFAULT 1, -- Owner is first member
    image_count     INTEGER NOT NULL DEFAULT 0,
    album_count     INTEGER NOT NULL DEFAULT 0,

    cover_image_id  UUID REFERENCES images(id) ON DELETE SET NULL,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ -- Soft delete
);

-- Indexes
CREATE INDEX idx_groups_owner_id ON groups(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_groups_type ON groups(group_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_groups_slug ON groups(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_groups_name_trgm ON groups USING gin(name gin_trgm_ops); -- Full-text search
CREATE INDEX idx_groups_created_at ON groups(created_at DESC) WHERE deleted_at IS NULL;

-- =====================================================
-- Group memberships table
-- =====================================================
CREATE TABLE group_memberships (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id        UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL DEFAULT 'member', -- owner, admin, member
    status          VARCHAR(20) NOT NULL DEFAULT 'active', -- invited, requested, active, banned
    invited_by      UUID REFERENCES users(id) ON DELETE SET NULL,

    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_group_user UNIQUE (group_id, user_id)
);

-- Indexes
CREATE INDEX idx_group_memberships_group_id ON group_memberships(group_id);
CREATE INDEX idx_group_memberships_user_id ON group_memberships(user_id);
CREATE INDEX idx_group_memberships_status ON group_memberships(status);
CREATE INDEX idx_group_memberships_role ON group_memberships(role) WHERE status = 'active';

-- =====================================================
-- Group albums table (shared albums within groups)
-- =====================================================
CREATE TABLE group_albums (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id        UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,

    title           VARCHAR(255) NOT NULL,
    description     TEXT DEFAULT '',
    cover_image_id  UUID REFERENCES images(id) ON DELETE SET NULL,
    image_count     INTEGER NOT NULL DEFAULT 0,

    -- Visibility can be inherited from group or custom
    visibility      VARCHAR(20) NOT NULL DEFAULT 'group', -- group, public, private

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ -- Soft delete
);

-- Indexes
CREATE INDEX idx_group_albums_group_id ON group_albums(group_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_group_albums_creator ON group_albums(created_by_user_id);
CREATE INDEX idx_group_albums_created_at ON group_albums(created_at DESC);

-- =====================================================
-- Group album images (many-to-many)
-- =====================================================
CREATE TABLE group_album_images (
    group_album_id  UUID NOT NULL REFERENCES group_albums(id) ON DELETE CASCADE,
    image_id        UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    added_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    display_order   INTEGER NOT NULL DEFAULT 0,
    added_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (group_album_id, image_id)
);

-- Indexes
CREATE INDEX idx_group_album_images_album_order ON group_album_images(group_album_id, display_order);
CREATE INDEX idx_group_album_images_image_id ON group_album_images(image_id);

-- =====================================================
-- Group images (shared image pool)
-- =====================================================
CREATE TABLE group_images (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id        UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    image_id        UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    shared_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,

    status          VARCHAR(20) NOT NULL DEFAULT 'approved', -- pending, approved, rejected
    reviewed_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at     TIMESTAMPTZ,

    shared_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_group_image UNIQUE (group_id, image_id)
);

-- Indexes
CREATE INDEX idx_group_images_group_id ON group_images(group_id, shared_at DESC);
CREATE INDEX idx_group_images_image_id ON group_images(image_id);
CREATE INDEX idx_group_images_status ON group_images(status) WHERE status = 'pending';
CREATE INDEX idx_group_images_shared_by ON group_images(shared_by_user_id);

-- =====================================================
-- Group invitations
-- =====================================================
CREATE TABLE group_invitations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id        UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    inviter_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invitee_email   VARCHAR(255) NOT NULL,
    invitee_user_id UUID REFERENCES users(id) ON DELETE CASCADE, -- If registered user

    token           VARCHAR(64) NOT NULL UNIQUE, -- For email acceptance
    status          VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, accepted, declined, expired

    expires_at      TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '7 days'),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_group_invitations_group_id ON group_invitations(group_id);
CREATE INDEX idx_group_invitations_token ON group_invitations(token) WHERE status = 'pending';
CREATE INDEX idx_group_invitations_expires ON group_invitations(expires_at) WHERE status = 'pending';
CREATE INDEX idx_group_invitations_email ON group_invitations(invitee_email);

-- =====================================================
-- Group activity (for group-specific activity feed)
-- =====================================================
CREATE TABLE group_activities (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id        UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    actor_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    activity_type   VARCHAR(50) NOT NULL, -- member_joined, image_shared, album_created, etc.
    target_type     VARCHAR(50), -- image, album, user, group
    target_id       UUID,

    metadata        JSONB,

    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_group_activities_group_id ON group_activities(group_id, occurred_at DESC);
CREATE INDEX idx_group_activities_actor_id ON group_activities(actor_id);
CREATE INDEX idx_group_activities_type ON group_activities(activity_type);

-- =====================================================
-- Triggers for updating counts
-- =====================================================

-- Update groups.member_count when membership changes
CREATE OR REPLACE FUNCTION update_group_member_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' AND NEW.status = 'active' THEN
        UPDATE groups SET member_count = member_count + 1 WHERE id = NEW.group_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.status != 'active' AND NEW.status = 'active' THEN
        UPDATE groups SET member_count = member_count + 1 WHERE id = NEW.group_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.status = 'active' AND NEW.status != 'active' THEN
        UPDATE groups SET member_count = member_count - 1 WHERE id = NEW.group_id;
    ELSIF TG_OP = 'DELETE' AND OLD.status = 'active' THEN
        UPDATE groups SET member_count = member_count - 1 WHERE id = OLD.group_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_group_member_count
AFTER INSERT OR UPDATE OR DELETE ON group_memberships
FOR EACH ROW EXECUTE FUNCTION update_group_member_count();

-- Update groups.image_count when images shared to group
CREATE OR REPLACE FUNCTION update_group_image_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' AND NEW.status = 'approved' THEN
        UPDATE groups SET image_count = image_count + 1 WHERE id = NEW.group_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.status != 'approved' AND NEW.status = 'approved' THEN
        UPDATE groups SET image_count = image_count + 1 WHERE id = NEW.group_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.status = 'approved' AND NEW.status != 'approved' THEN
        UPDATE groups SET image_count = image_count - 1 WHERE id = NEW.group_id;
    ELSIF TG_OP = 'DELETE' AND OLD.status = 'approved' THEN
        UPDATE groups SET image_count = image_count - 1 WHERE id = OLD.group_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_group_image_count
AFTER INSERT OR UPDATE OR DELETE ON group_images
FOR EACH ROW EXECUTE FUNCTION update_group_image_count();

-- Update group_albums.image_count when images added to album
CREATE OR REPLACE FUNCTION update_group_album_image_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE group_albums SET image_count = image_count + 1 WHERE id = NEW.group_album_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE group_albums SET image_count = image_count - 1 WHERE id = OLD.group_album_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_group_album_image_count
AFTER INSERT OR DELETE ON group_album_images
FOR EACH ROW EXECUTE FUNCTION update_group_album_image_count();
```

---

## API Endpoints

### Group Management

#### 1. Create Group
```yaml
POST /api/v1/groups
Authorization: Bearer {token}

Request:
{
  "name": "Landscape Photography",
  "description": "Share your best landscape shots",
  "group_type": "public",
  "settings": {
    "require_approval": false,
    "allow_member_invites": true,
    "allow_member_albums": true,
    "max_members": 0
  }
}

Response: 201 Created
{
  "id": "uuid",
  "name": "Landscape Photography",
  "slug": "landscape-photography",
  "description": "Share your best landscape shots",
  "group_type": "public",
  "owner": { "id": "uuid", "username": "john" },
  "member_count": 1,
  "image_count": 0,
  "album_count": 0,
  "settings": { ... },
  "created_at": "2026-01-12T10:00:00Z"
}
```

#### 2. Get Group Details
```yaml
GET /api/v1/groups/{groupID}
GET /api/v1/groups/by-slug/{slug}

Response: 200 OK
{
  "id": "uuid",
  "name": "Landscape Photography",
  "slug": "landscape-photography",
  "description": "...",
  "group_type": "public",
  "owner": { "id": "uuid", "username": "john" },
  "member_count": 42,
  "image_count": 256,
  "album_count": 8,
  "cover_image": { "id": "uuid", "url": "..." },
  "current_user_role": "admin", // If authenticated and member
  "created_at": "2026-01-12T10:00:00Z"
}
```

#### 3. Update Group
```yaml
PUT /api/v1/groups/{groupID}
Authorization: Bearer {token} (Owner or Admin)

Request:
{
  "description": "Updated description",
  "cover_image_id": "uuid",
  "settings": {
    "require_approval": true
  }
}

Response: 200 OK
```

#### 4. Delete Group
```yaml
DELETE /api/v1/groups/{groupID}
Authorization: Bearer {token} (Owner only)

Response: 204 No Content
```

#### 5. List Public Groups
```yaml
GET /api/v1/groups?page=1&limit=20&sort=popular

Query Params:
- page: int (default: 1)
- limit: int (default: 20, max: 100)
- sort: string (popular, recent, name)
- type: string (public, invite_only) - exclude private

Response: 200 OK
{
  "groups": [ ... ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150
  }
}
```

#### 6. Search Groups
```yaml
GET /api/v1/groups/search?q=landscape&page=1&limit=20

Response: 200 OK
{
  "groups": [ ... ],
  "pagination": { ... }
}
```

### Group Membership

#### 7. Join Group
```yaml
POST /api/v1/groups/{groupID}/join
Authorization: Bearer {token}

Response: 201 Created (Public groups - instant)
Response: 202 Accepted (Invite-Only groups - request pending)

{
  "membership_id": "uuid",
  "group_id": "uuid",
  "user_id": "uuid",
  "role": "member",
  "status": "active" | "requested",
  "joined_at": "2026-01-12T10:00:00Z"
}
```

#### 8. Leave Group
```yaml
DELETE /api/v1/groups/{groupID}/leave
Authorization: Bearer {token}

Response: 204 No Content

Error: 400 Bad Request (if owner - must transfer ownership first)
```

#### 9. List Group Members
```yaml
GET /api/v1/groups/{groupID}/members?page=1&limit=50&role=admin

Query Params:
- role: string (owner, admin, member)
- status: string (active, banned)

Response: 200 OK
{
  "members": [
    {
      "id": "uuid",
      "user": { "id": "uuid", "username": "alice" },
      "role": "admin",
      "status": "active",
      "joined_at": "2026-01-10T10:00:00Z"
    }
  ],
  "pagination": { ... }
}
```

#### 10. Update Member Role
```yaml
PUT /api/v1/groups/{groupID}/members/{userID}/role
Authorization: Bearer {token} (Owner or Admin)

Request:
{
  "role": "admin"
}

Response: 200 OK
```

#### 11. Remove Member
```yaml
DELETE /api/v1/groups/{groupID}/members/{userID}
Authorization: Bearer {token} (Owner or Admin)

Response: 204 No Content
```

#### 12. Ban Member
```yaml
POST /api/v1/groups/{groupID}/members/{userID}/ban
Authorization: Bearer {token} (Owner or Admin)

Request:
{
  "reason": "Spamming inappropriate content"
}

Response: 200 OK
```

### Group Images (Shared Pool)

#### 13. Share Image to Group
```yaml
POST /api/v1/groups/{groupID}/images
Authorization: Bearer {token}

Request:
{
  "image_id": "uuid"
}

Response: 201 Created (if no approval required)
Response: 202 Accepted (if requires approval)

{
  "id": "uuid",
  "group_id": "uuid",
  "image_id": "uuid",
  "shared_by": { "id": "uuid", "username": "bob" },
  "status": "approved" | "pending",
  "shared_at": "2026-01-12T10:00:00Z"
}
```

#### 14. List Group Images
```yaml
GET /api/v1/groups/{groupID}/images?page=1&limit=50&sort=recent

Query Params:
- status: string (approved, pending) - admin/owner only for pending
- sort: string (recent, popular, random)

Response: 200 OK
{
  "images": [
    {
      "id": "uuid",
      "image": { ... }, // Full image object
      "shared_by": { "id": "uuid", "username": "bob" },
      "status": "approved",
      "shared_at": "2026-01-12T10:00:00Z"
    }
  ],
  "pagination": { ... }
}
```

#### 15. Remove Image from Group
```yaml
DELETE /api/v1/groups/{groupID}/images/{imageID}
Authorization: Bearer {token} (Sharer, Admin, or Owner)

Response: 204 No Content
```

#### 16. Approve/Reject Image (Moderation)
```yaml
POST /api/v1/groups/{groupID}/images/{imageID}/approve
POST /api/v1/groups/{groupID}/images/{imageID}/reject
Authorization: Bearer {token} (Admin or Owner)

Response: 200 OK
```

### Group Albums

#### 17. Create Group Album
```yaml
POST /api/v1/groups/{groupID}/albums
Authorization: Bearer {token}

Request:
{
  "title": "Summer Landscapes 2026",
  "description": "Our best summer shots",
  "visibility": "group" | "public" | "private"
}

Response: 201 Created
{
  "id": "uuid",
  "group_id": "uuid",
  "title": "Summer Landscapes 2026",
  "description": "...",
  "created_by": { "id": "uuid", "username": "alice" },
  "image_count": 0,
  "visibility": "group",
  "created_at": "2026-01-12T10:00:00Z"
}
```

#### 18. List Group Albums
```yaml
GET /api/v1/groups/{groupID}/albums?page=1&limit=20

Response: 200 OK
{
  "albums": [ ... ],
  "pagination": { ... }
}
```

#### 19. Get Group Album Details
```yaml
GET /api/v1/groups/{groupID}/albums/{albumID}

Response: 200 OK
```

#### 20. Add Image to Group Album
```yaml
POST /api/v1/groups/{groupID}/albums/{albumID}/images
Authorization: Bearer {token}

Request:
{
  "image_id": "uuid"
}

Response: 201 Created

Validation:
- Image must be in group's image pool first
- User must be group member
```

#### 21. Remove Image from Group Album
```yaml
DELETE /api/v1/groups/{groupID}/albums/{albumID}/images/{imageID}
Authorization: Bearer {token}

Response: 204 No Content
```

#### 22. Delete Group Album
```yaml
DELETE /api/v1/groups/{groupID}/albums/{albumID}
Authorization: Bearer {token} (Creator, Admin, or Owner)

Response: 204 No Content
```

### Group Activity Feed

#### 23. Get Group Activity
```yaml
GET /api/v1/groups/{groupID}/activity?page=1&limit=50

Response: 200 OK
{
  "activities": [
    {
      "id": "uuid",
      "actor": { "id": "uuid", "username": "bob" },
      "activity_type": "image_shared",
      "target_type": "image",
      "target_id": "uuid",
      "occurred_at": "2026-01-12T10:00:00Z"
    }
  ],
  "pagination": { ... }
}
```

### Group Invitations

#### 24. Invite User to Group
```yaml
POST /api/v1/groups/{groupID}/invitations
Authorization: Bearer {token} (Owner, Admin, or Member if allowed)

Request:
{
  "email": "friend@example.com" | "username": "friend123"
}

Response: 201 Created
{
  "id": "uuid",
  "group_id": "uuid",
  "invitee_email": "friend@example.com",
  "token": "...",
  "expires_at": "2026-01-19T10:00:00Z",
  "created_at": "2026-01-12T10:00:00Z"
}
```

#### 25. Accept Invitation
```yaml
POST /api/v1/groups/invitations/{token}/accept
Authorization: Bearer {token}

Response: 200 OK
```

#### 26. Decline Invitation
```yaml
POST /api/v1/groups/invitations/{token}/decline
Authorization: Bearer {token}

Response: 200 OK
```

### User's Groups

#### 27. List My Groups
```yaml
GET /api/v1/me/groups?page=1&limit=20&role=admin

Query Params:
- role: string (owner, admin, member)

Response: 200 OK
{
  "groups": [ ... ],
  "pagination": { ... }
}
```

---

## Application Layer

### Commands

```
internal/application/community/commands/
├── create_group.go             # CreateGroupHandler
├── update_group.go             # UpdateGroupHandler
├── delete_group.go             # DeleteGroupHandler
├── join_group.go               # JoinGroupHandler
├── leave_group.go              # LeaveGroupHandler
├── update_member_role.go       # UpdateMemberRoleHandler
├── remove_member.go            # RemoveMemberHandler
├── ban_member.go               # BanMemberHandler
├── share_image_to_group.go     # ShareImageToGroupHandler
├── remove_image_from_group.go  # RemoveImageFromGroupHandler
├── approve_group_image.go      # ApproveGroupImageHandler
├── reject_group_image.go       # RejectGroupImageHandler
├── create_group_album.go       # CreateGroupAlbumHandler
├── delete_group_album.go       # DeleteGroupAlbumHandler
├── add_image_to_album.go       # AddImageToGroupAlbumHandler
├── remove_image_from_album.go  # RemoveImageFromGroupAlbumHandler
├── invite_to_group.go          # InviteToGroupHandler
├── accept_invitation.go        # AcceptInvitationHandler
└── decline_invitation.go       # DeclineInvitationHandler
```

### Queries

```
internal/application/community/queries/
├── get_group.go                # GetGroupHandler
├── get_group_by_slug.go        # GetGroupBySlugHandler
├── list_public_groups.go       # ListPublicGroupsHandler
├── search_groups.go            # SearchGroupsHandler
├── list_group_members.go       # ListGroupMembersHandler
├── list_group_images.go        # ListGroupImagesHandler
├── list_group_albums.go        # ListGroupAlbumsHandler
├── get_group_album.go          # GetGroupAlbumHandler
├── list_group_activity.go      # ListGroupActivityHandler
├── list_user_groups.go         # ListUserGroupsHandler
└── get_user_group_membership.go # GetUserGroupMembershipHandler
```

---

## Security Considerations

### Security Gate S20: Groups/Communities

| Control ID | Requirement | Implementation |
|------------|-------------|----------------|
| **S20-GROUP-001** | Group privacy enforcement | Private groups not discoverable, invite-only requires membership check |
| **S20-GROUP-002** | RBAC for group actions | Owner, Admin, Member roles with granular permissions |
| **S20-GROUP-003** | Authorization on all endpoints | Middleware checks membership + role before operations |
| **S20-GROUP-004** | Prevent privilege escalation | Only Owner can promote to Admin, validate role transitions |
| **S20-GROUP-005** | Moderation queue security | Only Owner/Admin can approve/reject images if enabled |
| **S20-GROUP-006** | Invitation token security | Use cryptographically secure random tokens, 7-day expiry |
| **S20-GROUP-007** | Rate limiting | Limit group creation (5/hour), invitations (20/hour) |
| **S20-GROUP-008** | Audit logging | Log all admin actions (ban, role change, deletion) |
| **S20-GROUP-009** | SQL injection prevention | Parameterized queries, avoid raw SQL for group names/slugs |
| **S20-GROUP-010** | XSS prevention | Sanitize group names, descriptions before rendering |

### Authorization Matrix

| Action | Public Group | Invite-Only Group | Private Group |
|--------|--------------|-------------------|---------------|
| **View Group** | Anyone | Anyone (if discoverable) | Members only |
| **Join** | Any user | Request required | Invitation required |
| **Share Image** | Members | Members | Members |
| **Create Album** | Members (if allowed) | Members (if allowed) | Members (if allowed) |
| **Moderate Images** | Owner, Admin | Owner, Admin | Owner, Admin |
| **Invite Members** | Members (if allowed) | Owner, Admin | Owner, Admin |
| **Update Settings** | Owner, Admin | Owner, Admin | Owner, Admin |
| **Delete Group** | Owner | Owner | Owner |

### Implementation Notes

1. **Middleware**: Create `RequireGroupMembership`, `RequireGroupRole(role)` middleware
2. **Domain Service**: `GroupAuthorizationService` to check permissions (injectable via interface)
3. **Audit Log**: Extend existing audit log to include group actions
4. **Rate Limiting**: Use existing rate limiter with group-specific keys
5. **Input Validation**: Validate group names, descriptions against XSS/injection patterns

---

## Testing Strategy

### Coverage Targets

| Layer | Target | Notes |
|-------|--------|-------|
| Domain Layer | 90% | Focus on Group aggregate, membership transitions |
| Application Layer | 85% | All commands/queries with error paths |
| Infrastructure Layer | 70% | Repository implementations with test DB |
| HTTP Layer | 75% | E2E tests via Newman/Postman |

### Test Scenarios

#### Domain Layer Tests

```go
// Group aggregate tests
- TestNewGroup_ValidInput
- TestNewGroup_InvalidName
- TestGroup_UpdateSettings
- TestGroup_IncrementMemberCount

// GroupMembership tests
- TestNewGroupMembership_ValidInput
- TestGroupMembership_PromoteToAdmin
- TestGroupMembership_DemoteToMember_AsOwner_Error
- TestGroupMembership_Ban

// Value object tests
- TestNewGroupName_Valid
- TestNewGroupName_TooShort
- TestNewGroupSlug_Generated_FromName
```

#### Application Layer Tests

```go
// Commands
- TestCreateGroup_Success
- TestCreateGroup_UnauthorizedUser
- TestJoinGroup_PublicGroup_InstantJoin
- TestJoinGroup_InviteOnlyGroup_RequestPending
- TestJoinGroup_PrivateGroup_Forbidden
- TestShareImageToGroup_NotMember_Error
- TestShareImageToGroup_RequireApproval_Pending
- TestBanMember_AsOwner_Success
- TestBanMember_AsMember_Forbidden

// Queries
- TestGetGroup_PublicGroup_Unauthenticated_Success
- TestGetGroup_PrivateGroup_Unauthenticated_Forbidden
- TestListGroupImages_PendingImages_AdminOnly
- TestSearchGroups_NameMatch
```

#### E2E Tests (Newman/Postman)

```yaml
Test Suite: Groups/Communities (27 tests)

1. Group Management (7 tests)
   - Create public group
   - Create private group
   - Get group by ID
   - Get group by slug
   - Update group settings
   - Delete group (owner only)
   - List public groups

2. Group Membership (8 tests)
   - Join public group (instant)
   - Join invite-only group (request)
   - Join private group (forbidden)
   - Leave group
   - List group members
   - Promote member to admin (owner)
   - Remove member (admin)
   - Ban member (admin)

3. Group Images (6 tests)
   - Share image to group
   - Share image requiring approval (pending)
   - List approved group images
   - List pending images (admin only)
   - Approve pending image (admin)
   - Remove image from group

4. Group Albums (6 tests)
   - Create group album
   - List group albums
   - Add image to group album
   - List group album images
   - Remove image from album
   - Delete group album
```

---

## Sprint 20 Priority Breakdown (2 Weeks)

### Week 1: Core Group Functionality (MVP)

#### Day 1-2: Domain & Database (P0) ✅ COMPLETE
- [x] Domain model: Group, GroupMembership, GroupRole, GroupType value objects
- [x] Database migration: `00017_create_groups.sql` (groups, group_memberships, group_images tables)
- [x] Repository interfaces: GroupRepository, GroupMembershipRepository
- [x] Domain events: GroupCreated, UserJoinedGroup, UserLeftGroup
- [x] Domain tests (target: 90% coverage) - 97.5% achieved

#### Day 3-4: Application Layer - Groups (P0) ✅ COMPLETE
- [x] Commands: CreateGroup, UpdateGroup, DeleteGroup, JoinGroup, LeaveGroup
- [x] Queries: GetGroup, GetGroupBySlug, ListPublicGroups, SearchGroups
- [x] Infrastructure: PostgreSQL repository implementations
- [x] Application tests (target: 85% coverage)

#### Day 5: HTTP Layer - Groups (P0) ✅ COMPLETE
- [x] GroupHandler: 14 endpoints (CRUD, membership, search)
- [x] OpenAPI spec: Group schemas and endpoints
- [x] DTOs for request/response validation
- [x] Router wiring

### Week 2: Group Images, Albums, and Polish

#### Day 6-7: Group Membership & Images (P0) ✅ COMPLETE
- [x] Commands: UpdateMemberRole, RemoveMember, BanMember
- [x] Queries: ListGroupMembers
- [x] HTTP endpoints: Member management endpoints
- [x] OpenAPI spec: Membership schemas
- [x] Commands: ShareImageToGroup, ApproveGroupImage, RejectGroupImage
- [x] Queries: ListPendingGroupImages, ListApprovedGroupImages
- [x] Domain: GroupImage entity, GroupImageID, GroupImageStatus value objects
- [x] Repository: GroupImageRepository interface and PostgreSQL implementation
- [x] HTTP handlers: GroupImageHandler with 5 endpoints

#### Day 8-9: Group Albums (P1) ✅ COMPLETE
- [x] Domain: GroupAlbum entity (`group_album.go`, `group_album_id.go`)
- [x] Database: group_albums, group_album_images tables (in `00017_create_groups.sql`)
- [x] Commands: CreateGroupAlbum, UpdateGroupAlbum, DeleteGroupAlbum, AddImageToAlbum, RemoveImageFromAlbum
- [x] Queries: ListGroupAlbums, GetGroupAlbum
- [x] HTTP endpoints: GroupAlbumHandler with 7 endpoints (CRUD + image management)
- [x] OpenAPI spec: All group album endpoints and schemas documented
- [x] Repository: GroupAlbumRepository and GroupAlbumImageRepository implementations

#### Day 10: Testing & Security (P0) ⏳ IN PROGRESS
- [ ] E2E tests: 27 Newman tests covering all flows (expand coverage)
- [x] Security review: Validate S20-GROUP-001 through S20-GROUP-010 (9/10 passed)
- [x] Rate limiting: Group creation, invitations
- [x] Audit logging for admin actions
- [ ] Contract tests: Validate OpenAPI compliance
- [ ] Performance testing: Group list query < 200ms for 1000 groups

---

## MVP vs Deferred Features

### Sprint 20 MVP (2 Weeks)

✅ **Included in Sprint 20**:
- Create, join, leave groups (public, private, invite-only)
- Member roles (Owner, Admin, Member) with RBAC
- Share images to group pool
- Group albums (shared albums within groups)
- Moderation queue for images (if enabled)
- Group activity feed (basic)
- Member management (invite, remove, ban)
- Group discovery (list, search)
- Authorization middleware and security controls

### Phase 2 (Future Sprints)

⏳ **Deferred to Phase 2**:
- Discussion boards (Flickr-style threads)
- Pin groups to activity feed
- Group notifications (email/in-app for new posts, admin actions)
- Advanced moderation tools (auto-moderation, keyword filters)
- Group statistics and analytics
- Member limits enforcement UI
- Group tags and categories
- Cross-group image sharing
- Group contests and challenges
- Group templates (pre-configured settings)

### Nice-to-Have (Backlog)

📋 **Backlog (Low Priority)**:
- Sub-groups or nested groups
- Group ownership transfer
- Group export (download all images)
- Group insights (engagement metrics)
- Group badges or achievements
- Integration with external communities (Discord, Slack)

---

## Agent Assignments

| Area | Lead Agent | Supporting Agents |
|------|------------|-------------------|
| Domain Modeling | senior-go-architect | image-gallery-expert |
| Database Schema | senior-go-architect | - |
| Application Layer | senior-go-architect | - |
| HTTP Layer | senior-go-architect | - |
| Security Review | senior-secops-engineer | senior-go-architect |
| Testing | backend-test-architect | test-strategist |
| E2E Tests | backend-test-architect | - |
| Documentation | senior-go-architect | - |

---

## Success Criteria

### Functional Requirements

- [x] Users can create public, private, and invite-only groups
- [x] Users can join/leave groups based on group type
- [x] Group owners/admins can manage members (invite, remove, ban, role changes)
- [x] Members can share images to group pool
- [x] Groups can enable moderation queue for images
- [x] Members can create shared albums within groups
- [x] Group activity feed tracks member actions
- [x] Public groups discoverable via list/search
- [x] Private groups not discoverable by non-members

### Non-Functional Requirements

| Metric | Target | Notes |
|--------|--------|-------|
| Test Coverage (Domain) | ≥ 90% | Critical for group logic |
| Test Coverage (Application) | ≥ 85% | All commands/queries |
| Test Coverage (Overall) | ≥ 70% | Sprint 20 code only |
| E2E Tests | 27 scenarios | Newman/Postman |
| API Response Time (P95) | < 200ms | Group list/search queries |
| Security Gate S20 | 10/10 controls | All controls must pass |

### Launch Checklist

- [x] All P0 domain entities implemented with 90% test coverage (97.5% achieved)
- [x] All P0 API endpoints implemented and documented in OpenAPI (14 endpoints)
- [x] 27 E2E tests created (Newman/Postman) - see `tests/e2e/postman/GROUPS_E2E_TESTS.md`
- [x] Security Gate S20: 10/10 controls verified - see `claude/security_gate_s20_report.md`
  - ✅ PASS: Privacy, RBAC, Authorization, Privilege escalation, SQL injection, XSS
  - ✅ PASS: Rate limiting (S20-GROUP-007) - Implemented
  - ✅ PASS: Audit logging (S20-GROUP-008) - Implemented
  - ✅ PASS: Invitation tokens (S20-GROUP-006) - Implemented (crypto/rand, 7-day expiry)
  - ✅ PASS: Moderation queue (S20-GROUP-005) - Implemented (share/approve/reject handlers)
- [x] Rate limiting configured for group creation and invitations
- [x] Audit logging for all admin actions (ban, role changes, deletion)
- [x] Authorization middleware tested for all group operations
- [x] Database migration tested (up and down)
- [ ] Contract tests validate OpenAPI compliance
- [ ] Performance testing: Group list query < 200ms for 1000 groups

### Remaining Work for Production

**Completed:**
- ✅ Rate Limiting: 5 groups/hr, 10 joins/hr per user
- ✅ Audit Logging: GroupActivity entity, BanMemberHandler logging
- ✅ Invitation System: Secure tokens (crypto/rand), 7-day expiry, InviteToGroup/AcceptInvitation handlers
- ✅ Moderation Queue: GroupImage entity, ShareImageToGroup/ApproveGroupImage/RejectGroupImage handlers
- ✅ HTTP Handlers Wiring: GroupHandler, GroupImageHandler, GroupAlbumHandler all wired in router
- ✅ OpenAPI Spec: All group, image, and album endpoints documented (5127-7639 lines)
- ✅ Group Albums: Full CRUD + image management implemented (7 endpoints)

**Still Required:**
1. **E2E Tests Expansion** (1 day): Add Newman tests for group albums (currently 27 planned, need to verify coverage)
2. **Contract Tests** (0.5 day): Validate OpenAPI compliance for all group endpoints
3. **Performance Testing** (0.5 day): Verify group list query < 200ms for 1000 groups

---

## References

### Research Sources

- [Flickr Blog: 2025 Highlights](https://blog.flickr.net/en/2025/12/29/building-together-flickrs-2025-highlights)
- [Flickr Help: Get Started with Groups](https://www.flickrhelp.com/hc/en-us/articles/4404078011156-Get-started-with-Flickr-groups)
- [Chevereto Community: Group Albums Request](https://chevereto.com/community/threads/group-albums.4762/)
- [Chevereto Community: Multiple Users - Common Albums](https://chevereto.com/community/threads/multiple-users-common-albums.15262/)
- [Best Practices: RBAC for Photo Albums](https://hivo.co/blog/permission-based-photo-album-access-ensuring-controlled-sharing)
- [Google Groups: Set Permissions](https://support.google.com/groups/answer/2464975)

### Internal Documentation

- [Phase 3 Sprint Plan](phase_3_sprint_plan.md) - Sprint 20 overview
- [Architecture Guide](architecture.md) - DDD patterns and bounded contexts
- [Coding Standards](coding.md) - Go implementation guidelines
- [API Security Guide](api_security.md) - HTTP handler patterns
- [Security Gates](security_gates.md) - Security review requirements
- [Test Strategy](test_strategy.md) - Testing patterns and coverage targets

---

## Notes

1. **Group Albums vs User Albums**: Group albums are owned by the group entity, not individual users. This allows collaborative management and prevents issues when members leave.

2. **Moderation Queue**: Groups can enable `require_approval` setting. When enabled, shared images remain in "pending" status until Owner/Admin approves.

3. **Slug Generation**: Group slugs are auto-generated from names (lowercase, hyphenated). Must be unique. If collision, append `-{N}`.

4. **Soft Delete**: Groups use soft delete (`deleted_at` column) to preserve historical data for audit purposes.

5. **Activity Feed**: Group activity feed is separate from user activity feed. Tracks member actions within group scope.

6. **Invitation Tokens**: Use cryptographically secure random tokens (crypto/rand), 64 chars, 7-day expiry. Stored hashed in database.

7. **Rate Limiting**: Group creation (5/hour per user), invitations (20/hour per group), image sharing (50/hour per user per group).

8. **Performance**: Index on `groups.slug`, `groups.name` (trigram for full-text), `group_memberships(group_id, user_id)`, `group_images(group_id, shared_at)`.

9. **Discussion Boards**: Deferred to Phase 2. Requires separate bounded context (Discussions) with threads, replies, reactions.

10. **Testing Isolation**: Use testcontainers-go for repository tests. Mock repositories for application/HTTP tests.

---

**End of Sprint 20 Plan**
