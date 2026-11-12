-- M4.1 · Dokumen dan baris. Lihat migrasi 00021.

-- name: GetNumberForDocument :one
-- GetNumberForDocument mengunci nomor register yang akan dipakai dokumen.
SELECT n.id, n.number, n.customer_id, n.document_type_id, n.status, n.initial_revision, d.key AS document_type_key
FROM document_numbers n
JOIN document_types d ON d.id = n.document_type_id
WHERE n.id = $1
FOR NO KEY UPDATE OF n;


-- name: DocumentExistsForNumber :one
SELECT EXISTS (SELECT 1 FROM documents WHERE document_number_id = $1);


-- name: GetCustomerForDocument :one
SELECT id, default_route_id, default_form_set_id FROM customers WHERE id = $1;


-- name: LatestPublishedFormSetVersion :one
-- Versi terbit terakhir satu form set, beserta nama set dan versi format ketiga sheet.
SELECT v.id, s.name, v.pfd_format_version_id, v.pfmea_format_version_id, v.cp_format_version_id
FROM form_set_versions v
JOIN form_sets s ON s.id = v.form_set_id
WHERE v.form_set_id = $1 AND v.status = 'published'
ORDER BY v.version DESC
LIMIT 1;


-- name: GetModelOwner :one
SELECT customer_id FROM models WHERE id = $1;


-- name: GetPartNumberOwner :one
SELECT customer_id, model_id FROM part_numbers WHERE id = $1;


-- name: RouteExists :one
SELECT EXISTS (SELECT 1 FROM approval_routes WHERE id = $1);


-- name: CreateDocument :one
INSERT INTO documents (document_number_id, customer_id, document_type_id, revision, form_set_version_id,
                       format_version_id, approval_route_id, route_reason, created_by_person_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;


-- name: AddDocumentModel :exec
INSERT INTO document_models (document_id, model_id) VALUES ($1, $2);


-- name: AddDocumentPartNumber :exec
INSERT INTO document_part_numbers (document_id, part_number_id) VALUES ($1, $2);


-- name: GetDocument :one
SELECT d.id, d.document_number_id, n.number, d.customer_id, c.code AS customer_code, d.document_type_id,
       t.key AS document_type_key, t.name AS document_type_name, d.revision, d.status,
       d.form_set_version_id, s.name AS form_set_name, d.format_version_id,
       d.approval_route_id, r.name AS route_name, d.route_reason, d.header, d.created_at, d.updated_at
FROM documents d
JOIN document_numbers n ON n.id = d.document_number_id
JOIN customers c ON c.id = d.customer_id
JOIN document_types t ON t.id = d.document_type_id
JOIN form_set_versions fv ON fv.id = d.form_set_version_id
JOIN form_sets s ON s.id = fv.form_set_id
LEFT JOIN approval_routes r ON r.id = d.approval_route_id
WHERE d.id = $1;


-- name: LockDocument :exec
SELECT id FROM documents WHERE id = $1 FOR NO KEY UPDATE;


-- name: ListDocuments :many
SELECT d.id, n.number, t.key AS document_type_key, t.name AS document_type_name, d.revision, d.status,
       s.name AS form_set_name, d.updated_at,
       (SELECT count(*) FROM document_rows r WHERE r.document_id = d.id)::int AS row_count
FROM documents d
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_types t ON t.id = d.document_type_id
JOIN form_set_versions fv ON fv.id = d.form_set_version_id
JOIN form_sets s ON s.id = fv.form_set_id
WHERE d.customer_id = $1
ORDER BY n.number;


-- name: ListDocumentModels :many
SELECT m.id, m.name
FROM document_models dm JOIN models m ON m.id = dm.model_id
WHERE dm.document_id = $1
ORDER BY m.name;


-- name: ListDocumentPartNumbers :many
SELECT p.id, p.number, p.name
FROM document_part_numbers dp JOIN part_numbers p ON p.id = dp.part_number_id
WHERE dp.document_id = $1
ORDER BY p.number;


-- name: ListDocumentRows :many
SELECT r.id, r.position, r.process_id, n.name AS process_name, r.operation_number, r.content
FROM document_rows r
LEFT JOIN process_names n ON n.process_id = r.process_id AND n.kind = 'display'
WHERE r.document_id = $1
ORDER BY r.position;


-- name: AddDocumentRow :one
INSERT INTO document_rows (document_id, position, process_id, operation_number, content)
VALUES ($1, (SELECT coalesce(max(position), 0) + 1 FROM document_rows WHERE document_id = $1), $2, $3, $4)
RETURNING id, position;


-- name: UpdateDocumentRow :execrows
UPDATE document_rows
SET process_id = $3, operation_number = $4, content = $5, updated_at = now()
WHERE id = $1 AND document_id = $2;


-- name: SetDocumentHeader :exec
UPDATE documents SET header = $2, updated_at = now() WHERE id = $1;


-- name: TouchDocument :exec
UPDATE documents SET updated_at = now() WHERE id = $1;


-- name: ProcessExists :one
SELECT EXISTS (SELECT 1 FROM processes WHERE id = $1);