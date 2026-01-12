-- +goose Up
-- +goose StatementBegin
-- Groups/Communities feature - Sprint 20
-- Enables users to create and join interest-based groups with shared albums and image pools

-- Enable pg_trgm extension for fuzzy text search on group names
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ============================================================================
-- Main groups table
-- ============================================================================
CREATE TABLE groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    group_type VARCHAR(20) NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Settings (denormalized for query performance)
    require_approval BOOLEAN NOT NULL DEFAULT FALSE,
    allow_member_invites BOOLEAN NOT NULL DEFAULT TRUE,
    allow_member_albums BOOLEAN NOT NULL DEFAULT TRUE,
    max_members INTEGER NOT NULL DEFAULT 0,

    -- Denormalized counts (updated via triggers)
    member_count INTEGER NOT NULL DEFAULT 1,
    image_count INTEGER NOT NULL DEFAULT 0,
    album_count INTEGER NOT NULL DEFAULT 0,

    -- Optional cover image
    cover_image_id UUID REFERENCES images(id) ON DELETE SET NULL,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT groups_name_not_empty CHECK (LENGTH(TRIM(name)) > 0),
    CONSTRAINT groups_name_length CHECK (LENGTH(name) BETWEEN 3 AND 100),
    CONSTRAINT groups_slug_not_empty CHECK (LENGTH(TRIM(slug)) > 0),
    CONSTRAINT groups_slug_format CHECK (slug ~ '^[a-z0-9-]+$'),
    CONSTRAINT groups_description_length CHECK (LENGTH(description) <= 1000),
    CONSTRAINT groups_type_check CHECK (group_type IN ('public', 'private', 'invite_only')),
    CONSTRAINT groups_max_members_positive CHECK (max_members >= 0),
    CONSTRAINT groups_member_count_positive CHECK (member_count >= 0),
    CONSTRAINT groups_image_count_positive CHECK (image_count >= 0),
    CONSTRAINT groups_album_count_positive CHECK (album_count >= 0)
);

