-- +goose Up
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    metadata JSONB DEFAULT '{}',
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_recipient_created ON notifications(recipient_id, created_at DESC);
CREATE INDEX idx_notifications_recipient_unread ON notifications(recipient_id) WHERE read_at IS NULL;
CREATE INDEX idx_notifications_type ON notifications(notification_type);

COMMENT ON TABLE notifications IS 'In-app notifications for users in the Notification bounded context';
COMMENT ON COLUMN notifications.id IS 'Unique notification identifier';
COMMENT ON COLUMN notifications.recipient_id IS 'ID of the user who receives this notification';
COMMENT ON COLUMN notifications.notification_type IS 'Type of notification (new_follower, new_photos, etc.)';
COMMENT ON COLUMN notifications.title IS 'Notification title (short summary)';
COMMENT ON COLUMN notifications.body IS 'Notification body (detailed message)';
COMMENT ON COLUMN notifications.metadata IS 'Flexible JSON payload for notification-specific data (user IDs, image IDs, etc.)';
COMMENT ON COLUMN notifications.read_at IS 'Timestamp when notification was marked as read (NULL if unread)';
COMMENT ON COLUMN notifications.created_at IS 'Timestamp when notification was created';

-- +goose Down
DROP TABLE IF EXISTS notifications;
