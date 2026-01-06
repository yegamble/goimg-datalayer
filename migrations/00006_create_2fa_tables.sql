-- +goose Up
-- +goose StatementBegin

-- Table: user_totp_secrets
-- Stores encrypted TOTP secrets for two-factor authentication
CREATE TABLE user_totp_secrets (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    encrypted_secret BYTEA NOT NULL,
    issuer VARCHAR(100) NOT NULL DEFAULT 'goimg',
    account_name VARCHAR(255) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_totp_enabled ON user_totp_secrets(user_id) WHERE enabled = true;

COMMENT ON TABLE user_totp_secrets IS 'Encrypted TOTP secrets for two-factor authentication';
COMMENT ON COLUMN user_totp_secrets.encrypted_secret IS 'AES-256-GCM encrypted base32 TOTP secret';
COMMENT ON COLUMN user_totp_secrets.issuer IS 'App name shown in authenticator app (e.g., "goimg")';
COMMENT ON COLUMN user_totp_secrets.account_name IS 'Account identifier shown in authenticator (usually email)';
COMMENT ON COLUMN user_totp_secrets.enabled IS 'Whether 2FA is active (false during setup until first successful verify)';
COMMENT ON COLUMN user_totp_secrets.verified_at IS 'Timestamp of first successful TOTP verification';

-- Table: user_backup_codes
-- Stores hashed one-time backup codes (like passwords)
CREATE TABLE user_backup_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(255) NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backup_codes_user ON user_backup_codes(user_id);
CREATE INDEX idx_backup_codes_unused ON user_backup_codes(user_id) WHERE used = false;

COMMENT ON TABLE user_backup_codes IS 'One-time use backup codes for 2FA recovery (hashed like passwords)';
COMMENT ON COLUMN user_backup_codes.code_hash IS 'Argon2id hashed backup code (8 character base32 plaintext)';
COMMENT ON COLUMN user_backup_codes.used IS 'Whether this code has been consumed';
COMMENT ON COLUMN user_backup_codes.used_at IS 'Timestamp when code was used';

-- Table: user_devices
-- Tracks known devices for unusual login detection
CREATE TABLE user_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fingerprint_hash VARCHAR(64) NOT NULL,
    ip_address INET NOT NULL,
    user_agent TEXT NOT NULL,
    device_name VARCHAR(100) NOT NULL DEFAULT 'Unknown Device',
    trusted BOOLEAN NOT NULL DEFAULT false,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_devices_user ON user_devices(user_id);
CREATE INDEX idx_user_devices_fingerprint ON user_devices(user_id, fingerprint_hash);
CREATE INDEX idx_user_devices_trusted ON user_devices(user_id) WHERE trusted = true;
CREATE UNIQUE INDEX idx_user_devices_unique ON user_devices(user_id, fingerprint_hash);

COMMENT ON TABLE user_devices IS 'Known devices for unusual login detection and notifications';
COMMENT ON COLUMN user_devices.fingerprint_hash IS 'SHA-256 hash of (IP address | UserAgent) for device identification';
COMMENT ON COLUMN user_devices.device_name IS 'Human-readable device name derived from user agent';
COMMENT ON COLUMN user_devices.trusted IS 'Whether this device is marked as trusted (prevents unusual login alerts)';
COMMENT ON COLUMN user_devices.first_seen_at IS 'First login from this device';
COMMENT ON COLUMN user_devices.last_seen_at IS 'Most recent login from this device';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_devices;
DROP TABLE IF EXISTS user_backup_codes;
DROP TABLE IF EXISTS user_totp_secrets;
-- +goose StatementEnd
