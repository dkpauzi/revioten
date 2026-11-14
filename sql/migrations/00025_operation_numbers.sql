-- M4.6 · Urutan dan nomor operasi (R-CMP-01): nomor operasi diisi otomatis
-- per langkah proses dan bisa diubah manual.
--
--   document_rows.operation_manual   true bila nomor operasi diketik atau
--                                    ditempel penyusun. Penomoran otomatis
--                                    tidak menimpa nomor seperti itu.
--
-- Baris yang sudah ada dan punya nomor dianggap diketik penyusun.

-- +goose Up
ALTER TABLE document_rows ADD COLUMN operation_manual boolean NOT NULL DEFAULT false;
UPDATE document_rows SET operation_manual = true WHERE operation_number IS NOT NULL;
ALTER TABLE document_rows ADD CONSTRAINT document_rows_operation_manual_check CHECK (NOT operation_manual OR operation_number IS NOT NULL);
GRANT UPDATE (operation_manual) ON document_rows TO revioten_app;

-- +goose Down
REVOKE UPDATE (operation_manual) ON document_rows FROM revioten_app;
ALTER TABLE document_rows DROP CONSTRAINT document_rows_operation_manual_check, DROP COLUMN operation_manual;