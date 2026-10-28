-- name: ListModelGroups :many
-- ListModelGroups mengambil group model satu customer, urut menurut nama.
SELECT id, customer_id, name, created_at FROM model_groups WHERE customer_id = $1 ORDER BY name;

-- name: GetModelGroup :one
SELECT id, customer_id, name, created_at FROM model_groups WHERE id = $1;

-- name: LockModelGroup :exec
SELECT id FROM model_groups WHERE id = $1 FOR NO KEY UPDATE;

-- name: CreateModelGroup :one
INSERT INTO model_groups (customer_id, name) VALUES ($1, $2)
RETURNING id, customer_id, name, created_at;

-- name: RenameModelGroup :exec
UPDATE model_groups SET name = $2 WHERE id = $1;

-- name: ListModels :many
-- ListModels mengambil model satu customer beserta nama group-nya, urut menurut nama.
SELECT m.id, m.customer_id, m.model_group_id, g.name AS model_group_name, m.name, m.created_at
FROM models m
LEFT JOIN model_groups g ON g.id = m.model_group_id
WHERE m.customer_id = $1
ORDER BY m.name;

-- name: GetModel :one
SELECT m.id, m.customer_id, m.model_group_id, g.name AS model_group_name, m.name, m.created_at
FROM models m
LEFT JOIN model_groups g ON g.id = m.model_group_id
WHERE m.id = $1;

-- name: LockModel :exec
SELECT id FROM models WHERE id = $1 FOR NO KEY UPDATE;

-- name: CreateModel :one
INSERT INTO models (customer_id, model_group_id, name) VALUES ($1, $2, $3)
RETURNING id;

-- name: UpdateModel :exec
UPDATE models SET name = $2, model_group_id = $3 WHERE id = $1;

-- name: ListPartNumbers :many
-- ListPartNumbers mengambil part number satu customer beserta nama modelnya.
SELECT p.id, p.customer_id, p.model_id, m.name AS model_name, p.number, p.name, p.created_at
FROM part_numbers p
JOIN models m ON m.id = p.model_id
WHERE p.customer_id = $1
ORDER BY m.name, p.number;

-- name: GetPartNumber :one
SELECT p.id, p.customer_id, p.model_id, m.name AS model_name, p.number, p.name, p.created_at
FROM part_numbers p
JOIN models m ON m.id = p.model_id
WHERE p.id = $1;

-- name: LockPartNumber :exec
SELECT id FROM part_numbers WHERE id = $1 FOR NO KEY UPDATE;

-- name: CreatePartNumber :one
INSERT INTO part_numbers (customer_id, model_id, number, name) VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: UpdatePartNumber :exec
UPDATE part_numbers SET number = $2, name = $3 WHERE id = $1;
