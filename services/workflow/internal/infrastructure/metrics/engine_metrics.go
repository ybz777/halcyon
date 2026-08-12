package metrics

import (
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// EngineMetrics records workflow engine observability.
type EngineMetrics struct {
	executions      *prometheus.CounterVec
	execFailures    *prometheus.CounterVec
	execDuration    prometheus.Histogram
	stepExecutions  *prometheus.CounterVec
	stepFailures    *prometheus.CounterVec
	jobClaims       prometheus.Counter
	jobRetries      *prometheus.CounterVec
	jobRecoveries   prometheus.Counter
	jobQueueDepth   prometheus.Gauge
	jobDuration     *prometheus.HistogramVec
	cancellations   prometheus.Counter
	idempotencyHits prometheus.Counter
}

func NewEngineMetrics() *EngineMetrics {
	return &EngineMetrics{
		executions: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "workflow_executions_total", Help: "Workflow executions by status.",
		}, []string{"status"}),
		execFailures: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "workflow_execution_failures_total", Help: "Workflow execution failures.",
		}, []string{"failure_category"}),
		execDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "workflow_execution_duration_seconds", Help: "Workflow execution duration.",
			Buckets: prometheus.DefBuckets,
		}),
		stepExecutions: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "workflow_step_executions_total", Help: "Step executions by type and status.",
		}, []string{"step_type", "status"}),
		stepFailures: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "workflow_step_failures_total", Help: "Step failures.",
		}, []string{"step_type", "failure_category"}),
		jobClaims: promauto.NewCounter(prometheus.CounterOpts{
			Name: "workflow_job_claims_total", Help: "Jobs claimed by workers.",
		}),
		jobRetries: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "workflow_job_retries_total", Help: "Job retries.",
		}, []string{"step_type"}),
		jobRecoveries: promauto.NewCounter(prometheus.CounterOpts{
			Name: "workflow_job_recoveries_total", Help: "Jobs recovered from expired leases.",
		}),
		jobQueueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "workflow_job_queue_depth", Help: "Ready workflow jobs.",
		}),
		jobDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name: "workflow_job_execution_duration_seconds", Help: "Job execution duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"step_type"}),
		cancellations: promauto.NewCounter(prometheus.CounterOpts{
			Name: "workflow_cancellations_total", Help: "Workflow cancellations.",
		}),
		idempotencyHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "workflow_idempotency_hits_total", Help: "Idempotent execution deduplications.",
		}),
	}
}

func (m *EngineMetrics) RecordExecution(status string)              { m.executions.WithLabelValues(status).Inc() }
func (m *EngineMetrics) RecordExecutionFailure(category string)     { m.execFailures.WithLabelValues(category).Inc() }
func (m *EngineMetrics) ObserveExecutionDuration(d time.Duration)   { m.execDuration.Observe(d.Seconds()) }
func (m *EngineMetrics) RecordStepExecution(stepType, status string) { m.stepExecutions.WithLabelValues(stepType, status).Inc() }
func (m *EngineMetrics) RecordStepFailure(stepType, category string) { m.stepFailures.WithLabelValues(stepType, category).Inc() }
func (m *EngineMetrics) RecordJobClaim()                            { m.jobClaims.Inc() }
func (m *EngineMetrics) RecordJobRetry(stepType string)             { m.jobRetries.WithLabelValues(stepType).Inc() }
func (m *EngineMetrics) RecordJobRecovery()                         { m.jobRecoveries.Inc() }
func (m *EngineMetrics) SetJobQueueDepth(depth int64)               { m.jobQueueDepth.Set(float64(depth)) }
func (m *EngineMetrics) ObserveJobDuration(stepType string, d time.Duration) {
	m.jobDuration.WithLabelValues(stepType).Observe(d.Seconds())
}
func (m *EngineMetrics) RecordCancellation()  { m.cancellations.Inc() }
func (m *EngineMetrics) RecordIdempotencyHit() { m.idempotencyHits.Inc() }

var _ ports.WorkflowEngineMetrics = (*EngineMetrics)(nil)
