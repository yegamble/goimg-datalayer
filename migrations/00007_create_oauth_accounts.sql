-- +goose Up
-- +goose StatementBegin

-- Table: oauth_accounts
-- Links local user accounts to OAuth provider accounts (Google, GitHub)
CREATE TABLE oauth_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(20) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    avatar_url VARCHAR(512),
    access_token_encrypted BYTEA,
    refresh_token_encrypted BYTEA,
    token_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint: one provider account can only be linked to one user
    CONSTRAINT uq_oauth_provider_user UNIQUE(provider, provider_user_id),

    -- Check constraint: provider must be a valid value
    CONSTRAINT chk_oauth_provider CHECK (provider IN ('google', 'github'))
);

-- Index: Fast lookups by user ID (for "show all linked accounts")
CREATE INDEX idx_oauth_accounts_user_id ON oauth_accounts(user_id);

-- Index: Fast lookups by provider and email (for account linking suggestions)
CREATE INDEX idx_oauth_accounts_provider_email ON oauth_accounts(provider, email);

-- Index: Fast lookups by user and provider (for "is this provider linked?")
CREATE INDEX idx_oauth_accounts_user_provider ON oauth_accounts(user_id, provider);

COMMENT ON TABLE oauth_accounts IS 'OAuth 2.0 provider accounts linked to local users';
COMMENT ON COLUMN oauth_accounts.id IS 'Unique OAuth account identifier (UUID)';
COMMENT ON COLUMN oauth_accounts.user_id IS 'Local user this OAuth account is linked to';
COMMENT ON COLUMN oauth_accounts.provider IS 'OAuth provider name (google, github)';
COMMENT ON COLUMN oauth_accounts.provider_user_id IS 'User ID from provider (stable identifier, e.g., Google sub claim)';
COMMENT ON COLUMN oauth_accounts.email IS 'Email address from OAuth provider (for convenience, not primary key)';
COMMENT ON COLUMN oauth_accounts.display_name IS 'Display name from OAuth provider';
COMMENT ON COLUMN oauth_accounts.avatar_url IS 'Profile picture URL from OAuth provider';
COMMENT ON COLUMN oauth_accounts.access_token_encrypted IS 'AES-256-GCM encrypted OAuth access token (optional, for API access)';
COMMENT ON COLUMN oauth_accounts.refresh_token_encrypted IS 'AES-256-GCM encrypted OAuth refresh token (optional, for token refresh)';
COMMENT ON COLUMN oauth_accounts.token_expires_at IS 'When the access token expires (NULL if not stored)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS oauth_accounts;
-- +goose StatementEnd
