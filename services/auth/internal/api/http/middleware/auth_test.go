package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpmiddleware "github.com/halcyon-io/halcyon/services/auth/internal/api/http/middleware"
	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	jwtinfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/jwt"
)

func TestRequirePermission_Forbidden(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := httpmiddleware.RequirePermission(domain.PermissionAdminAccess)(next)

	principal := appauth.Principal{
		Permissions: []domain.Permission{domain.PermissionProfileRead},
	}
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req = req.WithContext(httpmiddleware.WithPrincipal(req.Context(), principal))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestAuthenticate_ValidToken(t *testing.T) {
	t.Parallel()

	jwtService, err := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	userID, _ := domain.ParseUserID("usr_integrationtest0301")
	email, _ := domain.ParseEmail("middleware@example.com")
	role, _ := domain.ParseRole("user")
	token, _, err := jwtService.IssueAccessToken(context.Background(), ports.TokenClaims{
		UserID:      userID,
		Email:       email,
		Roles:       []domain.Role{role},
		Permissions: domain.PermissionsForRoles([]domain.Role{role}),
	})
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	validateUC := appauth.NewValidateTokenUseCase(jwtService, nil)
	called := false
	handler := httpmiddleware.Authenticate(validateUC)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := httpmiddleware.PrincipalFromContext(r.Context()); !ok {
			t.Fatal("expected principal in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !called {
		t.Fatalf("status = %d, called = %v", rec.Code, called)
	}
}
