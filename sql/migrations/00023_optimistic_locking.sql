-- M4.3 · Penguncian optimistik (R-DOC-07): dua penyusun tidak saling menimpa.
--
-- Setiap baris dan header dokumen punya nomor versi yang naik setiap kali
-- disimpan. Perubahan dikirim bersama versi yang dibaca penyusun; bila versi
-- di database sudah lebih baru, perubahan ditolak sebagai konflik dan layar
-- memuat isi terbaru. Tidak ada kunci yang ditahan selama orang mengetik.

-- +goose Up
ALTER TABLE document_rows ADD COLUMN version integer NOT NULL DEFAULT 1;
ALTER TABLE documents ADD COLUMN header_version integer NOT NULL DEFAULT 1;
GRANT UPDATE (version) ON document_rows TO revioten_app;
GRANT UPDATE (header_version) ON documents TO revioten_app;

-- +goose Down
REVOKE UPDATE (header_version) ON documents FROM revioten_app;
REVOKE UPDATE (version) ON document_rows FROM revioten_app;
ALTER TABLE documents DROP COLUMN header_version;
ALTER TABLE document_rows DROP COLUMN version;