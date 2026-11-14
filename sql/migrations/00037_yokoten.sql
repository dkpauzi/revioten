-- M5.9 · Yokoten (R-CHG-04, D-40, D-118, UC-43).
--
--   yokotens                  satu penyebaran perbaikan: perubahan sumber yang
--                             sudah diterima, alasan, dan pembuatnya. seq memberi
--                             nomor rujukan YKT-0001 (ASUMSI E-106).
--   yokoten_targets           satu target per model: keputusan pemilik target
--                             (belum, apply, adapt, atau not applicable dengan
--                             alasan), siapa dan kapan, dan perubahan yang dibuat
--                             untuk model itu.
--   yokoten_target_documents  dokumen model itu yang cocok dengan saringan.
--   changes.source_change_id  perubahan bermekanisme yokoten menunjuk perubahan
--                             sumbernya.

-- +goose Up
CREATE TABLE yokotens (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    seq bigint GENERATED ALWAYS AS IDENTITY,
    source_change_id uuid NOT NULL REFERENCES changes (id),
    reason text NOT NULL CHECK (reason <> '' AND reason = btrim(reason) AND char_length(reason) <= 2000),
    filter_process_id uuid REFERENCES processes (id),
    filter_text text CHECK (filter_text IS NULL OR (filter_text <> '' AND char_length(filter_text) <= 200)),
    created_by_person_id uuid NOT NULL REFERENCES people (id),
    created_by_account_id uuid NOT NULL REFERENCES accounts (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT yokotens_seq_key UNIQUE (seq)
);
GRANT SELECT, INSERT ON yokotens TO revioten_app;

CREATE TABLE yokoten_targets (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    yokoten_id uuid NOT NULL REFERENCES yokotens (id),
    model_id uuid NOT NULL REFERENCES models (id),
    customer_id uuid NOT NULL REFERENCES customers (id),
    decision text NOT NULL DEFAULT 'pending' CHECK (decision IN ('pending', 'apply', 'adapt', 'not_applicable')),
    reason text CHECK (reason IS NULL OR (reason <> '' AND reason = btrim(reason) AND char_length(reason) <= 500)),
    decided_by_person_id uuid REFERENCES people (id),
    decided_at timestamptz,
    change_id uuid REFERENCES changes (id),
    CONSTRAINT yokoten_targets_model_key UNIQUE (yokoten_id, model_id),
    CONSTRAINT yokoten_targets_decided_check CHECK ((decision = 'pending') = (decided_at IS NULL) AND (decided_at IS NULL) = (decided_by_person_id IS NULL)),
    -- Tidak berlaku selalu dengan alasan; apply dan adapt selalu dengan perubahan.
    CONSTRAINT yokoten_targets_reason_needed CHECK (decision <> 'not_applicable' OR reason IS NOT NULL),
    CONSTRAINT yokoten_targets_change_check CHECK ((decision IN ('apply', 'adapt')) = (change_id IS NOT NULL))
);
GRANT SELECT, INSERT ON yokoten_targets TO revioten_app;
GRANT UPDATE (decision, reason, decided_by_person_id, decided_at, change_id) ON yokoten_targets TO revioten_app;

CREATE TABLE yokoten_target_documents (
    target_id uuid NOT NULL REFERENCES yokoten_targets (id),
    document_id uuid NOT NULL REFERENCES documents (id),
    PRIMARY KEY (target_id, document_id)
);
GRANT SELECT, INSERT ON yokoten_target_documents TO revioten_app;

ALTER TABLE changes
    ADD COLUMN source_change_id uuid REFERENCES changes (id),
    ADD CONSTRAINT changes_yokoten_check CHECK ((mechanism = 'yokoten') = (source_change_id IS NOT NULL));

-- +goose Down
ALTER TABLE changes DROP CONSTRAINT changes_yokoten_check, DROP COLUMN source_change_id;
DROP TABLE yokoten_target_documents;
DROP TABLE yokoten_targets;
DROP TABLE yokotens;
