package health

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolReadiness reports PostgreSQL pool connectivity.
type PoolReadiness struct {
	Pool *pgxpool.Pool
	Name string
}

// Check returns dependency status for readiness endpoints.
func (p PoolReadiness) Check(ctx context.Context) map[string]string {
	key := p.Name
	if key == "" {
		key = "database"
	}
	status := map[string]string{key: "ok"}
	if p.Pool == nil {
		status[key] = "not_configured"
		return status
	}
	if err := p.Pool.Ping(ctx); err != nil {
		status[key] = "unavailable"
	}
	return status
}