-- Indexes for groups
CREATE INDEX idx_groups_slug ON groups(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_groups_owner_id ON groups(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_groups_type ON groups(group_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_groups_created_at ON groups(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_groups_member_count ON groups(member_count DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_groups_name_trgm ON groups USING gin(name gin_trgm_ops) WHERE deleted_at IS NULL;

COMMENT ON TABLE groups IS 'Interest-based communities for shared image collections';
COMMENT ON COLUMN groups.name IS 'Display name of the group (3-100 chars)';
COMMENT ON COLUMN groups.slug IS 'URL-safe unique identifier';
COMMENT ON COLUMN groups.group_type IS 'Access control: public, private, invite_only';
COMMENT ON COLUMN groups.require_approval IS 'If true, images need admin approval before appearing';
COMMENT ON COLUMN groups.allow_member_invites IS 'If true, members can invite others';
COMMENT ON COLUMN groups.allow_member_albums IS 'If true, members can create group albums';
COMMENT ON COLUMN groups.max_members IS 'Maximum member capacity (0 = unlimited)';
COMMENT ON COLUMN groups.member_count IS 'Denormalized count of active members';
COMMENT ON COLUMN groups.image_count IS 'Denormalized count of images in group pool';
COMMENT ON COLUMN groups.album_count IS 'Denormalized count of group albums';

-- ============================================================================
-- Group memberships table
-- ============================================================================
CREATE TABLE group_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    invited_by UUID REFERENCES users(id) ON DELETE SET NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT memberships_role_check CHECK (role IN ('member', 'admin', 'owner')),
    CONSTRAINT memberships_status_check CHECK (status IN ('invited', 'requested', 'active', 'banned')),

    -- Unique constraint: one active membership per user per group
    UNIQUE(group_id, user_id)
);

-- Indexes for group memberships
CREATE INDEX idx_memberships_group_id ON group_memberships(group_id);
CREATE INDEX idx_memberships_user_id ON group_memberships(user_id);
CREATE INDEX idx_memberships_status ON group_memberships(group_id, status);
CREATE INDEX idx_memberships_role ON group_memberships(group_id, role);
CREATE INDEX idx_memberships_joined_at ON group_memberships(joined_at DESC);

COMMENT ON TABLE group_memberships IS 'User membership in groups with roles and status';
COMMENT ON COLUMN group_memberships.role IS 'Permission level: member, admin, owner';
COMMENT ON COLUMN group_memberships.status IS 'Membership state: invited, requested, active, banned';
COMMENT ON COLUMN group_memberships.invited_by IS 'User who invited this member (NULL for self-join)';

-- ============================================================================
-- Group albums table (shared albums within groups)
-- ============================================================================
CREATE TABLE group_albums (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    cover_image_id UUID REFERENCES images(id) ON DELETE SET NULL,
    image_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT group_albums_title_not_empty CHECK (LENGTH(TRIM(title)) > 0),
    CONSTRAINT group_albums_title_length CHECK (LENGTH(title) BETWEEN 1 AND 255),
    CONSTRAINT group_albums_description_length CHECK (LENGTH(description) <= 1000),
    CONSTRAINT group_albums_image_count_positive CHECK (image_count >= 0)
);

-- Indexes for group albums
CREATE INDEX idx_group_albums_group_id ON group_albums(group_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_group_albums_created_by ON group_albums(created_by) WHERE deleted_at IS NULL;
CREATE INDEX idx_group_albums_created_at ON group_albums(created_at DESC) WHERE deleted_at IS NULL;

COMMENT ON TABLE group_albums IS 'Shared albums within groups';
COMMENT ON COLUMN group_albums.created_by IS 'User who created the album';
COMMENT ON COLUMN group_albums.image_count IS 'Denormalized count of images in album';

-- ============================================================================
-- Group album images table (images in group albums)
-- ============================================================================
CREATE TABLE group_album_images (
    group_album_id UUID NOT NULL REFERENCES group_albums(id) ON DELETE CASCADE,
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    added_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    position INTEGER NOT NULL DEFAULT 0,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (group_album_id, image_id),
    CONSTRAINT group_album_images_position_positive CHECK (position >= 0)
);

-- Indexes for group album images
CREATE INDEX idx_group_album_images_album_position ON group_album_images(group_album_id, position);
CREATE INDEX idx_group_album_images_image_id ON group_album_images(image_id);
CREATE INDEX idx_group_album_images_added_by ON group_album_images(added_by);

COMMENT ON TABLE group_album_images IS 'Association between group albums and images with ordering';
COMMENT ON COLUMN group_album_images.position IS 'Display order within the album';
COMMENT ON COLUMN group_album_images.added_by IS 'User who added the image to the album';

-- ============================================================================
-- Group images table (images shared directly to group pool)
-- ============================================================================
CREATE TABLE group_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    shared_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    shared_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT group_images_status_check CHECK (status IN ('pending', 'approved', 'rejected')),

    -- Unique constraint: one instance of an image per group
    UNIQUE(group_id, image_id)
);

-- Indexes for group images
CREATE INDEX idx_group_images_group_id ON group_images(group_id, status);
CREATE INDEX idx_group_images_image_id ON group_images(image_id);
CREATE INDEX idx_group_images_shared_by ON group_images(shared_by);
CREATE INDEX idx_group_images_shared_at ON group_images(shared_at DESC);
CREATE INDEX idx_group_images_status ON group_images(status) WHERE status = 'pending';

COMMENT ON TABLE group_images IS 'Images shared to group pools (may require approval)';
COMMENT ON COLUMN group_images.status IS 'Approval status: pending, approved, rejected';
COMMENT ON COLUMN group_images.reviewed_by IS 'Admin/owner who approved or rejected the image';

-- ============================================================================
-- Group invitations table (invitation tokens)
-- ============================================================================
CREATE TABLE group_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    invited_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT group_invitations_target_check CHECK (
        (email IS NOT NULL AND user_id IS NULL) OR
        (email IS NULL AND user_id IS NOT NULL)
    ),
    CONSTRAINT group_invitations_expires_future CHECK (expires_at > created_at)
);

-- Indexes for group invitations
CREATE INDEX idx_group_invitations_group_id ON group_invitations(group_id);
CREATE INDEX idx_group_invitations_user_id ON group_invitations(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_group_invitations_email ON group_invitations(email) WHERE email IS NOT NULL;
CREATE INDEX idx_group_invitations_token ON group_invitations(token);
CREATE INDEX idx_group_invitations_expires_at ON group_invitations(expires_at) WHERE used_at IS NULL;

COMMENT ON TABLE group_invitations IS 'Group invitation tokens for email or existing users';
COMMENT ON COLUMN group_invitations.email IS 'Email address for non-registered users';
COMMENT ON COLUMN group_invitations.user_id IS 'User ID for existing registered users';
COMMENT ON COLUMN group_invitations.token IS 'Unique invitation token';
COMMENT ON COLUMN group_invitations.expires_at IS 'Expiration timestamp for invitation';

-- ============================================================================
-- Group activities table (activity feed)
-- ============================================================================
CREATE TABLE group_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    activity_type VARCHAR(50) NOT NULL,
    target_id UUID,
    target_type VARCHAR(20),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT group_activities_type_check CHECK (activity_type IN (
        'member_joined', 'member_left', 'member_promoted', 'member_demoted',
        'image_shared', 'image_approved', 'image_rejected',
        'album_created', 'album_deleted',
        'settings_updated', 'description_updated'
    )),
    CONSTRAINT group_activities_target_check CHECK (
        (target_id IS NULL AND target_type IS NULL) OR
        (target_id IS NOT NULL AND target_type IS NOT NULL)
    ),
    CONSTRAINT group_activities_target_type_check CHECK (
        target_type IS NULL OR target_type IN ('user', 'image', 'album', 'group')
    )
);

-- Indexes for group activities
CREATE INDEX idx_group_activities_group_id ON group_activities(group_id, created_at DESC);
CREATE INDEX idx_group_activities_actor_id ON group_activities(actor_id);
CREATE INDEX idx_group_activities_type ON group_activities(group_id, activity_type);
CREATE INDEX idx_group_activities_created_at ON group_activities(created_at DESC);

COMMENT ON TABLE group_activities IS 'Activity feed for group events';
COMMENT ON COLUMN group_activities.activity_type IS 'Type of activity event';
COMMENT ON COLUMN group_activities.target_id IS 'ID of the affected entity (image, album, user)';
COMMENT ON COLUMN group_activities.target_type IS 'Type of the target entity';
COMMENT ON COLUMN group_activities.metadata IS 'Flexible JSON metadata for activity details';

-- ============================================================================
-- Triggers for automatic count updates
-- ============================================================================

-- Trigger to update groups.member_count
CREATE OR REPLACE FUNCTION update_group_member_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status = 'active' THEN
            UPDATE groups SET member_count = member_count + 1, updated_at = NOW()
            WHERE id = NEW.group_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        -- Status changed to active
        IF OLD.status != 'active' AND NEW.status = 'active' THEN
            UPDATE groups SET member_count = member_count + 1, updated_at = NOW()
            WHERE id = NEW.group_id;
        -- Status changed from active
        ELSIF OLD.status = 'active' AND NEW.status != 'active' THEN
            UPDATE groups SET member_count = GREATEST(member_count - 1, 0), updated_at = NOW()
            WHERE id = NEW.group_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.status = 'active' THEN
            UPDATE groups SET member_count = GREATEST(member_count - 1, 0), updated_at = NOW()
            WHERE id = OLD.group_id;
        END IF;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_group_member_count
    AFTER INSERT OR UPDATE OF status OR DELETE ON group_memberships
    FOR EACH ROW
    EXECUTE FUNCTION update_group_member_count();

-- Trigger to update groups.image_count
CREATE OR REPLACE FUNCTION update_group_image_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status = 'approved' THEN
            UPDATE groups SET image_count = image_count + 1, updated_at = NOW()
            WHERE id = NEW.group_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        -- Status changed to approved
        IF OLD.status != 'approved' AND NEW.status = 'approved' THEN
            UPDATE groups SET image_count = image_count + 1, updated_at = NOW()
            WHERE id = NEW.group_id;
        -- Status changed from approved
        ELSIF OLD.status = 'approved' AND NEW.status != 'approved' THEN
            UPDATE groups SET image_count = GREATEST(image_count - 1, 0), updated_at = NOW()
            WHERE id = NEW.group_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.status = 'approved' THEN
            UPDATE groups SET image_count = GREATEST(image_count - 1, 0), updated_at = NOW()
            WHERE id = OLD.group_id;
        END IF;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_group_image_count
    AFTER INSERT OR UPDATE OF status OR DELETE ON group_images
    FOR EACH ROW
    EXECUTE FUNCTION update_group_image_count();

-- Trigger to update groups.album_count
CREATE OR REPLACE FUNCTION update_group_album_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE groups SET album_count = album_count + 1, updated_at = NOW()
        WHERE id = NEW.group_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.deleted_at IS NULL THEN
            UPDATE groups SET album_count = GREATEST(album_count - 1, 0), updated_at = NOW()
            WHERE id = OLD.group_id;
        END IF;
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        -- Soft delete
        IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
            UPDATE groups SET album_count = GREATEST(album_count - 1, 0), updated_at = NOW()
            WHERE id = NEW.group_id;
        -- Restore from soft delete
        ELSIF OLD.deleted_at IS NOT NULL AND NEW.deleted_at IS NULL THEN
            UPDATE groups SET album_count = album_count + 1, updated_at = NOW()
            WHERE id = NEW.group_id;
        END IF;
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_group_album_count
    AFTER INSERT OR UPDATE OF deleted_at OR DELETE ON group_albums
    FOR EACH ROW
    EXECUTE FUNCTION update_group_album_count();

-- Trigger to update group_albums.image_count
CREATE OR REPLACE FUNCTION update_group_album_image_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE group_albums SET image_count = image_count + 1, updated_at = NOW()
        WHERE id = NEW.group_album_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE group_albums SET image_count = GREATEST(image_count - 1, 0), updated_at = NOW()
        WHERE id = OLD.group_album_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_group_album_image_count
    AFTER INSERT OR DELETE ON group_album_images
    FOR EACH ROW
    EXECUTE FUNCTION update_group_album_image_count();

-- Trigger to update groups.updated_at
CREATE OR REPLACE FUNCTION update_groups_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_groups_updated_at
    BEFORE UPDATE ON groups
    FOR EACH ROW
    EXECUTE FUNCTION update_groups_updated_at();

-- Trigger to update group_memberships.updated_at
CREATE TRIGGER trg_group_memberships_updated_at
    BEFORE UPDATE ON group_memberships
    FOR EACH ROW
    EXECUTE FUNCTION update_groups_updated_at();

-- Trigger to update group_albums.updated_at
CREATE TRIGGER trg_group_albums_updated_at
    BEFORE UPDATE ON group_albums
    FOR EACH ROW
    EXECUTE FUNCTION update_groups_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop in reverse order to handle dependencies

-- Drop triggers
DROP TRIGGER IF EXISTS trg_group_albums_updated_at ON group_albums;
DROP TRIGGER IF EXISTS trg_group_memberships_updated_at ON group_memberships;
DROP TRIGGER IF EXISTS trg_groups_updated_at ON groups;
DROP TRIGGER IF EXISTS trg_group_album_image_count ON group_album_images;
DROP TRIGGER IF EXISTS trg_group_album_count ON group_albums;
DROP TRIGGER IF EXISTS trg_group_image_count ON group_images;
DROP TRIGGER IF EXISTS trg_group_member_count ON group_memberships;

-- Drop functions
DROP FUNCTION IF EXISTS update_groups_updated_at();
DROP FUNCTION IF EXISTS update_group_album_image_count();
DROP FUNCTION IF EXISTS update_group_album_count();
DROP FUNCTION IF EXISTS update_group_image_count();
DROP FUNCTION IF EXISTS update_group_member_count();

-- Drop tables
DROP TABLE IF EXISTS group_activities;
DROP TABLE IF EXISTS group_invitations;
DROP TABLE IF EXISTS group_images;
DROP TABLE IF EXISTS group_album_images;
DROP TABLE IF EXISTS group_albums;
DROP TABLE IF EXISTS group_memberships;
DROP TABLE IF EXISTS groups;

-- Note: We don't drop pg_trgm extension as it may be used elsewhere
-- +goose StatementEnd
