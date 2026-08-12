//go:build integration

package integration_test

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setupPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, func(), error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, nil, err
	}
	cfg.MaxConns = 5
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, err
	}
	return pool, func() { pool.Close() }, nil
}
