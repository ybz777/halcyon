package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	httpapi "github.com/halcyon-io/halcyon/services/gateway/internal/api/http"
	"github.com/halcyon-io/halcyon/services/gateway/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/gateway/internal/config"
	gwmetrics "github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/metrics"
	"github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/proxy"
)

var testProxyMetrics = gwmetrics.NewGatewayMetrics()

type stubValidator struct {
	claims ports.TokenClaims
}

func (s stubValidator) Validate(_ context.Context, _ string) (ports.TokenClaims, error) {
	return s.claims, nil
}

func TestRouter_HealthIsPublic(t *testing.T) {
	t.Parallel()

	authUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer authUpstream.Close()

	upstream, err := proxy.NewUpstream("auth", authUpstream.URL, authUpstream.Client(), nil, nil, "gateway", testProxyMetrics)
	if err != nil {
		t.Fatalf("NewUpstream() error = %v", err)
	}

	router := httpapi.NewRouter(httpapi.RouterDeps{
		ServiceName:  "gateway",
		Config:       config.Config{MaxRequestBodyBytes: 1024, SecurityHeaders: config.SecurityHeadersConfig{}},
		ProxyMetrics: testProxyMetrics,
		Upstreams:    map[string]*proxy.Upstream{"auth": upstream},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestRouter_ProtectedWithoutTokenRejected(t *testing.T) {
	t.Parallel()

	router := httpapi.NewRouter(httpapi.RouterDeps{
		ServiceName:  "gateway",
		Config:       config.Config{MaxRequestBodyBytes: 1024, SecurityHeaders: config.SecurityHeadersConfig{}},
		ProxyMetrics: testProxyMetrics,
		Upstreams:    map[string]*proxy.Upstream{},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/org_test", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRouter_APIKeysForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	upstreamCalls := 0
	authUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer authUpstream.Close()

	upstream, err := proxy.NewUpstream("auth", authUpstream.URL, authUpstream.Client(), nil, nil, "gateway", testProxyMetrics)
	if err != nil {
		t.Fatalf("NewUpstream() error = %v", err)
	}

	router := httpapi.NewRouter(httpapi.RouterDeps{
		ServiceName:  "gateway",
		Config:       config.Config{MaxRequestBodyBytes: 1024, SecurityHeaders: config.SecurityHeadersConfig{}},
		ProxyMetrics: testProxyMetrics,
		Validator: stubValidator{claims: ports.TokenClaims{
			UserID:      "usr_test",
			Email:       "user@example.com",
			Permissions: []string{"profile:read"},
		}},
		Upstreams: map[string]*proxy.Upstream{"auth": upstream},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/api-keys", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if upstreamCalls != 0 {
		t.Fatalf("upstream calls = %d, want 0", upstreamCalls)
	}
}

func TestRouter_APIKeysAllowedWithPermission(t *testing.T) {
	t.Parallel()

	upstreamCalls := 0
	authUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer authUpstream.Close()

	upstream, err := proxy.NewUpstream("auth", authUpstream.URL, authUpstream.Client(), nil, nil, "gateway", testProxyMetrics)
	if err != nil {
		t.Fatalf("NewUpstream() error = %v", err)
	}

	router := httpapi.NewRouter(httpapi.RouterDeps{
		ServiceName:  "gateway",
		Config:       config.Config{MaxRequestBodyBytes: 1024, SecurityHeaders: config.SecurityHeadersConfig{}},
		ProxyMetrics: testProxyMetrics,
		Validator: stubValidator{claims: ports.TokenClaims{
			UserID:      "usr_admin",
			Email:       "admin@example.com",
			Roles:       []string{"admin"},
			Permissions: []string{"apikey:manage"},
		}},
		Upstreams: map[string]*proxy.Upstream{"auth": upstream},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/api-keys", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if upstreamCalls != 1 {
		t.Fatalf("upstream calls = %d, want 1", upstreamCalls)
	}
}
