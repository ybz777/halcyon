package jobs

import (
	"testing"
	"time"
)

func BenchmarkBackoffDelay(b *testing.B) {
	cfg := DefaultRetryConfig()
	for i := 0; i < b.N; i++ {
		_ = BackoffDelay(cfg, 3)
	}
}

func BenchmarkNextAvailableAt(b *testing.B) {
	cfg := DefaultRetryConfig()
	now := time.Now()
	for i := 0; i < b.N; i++ {
		_ = NextAvailableAt(cfg, 3, now)
	}
}

func BenchmarkPolicyLookup(b *testing.B) {
	reg := NewPolicyRegistry()
	for i := 0; i < b.N; i++ {
		_ = reg.Lookup(TypeWebhookDelivery)
	}
}
