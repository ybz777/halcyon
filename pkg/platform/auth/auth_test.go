package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	"github.com/halcyon-io/halcyon/pkg/platform/servicetoken"
)

const testSecret = "super-secret-jwt-key-at-least-32-bytes"

func signToken(t *testing.T, claims platformauth.AccessClaims, secret []byte, method jwt.SigningMethod) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return signed
}

func validClaims() platformauth.AccessClaims {
	now := time.Now().UTC()
	return platformauth.AccessClaims{
		UserID: "usr_test0001",
		Email:  "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "auth",
			Subject:   "usr_test0001",
			Audience:  jwt.ClaimStrings{"halcyon-api"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
}

func newValidator(t *testing.T, audience string) *platformauth.HS256Validator {
	t.Helper()
	v, err := platformauth.NewHS256Validator(platformauth.HS256Config{
		Secret:   []byte(testSecret),
		Issuer:   "auth",
		Audience: audience,
	})
	if err != nil {
		t.Fatalf("NewHS256Validator() error = %v", err)
	}
	return v
}

func TestHS256Validator_ValidToken(t *testing.T) {
	t.Parallel()
	v := newValidator(t, "halcyon-api")
	token := signToken(t, validClaims(), []byte(testSecret), jwt.SigningMethodHS256)
	claims, err := v.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.UserID != "usr_test0001" {
		t.Fatalf("user id = %q", claims.UserID)
	}
}

func TestHS256Validator_ExpiredToken(t *testing.T) {
	t.Parallel()
	v := newValidator(t, "")
	claims := validClaims()
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().UTC().Add(-time.Minute))
	token := signToken(t, claims, []byte(testSecret), jwt.SigningMethodHS256)
	if _, err := v.ValidateToken(token); err == nil {
		t.Fatal("expected expired token failure")
	}
}

func TestHS256Validator_NotBefore(t *testing.T) {
	t.Parallel()
	v := newValidator(t, "")
	claims := validClaims()
	claims.NotBefore = jwt.NewNumericDate(time.Now().UTC().Add(time.Hour))
	token := signToken(t, claims, []byte(testSecret), jwt.SigningMethodHS256)
	if _, err := v.ValidateToken(token); err == nil {
		t.Fatal("expected not-before failure")
	}
}

func TestHS256Validator_WrongIssuer(t *testing.T) {
	t.Parallel()
	v := newValidator(t, "")
	claims := validClaims()
	claims.Issuer = "other"
	token := signToken(t, claims, []byte(testSecret), jwt.SigningMethodHS256)
	if _, err := v.ValidateToken(token); err == nil {
		t.Fatal("expected wrong issuer failure")
	}
}

func TestHS256Validator_WrongAudience(t *testing.T) {
	t.Parallel()
	v := newValidator(t, "halcyon-api")
	claims := validClaims()
	claims.Audience = jwt.ClaimStrings{"other"}
	token := signToken(t, claims, []byte(testSecret), jwt.SigningMethodHS256)
	if _, err := v.ValidateToken(token); err == nil {
		t.Fatal("expected wrong audience failure")
	}
}

func TestHS256Validator_WrongSignature(t *testing.T) {
	t.Parallel()
	v := newValidator(t, "")
	token := signToken(t, validClaims(), []byte("another-secret-that-is-long-enough-1234"), jwt.SigningMethodHS256)
	if _, err := v.ValidateToken(token); err == nil {
		t.Fatal("expected wrong signature failure")
	}
}

func TestHS256Validator_WrongAlgorithm(t *testing.T) {
	t.Parallel()
	v := newValidator(t, "")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	now := time.Now().UTC()
	claims := validClaims()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	_ = now
	if _, err := v.ValidateToken(signed); err == nil {
		t.Fatal("expected wrong algorithm failure")
	}
}

func TestHS256Validator_MissingSubject(t *testing.T) {
	t.Parallel()
	v := newValidator(t, "")
	claims := validClaims()
	claims.Subject = ""
	claims.UserID = ""
	token := signToken(t, claims, []byte(testSecret), jwt.SigningMethodHS256)
	if _, err := v.ValidateToken(token); err == nil {
		t.Fatal("expected missing subject failure")
	}
}

func TestHS256Validator_MalformedToken(t *testing.T) {
	t.Parallel()
	v := newValidator(t, "")
	if _, err := v.ValidateToken("not.a.jwt"); err == nil {
		t.Fatal("expected malformed token failure")
	}
}

func TestGatewayTrust_RejectsSpoofedAPIKeyHeaders(t *testing.T) {
	t.Parallel()
	trust := platformauth.NewGatewayTrust(nil, platformauth.NoopMetrics{})
	req := httptestNewRequest(t)
	req.Header.Set("X-Halcyon-API-Key-ID", "key_spoofed")
	if _, err := trust.ValidateRequest(req); err == nil {
		t.Fatal("expected untrusted gateway failure")
	}
}

func TestGatewayTrust_AcceptsValidServiceToken(t *testing.T) {
	t.Parallel()
	priv, pub := testRSAKeyPair(t)
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
	trust := platformauth.NewGatewayTrust(validator, platformauth.NoopMetrics{})
	req := httptestNewRequest(t)
	req.Header.Set("X-Halcyon-API-Key-ID", "key_test")
	req.Header.Set(servicetoken.HeaderName, token)
	serviceID, err := trust.ValidateRequest(req)
	if err != nil {
		t.Fatalf("ValidateRequest() error = %v", err)
	}
	if serviceID != "gateway" {
		t.Fatalf("service id = %q", serviceID)
	}
}

func testRSAKeyPair(t *testing.T) (privatePEM, publicPEM string) {
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

func httptestNewRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "/api/v1/webhook-endpoints", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	return req
}
