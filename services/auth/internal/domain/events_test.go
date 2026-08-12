package domain

import (
	"testing"
	"time"
)

func TestUserRegistered_EventMetadata(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	id, _ := ParseUserID("usr_abcdefghijklmnopqrst")
	email, _ := ParseEmail("user@example.com")

	event := NewUserRegistered(id, email, now)

	if event.EventType() != "auth.user.registered" {
		t.Errorf("EventType() = %q", event.EventType())
	}
	if !event.AggregateID().Equal(id) {
		t.Error("AggregateID() mismatch")
	}
	if !event.OccurredAt().Equal(now) {
		t.Error("OccurredAt() mismatch")
	}
}

func TestUserEmailChanged_EventType(t *testing.T) {
	t.Parallel()

	event := UserEmailChanged{}
	if event.EventType() != "auth.user.email_changed" {
		t.Errorf("EventType() = %q", event.EventType())
	}
}

func TestUserDisabled_EventType(t *testing.T) {
	t.Parallel()

	event := UserDisabled{}
	if event.EventType() != "auth.user.disabled" {
		t.Errorf("EventType() = %q", event.EventType())
	}
}

func TestUserEnabled_EventType(t *testing.T) {
	t.Parallel()

	event := UserEnabled{}
	if event.EventType() != "auth.user.enabled" {
		t.Errorf("EventType() = %q", event.EventType())
	}
}
