package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/rs/zerolog"
)

// SMTPSender implements email sending via SMTP protocol.
// It supports both TLS and non-TLS connections, rate limiting, and configurable timeouts.
type SMTPSender struct {
	config  Config
	auth    smtp.Auth
	limiter *RateLimiter
	logger  zerolog.Logger
}

// NewSMTPSender creates a new SMTP email sender with the given configuration.
// Returns an error if the configuration is invalid.
func NewSMTPSender(cfg Config, logger zerolog.Logger) (*SMTPSender, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid smtp config: %w", err)
	}

	// Create SMTP auth if credentials provided
	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	// Create rate limiter
	limiter := NewRateLimiter(cfg.RateLimit, time.Hour)

	return &SMTPSender{
		config:  cfg,
		auth:    auth,
		limiter: limiter,
		logger:  logger,
	}, nil
}

// Send sends an email with both HTML and plain text versions.
// Returns ErrRateLimited if the rate limit has been exceeded.
// Returns ErrSMTPDisabled if SMTP is disabled in configuration.
func (s *SMTPSender) Send(ctx context.Context, to, subject, htmlBody, textBody string) error {
	if !s.config.Enabled {
		return ErrSMTPDisabled
	}

	// Check rate limit
	if !s.limiter.Allow() {
		s.logger.Warn().
			Str("recipient", to).
			Msg("email rate limit exceeded")
		return ErrRateLimited
	}

	// Build message
	msg := s.buildMessage(to, subject, htmlBody, textBody)

	// Send email
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	if s.config.UseTLS {
		return s.sendTLS(ctx, addr, to, msg)
	}

	return s.sendPlain(ctx, addr, to, msg)
}

// sendTLS sends email over a TLS connection.
func (s *SMTPSender) sendTLS(ctx context.Context, addr, to string, msg []byte) error {
	// Create TLS connection with timeout
	dialer := &net.Dialer{Timeout: s.config.Timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName: s.config.Host,
	})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	// Authenticate if credentials provided
	if s.auth != nil {
		if err := client.Auth(s.auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	// Send email
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

	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}

	s.logger.Info().
		Str("recipient", to).
		Msg("email sent successfully")

	return nil
}

// sendPlain sends email over a plain SMTP connection (no TLS).
// This should only be used for local development.
func (s *SMTPSender) sendPlain(ctx context.Context, addr, to string, msg []byte) error {
	err := smtp.SendMail(addr, s.auth, s.config.FromAddress, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("smtp send mail: %w", err)
	}

	s.logger.Info().
		Str("recipient", to).
		Msg("email sent successfully (plain)")

	return nil
}

// buildMessage constructs a MIME multipart email message with both HTML and plain text versions.
// This ensures compatibility with all email clients.
func (s *SMTPSender) buildMessage(to, subject, htmlBody, textBody string) []byte {
	boundary := "----=_Part_0_1234567890.1234567890"

	msg := ""
	msg += fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.FromAddress)
	msg += fmt.Sprintf("To: %s\r\n", to)
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "MIME-Version: 1.0\r\n"
	msg += fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary)
	msg += "\r\n"

	// Plain text version
	msg += fmt.Sprintf("--%s\r\n", boundary)
	msg += "Content-Type: text/plain; charset=utf-8\r\n"
	msg += "Content-Transfer-Encoding: 7bit\r\n"
	msg += "\r\n"
	msg += textBody
	msg += "\r\n\r\n"

	// HTML version
	msg += fmt.Sprintf("--%s\r\n", boundary)
	msg += "Content-Type: text/html; charset=utf-8\r\n"
	msg += "Content-Transfer-Encoding: 7bit\r\n"
	msg += "\r\n"
	msg += htmlBody
	msg += "\r\n\r\n"

	// End boundary
	msg += fmt.Sprintf("--%s--\r\n", boundary)

	return []byte(msg)
}

// SendNewFollowerEmail sends a notification when a user gains a new follower.
// This is a convenience method that uses a predefined template.
func (s *SMTPSender) SendNewFollowerEmail(ctx context.Context, recipientEmail, followerUsername string) error {
	subject := fmt.Sprintf("%s started following you", followerUsername)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>New Follower</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #4a5568;">New Follower!</h2>
        <p><strong>%s</strong> started following you on goimg Gallery.</p>
        <p>You can view their profile and photos on your feed.</p>
        <hr style="border: 1px solid #e2e8f0; margin: 20px 0;">
        <p style="font-size: 12px; color: #718096;">
            You received this email because you have email notifications enabled.
            You can change your notification preferences in your account settings.
        </p>
    </div>
</body>
</html>
`, followerUsername)

	textBody := fmt.Sprintf(`
New Follower!

%s started following you on goimg Gallery.

You can view their profile and photos on your feed.

---
You received this email because you have email notifications enabled.
You can change your notification preferences in your account settings.
`, followerUsername)

	return s.Send(ctx, recipientEmail, subject, htmlBody, textBody)
}

// IsEnabled returns true if SMTP sending is enabled.
func (s *SMTPSender) IsEnabled() bool {
	return s.config.Enabled
}

// RemainingQuota returns the number of emails that can still be sent in the current window.
func (s *SMTPSender) RemainingQuota() int {
	return s.limiter.Remaining()
}
