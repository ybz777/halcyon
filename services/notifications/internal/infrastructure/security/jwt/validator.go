package jwt

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
)

type Principal struct {
	UserID         string
	OrganizationID string
}

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
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

func (v *Validator) Validate(_ context.Context, bearerToken string) (Principal, error) {
	token, err := extractBearer(bearerToken)
	if err != nil {
		return Principal{}, domain.ErrUnauthorized
	}
	parsed, err := jwt.ParseWithClaims(token, &accessClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, domain.ErrUnauthorized
		}
		return v.secret, nil
	}, jwt.WithIssuer(v.issuer))
	if err != nil {
		return Principal{}, domain.ErrUnauthorized
	}
	claims, ok := parsed.Claims.(*accessClaims)
	if !ok || !parsed.Valid || claims.UserID == "" {
		return Principal{}, domain.ErrUnauthorized
	}
	return Principal{UserID: claims.UserID}, nil
}

func extractBearer(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", domain.ErrUnauthorized
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if raw == "" {
		return "", domain.ErrUnauthorized
	}
	return raw, nil
}
