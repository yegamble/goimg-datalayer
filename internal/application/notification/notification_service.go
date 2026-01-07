package notification

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/email"
)

// NotificationService provides application-level notification operations.
// It coordinates between notification persistence, user preferences, and email delivery.
//
// Business Rules:
// - In-app notifications are always created regardless of email preferences
// - Email notifications respect user preferences and SMTP configuration
// - Email failures do not fail the notification operation (logged only)
type NotificationService struct {
	notifications notification.NotificationRepository
	users         identity.UserRepository
	emailSender   *email.SMTPSender
	logger        zerolog.Logger
}

// NewNotificationService creates a new NotificationService with the given dependencies.
func NewNotificationService(
	notifications notification.NotificationRepository,
	users identity.UserRepository,
	emailSender *email.SMTPSender,
	logger zerolog.Logger,
) *NotificationService {
	return &NotificationService{
		notifications: notifications,
		users:         users,
		emailSender:   emailSender,
		logger:        logger,
	}
}

// NotifyNewFollower creates a notification when a user gains a new follower.
// This method:
// 1. Creates an in-app notification for the followed user
// 2. Checks the followed user's notification preferences
// 3. Sends an email if preferences allow and SMTP is enabled
//
// Parameters:
// - followerID: ID of the user who followed
// - followedID: ID of the user who was followed (notification recipient)
//
// Returns an error if notification creation fails. Email failures are logged but do not fail the operation.
func (s *NotificationService) NotifyNewFollower(
	ctx context.Context,
	followerID, followedID identity.UserID,
) error {
	// 1. Load follower user to get username for notification
	follower, err := s.users.FindByID(ctx, followerID)
	if err != nil {
		return fmt.Errorf("find follower user: %w", err)
	}

	// 2. Load followed user to check email preferences
	followed, err := s.users.FindByID(ctx, followedID)
	if err != nil {
		return fmt.Errorf("find followed user: %w", err)
	}

	// 3. Create notification title and body
	title := fmt.Sprintf("%s started following you", follower.Username().String())
	body := fmt.Sprintf(
		"%s is now following you. View their profile to see their photos.",
		follower.Username().String(),
	)

	// 4. Create metadata with follower information
	metadata := map[string]string{
		"follower_id":       followerID.String(),
		"follower_username": follower.Username().String(),
	}

	// 5. Create in-app notification via domain factory
	notif, err := notification.NewNotification(
		followedID,
		notification.TypeNewFollower,
		title,
		body,
		metadata,
	)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	// 6. Persist notification
	if err := s.notifications.Save(ctx, notif); err != nil {
		return fmt.Errorf("save notification: %w", err)
	}

	s.logger.Info().
		Str("notification_id", notif.ID().String()).
		Str("recipient_id", followedID.String()).
		Str("follower_id", followerID.String()).
		Msg("new follower notification created")

	// 7. Send email if preferences allow (best-effort, don't fail on email errors)
	s.sendEmailIfEnabled(ctx, followed, follower.Username().String(), notification.TypeNewFollower)

	return nil
}

// sendEmailIfEnabled sends an email notification if:
// 1. SMTP is enabled in configuration
// 2. User has email notifications enabled for this notification type
//
// This method logs errors but does not fail the notification operation.
func (s *NotificationService) sendEmailIfEnabled(
	ctx context.Context,
	recipient *identity.User,
	followerUsername string,
	notifType notification.NotificationType,
) {
	// Check if SMTP is enabled
	if !s.emailSender.IsEnabled() {
		s.logger.Debug().
			Str("recipient_id", recipient.ID().String()).
			Msg("smtp disabled, skipping email notification")
		return
	}

	// Check user's notification preferences
	prefs := recipient.NotificationPreferences()
	if !prefs.ShouldEmail(notifType) {
		s.logger.Debug().
			Str("recipient_id", recipient.ID().String()).
			Str("notification_type", notifType.String()).
			Msg("user opted out of email for this notification type")
		return
	}

	// Send email using SMTP sender's built-in template
	recipientEmail := recipient.Email().String()
	err := s.emailSender.SendNewFollowerEmail(ctx, recipientEmail, followerUsername)
	if err != nil {
		s.logger.Warn().
			Err(err).
			Str("recipient_email", recipientEmail).
			Str("follower_username", followerUsername).
			Msg("failed to send new follower email notification")
		return
	}

	s.logger.Info().
		Str("recipient_email", recipientEmail).
		Str("follower_username", followerUsername).
		Msg("new follower email sent successfully")
}
