-- ============================================================================
-- Backup/Restore Validation Seed Data Script
-- ============================================================================
-- Creates comprehensive test data for validating backup and restore operations
--
-- Data volume:
--   - 100 users (diverse roles and statuses)
--   - 200 sessions
--   - 500 images (with variants and metadata)
--   - 2000 image variants (thumbnail, small, medium, large)
--   - 50 albums (with cover images)
--   - 300 album-image associations
--   - 100 tags
--   - 800 image-tag associations
--   - 2000 likes
--   - 1000 comments
--
-- Features:
--   - Realistic UUIDs using gen_random_uuid()
--   - Edge cases: unicode content, null values, deleted records
--   - Foreign key relationships across all tables
--   - Triggers and constraints validation
-- ============================================================================

-- Set client encoding for unicode support
SET client_encoding = 'UTF8';

-- Disable triggers temporarily for faster insert
SET session_replication_role = replica;

-- ============================================================================
-- 1. Create 100 Users
-- ============================================================================
INSERT INTO users (id, email, username, password_hash, role, status, display_name, bio, created_at, updated_at, deleted_at)
SELECT
    gen_random_uuid(),
    'user' || i || '@testdomain' || (i % 5) || '.com',
    'testuser' || i,
    '$argon2id$v=19$m=65536,t=3,p=2$' || encode(gen_random_bytes(16), 'base64'),  -- Fake argon2 hash
    CASE (i % 10)
        WHEN 0 THEN 'admin'
        WHEN 1 THEN 'moderator'
        ELSE 'user'
    END,
    CASE (i % 20)
        WHEN 0 THEN 'pending'
        WHEN 1 THEN 'suspended'
        WHEN 19 THEN 'deleted'
        ELSE 'active'
    END,
    CASE (i % 5)
        WHEN 0 THEN 'Test User ' || i
        WHEN 1 THEN 'Tëst Üsér ' || i  -- Unicode characters
        WHEN 2 THEN '测试用户 ' || i     -- Chinese characters
        WHEN 3 THEN ''                  -- Empty display name
        ELSE 'User #' || i
    END,
    CASE (i % 4)
        WHEN 0 THEN 'Photography enthusiast and nature lover. 📸'
        WHEN 1 THEN ''  -- Empty bio
        WHEN 2 THEN repeat('Bio content for user ' || i || '. ', 10)  -- Long bio
        ELSE 'Test bio for user ' || i
    END,
    NOW() - (i || ' days')::interval,
    NOW() - (i / 2 || ' days')::interval,
    CASE WHEN (i % 20) = 19 THEN NOW() - (i / 3 || ' days')::interval ELSE NULL END
FROM generate_series(1, 100) AS i;

-- ============================================================================
-- 2. Create 200 Sessions (for active users)
-- ============================================================================
INSERT INTO sessions (id, user_id, refresh_token_hash, ip_address, user_agent, expires_at, created_at, revoked_at)
SELECT
    gen_random_uuid(),
    u.id,
    encode(sha256(gen_random_bytes(32)), 'hex'),
    ('192.168.' || (row_number() OVER () % 255) || '.' || (row_number() OVER () / 255 % 255))::inet,
    CASE (row_number() OVER () % 5)
        WHEN 0 THEN 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'
        WHEN 1 THEN 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36'
        WHEN 2 THEN 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36'
        WHEN 3 THEN 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)'
        ELSE 'curl/7.64.1'
    END,
    NOW() + ((row_number() OVER () % 30) || ' days')::interval,
    NOW() - ((row_number() OVER () % 10) || ' hours')::interval,
    CASE WHEN (row_number() OVER () % 10) = 0 THEN NOW() - (row_number() OVER () % 5 || ' hours')::interval ELSE NULL END
FROM (
    SELECT id FROM users
    WHERE status = 'active' AND deleted_at IS NULL
    ORDER BY created_at
    LIMIT 50
) u
CROSS JOIN generate_series(1, 4) s;

