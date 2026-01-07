-- +goose Up
CREATE TABLE user_follows (
    follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    followed_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, followed_id),
    CHECK (follower_id != followed_id)
);

CREATE INDEX idx_user_follows_follower ON user_follows(follower_id);
CREATE INDEX idx_user_follows_followed ON user_follows(followed_id);
CREATE INDEX idx_user_follows_created_at ON user_follows(created_at DESC);

COMMENT ON TABLE user_follows IS 'User follow relationships for the Identity bounded context';
COMMENT ON COLUMN user_follows.follower_id IS 'ID of the user who is following';
COMMENT ON COLUMN user_follows.followed_id IS 'ID of the user being followed';
COMMENT ON COLUMN user_follows.created_at IS 'Timestamp when the follow relationship was created';

-- +goose Down
DROP TABLE IF EXISTS user_follows;
