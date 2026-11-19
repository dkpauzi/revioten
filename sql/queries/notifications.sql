-- D-194 · Email notifikasi. Lihat migrasi 00046.

-- name: ListNotificationEvents :many
SELECT key, enabled FROM notification_events ORDER BY position;

-- name: NotificationEventEnabled :one
SELECT enabled FROM notification_events WHERE key = $1;

-- name: SetNotificationEvent :execrows
UPDATE notification_events SET enabled = $2 WHERE key = $1;

-- name: EnqueueNotification :exec
-- EnqueueNotification menulis satu email ke kotak keluar. Penerima tanpa
-- alamat email langsung berstatus skipped, supaya admin melihat siapa yang
-- belum punya alamat.
INSERT INTO notification_outbox (event, recipient_person_id, recipient_email, subject, body, status, reason)
SELECT sqlc.arg(event), p.id, p.email, sqlc.arg(subject), sqlc.arg(body),
       CASE WHEN p.email IS NULL THEN 'skipped' ELSE 'pending' END,
       CASE WHEN p.email IS NULL THEN 'no_email' END
FROM people p WHERE p.id = sqlc.arg(person_id);

-- name: ListOutbox :many
SELECT o.id, o.event, p.name AS recipient, o.recipient_email, o.subject, o.status, o.reason, o.attempts, o.created_at, o.sent_at
FROM notification_outbox o JOIN people p ON p.id = o.recipient_person_id
ORDER BY o.created_at DESC
LIMIT 100;

-- name: ClaimOutbox :many
-- ClaimOutbox mengambil email yang siap dikirim. SKIP LOCKED membuat dua
-- pekerja tidak mengambil baris yang sama.
UPDATE notification_outbox SET status = 'sending'
WHERE id IN (SELECT id FROM notification_outbox WHERE status = 'pending' AND next_attempt_at <= now()
             ORDER BY next_attempt_at LIMIT sqlc.arg(batch) FOR UPDATE SKIP LOCKED)
RETURNING id, recipient_email, subject, body, attempts;

-- name: MarkOutboxSent :exec
UPDATE notification_outbox SET status = 'sent', sent_at = now(), attempts = attempts + 1, reason = NULL WHERE id = $1;

-- name: MarkOutboxRetry :exec
-- MarkOutboxRetry menjadwalkan percobaan berikutnya, atau menandai gagal
-- bila percobaan sudah habis.
UPDATE notification_outbox
SET attempts = attempts + 1, reason = sqlc.arg(reason),
    status = CASE WHEN attempts + 1 >= sqlc.arg(max_attempts)::int THEN 'failed' ELSE 'pending' END,
    next_attempt_at = now() + make_interval(mins => (2 ^ (attempts + 1))::int)
WHERE id = sqlc.arg(id);

-- name: MarkOutboxSkipped :exec
UPDATE notification_outbox SET status = 'skipped', reason = $2 WHERE id = $1;

-- name: RequeueSendingOutbox :exec
-- RequeueSendingOutbox mengembalikan email yang tertahan berstatus sending
-- (server mati di tengah pengiriman) ke antrean.
UPDATE notification_outbox SET status = 'pending' WHERE status = 'sending';

-- name: GetPersonEmail :one
SELECT email FROM people WHERE id = $1;

-- name: SetPersonEmail :exec
UPDATE people SET email = $2 WHERE id = $1;

-- name: GetAccountPerson :one
SELECT a.person_id, a.login, p.name, p.email FROM accounts a JOIN people p ON p.id = a.person_id WHERE a.id = $1;

-- Penerima. Hanya orang dengan akun aktif.

-- name: PeopleWithPermission :many
-- PeopleWithPermission memberi orang yang memegang satu hak untuk customer
-- itu: lewat peran bercakupan semua customer, atau bercakupan Project Group
-- sendiri bila orangnya Project Leader atau Engineer Project Group customer.
-- customer_id NULL berarti hak tanpa customer.
SELECT DISTINCT a.person_id
FROM accounts a
JOIN account_roles ar ON ar.account_id = a.id
JOIN role_permissions rp ON rp.role_id = ar.role_id
WHERE a.disabled_at IS NULL AND rp.permission_code = sqlc.arg(permission)
  AND (ar.scope = 'all' OR (sqlc.narg(customer_id)::uuid IS NOT NULL AND EXISTS (
        SELECT 1 FROM position_holders h JOIN positions p ON p.id = h.position_id
        JOIN customers c ON c.project_group_id = h.project_group_id
        WHERE h.person_id = a.person_id AND c.id = sqlc.narg(customer_id)::uuid
          AND p.key IN ('project_leader', 'project_engineer'))));