-- ============================================================================
-- 3. Create 100 Tags (for later use)
-- ============================================================================
INSERT INTO tags (id, name, slug, usage_count, created_at)
SELECT
    gen_random_uuid(),
    tag_name,
    lower(regexp_replace(tag_name, '[^a-zA-Z0-9]+', '-', 'g')),
    0,  -- Will be updated by trigger
    NOW() - (row_number() OVER () || ' days')::interval
FROM (VALUES
    ('landscape'), ('portrait'), ('nature'), ('urban'), ('wildlife'),
    ('abstract'), ('sunset'), ('beach'), ('mountain'), ('forest'),
    ('architecture'), ('street'), ('night'), ('macro'), ('black-and-white'),
    ('colorful'), ('minimalist'), ('vintage'), ('modern'), ('travel'),
    ('food'), ('people'), ('animals'), ('flowers'), ('water'),
    ('sky'), ('clouds'), ('city'), ('countryside'), ('indoor'),
    ('outdoor'), ('summer'), ('winter'), ('spring'), ('autumn'),
    ('sunrise'), ('golden-hour'), ('blue-hour'), ('long-exposure'), ('hdr'),
    ('panorama'), ('aerial'), ('underwater'), ('sports'), ('action'),
    ('candid'), ('posed'), ('editorial'), ('commercial'), ('fine-art'),
    ('documentary'), ('artistic'), ('creative'), ('professional'), ('amateur'),
    ('digital'), ('film'), ('analog'), ('mobile'), ('drone'),
    ('canon'), ('nikon'), ('sony'), ('fujifilm'), ('olympus'),
    ('raw'), ('edited'), ('filtered'), ('unedited'), ('lightroom'),
    ('photoshop'), ('instagram'), ('pinterest'), ('flickr'), ('unsplash'),
    ('365project'), ('photooftheday'), ('photography'), ('photographer'), ('photo'),
    ('pic'), ('picture'), ('image'), ('snapshot'), ('capture'),
    ('moment'), ('memory'), ('beautiful'), ('amazing'), ('stunning'),
    ('gorgeous'), ('lovely'), ('pretty'), ('nice'), ('cool'),
    ('awesome'), ('incredible'), ('spectacular'), ('breathtaking'), ('magnificent')
) AS tag_names(tag_name);

