package proxy_test

import (
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/proxy"
)

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	t.Parallel()
	opened := false
	cb := proxy.NewCircuitBreaker(2, time.Minute, func() { opened = true })
	if !cb.Allow() {
		t.Fatal("expected closed circuit")
	}
	cb.RecordFailure()
	if !cb.Allow() {
		t.Fatal("expected still closed after one failure")
	}
	cb.RecordFailure()
	if cb.Allow() {
		t.Fatal("expected open circuit")
	}
	if !opened {
		t.Fatal("expected onOpen callback")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	t.Parallel()
	cb := proxy.NewCircuitBreaker(1, 10*time.Millisecond, nil)
	cb.RecordFailure()
	time.Sleep(15 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("expected half-open allow")
	}
}
