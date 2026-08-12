package postgres_test

import (
	"testing"

	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
)

func TestNotificationUnread(t *testing.T) {
	n := domain.Notification{}
	if !n.IsUnread() {
		t.Fatal("expected unread by default")
	}
}

func TestMandatorySecurityCannotDisableEmail(t *testing.T) {
	prefs := map[domain.Channel]bool{domain.ChannelEmail: false}
	if !domain.IsChannelEnabled(domain.TypeSecurityLoginAlert, domain.ChannelEmail, prefs) {
		t.Fatal("security email must remain enabled")
	}
}
