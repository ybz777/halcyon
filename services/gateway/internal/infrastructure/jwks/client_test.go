package jwks_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/jwks"
)

func TestClient_FetchesAndCachesKeys(t *testing.T) {
	t.Parallel()
	secret := []byte("super-secret-key-for-jwks-test-123456")
	encoded := base64.RawURLEncoding.EncodeToString(secret)
	body, _ := json.Marshal(map[string]any{
		"keys": []map[string]string{{
			"kty": "oct",
			"kid": "kid-1",
			"alg": "HS256",
			"k":   encoded,
		}},
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	client := jwks.NewClient(server.URL, time.Minute, server.Client(), nil)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "user"})
	token.Header["kid"] = "kid-1"
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	parsed, err := jwt.Parse(signed, client.KeyFunc(context.Background()))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !parsed.Valid {
		t.Fatal("expected valid token")
	}
}
