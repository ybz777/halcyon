package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	infraevents "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/events"
)

// LoginInput carries login credentials.
type LoginInput struct {
	Email    string
	Password string
}

// TokenResponse is returned after successful authentication.
type TokenResponse struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int64
}

// LoginUseCase authenticates a user and issues tokens.
type LoginUseCase struct {
	users         ports.UserRepository
	roles         ports.UserRoleRepository
	refreshTokens ports.RefreshTokenRepository
	hasher        ports.PasswordHasher
	issuer        ports.TokenIssuer
	refreshIssuer ports.RefreshTokenIssuer
	auth          domain.AuthenticationService
	rateLimiter   ports.LoginRateLimiter
	metrics       ports.AuthMetrics
	tx            ports.TransactionManager
	events        ports.EventPublisher
}

// NewLoginUseCase creates a LoginUseCase.
func NewLoginUseCase(
	users ports.UserRepository,
	roles ports.UserRoleRepository,
	refreshTokens ports.RefreshTokenRepository,
	hasher ports.PasswordHasher,
	issuer ports.TokenIssuer,
	refreshIssuer ports.RefreshTokenIssuer,
	rateLimiter ports.LoginRateLimiter,
	metrics ports.AuthMetrics,
	tx ports.TransactionManager,
	events ports.EventPublisher,
) *LoginUseCase {
	if rateLimiter == nil {
		rateLimiter = ports.NoopLoginRateLimiter{}
	}
	if events == nil {
		events = ports.NoopEventPublisher{}
	}
	if tx == nil {
		tx = ports.NoopTransactionManager{}
	}
	return &LoginUseCase{
		users:         users,
		roles:         roles,
		refreshTokens: refreshTokens,
		hasher:        hasher,
		issuer:        issuer,
		refreshIssuer: refreshIssuer,
		auth:          domain.NewAuthenticationService(),
		rateLimiter:   rateLimiter,
		metrics:       metrics,
		tx:            tx,
		events:        events,
	}
}

// Execute authenticates credentials and returns token pair.
func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (TokenResponse, error) {
	start := time.Now()
	if uc.metrics != nil {
		uc.metrics.ObserveAuthLatency("login", start)
		uc.metrics.RecordLoginAttempt()
	}

	email, err := domain.ParseEmail(input.Email)
	if err != nil {
		uc.recordFailure("invalid_email")
		return TokenResponse{}, domain.ErrInvalidCredentials
	}

	allowed, err := uc.rateLimiter.Allow(ctx, email.String())
	if err != nil {
		return TokenResponse{}, fmt.Errorf("rate limit check: %w", err)
	}
	if !allowed {
		uc.recordFailure("rate_limited")
		return TokenResponse{}, domain.ErrForbidden
	}

	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			uc.verifyDummyPassword(ctx, input.Password)
			uc.recordFailure("invalid_credentials")
			return TokenResponse{}, domain.ErrInvalidCredentials
		}
		return TokenResponse{}, fmt.Errorf("find user: %w", err)
	}

	if err := uc.auth.EnsureUserCanAuthenticate(user); err != nil {
		uc.recordFailure("user_disabled")
		return TokenResponse{}, err
	}

	ok, err := uc.hasher.Verify(ctx, input.Password, user.PasswordHash())
	if err != nil {
		return TokenResponse{}, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		uc.recordFailure("invalid_credentials")
		return TokenResponse{}, domain.ErrInvalidCredentials
	}

	roles, err := uc.loadRoles(ctx, user.ID())
	if err != nil {
		return TokenResponse{}, err
	}

	claims := ports.TokenClaims{
		UserID:      user.ID(),
		Email:       user.Email(),
		Roles:       roles,
		Permissions: uc.auth.ResolvePermissions(roles),
	}

	accessToken, expiresAt, err := uc.issuer.IssueAccessToken(ctx, claims)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("issue access token: %w", err)
	}

	familyID, err := domain.NewFamilyID()
	if err != nil {
		return TokenResponse{}, fmt.Errorf("generate family id: %w", err)
	}

	refreshToken, refreshClaims, err := uc.refreshIssuer.IssueRefreshToken(ctx, familyID, nil, user.ID())
	if err != nil {
		return TokenResponse{}, fmt.Errorf("issue refresh token: %w", err)
	}

	record, err := domain.NewRefreshTokenRecord(
		refreshClaims.JTI,
		refreshClaims.FamilyID,
		nil,
		user.ID(),
		domain.HashRefreshToken(refreshToken),
		refreshClaims.IssuedAt,
		refreshClaims.ExpiresAt,
		"",
		"",
	)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("build refresh record: %w", err)
	}

	if err := uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.refreshTokens.Save(txCtx, record); err != nil {
			return fmt.Errorf("store refresh token: %w", err)
		}
		loggedIn := domain.NewUserLoggedIn(user.ID(), user.Email(), time.Now().UTC())
		return infraevents.PublishDomainEvent(txCtx, uc.events, loggedIn, "")
	}); err != nil {
		return TokenResponse{}, err
	}

	return TokenResponse{
		AccessToken:  accessToken.String(),
		RefreshToken: refreshToken.String(),
		TokenType:    "Bearer",
		ExpiresIn:    int64(time.Until(expiresAt).Seconds()),
	}, nil
}

func (uc *LoginUseCase) loadRoles(ctx context.Context, userID domain.UserID) ([]domain.Role, error) {
	roles, err := uc.roles.FindRolesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find roles: %w", err)
	}
	if len(roles) == 0 {
		return uc.auth.DefaultRoles(), nil
	}
	return roles, nil
}

func (uc *LoginUseCase) verifyDummyPassword(ctx context.Context, password string) {
	dummy, _ := domain.ParsePasswordHash("$argon2id$v=19$m=65536,t=1,p=4$c2FsdHNhbHQ$hashvaluehashvaluehashvaluehashvaluehashvaluehashval")
	_, _ = uc.hasher.Verify(ctx, password, dummy)
}

func (uc *LoginUseCase) recordFailure(reason string) {
	if uc.metrics != nil {
		uc.metrics.RecordLoginFailure(reason)
	}
	if uc.events != nil {
		failed := domain.NewUserLoginFailed(domain.UserID{}, reason, time.Now().UTC())
		_ = infraevents.PublishDomainEvent(context.Background(), uc.events, failed, "")
	}
}

func (uc *LoginUseCase) observe(operation string, start time.Time) {
	if uc.metrics != nil {
		uc.metrics.ObserveAuthLatency(operation, start)
	}
}
