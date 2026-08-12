package metrics

import (
	"time"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	"github.com/prometheus/client_golang/prometheus"
)

// Collector records bounded job metrics.
type Collector struct {
	created       *prometheus.CounterVec
	claimed       *prometheus.CounterVec
	succeeded     *prometheus.CounterVec
	failed        *prometheus.CounterVec
	retried       *prometheus.CounterVec
	cancelled     *prometheus.CounterVec
	reclaimed     *prometheus.CounterVec
	deadLettered  *prometheus.CounterVec
	leaseExpired  *prometheus.CounterVec
	panics        *prometheus.CounterVec
	queueDepth    *prometheus.GaugeVec
	active        *prometheus.GaugeVec
	duration      *prometheus.HistogramVec
	queueDelay    *prometheus.HistogramVec
}

func NewCollector() *Collector {
	c := &Collector{
		created: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "jobs_created_total", Help: "Jobs created"}, []string{"job_type"}),
		claimed: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "jobs_started_total", Help: "Jobs claimed/started"}, []string{"job_type"}),
		succeeded: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "jobs_succeeded_total", Help: "Jobs succeeded"}, []string{"job_type"}),
		failed: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "jobs_failed_total", Help: "Jobs failed"}, []string{"job_type", "error_category"}),
		retried: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "jobs_retried_total", Help: "Jobs retried"}, []string{"job_type"}),
		cancelled: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "jobs_cancelled_total", Help: "Jobs cancelled"}, []string{"job_type"}),
		reclaimed: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "jobs_reclaimed_total", Help: "Jobs reclaimed after lease expiry"}, []string{"job_type"}),
		deadLettered: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "jobs_dead_lettered_total", Help: "Jobs moved to dead letter"}, []string{"job_type"}),
		leaseExpired: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "job_lease_expirations_total", Help: "Job lease expirations"}, []string{"job_type"}),
		panics: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "job_panics_total", Help: "Job handler panics recovered"}, []string{"job_type"}),
		queueDepth: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "jobs_queue_depth", Help: "Ready jobs in queue"}, []string{"job_type"}),
		active: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "job_active", Help: "Currently executing jobs"}, []string{"job_type"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "job_execution_duration_seconds", Help: "Job handler duration", Buckets: prometheus.DefBuckets,
		}, []string{"job_type"}),
		queueDelay: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "job_queue_delay_seconds", Help: "Time from creation to claim", Buckets: prometheus.DefBuckets,
		}, []string{"job_type"}),
	}
	prometheus.MustRegister(
		c.created, c.claimed, c.succeeded, c.failed, c.retried, c.cancelled,
		c.reclaimed, c.deadLettered, c.leaseExpired, c.panics,
		c.queueDepth, c.active, c.duration, c.queueDelay,
	)
	return c
}

func (c *Collector) IncCreated(jobType platformjobs.Type)     { c.created.WithLabelValues(string(jobType)).Inc() }
func (c *Collector) IncClaimed(jobType platformjobs.Type)     { c.claimed.WithLabelValues(string(jobType)).Inc() }
func (c *Collector) IncSucceeded(jobType platformjobs.Type)   { c.succeeded.WithLabelValues(string(jobType)).Inc() }
func (c *Collector) IncFailed(jobType platformjobs.Type, category platformjobs.ErrorCategory) {
	c.failed.WithLabelValues(string(jobType), string(category)).Inc()
}
func (c *Collector) IncRetried(jobType platformjobs.Type)     { c.retried.WithLabelValues(string(jobType)).Inc() }
func (c *Collector) IncCancelled(jobType platformjobs.Type)   { c.cancelled.WithLabelValues(string(jobType)).Inc() }
func (c *Collector) IncReclaimed(jobType platformjobs.Type)   { c.reclaimed.WithLabelValues(string(jobType)).Inc() }
func (c *Collector) IncDeadLettered(jobType platformjobs.Type) { c.deadLettered.WithLabelValues(string(jobType)).Inc() }
func (c *Collector) IncLeaseExpired(jobType platformjobs.Type) { c.leaseExpired.WithLabelValues(string(jobType)).Inc() }
func (c *Collector) IncPanics(jobType platformjobs.Type)      { c.panics.WithLabelValues(string(jobType)).Inc() }
func (c *Collector) ObserveDuration(jobType platformjobs.Type, d time.Duration) {
	c.duration.WithLabelValues(string(jobType)).Observe(d.Seconds())
}
func (c *Collector) ObserveQueueDelay(jobType platformjobs.Type, d time.Duration) {
	c.queueDelay.WithLabelValues(string(jobType)).Observe(d.Seconds())
}
func (c *Collector) SetActive(jobType platformjobs.Type, count float64) {
	c.active.WithLabelValues(string(jobType)).Set(count)
}

func (c *Collector) SetQueueDepth(jobType platformjobs.Type, depth float64) {
	c.queueDepth.WithLabelValues(string(jobType)).Set(depth)
}

var _ platformjobs.MetricsCollector = (*Collector)(nil)
