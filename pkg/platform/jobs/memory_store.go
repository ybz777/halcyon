package jobs

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// MemoryStore is an in-memory Repository for unit and concurrency tests.
type MemoryStore struct {
	mu   sync.Mutex
	jobs map[string]Job
	dlq  []DeadLetterRecord
}

// DeadLetterRecord captures a dead-lettered job for test assertions.
type DeadLetterRecord struct {
	JobID          ID
	OrganizationID string
	JobType        Type
	Category       ErrorCategory
	LastError      string
	AttemptCount   int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{jobs: make(map[string]Job)}
}

func (s *MemoryStore) Create(ctx context.Context, input CreateInput) (Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if input.IdempotencyKey != "" {
		for _, j := range s.jobs {
			if j.IdempotencyKey == input.IdempotencyKey && j.Type == input.Type && j.OrganizationID == input.OrganizationID {
				return j, true, nil
			}
		}
	}

	id, err := NewID()
	if err != nil {
		return Job{}, false, err
	}
	now := time.Now().UTC()
	if input.MaxAttempts <= 0 {
		input.MaxAttempts = 6
	}
	if input.Priority == "" {
		input.Priority = PriorityNormal
	}
	if input.AvailableAt.IsZero() {
		input.AvailableAt = now
	}
	payload := input.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	job := Job{
		ID: id, OrganizationID: input.OrganizationID, Type: input.Type,
		Status: StatusPending, Payload: payload, Priority: input.Priority,
		MaxAttempts: input.MaxAttempts, IdempotencyKey: input.IdempotencyKey,
		AvailableAt: input.AvailableAt.UTC(), CreatedAt: now,
	}
	s.jobs[id.String()] = job
	return job, false, nil
}

func (s *MemoryStore) Get(ctx context.Context, id ID) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id.String()]
	if !ok {
		return Job{}, ErrJobNotFound
	}
	return job, nil
}

func (s *MemoryStore) GetForOrg(ctx context.Context, orgID string, id ID) (Job, error) {
	job, err := s.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if job.OrganizationID != orgID {
		return Job{}, ErrJobNotFound
	}
	return job, nil
}

func (s *MemoryStore) List(ctx context.Context, orgID string, limit int, cursor string) ([]Job, string, error) {
	return nil, "", nil
}

func (s *MemoryStore) ListPlatform(ctx context.Context, limit int, cursor string) ([]Job, string, error) {
	return nil, "", nil
}

func (s *MemoryStore) Claim(ctx context.Context, workerID string, types []Type, limit int, lease time.Duration) ([]Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	leaseUntil := now.Add(lease)
	typeSet := make(map[Type]struct{}, len(types))
	for _, t := range types {
		typeSet[t] = struct{}{}
	}

	var claimed []Job
	for key, job := range s.jobs {
		if len(claimed) >= limit {
			break
		}
		if len(types) > 0 {
			if _, ok := typeSet[job.Type]; !ok {
				continue
			}
		}
		if job.Status != StatusPending && job.Status != StatusRetryWait {
			continue
		}
		if job.AvailableAt.After(now) {
			continue
		}
		if job.LockedAt != nil && job.LockedAt.After(now) {
			continue
		}
		job.Status = StatusRunning
		job.AttemptCount++
		job.LockedAt = &leaseUntil
		job.LockedBy = workerID
		if job.StartedAt == nil {
			job.StartedAt = &now
		}
		s.jobs[key] = job
		claimed = append(claimed, job)
	}
	return claimed, nil
}

func (s *MemoryStore) Complete(ctx context.Context, workerID string, id ID, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id.String()]
	if !ok || job.Status != StatusRunning || job.LockedBy != workerID {
		return ErrStaleLease
	}
	job.Status = StatusSucceeded
	job.CompletedAt = &at
	job.LockedAt = nil
	job.LockedBy = ""
	job.LastError = ""
	s.jobs[id.String()] = job
	return nil
}

