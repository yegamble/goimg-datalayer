-- +goose Up
CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_refresh_token_hash ON sessions(refresh_token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at) WHERE revoked_at IS NULL;

COMMENT ON TABLE sessions IS 'User authentication sessions with refresh tokens';
COMMENT ON COLUMN sessions.id IS 'Unique session identifier (UUID)';
COMMENT ON COLUMN sessions.user_id IS 'Reference to user who owns this session';
COMMENT ON COLUMN sessions.refresh_token_hash IS 'Hashed refresh token for session renewal';
COMMENT ON COLUMN sessions.ip_address IS 'Client IP address when session was created';
COMMENT ON COLUMN sessions.user_agent IS 'Client user agent string';
COMMENT ON COLUMN sessions.expires_at IS 'Timestamp when session expires';
COMMENT ON COLUMN sessions.revoked_at IS 'Timestamp when session was revoked (NULL if active)';

-- +goose Down
DROP TABLE IF EXISTS sessions;
