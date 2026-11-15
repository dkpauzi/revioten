-- D-186 · Scan hardcopy opsional untuk baseline migrasi.
--
-- Sejak migrasi 00040 setiap revisi terbit wajib punya scan bertanda tangan.
-- Pemilik memutuskan (15 November 2026) bahwa untuk dokumen lama yang penting
-- adalah verifikasi oleh petugas migrasi kedua; scan boleh diunggah sebagai
-- bukti tambahan, tetapi tidak wajib. Rilis biasa tetap wajib scan (D-173).

-- +goose Up
ALTER TABLE document_revisions DROP CONSTRAINT document_revisions_released_scan_check;
ALTER TABLE document_revisions ADD CONSTRAINT document_revisions_released_scan_check
    CHECK (status <> 'released' OR migrated OR scan_sha256 IS NOT NULL) NOT VALID;

-- +goose Down
ALTER TABLE document_revisions DROP CONSTRAINT document_revisions_released_scan_check;
ALTER TABLE document_revisions ADD CONSTRAINT document_revisions_released_scan_check
    CHECK (status <> 'released' OR scan_sha256 IS NOT NULL) NOT VALID;
