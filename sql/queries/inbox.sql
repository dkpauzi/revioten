-- Daftar "Waiting for you" di Home (D-202). Query memberi calon tugas semua
-- customer; internal/inbox menyaringnya menurut hak orang yang login.

-- name: InboxChangesToVerify :many
-- InboxChangesToVerify memberi permintaan perubahan yang menunggu verifikasi.
SELECT c.id, c.seq, c.customer_id, cu.code AS customer_code, c.reason, c.created_at, c.requested_by_person_id,
       coalesce(p.name, sa.name, '')::text AS requested_by
FROM changes c
JOIN customers cu ON cu.id = c.customer_id
LEFT JOIN people p ON p.id = c.requested_by_person_id
LEFT JOIN service_accounts sa ON sa.id = c.requested_by_service_account_id
WHERE c.status = 'submitted'
ORDER BY c.created_at
LIMIT 50;

-- name: InboxScansReady :many
-- InboxScansReady memberi draft yang scan bertanda tangannya sudah diunggah
-- dan menunggu rilis dicatat.
SELECT d.id AS document_id, d.customer_id, n.number, r.revision, r.scan_uploaded_at, r.created_by_person_id
FROM document_revisions r
JOIN documents d ON d.id = r.document_id
JOIN document_numbers n ON n.id = d.document_number_id
WHERE r.status = 'draft' AND r.scan_uploaded_at IS NOT NULL
ORDER BY r.scan_uploaded_at
LIMIT 50;

-- name: InboxYokotenPending :many
-- InboxYokotenPending memberi yokoten yang masih punya target belum
-- diputuskan, per customer target.
SELECT y.id, y.seq, y.reason, t.customer_id,
       count(*)::int AS targets,
       (count(*) FILTER (WHERE t.decision <> 'pending'))::int AS decided
FROM yokotens y
JOIN yokoten_targets t ON t.yokoten_id = y.id
GROUP BY y.id, y.seq, y.reason, t.customer_id
HAVING count(*) FILTER (WHERE t.decision = 'pending') > 0
ORDER BY y.seq
LIMIT 50;

-- name: InboxSignaturesWaiting :many
-- InboxSignaturesWaiting memberi permintaan tanda tangan yang menunggu orang
-- ini sekarang (D-198): slotnya adalah slot belum bertanda tangan pertama.
SELECT sr.id, sr.document_id, n.number, sr.revision, c.code AS customer_code, sr.sent_at, sl.print_title
FROM signature_requests sr
JOIN signature_slots sl ON sl.request_id = sr.id
JOIN documents d ON d.id = sr.document_id
JOIN document_numbers n ON n.id = d.document_number_id
JOIN customers c ON c.id = sr.customer_id
WHERE sr.status = 'waiting' AND sl.person_id = sqlc.arg(person_id)
  AND sl.slot_order = (SELECT min(x.slot_order) FROM signature_slots x
                       WHERE x.request_id = sr.id
                         AND NOT EXISTS (SELECT 1 FROM signatures g WHERE g.request_id = x.request_id AND g.slot_order = x.slot_order))
ORDER BY sr.sent_at;

-- name: InboxSignaturesCompleted :many
-- InboxSignaturesCompleted memberi draft yang sudah lengkap ditandatangani
-- digital dan menunggu rilis dicatat.
SELECT sr.document_id, n.number, sr.revision, sr.customer_id, sr.closed_at
FROM signature_requests sr
JOIN document_revisions r ON r.id = sr.revision_id
JOIN documents d ON d.id = sr.document_id
JOIN document_numbers n ON n.id = d.document_number_id
WHERE sr.status = 'completed' AND r.status = 'draft'
ORDER BY sr.closed_at;
