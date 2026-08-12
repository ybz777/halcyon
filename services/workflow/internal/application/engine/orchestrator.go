package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// Orchestrator processes claimed jobs and advances workflow state.
type Orchestrator struct {
	workflows   ports.WorkflowRepository
	executions  ports.ExecutionRepository
	jobs        ports.JobRepository
	publisher   ports.EventPublisher
	tx          ports.TransactionManager
	registry    *Registry
	metrics     ports.WorkflowEngineMetrics
	lease       time.Duration
}

// OrchestratorConfig configures the orchestrator.
type OrchestratorConfig struct {
	LeaseDuration time.Duration
}

func NewOrchestrator(
	workflows ports.WorkflowRepository,
	executions ports.ExecutionRepository,
	jobs ports.JobRepository,
	publisher ports.EventPublisher,
	tx ports.TransactionManager,
	registry *Registry,
	metrics ports.WorkflowEngineMetrics,
	cfg OrchestratorConfig,
) *Orchestrator {
	lease := cfg.LeaseDuration
	if lease <= 0 {
		lease = 30 * time.Second
	}
	return &Orchestrator{
		workflows: workflows, executions: executions, jobs: jobs,
		publisher: publisher, tx: tx, registry: registry, metrics: metrics, lease: lease,
	}
}

// ProcessJob executes a claimed job.
func (o *Orchestrator) ProcessJob(ctx context.Context, job domain.Job, workerID string) error {
	start := time.Now().UTC()

	execution, err := o.executions.FindExecution(ctx, job.OrganizationID, job.ExecutionID)
	if err != nil {
		return err
	}
	if execution.Status == domain.ExecutionCancelled {
		return o.cancelJob(ctx, job, execution)
	}

	def, err := o.loadDefinition(ctx, execution)
	if err != nil {
		return err
	}

	var stepDef *domain.WorkflowStepDefinition
	for i := range def.Steps {
		if def.Steps[i].StepKey == job.StepKey {
			stepDef = &def.Steps[i]
			break
		}
	}
	if stepDef == nil {
		return domain.PermanentExecutionError(fmt.Errorf("step %q not found", job.StepKey))
	}

	stepExec, err := o.executions.FindStepExecution(ctx, job.OrganizationID, job.ExecutionID, job.StepKey)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	// DELAY: schedule continuation without blocking the worker.
	if stepDef.Type == domain.StepTypeDelay && stepExec.Status == domain.StepPending {
		return o.scheduleDelay(ctx, job, stepExec, stepDef, now)
	}
	if stepDef.Type == domain.StepTypeDelay && stepExec.Status == domain.StepRetryWait {
		attemptID, _ := domain.NewAttemptID()
		attempt := &domain.ExecutionAttempt{
			ID: attemptID, JobID: job.ID, OrganizationID: job.OrganizationID,
			Attempt: job.Attempt, Status: "running", StartedAt: now,
		}
		if err := o.jobs.CreateAttempt(ctx, attempt); err != nil {
			return err
		}
		_ = domain.TransitionStep(stepExec, domain.StepRunning, now)
		result := StepResult{Output: &domain.OutputReference{Type: "delay", Location: "completed", Inline: "delay elapsed"}}
		return o.handleSuccess(ctx, job, execution, stepExec, stepDef, attempt, result, start, now)
	}

	_ = domain.TransitionStep(stepExec, domain.StepRunning, now)
	stepExec.AttemptCount = job.Attempt

	attemptID, _ := domain.NewAttemptID()
	attempt := &domain.ExecutionAttempt{
		ID: attemptID, JobID: job.ID, OrganizationID: job.OrganizationID,
		Attempt: job.Attempt, Status: "running", StartedAt: now,
	}
	if err := o.jobs.CreateAttempt(ctx, attempt); err != nil {
		return err
	}

	executor, ok := o.registry.Get(stepDef.Type)
	if !ok {
		return domain.PermanentExecutionError(fmt.Errorf("no executor for %s", stepDef.Type))
	}

	execCtx := StepContext{
		OrganizationID: job.OrganizationID, ExecutionID: job.ExecutionID,
		WorkflowID: execution.WorkflowID, CorrelationID: execution.CorrelationID,
		StepKey: job.StepKey, Attempt: job.Attempt, TriggerDepth: execution.TriggerDepth,
	}

	if o.metrics != nil {
		o.metrics.RecordStepExecution(string(stepDef.Type), "running")
	}

	result, execErr := executor.Execute(ctx, *stepDef, execCtx)
	if execErr != nil {
		return o.handleFailure(ctx, job, execution, stepExec, stepDef, attempt, execErr, now)
	}

	return o.handleSuccess(ctx, job, execution, stepExec, stepDef, attempt, result, start, now)
}

