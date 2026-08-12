package preference_test

import (
	"context"
	"testing"

	apppreference "github.com/halcyon-io/halcyon/services/notifications/internal/application/preference"
	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
)

type stubPrefs struct{}

func (stubPrefs) ListByUser(context.Context, string, int, string) ([]domain.NotificationPreference, string, error) {
	return nil, "", nil
}
func (stubPrefs) Upsert(context.Context, domain.NotificationPreference) error { return nil }

func TestSecurityEmailCannotBeDisabled(t *testing.T) {
	svc := apppreference.NewService(stubPrefs{})
	_, err := svc.Update(context.Background(), "usr_1", domain.TypeSecurityLoginAlert, domain.ChannelEmail, false)
	if err == nil {
		t.Fatal("expected error disabling security email")
	}
}

func TestDefaultPreferenceSecurityEnabled(t *testing.T) {
	if !domain.DefaultPreference(domain.TypeSecurityAPIKeyCreated, domain.ChannelEmail) {
		t.Fatal("security email should default enabled")
	}
}
