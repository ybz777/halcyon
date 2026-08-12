package perf_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/halcyon-io/halcyon/pkg/platform/auth"
	"github.com/halcyon-io/halcyon/pkg/platform/idempotency"
	"github.com/halcyon-io/halcyon/pkg/platform/jobs"
	"github.com/halcyon-io/halcyon/pkg/platform/pagination"
)

// Broad regression thresholds (local); not production SLOs.
const (
	maxCursorEncodeNs  = 5000
	maxFingerprintNs   = 10000
	maxJWTValidateNs   = 200000
	maxBackoffDelayNs  = 5000
)

func TestRegression_CursorEncode(t *testing.T) {
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	start := time.Now()
	for i := 0; i < 1000; i++ {
		_ = pagination.EncodeTimeIDCursor(at, "aud_regression00001")
	}
	perOp := time.Since(start) / 1000
	if perOp > maxCursorEncodeNs*time.Nanosecond {
		t.Fatalf("cursor encode regression: %v per op (max %v)", perOp, maxCursorEncodeNs*time.Nanosecond)
	}
}

func TestRegression_Fingerprint(t *testing.T) {
	in := idempotency.FingerprintInput{
		Method: "POST", RoutePattern: "/api/v1/organizations",
		PrincipalType: "user", PrincipalID: "usr_1",
		Body: []byte(`{"name":"Acme"}`),
	}
	start := time.Now()
	for i := 0; i < 1000; i++ {
		_, _ = idempotency.Fingerprint(in)
	}
	perOp := time.Since(start) / 1000
	if perOp > maxFingerprintNs*time.Nanosecond {
		t.Fatalf("fingerprint regression: %v per op (max %v)", perOp, maxFingerprintNs*time.Nanosecond)
	}
}

func TestRegression_JWTValidate(t *testing.T) {
	secret := []byte("regression-jwt-secret-at-least-32b")
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.AccessClaims{
		UserID: "usr_regression00001",
		Email:  "reg@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "auth", Subject: "usr_regression00001",
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	})
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	validator, err := auth.NewHS256Validator(auth.HS256Config{Secret: secret, Issuer: "auth"})
	if err != nil {
		t.Fatal(err)
	}
	bearer := "Bearer " + signed
	start := time.Now()
	for i := 0; i < 100; i++ {
		_, _ = validator.ValidateBearer(bearer)
	}
	perOp := time.Since(start) / 100
	if perOp > maxJWTValidateNs*time.Nanosecond {
		t.Fatalf("jwt validate regression: %v per op (max %v)", perOp, maxJWTValidateNs*time.Nanosecond)
	}
}

func TestRegression_BackoffDelay(t *testing.T) {
	cfg := jobs.DefaultRetryConfig()
	start := time.Now()
	for i := 0; i < 10000; i++ {
		_ = jobs.BackoffDelay(cfg, 3)
	}
	perOp := time.Since(start) / 10000
	if perOp > maxBackoffDelayNs*time.Nanosecond {
		t.Fatalf("backoff regression: %v per op (max %v)", perOp, maxBackoffDelayNs*time.Nanosecond)
	}
}
