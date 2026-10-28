-- name: ListProjectGroups :many
SELECT id, code, name, created_at FROM project_groups ORDER BY code;

-- name: GetProjectGroup :one
SELECT id, code, name, created_at FROM project_groups WHERE id = $1;

-- name: LockProjectGroup :exec
SELECT id FROM project_groups WHERE id = $1 FOR NO KEY UPDATE;

-- name: CreateProjectGroup :one
INSERT INTO project_groups (code, name) VALUES ($1, $2)
RETURNING id, code, name, created_at;

-- name: UpdateProjectGroup :exec
UPDATE project_groups SET code = $2, name = $3 WHERE id = $1;
