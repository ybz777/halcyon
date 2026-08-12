CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_external_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('user', 'admin')),
    org_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_roles_user_fk FOREIGN KEY (user_external_id) REFERENCES users (external_id),
    CONSTRAINT user_roles_unique UNIQUE (user_external_id, role, org_id)
);

CREATE INDEX idx_user_roles_user_external_id ON user_roles (user_external_id);
CREATE INDEX idx_user_roles_org_id ON user_roles (org_id) WHERE org_id IS NOT NULL;

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash TEXT NOT NULL UNIQUE,
    user_external_id TEXT NOT NULL,
    family_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    replaced_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT refresh_tokens_user_fk FOREIGN KEY (user_external_id) REFERENCES users (external_id)
);

CREATE INDEX idx_refresh_tokens_user_external_id ON refresh_tokens (user_external_id);
CREATE INDEX idx_refresh_tokens_family_id ON refresh_tokens (family_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens (expires_at) WHERE revoked_at IS NULL;
