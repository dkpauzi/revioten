-- M5.4: revisi dokumen dan draft (R-CHG-03, R-REV-01).

-- name: CreateDocumentRevision :one
INSERT INTO document_revisions (document_id, revision, created_by_person_id)
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetOpenDraftRevision :one
-- GetOpenDraftRevision memberi draft terbuka satu dokumen, bila ada.
SELECT id, revision FROM document_revisions WHERE document_id = $1 AND status = 'draft';

-- name: MaxDocumentRevision :one
SELECT max(revision)::smallint AS revision FROM document_revisions WHERE document_id = $1;

-- name: OpenDocumentDraft :exec
-- OpenDocumentDraft menjadikan revisi baru sebagai revisi yang dikerjakan.
-- released_revision tidak berubah: revisi yang berlaku tetap yang terbit.
UPDATE documents SET revision = $2, status = 'draft', updated_at = now() WHERE id = $1;

-- name: ListDocumentRevisions :many
SELECT r.id, r.revision, r.status, p.name AS created_by, r.created_at, r.released_at,
       r.snapshot_sha256, rp.name AS released_by, r.release_date, r.scan_file_name, r.migrated, r.ready_at
FROM document_revisions r JOIN people p ON p.id = r.created_by_person_id
LEFT JOIN people rp ON rp.id = r.released_by_person_id
WHERE r.document_id = $1
ORDER BY r.revision DESC;

-- name: LinkChangeDraft :exec
UPDATE change_documents SET draft_revision_id = $3, merged = $4 WHERE change_id = $1 AND document_id = $2;

-- name: ListRevisionChanges :many
-- ListRevisionChanges memberi perubahan yang masuk ke satu revisi.
SELECT c.id, c.seq, cd.merged
FROM change_documents cd JOIN changes c ON c.id = cd.change_id
WHERE cd.draft_revision_id = $1
ORDER BY c.seq;

-- M5.5: rilis dengan snapshot (R-REV-02).

-- name: ReleaseDocumentRevision :execrows
UPDATE document_revisions
SET status = 'released', released_at = now(), release_date = coalesce(release_date, (now() AT TIME ZONE 'Asia/Jakarta')::date),
    snapshot = sqlc.arg(snapshot), snapshot_sha256 = sqlc.arg(sha256),
    released_by_person_id = sqlc.arg(person_id), released_by_account_id = sqlc.arg(account_id)
WHERE id = sqlc.arg(id) AND status = 'draft';

-- name: MarkDocumentReleased :exec
UPDATE documents SET status = 'released', released_revision = $2, updated_at = now() WHERE id = $1;

-- name: PreparedDraft :one
-- PreparedDraft melaporkan apakah orang itu ikut menyusun draft yang
-- terbuka: ada entri audit perubahan isi dokumen olehnya sesudah rilis
-- terakhir dokumen itu (R-APR-02, D-166). Membuka draft, mencetak, dan
-- mengunggah scan tidak dihitung sebagai menyusun.
SELECT EXISTS (
    SELECT 1 FROM audit_log a
    WHERE a.target_id = sqlc.arg(document_id) AND a.actor_person_id = sqlc.arg(person_id)
      AND a.action LIKE 'document.%'
      -- Bukan perubahan isi: membuka draft, rilis, cetak, unggah scan, dan mode tautan template.
      AND a.action NOT IN ('document.draft_opened', 'document.released', 'document.printed', 'document.scan_uploaded', 'document.template_mode_changed', 'document.migration_verified')
      AND a.seq > coalesce((SELECT max(r.seq) FROM audit_log r WHERE r.target_id = sqlc.arg(document_id) AND r.action IN ('document.released', 'document.migration_verified')), 0)
);

-- name: GetDocumentRevision :one
SELECT r.id, r.revision, r.status, r.released_at, r.snapshot, r.snapshot_sha256
FROM document_revisions r WHERE r.document_id = $1 AND r.revision = $2;

-- name: MarkRevisionReady :execrows
-- MarkRevisionReady memberi tanda "siap cetak" pada draft terbuka satu
-- dokumen (D-202). Menandai lagi memperbarui waktunya.
UPDATE document_revisions SET ready_at = now(), ready_by_person_id = $2
WHERE document_id = $1 AND status = 'draft';
