package auth

import "testing"

func TestUpstreamService_CreateOrganization(t *testing.T) {
	if got := UpstreamService("/api/v1/organizations"); got != "organization" {
		t.Fatalf("UpstreamService(/api/v1/organizations) = %q, want organization", got)
	}
}
