-- D-130, D-156 · Master dokumen terkait (R-DOC-11).
--
--   referenced_documents       WI, SOP, QEP, dan dokumen sejenis: nomor, judul,
--                              level, revisi, dan tanggal revisi. Dicatat sekali,
--                              supaya nomor dan judul sama persis dengan sumbernya.
--                              Tidak pernah dihapus; yang tidak dipakai lagi
--                              dinonaktifkan.
--   referenced_document_links  dokumen bawaan per proses, dengan cakupan yang sama
--                              seperti poin template: umum, customer, group model,
--                              atau model (D-129). Untuk satu model semua tautan
--                              yang berlaku dijumlahkan.

-- +goose Up
CREATE TABLE referenced_documents (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    number text NOT NULL CHECK (number ~ '^[A-Z0-9][A-Z0-9-]{1,39}$'),
    title text NOT NULL CHECK (title <> '' AND title = btrim(title) AND char_length(title) <= 200),
    level text CHECK (level IS NULL OR (level <> '' AND level = btrim(level) AND char_length(level) <= 60)),
    revision text CHECK (revision IS NULL OR (revision <> '' AND revision = btrim(revision) AND char_length(revision) <= 10)),
    revision_date date,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT referenced_documents_number_key UNIQUE (number)
);
GRANT SELECT, INSERT ON referenced_documents TO revioten_app;
GRANT UPDATE (title, level, revision, revision_date, active, updated_at) ON referenced_documents TO revioten_app;

CREATE TABLE referenced_document_links (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    document_id uuid NOT NULL REFERENCES referenced_documents (id),
    process_id uuid NOT NULL REFERENCES processes (id),
    scope text NOT NULL DEFAULT 'general',
    customer_id uuid REFERENCES customers (id),
    model_group_id uuid REFERENCES model_groups (id),
    model_id uuid REFERENCES models (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT referenced_document_links_scope_check CHECK (
        (scope = 'general' AND customer_id IS NULL AND model_group_id IS NULL AND model_id IS NULL)
        OR (scope = 'customer' AND customer_id IS NOT NULL AND model_group_id IS NULL AND model_id IS NULL)
        OR (scope = 'model_group' AND customer_id IS NULL AND model_group_id IS NOT NULL AND model_id IS NULL)
        OR (scope = 'model' AND customer_id IS NULL AND model_group_id IS NULL AND model_id IS NOT NULL)
    )
);
-- Satu dokumen paling banyak satu tautan per proses dan sasaran cakupan.
-- NULLS NOT DISTINCT: tautan umum (semua sasaran kosong) juga tidak boleh ganda.
CREATE UNIQUE INDEX referenced_document_links_key
    ON referenced_document_links (document_id, process_id, scope, customer_id, model_group_id, model_id) NULLS NOT DISTINCT;
CREATE INDEX referenced_document_links_process_idx ON referenced_document_links (process_id);
-- Tautan adalah pengaturan, bukan catatan, jadi boleh dihapus (tercatat di audit log).
GRANT SELECT, INSERT, DELETE ON referenced_document_links TO revioten_app;

-- +goose Down
DROP TABLE referenced_document_links;
DROP TABLE referenced_documents;