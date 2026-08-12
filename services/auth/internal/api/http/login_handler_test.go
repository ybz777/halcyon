package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	jwtinfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/jwt"
)

func newTestLoginHandler(t *testing.T, user *domain.User, hasherOK bool) *LoginHandler {
	t.Helper()

	jwtService, err := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", 15*time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	loginUC := appauth.NewLoginUseCase(
		stubUserRepo{user: user},
		stubRoleRepo{},
		stubRefreshRepo{},
		stubHasher{ok: hasherOK},
		jwtService,
		jwtService,
		nil,
		nil,
		nil,
		nil,
	)
	return NewLoginHandler(loginUC)
}

func TestLoginHandler_Success(t *testing.T) {
	user := testUser(t, "login@example.com")
	handler := newTestLoginHandler(t, user, true)
	router := testAuthRouter(t, handler, nil, nil)

	rec := postJSON(t, router, "/api/v1/auth/login", `{"email":"login@example.com","password":"password"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, key := range []string{"accessToken", "refreshToken", "expiresIn", "tokenType"} {
		if resp[key] == nil || resp[key] == "" {
			t.Fatalf("missing response field %q: %+v", key, resp)
		}
	}
	if resp["tokenType"] != "Bearer" {
		t.Fatalf("tokenType = %v", resp["tokenType"])
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	user := testUser(t, "login@example.com")
	handler := newTestLoginHandler(t, user, false)
	router := testAuthRouter(t, handler, nil, nil)

	rec := postJSON(t, router, "/api/v1/auth/login", `{"email":"login@example.com","password":"wrong"}`)
	assertProblemStatus(t, rec, http.StatusUnauthorized)
}

func TestLoginHandler_UnknownEmail(t *testing.T) {
	handler := newTestLoginHandler(t, nil, true)
	router := testAuthRouter(t, handler, nil, nil)

	rec := postJSON(t, router, "/api/v1/auth/login", `{"email":"missing@example.com","password":"password"}`)
	assertProblemStatus(t, rec, http.StatusUnauthorized)
}

func TestLoginHandler_MalformedJSON(t *testing.T) {
	handler := newTestLoginHandler(t, testUser(t, "login@example.com"), true)
	router := testAuthRouter(t, handler, nil, nil)

	rec := postJSON(t, router, "/api/v1/auth/login", `{invalid`)
	assertProblemStatus(t, rec, http.StatusBadRequest)
}

func TestLoginHandler_MissingEmail(t *testing.T) {
	handler := newTestLoginHandler(t, testUser(t, "login@example.com"), true)
	router := testAuthRouter(t, handler, nil, nil)

	rec := postJSON(t, router, "/api/v1/auth/login", `{"email":"","password":"password"}`)
	assertProblemStatus(t, rec, http.StatusBadRequest)
}

func TestLoginHandler_MissingPassword(t *testing.T) {
	handler := newTestLoginHandler(t, testUser(t, "login@example.com"), true)
	router := testAuthRouter(t, handler, nil, nil)

	rec := postJSON(t, router, "/api/v1/auth/login", `{"email":"login@example.com","password":""}`)
	assertProblemStatus(t, rec, http.StatusBadRequest)
}

func testUser(t *testing.T, email string) *domain.User {
	t.Helper()
	id, _ := domain.ParseUserID("usr_handlerlogintest1")
	parsedEmail, _ := domain.ParseEmail(email)
	hash, _ := domain.ParsePasswordHash(validBcryptHash)
	user, err := domain.RegisterUser(id, parsedEmail, hash, time.Now().UTC())
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}
	return user
}

func testAuthRouter(t *testing.T, login *LoginHandler, refresh *RefreshHandler, auth *AuthHandler) http.Handler {
	t.Helper()
	if auth == nil {
		auth = NewAuthHandler(nil)
	}
	return NewRouter(RouterDeps{
		ServiceName:   "auth",
		Metrics:       testMetrics,
		Login:         login,
		Refresh:       refresh,
		Auth:          auth,
		ValidateToken: appauth.NewValidateTokenUseCase(nilValidator{}, nil),
		AuthEnabled:   true,
	})
}

func postJSON(t *testing.T, router http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertProblemStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, want, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", ct)
	}
}

type stubUserRepo struct{ user *domain.User }

func (s stubUserRepo) Create(context.Context, *domain.User) error { return nil }
func (s stubUserRepo) Update(context.Context, *domain.User) error { return nil }
func (s stubUserRepo) Delete(context.Context, domain.UserID) error {
	return nil
}
func (s stubUserRepo) FindByID(context.Context, domain.UserID) (*domain.User, error) {
	return s.user, nil
}
func (s stubUserRepo) FindByEmail(context.Context, domain.Email) (*domain.User, error) {
	if s.user == nil {
		return nil, domain.ErrUserNotFound
	}
	return s.user, nil
}
func (s stubUserRepo) ExistsByEmail(context.Context, domain.Email) (bool, error) {
	return s.user != nil, nil
}

type stubRoleRepo struct{}

func (stubRoleRepo) FindRolesByUserID(context.Context, domain.UserID) ([]domain.Role, error) {
	return nil, nil
}

func (stubRoleRepo) AssignRole(context.Context, domain.UserID, domain.Role) error {
	return nil
}

type stubRefreshRepo struct{}

func (stubRefreshRepo) Save(context.Context, *domain.RefreshTokenRecord) error { return nil }
func (stubRefreshRepo) FindByJTI(context.Context, domain.JTI) (*domain.RefreshTokenRecord, error) {
	return nil, domain.ErrInvalidRefreshToken
}
func (stubRefreshRepo) Consume(context.Context, domain.JTI) (*domain.RefreshTokenRecord, error) {
	return nil, domain.ErrInvalidRefreshToken
}
func (stubRefreshRepo) RevokeFamily(context.Context, domain.FamilyID) error { return nil }
func (stubRefreshRepo) RevokeByJTI(context.Context, domain.JTI) error       { return nil }
func (stubRefreshRepo) DeleteExpired(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (stubRefreshRepo) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type stubHasher struct{ ok bool }

func (s stubHasher) Hash(context.Context, string) (domain.PasswordHash, error) {
	return domain.PasswordHash{}, nil
}
func (s stubHasher) Verify(context.Context, string, domain.PasswordHash) (bool, error) {
	return s.ok, nil
}

const validBcryptHash = "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/X4.G2oYbOuSO.8K6O"
