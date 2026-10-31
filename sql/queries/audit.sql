-- name: LockAuditChain :exec
-- LockAuditChain memegang kunci ujung rantai audit sampai transaksi selesai,
-- supaya dua transaksi tidak membaca ujung rantai yang sama (E-06).
SELECT pg_advisory_xact_lock($1);

-- name: GetAuditChainHead :one
-- GetAuditChainHead mengambil entri terakhir rantai: nomor urut dan hash-nya.
SELECT seq, hash FROM audit_log ORDER BY seq DESC LIMIT 1;

-- name: GetAuditHash :one
-- GetAuditHash mengambil hash satu entri, untuk mencocokkan ujung rantai yang
-- dicatat di manifest backup (M1.7).
SELECT hash FROM audit_log WHERE seq = $1;

-- name: InsertAuditEntry :exec
INSERT INTO audit_log (
    seq, occurred_at, actor_type, actor_account_id, actor_person_id, request_id, source_ip,
    action, target_type, target_id, details, prev_hash, hash
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: ListAuditEntries :many
-- ListAuditEntries mengambil entri yang lebih lama dari before, terbaru dulu,
-- beserta login dan nama orang aktor (D-74) dan nama tampilan sasarannya.
SELECT l.seq, l.occurred_at, l.actor_type, l.actor_account_id, a.login AS actor_login,
       l.actor_person_id, p.name AS actor_person_name, l.request_id, l.source_ip,
       l.action, l.target_type, l.target_id,
       coalesce(ta.login, tc.code, tp.code, tg.name, tm.name, tn.number, tpr.name,
                tns.factory_code, tdt.name, tsc.code, tct.name, tcs.name, tfm.name,
                tpo.name, tar.name, tdn.number, tft.name, tptn.name, '') AS target_label,
       l.details, l.prev_hash, l.hash
FROM audit_log l
LEFT JOIN accounts a ON a.id = l.actor_account_id
LEFT JOIN people p ON p.id = l.actor_person_id
LEFT JOIN accounts ta ON l.target_type = 'account' AND ta.id = l.target_id
LEFT JOIN customers tc ON l.target_type = 'customer' AND tc.id = l.target_id
LEFT JOIN project_groups tp ON l.target_type = 'project_group' AND tp.id = l.target_id
LEFT JOIN model_groups tg ON l.target_type = 'model_group' AND tg.id = l.target_id
LEFT JOIN models tm ON l.target_type = 'model' AND tm.id = l.target_id
LEFT JOIN part_numbers tn ON l.target_type = 'part_number' AND tn.id = l.target_id
-- Proses ditampilkan dengan nama tampilan yang berlaku sekarang (D-06).
LEFT JOIN process_names tpr ON l.target_type = 'process' AND tpr.process_id = l.target_id AND tpr.kind = 'display'
LEFT JOIN number_settings tns ON l.target_type = 'number_settings' AND tns.id = l.target_id
LEFT JOIN document_types tdt ON l.target_type = 'document_type' AND tdt.id = l.target_id
LEFT JOIN special_codes tsc ON l.target_type = 'special_code' AND tsc.id = l.target_id
LEFT JOIN document_numbers tdn ON l.target_type = 'document_number' AND tdn.id = l.target_id
LEFT JOIN change_types tct ON l.target_type = 'change_type' AND tct.id = l.target_id
LEFT JOIN change_sources tcs ON l.target_type = 'change_source' AND tcs.id = l.target_id
LEFT JOIN four_m_categories tfm ON l.target_type = 'four_m_category' AND tfm.id = l.target_id
LEFT JOIN positions tpo ON l.target_type = 'position' AND tpo.id = l.target_id
LEFT JOIN approval_routes tar ON l.target_type = 'approval_route' AND tar.id = l.target_id
LEFT JOIN format_templates tft ON l.target_type = 'format_template' AND tft.id = l.target_id
-- Process template ditampilkan dengan nama tampilan prosesnya yang berlaku sekarang.
LEFT JOIN process_templates tpt ON l.target_type = 'process_template' AND tpt.id = l.target_id
LEFT JOIN process_names tptn ON tptn.process_id = tpt.process_id AND tptn.kind = 'display'
WHERE l.seq < sqlc.arg(before)
ORDER BY l.seq DESC
LIMIT sqlc.arg(max_entries);

-- name: ListAuditChain :many
-- ListAuditChain mengambil semua entri menurut urutan rantai, untuk diperiksa hash-nya.
SELECT seq, occurred_at, actor_type, actor_account_id, actor_person_id, request_id, source_ip,
       action, target_type, target_id, details, prev_hash, hash
FROM audit_log
ORDER BY seq;
