package commands_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/notification/commands"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

// StubNotificationRepository for benchmarking.
type StubNotificationRepository struct {
	RecipientID identity.UserID
}

func (r *StubNotificationRepository) Save(ctx context.Context, n *notification.Notification) error {
	return nil
}

func (r *StubNotificationRepository) FindByID(ctx context.Context, id notification.NotificationID) (*notification.Notification, error) {
	// Simulate DB latency
	time.Sleep(1 * time.Millisecond)

	// Return a dummy notification belonging to the configured recipient
	return notification.ReconstructNotification(
		id,
		r.RecipientID,
		notification.TypeNewFollower,
		"Title",
		"Body",
		nil,
		nil,
		time.Now(),
	), nil
}

func (r *StubNotificationRepository) FindByRecipient(ctx context.Context, recipientID identity.UserID, limit, offset int) ([]*notification.Notification, error) {
	return nil, nil
}

func (r *StubNotificationRepository) FindUnreadByRecipient(ctx context.Context, recipientID identity.UserID) ([]*notification.Notification, error) {
	return nil, nil
}

func (r *StubNotificationRepository) CountUnread(ctx context.Context, recipientID identity.UserID) (int, error) {
	return 0, nil
}

func (r *StubNotificationRepository) MarkAsRead(ctx context.Context, id notification.NotificationID) error {
	// Simulate DB latency
	time.Sleep(1 * time.Millisecond)
	return nil
}

func (r *StubNotificationRepository) MarkManyAsRead(ctx context.Context, ids []notification.NotificationID, recipientID identity.UserID) error {
	// Simulate DB latency (batch operation)
	time.Sleep(1 * time.Millisecond)
	return nil
}

func (r *StubNotificationRepository) MarkAllRead(ctx context.Context, recipientID identity.UserID) error {
	return nil
}

func (r *StubNotificationRepository) Delete(ctx context.Context, id notification.NotificationID) error {
	return nil
}

func (r *StubNotificationRepository) DeleteOlderThan(ctx context.Context, before time.Time) error {
	return nil
}

func BenchmarkMarkNotificationsReadHandler(b *testing.B) {
	userID := identity.NewUserID()
	repo := &StubNotificationRepository{RecipientID: userID}
	logger := zerolog.Nop()

	handler := commands.NewMarkNotificationsReadHandler(repo, logger)
	ctx := context.Background()

	// Prepare a command with 10 notification IDs
	ids := make([]string, 10)
	for i := 0; i < 10; i++ {
		ids[i] = notification.NewNotificationID().String()
	}

	cmd := commands.MarkNotificationsReadCommand{
		UserID:          userID.String(),
		NotificationIDs: ids,
		MarkAllAsRead:   false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := handler.Handle(ctx, cmd); err != nil {
			b.Fatal(err)
		}
	}
}
