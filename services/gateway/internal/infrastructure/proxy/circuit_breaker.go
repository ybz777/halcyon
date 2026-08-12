package proxy

import (
	"sync"
	"time"
)

// State represents circuit breaker state.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker protects upstream services from overload.
type CircuitBreaker struct {
	threshold   int
	openFor     time.Duration
	mu          sync.Mutex
	state       State
	failures    int
	openedAt    time.Time
	onOpen      func()
}

// NewCircuitBreaker creates a circuit breaker.
func NewCircuitBreaker(threshold int, openFor time.Duration, onOpen func()) *CircuitBreaker {
	return &CircuitBreaker{threshold: threshold, openFor: openFor, onOpen: onOpen}
}

// Allow reports whether a request may proceed.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case StateOpen:
		if time.Since(cb.openedAt) >= cb.openFor {
			cb.state = StateHalfOpen
			return true
		}
		return false
	default:
		return true
	}
}

// RecordSuccess resets failure tracking.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = StateClosed
}

// RecordFailure increments failures and may open the circuit.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	if cb.failures >= cb.threshold {
		if cb.state != StateOpen {
			cb.state = StateOpen
			cb.openedAt = time.Now()
			if cb.onOpen != nil {
				cb.onOpen()
			}
		}
	}
}

// StateValue returns the current state for metrics.
func (cb *CircuitBreaker) StateValue() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}
