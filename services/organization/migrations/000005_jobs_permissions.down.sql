DELETE FROM role_permissions WHERE permission_name IN ('jobs.read', 'jobs.cancel', 'platform.jobs.read', 'platform.jobs.cancel');
DELETE FROM permissions WHERE name IN ('jobs.read', 'jobs.cancel', 'platform.jobs.read', 'platform.jobs.cancel');
