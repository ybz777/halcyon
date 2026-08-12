package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// Principal represents an authenticated caller.
type ValidateTokenInput struct {
	AccessToken string
}

// ValidateTokenUseCase validates JWT access tokens.
type ValidateTokenUseCase struct {
	validator ports.TokenValidator
	metrics   ports.AuthMetrics
}

// NewValidateTokenUseCase creates a ValidateTokenUseCase.
func NewValidateTokenUseCase(validator ports.TokenValidator, metrics ports.AuthMetrics) *ValidateTokenUseCase {
	return &ValidateTokenUseCase{validator: validator, metrics: metrics}
}

// Execute validates an access token and returns the authenticated principal.
func (uc *ValidateTokenUseCase) Execute(ctx context.Context, input ValidateTokenInput) (Principal, error) {
	start := time.Now()
	if uc.metrics != nil {
		uc.metrics.ObserveAuthLatency("validate", start)
	}

	token, err := domain.ParseAccessToken(input.AccessToken)
	if err != nil {
		uc.recordValidation(false)
		return Principal{}, domain.ErrInvalidAccessToken
	}

	claims, err := uc.validator.ValidateAccessToken(ctx, token)
	if err != nil {
		uc.recordValidation(false)
		switch {
		case errors.Is(err, domain.ErrTokenExpired):
			return Principal{}, domain.ErrTokenExpired
		case errors.Is(err, domain.ErrInvalidAccessToken):
			return Principal{}, domain.ErrInvalidAccessToken
		default:
			return Principal{}, fmt.Errorf("validate token: %w", err)
		}
	}

	uc.recordValidation(true)

	return Principal{
		AuthMethod:  AuthMethodUserJWT,
		UserID:      claims.UserID,
		Email:       claims.Email,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
	}, nil
}

func (uc *ValidateTokenUseCase) recordValidation(success bool) {
	if uc.metrics != nil {
		uc.metrics.RecordTokenValidation(success)
	}
}

// LogoutUseCase revokes a refresh token.
type LogoutUseCase struct {
	refreshTokens ports.RefreshTokenRepository
	refreshIssuer ports.RefreshTokenIssuer
	metrics       ports.AuthMetrics
}

// NewLogoutUseCase creates a LogoutUseCase.
func NewLogoutUseCase(
	refreshTokens ports.RefreshTokenRepository,
	refreshIssuer ports.RefreshTokenIssuer,
	metrics ports.AuthMetrics,
) *LogoutUseCase {
	return &LogoutUseCase{refreshTokens: refreshTokens, refreshIssuer: refreshIssuer, metrics: metrics}
}

// Execute revokes the provided refresh token.
func (uc *LogoutUseCase) Execute(ctx context.Context, tokenRaw string) error {
	start := time.Now()
	if uc.metrics != nil {
		uc.metrics.ObserveAuthLatency("logout", start)
	}

	token, err := domain.ParseRefreshToken(tokenRaw)
	if err != nil {
		return domain.ErrInvalidRefreshToken
	}

	claims, err := uc.refreshIssuer.ValidateRefreshToken(ctx, token)
	if err != nil {
		return domain.ErrInvalidRefreshToken
	}

	if err := uc.refreshTokens.RevokeByJTI(ctx, claims.JTI); err != nil {
		if errors.Is(err, domain.ErrInvalidRefreshToken) {
			return nil
		}
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}
