-- name: ListProcesses :many
-- ListProcesses mengambil semua proses beserta nama tampilannya, urut menurut nama.
SELECT p.id, n.name, p.created_at
FROM processes p
JOIN process_names n ON n.process_id = p.id AND n.kind = 'display'
ORDER BY n.name_key;

-- name: ListProcessAliases :many
-- ListProcessAliases mengambil alias semua proses, urut menurut alias.
SELECT process_id, name FROM process_names WHERE kind = 'alias' ORDER BY name_key;

-- name: GetProcess :one
SELECT p.id, n.name, p.created_at
FROM processes p
JOIN process_names n ON n.process_id = p.id AND n.kind = 'display'
WHERE p.id = $1;

-- name: ListAliasesOfProcess :many
SELECT name FROM process_names WHERE process_id = $1 AND kind = 'alias' ORDER BY name_key;

-- name: LockProcess :exec
-- LockProcess mengunci nama tampilan satu proses sampai transaksi selesai,
-- supaya dua perubahan pada proses yang sama berjalan bergantian. Yang dikunci
-- baris nama, bukan baris processes: baris processes nanti dirujuk baris
-- dokumen (foreign key), dan kunci di sana tidak perlu ikut menunggu.
SELECT process_id FROM process_names WHERE process_id = $1 AND kind = 'display' FOR UPDATE;

-- name: CreateProcess :one
INSERT INTO processes DEFAULT VALUES RETURNING id;

-- name: AddProcessName :exec
INSERT INTO process_names (process_id, kind, name) VALUES ($1, $2, $3);

-- name: RenameProcess :exec
UPDATE process_names SET name = $2 WHERE process_id = $1 AND kind = 'display';

-- name: DeleteProcessAliases :exec
DELETE FROM process_names WHERE process_id = $1 AND kind = 'alias';
