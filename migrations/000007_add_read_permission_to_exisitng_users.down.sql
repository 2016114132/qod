DELETE FROM users_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE code = 'quotes:read');
