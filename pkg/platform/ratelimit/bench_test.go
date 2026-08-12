package ratelimit

import (
	"context"
	"testing"
	"time"
)

func BenchmarkAllow_Disabled(b *testing.B) {
	var limiter DisabledLimiter
	policy := Policy{Name: "api", Limit: 1000, Window: 0}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.Allow(context.Background(), "ip:bench", policy)
	}
}

func BenchmarkAllow_Memory(b *testing.B) {
	limiter := NewMemoryLimiter()
	policy := Policy{Name: "api", Limit: 1_000_000, Window: time.Second}
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = limiter.Allow(context.Background(), "ip:bench", policy)
		}
	})
}
