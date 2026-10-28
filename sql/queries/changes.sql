-- name: ListChangeTypes :many
-- ListChangeTypes mengambil jenis perubahan beserta rute approval-nya (M1.8).
-- Rute kosong berarti rute default customer (ASUMSI E-35).
SELECT t.id, t.key, t.name, t.approval_route_id, r.name AS approval_route_name
FROM change_types t
LEFT JOIN approval_routes r ON r.id = t.approval_route_id
ORDER BY t.position;

-- name: GetChangeType :one
SELECT t.id, t.key, t.name, t.approval_route_id, r.name AS approval_route_name
FROM change_types t
LEFT JOIN approval_routes r ON r.id = t.approval_route_id
WHERE t.id = $1;

-- name: LockChangeType :exec
SELECT id FROM change_types WHERE id = $1 FOR NO KEY UPDATE;

-- name: RenameChangeType :exec
UPDATE change_types SET name = $2 WHERE id = $1;

-- name: ListChangeSources :many
-- ListChangeSources mengambil semua sumber perubahan, urut menurut nama.
SELECT id, name, mechanism, active, created_at FROM change_sources ORDER BY lower(name);

-- name: GetChangeSource :one
SELECT id, name, mechanism, active, created_at FROM change_sources WHERE id = $1;

-- name: LockChangeSource :exec
SELECT id FROM change_sources WHERE id = $1 FOR NO KEY UPDATE;

-- name: CreateChangeSource :one
INSERT INTO change_sources (name) VALUES ($1) RETURNING id;

-- name: UpdateChangeSource :exec
UPDATE change_sources SET name = $2, active = $3 WHERE id = $1;

-- name: ListFourMCategories :many
SELECT id, key, name FROM four_m_categories ORDER BY position;

-- name: GetFourMCategory :one
SELECT id, key, name FROM four_m_categories WHERE id = $1;

-- name: LockFourMCategory :exec
SELECT id FROM four_m_categories WHERE id = $1 FOR NO KEY UPDATE;

-- name: RenameFourMCategory :exec
UPDATE four_m_categories SET name = $2 WHERE id = $1;
