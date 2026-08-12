package jobs

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/messaging"
)

// RetryConfig configures exponential backoff with jitter.
type RetryConfig struct {
	Initial    time.Duration
	MaxBackoff time.Duration
	Multiplier float64
	Jitter     float64 // fraction of delay used for jitter [0,1]
}

// JitterSource provides deterministic jitter in tests.
type JitterSource interface {
	Int63n(n int64) int64
}

var (
	globalRand   = rand.New(rand.NewSource(time.Now().UnixNano()))
	globalRandMu sync.Mutex
	testJitter   JitterSource
)

// SetTestJitter replaces jitter source for deterministic tests.
func SetTestJitter(src JitterSource) {
	testJitter = src
}

// ClearTestJitter restores production jitter.
func ClearTestJitter() {
	testJitter = nil
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Initial:    time.Second,
		MaxBackoff: 5 * time.Minute,
		Multiplier: 2,
		Jitter:     0.5,
	}
}

// BackoffDelay calculates bounded exponential backoff for an attempt (1-based).
func BackoffDelay(cfg RetryConfig, attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := messaging.Backoff(messaging.RetryConfig{
		Initial: cfg.Initial, MaxBackoff: cfg.MaxBackoff, Multiplier: cfg.Multiplier,
	}, attempt)
	if delay < 0 {
		delay = cfg.MaxBackoff
	}
	if delay > cfg.MaxBackoff {
		delay = cfg.MaxBackoff
	}
	return delay
}

// NextAvailableAt calculates the next retry time for an attempt.
func NextAvailableAt(cfg RetryConfig, attempt int, now time.Time) time.Time {
	delay := BackoffDelay(cfg, attempt)
	jitter := jitterDuration(cfg, delay)
	return now.Add(delay + jitter)
}

func jitterDuration(cfg RetryConfig, delay time.Duration) time.Duration {
	if cfg.Jitter <= 0 || delay <= 0 {
		return 0
	}
	maxJitter := int64(math.Round(float64(delay) * cfg.Jitter))
	if maxJitter <= 0 {
		return 0
	}
	var n int64
	if testJitter != nil {
		n = testJitter.Int63n(maxJitter + 1)
	} else {
		globalRandMu.Lock()
		n = globalRand.Int63n(maxJitter + 1)
		globalRandMu.Unlock()
	}
	return time.Duration(n)
}
