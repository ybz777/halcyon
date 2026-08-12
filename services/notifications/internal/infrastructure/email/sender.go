package email

import (
	"context"
	"log/slog"

	"github.com/halcyon-io/halcyon/services/notifications/internal/application/ports"
)

// LogSender is a development-safe email sink that never sends real mail.
type LogSender struct {
	provider string
}

func NewLogSender() *LogSender {
	return &LogSender{provider: "log"}
}

func (s *LogSender) ProviderName() string { return s.provider }

func (s *LogSender) Send(ctx context.Context, msg ports.EmailMessage) (ports.EmailResult, error) {
	slog.InfoContext(ctx, "email_sink_delivery",
		"to_domain", maskDomain(msg.To),
		"subject", msg.Subject,
		"idempotency_key", msg.IdempotencyKey,
	)
	return ports.EmailResult{ProviderMessageID: "log-" + msg.IdempotencyKey}, nil
}

func maskDomain(email string) string {
	for i := len(email) - 1; i >= 0; i-- {
		if email[i] == '@' {
			if i+1 < len(email) {
				return email[i+1:]
			}
			return ""
		}
	}
	return ""
}

func NewSender(provider string) ports.EmailSender {
	switch provider {
	case "", "log", "sink":
		return NewLogSender()
	default:
		return NewLogSender()
	}
}