func (o *Orchestrator) loadDefinition(ctx context.Context, execution *domain.WorkflowExecution) (*domain.WorkflowDefinition, error) {
	return o.workflows.FindDefinitionByVersionID(ctx, execution.OrganizationID, execution.WorkflowVersionID)
}

func (o *Orchestrator) handleSuccess(
	ctx context.Context, job domain.Job, execution *domain.WorkflowExecution,
	stepExec *domain.WorkflowStepExecution, stepDef *domain.WorkflowStepDefinition,
	attempt *domain.ExecutionAttempt, result StepResult, start, now time.Time,
) error {
	_ = domain.TransitionStep(stepExec, domain.StepSucceeded, now)
	stepExec.OutputReference = result.Output
	attempt.Status = "succeeded"
	attempt.CompletedAt = &now

	if err := o.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := o.executions.UpdateStepExecution(txCtx, stepExec); err != nil {
			return err
		}
		_ = domain.TransitionJob(&job, domain.JobSucceeded, now)
		if err := o.jobs.UpdateJob(txCtx, &job); err != nil {
			return err
		}
		if err := o.jobs.UpdateAttempt(txCtx, attempt); err != nil {
			return err
		}
		event := domain.NewWorkflowStepSucceeded(job.OrganizationID, job.ExecutionID, execution.WorkflowID, job.StepKey, now)
		if err := o.publisher.Publish(txCtx, event); err != nil {
			return err
		}
		return o.advanceWorkflow(txCtx, execution, stepDef)
	}); err != nil {
		return err
	}

	if o.metrics != nil {
		o.metrics.RecordStepExecution(string(stepDef.Type), "succeeded")
		o.metrics.ObserveJobDuration(string(stepDef.Type), time.Since(start))
	}
	return nil
}

func (o *Orchestrator) handleFailure(
	ctx context.Context, job domain.Job, execution *domain.WorkflowExecution,
	stepExec *domain.WorkflowStepExecution, stepDef *domain.WorkflowStepDefinition,
	attempt *domain.ExecutionAttempt, execErr error, now time.Time,
) error {
	stepExec.LastError = execErr.Error()
	attempt.Status = "failed"
	attempt.CompletedAt = &now
	attempt.ErrorMessage = execErr.Error()

	if domain.ShouldRetryExecution(execErr) && job.Attempt < stepDef.RetryPolicy.MaxAttempts {
		delay := domain.ComputeBackoff(stepDef.RetryPolicy, job.Attempt)
		_ = domain.TransitionStep(stepExec, domain.StepRetryWait, now)
		_ = domain.TransitionJob(&job, domain.JobRetryWait, now)
		job.AvailableAt = now.Add(delay)
		job.Status = domain.JobReady

		if o.metrics != nil {
			o.metrics.RecordJobRetry(string(stepDef.Type))
		}

		return o.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
			if err := o.executions.UpdateStepExecution(txCtx, stepExec); err != nil {
				return err
			}
			if err := o.jobs.UpdateJob(txCtx, &job); err != nil {
				return err
			}
			return o.jobs.UpdateAttempt(txCtx, attempt)
		})
	}

	_ = domain.TransitionStep(stepExec, domain.StepFailed, now)
	_ = domain.TransitionJob(&job, domain.JobFailed, now)
	_ = domain.TransitionExecution(execution, domain.ExecutionFailed, now)

	if o.metrics != nil {
		o.metrics.RecordStepFailure(string(stepDef.Type), string(domain.ClassifyExecutionFailure(execErr)))
		o.metrics.RecordExecutionFailure(string(domain.ClassifyExecutionFailure(execErr)))
	}

	return o.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := o.executions.UpdateStepExecution(txCtx, stepExec); err != nil {
			return err
		}
		if err := o.jobs.UpdateJob(txCtx, &job); err != nil {
			return err
		}
		if err := o.jobs.UpdateAttempt(txCtx, attempt); err != nil {
			return err
		}
		if err := o.executions.UpdateExecution(txCtx, execution); err != nil {
			return err
		}
		stepEvent := domain.NewWorkflowStepFailed(job.OrganizationID, job.ExecutionID, execution.WorkflowID, job.StepKey, execErr.Error(), now)
		if err := o.publisher.Publish(txCtx, stepEvent); err != nil {
			return err
		}
		failEvent := domain.NewWorkflowExecutionFailed(job.OrganizationID, job.ExecutionID, execution.WorkflowID, execErr.Error(), now)
		return o.publisher.Publish(txCtx, failEvent)
	})
}