-- ============================================================================
-- 4. Create 500 Images (owned by active users)
-- ============================================================================
INSERT INTO images (
    id, owner_id, title, description, storage_provider, storage_key,
    original_filename, mime_type, file_size, width, height,
    status, visibility, scan_status, view_count,
    created_at, updated_at, deleted_at
)
SELECT
    gen_random_uuid(),
    (SELECT id FROM users WHERE status = 'active' AND deleted_at IS NULL ORDER BY random() LIMIT 1),
    CASE (i % 10)
        WHEN 0 THEN NULL  -- No title
        WHEN 1 THEN 'Beautiful Sunset 🌅 ' || i  -- Unicode emoji
        WHEN 2 THEN 'Фото ' || i  -- Cyrillic
        WHEN 3 THEN repeat('Long title ', 20) || i  -- Long title
        ELSE 'Test Image ' || i
    END,
    CASE (i % 5)
        WHEN 0 THEN NULL  -- No description
        WHEN 1 THEN ''  -- Empty description
        WHEN 2 THEN 'A beautiful image captured at the perfect moment. This photo represents...' || repeat(' amazing ', 50)  -- Long description
        ELSE 'Test description for image ' || i
    END,
    CASE (i % 4)
        WHEN 0 THEN 'local'
        WHEN 1 THEN 's3'
        WHEN 2 THEN 'spaces'
        ELSE 'b2'
    END,
    'images/' || to_char(NOW() - (i || ' days')::interval, 'YYYY/MM/DD') || '/' || encode(gen_random_bytes(16), 'hex') || '.jpg',
    'IMG_' || LPAD(i::text, 4, '0') || '.jpg',
    CASE (i % 5)
        WHEN 0 THEN 'image/jpeg'
        WHEN 1 THEN 'image/png'
        WHEN 2 THEN 'image/gif'
        WHEN 3 THEN 'image/webp'
        ELSE 'image/jpeg'
    END,
    1048576 + (i * 1024)::bigint,  -- File sizes from ~1MB to ~500MB
    800 + (i % 20) * 100,  -- Widths from 800 to 2700
    600 + (i % 15) * 100,  -- Heights from 600 to 2100
    CASE (i % 20)
        WHEN 0 THEN 'processing'
        WHEN 1 THEN 'failed'
        WHEN 19 THEN 'deleted'
        ELSE 'active'
    END,
    CASE (i % 3)
        WHEN 0 THEN 'public'
        WHEN 1 THEN 'private'
        ELSE 'unlisted'
    END,
    CASE (i % 25)
        WHEN 0 THEN 'pending'
        WHEN 24 THEN 'infected'  -- Edge case: infected image
        WHEN 23 THEN 'error'
        ELSE 'clean'
    END,
    (i * 7) % 10000,  -- View counts from 0 to 9999
    NOW() - (i || ' days')::interval,
    NOW() - (i / 2 || ' days')::interval,
    CASE WHEN (i % 20) = 19 THEN NOW() - (i / 3 || ' days')::interval ELSE NULL END
FROM generate_series(1, 500) AS i;

-- ============================================================================
-- 5. Create 2000 Image Variants (4 variants per active image)
-- ============================================================================
INSERT INTO image_variants (
    id, image_id, variant_type, storage_key, width, height, file_size, format, created_at
)
SELECT
    gen_random_uuid(),
    img.id,
    variant_type,
    regexp_replace(img.storage_key, '\.[^.]+$', '') || '_' || variant_type || '.' || format,
    CASE variant_type
        WHEN 'thumbnail' THEN 160
        WHEN 'small' THEN 320
        WHEN 'medium' THEN 800
        WHEN 'large' THEN 1600
        ELSE img.width
    END,
    CASE variant_type
        WHEN 'thumbnail' THEN (160.0 / img.width * img.height)::integer
        WHEN 'small' THEN (320.0 / img.width * img.height)::integer
        WHEN 'medium' THEN (800.0 / img.width * img.height)::integer
        WHEN 'large' THEN (1600.0 / img.width * img.height)::integer
        ELSE img.height
    END,
    CASE variant_type
        WHEN 'thumbnail' THEN 10240
        WHEN 'small' THEN 51200
        WHEN 'medium' THEN 204800
        WHEN 'large' THEN 819200
        ELSE img.file_size
    END,
    format,
    img.created_at + '5 minutes'::interval
FROM images img
CROSS JOIN (VALUES ('thumbnail'), ('small'), ('medium'), ('large')) AS v(variant_type)
CROSS JOIN (VALUES ('jpeg')) AS f(format)
WHERE img.status = 'active' AND img.deleted_at IS NULL;

-- ============================================================================
-- 6. Create 50 Albums
-- ============================================================================
INSERT INTO albums (
    id, owner_id, title, description, visibility, cover_image_id, image_count,
    created_at, updated_at, deleted_at
)
SELECT
    gen_random_uuid(),
    (SELECT id FROM users WHERE status = 'active' AND deleted_at IS NULL ORDER BY random() LIMIT 1),
    CASE (i % 8)
        WHEN 0 THEN 'Vacation 2024 🏖️'  -- Unicode emoji
        WHEN 1 THEN 'Портфолио ' || i  -- Cyrillic
        WHEN 2 THEN repeat('Album Title ', 10) || i  -- Long title
        ELSE 'Test Album ' || i
    END,
    CASE (i % 4)
        WHEN 0 THEN NULL
        WHEN 1 THEN ''
        ELSE 'Description for album ' || i || '. This album contains my best photos from...'
    END,
    CASE (i % 3)
        WHEN 0 THEN 'public'
        WHEN 1 THEN 'private'
        ELSE 'unlisted'
    END,
    NULL,  -- Will be updated after album_images insertion
    0,  -- Will be updated by trigger
    NOW() - (i * 3 || ' days')::interval,
    NOW() - (i || ' days')::interval,
    CASE WHEN (i % 25) = 0 THEN NOW() - (i / 2 || ' days')::interval ELSE NULL END
