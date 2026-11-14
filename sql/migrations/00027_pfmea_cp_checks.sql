-- M4.9 · PFMEA dan CP berdampingan (R-DOC-06, D-65).
--
--   pfmea_cp_checks   tanda "dicek oleh, tanggal" per proses per revisi untuk
--                     satu pasangan dokumen PFMEA dan CP di set yang sama.
--                     Revisi kedua dokumen saat dicek ikut disimpan, jadi
--                     tanda revisi lama tidak berlaku untuk revisi baru.
--
-- Tanda dicabut dengan menghapus barisnya; setiap pemberian dan pencabutan
-- tercatat di audit log.

-- +goose Up
CREATE TABLE pfmea_cp_checks (
    pfmea_document_id uuid NOT NULL REFERENCES documents (id),
    cp_document_id uuid NOT NULL REFERENCES documents (id),
    process_id uuid NOT NULL REFERENCES processes (id),
    pfmea_revision smallint NOT NULL,
    cp_revision smallint NOT NULL,
    checked_by_person_id uuid NOT NULL REFERENCES people (id),
    checked_at timestamptz NOT NULL DEFAULT now(),
    -- Sidik jari baris proses itu di kedua dokumen saat dicek (SHA-256 dari ID
    -- dan versi setiap baris). Bila berbeda dengan sekarang, ada baris yang
    -- ditambah, diubah, atau dihapus sesudah dicek.
    rows_fingerprint text NOT NULL CHECK (rows_fingerprint ~ '^[0-9a-f]{64}$'),
    PRIMARY KEY (pfmea_document_id, cp_document_id, process_id, pfmea_revision, cp_revision),
    CONSTRAINT pfmea_cp_checks_pair_check CHECK (pfmea_document_id <> cp_document_id)
);
GRANT SELECT, INSERT, DELETE ON pfmea_cp_checks TO revioten_app;
-- Cek ulang memperbarui tanda yang perlu dicek ulang.
GRANT UPDATE (checked_by_person_id, checked_at, rows_fingerprint) ON pfmea_cp_checks TO revioten_app;

-- +goose Down
DROP TABLE pfmea_cp_checks;