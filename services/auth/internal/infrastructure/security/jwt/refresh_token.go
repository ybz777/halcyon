package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

type refreshClaims struct {
	FamilyID string `json:"family_id"`
	jwt.RegisteredClaims
}

// IssueRefreshToken signs a refresh JWT with jti, family_id, sub, exp, and iat.
func (s *Service) IssueRefreshToken(
	_ context.Context,
	familyID domain.FamilyID,
	parent *domain.JTI,
	userID domain.UserID,
) (domain.RefreshToken, ports.RefreshTokenClaims, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.refreshTTL)
	jti, err := domain.ParseJTI(uuid.NewString())
	if err != nil {
		return domain.RefreshToken{}, ports.RefreshTokenClaims{}, err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims{
		FamilyID: familyID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti.String(),
			Issuer:    s.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	signed, err := token.SignedString(s.secret)
	if err != nil {
		return domain.RefreshToken{}, ports.RefreshTokenClaims{}, fmt.Errorf("sign refresh token: %w", err)
	}

	refreshToken, err := domain.ParseRefreshToken(signed)
	if err != nil {
		return domain.RefreshToken{}, ports.RefreshTokenClaims{}, err
	}

	_ = parent

	return refreshToken, ports.RefreshTokenClaims{
		JTI:       jti,
		FamilyID:  familyID,
		UserID:    userID,
		IssuedAt:  now,
		ExpiresAt: expiresAt,
	}, nil
}

// ValidateRefreshToken validates a refresh JWT and returns its claims.
func (s *Service) ValidateRefreshToken(_ context.Context, token domain.RefreshToken) (ports.RefreshTokenClaims, error) {
	parsed, err := jwt.ParseWithClaims(token.String(), &refreshClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("%w: unexpected signing method", domain.ErrInvalidRefreshToken)
		}
		return s.secret, nil
	}, jwt.WithIssuer(s.issuer))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return ports.RefreshTokenClaims{}, domain.ErrTokenExpired
		}
		return ports.RefreshTokenClaims{}, domain.ErrInvalidRefreshToken
	}

	claims, ok := parsed.Claims.(*refreshClaims)
	if !ok || !parsed.Valid {
		return ports.RefreshTokenClaims{}, domain.ErrInvalidRefreshToken
	}

	jti, err := domain.ParseJTI(claims.ID)
	if err != nil {
		return ports.RefreshTokenClaims{}, domain.ErrInvalidRefreshToken
	}

	familyID, err := domain.ParseFamilyID(claims.FamilyID)
	if err != nil {
		return ports.RefreshTokenClaims{}, domain.ErrInvalidRefreshToken
	}

	userID, err := domain.ParseUserID(claims.Subject)
	if err != nil {
		return ports.RefreshTokenClaims{}, domain.ErrInvalidRefreshToken
	}

	var issuedAt, expiresAt time.Time
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Time.UTC()
	}
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time.UTC()
	}

	return ports.RefreshTokenClaims{
		JTI:       jti,
		FamilyID:  familyID,
		UserID:    userID,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}, nil
}

var _ ports.RefreshTokenIssuer = (*Service)(nil)