FROM generate_series(1, 50) AS i;

-- ============================================================================
-- 7. Create Album-Image Associations (300 associations, ~6 images per album)
-- ============================================================================
WITH active_albums AS (
    SELECT id FROM albums WHERE deleted_at IS NULL
),
active_images AS (
    SELECT id FROM images WHERE status = 'active' AND deleted_at IS NULL
),
numbered_albums AS (
    SELECT id, row_number() OVER () as album_num FROM active_albums
),
numbered_images AS (
    SELECT id, row_number() OVER () as image_num FROM active_images
)
INSERT INTO album_images (album_id, image_id, position, added_at)
SELECT
    na.id,
    ni.id,
    (ni.image_num % 20)::integer,  -- Position within album
    NOW() - ((ni.image_num % 100) || ' hours')::interval
FROM numbered_albums na
CROSS JOIN LATERAL (
    SELECT id, image_num
    FROM numbered_images
    WHERE image_num % (SELECT COUNT(*) FROM numbered_albums) = (na.album_num - 1) % 20
    LIMIT 8
) ni;

-- Update cover_image_id for albums (first image in each album)
WITH album_covers AS (
    SELECT DISTINCT ON (ai.album_id)
        ai.album_id,
        ai.image_id
    FROM album_images ai
    ORDER BY ai.album_id, ai.position
)
UPDATE albums a
SET cover_image_id = ac.image_id
FROM album_covers ac
WHERE a.id = ac.album_id;

-- ============================================================================
-- 8. Create Image-Tag Associations (800 associations, ~1-5 tags per image)
-- ============================================================================
WITH active_images AS (
    SELECT id FROM images WHERE status = 'active' AND deleted_at IS NULL
),
numbered_images AS (
    SELECT id, row_number() OVER () as image_num FROM active_images
)
INSERT INTO image_tags (image_id, tag_id, tagged_at)
SELECT DISTINCT
    ni.id,
    t.id,
    NOW() - ((ni.image_num % 100) || ' hours')::interval
FROM numbered_images ni
CROSS JOIN LATERAL (
    SELECT id
    FROM tags
    ORDER BY random()
    LIMIT CASE (ni.image_num % 5)
        WHEN 0 THEN 1
        WHEN 1 THEN 2
        WHEN 2 THEN 3
        WHEN 3 THEN 4
        ELSE 5
    END
) t;

-- ============================================================================
-- 9. Create 2000 Likes
-- ============================================================================
WITH active_users AS (
    SELECT id FROM users WHERE status = 'active' AND deleted_at IS NULL
),
active_images AS (
    SELECT id FROM images WHERE status = 'active' AND deleted_at IS NULL
),
numbered_users AS (
    SELECT id, row_number() OVER () as user_num FROM active_users
),
numbered_images AS (
    SELECT id, row_number() OVER () as image_num FROM active_images
)
INSERT INTO likes (user_id, image_id, created_at)
SELECT DISTINCT
    nu.id,
    ni.id,
    NOW() - ((row_number() OVER () % 1000) || ' hours')::interval
FROM numbered_users nu
CROSS JOIN LATERAL (
    SELECT id, image_num
    FROM numbered_images
    ORDER BY random()
    LIMIT 25  -- Each user likes ~25 images
) ni
LIMIT 2000;

