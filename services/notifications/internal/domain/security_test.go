package domain_test

import (
	"strings"
	"testing"

	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/email"
)

func TestEmailTemplateExcludesSecrets(t *testing.T) {
	n := domain.Notification{
		NotificationType: domain.TypeSecurityAPIKeyCreated,
		Title:            "API key created",
		Body:             "A new API key was created for your organization.",
	}
	msg := email.Render("Halcyon", "noreply@test.com", n, "user@example.com")
	if strings.Contains(msg.Subject, "Bearer ") || strings.Contains(msg.PlainTextBody, "whsec_") {
		t.Fatal("email must not include sensitive token material")
	}
}
