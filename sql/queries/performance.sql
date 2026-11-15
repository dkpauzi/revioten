-- Laporan kinerja PPAP Engineer (D-197). Query memberi kejadian mentah dalam
-- satu periode [from, to); angka dihitung di internal/metrics.

-- name: PerformanceChanges :many
-- PerformanceChanges memberi perubahan yang diajukan atau diverifikasi dalam
-- periode, beserta pemicu dan pemohonnya.
SELECT c.id, c.seq, c.customer_id, c.mechanism, c.status, c.created_at, c.verified_at,
       coalesce(src.name, '')::text AS trigger_name,
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
-- PerformanceYokotens memberi yokoten yang dibuat dalam periode.
SELECT y.id, y.seq, y.created_at, y.created_by_person_id, p.name AS created_by,
       (SELECT count(*) FROM yokoten_targets t WHERE t.yokoten_id = y.id
          AND (sqlc.narg(customer_id)::uuid IS NULL OR t.customer_id = sqlc.narg(customer_id)::uuid))::int AS targets
FROM yokotens y
JOIN people p ON p.id = y.created_by_person_id
JOIN changes src ON src.id = y.source_change_id
WHERE y.created_at >= sqlc.arg(from_at) AND y.created_at < sqlc.arg(to_at)
  AND (sqlc.narg(customer_id)::uuid IS NULL OR src.customer_id = sqlc.narg(customer_id)::uuid
       OR EXISTS (SELECT 1 FROM yokoten_targets t WHERE t.yokoten_id = y.id AND t.customer_id = sqlc.narg(customer_id)::uuid))
ORDER BY y.seq;

-- name: PerformanceReleases :many
-- PerformanceReleases memberi revisi yang terbit dalam periode lewat
-- perubahan yang diterima (revisi awal dan dokumen lama tidak ikut), beserta
-- waktu di setiap tahap: perubahan pertama diterima, draft dibuka, cetak untuk
-- tanda tangan terakhir sebelum scan diunggah, scan diunggah, dan rilis.
SELECT r.id, n.number, t.key AS type_key, r.revision, d.customer_id,
       r.created_at, r.released_at, r.scan_uploaded_at,
       -- Draft dibuka saat perubahan pertama diterima, jadi waktu itu menjadi
       -- cadangan bila waktu verifikasinya tidak ada.
       coalesce((SELECT min(c.verified_at) FROM change_documents cd JOIN changes c ON c.id = cd.change_id
                 WHERE cd.draft_revision_id = r.id AND c.status = 'accepted'), r.created_at)::timestamptz AS accepted_at,
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
