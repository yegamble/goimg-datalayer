-- +goose Up
-- Add IPFS storage support to images and variants

-- Add IPFS fields to images table
ALTER TABLE images
    ADD COLUMN ipfs_cid VARCHAR(100),
    ADD COLUMN ipfs_pinned BOOLEAN DEFAULT FALSE,
    ADD COLUMN ipfs_pinned_at TIMESTAMPTZ;

-- Update storage_provider constraint to include ipfs
ALTER TABLE images DROP CONSTRAINT IF EXISTS images_storage_provider_check;
-- Note: Using CHECK constraint comment for documentation since provider is flexible

-- Create index for IPFS lookups
CREATE INDEX idx_images_ipfs_cid ON images(ipfs_cid) WHERE ipfs_cid IS NOT NULL;
CREATE INDEX idx_images_ipfs_pinned ON images(ipfs_pinned) WHERE ipfs_pinned = TRUE;

-- Add IPFS fields to image_variants table
ALTER TABLE image_variants
    ADD COLUMN ipfs_cid VARCHAR(100),
    ADD COLUMN ipfs_pinned BOOLEAN DEFAULT FALSE;

-- Create index for variant IPFS lookups
CREATE INDEX idx_variants_ipfs_cid ON image_variants(ipfs_cid) WHERE ipfs_cid IS NOT NULL;

-- Add comments for IPFS columns
COMMENT ON COLUMN images.ipfs_cid IS 'IPFS Content Identifier (CID) for decentralized backup. Supports both CIDv0 (Qm...) and CIDv1 (bafy...)';
COMMENT ON COLUMN images.ipfs_pinned IS 'Whether the image is pinned to IPFS node (protected from garbage collection)';
COMMENT ON COLUMN images.ipfs_pinned_at IS 'Timestamp when image was pinned to IPFS';

COMMENT ON COLUMN image_variants.ipfs_cid IS 'IPFS Content Identifier (CID) for variant backup';
COMMENT ON COLUMN image_variants.ipfs_pinned IS 'Whether the variant is pinned to IPFS node';

-- +goose Down
-- Remove IPFS fields from image_variants
DROP INDEX IF EXISTS idx_variants_ipfs_cid;
ALTER TABLE image_variants
    DROP COLUMN IF EXISTS ipfs_cid,
    DROP COLUMN IF EXISTS ipfs_pinned;

-- Remove IPFS fields from images
DROP INDEX IF EXISTS idx_images_ipfs_pinned;
DROP INDEX IF EXISTS idx_images_ipfs_cid;
ALTER TABLE images
    DROP COLUMN IF EXISTS ipfs_cid,
    DROP COLUMN IF EXISTS ipfs_pinned,
    DROP COLUMN IF EXISTS ipfs_pinned_at;
