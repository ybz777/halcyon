package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// RotateRefreshTokenCommand carries a refresh token for rotation.
type RotateRefreshTokenCommand struct {
	RefreshToken string
	IPAddress    string
	UserAgent    string
}

// RotateRefreshTokenHandler rotates refresh tokens and issues a new access token.
type RotateRefreshTokenHandler struct {
	users         ports.UserRepository
	roles         ports.UserRoleRepository
	refreshTokens ports.RefreshTokenRepository
	issuer        ports.TokenIssuer
	refreshIssuer ports.RefreshTokenIssuer
	revokeFamily  *RevokeTokenFamilyHandler
	auth          domain.AuthenticationService
	metrics       ports.AuthMetrics
}

// NewRotateRefreshTokenHandler creates a RotateRefreshTokenHandler.
func NewRotateRefreshTokenHandler(
	users ports.UserRepository,
	roles ports.UserRoleRepository,
	refreshTokens ports.RefreshTokenRepository,
	issuer ports.TokenIssuer,
	refreshIssuer ports.RefreshTokenIssuer,
	revokeFamily *RevokeTokenFamilyHandler,
	metrics ports.AuthMetrics,
) *RotateRefreshTokenHandler {
	return &RotateRefreshTokenHandler{
		users:         users,
		roles:         roles,
		refreshTokens: refreshTokens,
		issuer:        issuer,
		refreshIssuer: refreshIssuer,
		revokeFamily:  revokeFamily,
		auth:          domain.NewAuthenticationService(),
		metrics:       metrics,
	}
}

// Handle validates a refresh token and returns a rotated token pair.
func (h *RotateRefreshTokenHandler) Handle(ctx context.Context, cmd RotateRefreshTokenCommand) (TokenResponse, error) {
	token, err := domain.ParseRefreshToken(cmd.RefreshToken)
	if err != nil {
		h.recordFailure()
		return TokenResponse{}, domain.ErrInvalidRefreshToken
	}

	claims, err := h.refreshIssuer.ValidateRefreshToken(ctx, token)
	if err != nil {
		h.recordFailure()
		switch {
		case errors.Is(err, domain.ErrTokenExpired):
			return TokenResponse{}, domain.ErrTokenExpired
		default:
			return TokenResponse{}, domain.ErrInvalidRefreshToken
		}
	}

	record, err := h.refreshTokens.FindByJTI(ctx, claims.JTI)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRefreshToken) {
			h.recordFailure()
			return TokenResponse{}, domain.ErrInvalidRefreshToken
		}
		return TokenResponse{}, fmt.Errorf("find refresh token: %w", err)
	}

	if !domain.SecureCompareHashes(record.TokenHash(), domain.HashRefreshToken(token)) {
		h.recordFailure()
		return TokenResponse{}, domain.ErrInvalidRefreshToken
	}

	if err := record.ValidateForRefresh(time.Now().UTC()); err != nil {
		if errors.Is(err, domain.ErrRefreshTokenReused) {
			return h.handleReplay(ctx, record.FamilyID())
		}
		h.recordFailure()
		return TokenResponse{}, err
	}

	if !record.FamilyID().Equal(claims.FamilyID) || !record.UserID().Equal(claims.UserID) {
		h.recordFailure()
		return TokenResponse{}, domain.ErrInvalidRefreshToken
	}

	user, err := h.users.FindByID(ctx, record.UserID())
	if err != nil {
		h.recordFailure()
		return TokenResponse{}, domain.ErrInvalidRefreshToken
	}

	if err := h.auth.EnsureUserCanAuthenticate(user); err != nil {
		h.recordFailure()
		return TokenResponse{}, err
	}

	roles, err := h.loadRoles(ctx, user.ID())
	if err != nil {
		return TokenResponse{}, err
	}

	accessClaims := ports.TokenClaims{
		UserID:      user.ID(),
		Email:       user.Email(),
		Roles:       roles,
		Permissions: h.auth.ResolvePermissions(roles),
	}

	accessToken, expiresAt, err := h.issuer.IssueAccessToken(ctx, accessClaims)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("issue access token: %w", err)
	}

	parent := record.TokenID()
	newRefresh, newClaims, err := h.refreshIssuer.IssueRefreshToken(ctx, record.FamilyID(), &parent, user.ID())
	if err != nil {
		return TokenResponse{}, fmt.Errorf("issue refresh token: %w", err)
	}

	newRecord, err := domain.NewRefreshTokenRecord(
		newClaims.JTI,
		newClaims.FamilyID,
		&parent,
		user.ID(),
		domain.HashRefreshToken(newRefresh),
		newClaims.IssuedAt,
		newClaims.ExpiresAt,
		cmd.IPAddress,
		cmd.UserAgent,
	)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("build refresh record: %w", err)
	}

	err = h.refreshTokens.WithinTransaction(ctx, func(txCtx context.Context) error {
		consumed, consumeErr := h.refreshTokens.Consume(txCtx, claims.JTI)
		if consumeErr != nil {
			if errors.Is(consumeErr, domain.ErrRefreshTokenReused) {
				return consumeErr
			}
			return consumeErr
		}
		if !consumed.TokenID().Equal(claims.JTI) {
			return domain.ErrInvalidRefreshToken
		}
		return h.refreshTokens.Save(txCtx, newRecord)
	})
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenReused) {
			return h.handleReplay(ctx, record.FamilyID())
		}
		h.recordFailure()
		return TokenResponse{}, fmt.Errorf("rotate refresh token: %w", err)
	}

	h.recordSuccess()
	slog.Info("refresh_rotated",
		"user_id", user.ID().String(),
		"family_id", record.FamilyID().String(),
		"old_jti", record.TokenID().String(),
		"new_jti", newClaims.JTI.String(),
	)

	return TokenResponse{
		AccessToken:  accessToken.String(),
		RefreshToken: newRefresh.String(),
		TokenType:    "Bearer",
		ExpiresIn:    int64(time.Until(expiresAt).Seconds()),
	}, nil
}

func (h *RotateRefreshTokenHandler) handleReplay(ctx context.Context, familyID domain.FamilyID) (TokenResponse, error) {
	_ = h.revokeFamily.Handle(ctx, RevokeTokenFamilyCommand{FamilyID: familyID})
	if h.metrics != nil {
		h.metrics.RecordRefreshReplay()
	}
	slog.Warn("refresh_replay_detected", "family_id", familyID.String())
	h.recordFailure()
	return TokenResponse{}, domain.ErrRefreshTokenReused
}

func (h *RotateRefreshTokenHandler) loadRoles(ctx context.Context, userID domain.UserID) ([]domain.Role, error) {
	roles, err := h.roles.FindRolesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find roles: %w", err)
	}
	if len(roles) == 0 {
		return h.auth.DefaultRoles(), nil
	}
	return roles, nil
}

func (h *RotateRefreshTokenHandler) recordSuccess() {
	if h.metrics == nil {
		return
	}
	h.metrics.RecordTokenRefresh(true)
	h.metrics.RecordRefreshSuccess()
	h.metrics.RecordRefreshRotation()
}

func (h *RotateRefreshTokenHandler) recordFailure() {
	if h.metrics != nil {
		h.metrics.RecordTokenRefresh(false)
	}
}
