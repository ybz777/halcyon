package ports

import (
"context"

"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// UserRoleRepository loads and persists role assignments for users.
type UserRoleRepository interface {
FindRolesByUserID(ctx context.Context, userID domain.UserID) ([]domain.Role, error)
AssignRole(ctx context.Context, userID domain.UserID, role domain.Role) error
}
