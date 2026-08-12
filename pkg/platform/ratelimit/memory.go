package ratelimit

import (
	"context"
	"sync"
	"time"
)

type memoryEntry struct {
	count   int64
	expires time.Time
}

// MemoryLimiter is an in-memory fixed-window limiter for tests.
type MemoryLimiter struct {
	mu    sync.Mutex
	clock Clock
	store map[string]memoryEntry
}

// NewMemoryLimiter creates a test limiter.
func NewMemoryLimiter() *MemoryLimiter {
	return &MemoryLimiter{clock: systemClock{}, store: make(map[string]memoryEntry)}
}

func (m *MemoryLimiter) Allow(_ context.Context, key string, policy Policy) (Decision, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.clock.Now()
	entry, ok := m.store[key]
	if !ok || now.After(entry.expires) {
		entry = memoryEntry{count: 0, expires: now.Add(policy.Window)}
	}
	entry.count++
	m.store[key] = entry

	remaining := policy.Limit - entry.count
	if remaining < 0 {
		remaining = 0
	}
	retryAfter := time.Until(entry.expires)
	if retryAfter < 0 {
		retryAfter = 0
	}
	return Decision{
		Allowed:    entry.count <= policy.Limit,
		Remaining:  remaining,
		ResetAt:    entry.expires,
		RetryAfter: retryAfter,
	}, nil
}

// AllowAll checks multiple keys against the same policy atomically.
// When any key exceeds the limit, prior increments in the batch are rolled back.
func (m *MemoryLimiter) AllowAll(ctx context.Context, keys []string, policy Policy) (Decision, error) {
	if len(keys) == 0 {
		return Decision{Allowed: true, Remaining: policy.Limit}, nil
	}
	if len(keys) == 1 {
		return m.Allow(ctx, keys[0], policy)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.clock.Now()
	snapshots := make([]memoryEntry, len(keys))
	for i, key := range keys {
		snapshots[i] = m.store[key]
		entry := snapshots[i]
		if !entry.expires.IsZero() && now.After(entry.expires) {
			entry = memoryEntry{}
		}
		if entry.expires.IsZero() {
			entry.expires = now.Add(policy.Window)
		}
		entry.count++
		m.store[key] = entry
		if entry.count > policy.Limit {
			for j := 0; j <= i; j++ {
				m.store[keys[j]] = snapshots[j]
			}
			retryAfter := time.Until(entry.expires)
			if retryAfter < 0 {
				retryAfter = 0
			}
			return Decision{
				Allowed:    false,
				Remaining:  0,
				ResetAt:    entry.expires,
				RetryAfter: retryAfter,
			}, nil
		}
	}

	worst := Decision{Allowed: true, Remaining: policy.Limit}
	for _, key := range keys {
		entry := m.store[key]
		remaining := policy.Limit - entry.count
		if remaining < 0 {
			remaining = 0
		}
		if worst.Remaining < 0 || remaining < worst.Remaining {
			retryAfter := time.Until(entry.expires)
			if retryAfter < 0 {
				retryAfter = 0
			}
			worst = Decision{
				Allowed:    true,
				Remaining:  remaining,
				ResetAt:    entry.expires,
				RetryAfter: retryAfter,
			}
		}
	}
	return worst, nil
}
