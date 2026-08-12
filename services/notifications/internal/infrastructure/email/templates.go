package email

import (
	"fmt"
	"strings"
	"time"

	"github.com/halcyon-io/halcyon/services/notifications/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
)

// Render builds a safe email message from a notification.
func Render(fromName, fromAddress string, n domain.Notification, recipientEmail string) ports.EmailMessage {
	template := templateForType(n.NotificationType)
	subject, plain, html := renderTemplate(template, fromName, n)
	_ = fromAddress
	return ports.EmailMessage{
		To:             recipientEmail,
		Subject:        subject,
		PlainTextBody:  plain,
		HTMLBody:       html,
		IdempotencyKey: n.ID.String(),
	}
}

func TemplateForType(t domain.NotificationType) string {
	return templateForType(t)
}

func templateForType(t domain.NotificationType) string {
	switch t {
	case domain.TypeSecurityLoginAlert:
		return "security_alert"
	case domain.TypeWorkflowExecutionFailed:
		return "workflow_failed"
	case domain.TypeWorkflowExecutionCompleted:
		return "workflow_completed"
	case domain.TypeSecurityAPIKeyCreated:
		return "api_key_created"
	case domain.TypeSecurityAPIKeyRevoked:
		return "api_key_revoked"
	case domain.TypeOrganizationRoleChanged, domain.TypeOrganizationMemberAdded, domain.TypeOrganizationMemberRemoved:
		return "membership_change"
	default:
		return "generic"
	}
}

func renderTemplate(template, productName string, n domain.Notification) (subject, plain, html string) {
	ts := n.CreatedAt.UTC().Format(time.RFC3339)
	subject = fmt.Sprintf("[%s] %s", productName, n.Title)
	plain = fmt.Sprintf("%s\n\n%s\n\nTime: %s\n", n.Title, n.Body, ts)
	if n.ResourceType != "" && n.ResourceID != "" {
		plain += fmt.Sprintf("Resource: %s\n", n.ResourceType)
	}
	html = fmt.Sprintf("<h2>%s</h2><p>%s</p><p><small>%s</small></p>",
		escapeHTML(n.Title), escapeHTML(n.Body), escapeHTML(ts))
	_ = template
	_ = strings.Builder{}
	return subject, plain, html
}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
