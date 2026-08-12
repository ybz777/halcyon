package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func BenchmarkHS256ValidateBearer(b *testing.B) {
	secret := []byte("benchmark-jwt-secret-at-least-32b")
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, AccessClaims{
		UserID: "usr_benchmark00000001",
		Email:  "bench@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "auth",
			Subject:   "usr_benchmark00000001",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	})
	signed, err := token.SignedString(secret)
	if err != nil {
		b.Fatal(err)
	}
	validator, err := NewHS256Validator(HS256Config{Secret: secret, Issuer: "auth"})
	if err != nil {
		b.Fatal(err)
	}
	bearer := "Bearer " + signed
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.ValidateBearer(bearer)
	}
}

func BenchmarkStaticAuthorizer_Can(b *testing.B) {
	authz := StaticAuthorizer{}
	identity := Identity{
		SubjectID:   "usr_benchmark00000001",
		Permissions: []string{"organization.read", "members.read", "members.update"},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = authz.Can(identity, "members.read")
	}
}
