-- D-130, D-156: master dokumen terkait (R-DOC-11).

-- name: ListReferencedDocuments :many
SELECT id, number, title, level, revision, revision_date, active, created_at
FROM referenced_documents
ORDER BY number;

-- name: GetReferencedDocument :one
SELECT id, number, title, level, revision, revision_date, active, created_at
FROM referenced_documents
WHERE id = $1;

-- name: LockReferencedDocument :exec
SELECT id FROM referenced_documents WHERE id = $1 FOR NO KEY UPDATE;

-- name: CreateReferencedDocument :one
INSERT INTO referenced_documents (number, title, level, revision, revision_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: UpdateReferencedDocument :exec
UPDATE referenced_documents
SET title = $2, level = $3, revision = $4, revision_date = $5, active = $6, updated_at = now()
WHERE id = $1;

-- name: ListReferencedDocumentLinks :many
-- ListReferencedDocumentLinks mengambil semua tautan beserta nama proses dan
-- nama tampilan sasaran cakupannya, menurut nama proses lalu cakupan.
SELECT l.id, l.document_id, l.process_id, n.name AS process_name,
       l.scope, l.customer_id, l.model_group_id, l.model_id,
       c.code AS customer_code, g.name AS model_group_name, m.name AS model_name
FROM referenced_document_links l
JOIN process_names n ON n.process_id = l.process_id AND n.kind = 'display'
LEFT JOIN customers c ON c.id = l.customer_id
LEFT JOIN model_groups g ON g.id = l.model_group_id
LEFT JOIN models m ON m.id = l.model_id
ORDER BY n.name_key, CASE l.scope WHEN 'general' THEN 0 WHEN 'customer' THEN 1 WHEN 'model_group' THEN 2 ELSE 3 END, coalesce(c.code, g.name, m.name);

-- name: AddReferencedDocumentLink :one
INSERT INTO referenced_document_links (document_id, process_id, scope, customer_id, model_group_id, model_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: DeleteReferencedDocumentLink :one
DELETE FROM referenced_document_links WHERE id = $1 AND document_id = $2
RETURNING process_id, scope;

-- name: ReferencedNumbersForModel :many
-- ReferencedNumbersForModel memberi nomor dokumen aktif yang tertaut ke satu
-- proses dan berlaku untuk satu model: tautan umum, customer model itu, group
-- model-nya, dan model itu sendiri, dijumlahkan dan diurutkan.
SELECT DISTINCT d.number
FROM referenced_document_links l
JOIN referenced_documents d ON d.id = l.document_id
WHERE l.process_id = sqlc.arg(process_id) AND d.active
  AND (l.scope = 'general'
       OR l.customer_id = sqlc.arg(customer_id)
       OR l.model_group_id = sqlc.narg(model_group_id)
       OR l.model_id = sqlc.arg(model_id))
ORDER BY d.number;