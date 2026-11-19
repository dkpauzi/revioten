-- name: ListGrants :many
-- ListGrants mengambil semua hak sebuah akun beserta cakupannya: dari peran
-- yang dipegang akun, ditambah hak peran Viewer yang dimiliki semua akun
-- kerja (akun admin memegang hak dasar lewat peran Admin, D-199).
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
WHERE rp.role_id = 'viewer' AND a.disabled_at IS NULL AND a.kind = 'work';

-- name: ListRolesOfAccount :many
SELECT ar.role_id, ar.scope
FROM account_roles ar
JOIN roles r ON r.id = ar.role_id
WHERE ar.account_id = $1
ORDER BY r.sort_order;

-- name: ListRoles :many
-- ListRoles memberi katalog peran beserta jumlah akun pemegangnya (D-199).
SELECT r.id, r.level, r.name_en, r.name_id, r.active, r.builtin, r.group_scope,
       (SELECT count(*) FROM account_roles ar WHERE ar.role_id = r.id)::integer AS holders
FROM roles r
ORDER BY r.sort_order;

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


-- name: InOwnProjectGroup :one
-- InOwnProjectGroup melaporkan apakah orang itu Project Leader atau Project
-- Engineer di Project Group tempat customer dialokasikan (D-71, E-05).
SELECT EXISTS (
    SELECT 1
    FROM position_holders h
    JOIN positions p ON p.id = h.position_id
    JOIN customers c ON c.project_group_id = h.project_group_id
    WHERE h.person_id = sqlc.arg(person_id) AND c.id = sqlc.arg(customer_id)
      AND p.key IN ('project_leader', 'project_engineer')
);

-- M8.1: isi peran diatur admin (D-66).

-- name: ListPermissions :many
SELECT code, level FROM permissions ORDER BY sort_order;

-- name: LockRole :one
-- LockRole memberi level satu peran dan memegang kunci penasihat (advisory
-- lock) atas peran itu sampai transaksi selesai, supaya dua admin bergantian.
-- Role aplikasi tidak punya UPDATE pada roles, jadi FOR UPDATE tidak dipakai.
SELECT r.level, r.active, r.builtin, r.group_scope
FROM roles r, pg_advisory_xact_lock(hashtext('role:' || sqlc.arg(id)::text))
WHERE r.id = sqlc.arg(id);

-- name: ListPermissionsOfRole :many
SELECT rp.permission_code
FROM role_permissions rp JOIN permissions p ON p.code = rp.permission_code
WHERE rp.role_id = $1 ORDER BY p.sort_order;

-- name: AddRolePermission :exec
INSERT INTO role_permissions (role_id, permission_code) VALUES ($1, $2);

-- name: RemoveRolePermission :exec
DELETE FROM role_permissions WHERE role_id = $1 AND permission_code = $2;

-- D-199: admin menambah, mengganti nama, dan menonaktifkan peran.

-- name: LockRoleCatalog :exec
-- LockRoleCatalog memegang kunci penasihat atas katalog peran sampai
-- transaksi selesai: dua admin yang menambah peran bersamaan bergantian,
-- supaya urutan dan pemeriksaan nama tidak bertabrakan.
SELECT pg_advisory_xact_lock(hashtext('role-catalog'));

-- name: RoleNameTaken :one
-- RoleNameTaken melaporkan apakah nama Inggris atau Indonesia sudah dipakai
-- peran lain (huruf besar dan kecil dianggap sama).
SELECT EXISTS (
    SELECT 1 FROM roles
    WHERE id <> sqlc.arg(id)::text
      AND (lower(name_en) = lower(sqlc.arg(name_en)::text) OR lower(name_id) = lower(sqlc.arg(name_id)::text))
);

-- name: RoleIDTaken :one
SELECT EXISTS (SELECT 1 FROM roles WHERE id = $1);

-- name: CreateRole :exec
-- Peran baru ditaruh sesudah peran aktif terakhir dan sebelum peran lama.
INSERT INTO roles (id, level, sort_order, name_en, name_id, builtin, group_scope)
SELECT sqlc.arg(id), 'edit',
       coalesce((SELECT max(sort_order) FROM roles WHERE sort_order < 100), 0) + 1,
       sqlc.arg(name_en), sqlc.arg(name_id), false, sqlc.arg(group_scope);

-- name: UpdateRole :exec
UPDATE roles SET name_en = $2, name_id = $3, group_scope = $4 WHERE id = $1;

-- name: SetRoleActive :exec
UPDATE roles SET active = $2 WHERE id = $1;

-- name: CountRoleHolders :one
SELECT count(*)::integer FROM account_roles WHERE role_id = $1;

-- name: GetRole :one
SELECT r.id, r.level, r.name_en, r.name_id, r.active, r.builtin, r.group_scope,
       (SELECT count(*) FROM account_roles ar WHERE ar.role_id = r.id)::integer AS holders
FROM roles r
WHERE r.id = $1;

-- name: CountScopedRoleHolders :one
SELECT count(*)::integer FROM account_roles WHERE role_id = $1 AND scope = 'own_project_group';
