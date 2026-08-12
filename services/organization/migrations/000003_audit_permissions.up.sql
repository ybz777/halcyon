INSERT INTO permissions (name) VALUES
    ('audit.read.security'),
    ('audit.read.platform')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'audit.read.security'),
    ('owner', 'audit.read.security'),
    ('owner', 'audit.read.platform')
ON CONFLICT DO NOTHING;
