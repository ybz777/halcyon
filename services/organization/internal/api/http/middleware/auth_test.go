package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	httpmiddleware "github.com/halcyon-io/halcyon/services/organization/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
)

func TestAuthenticate_MissingToken(t *testing.T) {
	t.Parallel()

	handler := httpmiddleware.Authenticate(stubValidator{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

type stubValidator struct{}

func (stubValidator) Validate(context.Context, string) (ports.AuthenticatedUser, error) {
	return ports.AuthenticatedUser{}, domain.ErrUnauthorized
}
