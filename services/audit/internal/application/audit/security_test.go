package audit_test

import (
	"strings"
	"testing"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	appaudit "github.com/halcyon-io/halcyon/services/audit/internal/application/audit"
	"github.com/halcyon-io/halcyon/services/audit/internal/domain"
)

func TestJWTNeverInMetadata(t *testing.T) {
	raw := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature"
	out := domain.SanitizeMetadata(map[string]string{"note": raw})
	if len(out) != 0 && strings.Contains(out["note"], "eyJ") {
		t.Fatal("jwt-like value must not be stored")
	}
}

func TestPasswordNeverInMetadata(t *testing.T) {
	out := domain.SanitizeMetadata(map[string]string{"password": "hunter2", "reason": "invalid_credentials"})
	if _, ok := out["password"]; ok {
		t.Fatal("password must not be stored")
	}
}

func TestSecurityEventMapping(t *testing.T) {
	payload, _ := platformevents.MarshalPayload(platformevents.TypeSecurityInvalidToken, platformevents.SecurityInvalidTokenPayload{
		Reason: "expired_token", IPAddress: "203.0.113.1",
	})
	envelope := platformevents.Envelope{
		EventType: platformevents.TypeSecurityInvalidToken,
		Payload:   payload,
	}
	event, ok, err := appaudit.MapEnvelope(envelope, "gateway")
	if err != nil || !ok {
		t.Fatalf("map security event: ok=%v err=%v", ok, err)
	}
	if event.EventType != domain.EventSecurityInvalidToken {
		t.Fatalf("unexpected event type %s", event.EventType)
	}
	if strings.Contains(event.UserAgent, "Bearer") {
		t.Fatal("authorization material must not appear")
	}
}
