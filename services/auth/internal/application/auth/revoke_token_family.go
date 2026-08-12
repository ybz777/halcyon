package auth

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// RevokeTokenFamilyCommand revokes every token in a rotation family.
type RevokeTokenFamilyCommand struct {
	FamilyID domain.FamilyID
}

// RevokeTokenFamilyHandler revokes all refresh tokens in a family.
type RevokeTokenFamilyHandler struct {
	refreshTokens ports.RefreshTokenRepository
	metrics       ports.AuthMetrics
}

// NewRevokeTokenFamilyHandler creates a RevokeTokenFamilyHandler.
func NewRevokeTokenFamilyHandler(refreshTokens ports.RefreshTokenRepository, metrics ports.AuthMetrics) *RevokeTokenFamilyHandler {
	return &RevokeTokenFamilyHandler{refreshTokens: refreshTokens, metrics: metrics}
}

// Handle revokes the token family and records observability signals.
func (h *RevokeTokenFamilyHandler) Handle(ctx context.Context, cmd RevokeTokenFamilyCommand) error {
	if err := h.refreshTokens.RevokeFamily(ctx, cmd.FamilyID); err != nil {
		return fmt.Errorf("revoke token family: %w", err)
	}
	if h.metrics != nil {
		h.metrics.RecordFamilyRevoked()
	}
	slog.Warn("family_revoked", "family_id", cmd.FamilyID.String())
	return nil
}
