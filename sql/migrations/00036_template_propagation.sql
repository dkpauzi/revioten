-- M5.8 · Propagasi template (R-TPL-06, R-TPL-07, R-TPL-08, R-TPL-09).
--
--   document_template_modes   mode tautan satu dokumen ke satu process
--                             template: follow (bawaan, tidak perlu baris),
--                             pinned (tetap di versinya kecuali cakupan all),
--                             atau detached (tidak pernah ikut propagasi).
--   changes                   perubahan bermekanisme template menyimpan versi
--                             template yang dipropagasikan, cakupannya
--                             (following atau all), aturan konflik (keep atau
--                             overwrite), dan tanggal berlaku. Cakupan all
--                             wajib rujukan revisi SQAM dan tanggal berlaku
--                             (R-TPL-08); alasannya adalah alasan perubahan.
--   change_documents          hasil penerapan per dokumen: baris yang
--                             diperbarui, yang berkonflik dan dibiarkan, dan
--                             yang poinnya tidak ada lagi di versi baru.
--
-- Isi template ditulis ke draft saat perubahan diterima (M5.4), tidak pernah
-- langsung ke dokumen (R-TPL-06), dan revisi terbit tidak pernah ditulis
-- ulang (R-TPL-07).

-- +goose Up
CREATE TABLE document_template_modes (
    document_id uuid NOT NULL REFERENCES documents (id),
    process_template_id uuid NOT NULL REFERENCES process_templates (id),
    mode text NOT NULL CHECK (mode IN ('follow', 'pinned', 'detached')),
    reason text CHECK (reason IS NULL OR (reason <> '' AND reason = btrim(reason) AND char_length(reason) <= 500)),
    updated_by_person_id uuid NOT NULL REFERENCES people (id),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (document_id, process_template_id),
    CONSTRAINT document_template_modes_reason_needed CHECK (mode = 'follow' OR reason IS NOT NULL)
);
GRANT SELECT, INSERT ON document_template_modes TO revioten_app;
GRANT UPDATE (mode, reason, updated_by_person_id, updated_at) ON document_template_modes TO revioten_app;

ALTER TABLE changes
    ADD COLUMN template_version_id uuid REFERENCES process_template_versions (id),
    ADD COLUMN propagation_scope text CHECK (propagation_scope IS NULL OR propagation_scope IN ('following', 'all')),
    ADD COLUMN conflict_policy text CHECK (conflict_policy IS NULL OR conflict_policy IN ('keep', 'overwrite')),
    ADD COLUMN effective_date date,
    ADD CONSTRAINT changes_template_check CHECK (
        (mechanism = 'template') = (template_version_id IS NOT NULL)
        AND (template_version_id IS NULL) = (propagation_scope IS NULL)
        AND (template_version_id IS NULL) = (conflict_policy IS NULL)),
    ADD CONSTRAINT changes_scope_all_check CHECK (
        propagation_scope IS DISTINCT FROM 'all' OR (source_reference IS NOT NULL AND effective_date IS NOT NULL));

ALTER TABLE change_documents
    ADD COLUMN rows_updated integer CHECK (rows_updated IS NULL OR rows_updated >= 0),
    ADD COLUMN rows_conflict integer CHECK (rows_conflict IS NULL OR rows_conflict >= 0),
    ADD COLUMN rows_removed integer CHECK (rows_removed IS NULL OR rows_removed >= 0);
GRANT UPDATE (rows_updated, rows_conflict, rows_removed) ON change_documents TO revioten_app;

GRANT UPDATE (template_version_id) ON document_rows TO revioten_app;

-- +goose Down
REVOKE UPDATE (template_version_id) ON document_rows FROM revioten_app;
REVOKE UPDATE (rows_updated, rows_conflict, rows_removed) ON change_documents FROM revioten_app;
ALTER TABLE change_documents DROP COLUMN rows_removed, DROP COLUMN rows_conflict, DROP COLUMN rows_updated;
ALTER TABLE changes
    DROP CONSTRAINT changes_scope_all_check,
    DROP CONSTRAINT changes_template_check,
    DROP COLUMN effective_date, DROP COLUMN conflict_policy, DROP COLUMN propagation_scope, DROP COLUMN template_version_id;
DROP TABLE document_template_modes;
