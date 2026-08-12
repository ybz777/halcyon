package audit_test

import (
	"testing"
	"time"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	appaudit "github.com/halcyon-io/halcyon/services/audit/internal/application/audit"
	"github.com/halcyon-io/halcyon/services/audit/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/audit/internal/domain"
)

func TestMapUserLoggedIn(t *testing.T) {
	payload, err := platformevents.MarshalPayload(platformevents.TypeUserLoggedIn, platformevents.UserLoggedInPayload{
		UserID: "usr_123", Email: "user@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	envelope := platformevents.Envelope{
		EventType: platformevents.TypeUserLoggedIn, OrganizationID: "org_abc",
		ActorID: "usr_123", Payload: payload, OccurredAt: time.Now().UTC(),
	}
	event, ok, err := appaudit.MapEnvelope(envelope, "auth")
	if err != nil || !ok {
		t.Fatalf("map envelope: ok=%v err=%v", ok, err)
	}
	if event.EventType != domain.EventAuthLoginSucceeded {
		t.Fatalf("event type = %s", event.EventType)
	}
	if event.Metadata["password"] != "" {
		t.Fatal("password should not appear in metadata")
	}
}

func TestSanitizeMetadataRemovesSecrets(t *testing.T) {
	out := domain.SanitizeMetadata(map[string]string{
		"password": "secret", "reason": "invalid_credentials", "token": "abc",
	})
	if _, ok := out["password"]; ok {
		t.Fatal("password must be removed")
	}
	if out["reason"] != "invalid_credentials" {
		t.Fatal("safe metadata should remain")
	}
}

func TestCursorRoundTrip(t *testing.T) {
	at := time.Now().UTC().Truncate(time.Microsecond)
	cursor := domain.EncodeCursor(at, "aud_test123")
	gotAt, id, err := domain.DecodeCursor(cursor)
	if err != nil {
		t.Fatal(err)
	}
	if !gotAt.Equal(at) || id != "aud_test123" {
		t.Fatalf("cursor mismatch: %v %s", gotAt, id)
	}
}

func TestAuthorizerBlocksCrossTenant(t *testing.T) {
	authz := appaudit.NewAuthorizer()
	principal := ports.Principal{Permissions: map[string]struct{}{ports.PermissionAuditRead: {}}}
	_, err := authz.ResolveOrganizationScope(principal, "org_a", "org_b")
	if err == nil {
		t.Fatal("expected cross-tenant denial")
	}
}

func TestAuthorizerBlocksSecurityEvents(t *testing.T) {
	authz := appaudit.NewAuthorizer()
	principal := ports.Principal{Permissions: map[string]struct{}{ports.PermissionAuditRead: {}}}
	event := domain.AuditEvent{EventType: domain.EventSecurityInvalidToken, OrganizationID: "org_a"}
	if err := authz.CanViewEvent(principal, "org_a", event); err == nil {
		t.Fatal("expected security event denial")
	}
}

func TestNormalizePageSize(t *testing.T) {
	if domain.NormalizePageSize(0) != domain.DefaultPageSize {
		t.Fatal("default page size")
	}
	if domain.NormalizePageSize(1_000_000) != domain.MaxPageSize {
		t.Fatal("max page size")
	}
}
