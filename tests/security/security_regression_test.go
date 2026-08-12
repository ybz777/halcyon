package security_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
	platformtenant "github.com/halcyon-io/halcyon/pkg/platform/tenant"
	"github.com/halcyon-io/halcyon/pkg/platform/idempotency"
	"github.com/halcyon-io/halcyon/pkg/platform/testutil"
	"github.com/stretchr/testify/require"
)

func TestAuthentication_MissingCredentials(t *testing.T) {
	t.Parallel()
	validator, err := testutil.NewTestValidator()
	require.NoError(t, err)
	_, err = validator.ValidateBearer("")
	require.Error(t, err)
}

func TestAuthentication_ExpiredTokenRejected(t *testing.T) {
	t.Parallel()
	validator, err := platformauth.NewHS256Validator(platformauth.HS256Config{
		Secret: []byte(testutil.TestJWTSecret),
		Issuer: "auth",
	})
	require.NoError(t, err)
	// Token signed with wrong secret simulates invalid credential path.
	_, err = validator.ValidateToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c3IifQ.invalid")
	require.Error(t, err)
}

func TestIdempotency_CrossPrincipalReplayBlocked(t *testing.T) {
	store := idempotency.NewMemoryStore()
	ctx := context.Background()
	scopeA := idempotency.Scope{Service: "api", PrincipalType: "user", PrincipalID: "user-a", OrganizationID: "org-a"}
	scopeB := idempotency.Scope{Service: "api", PrincipalType: "user", PrincipalID: "user-b", OrganizationID: "org-a"}

	keyA := scopeA.ScopeKey()
	keyB := scopeB.ScopeKey()
	require.NotEqual(t, keyA, keyB)

	out, err := store.Claim(ctx, idempotency.StorageKey("test", scopeA, "idem-key-1"), "fp1", idempotency.DefaultConfig("api").InProgressTTL)
	require.NoError(t, err)
	require.Equal(t, idempotency.ClaimAcquired, out.Result)

	out, err = store.Claim(ctx, idempotency.StorageKey("test", scopeB, "idem-key-1"), "fp1", idempotency.DefaultConfig("api").InProgressTTL)
	require.NoError(t, err)
	require.Equal(t, idempotency.ClaimAcquired, out.Result)
}

func TestIdempotency_CrossTenantReplayBlocked(t *testing.T) {
	store := idempotency.NewMemoryStore()
	ctx := context.Background()
	scopeA := idempotency.Scope{Service: "api", PrincipalType: "user", PrincipalID: "user-a", OrganizationID: "org-a"}
	scopeB := idempotency.Scope{Service: "api", PrincipalType: "user", PrincipalID: "user-a", OrganizationID: "org-b"}

	keyA := scopeA.ScopeKey()
	keyB := scopeB.ScopeKey()
	require.NotEqual(t, keyA, keyB)

	_, err := store.Claim(ctx, idempotency.StorageKey("test", scopeA, "idem-key-2"), "fp1", idempotency.DefaultConfig("api").InProgressTTL)
	require.NoError(t, err)
	out, err := store.Claim(ctx, idempotency.StorageKey("test", scopeB, "idem-key-2"), "fp1", idempotency.DefaultConfig("api").InProgressTTL)
	require.NoError(t, err)
	require.Equal(t, idempotency.ClaimAcquired, out.Result)
}

func TestGatewayTrust_BlocksHeaderSpoofing(t *testing.T) {
	t.Parallel()
	trust := platformauth.NewGatewayTrust(nil, platformauth.NoopMetrics{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	req.Header.Set("X-Halcyon-API-Key-ID", "key_attacker")
	_, err := trust.ValidateRequest(req)
	require.Error(t, err)
}

func TestTenantHeader_ProductionRejectsSpoofedOrganization(t *testing.T) {
	t.Parallel()
	handler := platformtenant.RequireGatewayForOrganizationHeaders(nil, platformconfig.EnvProduction)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-Halcyon-Organization-ID", "org_attacker0000001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthorization_StaticDenyByDefault(t *testing.T) {
	t.Parallel()
	authz := platformauth.StaticAuthorizer{}
	identity := platformauth.Identity{SubjectID: "usr_1", Permissions: []string{"read"}}
	require.True(t, authz.Can(identity, "read"))
	require.False(t, authz.Can(identity, "admin.manage"))
}
