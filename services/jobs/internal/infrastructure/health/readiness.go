package health

import (
	"context"

	"github.com/halcyon-io/halcyon/services/jobs/internal/application/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReadinessChecker reports database readiness.
type ReadinessChecker struct {
	pool *pgxpool.Pool
}

func NewReadinessChecker(pool *pgxpool.Pool) *ReadinessChecker {
	return &ReadinessChecker{pool: pool}
}

func (c *ReadinessChecker) Check(ctx context.Context) map[string]string {
	status := map[string]string{"database": "ok"}
	if c.pool == nil {
		status["database"] = "not_configured"
		return status
	}
	if err := c.pool.Ping(ctx); err != nil {
		status["database"] = "unavailable"
	}
	return status
}

var _ ports.ReadinessChecker = (*ReadinessChecker)(nil)
