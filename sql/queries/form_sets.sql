-- Form set (langkah D, D-133, D-139). Lihat migrasi 00019.

-- name: ListFormSets :many
-- Set bawaan dulu, lalu menurut waktu dibuat.
SELECT id, key, name, is_default, created_at
FROM form_sets
ORDER BY is_default DESC, created_at, id;


-- name: GetFormSet :one
SELECT id, key, name, is_default, created_at
FROM form_sets
WHERE id = $1;


-- name: GetDefaultFormSet :one
SELECT id, key, name, is_default, created_at
FROM form_sets
WHERE is_default;


-- name: ListFormSetKeys :many
SELECT key FROM form_sets;


-- name: CreateFormSet :one
INSERT INTO form_sets (key, name)
VALUES ($1, $2)
RETURNING id;


-- name: ListFormSetVersions :many
-- ListFormSetVersions mengambil semua versi semua form set beserta nama
-- template, nomor versi, dan status ketiga sheet-nya.
SELECT v.id, v.form_set_id, v.version, v.status, v.published_at, v.prepared_by_person_id,
       v.pfd_format_version_id, pfd_t.name AS pfd_template_name, pfd.version AS pfd_version, pfd.status AS pfd_status,
       v.pfmea_format_version_id, pfmea_t.name AS pfmea_template_name, pfmea.version AS pfmea_version, pfmea.status AS pfmea_status,
       v.cp_format_version_id, cp_t.name AS cp_template_name, cp.version AS cp_version, cp.status AS cp_status
FROM form_set_versions v
JOIN format_template_versions pfd ON pfd.id = v.pfd_format_version_id
JOIN format_templates pfd_t ON pfd_t.id = pfd.template_id
JOIN format_template_versions pfmea ON pfmea.id = v.pfmea_format_version_id
JOIN format_templates pfmea_t ON pfmea_t.id = pfmea.template_id
JOIN format_template_versions cp ON cp.id = v.cp_format_version_id
JOIN format_templates cp_t ON cp_t.id = cp.template_id
ORDER BY v.form_set_id, v.version;


-- name: GetFormSetVersion :one
SELECT id, form_set_id, version, status, prepared_by_person_id,
       pfd_format_version_id, pfmea_format_version_id, cp_format_version_id
FROM form_set_versions
WHERE id = $1;


-- name: LockFormSetVersion :exec
-- LockFormSetVersion mengunci baris versi sampai transaksi selesai.
SELECT id FROM form_set_versions WHERE id = $1 FOR NO KEY UPDATE;


-- name: LastFormSetVersion :one
SELECT id, form_set_id, version, status, prepared_by_person_id,
       pfd_format_version_id, pfmea_format_version_id, cp_format_version_id
FROM form_set_versions
WHERE form_set_id = $1
ORDER BY version DESC
LIMIT 1;


-- name: CreateFormSetVersion :one
INSERT INTO form_set_versions (form_set_id, version, pfd_format_version_id, pfmea_format_version_id, cp_format_version_id, prepared_by_person_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;


-- name: SetFormSetSheets :exec
UPDATE form_set_versions
SET pfd_format_version_id = $2, pfmea_format_version_id = $3, cp_format_version_id = $4, prepared_by_person_id = $5
WHERE id = $1;


-- name: PublishFormSetVersion :execrows
UPDATE form_set_versions
SET status = 'published', published_at = now()
WHERE id = $1 AND status = 'draft';


-- name: GetFormatVersionSheet :one
-- GetFormatVersionSheet memberi ciri satu versi format untuk diperiksa
-- sebagai isi sheet: jenis dokumen, nama template, nomor versi, dan status.
SELECT v.id, v.version, v.status, t.name AS template_name, d.key AS document_type_key
FROM format_template_versions v
JOIN format_templates t ON t.id = v.template_id
JOIN document_types d ON d.id = t.document_type_id
WHERE v.id = $1;


-- name: ListCustomerFormSets :many
-- ListCustomerFormSets mengambil form set bawaan setiap customer; kosong
-- berarti set bawaan sistem.
SELECT id, code, name, default_form_set_id
FROM customers
ORDER BY code;


-- name: GetCustomerFormSet :one
SELECT c.id, c.code, c.default_form_set_id
FROM customers c
WHERE c.id = $1;


-- name: SetCustomerFormSet :exec
UPDATE customers SET default_form_set_id = $2 WHERE id = $1;