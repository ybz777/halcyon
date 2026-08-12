package postgres

import (
    "context"
    "fmt"

    "github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
    "github.com/halcyon-io/halcyon/services/auth/internal/domain"
    "github.com/jackc/pgx/v5/pgxpool"
)

// UserRoleRepository loads and persists user role assignments from PostgreSQL.
type UserRoleRepository struct {
    pool *pgxpool.Pool
}

// NewUserRoleRepository creates a UserRoleRepository.
func NewUserRoleRepository(pool *pgxpool.Pool) *UserRoleRepository {
    return &UserRoleRepository{pool: pool}
}

// FindRolesByUserID returns roles assigned to a user.
func (r *UserRoleRepository) FindRolesByUserID(ctx context.Context, userID domain.UserID) ([]domain.Role, error) {
    rows, err := QuerierFromContext(ctx, r.pool).Query(ctx, `
        SELECT role, COALESCE(org_id, '')
        FROM user_roles
        WHERE user_external_id = $1
        ORDER BY role
    `, userID.String())
    if err != nil {
        return nil, fmt.Errorf("find roles: %w", err)
    }
    defer rows.Close()

    var roles []domain.Role
    for rows.Next() {
        var name, orgID string
        if err := rows.Scan(&name, &orgID); err != nil {
            return nil, fmt.Errorf("scan role: %w", err)
        }

        role, err := domain.NewRole(domain.RoleName(name), orgID)
        if err != nil {
            return nil, fmt.Errorf("parse role: %w", err)
        }
        roles = append(roles, role)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterate roles: %w", err)
    }

    return roles, nil
}

// AssignRole assigns a role to a user.
func (r *UserRoleRepository) AssignRole(ctx context.Context, userID domain.UserID, role domain.Role) error {
    _, err := QuerierFromContext(ctx, r.pool).Exec(ctx, `
        INSERT INTO user_roles (
            user_external_id,
            role,
            org_id
        )
        VALUES ($1, $2, NULLIF($3, ''))
        ON CONFLICT (user_external_id, role, org_id) DO NOTHING
    `,
        userID.String(),
        string(role.Name()),
        role.OrgID(),
    )
    if err != nil {
        return fmt.Errorf("assign role: %w", err)
    }

    return nil
}

var _ ports.UserRoleRepository = (*UserRoleRepository)(nil)
