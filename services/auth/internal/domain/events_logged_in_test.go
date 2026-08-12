package domain

import (
	"testing"
	"time"
)

func TestUserLoggedIn_EventMetadata(t *testing.T) {
	t.Parallel()

	id, _ := ParseUserID("usr_test12345678901")
	email, _ := ParseEmail("login@example.com")
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	event := NewUserLoggedIn(id, email, now)
	if event.EventType() != "auth.user.logged_in" {
		t.Fatalf("event type = %q", event.EventType())
	}
	if !event.AggregateID().Equal(id) {
		t.Fatal("aggregate id mismatch")
	}
}
