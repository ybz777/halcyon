package ratelimit

import (
	"context"

	platformratelimit "github.com/halcyon-io/halcyon/pkg/platform/ratelimit"
)

// LoginAdapter gates login attempts using the platform limiter.
type LoginAdapter struct {
	limiter platformratelimit.Limiter
	policy  platformratelimit.Policy
}

// NewLoginAdapter creates a login rate limiter.
func NewLoginAdapter(limiter platformratelimit.Limiter, policy platformratelimit.Policy) *LoginAdapter {
	return &LoginAdapter{limiter: limiter, policy: policy}
}

// Allow implements ports.LoginRateLimiter.
func (a *LoginAdapter) Allow(ctx context.Context, email string) (bool, error) {
	if a == nil || a.limiter == nil {
		return true, nil
	}
	decision, err := a.limiter.Allow(ctx, platformratelimit.IdentityKey(platformratelimit.IdentityUser, email), a.policy)
	return decision.Allowed, err
}

// APIKeyAdapter gates API key authentication failures.
type APIKeyAdapter struct {
	limiter platformratelimit.Limiter
	policy  platformratelimit.Policy
}

// NewAPIKeyAdapter creates an API key auth rate limiter.
func NewAPIKeyAdapter(limiter platformratelimit.Limiter, policy platformratelimit.Policy) *APIKeyAdapter {
	return &APIKeyAdapter{limiter: limiter, policy: policy}
}

// Allow implements ports.APIKeyRateLimiter.
func (a *APIKeyAdapter) Allow(ctx context.Context, prefix string) (bool, error) {
	if a == nil || a.limiter == nil {
		return true, nil
	}
	decision, err := a.limiter.Allow(ctx, platformratelimit.IdentityKey(platformratelimit.IdentityAPIKey, prefix), a.policy)
	return decision.Allowed, err
}
