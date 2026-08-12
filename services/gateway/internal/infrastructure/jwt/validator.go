package jwt

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/halcyon-io/halcyon/services/gateway/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/gateway/internal/domain"
	jwksclient "github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/jwks"
)

type accessClaims struct {
	UserID      string   `json:"uid"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"perms"`
	jwt.RegisteredClaims
}

// Validator validates JWT access tokens using JWKS.
type Validator struct {
	issuer   string
	audience string
	jwks     *jwksclient.Client
}

// NewValidator creates a JWT validator.
func NewValidator(issuer, audience string, jwks *jwksclient.Client) *Validator {
	return &Validator{issuer: issuer, audience: audience, jwks: jwks}
}

// Validate parses and validates a bearer token.
func (v *Validator) Validate(ctx context.Context, tokenString string) (ports.TokenClaims, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return ports.TokenClaims{}, domain.ErrUnauthorized
	}

	opts := []jwt.ParserOption{jwt.WithIssuer(v.issuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})}
	if v.audience != "" {
		opts = append(opts, jwt.WithAudience(v.audience))
	}

	parsed, err := jwt.ParseWithClaims(tokenString, &accessClaims{}, v.jwks.KeyFunc(ctx), opts...)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return ports.TokenClaims{}, domain.ErrTokenExpired
		}
		return ports.TokenClaims{}, domain.ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*accessClaims)
	if !ok || !parsed.Valid {
		return ports.TokenClaims{}, domain.ErrInvalidToken
	}
	if claims.UserID == "" {
		return ports.TokenClaims{}, domain.ErrInvalidToken
	}
	return ports.TokenClaims{
		UserID:      claims.UserID,
		Email:       claims.Email,
		Roles:       normalizeStrings(claims.Roles),
		Permissions: normalizeStrings(claims.Permissions),
	}, nil
}

func normalizeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

// IsAPIKeyToken reports whether the bearer value is an API key.
func IsAPIKeyToken(token string) bool {
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	return strings.HasPrefix(token, "hal_live_") || strings.HasPrefix(token, "hal_test_")
}

// ExtractBearer extracts the token from an Authorization header.
func ExtractBearer(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", fmt.Errorf("%w", domain.ErrUnauthorized)
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix)), nil
}
