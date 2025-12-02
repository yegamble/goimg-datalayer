# Notifications & Email System

> Load this guide when implementing notification features, SMTP email, or user follow functionality.

## Overview

goimg-datalayer supports a dual notification system:
- **Internal notifications**: In-app notifications stored in the database
- **Email notifications**: SMTP-based emails for opted-in users

Users control their preferences and can limit notifications to internal-only or opt-in to email delivery.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Domain Events                                │
│     (ImageUploaded, UserFollowed, UserBanned, ReportCreated)    │
├─────────────────────────────────────────────────────────────────┤
│                  Notification Service                            │
│              (Application Layer Orchestrator)                    │
├──────────────────────┬──────────────────────────────────────────┤
│  Internal Notifier   │           Email Notifier                  │
│  (DB persistence)    │           (SMTP sender)                   │
│                      │                                           │
│  - Always active     │   - Opt-in per user                       │
│  - Real-time push    │   - Batched/grouped delivery              │
│  - Read/unread state │   - Rate-limited                          │
└──────────────────────┴──────────────────────────────────────────┘
```

## Notification Types

### User Notifications

| Type | Description | Grouping | Email Eligible |
|------|-------------|----------|----------------|
| `new_follower` | Someone followed you | No | Yes |
| `new_photos` | Followed user uploaded photos | Yes (batch by uploader) | Yes |
| `account_suspended` | Account temporarily suspended | No | Always |
| `account_banned` | Account permanently banned | No | Always |
| `account_reinstated` | Account restored | No | Always |

### Admin/Moderator Notifications

| Type | Description | Grouping | Email Eligible |
|------|-------------|----------|----------------|
| `abuse_report` | New abuse report submitted | No | Yes |
| `report_escalated` | Report escalated for review | No | Yes |
| `mod_action_required` | Pending moderation queue | Yes (daily digest) | Yes |

## Domain Layer

### Notification Entity

```go
// internal/domain/notification/notification.go

package notification

type Notification struct {
    id          NotificationID
    recipientID UserID
    notifType   NotificationType
    title       string
    body        string
    metadata    map[string]string  // Flexible payload (image IDs, user IDs, etc.)
    readAt      *time.Time
    createdAt   time.Time
}

// Factory with validation
func NewNotification(
    recipientID UserID,
    notifType NotificationType,
    title, body string,
    metadata map[string]string,
) (*Notification, error) {
    if recipientID.IsZero() {
        return nil, ErrRecipientRequired
    }
    if title == "" {
        return nil, ErrTitleRequired
    }
    return &Notification{
        id:          NewNotificationID(),
        recipientID: recipientID,
        notifType:   notifType,
        title:       title,
        body:        body,
        metadata:    metadata,
        createdAt:   time.Now().UTC(),
    }, nil
}

func (n *Notification) MarkRead() {
    if n.readAt == nil {
        now := time.Now().UTC()
        n.readAt = &now
    }
}
```

### Notification Types

```go
// internal/domain/notification/types.go

package notification

type NotificationType string

const (
    // User notifications
    TypeNewFollower       NotificationType = "new_follower"
    TypeNewPhotos         NotificationType = "new_photos"
    TypeAccountSuspended  NotificationType = "account_suspended"
    TypeAccountBanned     NotificationType = "account_banned"
    TypeAccountReinstated NotificationType = "account_reinstated"

    // Admin/Mod notifications
    TypeAbuseReport        NotificationType = "abuse_report"
    TypeReportEscalated    NotificationType = "report_escalated"
    TypeModActionRequired  NotificationType = "mod_action_required"
)

// RequiresEmail returns true for notifications that always send email
func (t NotificationType) RequiresEmail() bool {
    switch t {
    case TypeAccountSuspended, TypeAccountBanned, TypeAccountReinstated:
        return true
    default:
        return false
    }
}

// IsAdminOnly returns true for admin/mod-only notifications
func (t NotificationType) IsAdminOnly() bool {
    switch t {
    case TypeAbuseReport, TypeReportEscalated, TypeModActionRequired:
        return true
    default:
        return false
    }
}
```

### User Notification Preferences

```go
// internal/domain/identity/notification_preferences.go

