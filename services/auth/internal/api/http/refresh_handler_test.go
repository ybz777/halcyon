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

func newTestRefreshHandler(t *testing.T, repo ports.RefreshTokenRepository) *RefreshHandler {
	t.Helper()

	jwtService, err := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", 15*time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	user := testUser(t, "refresh@example.com")
	refreshUC := appauth.NewRefreshTokenUseCase(
		stubUserRepo{user: user},
		stubRoleRepo{},
		repo,
		jwtService,
		jwtService,
		nil,
	)
	return NewRefreshHandler(refreshUC)
}

func TestRefreshHandler_Success(t *testing.T) {
	ctx := context.Background()
	jwtService, err := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", 15*time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	user := testUser(t, "refresh@example.com")
	repo := &httpRefreshRepo{records: make(map[string]*domain.RefreshTokenRecord)}
	family, _ := domain.NewFamilyID()
	refreshToken, claims, err := jwtService.IssueRefreshToken(ctx, family, nil, user.ID())
	if err != nil {
		t.Fatalf("IssueRefreshToken() error = %v", err)
	}
	record, err := domain.NewRefreshTokenRecord(
		claims.JTI, claims.FamilyID, nil, claims.UserID,
		domain.HashRefreshToken(refreshToken), claims.IssuedAt, claims.ExpiresAt, "", "",
	)
	if err != nil {
		t.Fatalf("NewRefreshTokenRecord() error = %v", err)
	}
	repo.records[claims.JTI.String()] = record

	handler := newTestRefreshHandler(t, repo)
	router := NewRouter(RouterDeps{ServiceName: "auth", Metrics: testMetrics, Refresh: handler, AuthEnabled: true})

	body, _ := json.Marshal(map[string]string{"refreshToken": refreshToken.String()})
	rec := postJSON(t, router, "/api/v1/auth/refresh", string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestRefreshHandler_ReplayDetected(t *testing.T) {
	ctx := context.Background()
	jwtService, err := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", 15*time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	user := testUser(t, "refresh@example.com")
	repo := &httpRefreshRepo{records: make(map[string]*domain.RefreshTokenRecord)}
	family, _ := domain.NewFamilyID()
	refreshToken, claims, err := jwtService.IssueRefreshToken(ctx, family, nil, user.ID())
	if err != nil {
		t.Fatalf("IssueRefreshToken() error = %v", err)
	}
	record, err := domain.NewRefreshTokenRecord(
		claims.JTI, claims.FamilyID, nil, claims.UserID,
		domain.HashRefreshToken(refreshToken), claims.IssuedAt, claims.ExpiresAt, "", "",
	)
	if err != nil {
		t.Fatalf("NewRefreshTokenRecord() error = %v", err)
	}
	_ = record.MarkConsumed(time.Now().UTC())
	repo.records[claims.JTI.String()] = record

	handler := newTestRefreshHandler(t, repo)
	router := NewRouter(RouterDeps{ServiceName: "auth", Metrics: testMetrics, Refresh: handler, AuthEnabled: true})

	body, _ := json.Marshal(map[string]string{"refreshToken": refreshToken.String()})
	rec := postJSON(t, router, "/api/v1/auth/refresh", string(body))
	assertProblemStatus(t, rec, http.StatusUnauthorized)

	var problem map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem["title"] != "Refresh Token Reuse Detected" {
		t.Fatalf("title = %v", problem["title"])
	}
}

func TestRefreshHandler_InvalidToken(t *testing.T) {
	handler := newTestRefreshHandler(t, stubRefreshRepo{})
	router := NewRouter(RouterDeps{ServiceName: "auth", Metrics: testMetrics, Refresh: handler, AuthEnabled: true})
	rec := postJSON(t, router, "/api/v1/auth/refresh", `{"refreshToken":"short"}`)
	assertProblemStatus(t, rec, http.StatusUnauthorized)
}

func TestRefreshHandler_MissingToken(t *testing.T) {
	handler := newTestRefreshHandler(t, stubRefreshRepo{})
	router := NewRouter(RouterDeps{ServiceName: "auth", Metrics: testMetrics, Refresh: handler, AuthEnabled: true})
	rec := postJSON(t, router, "/api/v1/auth/refresh", `{"refreshToken":""}`)
	assertProblemStatus(t, rec, http.StatusBadRequest)
}

func TestMetricsEndpoint_StillExposed(t *testing.T) {
	router := NewRouter(RouterDeps{ServiceName: "auth", Metrics: testMetrics, AuthEnabled: false})
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics status = %d", rec.Code)
	}
}
