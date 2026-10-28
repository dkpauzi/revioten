-- name: ListGrants :many
-- ListGrants mengambil semua hak sebuah akun beserta cakupannya: dari peran
-- yang dipegang akun, ditambah hak peran Viewer yang dimiliki semua akun.
-- Akun yang nonaktif (atau tidak ada) tidak punya hak apa pun.
SELECT rp.permission_code, ar.scope
FROM account_roles ar
JOIN role_permissions rp ON rp.role_id = ar.role_id
JOIN accounts a ON a.id = ar.account_id
WHERE ar.account_id = $1 AND a.disabled_at IS NULL
UNION
SELECT rp.permission_code, 'all'
FROM role_permissions rp
JOIN accounts a ON a.id = $1
WHERE rp.role_id = 'viewer' AND a.disabled_at IS NULL;

-- name: ListRolesOfAccount :many
SELECT ar.role_id, ar.scope
FROM account_roles ar
JOIN roles r ON r.id = ar.role_id
WHERE ar.account_id = $1
ORDER BY r.sort_order;

-- name: ListRoles :many
SELECT id, level FROM roles ORDER BY sort_order;

-- name: ListRolePermissions :many
SELECT rp.role_id, rp.permission_code
FROM role_permissions rp
JOIN permissions p ON p.code = rp.permission_code
ORDER BY p.sort_order;

-- name: ListAccountRoles :many
SELECT ar.account_id, ar.role_id, ar.scope
FROM account_roles ar
JOIN roles r ON r.id = ar.role_id
ORDER BY r.sort_order;

-- name: DeleteAccountRoles :exec
DELETE FROM account_roles WHERE account_id = $1;

-- name: AddAccountRole :exec
INSERT INTO account_roles (account_id, role_id, scope) VALUES ($1, $2, $3);

