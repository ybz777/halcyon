package idempotency

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements Store using Redis atomic operations.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a Redis-backed idempotency store.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func (s *RedisStore) Claim(ctx context.Context, key, fingerprint string, inProgressTTL time.Duration) (ClaimOutcome, error) {
	rec := Record{
		State:       StateInProgress,
		Fingerprint: fingerprint,
		CreatedAt:   time.Now().UTC(),
	}
	payload, err := encodeRecord(rec)
	if err != nil {
		return ClaimOutcome{}, err
	}

	// Fast path: first claim avoids a GET round trip on new keys.
	ok, err := s.client.SetNX(ctx, key, payload, inProgressTTL).Result()
	if err != nil {
		return ClaimOutcome{}, fmt.Errorf("%w: %v", ErrStoreUnavailable, err)
	}
	if ok {
		return ClaimOutcome{Result: ClaimAcquired, Record: rec}, nil
	}

	existing, err := s.client.Get(ctx, key).Bytes()
	if err != nil && err != redis.Nil {
		return ClaimOutcome{}, fmt.Errorf("%w: %v", ErrStoreUnavailable, err)
	}
	if err == redis.Nil {
		return s.Claim(ctx, key, fingerprint, inProgressTTL)
	}

	stored, err := decodeRecord(existing)
	if err != nil {
		return ClaimOutcome{}, fmt.Errorf("%w: corrupt record", ErrStoreUnavailable)
	}
	if stored.Fingerprint != fingerprint {
		return ClaimOutcome{Result: ClaimConflict, Record: stored}, nil
	}
	if stored.State == StateCompleted {
		return ClaimOutcome{Result: ClaimReplay, Record: stored}, nil
	}
	return ClaimOutcome{Result: ClaimInProgress, Record: stored}, nil
}

func (s *RedisStore) Complete(ctx context.Context, key, fingerprint string, record Record, completedTTL time.Duration) error {
	current, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStoreUnavailable, err)
	}
	existing, err := decodeRecord(current)
	if err != nil {
		return fmt.Errorf("%w: corrupt record", ErrStoreUnavailable)
	}
	if existing.Fingerprint != fingerprint {
		return ErrFingerprintMismatch
	}
	record.State = StateCompleted
	record.Fingerprint = fingerprint
	record.CompletedAt = time.Now().UTC()
	payload, err := encodeRecord(record)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, payload, completedTTL).Err()
}

func (s *RedisStore) Release(ctx context.Context, key, fingerprint string) error {
	current, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStoreUnavailable, err)
	}
	rec, err := decodeRecord(current)
	if err != nil {
		_ = s.client.Del(ctx, key).Err()
		return nil
	}
	if rec.State == StateInProgress && rec.Fingerprint == fingerprint {
		return s.client.Del(ctx, key).Err()
	}
	return nil
}