package identity

type NotificationPreferences struct {
    emailEnabled     bool                       // Global email opt-in
    emailTypes       map[NotificationType]bool  // Per-type email opt-in
    digestFrequency  DigestFrequency            // For grouped notifications
}

type DigestFrequency string

const (
    DigestImmediate DigestFrequency = "immediate"  // Send as they happen
    DigestHourly    DigestFrequency = "hourly"
    DigestDaily     DigestFrequency = "daily"
    DigestWeekly    DigestFrequency = "weekly"
)

func DefaultNotificationPreferences() NotificationPreferences {
    return NotificationPreferences{
        emailEnabled:    false,  // Opt-in by default
        emailTypes:      make(map[NotificationType]bool),
        digestFrequency: DigestDaily,
    }
}

func (p *NotificationPreferences) EnableEmail() {
    p.emailEnabled = true
}

func (p *NotificationPreferences) DisableEmail() {
    p.emailEnabled = false
}

func (p *NotificationPreferences) ShouldEmail(notifType NotificationType) bool {
    // Account status emails always send
    if notifType.RequiresEmail() {
        return true
    }
    // Otherwise respect user preferences
    if !p.emailEnabled {
        return false
    }
    // Check per-type preference (default to true if email is globally enabled)
    enabled, exists := p.emailTypes[notifType]
    if !exists {
        return true
    }
    return enabled
}
```

### Follow Aggregate

```go
// internal/domain/identity/follow.go

package identity

type Follow struct {
    id          FollowID
    followerID  UserID  // Who is following
    followedID  UserID  // Who is being followed
    createdAt   time.Time
}

func NewFollow(followerID, followedID UserID) (*Follow, error) {
    if followerID.IsZero() || followedID.IsZero() {
        return nil, ErrInvalidFollowIDs
    }
    if followerID == followedID {
        return nil, ErrCannotFollowSelf
    }
    return &Follow{
        id:         NewFollowID(),
        followerID: followerID,
        followedID: followedID,
        createdAt:  time.Now().UTC(),
    }, nil
}
```

### Repository Interfaces

```go
// internal/domain/notification/repository.go

package notification

type NotificationRepository interface {
    Save(ctx context.Context, n *Notification) error
    FindByID(ctx context.Context, id NotificationID) (*Notification, error)
    FindByRecipient(ctx context.Context, recipientID UserID, opts ListOptions) ([]*Notification, error)
    FindUnreadByRecipient(ctx context.Context, recipientID UserID) ([]*Notification, error)
    CountUnread(ctx context.Context, recipientID UserID) (int, error)
    MarkAllRead(ctx context.Context, recipientID UserID) error
    Delete(ctx context.Context, id NotificationID) error
    DeleteOlderThan(ctx context.Context, before time.Time) error
}

// internal/domain/identity/follow_repository.go

type FollowRepository interface {
    Save(ctx context.Context, f *Follow) error
    Delete(ctx context.Context, followerID, followedID UserID) error
    FindByFollower(ctx context.Context, followerID UserID) ([]*Follow, error)
    FindByFollowed(ctx context.Context, followedID UserID) ([]*Follow, error)
    Exists(ctx context.Context, followerID, followedID UserID) (bool, error)
    CountFollowers(ctx context.Context, userID UserID) (int, error)
    CountFollowing(ctx context.Context, userID UserID) (int, error)
}
```

## Domain Events

```go
// internal/domain/identity/events.go

type UserFollowed struct {
    FollowerID  UserID
    FollowedID  UserID
    OccurredOn  time.Time
}

type UserUnfollowed struct {
    FollowerID  UserID
    FollowedID  UserID
    OccurredOn  time.Time
}

type UserSuspended struct {
    UserID     UserID
    Reason     string
    ExpiresAt  *time.Time
    OccurredOn time.Time
}

type UserBanned struct {
    UserID     UserID
    Reason     string
    OccurredOn time.Time
}

type UserReinstated struct {
    UserID     UserID
    OccurredOn time.Time
}

// internal/domain/gallery/events.go

type ImagesUploaded struct {
    UploaderID UserID
    ImageIDs   []ImageID
    OccurredOn time.Time
}

