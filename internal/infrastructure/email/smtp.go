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

type SMTPSender struct {
	config  Config
	auth    smtp.Auth
	limiter *RateLimiter
	logger  zerolog.Logger
}

func NewSMTPSender(cfg Config, logger zerolog.Logger) (*SMTPSender, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid smtp config: %w", err)
	}

	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	limiter := NewRateLimiter(cfg.RateLimit, time.Hour)

	return &SMTPSender{
		config:  cfg,
		auth:    auth,
		limiter: limiter,
		logger:  logger,
	}, nil
}

func (s *SMTPSender) Send(ctx context.Context, to, subject, htmlBody, textBody string) error {
	if !s.config.Enabled {
		return ErrSMTPDisabled
	}

	if !s.limiter.Allow() {
		s.logger.Warn().
			Str("recipient", to).
			Msg("email rate limit exceeded")
		return ErrRateLimited
	}

	msg := s.buildMessage(to, subject, htmlBody, textBody)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	if s.config.UseTLS {
		return s.sendTLS(ctx, addr, to, msg)
	}

	return s.sendPlain(ctx, addr, to, msg)
}

func (s *SMTPSender) sendTLS(ctx context.Context, addr, to string, msg []byte) error {
	dialer := &net.Dialer{Timeout: s.config.Timeout}
	//nolint:gosec // G402: TLS MinVersion is explicitly set to TLS 1.2
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName: s.config.Host,
		MinVersion: tls.VersionTLS12,
	})
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

	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}

	s.logger.Info().
		Str("recipient", to).
		Msg("email sent successfully")

	return nil
}

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

func (s *SMTPSender) buildMessage(to, subject, htmlBody, textBody string) []byte {
	boundary := "----=_Part_0_1234567890.1234567890"

	msg := ""
	msg += fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.FromAddress)
	msg += fmt.Sprintf("To: %s\r\n", to)
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "MIME-Version: 1.0\r\n"
	msg += fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary)
	msg += "\r\n"

	msg += fmt.Sprintf("--%s\r\n", boundary)
	msg += "Content-Type: text/plain; charset=utf-8\r\n"
	msg += "Content-Transfer-Encoding: 7bit\r\n"
	msg += "\r\n"
	msg += textBody
	msg += "\r\n\r\n"

	msg += fmt.Sprintf("--%s\r\n", boundary)
	msg += "Content-Type: text/html; charset=utf-8\r\n"
	msg += "Content-Transfer-Encoding: 7bit\r\n"
	msg += "\r\n"
	msg += htmlBody
	msg += "\r\n\r\n"

	msg += fmt.Sprintf("--%s--\r\n", boundary)

	return []byte(msg)
}

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

func (s *SMTPSender) SendMalwareDetectedEmail(ctx context.Context, recipientEmail, username, filename string) error {
	subject := "Security Alert: Malware Detected in Your Upload"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Malware Detected</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #e53e3e;">Security Alert</h2>
        <p>Hello <strong>%s</strong>,</p>
        <p>We detected that an image you uploaded contains malware and has been removed.</p>
        <div style="background-color: #fff5f5; border-left: 4px solid #e53e3e; padding: 15px; margin: 20px 0;">
            <p style="margin: 0;"><strong>File:</strong> %s</p>
            <p style="margin: 0;"><strong>Status:</strong> Removed</p>
        </div>
        <p>Please ensure your files are safe before uploading. Repeated violations may result in account suspension.</p>
        <hr style="border: 1px solid #e2e8f0; margin: 20px 0;">
        <p style="font-size: 12px; color: #718096;">
            This is an automated security notification.
        </p>
    </div>
</body>
</html>
`, username, filename)

	textBody := fmt.Sprintf(`
Security Alert: Malware Detected

Hello %s,

We detected that an image you uploaded contains malware and has been removed.

File: %s
Status: Removed

Please ensure your files are safe before uploading. Repeated violations may result in account suspension.

---
This is an automated security notification.
`, username, filename)

	return s.Send(ctx, recipientEmail, subject, htmlBody, textBody)
}

func (s *SMTPSender) SendPasswordResetEmail(ctx context.Context, recipientEmail, token string) error {
	subject := "Reset your password"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Password Reset</title></head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #4a5568;">Reset Your Password</h2>
        <p>We received a request to reset the password for your goimg account.</p>
        <p>Use the following token to reset your password. It expires in 1 hour.</p>
        <div style="background-color: #f7fafc; border: 1px solid #e2e8f0; padding: 15px; margin: 20px 0; word-break: break-all;">
            <code>%s</code>
        </div>
        <p>If you did not request a password reset, you can safely ignore this email.</p>
        <hr style="border: 1px solid #e2e8f0; margin: 20px 0;">
        <p style="font-size: 12px; color: #718096;">This link expires in 1 hour.</p>
    </div>
</body>
</html>
`, token)

	textBody := fmt.Sprintf(`Reset Your Password

We received a request to reset your password.

Your password reset token (expires in 1 hour):
%s

If you did not request a password reset, ignore this email.
`, token)

	return s.Send(ctx, recipientEmail, subject, htmlBody, textBody)
}

func (s *SMTPSender) SendVerificationEmail(ctx context.Context, recipientEmail, token string) error {
	subject := "Verify your email address"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Verify Email</title></head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #4a5568;">Verify Your Email</h2>
        <p>Thank you for registering with goimg. Please verify your email address.</p>
        <div style="background-color: #f7fafc; border: 1px solid #e2e8f0; padding: 15px; margin: 20px 0; word-break: break-all;">
            <code>%s</code>
        </div>
        <p>If you did not create an account, you can safely ignore this email.</p>
        <hr style="border: 1px solid #e2e8f0; margin: 20px 0;">
        <p style="font-size: 12px; color: #718096;">This token expires in 24 hours.</p>
    </div>
</body>
</html>
`, token)

	textBody := fmt.Sprintf(`Verify Your Email

Please verify your email address using this token:
%s

If you did not create an account, ignore this email.
`, token)

	return s.Send(ctx, recipientEmail, subject, htmlBody, textBody)
}

func (s *SMTPSender) IsEnabled() bool {
	return s.config.Enabled
}

func (s *SMTPSender) RemainingQuota() int {
	return s.limiter.Remaining()
}
