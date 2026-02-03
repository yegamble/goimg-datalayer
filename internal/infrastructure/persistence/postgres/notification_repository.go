package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

// SQL queries for notification operations.
const (
	sqlInsertNotification = `
		INSERT INTO notifications (id, recipient_id, notification_type, title, body, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	sqlSelectNotificationByID = `
		SELECT id, recipient_id, notification_type, title, body, metadata, read_at, created_at
		FROM notifications
		WHERE id = $1
	`

	sqlSelectNotificationsByRecipient = `
		SELECT id, recipient_id, notification_type, title, body, metadata, read_at, created_at
		FROM notifications
		WHERE recipient_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlSelectUnreadNotificationsByRecipient = `
		SELECT id, recipient_id, notification_type, title, body, metadata, read_at, created_at
		FROM notifications
		WHERE recipient_id = $1 AND read_at IS NULL
		ORDER BY created_at DESC
	`

	sqlCountUnreadNotifications = `
		SELECT COUNT(*)
		FROM notifications
		WHERE recipient_id = $1 AND read_at IS NULL
	`

	sqlMarkNotificationAsRead = `
		UPDATE notifications
		SET read_at = $1
		WHERE id = $2 AND read_at IS NULL
	`

	sqlMarkManyNotificationsAsRead = `
		UPDATE notifications
		SET read_at = $1
		WHERE recipient_id = $2 AND id = ANY($3) AND read_at IS NULL
	`

	sqlMarkAllNotificationsRead = `
		UPDATE notifications
		SET read_at = $1
		WHERE recipient_id = $2 AND read_at IS NULL
	`

	sqlDeleteNotification = `
		DELETE FROM notifications
		WHERE id = $1
	`

	sqlDeleteOldNotifications = `
		DELETE FROM notifications
		WHERE created_at < $1
	`
)

// notificationRow represents a notification row in the database.
type notificationRow struct {
	ID               string       `db:"id"`
	RecipientID      string       `db:"recipient_id"`
	NotificationType string       `db:"notification_type"`
	Title            string       `db:"title"`
	Body             string       `db:"body"`
	Metadata         []byte       `db:"metadata"`
	ReadAt           sql.NullTime `db:"read_at"`
	CreatedAt        time.Time    `db:"created_at"`
}

// toDomain converts a database row to a domain Notification entity.
func (r *notificationRow) toDomain() (*notification.Notification, error) {
	notificationID, err := notification.ParseNotificationID(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parse notification id: %w", err)
	}

	recipientID, err := identity.ParseUserID(r.RecipientID)
	if err != nil {
		return nil, fmt.Errorf("parse recipient id: %w", err)
	}

	notifType := notification.NotificationType(r.NotificationType)
	if !notifType.IsValid() {
		return nil, fmt.Errorf("invalid notification type: %s", r.NotificationType)
	}

	var readAt *time.Time
	if r.ReadAt.Valid {
		readAt = &r.ReadAt.Time
	}

	return notification.ReconstructNotification(
		notificationID,
		recipientID,
		notifType,
		r.Title,
		r.Body,
		r.Metadata,
		readAt,
		r.CreatedAt,
	), nil
}

// fromDomain converts a domain Notification entity to a database row.
func notificationFromDomain(n *notification.Notification) (*notificationRow, error) {
	// Use raw metadata (avoids unmarshal/marshal cycle if not modified)
	metadataJSON := n.MetadataRaw()

	row := &notificationRow{
		ID:               n.ID().String(),
		RecipientID:      n.RecipientID().String(),
		NotificationType: n.Type().String(),
		Title:            n.Title(),
		Body:             n.Body(),
		Metadata:         metadataJSON,
		CreatedAt:        n.CreatedAt(),
	}

	if n.ReadAt() != nil {
		row.ReadAt = sql.NullTime{Time: *n.ReadAt(), Valid: true}
	}

	return row, nil
}

// NotificationRepository implements the notification.NotificationRepository interface for PostgreSQL.
type NotificationRepository struct {
	db *sqlx.DB
}