-- name: ChangeVerifiers :many
-- ChangeVerifiers memberi orang yang boleh memverifikasi perubahan customer
-- itu: pemegang V-1 yang juga memegang jabatan verifikator (D-51, D-54).
SELECT DISTINCT h.person_id
FROM position_holders h
JOIN positions p ON p.id = h.position_id
LEFT JOIN customers c ON c.project_group_id = h.project_group_id AND c.id = sqlc.arg(customer_id)
JOIN accounts a ON a.person_id = h.person_id AND a.disabled_at IS NULL
JOIN account_roles ar ON ar.account_id = a.id
JOIN role_permissions rp ON rp.role_id = ar.role_id AND rp.permission_code = 'V-1'
-- Project Leader Project Group lain ikut bila hak V-1-nya tidak dibatasi (D-206).
WHERE p.key = 'apqp_ppap_controller' OR (p.key IN ('project_leader', 'project_engineer') AND (c.id IS NOT NULL OR ar.scope = 'all'));

-- name: GroupLeadersOfCustomer :many
-- GroupLeadersOfCustomer memberi Project Leader dan Project Engineer
-- Project Group customer itu.
SELECT DISTINCT h.person_id
FROM position_holders h
JOIN positions p ON p.id = h.position_id
JOIN customers c ON c.project_group_id = h.project_group_id
WHERE c.id = $1 AND p.key IN ('project_leader', 'project_engineer');

-- name: AdminPeople :many
SELECT DISTINCT person_id FROM accounts WHERE kind = 'admin' AND disabled_at IS NULL;

-- Bahan isi email.

-- name: ChangeForNotice :one
SELECT c.id, c.seq, c.customer_id, cu.code AS customer_code, c.reason, c.status, c.requested_by_person_id,
       coalesce(p.name, sa.name)::text AS requested_by, c.verification_note,
       coalesce((SELECT string_agg(n.number, ', ' ORDER BY n.number) FROM change_documents cd
                 JOIN documents d ON d.id = cd.document_id JOIN document_numbers n ON n.id = d.document_number_id
                 WHERE cd.change_id = c.id), '')::text AS documents
FROM changes c
JOIN customers cu ON cu.id = c.customer_id
LEFT JOIN people p ON p.id = c.requested_by_person_id
LEFT JOIN service_accounts sa ON sa.id = c.requested_by_service_account_id
WHERE c.id = $1;

-- name: DocumentForNotice :one
SELECT d.id, n.number, d.customer_id, cu.code AS customer_code, d.revision, d.created_by_person_id
FROM documents d
JOIN document_numbers n ON n.id = d.document_number_id
JOIN customers cu ON cu.id = d.customer_id
WHERE d.id = $1;

-- name: RequestersOfRevision :many
-- RequestersOfRevision memberi pemohon (orang) perubahan yang masuk ke satu
-- revisi dokumen.
SELECT DISTINCT c.requested_by_person_id::uuid AS person_id
FROM change_documents cd
JOIN changes c ON c.id = cd.change_id
JOIN document_revisions r ON r.id = cd.draft_revision_id
WHERE cd.document_id = $1 AND r.revision = $2 AND c.requested_by_person_id IS NOT NULL;

-- name: YokotenForNotice :one
SELECT y.id, y.seq, y.reason, src.seq AS source_seq,
       coalesce((SELECT array_agg(DISTINCT t.customer_id::text) FROM yokoten_targets t WHERE t.yokoten_id = y.id), '{}')::text[] AS customer_ids,
       (SELECT count(*) FROM yokoten_targets t WHERE t.yokoten_id = y.id)::int AS targets
FROM yokotens y JOIN changes src ON src.id = y.source_change_id
WHERE y.id = $1;

-- name: PendingFormApprovers :many
-- PendingFormApprovers memberi pemegang V-3 yang belum menyetujui versi draft
-- target itu (form set atau format), selain penyusun drafnya.
SELECT DISTINCT a.person_id
FROM accounts a
JOIN account_roles ar ON ar.account_id = a.id
JOIN role_permissions rp ON rp.role_id = ar.role_id AND rp.permission_code = 'V-3'
WHERE a.disabled_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM form_approvals fa WHERE fa.target_id = sqlc.arg(version_id) AND fa.person_id = a.person_id)
  AND a.person_id IS DISTINCT FROM sqlc.narg(preparer_id)::uuid;

-- name: DraftFormSetVersion :one
SELECT v.id, v.version, s.name, v.prepared_by_person_id
FROM form_set_versions v JOIN form_sets s ON s.id = v.form_set_id
WHERE v.form_set_id = $1 AND v.status = 'draft';

-- name: DraftFormatVersion :one
SELECT v.id, v.version, t.name, v.changed_by_person_id
FROM format_template_versions v JOIN format_templates t ON t.id = v.template_id
WHERE v.template_id = $1 AND v.status = 'draft';

-- name: ExpiringTokens :many
-- ExpiringTokens memberi token aktif yang kedaluwarsa dalam 7 hari dan
-- belum diberitahukan.
SELECT t.id, t.prefix, t.expires_at, sa.name
FROM service_tokens t JOIN service_accounts sa ON sa.id = t.service_account_id
WHERE t.revoked_at IS NULL AND sa.disabled_at IS NULL AND t.expiry_notified_at IS NULL
  AND t.expires_at > now() AND t.expires_at <= now() + interval '7 days';

-- name: MarkTokenExpiryNotified :exec
UPDATE service_tokens SET expiry_notified_at = now() WHERE id = $1;
