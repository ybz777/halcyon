DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;

DROP INDEX IF EXISTS idx_invitations_expires_at;
DROP INDEX IF EXISTS idx_invitations_token_hash;
DROP INDEX IF EXISTS idx_invitations_email;

ALTER TABLE invitations DROP COLUMN IF EXISTS revoked_at;

ALTER TABLE memberships DROP COLUMN IF EXISTS updated_at;
ALTER TABLE memberships DROP COLUMN IF EXISTS status;

DROP INDEX IF EXISTS idx_organizations_status;
ALTER TABLE organizations DROP COLUMN IF EXISTS status;
