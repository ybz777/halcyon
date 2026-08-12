package jwt

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/halcyon-io/halcyon/services/jobs/internal/application/ports"
)

// Validator validates auth service JWT access tokens.
type Validator struct {
	secret []byte
	issuer string
}

func NewValidator(secret []byte, issuer string) (*Validator, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("jwt secret must be at least 32 bytes")
	}
	return &Validator{secret: secret, issuer: issuer}, nil
}

type accessClaims struct {
	UserID      string   `json:"uid"`
	Permissions []string `json:"perms"`
	jwt.RegisteredClaims
}

func (v *Validator) Validate(_ context.Context, bearerToken string) (ports.Principal, error) {
	token, err := extractBearer(bearerToken)
	if err != nil {
		return ports.Principal{}, err
	}
	parsed, err := jwt.ParseWithClaims(token, &accessClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("invalid signing method")
		}
		return v.secret, nil
	}, jwt.WithIssuer(v.issuer))
	if err != nil {
		return ports.Principal{}, err
	}
	claims, ok := parsed.Claims.(*accessClaims)
	if !ok || !parsed.Valid {
		return ports.Principal{}, errors.New("invalid token")
	}
	perms := make(map[string]struct{}, len(claims.Permissions))
	for _, perm := range claims.Permissions {
		perms[strings.ToLower(strings.TrimSpace(perm))] = struct{}{}
	}
	return ports.Principal{UserID: claims.UserID, Permissions: perms}, nil
}

var _ ports.TokenValidator = (*Validator)(nil)

func extractBearer(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", errors.New("missing bearer token")
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if raw == "" {
		return "", errors.New("empty bearer token")
	}
	return raw, nil
}
