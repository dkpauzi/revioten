-- Fase 4 · Dasbor admin (D-205).

-- name: AddErrorReport :exec
INSERT INTO error_reports (request_id, method, path, account_id, message)
VALUES ($1, $2, $3, sqlc.narg(account_id), $4);

-- name: CountErrorReportsSince :one
SELECT count(*)::integer FROM error_reports WHERE occurred_at >= $1;

-- name: ListErrorReports :many
SELECT e.id, e.occurred_at, e.request_id, e.method, e.path, e.message, a.login
FROM error_reports e
LEFT JOIN accounts a ON a.id = e.account_id
ORDER BY e.occurred_at DESC
LIMIT sqlc.arg(max_rows);

-- name: AddProblemReport :one
INSERT INTO problem_reports (person_id, account_id, page, description, last_request_id)
VALUES ($1, $2, $3, $4, sqlc.narg(last_request_id))
RETURNING id;

-- name: ListProblemReports :many
SELECT r.id, r.created_at, pe.name AS reported_by, a.login, r.page, r.description, r.last_request_id,
       r.status, hp.name AS handled_by, r.handled_at, r.admin_note
FROM problem_reports r
JOIN people pe ON pe.id = r.person_id
JOIN accounts a ON a.id = r.account_id
LEFT JOIN people hp ON hp.id = r.handled_by_person_id
ORDER BY (r.status = 'done'), r.created_at DESC
LIMIT sqlc.arg(max_rows);

-- name: CountOpenProblemReports :one
SELECT count(*)::integer FROM problem_reports WHERE status <> 'done';

-- name: UpdateProblemReport :execrows
UPDATE problem_reports
SET status = $2, admin_note = sqlc.narg(admin_note), handled_by_person_id = $3, handled_at = now()
WHERE id = $1;

-- name: WaitingSignatureSlots :many
-- WaitingSignatureSlots memberi setiap permintaan tanda tangan yang menunggu
-- beserta slot yang sedang ditunggu dan sejak kapan: tanda tangan sebelumnya,
-- atau waktu dikirim bila slot itu yang pertama ditunggu. Hanya nomor, slot,
-- dan nama, tanpa isi dokumen.
SELECT sr.id, sr.document_id, n.number, sr.revision, c.code AS customer_code, sr.customer_id,
       sl.print_title, sl.person_name, sr.sent_at, sp.name AS sent_by,
       coalesce((SELECT max(g.signed_at) FROM signatures g WHERE g.request_id = sr.id), sr.sent_at)::timestamptz AS waiting_since
FROM signature_requests sr
JOIN signature_slots sl ON sl.request_id = sr.id
JOIN documents d ON d.id = sr.document_id
JOIN document_numbers n ON n.id = d.document_number_id
JOIN customers c ON c.id = sr.customer_id
JOIN people sp ON sp.id = sr.sent_by_person_id
WHERE sr.status = 'waiting'
  AND sl.slot_order = (SELECT min(x.slot_order) FROM signature_slots x
                       WHERE x.request_id = sr.id
                         AND NOT EXISTS (SELECT 1 FROM signatures g WHERE g.request_id = x.request_id AND g.slot_order = x.slot_order))
ORDER BY waiting_since;

-- name: ProcessedRevisions :many
-- ProcessedRevisions memberi kelompok pemicu setiap revisi yang terbit dalam
-- rentang, termasuk revisi awal (tanpa perubahan) dan dokumen lama.
SELECT r.id, r.migrated,
       coalesce((SELECT src.report_key FROM change_documents cd JOIN changes c ON c.id = cd.change_id
                 LEFT JOIN change_sources src ON src.id = c.trigger_source_id
                 WHERE cd.draft_revision_id = r.id AND c.status = 'accepted' ORDER BY c.verified_at LIMIT 1), '')::text AS report_key,
       coalesce((SELECT c.mechanism FROM change_documents cd JOIN changes c ON c.id = cd.change_id
                 WHERE cd.draft_revision_id = r.id AND c.status = 'accepted' ORDER BY c.verified_at LIMIT 1), '')::text AS mechanism,
       EXISTS (SELECT 1 FROM change_documents cd WHERE cd.draft_revision_id = r.id) AS by_change
FROM document_revisions r
WHERE r.status = 'released' AND r.released_at >= sqlc.arg(from_at) AND r.released_at < sqlc.arg(to_at);

-- name: OutboxHealth :one
SELECT count(*) FILTER (WHERE status = 'failed')::integer AS failed,
       count(*) FILTER (WHERE status = 'pending')::integer AS pending
FROM notification_outbox;

-- name: AccountHealth :one
SELECT count(*) FILTER (WHERE locked_until > now() AND disabled_at IS NULL)::integer AS locked,
       count(*) FILTER (WHERE disabled_at IS NOT NULL)::integer AS disabled,
       (SELECT count(*) FROM audit_log WHERE action = 'account.login_failed' AND occurred_at >= now() - interval '24 hours')::integer AS failed_logins
FROM accounts;
