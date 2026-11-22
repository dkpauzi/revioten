-- Tahap 7 · Migrasi dan bukti nilai. Lihat migrasi 00041.

-- name: ListMigratableNumbers :many
-- ListMigratableNumbers memberi nomor PFD, PFMEA, dan CP satu customer yang
-- belum punya dokumen dan masih hidup (M7.1): nomor lama hasil impor (M2.5)
-- atau nomor Reserved.
SELECT n.id, n.number, n.status, t.name AS document_type_name
FROM document_numbers n
JOIN document_types t ON t.id = n.document_type_id
WHERE n.customer_id = $1 AND n.status IN ('reserved', 'in_use', 'released')
  AND t.key IN ('pfd', 'pfmea', 'cp')
  AND NOT EXISTS (SELECT 1 FROM documents d WHERE d.document_number_id = n.id)
ORDER BY n.number;

-- name: CreateDocumentBaseline :exec
INSERT INTO document_baselines (document_id, revision, effective_date, prepared_by, approved_by, created_by_person_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetDocumentBaseline :one
SELECT b.document_id, b.revision, b.effective_date, b.prepared_by, b.approved_by, b.created_by_person_id, cp.name AS created_by, b.created_at,
       vp.name AS verified_by, b.verified_at, b.effort_minutes
FROM document_baselines b
JOIN people cp ON cp.id = b.created_by_person_id
LEFT JOIN people vp ON vp.id = b.verified_by_person_id
WHERE b.document_id = $1;

-- name: ListHistoryEntries :many
SELECT id, position, revision_label, entry_date, description
FROM document_history_entries WHERE document_id = $1 ORDER BY position;

-- name: DeleteHistoryEntries :exec
DELETE FROM document_history_entries WHERE document_id = $1;

-- name: AddHistoryEntry :exec
INSERT INTO document_history_entries (document_id, position, revision_label, entry_date, description, created_by_person_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: VerifyDocumentBaseline :execrows
UPDATE document_baselines SET verified_by_person_id = $2, verified_at = now(), effort_minutes = $3
WHERE document_id = $1 AND verified_at IS NULL;

-- name: ReleaseMigratedRevision :execrows
-- ReleaseMigratedRevision menerbitkan revisi baseline sesudah dicocokkan
-- dengan hardcopy: Released (migrated). Tanggal rilis adalah tanggal berlaku
-- di hardcopy (R-MIG-06, D-59).
UPDATE document_revisions
SET status = 'released', released_at = now(), release_date = sqlc.arg(release_date), migrated = true,
    snapshot = sqlc.arg(snapshot), snapshot_sha256 = sqlc.arg(sha256),
    released_by_person_id = sqlc.arg(person_id), released_by_account_id = sqlc.arg(account_id)
WHERE id = sqlc.arg(id) AND status = 'draft';

-- name: LatestPublishedProcessVersion :one
-- LatestPublishedProcessVersion memberi versi terbit terakhir process
-- template satu proses, untuk usulan penautan (M7.4).
SELECT id FROM process_template_versions
WHERE process_id = $1 AND status = 'published'
ORDER BY version DESC LIMIT 1;

-- name: LinkDocumentRow :execrows
-- LinkDocumentRow menautkan baris yang belum tertaut ke satu poin template
-- (R-MIG-05). Hasil 0 baris berarti baris itu sudah tertaut.
UPDATE document_rows
SET template_point_id = $3, template_version_id = $4, version = version + 1, updated_at = now()
WHERE id = $1 AND document_id = $2 AND template_point_id IS NULL;

-- M7.7: dasbor ukuran. Semua angka dihitung dari data, tidak ada yang diketik.

-- name: MetricRows :one
-- MetricRows menghitung baris dokumen menurut tautannya ke template.
SELECT count(*)::int AS total,
       (count(*) FILTER (WHERE r.template_point_id IS NOT NULL AND r.detached_at IS NULL))::int AS linked,
       (count(*) FILTER (WHERE r.detached_at IS NOT NULL))::int AS detached
FROM document_rows r JOIN documents d ON d.id = r.document_id
WHERE (sqlc.narg(customer_id)::uuid IS NULL OR d.customer_id = sqlc.narg(customer_id)::uuid);

-- name: MetricDocuments :many
-- MetricDocuments menghitung dokumen per jenis dan status. Status
-- released_migrated berarti revisi yang berlaku adalah baseline migrasi.
SELECT t.key AS document_type_key, t.name AS document_type_name,
       (CASE WHEN d.status = 'released' AND coalesce(rr.migrated, false) THEN 'released_migrated' ELSE d.status END)::text AS status,
       count(*)::int AS count
FROM documents d
JOIN document_types t ON t.id = d.document_type_id
LEFT JOIN document_revisions rr ON rr.document_id = d.id AND rr.revision = d.released_revision
WHERE (sqlc.narg(customer_id)::uuid IS NULL OR d.customer_id = sqlc.narg(customer_id)::uuid)
GROUP BY t.position, t.key, t.name, 3
ORDER BY t.position, 3;

-- name: MetricYokoten :many
SELECT yt.decision, count(*)::int AS count
FROM yokoten_targets yt
WHERE (sqlc.narg(customer_id)::uuid IS NULL OR yt.customer_id = sqlc.narg(customer_id)::uuid)
GROUP BY yt.decision ORDER BY yt.decision;

-- name: MetricRevisionTime :one
-- MetricRevisionTime mengukur waktu satu revisi: dari draft dibuka oleh
-- perubahan yang diterima sampai rilisnya dicatat. Revisi awal dan baseline
-- migrasi tidak dihitung.
SELECT count(*)::int AS revisions,
       (coalesce(avg(extract(epoch FROM r.released_at - r.created_at)), 0) / 3600)::float8 AS average_hours,
       (coalesce(percentile_cont(0.5) WITHIN GROUP (ORDER BY extract(epoch FROM r.released_at - r.created_at)), 0) / 3600)::float8 AS median_hours
FROM document_revisions r JOIN documents d ON d.id = r.document_id
WHERE r.status = 'released' AND NOT r.migrated
  AND EXISTS (SELECT 1 FROM change_documents cd WHERE cd.draft_revision_id = r.id)
  AND (sqlc.narg(customer_id)::uuid IS NULL OR d.customer_id = sqlc.narg(customer_id)::uuid);

-- name: MetricMigration :one
-- MetricMigration menghitung dokumen migrasi dan menit usahanya (M7.6).
SELECT count(*)::int AS documents,
       count(b.verified_at)::int AS verified,
       count(b.effort_minutes)::int AS measured,
       coalesce(sum(b.effort_minutes), 0)::int AS total_minutes
FROM document_baselines b JOIN documents d ON d.id = b.document_id
WHERE (sqlc.narg(customer_id)::uuid IS NULL OR d.customer_id = sqlc.narg(customer_id)::uuid);

-- name: MigrationReadiness :one
-- MigrationReadiness memberi prasyarat menu migrasi satu customer (D-215):
-- saklar mode migrasi, jumlah model customer, dan jumlah proses di master.
SELECT (SELECT s.migration_mode FROM system_settings s LIMIT 1)::boolean AS migration_mode,
       (SELECT count(*) FROM models m WHERE m.customer_id = $1 AND m.eol_at IS NULL) AS models,
       (SELECT count(*) FROM processes) AS processes;

-- name: ListLegacyDocuments :many
-- ListLegacyDocuments memberi papan status dokumen lama satu customer
-- (D-215): nomor, jenis, revisi baseline, jumlah baris, pemasuk, dan
-- pemeriksa hardcopy.
SELECT d.id, n.number, t.key AS document_type_key, t.name AS document_type_name, b.revision,
       (SELECT count(*) FROM document_rows r WHERE r.document_id = d.id) AS rows,
       cp.name AS entered_by, b.created_at AS entered_at, vp.name AS verified_by, b.verified_at
FROM documents d
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_types t ON t.id = d.document_type_id
JOIN document_baselines b ON b.document_id = d.id
JOIN people cp ON cp.id = b.created_by_person_id
LEFT JOIN people vp ON vp.id = b.verified_by_person_id
WHERE d.customer_id = $1 AND d.origin = 'migrated'
ORDER BY n.sequence, t.position, n.number;
