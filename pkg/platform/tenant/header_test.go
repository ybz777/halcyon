package tenant

import (
	"net/http"
	"net/http/httptest"
	"testing"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
)

func TestRequireGatewayForOrganizationHeaders_ProductionRejectsSpoofedOrg(t *testing.T) {
	t.Parallel()

	handler := RequireGatewayForOrganizationHeaders(nil, platformconfig.EnvProduction)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-Halcyon-Organization-ID", "org_victim000000001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRequireGatewayForOrganizationHeaders_DevelopmentAllowsDirectAccess(t *testing.T) {
	t.Parallel()

	handler := RequireGatewayForOrganizationHeaders(nil, platformconfig.EnvDevelopment)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-Halcyon-Organization-ID", "org_devtest000000001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
