package domain_test

import (
	"testing"

	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

func TestParseStatus(t *testing.T) {
	t.Parallel()
	if _, err := domain.ParseStatus("OPEN"); err != nil {
		t.Fatalf("parse OPEN: %v", err)
	}
	if _, err := domain.ParseStatus("invalid"); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestParseSeverity(t *testing.T) {
	t.Parallel()
	sev, err := domain.ParseSeverity("critical")
	if err != nil {
		t.Fatalf("parse severity: %v", err)
	}
	if sev != domain.SeverityCritical {
		t.Fatalf("severity = %s", sev)
	}
}
