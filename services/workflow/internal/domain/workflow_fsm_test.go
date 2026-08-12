package domain_test

import (
	"testing"

	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

func TestExecutionStateMachine_ValidTransitions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		from, to domain.ExecutionStatus
		ok       bool
	}{
		{domain.ExecutionPending, domain.ExecutionRunning, true},
		{domain.ExecutionRunning, domain.ExecutionSucceeded, true},
		{domain.ExecutionRunning, domain.ExecutionFailed, true},
		{domain.ExecutionRunning, domain.ExecutionCancelled, true},
		{domain.ExecutionPending, domain.ExecutionSucceeded, false},
		{domain.ExecutionSucceeded, domain.ExecutionFailed, false},
	}
	for _, tc := range cases {
		if got := domain.CanTransitionExecution(tc.from, tc.to); got != tc.ok {
			t.Fatalf("%s -> %s = %v want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}

func TestStepStateMachine_InvalidTransitions(t *testing.T) {
	t.Parallel()
	if domain.CanTransitionStep(domain.StepSucceeded, domain.StepRunning) {
		t.Fatal("expected invalid transition")
	}
}

func TestJobStateMachine_RetryWaitToReady(t *testing.T) {
	t.Parallel()
	if !domain.CanTransitionJob(domain.JobRetryWait, domain.JobReady) {
		t.Fatal("expected RETRY_WAIT -> READY")
	}
}

func TestValidateHTTPURL_BlocksSSRF(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"http://127.0.0.1/health",
		"http://localhost/admin",
		"http://169.254.169.254/latest/meta-data",
		"http://10.0.0.1/internal",
	}
	for _, u := range blocked {
		if err := domain.ValidateHTTPURL(u); err != domain.ErrSSRFBlocked {
			t.Fatalf("url %q: got %v want ErrSSRFBlocked", u, err)
		}
	}
	if err := domain.ValidateHTTPURL("https://api.example.com/v1"); err != nil {
		t.Fatalf("public url blocked: %v", err)
	}
}

func TestComputeBackoff_Increases(t *testing.T) {
	t.Parallel()
	policy := domain.DefaultRetryPolicy()
	d1 := domain.ComputeBackoff(policy, 1)
	d2 := domain.ComputeBackoff(policy, 3)
	if d2 < d1 {
		t.Fatalf("backoff should increase: %v vs %v", d1, d2)
	}
}
