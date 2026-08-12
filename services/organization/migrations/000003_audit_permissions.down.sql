DELETE FROM role_permissions WHERE permission_name IN ('audit.read.security', 'audit.read.platform');
DELETE FROM permissions WHERE name IN ('audit.read.security', 'audit.read.platform');
