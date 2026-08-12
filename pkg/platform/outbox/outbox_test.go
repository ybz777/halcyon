package outbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/pkg/platform/outbox"
)

type memoryStore struct {
	entries []outbox.Entry
}

func (m *memoryStore) Insert(_ context.Context, entry outbox.Entry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func (m *memoryStore) RequeueStalePublishing(context.Context, time.Duration) error {
	return nil
}

func (m *memoryStore) ClaimPending(_ context.Context, limit int) ([]outbox.Entry, error) {
	var pending []outbox.Entry
	for _, e := range m.entries {
		if e.Status == outbox.StatusPending {
			e.Status = outbox.StatusPublishing
			pending = append(pending, e)
		}
		if len(pending) >= limit {
			break
		}
	}
	return pending, nil
}

func (m *memoryStore) MarkFailed(_ context.Context, id string, attempts int, lastError string, _ *time.Time) error {
	for i := range m.entries {
		if m.entries[i].ID == id {
			m.entries[i].Attempts = attempts
			m.entries[i].LastError = lastError
			m.entries[i].Status = outbox.StatusPending
			return nil
		}
	}
	return nil
}

func (m *memoryStore) MarkDeadLetter(_ context.Context, id string, lastError string) error {
	for i := range m.entries {
		if m.entries[i].ID == id {
			m.entries[i].Status = outbox.StatusFailed
			m.entries[i].LastError = lastError
			return nil
		}
	}
	return nil
}

func (m *memoryStore) MarkPublished(_ context.Context, id string, publishedAt time.Time) error {
	for i := range m.entries {
		if m.entries[i].ID == id {
			m.entries[i].Status = outbox.StatusPublished
			m.entries[i].PublishedAt = &publishedAt
			return nil
		}
	}
	return nil
}

func (m *memoryStore) PendingCount(_ context.Context) (int64, error) {
	var count int64
	for _, e := range m.entries {
		if e.Status == outbox.StatusPending {
			count++
		}
	}
	return count, nil
}

func TestPublisherStoresOutboxEntry(t *testing.T) {
	t.Parallel()

	store := &memoryStore{}
	pub := outbox.NewPublisher(store, "auth")

	record, err := events.BuildRecord(
		events.TypeUserLoggedIn,
		"usr_test12345678901",
		"user",
		events.SystemOrganizationID,
		time.Now().UTC(),
		events.UserLoggedInPayload{UserID: "usr_test12345678901", Email: "user@example.com"},
	)
	if err != nil {
		t.Fatalf("build record: %v", err)
	}

	if err := pub.Publish(context.Background(), record); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if len(store.entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(store.entries))
	}
	if store.entries[0].Topic != "halcyon.auth" {
		t.Fatalf("topic = %q", store.entries[0].Topic)
	}
	if store.entries[0].Status != outbox.StatusPending {
		t.Fatalf("status = %q", store.entries[0].Status)
	}
}
