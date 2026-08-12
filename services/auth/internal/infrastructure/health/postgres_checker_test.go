package health

import (
	"context"
	"testing"
)

func TestPostgresChecker_NotConfigured(t *testing.T) {
	checker := NewPostgresChecker("", nil)

	checks := checker.Check(context.Background())
	if checks["database"] != "skipped" {
		t.Fatalf("database = %q, want skipped", checks["database"])
	}
}

func TestPostgresChecker_ConfiguredWithoutPool(t *testing.T) {
	checker := NewPostgresChecker("postgres://localhost/db", nil)

	checks := checker.Check(context.Background())
	if checks["database"] != "failed" {
		t.Fatalf("database = %q, want failed", checks["database"])
	}
}
