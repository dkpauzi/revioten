-- M5.8: mode tautan dokumen ke process template (R-TPL-08).

-- name: ListDocumentTemplates :many
-- ListDocumentTemplates memberi process template yang dipakai baris dokumen
-- ini beserta modenya; tanpa baris mode berarti follow.
SELECT t.id, pn.name AS process_name, coalesce(m.mode, 'follow')::text AS mode, m.reason,
       (SELECT max(v2.version) FROM document_rows r2 JOIN process_template_versions v2 ON v2.id = r2.template_version_id
        WHERE r2.document_id = sqlc.arg(document_id) AND v2.template_id = t.id)::int AS version
FROM process_templates t
JOIN process_names pn ON pn.process_id = t.process_id AND pn.kind = 'display'
LEFT JOIN document_template_modes m ON m.document_id = sqlc.arg(document_id) AND m.process_template_id = t.id
WHERE EXISTS (SELECT 1 FROM document_rows r JOIN process_template_versions v ON v.id = r.template_version_id
              WHERE r.document_id = sqlc.arg(document_id) AND v.template_id = t.id)
ORDER BY pn.name;

-- name: GetDocumentTemplateMode :one
SELECT mode FROM document_template_modes WHERE document_id = $1 AND process_template_id = $2;

-- name: SetDocumentTemplateMode :exec
INSERT INTO document_template_modes (document_id, process_template_id, mode, reason, updated_by_person_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (document_id, process_template_id)
DO UPDATE SET mode = EXCLUDED.mode, reason = EXCLUDED.reason, updated_by_person_id = EXCLUDED.updated_by_person_id, updated_at = now();

-- name: UpdateRowFromTemplate :exec
-- UpdateRowFromTemplate menulis isi versi template baru ke satu baris draft.
UPDATE document_rows SET content = $3, template_version_id = $4, version = version + 1, updated_at = now()
WHERE id = $1 AND document_id = $2;
