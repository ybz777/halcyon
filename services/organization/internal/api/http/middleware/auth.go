package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	"github.com/halcyon-io/halcyon/pkg/platform/tenant"
	appauth "github.com/halcyon-io/halcyon/services/organization/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/organization/internal/application/organization"
	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
)

type userKey struct{}
type principalKey struct{}

// WithAuthenticatedUser stores the authenticated user in context.
func WithAuthenticatedUser(ctx context.Context, user ports.AuthenticatedUser) context.Context {
	return context.WithValue(ctx, userKey{}, user)
}

// AuthenticatedUserFromContext returns the authenticated user.
func AuthenticatedUserFromContext(ctx context.Context) (ports.AuthenticatedUser, bool) {
	user, ok := ctx.Value(userKey{}).(ports.AuthenticatedUser)
	return user, ok
}

// WithPrincipal stores the organization principal in context.
func WithPrincipal(ctx context.Context, principal appauth.Principal) context.Context {
	ctx = tenant.WithOrganizationID(ctx, principal.OrganizationID().String())
	return context.WithValue(ctx, principalKey{}, principal)
}

// PrincipalFromContext returns the organization principal.
func PrincipalFromContext(ctx context.Context) (appauth.Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(appauth.Principal)
	return principal, ok
}

// Authenticate validates JWT bearer tokens.
func Authenticate(validator ports.TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := validator.Validate(r.Context(), r.Header.Get("Authorization"))
			if err != nil {
				slog.Warn("authentication failed", "request_id", middleware.GetReqID(r.Context()), "error", err)
				httperrors.WriteProblem(w, httperrors.ProblemDetails{
					Title: "Unauthorized", Status: http.StatusUnauthorized, Detail: "authentication required",
				})
				return
			}
			next.ServeHTTP(w, r.WithContext(WithAuthenticatedUser(r.Context(), user)))
		})
	}
}

// OrganizationContext resolves membership and establishes tenant principal.
func OrganizationContext(resolver OrganizationResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := AuthenticatedUserFromContext(r.Context())
			if !ok {
				httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: "Unauthorized", Status: http.StatusUnauthorized})
				return
			}
			orgID, err := domain.ParseOrganizationID(chi.URLParam(r, "organization_id"))
			if err != nil {
				httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: "Bad Request", Status: http.StatusBadRequest, Detail: "invalid organization id"})
				return
			}
			principal, err := resolver.Resolve(r.Context(), user, orgID)
			if err != nil {
				slog.Warn("organization context denied", "organization_id", orgID.String(), "user_id", user.UserID.String())
				httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: "Forbidden", Status: http.StatusForbidden, Detail: "insufficient permissions"})
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
}

// OrganizationResolver resolves organization principals.
type OrganizationResolver interface {
	Resolve(ctx context.Context, user ports.AuthenticatedUser, organizationID domain.OrganizationID) (appauth.Principal, error)
}

// MembershipResolver resolves principals from membership.
type MembershipResolver struct {
	members ports.MembershipRepository
	orgs    ports.OrganizationRepository
}

// NewMembershipResolver creates a MembershipResolver.
func NewMembershipResolver(members ports.MembershipRepository, orgs ports.OrganizationRepository) *MembershipResolver {
	return &MembershipResolver{members: members, orgs: orgs}
}

// Resolve implements OrganizationResolver.
func (r *MembershipResolver) Resolve(ctx context.Context, user ports.AuthenticatedUser, organizationID domain.OrganizationID) (appauth.Principal, error) {
	return organization.ResolvePrincipal(ctx, r.members, r.orgs, user, organizationID)
}

// RequirePermission returns middleware that checks organization permissions.
func RequirePermission(authz *organization.Authorizer, permission domain.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: "Forbidden", Status: http.StatusForbidden})
				return
			}
			orgID, err := domain.ParseOrganizationID(chi.URLParam(r, "organization_id"))
			if err != nil {
				httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: "Bad Request", Status: http.StatusBadRequest})
				return
			}
			if err := authz.RequirePermission(principal, orgID, permission); err != nil {
				httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: "Forbidden", Status: http.StatusForbidden, Detail: "insufficient permissions"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
