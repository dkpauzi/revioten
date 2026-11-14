-- M5.1: perubahan dan dokumen terdampaknya (R-CHG-01).

-- name: CreateChange :one
INSERT INTO changes (customer_id, mechanism, trigger_source_id, change_type_id, four_m_category_id, reason, source_reference,
                     requested_by_person_id, requested_by_account_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;

-- name: GetChange :one
SELECT c.id, c.seq, c.customer_id, cu.code AS customer_code, c.mechanism,
       c.trigger_source_id, s.name AS trigger_name,
       c.change_type_id, t.key AS change_type_key, t.name AS change_type_name,
       c.four_m_category_id, f.key AS four_m_key, f.name AS four_m_name,
       c.reason, c.source_reference, c.status, p.name AS requested_by, c.requested_by_person_id,
       c.created_at, c.updated_at
FROM changes c
JOIN customers cu ON cu.id = c.customer_id
JOIN change_types t ON t.id = c.change_type_id
JOIN people p ON p.id = c.requested_by_person_id
LEFT JOIN change_sources s ON s.id = c.trigger_source_id
LEFT JOIN four_m_categories f ON f.id = c.four_m_category_id
WHERE c.id = $1;

-- name: ListChangeIDs :many
-- ListChangeIDs memberi perubahan satu customer, yang terbaru dulu.
SELECT id FROM changes WHERE customer_id = $1 ORDER BY seq DESC;

-- name: ListChangeIDsForDocument :many
SELECT c.id FROM changes c JOIN change_documents cd ON cd.change_id = c.id
WHERE cd.document_id = $1 ORDER BY c.seq DESC;

-- name: LockChange :exec
SELECT id FROM changes WHERE id = $1 FOR NO KEY UPDATE;

-- name: ListChangeDocuments :many
-- ListChangeDocuments memberi dokumen terdampak menurut urutan jenis dokumen
-- (PFD, PFMEA, CP), lalu nomornya.
SELECT d.id, n.number, t.id AS document_type_id, t.key AS document_type_key, t.name AS document_type_name, d.revision, d.status
FROM change_documents cd
JOIN documents d ON d.id = cd.document_id
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_types t ON t.id = d.document_type_id
WHERE cd.change_id = $1
ORDER BY t.position, n.number;

-- name: AddChangeDocument :exec
INSERT INTO change_documents (change_id, document_id) VALUES ($1, $2);

-- name: RemoveChangeDocument :execrows
DELETE FROM change_documents WHERE change_id = $1 AND document_id = $2;

-- name: GetDocumentForChange :one
-- GetDocumentForChange membaca customer dan nomor satu dokumen, untuk
-- memeriksa bahwa dokumen terdampak milik customer perubahan.
SELECT d.id, d.customer_id, n.number
FROM documents d JOIN document_numbers n ON n.id = d.document_number_id
WHERE d.id = $1;

-- name: GetChangeSourceForChange :one
SELECT id, name, mechanism, active FROM change_sources WHERE id = $1;

-- name: ChangeTypeExists :one
SELECT EXISTS (SELECT 1 FROM change_types WHERE id = $1);

-- name: FourMCategoryExists :one
SELECT EXISTS (SELECT 1 FROM four_m_categories WHERE id = $1);


-- M5.2: lampiran.

-- name: AddChangeAttachment :one
INSERT INTO change_attachments (change_id, file_name, content_type, size_bytes, sha256, uploaded_by_person_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: ListChangeAttachments :many
SELECT a.id, a.file_name, a.content_type, a.size_bytes, a.sha256, p.name AS uploaded_by, a.uploaded_at
FROM change_attachments a
JOIN people p ON p.id = a.uploaded_by_person_id
WHERE a.change_id = $1 AND a.removed_at IS NULL
ORDER BY a.uploaded_at, a.id;

-- name: RemoveChangeAttachment :execrows
UPDATE change_attachments SET removed_at = now()
WHERE id = $1 AND change_id = $2 AND removed_at IS NULL;
