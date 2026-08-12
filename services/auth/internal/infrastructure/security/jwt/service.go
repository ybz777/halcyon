package jwt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// Service signs and validates JWT access and refresh tokens.
type Service struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewService creates a JWT token service.
func NewService(secret []byte, issuer string, accessTTL, refreshTTL time.Duration) (*Service, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("jwt secret must be at least 32 bytes")
	}
	return &Service{
		secret:     secret,
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}, nil
}

type accessClaims struct {
	UserID      string   `json:"uid"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"perms"`
	jwt.RegisteredClaims
}

// IssueAccessToken signs a JWT access token.
func (s *Service) IssueAccessToken(_ context.Context, claims ports.TokenClaims) (domain.AccessToken, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.accessTTL)

	roleNames := make([]string, 0, len(claims.Roles))
	for _, role := range claims.Roles {
		roleNames = append(roleNames, role.String())
	}

	permNames := make([]string, 0, len(claims.Permissions))
	for _, perm := range claims.Permissions {
		permNames = append(permNames, perm.String())
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims{
		UserID:      claims.UserID.String(),
		Email:       claims.Email.String(),
		Roles:       roleNames,
		Permissions: permNames,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   claims.UserID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	signed, err := token.SignedString(s.secret)
	if err != nil {
		return domain.AccessToken{}, time.Time{}, fmt.Errorf("sign token: %w", err)
	}

	accessToken, err := domain.ParseAccessToken(signed)
	if err != nil {
		return domain.AccessToken{}, time.Time{}, err
	}

	return accessToken, expiresAt, nil
}

// ValidateAccessToken validates a JWT access token.
func (s *Service) ValidateAccessToken(_ context.Context, token domain.AccessToken) (ports.TokenClaims, error) {
	parsed, err := jwt.ParseWithClaims(token.String(), &accessClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("%w: unexpected signing method", domain.ErrInvalidAccessToken)
		}
		return s.secret, nil
	}, jwt.WithIssuer(s.issuer))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return ports.TokenClaims{}, domain.ErrTokenExpired
		}
		return ports.TokenClaims{}, domain.ErrInvalidAccessToken
	}

	claims, ok := parsed.Claims.(*accessClaims)
	if !ok || !parsed.Valid {
		return ports.TokenClaims{}, domain.ErrInvalidAccessToken
	}

	userID, err := domain.ParseUserID(claims.UserID)
	if err != nil {
		return ports.TokenClaims{}, domain.ErrInvalidAccessToken
	}

	email, err := domain.ParseEmail(claims.Email)
	if err != nil {
		return ports.TokenClaims{}, domain.ErrInvalidAccessToken
	}

	roles := make([]domain.Role, 0, len(claims.Roles))
	for _, raw := range claims.Roles {
		role, err := domain.ParseRole(raw)
		if err != nil {
			return ports.TokenClaims{}, domain.ErrInvalidAccessToken
		}
		roles = append(roles, role)
	}

	permissions := make([]domain.Permission, 0, len(claims.Permissions))
	for _, raw := range claims.Permissions {
		perm, err := domain.ParsePermission(raw)
		if err != nil {
			return ports.TokenClaims{}, domain.ErrInvalidAccessToken
		}
		permissions = append(permissions, perm)
	}

	return ports.TokenClaims{
		UserID:      userID,
		Email:       email,
		Roles:       roles,
		Permissions: permissions,
	}, nil
}

var (
	_ ports.TokenIssuer    = (*Service)(nil)
	_ ports.TokenValidator = (*Service)(nil)
)

// ExtractBearerToken parses an Authorization header value.
func ExtractBearerToken(header string) (domain.AccessToken, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return domain.AccessToken{}, domain.ErrUnauthorized
	}
	return domain.ParseAccessToken(strings.TrimSpace(strings.TrimPrefix(header, prefix)))
}
