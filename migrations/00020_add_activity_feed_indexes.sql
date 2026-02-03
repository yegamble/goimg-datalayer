-- +goose Up
-- +goose StatementBegin

-- Add indexes to optimize activity feed queries
-- These indexes address the full table scan issue identified in the audit

-- Optimize follower lookup for feed generation
CREATE INDEX IF NOT EXISTS idx_user_follows_follower ON user_follows(follower_id);

-- Optimize activity lookup by actor (sorted by creation time)
-- This is crucial for the LATERAL JOIN performance
CREATE INDEX IF NOT EXISTS idx_activities_actor_created ON activities(actor_id, created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_activities_actor_created;
DROP INDEX IF EXISTS idx_user_follows_follower;

-- +goose StatementEnd
