CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id TEXT NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT organizations_external_id_unique UNIQUE (external_id),
    CONSTRAINT organizations_slug_unique UNIQUE (slug)
);

CREATE INDEX idx_organizations_slug ON organizations (slug);

CREATE TABLE memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id TEXT NOT NULL,
    organization_external_id TEXT NOT NULL,
    user_external_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'operator', 'viewer')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT memberships_external_id_unique UNIQUE (external_id),
    CONSTRAINT memberships_org_user_unique UNIQUE (organization_external_id, user_external_id),
    CONSTRAINT memberships_organization_fk FOREIGN KEY (organization_external_id)
        REFERENCES organizations (external_id)
);

CREATE INDEX idx_memberships_organization ON memberships (organization_external_id);
CREATE INDEX idx_memberships_user ON memberships (user_external_id);
CREATE INDEX idx_memberships_org_role ON memberships (organization_external_id, role);

CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id TEXT NOT NULL,
    organization_external_id TEXT NOT NULL,
    user_external_id TEXT,
    email TEXT,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'operator', 'viewer')),
    token_hash TEXT NOT NULL,
    invited_by_user_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'accepted', 'revoked')),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT invitations_external_id_unique UNIQUE (external_id),
    CONSTRAINT invitations_token_hash_unique UNIQUE (token_hash),
    CONSTRAINT invitations_organization_fk FOREIGN KEY (organization_external_id)
        REFERENCES organizations (external_id)
);

CREATE INDEX idx_invitations_organization ON invitations (organization_external_id);
CREATE INDEX idx_invitations_status ON invitations (status) WHERE status = 'pending';
