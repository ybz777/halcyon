package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	"github.com/halcyon-io/halcyon/pkg/platform/servicetoken"
)

// GatewayTrust validates internal service tokens from the API gateway.
type GatewayTrust struct {
	validator *servicetoken.Validator
	metrics   Metrics
}

// NewGatewayTrust creates a gateway trust validator.
func NewGatewayTrust(validator *servicetoken.Validator, metrics Metrics) *GatewayTrust {
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	return &GatewayTrust{validator: validator, metrics: metrics}
}

// HasForwardedIdentity reports whether the request carries gateway-forwarded identity headers.
func HasForwardedIdentity(r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Halcyon-API-Key-ID")) != "" {
		return true
	}
	if strings.TrimSpace(r.Header.Get("X-Halcyon-User-ID")) != "" {
		return true
	}
	if strings.TrimSpace(r.Header.Get("X-Halcyon-Organization-ID")) != "" {
		return true
	}
	return false
}

// ValidateRequest verifies the gateway service token when forwarded identity headers are present.
func (g *GatewayTrust) ValidateRequest(r *http.Request) (string, error) {
	if g == nil || g.validator == nil {
		if HasForwardedIdentity(r) {
			return "", &AuthError{Category: FailureUntrustedGateway, Err: ErrUnauthorized}
		}
		return "", nil
	}
	if !HasForwardedIdentity(r) {
		return "", nil
	}
	token := strings.TrimSpace(r.Header.Get(servicetoken.HeaderName))
	if token == "" {
		g.metrics.RecordAuthFailure(r.Method, routePattern(r), string(MethodServiceToken), string(FailureUntrustedGateway))
		return "", &AuthError{Category: FailureUntrustedGateway, Err: ErrUnauthorized}
	}
	claims, err := g.validator.Validate(token)
	if err != nil {
		g.metrics.RecordAuthFailure(r.Method, routePattern(r), string(MethodServiceToken), string(FailureInvalidToken))
		return "", &AuthError{Category: FailureInvalidToken, Err: ErrUnauthorized}
	}
	g.metrics.RecordAuthSuccess(r.Method, routePattern(r), string(MethodServiceToken))
	return claims.ServiceID, nil
}

// Middleware rejects spoofed gateway-forwarded identity headers.
func (g *GatewayTrust) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serviceID, err := g.ValidateRequest(r)
			if err != nil {
				WriteUnauthorized(w, "authentication required")
				return
			}
			ctx := r.Context()
			if serviceID != "" {
				ctx = WithServiceCaller(ctx, serviceID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireGatewayTrust is a package-level helper when metrics are not needed.
func RequireGatewayTrust(validator *servicetoken.Validator) func(http.Handler) http.Handler {
	return NewGatewayTrust(validator, nil).Middleware()
}

// WriteUnauthorized writes a stable 401 problem response.
func WriteUnauthorized(w http.ResponseWriter, detail string) {
	if detail == "" {
		detail = "authentication required"
	}
	httperrors.WriteProblem(w, httperrors.ProblemDetails{
		Title:  "Unauthorized",
		Status: http.StatusUnauthorized,
		Detail: detail,
	})
}

// WriteForbidden writes a stable 403 problem response.
func WriteForbidden(w http.ResponseWriter, detail string) {
	if detail == "" {
		detail = "insufficient permissions"
	}
	httperrors.WriteProblem(w, httperrors.ProblemDetails{
		Title:  "Forbidden",
		Status: http.StatusForbidden,
		Detail: detail,
	})
}

func routePattern(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	if pattern := r.Pattern; pattern != "" {
		return pattern
	}
	return r.URL.Path
}

// ServiceCallerFromRequest returns the validated gateway service ID attached to the request.
func ServiceCallerFromRequest(ctx context.Context) (string, bool) {
	return ServiceCallerFromContext(ctx)
}
