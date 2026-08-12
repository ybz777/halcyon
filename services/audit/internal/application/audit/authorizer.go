package audit

import (
	"strings"

	"github.com/halcyon-io/halcyon/services/audit/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/audit/internal/domain"
)

// Authorizer enforces audit API permissions.
type Authorizer struct{}

// NewAuthorizer creates an Authorizer.
func NewAuthorizer() *Authorizer {
	return &Authorizer{}
}

// ResolveOrganizationScope returns the organization ID for a query.
// Cross-tenant access requires audit.read.platform.
func (a *Authorizer) ResolveOrganizationScope(principal ports.Principal, trustedOrgID, requestedOrgID string) (string, error) {
	trustedOrgID = strings.TrimSpace(trustedOrgID)
	requestedOrgID = strings.TrimSpace(requestedOrgID)

	if !principal.HasPermission(ports.PermissionAuditRead) {
		return "", domain.ErrForbidden
	}

	if requestedOrgID != "" && requestedOrgID != trustedOrgID {
		if !principal.HasPermission(ports.PermissionAuditReadPlatform) {
			return "", domain.ErrForbidden
		}
		return requestedOrgID, nil
	}

	if trustedOrgID == "" {
		if principal.HasPermission(ports.PermissionAuditReadPlatform) && requestedOrgID != "" {
			return requestedOrgID, nil
		}
		return "", domain.ErrInvalidFilter
	}
	return trustedOrgID, nil
}

// CanViewEvent reports whether principal may read a specific audit record.
func (a *Authorizer) CanViewEvent(principal ports.Principal, organizationID string, event domain.AuditEvent) error {
	if event.IsSecurityEvent() && !principal.HasPermission(ports.PermissionAuditReadSecurity) {
		return domain.ErrForbidden
	}
	if event.OrganizationID != organizationID && !principal.HasPermission(ports.PermissionAuditReadPlatform) {
		return domain.ErrForbidden
	}
	if !principal.HasPermission(ports.PermissionAuditRead) {
		return domain.ErrForbidden
	}
	return nil
}

// FilterSecurityEvents removes security-sensitive events when caller lacks permission.
func (a *Authorizer) FilterSecurityEvents(principal ports.Principal, events []domain.AuditEvent) []domain.AuditEvent {
	if principal.HasPermission(ports.PermissionAuditReadSecurity) {
		return events
	}
	filtered := make([]domain.AuditEvent, 0, len(events))
	for _, event := range events {
		if !event.IsSecurityEvent() {
			filtered = append(filtered, event)
		}
	}
	return filtered
}
