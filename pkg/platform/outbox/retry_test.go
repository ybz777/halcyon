package outbox_test

import (
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/outbox"
)

func TestNextRetryAt_IncreasesWithAttempts(t *testing.T) {
	t.Parallel()

	first := outbox.NextRetryAt(1, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	second := outbox.NextRetryAt(2, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	if !second.After(first) {
		t.Fatalf("second retry %v should be after first %v", second, first)
	}
}
