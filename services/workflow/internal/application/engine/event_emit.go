package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/contextx"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// EventEmitExecutor publishes platform events via outbox.
type EventEmitExecutor struct {
	publisher ports.EventPublisher
}

func NewEventEmitExecutor(publisher ports.EventPublisher) *EventEmitExecutor {
	return &EventEmitExecutor{publisher: publisher}
}

func (e *EventEmitExecutor) Type() domain.StepType { return domain.StepTypeEventEmit }

func (e *EventEmitExecutor) Execute(ctx context.Context, step domain.WorkflowStepDefinition, execCtx StepContext) (StepResult, error) {
	var cfg domain.EventEmitConfig
	if err := json.Unmarshal(step.Configuration, &cfg); err != nil {
		return StepResult{}, domain.PermanentExecutionError(err)
	}
	if err := cfg.Validate(); err != nil {
		return StepResult{}, domain.PermanentExecutionError(err)
	}

	ctx = contextx.WithCorrelationID(ctx, execCtx.CorrelationID)
	event := domain.NewPlatformEmitEvent(cfg.EventType, execCtx.OrganizationID, execCtx.ExecutionID, cfg.Payload, time.Now().UTC())
	if err := e.publisher.Publish(ctx, event); err != nil {
		return StepResult{}, domain.TransientExecutionError(err)
	}
	return StepResult{Output: &domain.OutputReference{
		Type: "event", Location: cfg.EventType, Inline: "emitted",
	}}, nil
}
