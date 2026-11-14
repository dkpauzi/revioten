-- M5.1 · Entitas perubahan (R-CHG-01, D-113, UC-40).
--
--   changes            satu perubahan: customer, mekanisme (manual, yokoten,
--                      propagasi template, SQAM), pemicu (sumber perubahan
--                      biasa: PP, customer claim, ECR, temuan audit), jenis
--                      perubahan, kategori 4M opsional, alasan, rujukan
--                      sumber, status, dan pemohonnya. seq memberi nomor
--                      rujukan CHG-0001 (ASUMSI E-98).
--   change_documents   dokumen terdampak. Satu perubahan menunjuk satu atau
--                      lebih dokumen dari customer yang sama.
--
-- Perubahan tidak pernah dihapus (seperti nomor dan akun); statusnya berubah.

-- +goose Up
CREATE TABLE changes (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    seq bigint GENERATED ALWAYS AS IDENTITY,
    customer_id uuid NOT NULL REFERENCES customers (id),
    mechanism text NOT NULL DEFAULT 'manual' CHECK (mechanism IN ('manual', 'yokoten', 'template', 'sqam')),
    trigger_source_id uuid REFERENCES change_sources (id),
    change_type_id uuid NOT NULL REFERENCES change_types (id),
    four_m_category_id uuid REFERENCES four_m_categories (id),
    reason text NOT NULL CHECK (reason <> '' AND reason = btrim(reason) AND char_length(reason) <= 2000),
    source_reference text CHECK (source_reference IS NULL OR (source_reference <> '' AND source_reference = btrim(source_reference) AND char_length(source_reference) <= 200)),
    status text NOT NULL DEFAULT 'submitted' CHECK (status IN ('submitted')),
    requested_by_person_id uuid NOT NULL REFERENCES people (id),
    requested_by_account_id uuid NOT NULL REFERENCES accounts (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT changes_seq_key UNIQUE (seq),
    -- Perubahan manual selalu punya pemicu (R-CHG-06).
    CONSTRAINT changes_trigger_check CHECK (mechanism <> 'manual' OR trigger_source_id IS NOT NULL)
);
CREATE INDEX changes_customer_idx ON changes (customer_id, seq DESC);
GRANT SELECT, INSERT ON changes TO revioten_app;
GRANT UPDATE (status, updated_at) ON changes TO revioten_app;

CREATE TABLE change_documents (
    change_id uuid NOT NULL REFERENCES changes (id),
    document_id uuid NOT NULL REFERENCES documents (id),
    added_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (change_id, document_id)
);
CREATE INDEX change_documents_document_idx ON change_documents (document_id);
GRANT SELECT, INSERT, DELETE ON change_documents TO revioten_app;

-- +goose Down
DROP TABLE change_documents;
DROP TABLE changes;
