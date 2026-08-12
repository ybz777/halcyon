package idempotency

import "errors"

var (
	ErrKeyRequired      = errors.New("idempotency key required")
	ErrKeyInvalid       = errors.New("invalid idempotency key")
	ErrFingerprintMismatch = errors.New("idempotency key reused with different request")
	ErrInProgress       = errors.New("idempotency request already in progress")
	ErrStoreUnavailable = errors.New("idempotency store unavailable")
)

const (
	ProblemKeyRequired       = "https://halcyon.io/problems/idempotency-key-required"
	ProblemKeyInvalid        = "https://halcyon.io/problems/idempotency-key-invalid"
	ProblemConflict          = "https://halcyon.io/problems/idempotency-conflict"
	ProblemInProgress        = "https://halcyon.io/problems/idempotency-in-progress"
	ProblemStoreUnavailable  = "https://halcyon.io/problems/idempotency-store-unavailable"
)

const (
	HeaderKey    = "Idempotency-Key"
	HeaderReplay = "Idempotency-Replayed"
)