func (s *MemoryStore) Retry(ctx context.Context, workerID string, id ID, next time.Time, lastError string, category ErrorCategory) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id.String()]
	if !ok || job.Status != StatusRunning || job.LockedBy != workerID {
		return ErrStaleLease
	}
	job.Status = StatusRetryWait
	job.AvailableAt = next.UTC()
	job.LockedAt = nil
	job.LockedBy = ""
	job.LastError = lastError
	s.jobs[id.String()] = job
	return nil
}

func (s *MemoryStore) Fail(ctx context.Context, workerID string, id ID, lastError string, category ErrorCategory, at time.Time, deadLetter bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id.String()]
	if !ok || job.Status != StatusRunning || job.LockedBy != workerID {
		return ErrStaleLease
	}
	if deadLetter {
		job.Status = StatusDeadLettered
		s.dlq = append(s.dlq, DeadLetterRecord{
			JobID: job.ID, OrganizationID: job.OrganizationID, JobType: job.Type,
			Category: category, LastError: lastError, AttemptCount: job.AttemptCount,
		})
	} else {
		job.Status = StatusFailed
	}
	job.FailedAt = &at
	job.CompletedAt = &at
	job.LockedAt = nil
	job.LockedBy = ""
	job.LastError = lastError
	s.jobs[id.String()] = job
	return nil
}

func (s *MemoryStore) Cancel(ctx context.Context, id ID, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id.String()]
	if !ok {
		return ErrJobNotFound
	}
	job.Status = StatusCancelled
	job.CompletedAt = &at
	job.LockedAt = nil
	job.LockedBy = ""
	s.jobs[id.String()] = job
	return nil
}

func (s *MemoryStore) RecoverLeases(ctx context.Context, lease time.Duration, cfg RetryConfig) (map[Type]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().UTC().Add(-lease)
	counts := make(map[Type]int64)
	for key, job := range s.jobs {
		if job.Status != StatusRunning || job.LockedAt == nil || !job.LockedAt.Before(cutoff) {
			continue
		}
		if job.AttemptCount >= job.MaxAttempts {
			now := time.Now().UTC()
			job.Status = StatusDeadLettered
			job.FailedAt = &now
			job.CompletedAt = &now
			job.LastError = "lease expired"
			s.dlq = append(s.dlq, DeadLetterRecord{
				JobID: job.ID, OrganizationID: job.OrganizationID, JobType: job.Type,
				Category: CategoryTimeout, LastError: job.LastError, AttemptCount: job.AttemptCount,
			})
		} else {
			job.Status = StatusRetryWait
			job.AvailableAt = NextAvailableAt(cfg, job.AttemptCount, time.Now().UTC())
		}
		job.LockedAt = nil
		job.LockedBy = ""
		s.jobs[key] = job
		counts[job.Type]++
	}
	return counts, nil
}

func (s *MemoryStore) RenewLease(ctx context.Context, workerID string, id ID, lease time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id.String()]
	if !ok || job.Status != StatusRunning || job.LockedBy != workerID {
		return ErrStaleLease
	}
	until := time.Now().UTC().Add(lease)
	job.LockedAt = &until
	s.jobs[id.String()] = job
	return nil
}

func (s *MemoryStore) CountReady(ctx context.Context, jobType Type) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var count int64
	now := time.Now().UTC()
	for _, job := range s.jobs {
		if jobType != "" && job.Type != jobType {
			continue
		}
		if (job.Status == StatusPending || job.Status == StatusRetryWait) && !job.AvailableAt.After(now) {
			count++
		}
	}
	return count, nil
}

// SetLockedAtForTest mutates lease timestamp in tests.
func (s *MemoryStore) SetLockedAtForTest(id ID, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[id.String()]
	job.LockedAt = &at
	s.jobs[id.String()] = job
}
func (s *MemoryStore) DLQEntries() []DeadLetterRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]DeadLetterRecord, len(s.dlq))
	copy(out, s.dlq)
	return out
}

// EnqueueTestJob is a helper for tests.
func (s *MemoryStore) EnqueueTestJob(jobType Type, payload any) (Job, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Job{}, err
	}
	job, _, err := s.Create(context.Background(), CreateInput{
		Type: jobType, Payload: data, MaxAttempts: 3,
	})
	return job, err
}

var _ Repository = (*MemoryStore)(nil)
