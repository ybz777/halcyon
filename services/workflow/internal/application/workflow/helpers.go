package workflow

import (
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

func ensureRead(principal ports.Principal) error {
	if !principal.HasPermission(ports.PermissionWorkflowRead) {
		return domain.ErrForbidden
	}
	return nil
}

func ensureWrite(principal ports.Principal) error {
	if !principal.HasPermission(ports.PermissionWorkflowWrite) {
		return domain.ErrForbidden
	}
	return nil
}

func ensureExecute(principal ports.Principal) error {
	if !principal.HasPermission(ports.PermissionWorkflowExecute) {
		return domain.ErrForbidden
	}
	return nil
}

func ensureCancel(principal ports.Principal) error {
	if !principal.HasPermission(ports.PermissionWorkflowCancel) {
		return domain.ErrForbidden
	}
	return nil
}
