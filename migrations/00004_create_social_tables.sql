-- +goose Up
-- Create likes table for image likes
CREATE TABLE likes (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, image_id)
);

-- Indexes for likes
CREATE INDEX idx_likes_image_id ON likes(image_id);
CREATE INDEX idx_likes_user_id ON likes(user_id);
CREATE INDEX idx_likes_created_at ON likes(created_at);

-- Create comments table for image comments
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT comments_content_not_empty CHECK (LENGTH(TRIM(content)) > 0),
    CONSTRAINT comments_content_length CHECK (LENGTH(content) <= 5000)
);

-- Indexes for comments
CREATE INDEX idx_comments_image_id ON comments(image_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_user_id ON comments(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_created_at ON comments(created_at) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS likes;
