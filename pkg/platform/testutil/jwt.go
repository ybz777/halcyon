package testutil

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
)

// SignTestAccessToken issues a valid HS256 access token for tests.
func SignTestAccessToken(userID, email string, perms []string) (string, error) {
	now := time.Now().UTC()
	claims := platformauth.AccessClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "auth",
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	if len(perms) > 0 {
		claims.Permissions = perms
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(TestJWTSecret))
}

// NewTestValidator returns an HS256 validator using the test secret.
func NewTestValidator() (*platformauth.HS256Validator, error) {
	return platformauth.NewHS256Validator(platformauth.HS256Config{
		Secret: []byte(TestJWTSecret),
		Issuer: "auth",
	})
}
