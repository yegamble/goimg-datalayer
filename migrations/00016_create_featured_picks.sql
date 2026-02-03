-- +goose Up
-- +goose StatementBegin
-- Featured picks table - admin-curated featured images with scheduling (Sprint 19)

CREATE TABLE featured_picks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    featured_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT NOT NULL DEFAULT '',
    display_order INTEGER NOT NULL DEFAULT 0,
    featured_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    featured_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT featured_picks_reason_length CHECK (LENGTH(reason) <= 500),
    CONSTRAINT featured_picks_display_order_positive CHECK (display_order >= 0),
    CONSTRAINT featured_picks_date_range_valid CHECK (
        featured_until IS NULL OR featured_until > featured_from
    ),
    -- Prevent duplicate active features for the same image
    CONSTRAINT featured_picks_unique_active UNIQUE (image_id)
);

-- Indexes for efficient querying
CREATE INDEX idx_featured_picks_image_id ON featured_picks(image_id);
CREATE INDEX idx_featured_picks_featured_by ON featured_picks(featured_by);
CREATE INDEX idx_featured_picks_display_order ON featured_picks(display_order);
CREATE INDEX idx_featured_picks_date_range ON featured_picks(featured_from, featured_until);

-- Active featured picks (most common query) - filter by featured_until at query time
CREATE INDEX idx_featured_picks_active ON featured_picks(display_order, featured_until, featured_from DESC);

COMMENT ON TABLE featured_picks IS 'Admin-curated featured images with scheduling and display order';
COMMENT ON COLUMN featured_picks.image_id IS 'Image being featured';
COMMENT ON COLUMN featured_picks.featured_by IS 'Admin user who featured this image';
COMMENT ON COLUMN featured_picks.reason IS 'Optional reason/notes for featuring (max 500 chars)';
COMMENT ON COLUMN featured_picks.display_order IS 'Display priority (lower = higher priority, 0 = top)';
COMMENT ON COLUMN featured_picks.featured_from IS 'Start date/time for featuring';
COMMENT ON COLUMN featured_picks.featured_until IS 'End date/time for featuring (NULL = no expiration)';

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_featured_picks_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_featured_picks_updated_at
    BEFORE UPDATE ON featured_picks
    FOR EACH ROW
    EXECUTE FUNCTION update_featured_picks_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_featured_picks_updated_at ON featured_picks;
DROP FUNCTION IF EXISTS update_featured_picks_updated_at();
DROP TABLE IF EXISTS featured_picks;
-- +goose StatementEnd
