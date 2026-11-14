-- M5.2 · Lampiran perubahan (R-CHG-01, R-SEC-04).
--
--   change_attachments   berkas pendukung satu perubahan: nama asli, jenis,
--                        ukuran, SHA-256 isinya, dan pengunggahnya. Isinya
--                        ada di folder privat (ATTACHMENT_DIR) dengan nama
--                        SHA-256. Lampiran yang dilepas ditandai removed_at,
--                        tidak dihapus, supaya jejaknya tetap ada.

-- +goose Up
CREATE TABLE change_attachments (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    change_id uuid NOT NULL REFERENCES changes (id),
    file_name text NOT NULL CHECK (file_name <> '' AND char_length(file_name) <= 200),
    content_type text NOT NULL,
    size_bytes bigint NOT NULL CHECK (size_bytes > 0),
    sha256 text NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    uploaded_by_person_id uuid NOT NULL REFERENCES people (id),
    uploaded_at timestamptz NOT NULL DEFAULT now(),
    removed_at timestamptz
);
CREATE INDEX change_attachments_change_idx ON change_attachments (change_id);
GRANT SELECT, INSERT ON change_attachments TO revioten_app;
GRANT UPDATE (removed_at) ON change_attachments TO revioten_app;

-- +goose Down
DROP TABLE change_attachments;
