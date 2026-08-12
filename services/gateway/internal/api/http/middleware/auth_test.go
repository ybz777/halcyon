package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gwmiddleware "github.com/halcyon-io/halcyon/services/gateway/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/gateway/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/gateway/internal/domain"
)

type stubValidator struct {
	claims ports.TokenClaims
}

func (s stubValidator) Validate(_ context.Context, _ string) (ports.TokenClaims, error) {
	return s.claims, nil
}

func TestAuthorize_APIKeysRequiresPermission(t *testing.T) {
	t.Parallel()

	handler := gwmiddleware.Authenticate(stubValidator{
		claims: ports.TokenClaims{
			UserID:      "usr_test",
			Email:       "user@example.com",
			Permissions: []string{"profile:read"},
		},
	}, nil, nil, nil)(gwmiddleware.Authorize()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/api-keys", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestAuthorize_APIKeysAllowedWithPermission(t *testing.T) {
	t.Parallel()

	called := false
	handler := gwmiddleware.Authenticate(stubValidator{
		claims: ports.TokenClaims{
			UserID:      "usr_admin",
			Email:       "admin@example.com",
			Roles:       []string{"admin"},
			Permissions: []string{"apikey:manage"},
		},
	}, nil, nil, nil)(gwmiddleware.Authorize()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/api-keys", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418", rec.Code)
	}
	if !called {
		t.Fatal("expected downstream handler to run")
	}
}

func TestAuthorize_PassesThroughUnrestrictedRoutes(t *testing.T) {
	t.Parallel()

	called := false
	handler := gwmiddleware.Authenticate(stubValidator{
		claims: ports.TokenClaims{
			UserID:      "usr_test",
			Email:       "user@example.com",
			Permissions: []string{"profile:read"},
		},
	}, nil, nil, nil)(gwmiddleware.Authorize()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !called {
		t.Fatal("expected downstream handler to run")
	}
}

func TestNewUserPrincipal_PreservesPermissions(t *testing.T) {
	t.Parallel()

	principal := domain.NewUserPrincipal("usr_1", "user@example.com", []string{"admin"}, []string{"apikey:manage"})
	if !principal.HasPermission(domain.PermissionAPIKeyManage) {
		t.Fatal("expected apikey:manage on principal")
	}
}
