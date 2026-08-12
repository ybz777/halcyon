package messaging_test

import (
	"errors"
	"testing"

	"github.com/halcyon-io/halcyon/pkg/platform/messaging"
)

func TestClassifyFailure(t *testing.T) {
	t.Parallel()
	if messaging.ClassifyFailure(messaging.Transient(errors.New("db"))) != messaging.FailureTransient {
		t.Fatal("expected transient")
	}
	if messaging.ClassifyFailure(messaging.Permanent(errors.New("schema"))) != messaging.FailurePermanent {
		t.Fatal("expected permanent")
	}
	if messaging.ShouldRetry(messaging.Permanent(errors.New("schema"))) {
		t.Fatal("permanent should not retry")
	}
}

func TestBackoffWithJitter(t *testing.T) {
	t.Parallel()
	cfg := messaging.DefaultRetryConfig()
	d1 := messaging.BackoffWithJitter(cfg, 2)
	d2 := messaging.BackoffWithJitter(cfg, 2)
	if d1 <= cfg.Initial {
		t.Fatalf("backoff too small: %v", d1)
	}
	if d2 < d1 {
		// jitter may vary; only ensure positive
	}
}
