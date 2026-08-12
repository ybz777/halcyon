package jobs

import "time"

// TypePolicy defines retry and execution limits for a job type.
type TypePolicy struct {
	MaxAttempts      int
	Retry            RetryConfig
	ExecutionTimeout time.Duration
}

// DefaultPolicies returns production defaults for registered job types.
func DefaultPolicies() map[Type]TypePolicy {
	base := RetryConfig{
		Initial:    time.Second,
		MaxBackoff: 5 * time.Minute,
		Multiplier: 2,
		Jitter:     0.5,
	}
	return map[Type]TypePolicy{
		TypeWebhookDelivery: {
			MaxAttempts:      6,
			Retry:            base,
			ExecutionTimeout: 30 * time.Second,
		},
		TypeEmailDelivery: {
			MaxAttempts:      6,
			Retry:            base,
			ExecutionTimeout: 30 * time.Second,
		},
		TypeNotificationDispatch: {
			MaxAttempts:      5,
			Retry:            base,
			ExecutionTimeout: 30 * time.Second,
		},
		TypeWorkflowExecution: {
			MaxAttempts:      8,
			Retry:            base,
			ExecutionTimeout: 10 * time.Minute,
		},
		TypeAuditPersistence: {
			MaxAttempts:      4,
			Retry:            base,
			ExecutionTimeout: 15 * time.Second,
		},
	}
}

// PolicyRegistry resolves per-type execution policy.
type PolicyRegistry struct {
	policies map[Type]TypePolicy
	fallback TypePolicy
}

// NewPolicyRegistry creates a registry with defaults.
func NewPolicyRegistry() *PolicyRegistry {
	return &PolicyRegistry{
		policies: DefaultPolicies(),
		fallback: TypePolicy{
			MaxAttempts:      6,
			Retry:            DefaultRetryConfig(),
			ExecutionTimeout: 5 * time.Minute,
		},
	}
}

// Lookup returns the policy for a job type.
func (r *PolicyRegistry) Lookup(jobType Type) TypePolicy {
	if p, ok := r.policies[jobType]; ok {
		return p
	}
	return r.fallback
}

// MaxAttemptsFor returns the configured max attempts for a type.
func (r *PolicyRegistry) MaxAttemptsFor(jobType Type) int {
	p := r.Lookup(jobType)
	if p.MaxAttempts <= 0 {
		return 6
	}
	return p.MaxAttempts
}
