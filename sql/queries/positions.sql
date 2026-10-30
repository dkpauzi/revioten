-- name: ListPositions :many
-- ListPositions mengambil semua jabatan menurut urutan bagan organisasi.
SELECT id, key, name, per_project_group, single_holder FROM positions ORDER BY sort_order;

-- name: GetPosition :one
SELECT id, key, name, per_project_group, single_holder FROM positions WHERE id = $1;

-- name: LockPosition :exec
SELECT id FROM positions WHERE id = $1 FOR NO KEY UPDATE;

-- name: RenamePosition :exec
UPDATE positions SET name = $2 WHERE id = $1;

-- name: ListPositionHolders :many
-- ListPositionHolders mengambil pemegang semua jabatan beserta nama orang dan
-- kode Project Group-nya, urut menurut kode group lalu nama orang.
SELECT h.id, h.position_id, h.person_id, pe.name AS person_name,
       h.project_group_id, g.code AS project_group_code, h.created_at
FROM position_holders h
JOIN people pe ON pe.id = h.person_id
LEFT JOIN project_groups g ON g.id = h.project_group_id
ORDER BY g.code NULLS FIRST, pe.name;

-- name: GetPositionHolder :one
SELECT h.id, h.position_id, h.person_id, pe.name AS person_name,
       h.project_group_id, g.code AS project_group_code, h.created_at
FROM position_holders h
JOIN people pe ON pe.id = h.person_id
LEFT JOIN project_groups g ON g.id = h.project_group_id
WHERE h.id = $1;

-- name: AddPositionHolder :one
INSERT INTO position_holders (position_id, per_project_group, person_id, project_group_id)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: CountPositionHolders :one
SELECT count(*) FROM position_holders WHERE position_id = $1;

-- name: DeletePositionHolder :exec
DELETE FROM position_holders WHERE id = $1;
