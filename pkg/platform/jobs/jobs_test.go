package jobs_test

import (
	"testing"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
)

func TestStatusTransitions(t *testing.T) {
	if !platformjobs.CanTransition(platformjobs.StatusPending, platformjobs.StatusRunning) {
		t.Fatal("pending -> running")
	}
	if platformjobs.CanTransition(platformjobs.StatusSucceeded, platformjobs.StatusRunning) {
		t.Fatal("succeeded -> running invalid")
	}
}

func TestReclaimTarget(t *testing.T) {
	if platformjobs.ReclaimTarget(5, 5) != platformjobs.StatusDeadLettered {
		t.Fatal("max attempts should dead letter")
	}
	if platformjobs.ReclaimTarget(2, 5) != platformjobs.StatusRetryWait {
		t.Fatal("under max should retry wait")
	}
}

func TestValidateJobType(t *testing.T) {
	if err := platformjobs.ValidateType(platformjobs.TypeWebhookDelivery); err != nil {
		t.Fatal(err)
	}
	if err := platformjobs.ValidateType("evil.type"); err == nil {
		t.Fatal("expected invalid type")
	}
}

func TestHandlerErrorRetryable(t *testing.T) {
	err := platformjobs.Retryable(platformjobs.ErrStaleLease)
	if !platformjobs.IsRetryable(err) {
		t.Fatal("expected retryable")
	}
	err2 := platformjobs.Permanent(platformjobs.ErrInvalidJobType)
	if platformjobs.IsRetryable(err2) {
		t.Fatal("expected permanent")
	}
}