// internal/domain/moderation/events.go

type AbuseReportCreated struct {
    ReportID    ReportID
    ReporterID  UserID
    TargetType  string      // "image", "user", "comment"
    TargetID    string
    Reason      string
    OccurredOn  time.Time
}
```

## Application Layer

### Notification Service

```go
// internal/application/notification/service.go

package notification

type NotificationService struct {
    notifRepo      NotificationRepository
    userRepo       UserRepository
    followRepo     FollowRepository
    internalSender InternalNotifier
    emailSender    EmailNotifier
    batcher        NotificationBatcher
}

// NotifyNewFollower sends notification when someone gains a follower
func (s *NotificationService) NotifyNewFollower(ctx context.Context, event UserFollowed) error {
    follower, err := s.userRepo.FindByID(ctx, event.FollowerID)
    if err != nil {
        return fmt.Errorf("find follower: %w", err)
    }

    followed, err := s.userRepo.FindByID(ctx, event.FollowedID)
    if err != nil {
        return fmt.Errorf("find followed: %w", err)
    }

    notif, err := NewNotification(
        followed.ID(),
        TypeNewFollower,
        fmt.Sprintf("%s started following you", follower.Username()),
        fmt.Sprintf("%s is now following your gallery", follower.Username()),
        map[string]string{
            "follower_id":       event.FollowerID.String(),
            "follower_username": follower.Username().String(),
        },
    )
    if err != nil {
        return fmt.Errorf("create notification: %w", err)
    }

    // Always save internal notification
    if err := s.notifRepo.Save(ctx, notif); err != nil {
        return fmt.Errorf("save notification: %w", err)
    }

    // Send real-time push
    s.internalSender.Push(ctx, notif)

    // Check email preferences
    prefs := followed.NotificationPreferences()
    if prefs.ShouldEmail(TypeNewFollower) {
        if err := s.emailSender.SendNewFollowerEmail(ctx, followed.Email(), follower); err != nil {
            // Log but don't fail - email is best-effort
            log.Error().Err(err).Msg("failed to send new follower email")
        }
    }

    return nil
}

// NotifyNewPhotos sends grouped notifications for new uploads
func (s *NotificationService) NotifyNewPhotos(ctx context.Context, event ImagesUploaded) error {
    uploader, err := s.userRepo.FindByID(ctx, event.UploaderID)
    if err != nil {
        return fmt.Errorf("find uploader: %w", err)
    }

    // Get all followers
    followers, err := s.followRepo.FindByFollowed(ctx, event.UploaderID)
    if err != nil {
        return fmt.Errorf("find followers: %w", err)
    }

    // Batch notification to avoid spam
    photoCount := len(event.ImageIDs)
    title := fmt.Sprintf("%s uploaded %d new photo", uploader.Username(), photoCount)
    if photoCount > 1 {
        title += "s"
    }

    for _, follow := range followers {
        follower, err := s.userRepo.FindByID(ctx, follow.FollowerID())
        if err != nil {
            continue
        }

        notif, _ := NewNotification(
            follow.FollowerID(),
            TypeNewPhotos,
            title,
            fmt.Sprintf("Check out the latest uploads from %s", uploader.Username()),
            map[string]string{
                "uploader_id": event.UploaderID.String(),
                "image_count": strconv.Itoa(photoCount),
                "image_ids":   strings.Join(imageIDsToStrings(event.ImageIDs), ","),
            },
        )

        if err := s.notifRepo.Save(ctx, notif); err != nil {
            continue
        }

        s.internalSender.Push(ctx, notif)

        // Batch emails based on user digest preferences
        prefs := follower.NotificationPreferences()
        if prefs.ShouldEmail(TypeNewPhotos) {
            s.batcher.Queue(ctx, follower.ID(), notif, prefs.DigestFrequency())
        }
    }

    return nil
}

