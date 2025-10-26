INSERT INTO users_permissions
VALUES (
    (SELECT id FROM users WHERE email = 'john@example.com'),
    (SELECT id FROM permissions WHERE  code = 'quotes:write')
)
