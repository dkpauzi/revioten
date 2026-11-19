-- Laporan kinerja PPAP Engineer (D-197, D-202). Query memberi kejadian mentah
-- dalam satu rentang [from, to); angka dihitung di internal/metrics.

-- name: PerformanceChanges :many
-- PerformanceChanges memberi perubahan yang diajukan atau diverifikasi dalam
-- rentang, beserta pemicu (dan kelompok laporannya) dan pemohonnya.
SELECT c.id, c.seq, c.customer_id, c.mechanism, c.status, c.created_at, c.verified_at,
       coalesce(src.name, '')::text AS trigger_name, coalesce(src.report_key, '')::text AS report_key,
       c.requested_by_person_id, coalesce(p.name, sa.name, '')::text AS requested_by
FROM changes c
LEFT JOIN change_sources src ON src.id = c.trigger_source_id
LEFT JOIN people p ON p.id = c.requested_by_person_id
LEFT JOIN service_accounts sa ON sa.id = c.requested_by_service_account_id
WHERE ((c.created_at >= sqlc.arg(from_at) AND c.created_at < sqlc.arg(to_at))
    OR (c.verified_at >= sqlc.arg(from_at) AND c.verified_at < sqlc.arg(to_at)))
  AND (sqlc.narg(customer_id)::uuid IS NULL OR c.customer_id = sqlc.narg(customer_id)::uuid)
ORDER BY c.seq;

-- name: PerformanceYokotens :many
-- PerformanceYokotens memberi yokoten yang dibuat dalam rentang, beserta
-- keputusan targetnya: diterapkan (apply atau adapt), tidak berlaku, dan
-- belum diputuskan.
SELECT y.id, y.seq, y.reason, y.created_at, y.created_by_person_id, p.name AS created_by,
       count(t.id)::int AS targets,
       (count(t.id) FILTER (WHERE t.decision IN ('apply', 'adapt')))::int AS applied,
       (count(t.id) FILTER (WHERE t.decision = 'not_applicable'))::int AS not_applicable,
       (count(t.id) FILTER (WHERE t.decision = 'pending'))::int AS pending,
       coalesce(max(t.decided_at), y.created_at)::timestamptz AS last_decided_at
FROM yokotens y
JOIN people p ON p.id = y.created_by_person_id
JOIN changes src ON src.id = y.source_change_id
LEFT JOIN yokoten_targets t ON t.yokoten_id = y.id
     AND (sqlc.narg(customer_id)::uuid IS NULL OR t.customer_id = sqlc.narg(customer_id)::uuid)
WHERE y.created_at >= sqlc.arg(from_at) AND y.created_at < sqlc.arg(to_at)
  AND (sqlc.narg(customer_id)::uuid IS NULL OR src.customer_id = sqlc.narg(customer_id)::uuid
       OR EXISTS (SELECT 1 FROM yokoten_targets t2 WHERE t2.yokoten_id = y.id AND t2.customer_id = sqlc.narg(customer_id)::uuid))
GROUP BY y.id, y.seq, y.reason, y.created_at, y.created_by_person_id, p.name
ORDER BY y.seq;

