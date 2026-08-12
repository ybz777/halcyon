package servicetoken_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/servicetoken"
)

func testKeyPair(t *testing.T) (privatePEM, publicPEM string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	privDER := x509.MarshalPKCS1PrivateKey(key)
	privatePEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER}))
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey() error = %v", err)
	}
	publicPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	return privatePEM, publicPEM
}

func TestServiceTokenRoundTrip(t *testing.T) {
	t.Parallel()
	priv, pub := testKeyPair(t)
	signer, err := servicetoken.NewSigner(priv, "kid-1", "gateway", "halcyon-internal", time.Minute)
	if err != nil {
		t.Fatalf("NewSigner() error = %v", err)
	}
	validator, err := servicetoken.NewValidator(pub, "gateway", "halcyon-internal")
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}
	token, err := signer.Sign("gateway")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	claims, err := validator.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.ServiceID != "gateway" {
		t.Fatalf("service id = %q", claims.ServiceID)
	}
}

func TestServiceTokenWrongAudience(t *testing.T) {
	t.Parallel()
	priv, pub := testKeyPair(t)
	signer, _ := servicetoken.NewSigner(priv, "kid-1", "gateway", "halcyon-internal", time.Minute)
	validator, _ := servicetoken.NewValidator(pub, "gateway", "other-audience")
	token, _ := signer.Sign("gateway")
	if _, err := validator.Validate(token); err == nil {
		t.Fatal("expected validation failure")
	}
}
