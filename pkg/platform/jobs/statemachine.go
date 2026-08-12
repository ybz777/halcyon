package jobs

// CanTransition reports whether a status change is valid.
func CanTransition(from, to Status) bool {
	switch from {
	case StatusPending:
		return to == StatusRunning || to == StatusCancelled
	case StatusRunning:
		return to == StatusSucceeded || to == StatusRetryWait || to == StatusFailed ||
			to == StatusDeadLettered || to == StatusCancelled
	case StatusRetryWait:
		return to == StatusPending || to == StatusRunning || to == StatusCancelled ||
			to == StatusDeadLettered
	case StatusSucceeded, StatusFailed, StatusDeadLettered, StatusCancelled:
		return false
	default:
		return false
	}
}

// ReclaimTarget returns the status after lease expiration.
func ReclaimTarget(attempts, maxAttempts int) Status {
	if attempts >= maxAttempts {
		return StatusDeadLettered
	}
	return StatusRetryWait
}
