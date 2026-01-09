-- +goose Up
-- Sprint 17: Nested Albums + Custom Variants

-- Add parent_id to albums for nested album support
ALTER TABLE albums ADD COLUMN parent_id UUID REFERENCES albums(id) ON DELETE SET NULL;

-- Index for finding children of an album
CREATE INDEX idx_albums_parent_id ON albums(parent_id) WHERE deleted_at IS NULL;

-- Prevent circular references via trigger
CREATE OR REPLACE FUNCTION check_album_hierarchy_depth()
RETURNS TRIGGER AS $$
DECLARE
    current_id UUID;
    depth INT := 0;
    max_depth INT := 10; -- Maximum nesting depth
BEGIN
    IF NEW.parent_id IS NULL THEN
        RETURN NEW;
    END IF;

    -- Check for circular reference
    current_id := NEW.parent_id;
    WHILE current_id IS NOT NULL LOOP
        IF current_id = NEW.id THEN
            RAISE EXCEPTION 'Circular reference detected in album hierarchy';
        END IF;

        depth := depth + 1;
        IF depth > max_depth THEN
            RAISE EXCEPTION 'Album nesting depth exceeds maximum of %', max_depth;
        END IF;

        SELECT parent_id INTO current_id FROM albums WHERE id = current_id;
    END LOOP;

    -- Verify parent belongs to same owner
    IF NOT EXISTS (
        SELECT 1 FROM albums
        WHERE id = NEW.parent_id
        AND owner_id = NEW.owner_id
        AND deleted_at IS NULL
    ) THEN
        RAISE EXCEPTION 'Parent album must belong to the same owner and exist';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_check_album_hierarchy
    BEFORE INSERT OR UPDATE OF parent_id ON albums
    FOR EACH ROW
    EXECUTE FUNCTION check_album_hierarchy_depth();

-- Variant configurations table for custom user-defined variants
CREATE TABLE variant_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    max_width INTEGER NOT NULL CHECK (max_width > 0 AND max_width <= 8192),
    max_height INTEGER NOT NULL CHECK (max_height > 0 AND max_height <= 8192),
    format VARCHAR(10) NOT NULL DEFAULT 'jpeg' CHECK (format IN ('jpeg', 'png', 'webp', 'avif')),
    quality INTEGER NOT NULL DEFAULT 85 CHECK (quality >= 1 AND quality <= 100),
    crop_mode VARCHAR(20) NOT NULL DEFAULT 'fit' CHECK (crop_mode IN ('fit', 'fill', 'crop')),
    is_preset BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Each user can have one config with a given name
    CONSTRAINT variant_configs_user_name_unique UNIQUE(user_id, name)
);

-- Index for user's variant configs
CREATE INDEX idx_variant_configs_user_id ON variant_configs(user_id);

-- Index for global presets
CREATE INDEX idx_variant_configs_presets ON variant_configs(is_preset) WHERE is_preset = TRUE;

-- Auto-update updated_at
CREATE TRIGGER trg_variant_configs_updated_at
    BEFORE UPDATE ON variant_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert default system presets (user_id is NULL for system presets)
-- We'll use a special system user ID for presets
-- For now, we skip system presets and let them be created per-user

-- Custom image variants table (stores generated custom variants)
CREATE TABLE custom_image_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    config_id UUID NOT NULL REFERENCES variant_configs(id) ON DELETE CASCADE,
    storage_key VARCHAR(512) NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    file_size BIGINT NOT NULL,
    format VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Each image can have one variant per config
    CONSTRAINT custom_variants_image_config_unique UNIQUE(image_id, config_id)
);

-- Index for finding all custom variants of an image
CREATE INDEX idx_custom_variants_image_id ON custom_image_variants(image_id);

-- Index for finding all variants using a config
CREATE INDEX idx_custom_variants_config_id ON custom_image_variants(config_id);

-- +goose Down
-- Remove in reverse order

DROP TABLE IF EXISTS custom_image_variants;
DROP TABLE IF EXISTS variant_configs;

DROP TRIGGER IF EXISTS trg_check_album_hierarchy ON albums;
DROP FUNCTION IF EXISTS check_album_hierarchy_depth();

DROP INDEX IF EXISTS idx_albums_parent_id;
ALTER TABLE albums DROP COLUMN IF EXISTS parent_id;
