package events_test

import (
	"testing"
	"time"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	infraevents "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/events"
)

func TestFromDomainEvent_UserLoggedIn(t *testing.T) {
	t.Parallel()

	userID, _ := domain.ParseUserID("usr_test12345678901")
	email, _ := domain.ParseEmail("user@example.com")
	event := domain.NewUserLoggedIn(userID, email, time.Now().UTC())

	record, err := infraevents.FromDomainEvent(event, "")
	if err != nil {
		t.Fatalf("FromDomainEvent() error = %v", err)
	}
	if record.Type != platformevents.TypeUserLoggedIn {
		t.Fatalf("type = %q", record.Type)
	}
}

func TestAPIKeyCreatedRecord(t *testing.T) {
	t.Parallel()

	id, _ := domain.NewAPIKeyID()
	keyID, _ := domain.NewKeyID()
	saID, err := domain.NewServiceAccountID()
	if err != nil {
		t.Fatalf("NewServiceAccountID() error = %v", err)
	}
	creator, _ := domain.ParseUserID("usr_test12345678901")
	now := time.Now().UTC()

	hash, _ := domain.ParsePasswordHash("$argon2id$v=19$m=65536,t=1,p=4$c2FsdHNhbHQ$hashvaluehashvaluehashvaluehashvaluehashvaluehashval")
	key, err := domain.NewAPIKey(id, keyID, saID, "CI", "hal_live_abcd", hash,
		[]domain.APIScope{domain.ScopeUsersRead}, creator, now, nil)
	if err != nil {
		t.Fatalf("NewAPIKey() error = %v", err)
	}

	record, err := infraevents.APIKeyCreatedRecord(key, "org_test12345678901")
	if err != nil {
		t.Fatalf("APIKeyCreatedRecord() error = %v", err)
	}
	if record.Type != platformevents.TypeAPIKeyCreated {
		t.Fatalf("type = %q", record.Type)
	}
}