// NotifyAccountStatus sends mandatory notifications for account changes
func (s *NotificationService) NotifyAccountSuspended(ctx context.Context, event UserSuspended) error {
    user, err := s.userRepo.FindByID(ctx, event.UserID)
    if err != nil {
        return fmt.Errorf("find user: %w", err)
    }

    notif, _ := NewNotification(
        event.UserID,
        TypeAccountSuspended,
        "Your account has been suspended",
        fmt.Sprintf("Reason: %s", event.Reason),
        map[string]string{
            "reason":     event.Reason,
            "expires_at": formatExpiry(event.ExpiresAt),
        },
    )

    if err := s.notifRepo.Save(ctx, notif); err != nil {
        return fmt.Errorf("save notification: %w", err)
    }

    // Account status emails always send (ignore preferences)
    if err := s.emailSender.SendAccountSuspendedEmail(ctx, user.Email(), event.Reason, event.ExpiresAt); err != nil {
        return fmt.Errorf("send suspension email: %w", err)
    }

    return nil
}
```

### Admin Notification Service

```go
// internal/application/notification/admin_service.go

package notification

type AdminNotificationService struct {
    notifRepo   NotificationRepository
    userRepo    UserRepository
    emailSender EmailNotifier
}

// NotifyAbuseReport sends notifications to admins/mods about new reports
func (s *AdminNotificationService) NotifyAbuseReport(ctx context.Context, event AbuseReportCreated) error {
    // Get all admins and mods
    admins, err := s.userRepo.FindByRoles(ctx, RoleAdmin, RoleModerator)
    if err != nil {
        return fmt.Errorf("find admins: %w", err)
    }

    for _, admin := range admins {
        notif, _ := NewNotification(
            admin.ID(),
            TypeAbuseReport,
            fmt.Sprintf("New abuse report: %s", event.TargetType),
            fmt.Sprintf("A %s has been reported for: %s", event.TargetType, event.Reason),
            map[string]string{
                "report_id":   event.ReportID.String(),
                "target_type": event.TargetType,
                "target_id":   event.TargetID,
                "reason":      event.Reason,
            },
        )

        if err := s.notifRepo.Save(ctx, notif); err != nil {
            continue
        }

        // Check admin email preferences
        prefs := admin.NotificationPreferences()
        if prefs.ShouldEmail(TypeAbuseReport) {
            if err := s.emailSender.SendAbuseReportEmail(ctx, admin.Email(), event); err != nil {
                log.Error().Err(err).Str("admin_id", admin.ID().String()).Msg("failed to send abuse report email")
            }
        }
    }

    return nil
}
```

## Infrastructure Layer

### SMTP Email Sender

```go
// internal/infrastructure/email/smtp.go

package email

import (
    "context"
    "crypto/tls"
    "fmt"
    "net/smtp"
    "time"
)

type SMTPConfig struct {
    Host         string        `env:"SMTP_HOST" envDefault:"localhost"`
    Port         int           `env:"SMTP_PORT" envDefault:"587"`
    Username     string        `env:"SMTP_USERNAME"`
    Password     string        `env:"SMTP_PASSWORD"`
    FromAddress  string        `env:"SMTP_FROM_ADDRESS"`
    FromName     string        `env:"SMTP_FROM_NAME" envDefault:"goimg Gallery"`
    UseTLS       bool          `env:"SMTP_USE_TLS" envDefault:"true"`
    Timeout      time.Duration `env:"SMTP_TIMEOUT" envDefault:"30s"`
    RateLimit    int           `env:"SMTP_RATE_LIMIT" envDefault:"100"`  // per hour
}

type SMTPSender struct {
    config    SMTPConfig
    auth      smtp.Auth
    templates *TemplateRenderer
    limiter   *RateLimiter
}

func NewSMTPSender(cfg SMTPConfig, templates *TemplateRenderer) (*SMTPSender, error) {
    if cfg.Host == "" {
        return nil, fmt.Errorf("smtp: host required")
    }
    if cfg.FromAddress == "" {
        return nil, fmt.Errorf("smtp: from address required")
    }

    var auth smtp.Auth
    if cfg.Username != "" {
        auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
    }

    return &SMTPSender{
        config:    cfg,
        auth:      auth,
        templates: templates,
        limiter:   NewRateLimiter(cfg.RateLimit, time.Hour),
    }, nil
}

