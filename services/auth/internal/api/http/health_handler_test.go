package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	"github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/health"
)

var testMetrics = observability.NewHTTPMetrics("auth_test")

func TestHealthHandler_Health(t *testing.T) {
	handler := NewHealthHandler("auth", health.NewPostgresChecker("", nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var body healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
	if body.Service != "auth" {
		t.Errorf("service = %q, want auth", body.Service)
	}
}

func TestHealthHandler_ReadyWithoutDatabase(t *testing.T) {
	handler := NewHealthHandler("auth", health.NewPostgresChecker("", nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil)
	rec := httptest.NewRecorder()

	handler.Ready(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var body readyResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Checks["database"] != "skipped" {
		t.Errorf("database check = %q, want skipped", body.Checks["database"])
	}
}

func TestHealthHandler_ReadyWithDatabaseConfigured(t *testing.T) {
	handler := NewHealthHandler("auth", health.NewPostgresChecker("postgres://localhost/db", nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil)
	rec := httptest.NewRecorder()

	handler.Ready(rec, req)

	var body readyResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Checks["database"] != "failed" {
		t.Errorf("database check = %q, want failed", body.Checks["database"])
	}
}

func TestRouter_HealthEndpoint(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestRouter_MetricsEndpoint(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func newTestRouter() http.Handler {
	return NewRouter(RouterDeps{
		ServiceName: "auth",
		Readiness:   health.NewPostgresChecker("", nil),
		Metrics:     testMetrics,
	})
}
