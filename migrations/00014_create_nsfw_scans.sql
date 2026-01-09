-- +goose Up
-- +goose StatementBegin

-- NSFW Scans table for AI-powered content moderation
-- Stores results from NSFW detection API providers (SightEngine, ModerateContent)
CREATE TABLE IF NOT EXISTS nsfw_scans (
    id UUID PRIMARY KEY,
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('sightengine', 'moderatecontent')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'scanning', 'completed', 'failed')),
    category VARCHAR(30) NOT NULL DEFAULT 'unknown' CHECK (category IN ('safe', 'suggestive', 'nudity', 'explicit', 'violence', 'unknown')),
    score DECIMAL(4,3) NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 1),

    -- Detailed scores from the provider
    nudity_score DECIMAL(4,3) NOT NULL DEFAULT 0 CHECK (nudity_score >= 0 AND nudity_score <= 1),
    weapon_score DECIMAL(4,3) NOT NULL DEFAULT 0 CHECK (weapon_score >= 0 AND weapon_score <= 1),
    violence_score DECIMAL(4,3) NOT NULL DEFAULT 0 CHECK (violence_score >= 0 AND violence_score <= 1),
    offensive_score DECIMAL(4,3) NOT NULL DEFAULT 0 CHECK (offensive_score >= 0 AND offensive_score <= 1),
    drug_score DECIMAL(4,3) NOT NULL DEFAULT 0 CHECK (drug_score >= 0 AND drug_score <= 1),

    -- Additional metadata
    sub_categories TEXT[] DEFAULT '{}',
    error_message TEXT,

    -- Timestamps
    scanned_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for finding scans by image (most common query)
CREATE INDEX idx_nsfw_scans_image_id ON nsfw_scans(image_id);

-- Index for finding scans by status (for processing queue)
CREATE INDEX idx_nsfw_scans_status ON nsfw_scans(status) WHERE status IN ('pending', 'scanning');

-- Index for finding NSFW content (for moderation dashboard)
CREATE INDEX idx_nsfw_scans_nsfw_content ON nsfw_scans(category)
WHERE category IN ('nudity', 'explicit', 'violence');

-- Index for finding content requiring review
CREATE INDEX idx_nsfw_scans_requires_review ON nsfw_scans(category)
WHERE category IN ('suggestive', 'nudity', 'explicit', 'violence', 'unknown');

-- Index for finding recent scans by provider
CREATE INDEX idx_nsfw_scans_provider_created ON nsfw_scans(provider, created_at DESC);

-- Composite index for finding latest scan per image
CREATE INDEX idx_nsfw_scans_image_created ON nsfw_scans(image_id, created_at DESC);

-- Add trigger to update updated_at on modification
CREATE OR REPLACE FUNCTION update_nsfw_scans_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER nsfw_scans_updated_at_trigger
    BEFORE UPDATE ON nsfw_scans
    FOR EACH ROW
    EXECUTE FUNCTION update_nsfw_scans_updated_at();

-- Add column to images table to track NSFW status (denormalized for performance)
ALTER TABLE images
ADD COLUMN IF NOT EXISTS nsfw_status VARCHAR(20) DEFAULT 'pending'
CHECK (nsfw_status IN ('pending', 'safe', 'flagged', 'blocked'));

-- Index for filtering images by NSFW status
CREATE INDEX IF NOT EXISTS idx_images_nsfw_status ON images(nsfw_status);

COMMENT ON TABLE nsfw_scans IS 'Stores AI-powered NSFW content scan results for images';
COMMENT ON COLUMN nsfw_scans.provider IS 'The NSFW detection API provider used';
COMMENT ON COLUMN nsfw_scans.status IS 'Current status of the scan (pending, scanning, completed, failed)';
COMMENT ON COLUMN nsfw_scans.category IS 'Detected NSFW category (safe, suggestive, nudity, explicit, violence)';
COMMENT ON COLUMN nsfw_scans.score IS 'Overall NSFW confidence score (0.0 to 1.0)';
COMMENT ON COLUMN nsfw_scans.sub_categories IS 'Additional subcategories detected by the provider';
COMMENT ON COLUMN images.nsfw_status IS 'Denormalized NSFW status for fast filtering';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove NSFW status column from images
ALTER TABLE images DROP COLUMN IF EXISTS nsfw_status;

-- Drop trigger and function
DROP TRIGGER IF EXISTS nsfw_scans_updated_at_trigger ON nsfw_scans;
DROP FUNCTION IF EXISTS update_nsfw_scans_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_nsfw_scans_image_created;
DROP INDEX IF EXISTS idx_nsfw_scans_provider_created;
DROP INDEX IF EXISTS idx_nsfw_scans_requires_review;
DROP INDEX IF EXISTS idx_nsfw_scans_nsfw_content;
DROP INDEX IF EXISTS idx_nsfw_scans_status;
DROP INDEX IF EXISTS idx_nsfw_scans_image_id;
DROP INDEX IF EXISTS idx_images_nsfw_status;

-- Drop table
DROP TABLE IF EXISTS nsfw_scans;

-- +goose StatementEnd
