-- +goose Up
-- +goose StatementBegin
ALTER TABLE group_albums
    ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN group_albums.is_public IS 'If true, album is visible to non-members when group is public';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE group_albums
    DROP COLUMN IF EXISTS is_public;
-- +goose StatementEnd
