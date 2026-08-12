-- Supports cursor pagination: ORDER BY joined_at DESC, external_id DESC scoped by org+status.
CREATE INDEX IF NOT EXISTS idx_memberships_org_status_joined_id
    ON memberships (organization_external_id, status, joined_at DESC, external_id DESC);

-- Supports cursor pagination: ORDER BY created_at DESC, external_id DESC scoped by org.
CREATE INDEX IF NOT EXISTS idx_invitations_org_created_id
    ON invitations (organization_external_id, created_at DESC, external_id DESC);
