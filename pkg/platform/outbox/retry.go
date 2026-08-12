package outbox

import (
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/messaging"
)

// NextRetryAt calculates the next publish attempt time using exponential backoff.
func NextRetryAt(attempt int, now time.Time) time.Time {
	delay := messaging.Backoff(messaging.RetryConfig{
		MaxAttempts:  20,
		Initial:      time.Second,
		MaxBackoff:   5 * time.Minute,
		Multiplier:   2,
	}, attempt)
	return now.Add(delay)
}
