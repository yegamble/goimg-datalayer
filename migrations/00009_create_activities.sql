-- +goose Up
CREATE TABLE activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id UUID NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for querying activities by actor (user's own activity history)
CREATE INDEX idx_activities_actor_created ON activities(actor_id, created_at DESC);

-- Index for querying activities by created_at (for cleanup and general queries)
CREATE INDEX idx_activities_created_at ON activities(created_at DESC);

-- Index for feed queries (join with user_follows to get activities from followed users)
-- This is critical for performance of the main feed query
CREATE INDEX idx_activities_actor_id ON activities(actor_id);

COMMENT ON TABLE activities IS 'User activities for the Activity feed bounded context';
COMMENT ON COLUMN activities.id IS 'Unique identifier for the activity';
COMMENT ON COLUMN activities.actor_id IS 'ID of the user who performed the activity';
COMMENT ON COLUMN activities.activity_type IS 'Type of activity (image_uploaded, image_liked, etc.)';
COMMENT ON COLUMN activities.target_type IS 'Type of the target entity (image, user, album, comment)';
COMMENT ON COLUMN activities.target_id IS 'ID of the target entity';
COMMENT ON COLUMN activities.metadata IS 'Additional contextual information in JSON format';
COMMENT ON COLUMN activities.created_at IS 'Timestamp when the activity occurred';

-- +goose Down
DROP TABLE IF EXISTS activities;
