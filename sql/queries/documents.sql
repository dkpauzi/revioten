-- M4.1 · Dokumen dan baris. Lihat migrasi 00021.

-- name: GetNumberForDocument :one
-- GetNumberForDocument mengunci nomor register yang akan dipakai dokumen.
SELECT n.id, n.number, n.customer_id, n.document_type_id, n.status, n.initial_revision, d.key AS document_type_key
FROM document_numbers n
JOIN document_types d ON d.id = n.document_type_id
WHERE n.id = $1
FOR NO KEY UPDATE OF n;


-- name: DocumentExistsForNumber :one
SELECT EXISTS (SELECT 1 FROM documents WHERE document_number_id = $1);


-- name: GetCustomerForDocument :one
SELECT id, default_route_id, default_form_set_id FROM customers WHERE id = $1;


-- name: LatestPublishedFormSetVersion :one
-- Versi terbit terakhir satu form set, beserta nama set dan versi format ketiga sheet.
SELECT v.id, s.name, v.pfd_format_version_id, v.pfmea_format_version_id, v.cp_format_version_id
FROM form_set_versions v
JOIN form_sets s ON s.id = v.form_set_id
WHERE v.form_set_id = $1 AND v.status = 'published'
ORDER BY v.version DESC
LIMIT 1;


-- name: GetModelOwner :one
SELECT customer_id FROM models WHERE id = $1;


-- name: GetPartNumberOwner :one
SELECT customer_id, model_id FROM part_numbers WHERE id = $1;


-- name: RouteExists :one
SELECT EXISTS (SELECT 1 FROM approval_routes WHERE id = $1);


