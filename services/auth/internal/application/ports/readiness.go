package ports

import "context"

// ReadinessChecker reports dependency health for readiness probes.
type ReadinessChecker interface {
	Check(ctx context.Context) map[string]string
}
