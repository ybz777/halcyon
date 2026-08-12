package auth

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
)

// RefreshTokenInput carries a refresh token for rotation.
type RefreshTokenInput struct {
	RefreshToken string
	IPAddress    string
	UserAgent    string
}

// RefreshTokenUseCase rotates refresh tokens and issues a new access token.
type RefreshTokenUseCase struct {
	handler *RotateRefreshTokenHandler
	metrics ports.AuthMetrics
}

// NewRefreshTokenUseCase creates a RefreshTokenUseCase.
func NewRefreshTokenUseCase(
	users ports.UserRepository,
	roles ports.UserRoleRepository,
	refreshTokens ports.RefreshTokenRepository,
	issuer ports.TokenIssuer,
	refreshIssuer ports.RefreshTokenIssuer,
	metrics ports.AuthMetrics,
) *RefreshTokenUseCase {
	revokeFamily := NewRevokeTokenFamilyHandler(refreshTokens, metrics)
	handler := NewRotateRefreshTokenHandler(
		users,
		roles,
		refreshTokens,
		issuer,
		refreshIssuer,
		revokeFamily,
		metrics,
	)
	return &RefreshTokenUseCase{handler: handler, metrics: metrics}
}

// Execute validates a refresh token and returns a rotated token pair.
func (uc *RefreshTokenUseCase) Execute(ctx context.Context, input RefreshTokenInput) (TokenResponse, error) {
	start := time.Now()
	if uc.metrics != nil {
		uc.metrics.ObserveAuthLatency("refresh", start)
	}

	return uc.handler.Handle(ctx, RotateRefreshTokenCommand{
		RefreshToken: input.RefreshToken,
		IPAddress:    input.IPAddress,
		UserAgent:    input.UserAgent,
	})
}
