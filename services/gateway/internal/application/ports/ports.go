package ports

import (
	"context"
	"time"
)

// TokenClaims holds validated JWT identity.
type TokenClaims struct {
	UserID      string
	Email       string
	Roles       []string
	Permissions []string
}

// APIKeyAuthenticator validates API keys.
type APIKeyAuthenticator interface {
	Authenticate(ctx context.Context, apiKey string) (APIKeyPrincipal, error)
}

// APIKeyPrincipal is a validated API key identity.
type APIKeyPrincipal struct {
	APIKeyID       string
	OrganizationID string
	Scopes         []string
}

// TokenValidator validates bearer JWT access tokens.
type TokenValidator interface {
	Validate(ctx context.Context, token string) (TokenClaims, error)
}

// RateLimitDecision describes a rate limit check outcome.
type RateLimitDecision struct {
	Allowed    bool
	RetryAfter time.Duration
}

// RateLimiter enforces distributed request limits.
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (RateLimitDecision, error)
}

// ServiceTokenSigner signs internal service credentials.
type ServiceTokenSigner interface {
	Sign(serviceID string) (string, error)
}

// ReadinessChecker reports dependency readiness.
type ReadinessChecker interface {
	Check(ctx context.Context) map[string]string
}

// SecurityEventPublisher emits security events for audit logging.
type SecurityEventPublisher interface {
	PublishInvalidToken(ctx context.Context, requestID, correlationID, reason, ip, userAgent string) error
	PublishRateLimited(ctx context.Context, requestID, correlationID, routeClass, ip string) error
}

// ProxyMetrics records gateway proxy observability.
type ProxyMetrics interface {
	RecordRequest(method, route, service, status string)
	ObserveDuration(method, route, service string, duration time.Duration)
	IncInFlight(service string, delta int)
	RecordUpstreamError(service, reason string)
	RecordRateLimited(routeClass string)
	RecordAuthFailure(reason string)
	RecordCircuitOpen(service string)
	RecordJWKSRefresh(success bool)
	RecordBodyRejected()
}
