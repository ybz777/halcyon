package ports

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// TokenClaims carries authenticated identity and authorization data.
type TokenClaims struct {
	UserID      domain.UserID
	Email       domain.Email
	Roles       []domain.Role
	Permissions []domain.Permission
}

// TokenIssuer signs access tokens.
type TokenIssuer interface {
	IssueAccessToken(ctx context.Context, claims TokenClaims) (domain.AccessToken, time.Time, error)
}

// TokenValidator validates access tokens.
type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, token domain.AccessToken) (TokenClaims, error)
}

// RefreshTokenClaims carries validated refresh JWT claims.
type RefreshTokenClaims struct {
	JTI       domain.JTI
	FamilyID  domain.FamilyID
	UserID    domain.UserID
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// RefreshTokenIssuer signs and validates refresh JWTs.
type RefreshTokenIssuer interface {
	IssueRefreshToken(ctx context.Context, familyID domain.FamilyID, parent *domain.JTI, userID domain.UserID) (domain.RefreshToken, RefreshTokenClaims, error)
	ValidateRefreshToken(ctx context.Context, token domain.RefreshToken) (RefreshTokenClaims, error)
}
