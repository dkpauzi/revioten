-- Process template atau pustaka proses (M3.2). Lihat migrasi 00012.

-- name: ListProcessTemplates :many
-- ListProcessTemplates mengambil semua process template beserta nama tampilan
-- prosesnya, urut menurut nama proses.
SELECT t.id, t.process_id, n.name AS process_name, t.created_at
FROM process_templates t
JOIN process_names n ON n.process_id = t.process_id AND n.kind = 'display'
ORDER BY n.name_key;

-- name: GetProcessTemplate :one
SELECT t.id, t.process_id, n.name AS process_name, t.created_at
FROM process_templates t
JOIN process_names n ON n.process_id = t.process_id AND n.kind = 'display'
WHERE t.id = $1;

-- name: CreateProcessTemplate :one
INSERT INTO process_templates (process_id) VALUES ($1) RETURNING id;

-- name: ListAllProcessVersions :many
-- ListAllProcessVersions mengambil ringkasan versi semua template, untuk daftar.
SELECT id, template_id, version, status
FROM process_template_versions
ORDER BY template_id, version;

-- name: ListProcessVersionsOfTemplate :many
SELECT id, template_id, version, status
FROM process_template_versions
WHERE template_id = $1
ORDER BY version;

-- name: CreateProcessVersion :one
INSERT INTO process_template_versions (template_id, process_id, version)
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetProcessVersion :one
SELECT v.id, v.template_id, v.process_id, n.name AS process_name, v.version, v.status, v.created_at
FROM process_template_versions v
JOIN process_names n ON n.process_id = v.process_id AND n.kind = 'display'
WHERE v.id = $1;

-- name: LockProcessVersion :exec
-- LockProcessVersion mengunci baris versi sampai transaksi selesai, supaya
-- dua perubahan pada versi yang sama berjalan bergantian.
SELECT id FROM process_template_versions WHERE id = $1 FOR NO KEY UPDATE;

-- name: CreateSubprocess :one
INSERT INTO subprocesses (process_id) VALUES ($1) RETURNING id;

-- name: AddVersionSubprocess :exec
INSERT INTO process_template_subprocesses (version_id, process_id, subprocess_id, name, position)
VALUES ($1, $2, $3, $4, $5);

-- name: ListVersionSubprocesses :many
-- ListVersionSubprocesses mengambil sub-proses satu versi menurut urutan.
SELECT subprocess_id, name, position, hidden
FROM process_template_subprocesses
WHERE version_id = $1
ORDER BY position;

-- name: UpdateVersionSubprocess :exec
UPDATE process_template_subprocesses
SET name = $3, hidden = $4
WHERE version_id = $1 AND subprocess_id = $2;

-- name: ShiftSubprocessPositions :exec
-- ShiftSubprocessPositions menggeser urutan sub-proses di antara first dan
-- last sebanyak delta (+1 atau -1), untuk memberi tempat pada sub-proses yang
-- dipindahkan. Keunikan urutan baru diperiksa di akhir transaksi.
UPDATE process_template_subprocesses
SET position = position + sqlc.arg(delta)::smallint
WHERE version_id = sqlc.arg(version_id)
  AND position BETWEEN sqlc.arg(first_position)::smallint AND sqlc.arg(last_position)::smallint;

-- name: SetSubprocessPosition :exec
UPDATE process_template_subprocesses SET position = $3 WHERE version_id = $1 AND subprocess_id = $2;

-- name: CreateTemplatePoint :one
INSERT INTO template_points (subprocess_id, document_type_id) VALUES ($1, $2) RETURNING id;

-- name: AddVersionPoint :exec
INSERT INTO process_template_points (version_id, subprocess_id, document_type_id, point_id, text, position)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListVersionPoints :many
-- ListVersionPoints mengambil poin satu versi: menurut urutan sub-proses,
-- lalu urutan jenis dokumen (PFD, PFMEA, CP), lalu urutan poin.
SELECT vp.point_id, vp.subprocess_id, vp.document_type_id, d.key AS document_type_key,
       d.name AS document_type_name, vp.text, vp.position, vp.hidden
FROM process_template_points vp
JOIN process_template_subprocesses vs
  ON vs.version_id = vp.version_id AND vs.subprocess_id = vp.subprocess_id
JOIN document_types d ON d.id = vp.document_type_id
WHERE vp.version_id = $1
ORDER BY vs.position, d.position, vp.position;

-- name: UpdateVersionPoint :exec
UPDATE process_template_points
SET text = $3, hidden = $4
WHERE version_id = $1 AND point_id = $2;

-- name: ShiftPointPositions :exec
-- ShiftPointPositions bekerja seperti ShiftSubprocessPositions, di dalam satu
-- sub-proses dan satu jenis dokumen.
UPDATE process_template_points
SET position = position + sqlc.arg(delta)::smallint
WHERE version_id = sqlc.arg(version_id) AND subprocess_id = sqlc.arg(subprocess_id)
  AND document_type_id = sqlc.arg(document_type_id)
  AND position BETWEEN sqlc.arg(first_position)::smallint AND sqlc.arg(last_position)::smallint;

-- name: SetPointPosition :exec
UPDATE process_template_points SET position = $3 WHERE version_id = $1 AND point_id = $2;
