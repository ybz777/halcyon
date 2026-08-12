package idempotency_test

import (
	"testing"

	"github.com/halcyon-io/halcyon/pkg/platform/idempotency"
	"github.com/stretchr/testify/require"
)

func BenchmarkFingerprint(b *testing.B) {
	body := []byte(`{"name":"Acme","tags":["a","b"]}`)
	in := idempotency.FingerprintInput{
		Method: "POST", RoutePattern: "/api/v1/organizations",
		PrincipalType: "user", PrincipalID: "usr_1", Body: body,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = idempotency.Fingerprint(in)
	}
}

func BenchmarkStorageKey(b *testing.B) {
	scope := idempotency.Scope{Service: "organization", PrincipalType: "user", PrincipalID: "usr_1"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = idempotency.StorageKey("organization", scope, "client-key-123")
	}
}

func TestStorageKeyStable(t *testing.T) {
	scope := idempotency.Scope{Service: "organization", PrincipalType: "user", PrincipalID: "usr_1", OrganizationID: "org_1"}
	k1 := idempotency.StorageKey("organization", scope, "abc")
	k2 := idempotency.StorageKey("organization", scope, "abc")
	require.Equal(t, k1, k2)
}
