-- M5.9: yokoten (R-CHG-04, D-40, D-118).

-- name: ListModelsOfCustomer :many
SELECT id FROM models WHERE customer_id = $1 ORDER BY name;

-- name: ListModelsOfGroup :many
SELECT id FROM models WHERE model_group_id = $1 ORDER BY name;

-- name: GetModelGroupCustomer :one
SELECT customer_id FROM model_groups WHERE id = $1;

-- name: ListYokotenCandidates :many
-- ListYokotenCandidates memberi dokumen model-model ini yang cocok dengan
-- saringan proses dan teks (keduanya boleh kosong), menurut customer, model,
-- lalu jenis dokumen. Teks dicari di isi baris; tanda % dan _ sudah di-escape
-- aplikasi.
SELECT m.id AS model_id, m.name AS model_name, m.customer_id, c.code AS customer_code,
       d.id AS document_id, n.number, t.key AS document_type_key
FROM document_models dm
JOIN models m ON m.id = dm.model_id
JOIN customers c ON c.id = m.customer_id
JOIN documents d ON d.id = dm.document_id
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_types t ON t.id = d.document_type_id
WHERE dm.model_id = ANY (sqlc.arg(model_ids)::uuid[])
  AND (sqlc.narg(process_id)::uuid IS NULL
       OR EXISTS (SELECT 1 FROM document_rows r WHERE r.document_id = d.id AND r.process_id = sqlc.narg(process_id)::uuid))
  AND (sqlc.narg(text)::text IS NULL
       OR EXISTS (SELECT 1 FROM document_rows r WHERE r.document_id = d.id AND r.content::text ILIKE '%' || sqlc.narg(text)::text || '%'))
ORDER BY c.code, m.name, t.position, n.number;

-- name: CreateYokoten :one
INSERT INTO yokotens (source_change_id, reason, filter_process_id, filter_text, created_by_person_id, created_by_account_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: AddYokotenTarget :one
INSERT INTO yokoten_targets (yokoten_id, model_id, customer_id) VALUES ($1, $2, $3) RETURNING id;

-- name: AddYokotenTargetDocument :exec
INSERT INTO yokoten_target_documents (target_id, document_id) VALUES ($1, $2);

-- name: GetYokoten :one
SELECT y.id, y.seq, y.source_change_id, sc.seq AS source_seq, sc.customer_id AS source_customer_id, y.reason,
       y.filter_process_id, pn.name AS filter_process_name, y.filter_text, p.name AS created_by, y.created_at
FROM yokotens y
JOIN changes sc ON sc.id = y.source_change_id
JOIN people p ON p.id = y.created_by_person_id
LEFT JOIN process_names pn ON pn.process_id = y.filter_process_id AND pn.kind = 'display'
WHERE y.id = $1;

-- name: ListYokotenIDs :many
SELECT id FROM yokotens ORDER BY seq DESC;

-- name: ListYokotenTargets :many
SELECT t.id, t.model_id, m.name AS model_name, t.customer_id, c.code AS customer_code, t.decision, t.reason,
       dp.name AS decided_by, t.decided_at, t.change_id, ch.seq AS change_seq, ch.status AS change_status
FROM yokoten_targets t
JOIN models m ON m.id = t.model_id
JOIN customers c ON c.id = t.customer_id
LEFT JOIN people dp ON dp.id = t.decided_by_person_id
LEFT JOIN changes ch ON ch.id = t.change_id
WHERE t.yokoten_id = $1
ORDER BY c.code, m.name;

-- name: ListYokotenTargetDocuments :many
SELECT d.id, n.number, ty.key AS document_type_key, d.revision, d.status, d.released_revision
FROM yokoten_target_documents td
JOIN documents d ON d.id = td.document_id
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_types ty ON ty.id = d.document_type_id
WHERE td.target_id = $1
ORDER BY ty.position, n.number;

-- name: GetYokotenTarget :one
SELECT id, yokoten_id, model_id, customer_id, decision FROM yokoten_targets WHERE id = $1 FOR UPDATE;

-- name: DecideYokotenTarget :execrows
UPDATE yokoten_targets
SET decision = sqlc.arg(decision), reason = sqlc.narg(reason), decided_by_person_id = sqlc.arg(person_id), decided_at = now(),
    change_id = sqlc.narg(change_id)
WHERE id = sqlc.arg(id) AND decision = 'pending';

-- name: CreateYokotenChange :one
-- CreateYokotenChange membuat perubahan bermekanisme yokoten untuk satu
-- target, dengan jenis perubahan dan kategori 4M dari perubahan sumbernya.
INSERT INTO changes (customer_id, mechanism, change_type_id, four_m_category_id, reason, source_reference,
                     requested_by_person_id, requested_by_account_id, source_change_id)
SELECT sqlc.arg(customer_id), 'yokoten', s.change_type_id, s.four_m_category_id, sqlc.arg(reason), sqlc.narg(source_reference),
       sqlc.arg(person_id), sqlc.arg(account_id), s.id
FROM changes s WHERE s.id = sqlc.arg(source_change_id)
RETURNING id;