func (s *SMTPSender) Send(ctx context.Context, to, subject, htmlBody, textBody string) error {
    if !s.limiter.Allow() {
        return ErrRateLimited
    }

    msg := s.buildMessage(to, subject, htmlBody, textBody)

    addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

    if s.config.UseTLS {
        return s.sendTLS(addr, to, msg)
    }
    return smtp.SendMail(addr, s.auth, s.config.FromAddress, []string{to}, msg)
}

func (s *SMTPSender) sendTLS(addr, to string, msg []byte) error {
    conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.config.Host})
    if err != nil {
        return fmt.Errorf("tls dial: %w", err)
    }
    defer conn.Close()

    client, err := smtp.NewClient(conn, s.config.Host)
    if err != nil {
        return fmt.Errorf("smtp client: %w", err)
    }
    defer client.Close()

    if s.auth != nil {
        if err := client.Auth(s.auth); err != nil {
            return fmt.Errorf("smtp auth: %w", err)
        }
    }

    if err := client.Mail(s.config.FromAddress); err != nil {
        return fmt.Errorf("smtp mail: %w", err)
    }

    if err := client.Rcpt(to); err != nil {
        return fmt.Errorf("smtp rcpt: %w", err)
    }

    w, err := client.Data()
    if err != nil {
        return fmt.Errorf("smtp data: %w", err)
    }

    if _, err := w.Write(msg); err != nil {
        return fmt.Errorf("smtp write: %w", err)
    }

    if err := w.Close(); err != nil {
        return fmt.Errorf("smtp close: %w", err)
    }

    return client.Quit()
}

func (s *SMTPSender) buildMessage(to, subject, htmlBody, textBody string) []byte {
    headers := make(map[string]string)
    headers["From"] = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromAddress)
    headers["To"] = to
    headers["Subject"] = subject
    headers["MIME-Version"] = "1.0"
    headers["Content-Type"] = "multipart/alternative; boundary=\"boundary\""

    var msg string
    for k, v := range headers {
        msg += fmt.Sprintf("%s: %s\r\n", k, v)
    }
    msg += "\r\n--boundary\r\n"
    msg += "Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n"
    msg += textBody
    msg += "\r\n--boundary\r\n"
    msg += "Content-Type: text/html; charset=\"utf-8\"\r\n\r\n"
    msg += htmlBody
    msg += "\r\n--boundary--"

    return []byte(msg)
}
```

### Email Templates

```go
// internal/infrastructure/email/templates.go

package email

type TemplateRenderer struct {
    templates *template.Template
}

func NewTemplateRenderer(templatesDir string) (*TemplateRenderer, error) {
    tmpl, err := template.ParseGlob(filepath.Join(templatesDir, "*.html"))
    if err != nil {
        return nil, fmt.Errorf("parse templates: %w", err)
    }
    return &TemplateRenderer{templates: tmpl}, nil
}

func (r *TemplateRenderer) RenderNewFollower(data NewFollowerEmailData) (html, text string, err error) {
    var htmlBuf, textBuf bytes.Buffer
    if err := r.templates.ExecuteTemplate(&htmlBuf, "new_follower.html", data); err != nil {
        return "", "", err
    }
    if err := r.templates.ExecuteTemplate(&textBuf, "new_follower.txt", data); err != nil {
        return "", "", err
    }
    return htmlBuf.String(), textBuf.String(), nil
}

// Template data structures
type NewFollowerEmailData struct {
    RecipientName    string
    FollowerName     string
    FollowerUsername string
    FollowerURL      string
    UnsubscribeURL   string
}

type NewPhotosEmailData struct {
    RecipientName   string
    UploaderName    string
    PhotoCount      int
    PreviewURLs     []string
    GalleryURL      string
    UnsubscribeURL  string
}

type AccountStatusEmailData struct {
    RecipientName string
    Status        string  // "suspended", "banned", "reinstated"
    Reason        string
    ExpiresAt     string  // Empty for bans/reinstatements
    AppealURL     string
    SupportEmail  string
}

type AbuseReportEmailData struct {
    RecipientName string
    TargetType    string
    TargetID      string
    ReportReason  string
    ReviewURL     string
}
```

### Notification Batcher

```go
// internal/infrastructure/notification/batcher.go

package notification

import (
    "context"
    "sync"
    "time"
)

