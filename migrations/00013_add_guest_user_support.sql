-- +goose Up
-- Add guest user support to the users table

-- Add user_type column (registered or guest)
ALTER TABLE users ADD COLUMN user_type VARCHAR(20) NOT NULL DEFAULT 'registered';

-- Add ip_address column for guest tracking
ALTER TABLE users ADD COLUMN ip_address INET;

-- Add expires_at column for guest account auto-cleanup
ALTER TABLE users ADD COLUMN expires_at TIMESTAMPTZ;

-- Add constraint to ensure user_type is valid
ALTER TABLE users ADD CONSTRAINT users_user_type_check CHECK (user_type IN ('registered', 'guest'));

-- Add constraint to ensure guest users have required fields
ALTER TABLE users ADD CONSTRAINT users_guest_ip_required CHECK (
    (user_type = 'guest' AND ip_address IS NOT NULL AND expires_at IS NOT NULL) OR
    (user_type = 'registered')
);

-- Add constraint to ensure expires_at is in the future for guest users
ALTER TABLE users ADD CONSTRAINT users_guest_expires_future CHECK (
    (user_type = 'guest' AND expires_at > created_at) OR
    (user_type = 'registered' AND expires_at IS NULL)
);

-- Index for guest cleanup job (daily job to delete expired guests)
CREATE INDEX idx_users_guest_expiry ON users(user_type, expires_at)
    WHERE user_type = 'guest' AND expires_at IS NOT NULL;

-- Index for guest IP tracking (rate limiting, security)
CREATE INDEX idx_users_guest_ip ON users(ip_address)
    WHERE user_type = 'guest';

-- Index for active guests (not expired - filter by expires_at at query time)
CREATE INDEX idx_users_active_guests ON users(user_type, expires_at, created_at DESC)
    WHERE user_type = 'guest';

COMMENT ON COLUMN users.user_type IS 'User type: registered (normal account), guest (temporary account)';
COMMENT ON COLUMN users.ip_address IS 'IP address for guest users (used for rate limiting and security)';
COMMENT ON COLUMN users.expires_at IS 'Expiration timestamp for guest accounts (NULL for registered users)';

-- +goose Down
-- Remove guest user support

DROP INDEX IF EXISTS idx_users_active_guests;
DROP INDEX IF EXISTS idx_users_guest_ip;
DROP INDEX IF EXISTS idx_users_guest_expiry;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_guest_expires_future;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_guest_ip_required;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_user_type_check;

ALTER TABLE users DROP COLUMN IF EXISTS expires_at;
ALTER TABLE users DROP COLUMN IF EXISTS ip_address;
ALTER TABLE users DROP COLUMN IF EXISTS user_type;
