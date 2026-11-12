-- M4.1 · Dokumen dan baris (R-DOC-01, R-DOC-02, R-DOC-08, D-47, D-133).
--
--   documents             satu dokumen PFD, PFMEA, atau CP. Terhubung ke satu
--                         nomor register (satu nomor satu dokumen), customer,
--                         jenis dokumen, versi form set dan versi format yang
--                         dipakai, rute approval, dan isi header (JSONB per
--                         ID field header).
--   document_models       model yang dicakup dokumen (R-DOC-08)
--   document_part_numbers part number yang dicakup dokumen
--   document_rows         baris datar: ID tetap, urutan, proses master, nomor
--                         operasi, dan isi JSONB per ID kolom (D-02, R-DOC-01).
--                         Proses dan nomor operasi adalah atribut baris, jadi
--                         kolom terisi otomatis (D-122, D-123) membacanya dari sini.
--
-- Status draft saja dulu; revisi, rilis, dan snapshot datang di Tahap 5 dan 6.

-- +goose Up
CREATE TABLE documents (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    document_number_id uuid NOT NULL REFERENCES document_numbers (id),
    customer_id uuid NOT NULL REFERENCES customers (id),
    document_type_id uuid NOT NULL REFERENCES document_types (id),
    revision smallint NOT NULL,
    status text NOT NULL DEFAULT 'draft',
    form_set_version_id uuid NOT NULL REFERENCES form_set_versions (id),
    format_version_id uuid NOT NULL REFERENCES format_template_versions (id),
    approval_route_id uuid REFERENCES approval_routes (id),
    -- Alasan bila rute berbeda dari rute bawaan customer (D-47).
    route_reason text,
    header jsonb NOT NULL DEFAULT '{}',
    created_by_person_id uuid NOT NULL REFERENCES people (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT documents_number_key UNIQUE (document_number_id),
    CONSTRAINT documents_status_check CHECK (status IN ('draft')),
    CONSTRAINT documents_revision_check CHECK (revision >= 0),
    CONSTRAINT documents_header_check CHECK (jsonb_typeof(header) = 'object'),
    CONSTRAINT documents_route_reason_check CHECK (
        route_reason IS NULL OR (route_reason <> '' AND route_reason = btrim(route_reason) AND char_length(route_reason) <= 500)
    )
);
CREATE INDEX documents_customer_idx ON documents (customer_id, created_at);

CREATE TABLE document_models (
    document_id uuid NOT NULL REFERENCES documents (id),
    model_id uuid NOT NULL REFERENCES models (id),
    PRIMARY KEY (document_id, model_id)
);

CREATE TABLE document_part_numbers (
    document_id uuid NOT NULL REFERENCES documents (id),
    part_number_id uuid NOT NULL REFERENCES part_numbers (id),
    PRIMARY KEY (document_id, part_number_id)
);

CREATE TABLE document_rows (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    document_id uuid NOT NULL REFERENCES documents (id),
    position integer NOT NULL,
    process_id uuid REFERENCES processes (id),
    operation_number text,
    content jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT document_rows_position_key UNIQUE (document_id, position) DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT document_rows_position_check CHECK (position >= 1),
    CONSTRAINT document_rows_content_check CHECK (jsonb_typeof(content) = 'object'),
    CONSTRAINT document_rows_operation_check CHECK (
        operation_number IS NULL OR (operation_number <> '' AND operation_number = btrim(operation_number) AND char_length(operation_number) <= 20)
    )
);
-- Isi baris bisa dicari per field (R-DOC-01: key fields are indexed).
CREATE INDEX document_rows_content_idx ON document_rows USING gin (content jsonb_path_ops);

-- Hak sekecil mungkin (E-06): dokumen tidak pernah dihapus, dan identitasnya
-- (nomor, customer, jenis, form set) tidak pernah diubah.
GRANT SELECT, INSERT ON documents TO revioten_app;
GRANT UPDATE (header, approval_route_id, route_reason, updated_at) ON documents TO revioten_app;
GRANT SELECT, INSERT ON document_models TO revioten_app;
GRANT SELECT, INSERT ON document_part_numbers TO revioten_app;
GRANT SELECT, INSERT ON document_rows TO revioten_app;
GRANT UPDATE (position, process_id, operation_number, content, updated_at) ON document_rows TO revioten_app;

-- +goose Down
DROP TABLE document_rows;
DROP TABLE document_part_numbers;
DROP TABLE document_models;
DROP TABLE documents;