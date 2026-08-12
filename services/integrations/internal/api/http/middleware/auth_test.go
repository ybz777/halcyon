package middleware_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	"github.com/halcyon-io/halcyon/pkg/platform/servicetoken"
	appwebhook "github.com/halcyon-io/halcyon/services/integrations/internal/application/webhook"
	httpmiddleware "github.com/halcyon-io/halcyon/services/integrations/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/integrations/internal/domain"
)

type stubValidator struct{}

func (stubValidator) Validate(context.Context, string) (appwebhook.Principal, error) {
	return appwebhook.Principal{}, domain.ErrUnauthorized
}

func testServiceToken(t *testing.T) (*platformauth.GatewayTrust, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	privDER := x509.MarshalPKCS1PrivateKey(key)
	privatePEM := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER}))
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey() error = %v", err)
	}
	publicPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	signer, err := servicetoken.NewSigner(privatePEM, "kid-1", "gateway", "halcyon-internal", time.Minute)
	if err != nil {
		t.Fatalf("NewSigner() error = %v", err)
	}
	validator, err := servicetoken.NewValidator(publicPEM, "gateway", "halcyon-internal")
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}
	token, err := signer.Sign("gateway")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	return platformauth.NewGatewayTrust(validator, platformauth.NoopMetrics{}), token
}

func TestAPIKeyHeadersRequireGatewayTrust(t *testing.T) {
	t.Parallel()
	trust, token := testServiceToken(t)
	called := false
	handler := httpmiddleware.Authenticate(httpmiddleware.AuthenticateDeps{
		Validator:    stubValidator{},
		GatewayTrust: trust,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
		if !ok || !principal.HasPermission(domain.PermissionIntegrationsWrite) {
			t.Fatal("expected integrations.write permission")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhook-endpoints", nil)
	req.Header.Set("X-Halcyon-API-Key-ID", "key_test123")
	req.Header.Set("X-Halcyon-API-Key-Scopes", "integrations.write")
	req.Header.Set(servicetoken.HeaderName, token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected handler invocation")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestAPIKeyHeadersRejectedWithoutServiceToken(t *testing.T) {
	t.Parallel()
	trust, _ := testServiceToken(t)
	handler := httpmiddleware.Authenticate(httpmiddleware.AuthenticateDeps{
		Validator:    stubValidator{},
		GatewayTrust: trust,
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhook-endpoints", nil)
	req.Header.Set("X-Halcyon-API-Key-ID", "key_spoofed")
	req.Header.Set("X-Halcyon-API-Key-Scopes", "integrations.write")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
