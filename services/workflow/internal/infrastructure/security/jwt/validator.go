package jwt

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// Validator validates auth service JWT access tokens.
type Validator struct {
	secret []byte
	issuer string
}

// NewValidator creates a JWT validator.
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

// Validate parses and validates a bearer token.
func (v *Validator) Validate(_ context.Context, bearerToken string) (ports.Principal, error) {
	token, err := extractBearer(bearerToken)
	if err != nil {
		return ports.Principal{}, err
	}

	parsed, err := jwt.ParseWithClaims(token, &accessClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, domain.ErrUnauthorized
		}
		return v.secret, nil
	}, jwt.WithIssuer(v.issuer))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return ports.Principal{}, domain.ErrUnauthorized
		}
		return ports.Principal{}, domain.ErrUnauthorized
	}

	claims, ok := parsed.Claims.(*accessClaims)
	if !ok || !parsed.Valid {
		return ports.Principal{}, domain.ErrUnauthorized
	}

	userID, err := domain.ParseUserID(claims.UserID)
	if err != nil {
		return ports.Principal{}, domain.ErrUnauthorized
	}

	perms := make(map[string]struct{}, len(claims.Permissions))
	for _, perm := range claims.Permissions {
		perms[strings.ToLower(strings.TrimSpace(perm))] = struct{}{}
	}

	return ports.Principal{UserID: userID, Permissions: perms}, nil
}

var _ ports.TokenValidator = (*Validator)(nil)

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
