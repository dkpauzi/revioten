-- Review berkala 3 tahun (D-138, D-196).

-- name: ReviewSets :many
-- ReviewSets memberi setiap set dokumen yang punya revisi terbit: PFD,
-- PFMEA, dan CP satu customer dengan nomor berjalan yang sama. Nomor Void
-- dan Obsolete tidak ikut. Tanggal rilis terakhir memakai tanggal di
-- dokumen bertanda tangan (D-59); untuk dokumen lama, tanggal berlaku di
-- hardcopy (M7.1). Row-level security menyaring dokumen menurut akun.
WITH docs AS (
    SELECT d.id, d.customer_id, n.customer_code, n.sequence, n.number, t.key AS type_key, t.position AS type_position,
           d.released_revision, d.status,
           (SELECT max(CASE WHEN r.migrated THEN coalesce(b.effective_date, r.release_date, (r.released_at AT TIME ZONE 'Asia/Jakarta')::date)
                            ELSE coalesce(r.release_date, (r.released_at AT TIME ZONE 'Asia/Jakarta')::date) END)
            FROM document_revisions r
            LEFT JOIN document_baselines b ON b.document_id = r.document_id AND b.revision = r.revision
            WHERE r.document_id = d.id AND r.status = 'released') AS last_release
    FROM documents d
    JOIN document_numbers n ON n.id = d.document_number_id
    JOIN document_types t ON t.id = d.document_type_id
    WHERE d.released_revision IS NOT NULL AND n.status NOT IN ('void', 'obsolete')
      AND (sqlc.narg(customer_id)::uuid IS NULL OR d.customer_id = sqlc.narg(customer_id)::uuid)
      AND (sqlc.narg(document_id)::uuid IS NULL OR (n.customer_code, n.sequence) = (
            SELECT n2.customer_code, n2.sequence FROM documents d2 JOIN document_numbers n2 ON n2.id = d2.document_number_id
            WHERE d2.id = sqlc.narg(document_id)::uuid))
)
SELECT docs.customer_id, cu.code AS customer_current_code, docs.customer_code, docs.sequence,
       max(docs.last_release)::date AS last_release,
       (SELECT max(pr.reviewed_on) FROM periodic_reviews pr
        WHERE pr.customer_code = docs.customer_code AND pr.sequence = docs.sequence AND pr.status = 'confirmed')::date AS last_review,
       coalesce((SELECT pr.id::text FROM periodic_reviews pr
        WHERE pr.customer_code = docs.customer_code AND pr.sequence = docs.sequence AND pr.status = 'submitted'), '')::text AS open_review_id,
       array_agg(docs.id::text ORDER BY docs.type_position, docs.number)::text[] AS document_ids,
       array_agg(docs.number ORDER BY docs.type_position, docs.number)::text[] AS numbers,
       array_agg(docs.type_key ORDER BY docs.type_position, docs.number)::text[] AS type_keys,
       array_agg(docs.released_revision::int ORDER BY docs.type_position, docs.number)::int[] AS revisions,
       -- Tanggal rilis terakhir per dokumen, urutan sama dengan document_ids (D-202).
       array_agg(docs.last_release ORDER BY docs.type_position, docs.number)::date[] AS release_dates,
       bool_or(docs.status = 'draft') AS draft_open,
       coalesce((SELECT array_agg(DISTINCT m.name ORDER BY m.name) FROM document_models dm JOIN models m ON m.id = dm.model_id
                 WHERE dm.document_id = ANY (array_agg(docs.id))), '{}')::text[] AS models,
       -- Set EOL: semua model dokumennya sudah ditandai EOL (D-200).
       coalesce((SELECT bool_and(m.eol_at IS NOT NULL) FROM document_models dm JOIN models m ON m.id = dm.model_id
                 WHERE dm.document_id = ANY (array_agg(docs.id))), false)::boolean AS eol,
       -- Engineer set: orang yang membuat dokumen pertama set itu (ASUMSI E-132).
       coalesce((SELECT p.name FROM documents dx JOIN people p ON p.id = dx.created_by_person_id
                 WHERE dx.id = ANY (array_agg(docs.id)) ORDER BY dx.created_at LIMIT 1), '')::text AS engineer
FROM docs JOIN customers cu ON cu.id = docs.customer_id
GROUP BY docs.customer_id, cu.code, docs.customer_code, docs.sequence
ORDER BY 5, docs.customer_code, docs.sequence;

