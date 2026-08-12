package messaging_test

import (
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/messaging"
)

func TestBackoff(t *testing.T) {
	t.Parallel()

	cfg := messaging.RetryConfig{
		MaxAttempts: 5,
		Initial:     100 * time.Millisecond,
		MaxBackoff:  1 * time.Second,
		Multiplier:  2,
	}

	if got := messaging.Backoff(cfg, 1); got != 100*time.Millisecond {
		t.Fatalf("attempt 1 = %v", got)
	}
	if got := messaging.Backoff(cfg, 3); got != 400*time.Millisecond {
		t.Fatalf("attempt 3 = %v", got)
	}
	if got := messaging.Backoff(cfg, 10); got != time.Second {
		t.Fatalf("attempt 10 = %v", got)
	}
}