// NewNotificationRepository creates a new NotificationRepository with the given database connection.
func NewNotificationRepository(db *sqlx.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Save persists a notification to the repository.
func (r *NotificationRepository) Save(ctx context.Context, n *notification.Notification) error {
	row, err := notificationFromDomain(n)
	if err != nil {
		return fmt.Errorf("convert notification to row: %w", err)
	}

	_, err = r.db.ExecContext(
		ctx,
		sqlInsertNotification,
		row.ID,
		row.RecipientID,
		row.NotificationType,
		row.Title,
		row.Body,
		row.Metadata,
		row.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save notification: %w", err)
	}

	return nil
}

// FindByID retrieves a notification by its ID.
// Returns ErrNotificationNotFound if not found.
func (r *NotificationRepository) FindByID(ctx context.Context, id notification.NotificationID) (*notification.Notification, error) {
	var row notificationRow
	err := r.db.GetContext(ctx, &row, sqlSelectNotificationByID, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notification.ErrNotificationNotFound
		}
		return nil, fmt.Errorf("find notification by id: %w", err)
	}

	n, err := row.toDomain()
	if err != nil {
		return nil, fmt.Errorf("convert row to domain: %w", err)
	}

	return n, nil
}

// FindByRecipient retrieves notifications for a specific user with pagination.
// Results are ordered by created_at DESC (newest first).
func (r *NotificationRepository) FindByRecipient(
	ctx context.Context,
	recipientID identity.UserID,
	limit, offset int,
) ([]*notification.Notification, error) {
	var rows []notificationRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectNotificationsByRecipient,
		recipientID.String(),
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("find notifications by recipient: %w", err)
	}

	notifications := make([]*notification.Notification, 0, len(rows))
	for _, row := range rows {
		n, err := row.toDomain()
		if err != nil {
			return nil, fmt.Errorf("convert row to domain: %w", err)
		}
		notifications = append(notifications, n)
	}

	return notifications, nil
}

// FindUnreadByRecipient retrieves all unread notifications for a user.
// Results are ordered by created_at DESC (newest first).
func (r *NotificationRepository) FindUnreadByRecipient(
	ctx context.Context,
	recipientID identity.UserID,
) ([]*notification.Notification, error) {
	var rows []notificationRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectUnreadNotificationsByRecipient,
		recipientID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("find unread notifications: %w", err)
	}

	notifications := make([]*notification.Notification, 0, len(rows))
	for _, row := range rows {
		n, err := row.toDomain()
		if err != nil {
			return nil, fmt.Errorf("convert row to domain: %w", err)
		}
		notifications = append(notifications, n)
	}

	return notifications, nil
}

// CountUnread returns the number of unread notifications for a user.
func (r *NotificationRepository) CountUnread(ctx context.Context, recipientID identity.UserID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, sqlCountUnreadNotifications, recipientID.String())
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}

	return count, nil
}

// MarkAsRead marks a specific notification as read.
// This operation is idempotent - marking an already-read notification succeeds.
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id notification.NotificationID) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, sqlMarkNotificationAsRead, now, id.String())
	if err != nil {
		return fmt.Errorf("mark notification as read: %w", err)
	}

	return nil
}

// MarkManyAsRead marks multiple notifications as read for a specific user.
func (r *NotificationRepository) MarkManyAsRead(ctx context.Context, ids []notification.NotificationID, recipientID identity.UserID) error {
	if len(ids) == 0 {
		return nil
	}

	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = id.String()
	}

	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, sqlMarkManyNotificationsAsRead, now, recipientID.String(), pq.Array(idStrings))
	if err != nil {
		return fmt.Errorf("mark many notifications as read: %w", err)
	}

	return nil
}

// MarkAllRead marks all notifications for a user as read.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, recipientID identity.UserID) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, sqlMarkAllNotificationsRead, now, recipientID.String())
	if err != nil {
		return fmt.Errorf("mark all notifications as read: %w", err)
	}

	return nil
}

// Delete removes a notification from storage.
func (r *NotificationRepository) Delete(ctx context.Context, id notification.NotificationID) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteNotification, id.String())
	if err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return notification.ErrNotificationNotFound
	}

	return nil
}

// DeleteOlderThan removes notifications older than the specified time.
// This is used for cleanup/retention policies.
func (r *NotificationRepository) DeleteOlderThan(ctx context.Context, before time.Time) error {
	_, err := r.db.ExecContext(ctx, sqlDeleteOldNotifications, before)
	if err != nil {
		return fmt.Errorf("delete old notifications: %w", err)
	}

	return nil
}
