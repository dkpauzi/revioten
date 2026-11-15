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
       c.reason, c.source_reference, c.status,
       -- Pemohon: orang, atau service account untuk usulan dari luar (M9.4).
       coalesce(p.name, sa.name)::text AS requested_by, c.requested_by_person_id, c.requested_by_service_account_id,
       c.created_at, c.updated_at,
       c.verified_by_person_id, v.name AS verified_by, c.verified_at, c.verification_note,
       c.template_version_id, c.propagation_scope, c.conflict_policy, c.effective_date,
       c.source_change_id, src.seq AS source_change_seq
FROM changes c
JOIN customers cu ON cu.id = c.customer_id
JOIN change_types t ON t.id = c.change_type_id
LEFT JOIN people p ON p.id = c.requested_by_person_id
LEFT JOIN service_accounts sa ON sa.id = c.requested_by_service_account_id
LEFT JOIN change_sources s ON s.id = c.trigger_source_id
LEFT JOIN four_m_categories f ON f.id = c.four_m_category_id
LEFT JOIN people v ON v.id = c.verified_by_person_id
LEFT JOIN changes src ON src.id = c.source_change_id
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
SELECT d.id, n.number, t.id AS document_type_id, t.key AS document_type_key, t.name AS document_type_name, d.revision, d.status,
       d.released_revision, dr.revision AS draft_revision, cd.merged,
       cd.rows_updated, cd.rows_conflict, cd.rows_removed
FROM change_documents cd
JOIN documents d ON d.id = cd.document_id
LEFT JOIN document_revisions dr ON dr.id = cd.draft_revision_id
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

-- M5.3: verifikasi atasan (R-CHG-02, D-51, D-54).

-- name: VerifyChange :execrows
UPDATE changes
SET status = sqlc.arg(status), verified_by_person_id = sqlc.arg(person_id), verified_by_account_id = sqlc.arg(account_id),
    verified_at = now(), verification_note = sqlc.arg(note), updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'submitted';

-- name: IsChangeVerifier :one
-- IsChangeVerifier melaporkan apakah orang itu boleh memverifikasi perubahan
-- customer ini: Project Engineer atau Project Leader dari Project Group
-- customer, atau APQP/PPAP Controller (D-51, D-54).
SELECT EXISTS (
    SELECT 1
    FROM position_holders h
    JOIN positions p ON p.id = h.position_id
    LEFT JOIN customers c ON c.project_group_id = h.project_group_id AND c.id = sqlc.arg(customer_id)
    WHERE h.person_id = sqlc.arg(person_id)
      AND (p.key = 'apqp_ppap_controller' OR (p.key IN ('project_leader', 'project_engineer') AND c.id IS NOT NULL))
);

-- M5.7: daftar periksa dampak (R-CHG-05).

-- name: ListChangeImpacts :many
SELECT i.id, i.kind, i.document_reference, i.referenced_document_id, i.decision, i.reason, p.name AS updated_by, i.updated_at
FROM change_impacts i JOIN people p ON p.id = i.updated_by_person_id
WHERE i.change_id = $1
ORDER BY i.position;

-- name: ClearChangeImpacts :exec
DELETE FROM change_impacts WHERE change_id = $1;

-- name: AddChangeImpact :exec
INSERT INTO change_impacts (change_id, position, kind, document_reference, decision, reason, updated_by_person_id, referenced_document_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetReferencedDocumentForImpact :one
SELECT id, number, title, revision, active FROM referenced_documents WHERE id = $1;

-- M5.8: propagasi template (R-TPL-06 sampai R-TPL-09).

-- name: CreateTemplateChange :one
INSERT INTO changes (customer_id, mechanism, change_type_id, reason, source_reference, requested_by_person_id, requested_by_account_id,
                     template_version_id, propagation_scope, conflict_policy, effective_date)
VALUES ($1, 'template', $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: ListTemplateLinkedDocuments :many
-- ListTemplateLinkedDocuments memberi dokumen yang punya baris tertaut ke
-- versi lain process template ini dan belum diputus, menurut customer lalu nomor.
SELECT DISTINCT d.id, d.customer_id, n.number
FROM document_rows r
JOIN process_template_versions v ON v.id = r.template_version_id
JOIN documents d ON d.id = r.document_id
JOIN document_numbers n ON n.id = d.document_number_id
WHERE v.template_id = sqlc.arg(template_id) AND r.template_version_id <> sqlc.arg(version_id) AND r.detached_at IS NULL
ORDER BY d.customer_id, n.number;

-- name: SetChangeDocumentResult :exec
UPDATE change_documents SET rows_updated = $3, rows_conflict = $4, rows_removed = $5 WHERE change_id = $1 AND document_id = $2;

-- name: DocumentHasTemplateChange :one
-- DocumentHasTemplateChange melaporkan apakah dokumen sudah punya perubahan
-- yang belum ditolak untuk versi template ini, supaya propagasi tidak dobel.
SELECT EXISTS (
    SELECT 1 FROM change_documents cd JOIN changes c ON c.id = cd.change_id
    WHERE cd.document_id = $1 AND c.template_version_id = $2 AND c.status <> 'rejected'
);

-- M5.10: perubahan surut SQAM (R-CHG-01, R-TPL-07).

-- name: ListActiveCustomerDocuments :many
-- ListActiveCustomerDocuments memberi dokumen customer yang nomornya belum
-- dibatalkan atau usang, menurut jenis dokumen lalu nomor. type_ids kosong
-- berarti semua jenis dokumen (D-171).
SELECT d.id, n.number
FROM documents d
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_types t ON t.id = d.document_type_id
WHERE d.customer_id = sqlc.arg(customer_id) AND n.status NOT IN ('obsolete', 'void')
  AND (cardinality(sqlc.arg(type_ids)::uuid[]) = 0 OR d.document_type_id = ANY (sqlc.arg(type_ids)::uuid[]))
ORDER BY t.position, n.number;

-- name: CreateSQAMChange :one
INSERT INTO changes (customer_id, mechanism, change_type_id, four_m_category_id, reason, source_reference, effective_date,
                     requested_by_person_id, requested_by_account_id)
VALUES ($1, 'sqam', $2, $3, $4, $5, $6, $7, $8)
RETURNING id;
