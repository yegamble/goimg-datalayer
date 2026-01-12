-- +goose Up
-- +goose StatementBegin
-- Add new activity types to group_activities table for admin audit logging
-- Sprint 20: Groups/Communities feature

-- Drop existing check constraint
ALTER TABLE group_activities DROP CONSTRAINT IF EXISTS group_activities_type_check;

-- Recreate check constraint with new activity types
ALTER TABLE group_activities ADD CONSTRAINT group_activities_type_check CHECK (activity_type IN (
    'member_joined',
    'member_left',
    'member_promoted',
    'member_demoted',
    'member_banned',
    'member_removed',
    'image_shared',
    'image_approved',
    'image_rejected',
    'album_created',
    'album_deleted',
    'group_deleted',
    'settings_updated',
    'description_updated'
));

COMMENT ON COLUMN group_activities.activity_type IS 'Type of activity event (includes admin actions: banned, removed, deleted)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Restore original check constraint (remove new activity types)

ALTER TABLE group_activities DROP CONSTRAINT IF EXISTS group_activities_type_check;

ALTER TABLE group_activities ADD CONSTRAINT group_activities_type_check CHECK (activity_type IN (
    'member_joined',
    'member_left',
    'member_promoted',
    'member_demoted',
    'image_shared',
    'image_approved',
    'image_rejected',
    'album_created',
    'album_deleted',
    'settings_updated',
    'description_updated'
));

COMMENT ON COLUMN group_activities.activity_type IS 'Type of activity event';

-- +goose StatementEnd
