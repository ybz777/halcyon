package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/halcyon-io/halcyon/services/jobs/internal/domain"
)

func TestPayloadSummaryRedactsSecrets(t *testing.T) {
	raw, _ := json.Marshal(map[string]string{
		"delivery_id": "whd_123",
		"api_key":     "super-secret",
		"token":       "abc",
	})
	summary := domain.PayloadSummary(raw)
	if summary["api_key"] != "[redacted]" {
		t.Fatalf("expected redacted api_key, got %#v", summary["api_key"])
	}
	if summary["delivery_id"] != "whd_123" {
		t.Fatalf("expected delivery_id preserved")
	}
}

func TestNormalizePageSize(t *testing.T) {
	if domain.NormalizePageSize(500) != 200 {
		t.Fatal("page size should cap at 200")
	}
}
