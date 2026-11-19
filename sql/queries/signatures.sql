-- Fase 3 · Tanda tangan digital (D-198, D-203, D-204).

-- name: RouteSlotCandidates :many
-- RouteSlotCandidates memberi setiap slot rute beserta pemegang jabatan yang
-- diizinkan slot itu untuk customer ini: jabatan tanpa Project Group, atau
-- Project Group customer. Slot tanpa pemegang tetap muncul dengan person_id NULL.
SELECT s.slot_order, s.stage, s.print_title,
       p.key AS position_key, p.name AS position_name,
       h.person_id, pe.name AS person_name
FROM route_slots s
LEFT JOIN route_slot_positions sp ON sp.slot_id = s.id
LEFT JOIN positions p ON p.id = sp.position_id
LEFT JOIN position_holders h ON h.position_id = sp.position_id
    AND (h.project_group_id IS NULL OR h.project_group_id = (SELECT cu.project_group_id FROM customers cu WHERE cu.id = sqlc.arg(customer_id)))
LEFT JOIN people pe ON pe.id = h.person_id
WHERE s.route_id = sqlc.arg(route_id)
ORDER BY s.slot_order, p.key, pe.name;

-- name: CustomerSignatureMode :one
SELECT signature_mode FROM customers WHERE id = $1;

-- name: SetCustomerSignatureMode :exec
UPDATE customers SET signature_mode = $2 WHERE id = $1;

-- name: CreateSignatureRequest :one
INSERT INTO signature_requests (document_id, revision_id, revision, customer_id, route_id, route_name, snapshot,
    content_sha256, release_date, note, sent_by_person_id, sent_by_account_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id;

-- name: AddSignatureSlot :exec
INSERT INTO signature_slots (request_id, slot_order, stage, print_title, position_name, person_id, person_name)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: AddSignature :exec
INSERT INTO signatures (request_id, slot_order, person_id, account_id, person_name, print_title, stage, content_sha256, source_ip)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: LockSignatureRequest :one
-- LockSignatureRequest membaca satu permintaan dan menguncinya sampai
-- transaksi selesai, supaya dua tanda tangan tidak bertabrakan.
SELECT sr.id, sr.document_id, sr.revision_id, sr.revision, sr.customer_id, sr.content_sha256, sr.status,
       sr.sent_by_person_id, sr.release_date
FROM signature_requests sr
WHERE sr.id = $1
FOR NO KEY UPDATE;

-- name: CloseSignatureRequest :exec
UPDATE signature_requests
SET status = sqlc.arg(status), closed_at = now(), closed_by_person_id = sqlc.narg(closed_by), reject_reason = sqlc.narg(reason)
WHERE id = sqlc.arg(id);

-- name: GetSignatureRequest :one
SELECT sr.id, sr.document_id, sr.revision_id, sr.revision, sr.customer_id, c.code AS customer_code, n.number,
       t.key AS document_type_key, sr.route_name, sr.content_sha256, sr.release_date, sr.note, sr.status,
       sr.sent_by_person_id, sp.name AS sent_by, sr.sent_at, sr.closed_at, cp.name AS closed_by, sr.reject_reason
FROM signature_requests sr
JOIN documents d ON d.id = sr.document_id
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_types t ON t.id = d.document_type_id
JOIN customers c ON c.id = sr.customer_id
JOIN people sp ON sp.id = sr.sent_by_person_id
LEFT JOIN people cp ON cp.id = sr.closed_by_person_id
WHERE sr.id = $1;

-- name: GetSignatureSnapshot :one
SELECT snapshot FROM signature_requests WHERE id = $1;

-- name: ListSignatureSlots :many
SELECT sl.slot_order, sl.stage, sl.print_title, sl.position_name, sl.person_id, sl.person_name,
       sg.signed_at, sg.content_sha256 AS signed_sha256
FROM signature_slots sl
LEFT JOIN signatures sg ON sg.request_id = sl.request_id AND sg.slot_order = sl.slot_order
WHERE sl.request_id = $1
ORDER BY sl.slot_order;

-- name: ListDocumentSignatureRequests :many
SELECT sr.id
FROM signature_requests sr
WHERE sr.document_id = $1
ORDER BY sr.sent_at DESC
LIMIT 20;

-- name: ActiveSignatureRequestOfDraft :one
-- ActiveSignatureRequestOfDraft memberi permintaan yang menunggu atau sudah
-- lengkap untuk draft terbuka dokumen ini.
SELECT sr.id, sr.status, sr.content_sha256
FROM signature_requests sr
JOIN document_revisions r ON r.id = sr.revision_id
WHERE sr.document_id = $1 AND r.status = 'draft' AND sr.status IN ('waiting', 'completed');

-- name: CompletedSignatureOfRevision :one
-- CompletedSignatureOfRevision memberi permintaan yang lengkap untuk satu
-- revisi, dipakai cetak dan rilis.
SELECT sr.id, sr.content_sha256, sr.release_date, sr.route_name
FROM signature_requests sr
WHERE sr.revision_id = $1 AND sr.status = 'completed';

-- name: SignatureRequestsWaitingFor :many
-- SignatureRequestsWaitingFor memberi permintaan yang menunggu tanda tangan
-- orang ini sekarang: slot belum bertanda tangan dengan urutan terkecil
-- adalah slotnya (D-49).
SELECT sr.id
FROM signature_requests sr
JOIN signature_slots sl ON sl.request_id = sr.id
WHERE sr.status = 'waiting' AND sl.person_id = sqlc.arg(person_id)
  AND sl.slot_order = (SELECT min(x.slot_order) FROM signature_slots x
                       WHERE x.request_id = sr.id
                         AND NOT EXISTS (SELECT 1 FROM signatures g WHERE g.request_id = x.request_id AND g.slot_order = x.slot_order))
ORDER BY sr.sent_at;

-- name: SignatureRequestsSentBy :many
SELECT sr.id
FROM signature_requests sr
WHERE sr.sent_by_person_id = $1
ORDER BY sr.sent_at DESC
LIMIT 50;

-- name: SignatureRequestsCompleted :many
-- SignatureRequestsCompleted memberi permintaan lengkap yang draftnya belum
-- terbit: menunggu rilis dicatat.
SELECT sr.id
FROM signature_requests sr
JOIN document_revisions r ON r.id = sr.revision_id
WHERE sr.status = 'completed' AND r.status = 'draft'
ORDER BY sr.closed_at;

-- name: NextSignerOf :one
-- NextSignerOf memberi slot berikutnya yang belum bertanda tangan.
SELECT sl.slot_order, sl.person_id, sl.person_name, sl.print_title
FROM signature_slots sl
WHERE sl.request_id = $1
  AND NOT EXISTS (SELECT 1 FROM signatures g WHERE g.request_id = sl.request_id AND g.slot_order = sl.slot_order)
ORDER BY sl.slot_order
LIMIT 1;

-- name: AccountOfPerson :one
-- AccountOfPerson memberi akun kerja aktif seseorang.
SELECT id FROM accounts WHERE person_id = $1 AND kind = 'work' AND disabled_at IS NULL;
