INSERT INTO permissions (name) VALUES
    ('integrations.read'),
    ('integrations.write')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'integrations.read'),
    ('admin', 'integrations.write'),
    ('owner', 'integrations.read'),
    ('owner', 'integrations.write')
ON CONFLICT DO NOTHING;
