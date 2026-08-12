package domain_test

import (
	"testing"

	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

func TestValidateCreateWorkflow_RejectsTooManySteps(t *testing.T) {
	t.Parallel()
	limits := domain.WorkflowLimits{MaxWorkflowSteps: 2, MaxWorkflowNameLen: 64, MaxRetryAttempts: 3, DefaultStepTimeout: 0, MaxStepTimeout: 0}
	steps := []domain.StepInput{
		{StepKey: "a", Name: "A", Type: "DELAY", Configuration: []byte(`{"duration":"1s"}`)},
		{StepKey: "b", Name: "B", Type: "DELAY", Configuration: []byte(`{"duration":"1s"}`)},
		{StepKey: "c", Name: "C", Type: "DELAY", Configuration: []byte(`{"duration":"1s"}`)},
	}
	err := domain.ValidateCreateWorkflow(domain.CreateWorkflowInput{Name: "test", Steps: steps}, limits)
	if err != domain.ErrWorkflowLimitExceeded {
		t.Fatalf("got %v", err)
	}
}