-- name: CreateDocument :one
INSERT INTO documents (document_number_id, customer_id, document_type_id, revision, form_set_version_id,
                       format_version_id, approval_route_id, route_reason, created_by_person_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;


-- name: AddDocumentModel :exec
INSERT INTO document_models (document_id, model_id) VALUES ($1, $2);


-- name: AddDocumentPartNumber :exec
INSERT INTO document_part_numbers (document_id, part_number_id) VALUES ($1, $2);


-- name: GetDocument :one
SELECT d.id, d.document_number_id, n.number, d.customer_id, c.code AS customer_code, d.document_type_id,
       t.key AS document_type_key, t.name AS document_type_name, d.revision, d.status,
       d.form_set_version_id, s.name AS form_set_name, d.format_version_id,
       d.approval_route_id, r.name AS route_name, d.route_reason, d.header, d.header_version, d.created_at, d.updated_at
FROM documents d
JOIN document_numbers n ON n.id = d.document_number_id
JOIN customers c ON c.id = d.customer_id
JOIN document_types t ON t.id = d.document_type_id
JOIN form_set_versions fv ON fv.id = d.form_set_version_id
JOIN form_sets s ON s.id = fv.form_set_id
LEFT JOIN approval_routes r ON r.id = d.approval_route_id
WHERE d.id = $1;


-- name: LockDocument :exec
SELECT id FROM documents WHERE id = $1 FOR NO KEY UPDATE;


-- name: ListDocuments :many
SELECT d.id, n.number, t.key AS document_type_key, t.name AS document_type_name, d.revision, d.status,
       s.name AS form_set_name, d.updated_at,
       (SELECT count(*) FROM document_rows r WHERE r.document_id = d.id)::int AS row_count,
       -- Nama model dokumen, menurut abjad (M4.7).
       (SELECT coalesce(array_agg(m.name ORDER BY m.name), '{}')::text[]
        FROM document_models dm JOIN models m ON m.id = dm.model_id WHERE dm.document_id = d.id) AS model_names
FROM documents d
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_types t ON t.id = d.document_type_id
JOIN form_set_versions fv ON fv.id = d.form_set_version_id
JOIN form_sets s ON s.id = fv.form_set_id
WHERE d.customer_id = $1
ORDER BY n.number;


-- name: ListDocumentModels :many
SELECT m.id, m.name
FROM document_models dm JOIN models m ON m.id = dm.model_id
WHERE dm.document_id = $1
ORDER BY m.name;


-- name: ListDocumentPartNumbers :many
SELECT p.id, p.number, p.name
FROM document_part_numbers dp JOIN part_numbers p ON p.id = dp.part_number_id
WHERE dp.document_id = $1
ORDER BY p.number;


-- name: ListDocumentRows :many
SELECT r.id, r.position, r.process_id, n.name AS process_name, r.operation_number, r.content, r.version,
       r.template_point_id, r.template_version_id, r.operation_manual, r.detached_at, r.detach_reason, r.section_id
FROM document_rows r
LEFT JOIN process_names n ON n.process_id = r.process_id AND n.kind = 'display'
WHERE r.document_id = $1
ORDER BY r.position;


-- name: AddDocumentRow :one
-- Nomor operasi yang dikirim saat baris dibuat berarti diketik atau ditempel
-- penyusun, jadi ditandai manual (M4.6).
INSERT INTO document_rows (document_id, position, process_id, operation_number, content, operation_manual, section_id)
VALUES ($1, (SELECT coalesce(max(position), 0) + 1 FROM document_rows WHERE document_id = $1), $2, sqlc.narg(operation_number), $3,
        sqlc.narg(operation_number)::text IS NOT NULL, sqlc.narg(section_id))
RETURNING id, position;


-- name: UpdateDocumentRow :execrows
-- UpdateDocumentRow hanya berhasil bila versi baris masih sama dengan yang
-- dibaca penyusun (M4.3); versinya lalu naik satu.
UPDATE document_rows
SET process_id = $3, operation_number = $4, content = $5, operation_manual = sqlc.arg(operation_manual),
    version = version + 1, updated_at = now()
WHERE id = $1 AND document_id = $2 AND version = sqlc.arg(expected_version);


-- name: SetDocumentHeader :execrows
UPDATE documents SET header = $2, header_version = header_version + 1, updated_at = now()
WHERE id = $1 AND header_version = sqlc.arg(expected_version);


-- name: TouchDocument :exec
UPDATE documents SET updated_at = now() WHERE id = $1;


-- name: ProcessExists :one
SELECT EXISTS (SELECT 1 FROM processes WHERE id = $1);

-- M4.2 · Editor tabel.

-- name: DeleteDocumentRow :one
DELETE FROM document_rows WHERE id = $1 AND document_id = $2
RETURNING position, operation_number, template_point_id;


-- name: ShiftRowsUp :exec
-- ShiftRowsUp menggeser baris sesudah posisi yang dihapus satu tempat naik.
UPDATE document_rows SET position = position - 1
WHERE document_id = $1 AND position > $2;


-- name: ListDocumentSet :many
-- ListDocumentSet mengambil dokumen yang nomornya didaftarkan dalam satu
-- pendaftaran (set PFD, PFMEA, CP, M2.4), menurut urutan jenis dokumen.
SELECT d.id, n.number, t.id AS document_type_id, t.key AS document_type_key, t.name AS document_type_name
FROM documents d
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_types t ON t.id = d.document_type_id
WHERE n.batch_id = (SELECT n2.batch_id FROM documents d2 JOIN document_numbers n2 ON n2.id = d2.document_number_id WHERE d2.id = $1)
ORDER BY t.position, n.number;

-- M4.5: perakitan dari process template (R-CMP-01 sampai R-CMP-04).


-- name: AddComposedRow :one
-- AddComposedRow menambah baris hasil perakitan di akhir dokumen, tertaut ke
-- poin asalnya dan versi template yang dipakai (R-CMP-04).
INSERT INTO document_rows (document_id, position, process_id, content, template_point_id, template_version_id, section_id)
VALUES ($1, (SELECT coalesce(max(position), 0) + 1 FROM document_rows WHERE document_id = $1), $2, $3, $4, $5, sqlc.narg(section_id))
RETURNING id, position;


-- name: ListPointExclusions :many
SELECT point_id, reason, section_id FROM document_point_exclusions WHERE document_id = $1;


-- name: ExcludePoint :exec
-- ExcludePoint mencatat poin yang dikecualikan dari satu dokumen (R-CMP-03).
-- Mengecualikan lagi mengganti alasannya.
INSERT INTO document_point_exclusions (document_id, point_id, reason, section_id)
VALUES ($1, $2, $3, sqlc.narg(section_id))
ON CONFLICT (document_id, section_id, point_id) DO UPDATE SET reason = EXCLUDED.reason, excluded_at = now();


-- name: ClearPointExclusion :exec
DELETE FROM document_point_exclusions
WHERE document_id = $1 AND point_id = $2 AND section_id IS NOT DISTINCT FROM sqlc.narg(section_id);


-- M4.6: urutan dan nomor operasi.


-- name: SetRowPosition :exec
-- SetRowPosition memindahkan satu baris. Batasan unik urutan ditunda sampai
-- akhir transaksi, jadi baris boleh bertukar tempat di tengah jalan.
UPDATE document_rows SET position = $3 WHERE id = $1 AND document_id = $2;


-- name: SetRowOperationNumber :exec
-- SetRowOperationNumber mengisi nomor operasi otomatis. Versi baris naik,
-- supaya layar yang masih memegang nomor lama tidak menimpanya (M4.3).
UPDATE document_rows
SET operation_number = $3, operation_manual = false, version = version + 1, updated_at = now()
WHERE id = $1 AND document_id = $2;


-- M4.7: salin komposisi dari dokumen model lain (R-CMP-05).


-- name: CopyDocumentRows :execrows
-- CopyDocumentRows menyalin semua baris satu dokumen ke dokumen lain dengan
-- ID baru. Urutan, proses, nomor operasi, isi, dan tautan template ikut.
-- Bagian dipetakan menurut namanya: bagian tujuan dibuat lebih dulu dengan nama yang sama (D-155).
INSERT INTO document_rows (document_id, position, process_id, operation_number, operation_manual, content, template_point_id, template_version_id, detached_at, detach_reason, section_id)
SELECT sqlc.arg(target_id)::uuid, s.position, s.process_id, s.operation_number, s.operation_manual, s.content, s.template_point_id, s.template_version_id, s.detached_at, s.detach_reason, ts.id
FROM document_rows s
LEFT JOIN document_sections ss ON ss.id = s.section_id
LEFT JOIN document_sections ts ON ts.document_id = sqlc.arg(target_id)::uuid AND ts.name = ss.name
WHERE s.document_id = sqlc.arg(source_id);


-- name: CopyPointExclusions :exec
INSERT INTO document_point_exclusions (document_id, point_id, reason, section_id)
SELECT sqlc.arg(target_id)::uuid, s.point_id, s.reason, ts.id
FROM document_point_exclusions s
LEFT JOIN document_sections ss ON ss.id = s.section_id
LEFT JOIN document_sections ts ON ts.document_id = sqlc.arg(target_id)::uuid AND ts.name = ss.name
WHERE s.document_id = sqlc.arg(source_id)
ON CONFLICT (document_id, section_id, point_id) DO NOTHING;


-- M4.8: putus tautan.


-- name: DetachDocumentRow :execrows
-- DetachDocumentRow memutus satu baris dari poin template-nya, dengan alasan
-- (R-CMP-04). Hasil 0 baris berarti baris itu tidak tertaut atau sudah putus.
UPDATE document_rows
SET detached_at = now(), detach_reason = $3, updated_at = now()
WHERE id = $1 AND document_id = $2 AND template_point_id IS NOT NULL AND detached_at IS NULL;


-- M4.9: tanda dicek PFMEA dan CP (R-DOC-06).


-- name: ListPfmeaCpChecks :many
-- ListPfmeaCpChecks mengambil tanda dicek satu pasangan dokumen untuk revisi
-- keduanya sekarang, beserta nama pemberinya.
SELECT c.process_id, c.section_key, c.pfmea_revision, c.cp_revision, c.checked_at, c.rows_fingerprint, p.name AS checked_by
FROM pfmea_cp_checks c
JOIN people p ON p.id = c.checked_by_person_id
WHERE c.pfmea_document_id = $1 AND c.cp_document_id = $2 AND c.pfmea_revision = $3 AND c.cp_revision = $4;


-- name: AddPfmeaCpCheck :execrows
-- AddPfmeaCpCheck memberi tanda. Tanda yang masih segar tidak diganti, jadi
-- pemberi dan waktunya tetap. Tanda yang sidik jarinya sudah berbeda (perlu
-- dicek ulang) diperbarui: pemberi, waktu, dan sidik jarinya yang baru.
INSERT INTO pfmea_cp_checks (pfmea_document_id, cp_document_id, process_id, pfmea_revision, cp_revision, checked_by_person_id, rows_fingerprint, section_key)
VALUES ($1, $2, $3, $4, $5, $6, $7, sqlc.narg(section_key))
ON CONFLICT (pfmea_document_id, cp_document_id, section_key, process_id, pfmea_revision, cp_revision) DO UPDATE
SET checked_by_person_id = EXCLUDED.checked_by_person_id, checked_at = now(), rows_fingerprint = EXCLUDED.rows_fingerprint
WHERE pfmea_cp_checks.rows_fingerprint <> EXCLUDED.rows_fingerprint;


-- name: RemovePfmeaCpCheck :execrows
DELETE FROM pfmea_cp_checks
WHERE pfmea_document_id = $1 AND cp_document_id = $2 AND process_id = $3 AND pfmea_revision = $4 AND cp_revision = $5
  AND section_key IS NOT DISTINCT FROM sqlc.narg(section_key);


-- name: GetProcessDisplayName :one
SELECT name FROM process_names WHERE process_id = $1 AND kind = 'display';


-- D-155 pola 2: bagian per PCB.


-- name: ListDocumentSections :many
-- ListDocumentSections mengambil bagian satu dokumen menurut urutan, beserta
-- part number-nya (nomor dipisah koma, menurut nomor).
SELECT s.id, s.section_key, s.position, s.name,
       coalesce(array_agg(p.id ORDER BY p.number) FILTER (WHERE p.id IS NOT NULL), '{}')::uuid[] AS part_number_ids,
       coalesce(array_agg(p.number ORDER BY p.number) FILTER (WHERE p.id IS NOT NULL), '{}')::text[] AS part_numbers,
       coalesce(array_agg(coalesce(p.name, '') ORDER BY p.number) FILTER (WHERE p.id IS NOT NULL), '{}')::text[] AS part_names
FROM document_sections s
LEFT JOIN document_section_part_numbers sp ON sp.section_key = s.section_key
LEFT JOIN part_numbers p ON p.id = sp.part_number_id
WHERE s.document_id = $1
GROUP BY s.id
ORDER BY s.position;


-- name: AddDocumentSection :one
INSERT INTO document_sections (document_id, section_key, position, name)
VALUES ($1, $2, (SELECT coalesce(max(position), 0) + 1 FROM document_sections WHERE document_id = $1), $3)
RETURNING id;


-- name: RenameDocumentSections :exec
-- RenameDocumentSections mengganti nama bagian di semua dokumen satu set.
UPDATE document_sections SET name = $2, updated_at = now() WHERE section_key = $1;


-- name: SetDocumentSectionPosition :exec
UPDATE document_sections SET position = $3, updated_at = now() WHERE document_id = $1 AND id = $2;


-- name: DeleteDocumentSections :exec
DELETE FROM document_sections WHERE section_key = $1;


-- name: ClearSectionPartNumbers :exec
DELETE FROM document_section_part_numbers WHERE section_key = $1;


-- name: AddSectionPartNumber :exec
INSERT INTO document_section_part_numbers (section_key, part_number_id) VALUES ($1, $2);


-- name: AssignRowsToSection :exec
-- AssignRowsToSection memindahkan baris tanpa bagian ke satu bagian; dipakai
-- saat bagian pertama dibuat, supaya baris yang sudah ada tidak tertinggal.
UPDATE document_rows SET section_id = $2 WHERE document_id = $1 AND section_id IS NULL;


-- name: AssignExclusionsToSection :exec
UPDATE document_point_exclusions SET section_id = $2 WHERE document_id = $1 AND section_id IS NULL;


-- name: CountSectionRows :one
SELECT count(*)::int FROM document_rows r JOIN document_sections s ON s.id = r.section_id WHERE s.section_key = $1;


-- name: CopySectionsFrom :exec
-- CopySectionsFrom membuat bagian dokumen tujuan dengan nama dan urutan yang
-- sama dengan dokumen sumber. section_key baru per bagian, sama di set tujuan
-- karena kunci yang sama dipakai untuk semua dokumen tujuan (lihat copy.go).
INSERT INTO document_sections (document_id, section_key, position, name)
VALUES ($1, $2, $3, $4);


-- name: NewSectionKey :one
-- NewSectionKey membuat kunci bagian baru, dipakai bersama semua dokumen satu set.
SELECT uuidv7()::uuid;


-- name: DeleteSectionExclusions :exec
DELETE FROM document_point_exclusions WHERE section_id IN (SELECT id FROM document_sections WHERE section_key = $1);
