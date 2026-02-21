package identity

import "context"

type EmailSender interface {
	SendPasswordResetEmail(ctx context.Context, email, token string) error

	SendVerificationEmail(ctx context.Context, email, token string) error
}
