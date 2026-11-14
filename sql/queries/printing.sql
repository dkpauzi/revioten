-- Tahap 6: cetak, layout, catatan cetak, scan, dan verifikasi.

-- name: GetDocumentLayout :one
-- GetDocumentLayout memberi layout yang dipakai customer untuk satu jenis
-- dokumen, atau layout bawaan jenis itu (M6.1).
SELECT l.id, l.name, l.document_type_id, l.is_default, l.layout, l.updated_at
FROM print_layouts l
WHERE l.id = coalesce(
    (SELECT c.layout_id FROM customer_print_layouts c WHERE c.customer_id = sqlc.arg(customer_id) AND c.document_type_id = sqlc.arg(document_type_id)),
    (SELECT d.id FROM print_layouts d WHERE d.document_type_id = sqlc.arg(document_type_id) AND d.is_default));

-- name: ListPrintLayouts :many
SELECT l.id, l.name, l.document_type_id, t.key AS document_type_key, l.is_default, l.layout, l.updated_at
FROM print_layouts l JOIN document_types t ON t.id = l.document_type_id
ORDER BY t.position, l.is_default DESC, l.name;

-- name: GetPrintLayout :one
SELECT l.id, l.name, l.document_type_id, t.key AS document_type_key, l.is_default, l.layout, l.updated_at
FROM print_layouts l JOIN document_types t ON t.id = l.document_type_id
WHERE l.id = $1;

-- name: CreatePrintLayout :one
INSERT INTO print_layouts (name, document_type_id, layout, updated_by_person_id) VALUES ($1, $2, $3, $4) RETURNING id;

-- name: UpdatePrintLayout :execrows
UPDATE print_layouts SET name = $2, layout = $3, updated_by_person_id = $4, updated_at = now() WHERE id = $1;

-- name: ListCustomerPrintLayouts :many
SELECT c.document_type_id, c.layout_id FROM customer_print_layouts c WHERE c.customer_id = $1;

-- name: SetCustomerPrintLayout :exec
INSERT INTO customer_print_layouts (customer_id, document_type_id, layout_id) VALUES ($1, $2, $3)
ON CONFLICT (customer_id, document_type_id) DO UPDATE SET layout_id = EXCLUDED.layout_id;

-- name: ClearCustomerPrintLayout :exec
DELETE FROM customer_print_layouts WHERE customer_id = $1 AND document_type_id = $2;

-- name: NextCopyNumber :one
SELECT (coalesce(max(copy_number), 0) + 1)::int AS copy_number FROM print_logs WHERE revision_id = $1;

-- name: AddPrintLog :one
INSERT INTO print_logs (document_id, revision_id, revision, purpose, format, copy_number, content_sha256, printed_by_person_id, printed_by_account_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, printed_at;

-- name: ListPrintLogs :many
SELECT l.id, l.revision, l.purpose, l.format, l.copy_number, l.content_sha256, p.name AS printed_by, l.printed_at
FROM print_logs l JOIN people p ON p.id = l.printed_by_person_id
WHERE l.document_id = $1
ORDER BY l.printed_at DESC
LIMIT 200;

-- name: ListControlledCopies :many
-- ListControlledCopies memberi salinan terkendali revisi lain yang masih
-- beredar, untuk ditarik saat revisi baru terbit (R-PRT-06).
SELECT l.id, l.revision, l.copy_number, p.name AS printed_by, l.printed_at
FROM print_logs l JOIN people p ON p.id = l.printed_by_person_id
WHERE l.document_id = $1 AND l.purpose = 'controlled' AND l.revision <> $2
ORDER BY l.revision, l.copy_number;

-- name: SetRevisionReleaseDate :exec
-- SetRevisionReleaseDate mengisi tanggal rilis draft saat dicetak untuk tanda
-- tangan (D-59). Cetak ulang memakai tanggal cetak terakhir.
UPDATE document_revisions SET release_date = $2 WHERE id = $1 AND status = 'draft';

-- name: SetRevisionScan :execrows
UPDATE document_revisions
SET scan_file_name = $2, scan_content_type = $3, scan_size_bytes = $4, scan_sha256 = $5, scan_uploaded_by_person_id = $6, scan_uploaded_at = now()
WHERE id = $1 AND status = 'draft';

-- name: GetRevisionPrintInfo :one
SELECT r.id, r.revision, r.status, r.released_at, r.release_date, r.snapshot, r.snapshot_sha256,
       r.scan_file_name, r.scan_content_type, r.scan_size_bytes, r.scan_sha256, sp.name AS scan_uploaded_by, r.scan_uploaded_at,
       rp.name AS released_by
FROM document_revisions r
LEFT JOIN people sp ON sp.id = r.scan_uploaded_by_person_id
LEFT JOIN people rp ON rp.id = r.released_by_person_id
WHERE r.document_id = $1 AND r.revision = $2;

-- name: ListRevisionHistory :many
-- ListRevisionHistory memberi revisi dokumen dengan alasan perubahannya,
-- dari yang terlama, untuk tabel riwayat revisi di cetakan (R-REV-04, M6.8).
SELECT r.revision, r.status, r.release_date, r.released_at,
       coalesce((SELECT string_agg(c.reason, '; ' ORDER BY c.seq)
                 FROM change_documents cd JOIN changes c ON c.id = cd.change_id
                 WHERE cd.draft_revision_id = r.id), '')::text AS description
FROM document_revisions r
WHERE r.document_id = $1
ORDER BY r.revision;

-- name: ListRouteSlotHolders :many
-- ListRouteSlotHolders memberi slot rute beserta nama pemegang jabatannya
-- untuk customer ini: jabatan tanpa Project Group, atau Project Group customer.
SELECT s.slot_order, s.stage, s.print_title,
       coalesce((SELECT array_agg(DISTINCT pe.name ORDER BY pe.name)
                 FROM route_slot_positions sp
                 JOIN position_holders h ON h.position_id = sp.position_id
                 JOIN people pe ON pe.id = h.person_id
                 WHERE sp.slot_id = s.id
                   AND (h.project_group_id IS NULL OR h.project_group_id = (SELECT cu.project_group_id FROM customers cu WHERE cu.id = sqlc.arg(customer_id)))),
                '{}')::text[] AS holders
FROM route_slots s
WHERE s.route_id = sqlc.arg(route_id)
ORDER BY s.slot_order;

-- name: GetRevisionForVerify :one
-- GetRevisionForVerify membaca satu revisi untuk halaman verifikasi (M6.5).
SELECT r.id, r.document_id, r.revision, r.status, r.released_at, r.release_date, r.snapshot_sha256,
       d.released_revision, d.customer_id, n.number, n.status AS number_status, t.name AS document_type_name, c.code AS customer_code
FROM document_revisions r
JOIN documents d ON d.id = r.document_id
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_types t ON t.id = d.document_type_id
JOIN customers c ON c.id = d.customer_id
WHERE r.id = $1;

-- name: GetQREnabled :one
SELECT qr_enabled FROM system_settings LIMIT 1;

-- name: GetRevisionDigits :one
SELECT revision_digits FROM number_settings LIMIT 1;
