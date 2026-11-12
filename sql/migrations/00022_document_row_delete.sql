-- M4.2 · Editor tabel: baris draft bisa dihapus.
--
-- Dokumen masih draft (belum ada revisi rilis sampai Tahap 6), jadi baris
-- yang salah boleh dihapus. ID baris tidak pernah dipakai ulang (uuidv7), jadi
-- R-DOC-02 tetap terjaga. Setiap penghapusan tercatat di audit log.

-- +goose Up
GRANT DELETE ON document_rows TO revioten_app;

-- +goose Down
REVOKE DELETE ON document_rows FROM revioten_app;