// NotificationBatcher groups notifications for digest emails
type NotificationBatcher struct {
    mu       sync.RWMutex
    pending  map[UserID][]*PendingNotification
    sender   EmailNotifier
    repo     NotificationRepository
    ticker   *time.Ticker
}

type PendingNotification struct {
    Notification *Notification
    QueuedAt     time.Time
    Frequency    DigestFrequency
}

func NewNotificationBatcher(sender EmailNotifier, repo NotificationRepository) *NotificationBatcher {
    b := &NotificationBatcher{
        pending: make(map[UserID][]*PendingNotification),
        sender:  sender,
        repo:    repo,
        ticker:  time.NewTicker(time.Minute),
    }
    go b.processLoop()
    return b
}

func (b *NotificationBatcher) Queue(ctx context.Context, userID UserID, n *Notification, freq DigestFrequency) {
    if freq == DigestImmediate {
        // Send immediately, don't batch
        b.sender.SendDigest(ctx, userID, []*Notification{n})
        return
    }

    b.mu.Lock()
    defer b.mu.Unlock()

    b.pending[userID] = append(b.pending[userID], &PendingNotification{
        Notification: n,
        QueuedAt:     time.Now(),
        Frequency:    freq,
    })
}

func (b *NotificationBatcher) processLoop() {
    for range b.ticker.C {
        b.processPending()
    }
}

func (b *NotificationBatcher) processPending() {
    b.mu.Lock()
    defer b.mu.Unlock()

    now := time.Now()

    for userID, pending := range b.pending {
        var toSend []*Notification
        var remaining []*PendingNotification

        for _, p := range pending {
            if b.shouldSend(p, now) {
                toSend = append(toSend, p.Notification)
            } else {
                remaining = append(remaining, p)
            }
        }

        if len(toSend) > 0 {
            ctx := context.Background()
            if err := b.sender.SendDigest(ctx, userID, toSend); err != nil {
                // Re-queue on failure
                remaining = append(remaining, pending...)
            }
        }

        if len(remaining) > 0 {
            b.pending[userID] = remaining
        } else {
            delete(b.pending, userID)
        }
    }
}

func (b *NotificationBatcher) shouldSend(p *PendingNotification, now time.Time) bool {
    elapsed := now.Sub(p.QueuedAt)
    switch p.Frequency {
    case DigestHourly:
        return elapsed >= time.Hour
    case DigestDaily:
        return elapsed >= 24*time.Hour
    case DigestWeekly:
        return elapsed >= 7*24*time.Hour
    default:
        return true
    }
}
```

## Configuration

### Environment Variables

```bash
# SMTP Configuration
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=notifications@example.com
SMTP_PASSWORD=your-secure-password
SMTP_FROM_ADDRESS=notifications@example.com
SMTP_FROM_NAME="goimg Gallery"
SMTP_USE_TLS=true
SMTP_TIMEOUT=30s
SMTP_RATE_LIMIT=100

# Notification Settings
NOTIFICATION_RETENTION_DAYS=90           # Auto-delete old notifications
NOTIFICATION_BATCH_INTERVAL=1m           # How often to check digest queue
NOTIFICATION_MAX_PER_EMAIL=50            # Max notifications per digest email
```

### Config Struct

```go
// internal/config/notification.go

package config

type NotificationConfig struct {
    RetentionDays     int           `env:"NOTIFICATION_RETENTION_DAYS" envDefault:"90"`
    BatchInterval     time.Duration `env:"NOTIFICATION_BATCH_INTERVAL" envDefault:"1m"`
    MaxPerEmail       int           `env:"NOTIFICATION_MAX_PER_EMAIL" envDefault:"50"`
    EnableEmail       bool          `env:"NOTIFICATION_ENABLE_EMAIL" envDefault:"true"`
    EnablePush        bool          `env:"NOTIFICATION_ENABLE_PUSH" envDefault:"false"`

    SMTP SMTPConfig
}
```

## API Endpoints

Update `api/openapi/` spec to include:

```yaml
paths:
  /api/v1/notifications:
    get:
      summary: List user notifications
      parameters:
        - name: unread_only
          in: query
          schema:
            type: boolean
        - name: limit
          in: query
          schema:
            type: integer
            default: 20
        - name: offset
          in: query
          schema:
            type: integer
            default: 0
      responses:
        '200':
          description: List of notifications

  /api/v1/notifications/read:
    post:
      summary: Mark notifications as read
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                notification_ids:
                  type: array
                  items:
                    type: string
                all:
                  type: boolean

  /api/v1/notifications/preferences:
    get:
      summary: Get notification preferences
    put:
      summary: Update notification preferences
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NotificationPreferences'

  /api/v1/users/{userId}/follow:
    post:
      summary: Follow a user
    delete:
      summary: Unfollow a user

  /api/v1/users/{userId}/followers:
    get:
      summary: List user's followers

  /api/v1/users/{userId}/following:
    get:
      summary: List users this user follows

