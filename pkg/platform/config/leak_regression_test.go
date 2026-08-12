package config_test

import (
	"encoding/json"
	"testing"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
	"github.com/halcyon-io/halcyon/pkg/platform/testutil"
	"github.com/stretchr/testify/require"
)

func TestSecretLeakRegression_SummaryAndJSON(t *testing.T) {
	t.Parallel()
	cleanup := platformconfig.SetTestEnv(map[string]string{
		"APP_ENV":    "test",
		"JWT_SECRET": testutil.TestJWTSecret,
		"DATABASE_URL": "postgres://user:realpassword@localhost:5432/db",
	})
	defer cleanup()

	env := platformconfig.FromEnv()
	auth, err := platformconfig.LoadAuth(env)
	require.NoError(t, err)

	raw, err := json.Marshal(struct {
		Secret platformconfig.Secret `json:"secret"`
	}{Secret: auth.JWTSecret})
	require.NoError(t, err)
	require.NotContains(t, string(raw), testutil.TestJWTSecret)
	require.Contains(t, string(raw), "[REDACTED]")

	summary := platformconfig.NewSummary(platformconfig.LoadApp(env, "test"))
	out, err := json.Marshal(summary)
	require.NoError(t, err)
	require.NotContains(t, string(out), "realpassword")
}

func TestValidateAuth_RejectsShortSecretInTest(t *testing.T) {
	t.Parallel()
	cleanup := platformconfig.SetTestEnv(map[string]string{
		"JWT_SECRET": "short",
	})
	defer cleanup()
	env := platformconfig.FromEnv()
	auth, err := platformconfig.LoadAuth(env)
	require.NoError(t, err)
	err = platformconfig.ValidateAuth(auth, true, env)
	require.Error(t, err)
}
