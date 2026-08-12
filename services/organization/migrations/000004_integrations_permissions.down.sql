DELETE FROM role_permissions WHERE permission_name IN ('integrations.read', 'integrations.write');
DELETE FROM permissions WHERE name IN ('integrations.read', 'integrations.write');
