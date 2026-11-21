-- D-207 · Form Control Submission per dokumen.

-- name: SubmissionRows :many
-- SubmissionRows memberi setiap revisi satu dokumen, lama ke baru, dengan
-- kolom form yang dihitung: perubahan pertama yang diterima (tanggal mulai),
-- rilis (tanggal selesai), tanda tangan top management (tanda tangan digital
-- terakhir, atau scan bertanda tangan diunggah), dan catatan submission.
SELECT r.id, r.revision, r.status, r.created_at, r.release_date, r.released_at, r.migrated,
       r.scan_uploaded_at,
       -- Tanggal mulai: perubahan pertama diterima; revisi awal: draft dibuat.
       coalesce((SELECT min(c.verified_at) FROM change_documents cd JOIN changes c ON c.id = cd.change_id
                 WHERE cd.draft_revision_id = r.id AND c.status = 'accepted'), r.created_at)::timestamptz AS started_at,
       -- Permintaan tanda tangan digital yang berjalan atau lengkap (paling banyak satu).
       sr.status AS signing, sr.closed_at AS signed_at,
       s.required_override, s.override_reason, s.submitted_on, s.package, s.result, s.result_on,
       s.result_scan_name, s.result_scan_sha256, s.remark, up.name AS updated_by, s.updated_at
FROM document_revisions r
LEFT JOIN signature_requests sr ON sr.revision_id = r.id AND sr.status IN ('waiting', 'completed')
LEFT JOIN document_submissions s ON s.revision_id = r.id
LEFT JOIN people up ON up.id = s.updated_by_person_id
WHERE r.document_id = $1
ORDER BY r.revision;

-- name: SubmissionReasons :many
-- SubmissionReasons memberi perubahan yang masuk ke satu revisi beserta
-- pemicunya, untuk kolom Reason.
SELECT c.seq, c.mechanism, coalesce(src.name, '')::text AS trigger_name
FROM change_documents cd
JOIN changes c ON c.id = cd.change_id
LEFT JOIN change_sources src ON src.id = c.trigger_source_id
WHERE cd.draft_revision_id = $1 AND c.status = 'accepted'
ORDER BY c.verified_at;

-- name: SubmissionHeader :one
SELECT d.id, n.number, t.key AS type_key, t.name AS type_name, c.code AS customer_code, c.name AS customer_name,
       c.submission_required,
       coalesce((SELECT string_agg(m.name, ', ' ORDER BY m.name) FROM document_models dm JOIN models m ON m.id = dm.model_id
                 WHERE dm.document_id = d.id), '')::text AS models,
       (SELECT submission_form_number FROM system_settings) AS form_number
FROM documents d
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_types t ON t.id = d.document_type_id
JOIN customers c ON c.id = d.customer_id
WHERE d.id = $1;

-- name: RevisionOfDocument :one
SELECT id, status FROM document_revisions WHERE document_id = $1 AND revision = $2;

-- name: LockSubmission :one
-- LockSubmission memastikan baris submission revisi ada, lalu menguncinya.
INSERT INTO document_submissions (revision_id, document_id, updated_by_person_id)
VALUES ($1, $2, $3)
ON CONFLICT (revision_id) DO UPDATE SET updated_at = document_submissions.updated_at
RETURNING revision_id;

-- name: SaveSubmission :exec
UPDATE document_submissions
SET required_override = sqlc.narg(required_override), override_reason = sqlc.narg(override_reason),
    submitted_on = sqlc.narg(submitted_on), package = sqlc.narg(package), result = sqlc.narg(result),
    result_on = sqlc.narg(result_on), remark = sqlc.narg(remark),
    updated_by_person_id = sqlc.arg(person_id), updated_at = now()
WHERE revision_id = sqlc.arg(revision_id);

-- name: SetSubmissionScan :exec
UPDATE document_submissions
SET result_scan_name = $2, result_scan_type = $3, result_scan_size = $4, result_scan_sha256 = $5,
    updated_by_person_id = $6, updated_at = now()
WHERE revision_id = $1;

-- name: GetSubmissionScan :one
SELECT s.result_scan_name, s.result_scan_type, s.result_scan_size, s.result_scan_sha256
FROM document_submissions s
JOIN document_revisions r ON r.id = s.revision_id
WHERE r.document_id = $1 AND r.revision = $2 AND s.result_scan_sha256 IS NOT NULL;

-- name: SetCustomerSubmissionRequired :exec
UPDATE customers SET submission_required = $2 WHERE id = $1;

-- name: SetSubmissionFormNumber :exec
UPDATE system_settings SET submission_form_number = $1;
