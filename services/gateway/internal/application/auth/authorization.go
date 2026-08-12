package auth

import (
	"strings"

	"github.com/halcyon-io/halcyon/services/gateway/internal/domain"
)

// RequiredPermission returns the permission required to access a route, if any.
func RequiredPermission(_ string, path string) (domain.Permission, bool) {
	path = strings.TrimSuffix(path, "/")

	if strings.HasPrefix(path, "/api/v1/auth/api-keys") ||
		strings.HasPrefix(path, "/api/v1/api-keys") {
		return domain.PermissionAPIKeyManage, true
	}

	return "", false
}