-- ============================================================================
-- 10. Create 1000 Comments
-- ============================================================================
WITH active_users AS (
    SELECT id FROM users WHERE status = 'active' AND deleted_at IS NULL
),
active_images AS (
    SELECT id FROM images WHERE status = 'active' AND deleted_at IS NULL
),
numbered_users AS (
    SELECT id, row_number() OVER () as user_num FROM active_users
)
INSERT INTO comments (
    id, user_id, image_id, content, created_at, updated_at, deleted_at
)
SELECT
    gen_random_uuid(),
    nu.id,
    (SELECT id FROM active_images ORDER BY random() LIMIT 1),
    CASE (nu.user_num % 10)
        WHEN 0 THEN 'Great shot! 👍'  -- Unicode emoji
        WHEN 1 THEN 'Прекрасное фото!'  -- Cyrillic
        WHEN 2 THEN '很棒的照片！'  -- Chinese
        WHEN 3 THEN repeat('Amazing image! ', 50)  -- Long comment
        WHEN 4 THEN 'Nice.'  -- Short comment
        WHEN 5 THEN 'Beautiful composition and lighting. The colors are stunning and the framing is perfect. This really captures the essence of the moment.'
        WHEN 6 THEN ''  -- Edge case handled by constraint
        ELSE 'Comment #' || row_number() OVER () || ' from user ' || nu.user_num
    END,
    NOW() - ((row_number() OVER () % 500) || ' hours')::interval,
    NOW() - ((row_number() OVER () % 400) || ' hours')::interval,
    CASE WHEN (row_number() OVER () % 50) = 0 THEN NOW() - ((row_number() OVER () % 100) || ' hours')::interval ELSE NULL END
FROM numbered_users nu
CROSS JOIN generate_series(1, 13) s  -- ~13 comments per user
LIMIT 1000;

-- Re-enable triggers
SET session_replication_role = DEFAULT;

-- ============================================================================
-- Verification Queries
-- ============================================================================
DO $$
DECLARE
    v_users_count INTEGER;
    v_sessions_count INTEGER;
    v_images_count INTEGER;
    v_variants_count INTEGER;
    v_albums_count INTEGER;
    v_album_images_count INTEGER;
    v_tags_count INTEGER;
    v_image_tags_count INTEGER;
    v_likes_count INTEGER;
    v_comments_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_users_count FROM users;
    SELECT COUNT(*) INTO v_sessions_count FROM sessions;
    SELECT COUNT(*) INTO v_images_count FROM images;
    SELECT COUNT(*) INTO v_variants_count FROM image_variants;
    SELECT COUNT(*) INTO v_albums_count FROM albums;
    SELECT COUNT(*) INTO v_album_images_count FROM album_images;
    SELECT COUNT(*) INTO v_tags_count FROM tags;
    SELECT COUNT(*) INTO v_image_tags_count FROM image_tags;
    SELECT COUNT(*) INTO v_likes_count FROM likes;
    SELECT COUNT(*) INTO v_comments_count FROM comments;

    RAISE NOTICE '================================================';
    RAISE NOTICE 'Seed Data Verification';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Users:          %', v_users_count;
    RAISE NOTICE 'Sessions:       %', v_sessions_count;
    RAISE NOTICE 'Images:         %', v_images_count;
    RAISE NOTICE 'Image Variants: %', v_variants_count;
    RAISE NOTICE 'Albums:         %', v_albums_count;
    RAISE NOTICE 'Album-Images:   %', v_album_images_count;
    RAISE NOTICE 'Tags:           %', v_tags_count;
    RAISE NOTICE 'Image-Tags:     %', v_image_tags_count;
    RAISE NOTICE 'Likes:          %', v_likes_count;
    RAISE NOTICE 'Comments:       %', v_comments_count;
    RAISE NOTICE '================================================';
END $$;
