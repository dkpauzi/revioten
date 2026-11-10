-- name: ListFormatTemplates :many
-- ListFormatTemplates mengambil semua format template, menurut urutan jenis
-- dokumen lalu nama (M3.1).
SELECT t.id, t.name, t.document_type_id, d.key AS document_type_key, d.name AS document_type_name,
       t.methodology, t.customer_id, c.code AS customer_code, t.created_at
FROM format_templates t
JOIN document_types d ON d.id = t.document_type_id
LEFT JOIN customers c ON c.id = t.customer_id
ORDER BY d.position, lower(t.name);

-- name: GetFormatTemplate :one
SELECT t.id, t.name, t.document_type_id, d.key AS document_type_key, d.name AS document_type_name,
       t.methodology, t.customer_id, c.code AS customer_code, t.created_at
FROM format_templates t
JOIN document_types d ON d.id = t.document_type_id
LEFT JOIN customers c ON c.id = t.customer_id
WHERE t.id = $1;

-- name: LockFormatTemplate :exec
-- LockFormatTemplate mengunci baris template sampai transaksi selesai.
-- FOR NO KEY UPDATE tidak bentrok dengan pemeriksaan foreign key (lihat LockAccount).
SELECT id FROM format_templates WHERE id = $1 FOR NO KEY UPDATE;

-- name: CreateFormatTemplate :one
INSERT INTO format_templates (name, document_type_id, methodology, customer_id)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: RenameFormatTemplate :exec
UPDATE format_templates SET name = $2 WHERE id = $1;

-- name: ListFormatVersions :many
-- ListFormatVersions mengambil semua versi semua template, untuk daftar template.
SELECT id, template_id, version, status, risk_method, highlight_rpn_above, highlight_severity_at_least, created_at, published_at
FROM format_template_versions
ORDER BY template_id, version;

-- name: GetFormatVersion :one
SELECT id, template_id, version, status, risk_method, highlight_rpn_above, highlight_severity_at_least, created_at, published_at
FROM format_template_versions
WHERE id = $1;

-- name: LockFormatVersion :exec
-- LockFormatVersion mengunci baris versi sampai transaksi selesai, supaya dua
-- perubahan pada versi yang sama berjalan bergantian.
SELECT id FROM format_template_versions WHERE id = $1 FOR NO KEY UPDATE;

-- name: CreateFormatVersion :one
INSERT INTO format_template_versions (template_id, version, risk_method, highlight_rpn_above, highlight_severity_at_least)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: SetFormatHighlights :exec
UPDATE format_template_versions
SET highlight_rpn_above = $2, highlight_severity_at_least = $3
WHERE id = $1;

-- name: CreateFormatField :one
INSERT INTO format_fields (template_id, area, kind, rating_scale, rating_set)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: AddVersionField :exec
INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position, required, hidden, options)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: ListVersionFields :many
-- ListVersionFields mengambil field satu versi: field header dulu, lalu
-- kolom baris, masing-masing menurut urutan.
SELECT vf.field_id, vf.area, f.kind, f.rating_scale, f.rating_set,
       vf.label, vf.position, vf.required, vf.hidden, vf.options
FROM format_version_fields vf
JOIN format_fields f ON f.id = vf.field_id
WHERE vf.version_id = $1
ORDER BY vf.area = 'row', vf.position;

-- name: UpdateVersionField :exec
UPDATE format_version_fields
SET label = $3, required = $4, hidden = $5, options = $6
WHERE version_id = $1 AND field_id = $2;

-- name: ShiftFieldPositions :exec
-- ShiftFieldPositions menggeser urutan field di antara first dan last
-- sebanyak delta (+1 atau -1), untuk memberi tempat pada field yang
-- dipindahkan. Keunikan urutan baru diperiksa di akhir transaksi.
UPDATE format_version_fields
SET position = position + sqlc.arg(delta)::smallint
WHERE version_id = sqlc.arg(version_id) AND area = sqlc.arg(area)
  AND position BETWEEN sqlc.arg(first_position)::smallint AND sqlc.arg(last_position)::smallint;

-- name: SetFieldPosition :exec
UPDATE format_version_fields SET position = $3 WHERE version_id = $1 AND field_id = $2;

-- name: ListRatingCriteria :many
-- ListRatingCriteria mengambil tabel skala satu versi: severity, occurrence,
-- lalu detection, masing-masing dari nilai 1 sampai 10.
SELECT scale, value, criteria
FROM format_rating_criteria
WHERE version_id = $1
ORDER BY array_position(ARRAY['severity', 'occurrence', 'detection'], scale), value;

-- name: AddRatingCriterion :exec
INSERT INTO format_rating_criteria (version_id, scale, value, criteria) VALUES ($1, $2, $3, $4);

-- name: SetRatingCriterion :exec
UPDATE format_rating_criteria SET criteria = $4 WHERE version_id = $1 AND scale = $2 AND value = $3;

-- name: ListFormatSymbols :many
SELECT key, name, position, automatic FROM format_symbols WHERE version_id = $1 ORDER BY position;

-- name: AddFormatSymbol :exec
-- AddFormatSymbol menambah simbol. automatic hanya true saat menyalin simbol
-- yang dipasang sistem (R-REV-09); simbol baru dari layar selalu false.
INSERT INTO format_symbols (version_id, key, name, position, automatic) VALUES ($1, $2, $3, $4, $5);

-- name: RenameFormatSymbol :exec
UPDATE format_symbols SET name = $3 WHERE version_id = $1 AND key = $2;

-- name: ListVersionsOfTemplate :many
SELECT id, template_id, version, status, risk_method, highlight_rpn_above, highlight_severity_at_least, created_at, published_at
FROM format_template_versions
WHERE template_id = $1
ORDER BY version;


-- M3.4: versi terbit dan versi baru. Lihat migrasi 00016.

-- name: PublishFormatVersion :execrows
-- PublishFormatVersion menerbitkan versi draft. Hasil 0 baris berarti versi
-- itu tidak ada atau sudah terbit.
UPDATE format_template_versions
SET status = 'published', published_at = now()
WHERE id = $1 AND status = 'draft';


-- name: CopyFormatVersionFields :exec
-- CopyFormatVersionFields menyalin field satu versi ke versi baru template
-- yang sama. ID field tetap, jadi data dokumen tetap terbaca (R-TPL-02).
INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position, required, hidden, options)
SELECT sqlc.arg(target_id)::uuid, s.field_id, s.template_id, s.area, s.label, s.position, s.required, s.hidden, s.options
FROM format_version_fields s
WHERE s.version_id = sqlc.arg(source_id);


-- name: CopyFormatRatingCriteria :exec
INSERT INTO format_rating_criteria (version_id, scale, value, criteria)
SELECT sqlc.arg(target_id)::uuid, s.scale, s.value, s.criteria
FROM format_rating_criteria s
WHERE s.version_id = sqlc.arg(source_id);


-- name: CopyFormatSymbols :exec
INSERT INTO format_symbols (version_id, key, name, position, automatic)
SELECT sqlc.arg(target_id)::uuid, s.key, s.name, s.position, s.automatic
FROM format_symbols s
WHERE s.version_id = sqlc.arg(source_id);
