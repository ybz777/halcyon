package ports

import "context"

// EmailMessage is a rendered outbound email.
type EmailMessage struct {
	To              string
	Subject         string
	PlainTextBody   string
	HTMLBody        string
	IdempotencyKey  string
}

// EmailResult captures provider response metadata.
type EmailResult struct {
	ProviderMessageID string
	Retryable         bool
	ErrorSummary      string
}

// EmailSender delivers email via a configured provider.
type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) (EmailResult, error)
	ProviderName() string
}
