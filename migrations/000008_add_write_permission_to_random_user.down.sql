DELETE FROM users_permissions
WHERE user_id = (SELECT id FROM users WHERE email = 'john@example.com')
AND permission_id = (SELECT id FROM permissions WHERE code = 'quotes:write');
