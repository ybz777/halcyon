package migrations

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

const platformMigrationsTable = "platform_schema_migrations"

// Run applies embedded platform migrations.
func Run(databaseURL string) error {
	migrationURL, err := normalizeDatabaseURL(databaseURL)
	if err != nil {
		return err
	}

	source, err := iofs.New(Files, ".")
	if err != nil {
		return fmt.Errorf("load migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance(
		"iofs",
		source,
		migrationURL,
	)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply platform migrations: %w", err)
	}

	return nil
}

func normalizeDatabaseURL(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("database url is required")
	}

	switch {
	case strings.HasPrefix(raw, "postgres://"):
		raw = "pgx5://" + strings.TrimPrefix(raw, "postgres://")

	case strings.HasPrefix(raw, "postgresql://"):
		raw = "pgx5://" + strings.TrimPrefix(raw, "postgresql://")

	case strings.HasPrefix(raw, "pgx5://"):
		// Already normalized.

	default:
		return "", fmt.Errorf("unsupported database URL scheme")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse database URL: %w", err)
	}

	query := parsed.Query()
	query.Set("x-migrations-table", platformMigrationsTable)
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}