package email

import "errors"

var (
	// ErrRateLimited is returned when the rate limit for email sending has been exceeded.
	ErrRateLimited = errors.New("email rate limit exceeded")

	// ErrSMTPNotConfigured is returned when attempting to send email with incomplete configuration.
	ErrSMTPNotConfigured = errors.New("smtp not configured")

	// ErrSMTPDisabled is returned when attempting to send email but SMTP is disabled.
	ErrSMTPDisabled = errors.New("smtp is disabled")

	// ErrInvalidRecipient is returned when the recipient email address is invalid.
	ErrInvalidRecipient = errors.New("invalid recipient email address")

	// ErrTemplateMissing is returned when a required email template is not found.
	ErrTemplateMissing = errors.New("email template not found")

	// ErrTemplateRender is returned when template rendering fails.
	ErrTemplateRender = errors.New("failed to render email template")
)
