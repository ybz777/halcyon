package health

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ReadinessChecker checks audit dependencies.
type ReadinessChecker struct {
	pool *pgxpool.Pool
}

// NewReadinessChecker creates a readiness checker.
func NewReadinessChecker(pool *pgxpool.Pool) *ReadinessChecker {
	return &ReadinessChecker{pool: pool}
}

// Check reports dependency health.
func (c *ReadinessChecker) Check(ctx context.Context) map[string]string {
	status := map[string]string{"database": "ok"}
	if c.pool == nil {
		status["database"] = "unavailable"
		return status
	}
	if err := c.pool.Ping(ctx); err != nil {
		status["database"] = "unavailable"
	}
	return status
}