components:
  schemas:
    NotificationPreferences:
      type: object
      properties:
        email_enabled:
          type: boolean
        email_types:
          type: object
          additionalProperties:
            type: boolean
        digest_frequency:
          type: string
          enum: [immediate, hourly, daily, weekly]
```

## Testing

### Unit Test Example

```go
func TestNotificationService_NotifyNewFollower(t *testing.T) {
    mockNotifRepo := mocks.NewMockNotificationRepository(t)
    mockUserRepo := mocks.NewMockUserRepository(t)
    mockEmailSender := mocks.NewMockEmailNotifier(t)

    svc := NewNotificationService(mockNotifRepo, mockUserRepo, nil, nil, mockEmailSender, nil)

    follower := testUser("follower", "follower@test.com")
    followed := testUserWithEmailPrefs("followed", "followed@test.com", true)

    mockUserRepo.On("FindByID", mock.Anything, follower.ID()).Return(follower, nil)
    mockUserRepo.On("FindByID", mock.Anything, followed.ID()).Return(followed, nil)
    mockNotifRepo.On("Save", mock.Anything, mock.AnythingOfType("*Notification")).Return(nil)
    mockEmailSender.On("SendNewFollowerEmail", mock.Anything, followed.Email(), follower).Return(nil)

    event := UserFollowed{
        FollowerID: follower.ID(),
        FollowedID: followed.ID(),
        OccurredOn: time.Now(),
    }

    err := svc.NotifyNewFollower(context.Background(), event)
    require.NoError(t, err)

    mockNotifRepo.AssertCalled(t, "Save", mock.Anything, mock.AnythingOfType("*Notification"))
    mockEmailSender.AssertCalled(t, "SendNewFollowerEmail", mock.Anything, followed.Email(), follower)
}

func TestNotificationPreferences_ShouldEmail(t *testing.T) {
    tests := []struct {
        name          string
        prefs         NotificationPreferences
        notifType     NotificationType
        expected      bool
    }{
        {
            name:      "account banned always emails",
            prefs:     NotificationPreferences{emailEnabled: false},
            notifType: TypeAccountBanned,
            expected:  true,
        },
        {
            name:      "email disabled blocks optional notifications",
            prefs:     NotificationPreferences{emailEnabled: false},
            notifType: TypeNewFollower,
            expected:  false,
        },
        {
            name:      "email enabled sends by default",
            prefs:     NotificationPreferences{emailEnabled: true, emailTypes: map[NotificationType]bool{}},
            notifType: TypeNewFollower,
            expected:  true,
        },
        {
            name:      "per-type disable respected",
            prefs:     NotificationPreferences{emailEnabled: true, emailTypes: map[NotificationType]bool{TypeNewFollower: false}},
            notifType: TypeNewFollower,
            expected:  false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := tt.prefs.ShouldEmail(tt.notifType)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## Quick Reference

| Feature | User Control | Email Behavior |
|---------|--------------|----------------|
| New follower | Opt-in | Immediate or digest |
| New photos | Opt-in | Batched by uploader, respects digest frequency |
| Account suspended | None | Always sends |
| Account banned | None | Always sends |
| Account reinstated | None | Always sends |
| Abuse report (admin) | Opt-in | Immediate |

## See Also

- Architecture: `claude/architecture.md`
- API & Security: `claude/api_security.md`
- Testing: `claude/testing_ci.md`
