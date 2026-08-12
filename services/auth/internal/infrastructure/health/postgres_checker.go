package health

import (
	"context"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresChecker verifies PostgreSQL connectivity for readiness probes.
type PostgresChecker struct {
	pool       *pgxpool.Pool
	configured bool
}

// NewPostgresChecker creates a readiness checker for PostgreSQL.
func NewPostgresChecker(databaseURL string, pool *pgxpool.Pool) *PostgresChecker {
	return &PostgresChecker{
		pool:       pool,
		configured: databaseURL != "",
	}
}

// Check returns database readiness status.
func (c *PostgresChecker) Check(ctx context.Context) map[string]string {
	if !c.configured {
		return map[string]string{"database": "skipped"}
	}

	if err := postgres.Ping(ctx, c.pool); err != nil {
		return map[string]string{"database": "failed"}
	}

	return map[string]string{"database": "ok"}
}

var _ ports.ReadinessChecker = (*PostgresChecker)(nil)
