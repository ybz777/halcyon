-- Organization status and soft-delete support.
ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'suspended', 'deleted'));

CREATE INDEX IF NOT EXISTS idx_organizations_status ON organizations (status);

-- Membership lifecycle.
ALTER TABLE memberships
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'suspended', 'removed')),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_memberships_org_status
    ON memberships (organization_external_id, status);

-- Rename operator role to member.
UPDATE memberships SET role = 'member' WHERE role = 'operator';
UPDATE invitations SET role = 'member' WHERE role = 'operator';

ALTER TABLE memberships DROP CONSTRAINT IF EXISTS memberships_role_check;
ALTER TABLE memberships ADD CONSTRAINT memberships_role_check
    CHECK (role IN ('owner', 'admin', 'member', 'viewer'));

ALTER TABLE invitations DROP CONSTRAINT IF EXISTS invitations_role_check;
ALTER TABLE invitations ADD CONSTRAINT invitations_role_check
    CHECK (role IN ('owner', 'admin', 'member', 'viewer'));

-- Invitation lifecycle hardening.
ALTER TABLE invitations
    ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ;

ALTER TABLE invitations DROP CONSTRAINT IF EXISTS invitations_status_check;
ALTER TABLE invitations ADD CONSTRAINT invitations_status_check
    CHECK (status IN ('pending', 'accepted', 'expired', 'revoked'));

CREATE INDEX IF NOT EXISTS idx_invitations_email ON invitations (email) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_invitations_token_hash ON invitations (token_hash);
CREATE INDEX IF NOT EXISTS idx_invitations_expires_at ON invitations (expires_at) WHERE status = 'pending';

-- RBAC reference tables (seeded deterministically).
CREATE TABLE IF NOT EXISTS roles (
    name TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS permissions (
    name TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_name TEXT NOT NULL REFERENCES roles (name) ON DELETE CASCADE,
    permission_name TEXT NOT NULL REFERENCES permissions (name) ON DELETE CASCADE,
    PRIMARY KEY (role_name, permission_name)
);

INSERT INTO permissions (name) VALUES
    ('organization.read'),
    ('organization.update'),
    ('organization.delete'),
    ('members.read'),
    ('members.invite'),
    ('members.update'),
    ('members.remove'),
    ('roles.read'),
    ('roles.manage'),
    ('users.read'),
    ('users.update'),
    ('incidents.read'),
    ('incidents.create'),
    ('incidents.update'),
    ('incidents.delete'),
    ('audit.read'),
    ('api_keys.read'),
    ('api_keys.create'),
    ('api_keys.revoke')
ON CONFLICT (name) DO NOTHING;

INSERT INTO roles (name, description) VALUES
    ('owner', 'Full organization access'),
    ('admin', 'Administrative access'),
    ('member', 'Standard member access'),
    ('viewer', 'Read-only access')
ON CONFLICT (name) DO NOTHING;

-- Owner: all permissions.
INSERT INTO role_permissions (role_name, permission_name)
SELECT 'owner', name FROM permissions
ON CONFLICT DO NOTHING;

-- Admin matrix.
INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'organization.read'),
    ('admin', 'organization.update'),
    ('admin', 'members.read'),
    ('admin', 'members.invite'),
    ('admin', 'members.update'),
    ('admin', 'members.remove'),
    ('admin', 'roles.read'),
    ('admin', 'roles.manage'),
    ('admin', 'users.read'),
    ('admin', 'users.update'),
    ('admin', 'incidents.read'),
    ('admin', 'incidents.create'),
    ('admin', 'incidents.update'),
    ('admin', 'incidents.delete'),
    ('admin', 'audit.read'),
    ('admin', 'api_keys.read'),
    ('admin', 'api_keys.create'),
    ('admin', 'api_keys.revoke')
ON CONFLICT DO NOTHING;

-- Member matrix.
INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('member', 'organization.read'),
    ('member', 'members.read'),
    ('member', 'users.read'),
    ('member', 'incidents.read'),
    ('member', 'incidents.create'),
    ('member', 'incidents.update')
ON CONFLICT DO NOTHING;

-- Viewer matrix.
INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('viewer', 'organization.read'),
    ('viewer', 'members.read'),
    ('viewer', 'users.read'),
    ('viewer', 'incidents.read')
ON CONFLICT DO NOTHING;
