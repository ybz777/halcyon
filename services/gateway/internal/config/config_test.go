package config_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"github.com/halcyon-io/halcyon/services/gateway/internal/config"
)

func testPrivateKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}))
}

func TestLoad_MissingRequired(t *testing.T) {
	t.Parallel()
	os.Clearenv()
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoad_DerivesJWKSURL(t *testing.T) {
	t.Parallel()
	os.Clearenv()
	t.Setenv("AUTH_SERVICE_URL", "http://auth:8080")
	t.Setenv("ORG_SERVICE_URL", "http://org:8081")
	t.Setenv("SERVICE_PRIVATE_KEY", testPrivateKeyPEM(t))
	t.Setenv("RATE_LIMIT_ENABLED", "false")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.JWKSURL != "http://auth:8080/api/v1/auth/jwks" {
		t.Fatalf("jwks url = %q", cfg.JWKSURL)
	}
}

func TestLoad_RejectsInvalidUpstreamURL(t *testing.T) {
	t.Parallel()
	os.Clearenv()
	t.Setenv("AUTH_SERVICE_URL", "ftp://auth:8080")
	t.Setenv("ORG_SERVICE_URL", "http://org:8081")
	t.Setenv("SERVICE_PRIVATE_KEY", testPrivateKeyPEM(t))
	t.Setenv("RATE_LIMIT_ENABLED", "false")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected invalid auth url error")
	}
}
