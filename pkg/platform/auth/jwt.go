package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// HS256Config configures symmetric JWT validation.
type HS256Config struct {
	Secret   []byte
	Issuer   string
	Audience string
	// AllowedAlgorithm pins the signing algorithm; defaults to HS256 when empty.
	AllowedAlgorithm string
	// ClockSkew allows small clock drift for exp/nbf validation.
	ClockSkew time.Duration
}

// AccessClaims are the standard Halcyon user access token claims.
type AccessClaims struct {
	UserID      string   `json:"uid"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"perms"`
	jwt.RegisteredClaims
}

// HS256Validator validates bearer JWT access tokens with pinned algorithms.
type HS256Validator struct {
	cfg HS256Config
}

// NewHS256Validator creates a validator from configuration.
func NewHS256Validator(cfg HS256Config) (*HS256Validator, error) {
	if len(cfg.Secret) < 32 {
		return nil, fmt.Errorf("jwt secret must be at least 32 bytes")
	}
	if strings.TrimSpace(cfg.Issuer) == "" {
		return nil, fmt.Errorf("jwt issuer is required")
	}
	if cfg.AllowedAlgorithm == "" {
		cfg.AllowedAlgorithm = jwt.SigningMethodHS256.Alg()
	}
	if cfg.ClockSkew <= 0 {
		cfg.ClockSkew = 30 * time.Second
	}
	return &HS256Validator{cfg: cfg}, nil
}

// ParseBearer extracts the raw token from an Authorization header.
func ParseBearer(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", &AuthError{Category: FailureMissingCredential, Err: ErrUnauthorized}
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if raw == "" {
		return "", &AuthError{Category: FailureMissingCredential, Err: ErrUnauthorized}
	}
	return raw, nil
}

// ValidateBearer parses and validates a bearer Authorization header.
func (v *HS256Validator) ValidateBearer(bearerHeader string) (AccessClaims, error) {
	raw, err := ParseBearer(bearerHeader)
	if err != nil {
		return AccessClaims{}, err
	}
	return v.ValidateToken(raw)
}

// ValidateToken parses and validates a raw JWT string.
func (v *HS256Validator) ValidateToken(tokenString string) (AccessClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return AccessClaims{}, &AuthError{Category: FailureMissingCredential, Err: ErrUnauthorized}
	}

	opts := []jwt.ParserOption{
		jwt.WithIssuer(v.cfg.Issuer),
		jwt.WithLeeway(v.cfg.ClockSkew),
		jwt.WithValidMethods([]string{v.cfg.AllowedAlgorithm}),
	}
	if audience := strings.TrimSpace(v.cfg.Audience); audience != "" {
		opts = append(opts, jwt.WithAudience(audience))
	}

	parsed, err := jwt.ParseWithClaims(tokenString, &AccessClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method == nil || t.Method.Alg() != v.cfg.AllowedAlgorithm {
			return nil, &AuthError{Category: FailureWrongAlgorithm, Err: ErrInvalidCredential}
		}
		return v.cfg.Secret, nil
	}, opts...)
	if err != nil {
		return AccessClaims{}, mapJWTParseError(err)
	}

	claims, ok := parsed.Claims.(*AccessClaims)
	if !ok || !parsed.Valid {
		return AccessClaims{}, &AuthError{Category: FailureInvalidToken, Err: ErrInvalidCredential}
	}
	if strings.TrimSpace(claims.Subject) == "" && strings.TrimSpace(claims.UserID) == "" {
		return AccessClaims{}, &AuthError{Category: FailureMissingSubject, Err: ErrInvalidCredential}
	}
	return *claims, nil
}

func mapJWTParseError(err error) error {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return &AuthError{Category: FailureExpiredToken, Err: ErrUnauthorized}
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return &AuthError{Category: FailureNotBefore, Err: ErrUnauthorized}
	case errors.Is(err, jwt.ErrTokenMalformed):
		return &AuthError{Category: FailureMalformedToken, Err: ErrInvalidCredential}
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return &AuthError{Category: FailureInvalidToken, Err: ErrInvalidCredential}
	case errors.Is(err, jwt.ErrTokenUnverifiable):
		return &AuthError{Category: FailureWrongAlgorithm, Err: ErrInvalidCredential}
	default:
		msg := strings.ToLower(err.Error())
		switch {
		case strings.Contains(msg, "issuer"):
			return &AuthError{Category: FailureWrongIssuer, Err: ErrInvalidCredential}
		case strings.Contains(msg, "audience"):
			return &AuthError{Category: FailureWrongAudience, Err: ErrInvalidCredential}
		default:
			return &AuthError{Category: FailureInvalidToken, Err: ErrInvalidCredential}
		}
	}
}
