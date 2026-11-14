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
SELECT r.id, r.revision, r.status, p.name AS created_by, r.created_at, r.released_at
FROM document_revisions r JOIN people p ON p.id = r.created_by_person_id
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
