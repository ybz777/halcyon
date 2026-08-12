INSERT INTO permissions (name) VALUES
    ('jobs.read'),
    ('jobs.cancel'),
    ('platform.jobs.read'),
    ('platform.jobs.cancel')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('owner', 'jobs.read'),
    ('owner', 'jobs.cancel'),
    ('owner', 'platform.jobs.read'),
    ('owner', 'platform.jobs.cancel'),
    ('admin', 'jobs.read'),
    ('admin', 'jobs.cancel')
ON CONFLICT (role_name, permission_name) DO NOTHING;
