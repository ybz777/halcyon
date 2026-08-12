package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/postgres/sqlcgen"
	"github.com/halcyon-io/halcyon/services/auth/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations applies embedded SQL migrations.
func RunMigrations(databaseURL string) error {
	url, err := normalizeDatabaseURL(databaseURL)
	if err != nil {
		return err
	}

	source, err := iofs.New(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("load migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, url)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

// Ping verifies database connectivity.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	queries := sqlcgen.New(pool)
	if _, err := queries.Ping(ctx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}

	return nil
}
