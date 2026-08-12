package postgres

import (
	"fmt"
	"strings"
)

func migrateURL(databaseURL string) string {
	switch {
	case strings.HasPrefix(databaseURL, "pgx5://"):
		return databaseURL
	case strings.HasPrefix(databaseURL, "postgres://"):
		return "pgx5://" + strings.TrimPrefix(databaseURL, "postgres://")
	case strings.HasPrefix(databaseURL, "postgresql://"):
		return "pgx5://" + strings.TrimPrefix(databaseURL, "postgresql://")
	default:
		return databaseURL
	}
}

func normalizeDatabaseURL(databaseURL string) (string, error) {
	if databaseURL == "" {
		return "", fmt.Errorf("database url is required")
	}
	return migrateURL(databaseURL), nil
}
