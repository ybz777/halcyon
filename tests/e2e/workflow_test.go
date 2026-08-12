//go:build e2e

package e2e_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	"github.com/halcyon-io/halcyon/pkg/platform/testutil"
	"github.com/stretchr/testify/require"
)

func TestE2E_AuthenticationAndHealth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	baseURL := os.Getenv("AUTH_SERVICE_URL")
	if baseURL == "" {
		t.Skip("AUTH_SERVICE_URL not set")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/health", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	validator, err := platformauth.NewHS256Validator(platformauth.HS256Config{
		Secret: []byte(os.Getenv("JWT_SECRET")),
		Issuer: "auth",
	})
	if os.Getenv("JWT_SECRET") == "" {
		validator, err = testutil.NewTestValidator()
	}
	require.NoError(t, err)

	token, err := testutil.SignTestAccessToken("usr_e2e_test01", "e2e@example.com", nil)
	require.NoError(t, err)
	claims, err := validator.ValidateBearer("Bearer " + token)
	if os.Getenv("JWT_SECRET") != "" && os.Getenv("JWT_SECRET") != testutil.TestJWTSecret {
		t.Skip("local JWT secret differs from test signer")
	}
	require.NoError(t, err)
	require.Equal(t, "usr_e2e_test01", claims.UserID)
}

func TestE2E_OrganizationServiceHealth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	baseURL := os.Getenv("ORG_SERVICE_URL")
	if baseURL == "" {
		t.Skip("ORG_SERVICE_URL not set")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/health", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