-- name: PerformanceReleases :many
-- PerformanceReleases memberi revisi yang terbit dalam rentang lewat perubahan
-- yang diterima (revisi awal dan dokumen lama tidak ikut), beserta waktu
-- setiap tahap: perubahan pertama diterima, edit pertama sesudah draft dibuka
-- (dari audit log), tanda siap cetak, cetak untuk tanda tangan terakhir
-- sebelum scan diunggah, scan diunggah, dan rilis. Pemicu revisi adalah
-- pemicu perubahan pertama yang diterima ke draft itu.
SELECT r.id, n.number, t.key AS type_key, r.revision, d.customer_id,
       r.created_at, r.released_at, r.scan_uploaded_at, r.ready_at,
       -- Draft dibuka saat perubahan pertama diterima, jadi waktu itu menjadi
       -- cadangan bila waktu verifikasinya tidak ada.
       coalesce((SELECT min(c.verified_at) FROM change_documents cd JOIN changes c ON c.id = cd.change_id
                 WHERE cd.draft_revision_id = r.id AND c.status = 'accepted'), r.created_at)::timestamptz AS accepted_at,
       coalesce((SELECT min(a.occurred_at) FROM audit_log a
                 WHERE a.target_id = r.document_id AND a.occurred_at >= r.created_at
                   AND a.action IN ('document.row_added', 'document.rows_pasted', 'document.row_changed', 'document.row_removed', 'document.row_detached',
                                    'document.header_changed', 'document.composed', 'document.copied', 'document.rows_reordered',
                                    'document.references_filled', 'document.section_added', 'document.section_changed')),
                r.created_at)::timestamptz AS first_edit_at,
       coalesce((SELECT src.name FROM change_documents cd JOIN changes c ON c.id = cd.change_id
                 LEFT JOIN change_sources src ON src.id = c.trigger_source_id
                 WHERE cd.draft_revision_id = r.id AND c.status = 'accepted' ORDER BY c.verified_at LIMIT 1), '')::text AS trigger_name,
       coalesce((SELECT src.report_key FROM change_documents cd JOIN changes c ON c.id = cd.change_id
                 LEFT JOIN change_sources src ON src.id = c.trigger_source_id
                 WHERE cd.draft_revision_id = r.id AND c.status = 'accepted' ORDER BY c.verified_at LIMIT 1), '')::text AS report_key,
       coalesce((SELECT c.mechanism FROM change_documents cd JOIN changes c ON c.id = cd.change_id
                 WHERE cd.draft_revision_id = r.id AND c.status = 'accepted' ORDER BY c.verified_at LIMIT 1), '')::text AS mechanism,
       pl.printed_at AS printed_at, pl.printed_by_person_id AS printed_by_person_id, pp.name AS printed_by
FROM document_revisions r
JOIN documents d ON d.id = r.document_id
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_types t ON t.id = d.document_type_id
LEFT JOIN print_logs pl ON pl.id = (
    SELECT l.id FROM print_logs l
    WHERE l.revision_id = r.id AND l.purpose = 'signature' AND (r.scan_uploaded_at IS NULL OR l.printed_at <= r.scan_uploaded_at)
    ORDER BY l.printed_at DESC LIMIT 1)
LEFT JOIN people pp ON pp.id = pl.printed_by_person_id
WHERE r.status = 'released' AND NOT r.migrated
  AND r.released_at >= sqlc.arg(from_at) AND r.released_at < sqlc.arg(to_at)
  AND EXISTS (SELECT 1 FROM change_documents cd WHERE cd.draft_revision_id = r.id)
  AND (sqlc.narg(customer_id)::uuid IS NULL OR d.customer_id = sqlc.narg(customer_id)::uuid)
ORDER BY r.released_at;

-- name: PerformanceOpen :many
-- PerformanceOpen memberi dokumen dari perubahan yang sudah diterima sebelum
-- at tetapi draft revisinya belum terbit pada at: pekerjaan yang masih
-- terbuka di akhir periode, per pemohon.
SELECT c.requested_by_person_id, coalesce(p.name, sa.name, '')::text AS requested_by, count(*)::int AS open
FROM change_documents cd
JOIN changes c ON c.id = cd.change_id
JOIN document_revisions r ON r.id = cd.draft_revision_id
LEFT JOIN people p ON p.id = c.requested_by_person_id
LEFT JOIN service_accounts sa ON sa.id = c.requested_by_service_account_id
WHERE c.status = 'accepted' AND c.verified_at < sqlc.arg(at)
  AND (r.released_at IS NULL OR r.released_at >= sqlc.arg(at))
  AND (sqlc.narg(customer_id)::uuid IS NULL OR c.customer_id = sqlc.narg(customer_id)::uuid)
GROUP BY c.requested_by_person_id, 2;
