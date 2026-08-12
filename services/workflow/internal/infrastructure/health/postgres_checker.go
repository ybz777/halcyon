package health

import (
	"context"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresChecker reports PostgreSQL readiness.
type PostgresChecker struct {
	databaseURL string
	pool        *pgxpool.Pool
}

// NewPostgresChecker creates a readiness checker.
func NewPostgresChecker(databaseURL string, pool *pgxpool.Pool) *PostgresChecker {
	return &PostgresChecker{databaseURL: databaseURL, pool: pool}
}

// Check implements ports.ReadinessChecker.
func (c *PostgresChecker) Check(ctx context.Context) map[string]string {
	status := map[string]string{"database": "disabled"}
	if c.databaseURL == "" {
		return status
	}
	if err := postgres.Ping(ctx, c.pool); err != nil {
		status["database"] = "unhealthy: " + err.Error()
		return status
	}
	status["database"] = "ok"
	return status
}

var _ ports.ReadinessChecker = (*PostgresChecker)(nil)