-- name: GetReviewSettings :one
SELECT review_period_months, review_warning_days FROM system_settings;

-- name: SetReleasedRevisions :many
-- SetReleasedRevisions memberi revisi yang berlaku setiap dokumen satu set,
-- untuk dibekukan di catatan review.
SELECT d.id AS document_id, d.customer_id, n.number, t.key AS type_key, r.revision, r.snapshot_sha256
FROM documents d
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_types t ON t.id = d.document_type_id
JOIN document_revisions r ON r.document_id = d.id AND r.revision = d.released_revision
WHERE n.customer_code = $1 AND n.sequence = $2 AND n.status NOT IN ('void', 'obsolete')
ORDER BY t.position, n.number;

-- name: SetOfDocument :one
SELECT d.customer_id, n.customer_code, n.sequence
FROM documents d JOIN document_numbers n ON n.id = d.document_number_id
WHERE d.id = $1;

-- name: CreatePeriodicReview :one
INSERT INTO periodic_reviews (customer_id, customer_code, sequence, reviewed_on, participants, conclusion, checklist, revisions,
                              recorded_by_person_id, recorded_by_account_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: GetPeriodicReview :one
SELECT pr.id, pr.seq, pr.customer_id, pr.customer_code, pr.sequence, pr.reviewed_on, pr.participants, pr.conclusion,
       pr.checklist, pr.revisions, pr.status, pr.recorded_by_person_id, rp.name AS recorded_by, pr.recorded_at,
       cp.name AS confirmed_by, pr.confirmed_at
FROM periodic_reviews pr
JOIN people rp ON rp.id = pr.recorded_by_person_id
LEFT JOIN people cp ON cp.id = pr.confirmed_by_person_id
WHERE pr.id = $1;

-- name: ListPeriodicReviews :many
-- ListPeriodicReviews memberi catatan review terbaru, satu customer atau
-- semua, dan hanya yang menunggu konfirmasi bila diminta.
SELECT pr.id, pr.seq, pr.customer_id, pr.customer_code, pr.sequence, pr.reviewed_on, pr.participants, pr.conclusion,
       pr.checklist, pr.revisions, pr.status, pr.recorded_by_person_id, rp.name AS recorded_by, pr.recorded_at,
       cp.name AS confirmed_by, pr.confirmed_at
FROM periodic_reviews pr
JOIN people rp ON rp.id = pr.recorded_by_person_id
LEFT JOIN people cp ON cp.id = pr.confirmed_by_person_id
WHERE (sqlc.narg(customer_id)::uuid IS NULL OR pr.customer_id = sqlc.narg(customer_id)::uuid)
  AND (NOT sqlc.arg(open_only)::boolean OR pr.status = 'submitted')
ORDER BY pr.seq DESC
LIMIT 200;

-- name: ConfirmPeriodicReview :execrows
UPDATE periodic_reviews
SET status = 'confirmed', confirmed_by_person_id = $2, confirmed_by_account_id = $3, confirmed_at = now()
WHERE id = $1 AND status = 'submitted';

-- name: ReviewNoticeSent :one
SELECT EXISTS (SELECT 1 FROM periodic_review_notices WHERE customer_code = $1 AND sequence = $2 AND due_on = $3 AND stage = $4);

-- name: AddReviewNotice :exec
INSERT INTO periodic_review_notices (customer_code, sequence, due_on, stage) VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING;

-- name: ReviewedThisYear :one
-- ReviewedThisYear menghitung set yang direview sejak from: tanpa perubahan
-- (catatan terkonfirmasi) dan dengan perubahan (perubahan berpemicu Periodic
-- review yang diterima).
SELECT
    (SELECT count(*) FROM periodic_reviews pr
     WHERE pr.status = 'confirmed' AND pr.confirmed_at >= sqlc.arg(from_at)
       AND (sqlc.narg(customer_id)::uuid IS NULL OR pr.customer_id = sqlc.narg(customer_id)::uuid))::int AS no_change,
    (SELECT count(*) FROM changes c JOIN change_sources s ON s.id = c.trigger_source_id
     WHERE s.report_key = 'periodic_review' AND c.status = 'accepted' AND c.verified_at >= sqlc.arg(from_at)
       AND (sqlc.narg(customer_id)::uuid IS NULL OR c.customer_id = sqlc.narg(customer_id)::uuid))::int AS revised;
