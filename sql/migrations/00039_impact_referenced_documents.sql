-- D-168 · Daftar periksa dampak memilih dokumen terkait dari master (R-DOC-11,
-- R-CHG-05). Baris daftar periksa boleh menunjuk satu dokumen di master
-- dokumen terkait; nomor dan judulnya tetap disimpan sebagai teks di
-- document_reference, jadi daftar yang tersimpan tidak berubah bila master
-- diubah. Dokumen yang belum ada di master tetap ditulis bebas.

-- +goose Up
ALTER TABLE change_impacts ADD COLUMN referenced_document_id uuid REFERENCES referenced_documents (id);

-- +goose Down
ALTER TABLE change_impacts DROP COLUMN referenced_document_id;
