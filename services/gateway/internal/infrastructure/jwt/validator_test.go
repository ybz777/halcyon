package jwt_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	jwtinfra "github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/jwt"
	jwksclient "github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/jwks"
)

func TestValidator_RejectsUnsignedJWT(t *testing.T) {
	t.Parallel()
	validator := newTestValidator(t)
	_, err := validator.Validate(context.Background(), "not-a-jwt")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidator_RejectsWrongIssuer(t *testing.T) {
	t.Parallel()
	validator, secret := newTestValidatorWithSecret(t)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": "usr_1",
		"iss": "wrong",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	token.Header["kid"] = "kid-1"
	signed, _ := token.SignedString(secret)
	_, err := validator.Validate(context.Background(), signed)
	if err == nil {
		t.Fatal("expected issuer error")
	}
}

func TestValidator_ParsesPermissionsAndRoles(t *testing.T) {
	t.Parallel()

	validator, secret := newTestValidatorWithSecret(t)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":   "usr_admin",
		"email": "admin@example.com",
		"roles": []string{"admin"},
		"perms": []string{"apikey:manage", "profile:read"},
		"iss":   "auth",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	token.Header["kid"] = "kid-1"
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	claims, err := validator.Validate(context.Background(), signed)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.UserID != "usr_admin" {
		t.Fatalf("UserID = %q", claims.UserID)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "admin" {
		t.Fatalf("Roles = %#v", claims.Roles)
	}
	if len(claims.Permissions) != 2 || claims.Permissions[0] != "apikey:manage" || claims.Permissions[1] != "profile:read" {
		t.Fatalf("Permissions = %#v", claims.Permissions)
	}
}

func newTestValidator(t *testing.T) *jwtinfra.Validator {
	t.Helper()
	validator, _ := newTestValidatorWithSecret(t)
	return validator
}

func newTestValidatorWithSecret(t *testing.T) (*jwtinfra.Validator, []byte) {
	t.Helper()
	secret := []byte("super-secret-key-for-jwt-validator-test")
	encoded := base64.RawURLEncoding.EncodeToString(secret)
	body, _ := json.Marshal(map[string]any{
		"keys": []map[string]string{{"kty": "oct", "kid": "kid-1", "alg": "HS256", "k": encoded}},
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	client := jwksclient.NewClient(server.URL, time.Minute, server.Client(), nil)
	_ = client.Refresh(context.Background())
	return jwtinfra.NewValidator("auth", "", client), secret
}
