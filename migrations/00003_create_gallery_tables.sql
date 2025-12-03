-- +goose Up
-- Gallery context tables: images, variants, albums, tags

-- Images table - core image storage metadata
CREATE TABLE images (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255),
    description TEXT,
    storage_provider VARCHAR(20) NOT NULL,
    storage_key VARCHAR(512) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(50) NOT NULL,
    file_size BIGINT NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'processing',
    visibility VARCHAR(20) NOT NULL DEFAULT 'private',
    scan_status VARCHAR(20) DEFAULT 'pending',
    view_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT images_status_check CHECK (status IN ('processing', 'active', 'failed', 'deleted')),
    CONSTRAINT images_visibility_check CHECK (visibility IN ('public', 'private', 'unlisted')),
    CONSTRAINT images_scan_status_check CHECK (scan_status IN ('pending', 'clean', 'infected', 'error')),
    CONSTRAINT images_file_size_positive CHECK (file_size > 0),
    CONSTRAINT images_dimensions_positive CHECK (width > 0 AND height > 0)
);

CREATE INDEX idx_images_owner_id ON images(owner_id);
CREATE INDEX idx_images_status ON images(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_images_visibility ON images(visibility) WHERE status = 'active' AND deleted_at IS NULL;
CREATE INDEX idx_images_created_at ON images(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_images_owner_created ON images(owner_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_images_storage_key ON images(storage_provider, storage_key);

COMMENT ON TABLE images IS 'Core image metadata and storage references';
COMMENT ON COLUMN images.storage_provider IS 'Storage backend: local, s3, spaces, b2';
COMMENT ON COLUMN images.storage_key IS 'Key/path within the storage provider';
COMMENT ON COLUMN images.status IS 'Processing state: processing, active, failed, deleted';
COMMENT ON COLUMN images.visibility IS 'Access level: public, private, unlisted';
COMMENT ON COLUMN images.scan_status IS 'ClamAV scan result: pending, clean, infected, error';

-- Image variants table - generated thumbnails and resizes
CREATE TABLE image_variants (
    id UUID PRIMARY KEY,
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    variant_type VARCHAR(20) NOT NULL,
    storage_key VARCHAR(512) NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    file_size BIGINT NOT NULL,
    format VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT variants_type_check CHECK (variant_type IN ('thumbnail', 'small', 'medium', 'large', 'original')),
    CONSTRAINT variants_format_check CHECK (format IN ('jpeg', 'png', 'gif', 'webp')),
    CONSTRAINT variants_dimensions_positive CHECK (width > 0 AND height > 0),
    CONSTRAINT variants_file_size_positive CHECK (file_size > 0),
    UNIQUE(image_id, variant_type)
);

CREATE INDEX idx_variants_image_id ON image_variants(image_id);

COMMENT ON TABLE image_variants IS 'Generated image variants (thumbnails, resizes)';
COMMENT ON COLUMN image_variants.variant_type IS 'Size variant: thumbnail (160px), small (320px), medium (800px), large (1600px), original';
COMMENT ON COLUMN image_variants.format IS 'Output format: jpeg, png, gif, webp';

-- Albums table - image collections
CREATE TABLE albums (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    visibility VARCHAR(20) NOT NULL DEFAULT 'private',
    cover_image_id UUID REFERENCES images(id) ON DELETE SET NULL,
    image_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT albums_visibility_check CHECK (visibility IN ('public', 'private', 'unlisted'))
);

CREATE INDEX idx_albums_owner_id ON albums(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_albums_visibility ON albums(visibility) WHERE deleted_at IS NULL;
CREATE INDEX idx_albums_created_at ON albums(created_at DESC) WHERE deleted_at IS NULL;

COMMENT ON TABLE albums IS 'User-created image collections';
COMMENT ON COLUMN albums.cover_image_id IS 'Featured image for album thumbnail';
COMMENT ON COLUMN albums.image_count IS 'Denormalized count of images in album';

-- Album-image relationship (many-to-many with ordering)
CREATE TABLE album_images (
    album_id UUID NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    position INTEGER NOT NULL DEFAULT 0,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (album_id, image_id),
    CONSTRAINT album_images_position_positive CHECK (position >= 0)
);

CREATE INDEX idx_album_images_album_position ON album_images(album_id, position);
CREATE INDEX idx_album_images_image_id ON album_images(image_id);

COMMENT ON TABLE album_images IS 'Association between albums and images with ordering';
COMMENT ON COLUMN album_images.position IS 'Display order within the album';

-- Tags table - reusable image tags
CREATE TABLE tags (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    usage_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT tags_name_not_empty CHECK (name <> ''),
    CONSTRAINT tags_slug_not_empty CHECK (slug <> ''),
    CONSTRAINT tags_usage_count_positive CHECK (usage_count >= 0)
);

CREATE INDEX idx_tags_slug ON tags(slug);
CREATE INDEX idx_tags_usage_count ON tags(usage_count DESC);

COMMENT ON TABLE tags IS 'Reusable image tags for categorization';
COMMENT ON COLUMN tags.slug IS 'URL-safe lowercase version of name';
COMMENT ON COLUMN tags.usage_count IS 'Denormalized count of images using this tag';

-- Image-tag relationship (many-to-many)
CREATE TABLE image_tags (
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    tagged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (image_id, tag_id)
);

CREATE INDEX idx_image_tags_tag_id ON image_tags(tag_id);
CREATE INDEX idx_image_tags_image_id ON image_tags(image_id);

COMMENT ON TABLE image_tags IS 'Association between images and tags';

-- Trigger to update album.image_count
CREATE OR REPLACE FUNCTION update_album_image_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE albums SET image_count = image_count + 1, updated_at = NOW()
        WHERE id = NEW.album_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE albums SET image_count = image_count - 1, updated_at = NOW()
        WHERE id = OLD.album_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_album_image_count
    AFTER INSERT OR DELETE ON album_images
    FOR EACH ROW
    EXECUTE FUNCTION update_album_image_count();

-- Trigger to update tags.usage_count
CREATE OR REPLACE FUNCTION update_tag_usage_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE tags SET usage_count = usage_count + 1 WHERE id = NEW.tag_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE tags SET usage_count = usage_count - 1 WHERE id = OLD.tag_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_tag_usage_count
    AFTER INSERT OR DELETE ON image_tags
    FOR EACH ROW
    EXECUTE FUNCTION update_tag_usage_count();

-- Trigger to update images.updated_at on modification
CREATE OR REPLACE FUNCTION update_images_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_images_updated_at
    BEFORE UPDATE ON images
    FOR EACH ROW
    EXECUTE FUNCTION update_images_updated_at();

-- Trigger to update albums.updated_at on modification
CREATE TRIGGER trg_albums_updated_at
    BEFORE UPDATE ON albums
    FOR EACH ROW
    EXECUTE FUNCTION update_images_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS trg_albums_updated_at ON albums;
DROP TRIGGER IF EXISTS trg_images_updated_at ON images;
DROP TRIGGER IF EXISTS trg_tag_usage_count ON image_tags;
DROP TRIGGER IF EXISTS trg_album_image_count ON album_images;
DROP FUNCTION IF EXISTS update_images_updated_at();
DROP FUNCTION IF EXISTS update_tag_usage_count();
DROP FUNCTION IF EXISTS update_album_image_count();
DROP TABLE IF EXISTS image_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS album_images;
DROP TABLE IF EXISTS albums;
DROP TABLE IF EXISTS image_variants;
DROP TABLE IF EXISTS images;