func (o *Orchestrator) advanceWorkflow(ctx context.Context, execution *domain.WorkflowExecution, completedStep *domain.WorkflowStepDefinition) error {
	def, err := o.loadDefinition(ctx, execution)
	if err != nil {
		return err
	}

	steps, err := o.executions.ListStepExecutions(ctx, execution.OrganizationID, execution.ID)
	if err != nil {
		return err
	}
	completed := make(map[string]domain.StepStatus, len(steps))
	for _, s := range steps {
		completed[s.StepKey] = s.Status
	}

	next, ok := domain.NextRunnableStep(def.Steps, completed)
	if !ok {
		now := time.Now().UTC()
		_ = domain.TransitionExecution(execution, domain.ExecutionSucceeded, now)
		if err := o.executions.UpdateExecution(ctx, execution); err != nil {
			return err
		}
		event := domain.NewWorkflowExecutionSucceeded(execution.OrganizationID, execution.ID, execution.WorkflowID, now)
		if err := o.publisher.Publish(ctx, event); err != nil {
			return err
		}
		if o.metrics != nil {
			o.metrics.RecordExecution(string(domain.ExecutionSucceeded))
		}
		return nil
	}

	stepExec, err := o.executions.FindStepExecution(ctx, execution.OrganizationID, execution.ID, next.StepKey)
	if err != nil {
		return err
	}

	jobID, err := domain.NewJobID()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	job := &domain.Job{
		ID: jobID, OrganizationID: execution.OrganizationID, ExecutionID: execution.ID,
		StepExecutionID: stepExec.ID, StepKey: next.StepKey,
		Status: domain.JobReady, AvailableAt: now, CreatedAt: now,
	}
	stepEvent := domain.NewWorkflowStepStarted(execution.OrganizationID, execution.ID, execution.WorkflowID, next.StepKey, 1, now)
	if err := o.publisher.Publish(ctx, stepEvent); err != nil {
		return err
	}
	return o.jobs.CreateJob(ctx, job)
}

func (o *Orchestrator) scheduleDelay(ctx context.Context, job domain.Job, stepExec *domain.WorkflowStepExecution, stepDef *domain.WorkflowStepDefinition, now time.Time) error {
	var cfg domain.DelayConfig
	if err := json.Unmarshal(stepDef.Configuration, &cfg); err != nil {
		return domain.PermanentExecutionError(err)
	}
	d, err := time.ParseDuration(cfg.Duration)
	if err != nil {
		return domain.PermanentExecutionError(err)
	}
	_ = domain.TransitionStep(stepExec, domain.StepRetryWait, now)
	job.Status = domain.JobReady
	job.AvailableAt = now.Add(d)
	job.LockedAt = nil
	job.LockedBy = ""
	return o.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := o.executions.UpdateStepExecution(txCtx, stepExec); err != nil {
			return err
		}
		return o.jobs.UpdateJob(txCtx, &job)
	})
}

func (o *Orchestrator) cancelJob(ctx context.Context, job domain.Job, execution *domain.WorkflowExecution) error {
	now := time.Now().UTC()
	_ = domain.TransitionJob(&job, domain.JobCancelled, now)
	return o.jobs.UpdateJob(ctx, &job)
}

// RecoverLeases returns expired running jobs to READY.
func (o *Orchestrator) RecoverLeases(ctx context.Context) (int64, error) {
	n, err := o.jobs.RecoverExpiredLeases(ctx, o.lease)
	if err == nil && n > 0 && o.metrics != nil {
		o.metrics.RecordJobRecovery()
	}
	return n, err
}
