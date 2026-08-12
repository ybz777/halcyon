package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	jwtinfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/jwt"
)

func TestAuthHandler_LoginAndMe(t *testing.T) {
	jwtService, err := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	user := testUser(t, "handler@example.com")
	loginUC := appauth.NewLoginUseCase(
		stubUserRepo{user: user},
		stubRoleRepo{},
		stubRefreshRepo{},
		stubHasher{ok: true},
		jwtService,
		jwtService,
		nil,
		nil,
		nil,
		nil,
	)
	validateUC := appauth.NewValidateTokenUseCase(jwtService, nil)

	router := NewRouter(RouterDeps{
		ServiceName:   "auth",
		Metrics:       testMetrics,
		Login:         NewLoginHandler(loginUC),
		Auth:          NewAuthHandler(nil),
		ValidateToken: validateUC,
		AuthEnabled:   true,
	})

	rec := postJSON(t, router, "/api/v1/auth/login", `{"email":"handler@example.com","password":"password"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var tokenResp struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status = %d, body = %s", meRec.Code, meRec.Body.String())
	}
}

func TestAuthHandler_Me_Unauthorized(t *testing.T) {
	router := NewRouter(RouterDeps{
		ServiceName:   "auth",
		Metrics:       testMetrics,
		Auth:          NewAuthHandler(nil),
		ValidateToken: appauth.NewValidateTokenUseCase(nilValidator{}, nil),
		AuthEnabled:   true,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

type nilValidator struct{}

func (nilValidator) ValidateAccessToken(context.Context, domain.AccessToken) (ports.TokenClaims, error) {
	return ports.TokenClaims{}, domain.ErrInvalidAccessToken
}
