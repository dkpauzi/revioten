-- Fase 1 desain (D-200, D-202).
--
--   models.eol_at, eol_by_person_id      model end of life. Ditandai PPAP Engineer
--                                        atau Project Leader (E-2 customer itu,
--                                        D-200). Set yang semua modelnya EOL
--                                        keluar dari review berkala.
--   document_revisions.ready_at, ready_by_person_id
--                                        tanda "siap cetak" pada draft: batas
--                                        tahap "edit sampai siap cetak" dan
--                                        "menunggu cetak" di laporan kinerja.
--   change_sources.report_key            kelompok pemicu di laporan kinerja yang
--                                        tidak berubah walau nama sumbernya
--                                        diganti admin: ecr, customer_claim,
--                                        audit_finding, periodic_review.

-- +goose Up
ALTER TABLE models
    ADD COLUMN eol_at timestamptz,
    ADD COLUMN eol_by_person_id uuid REFERENCES people (id),
    ADD CONSTRAINT models_eol_check CHECK ((eol_at IS NULL) = (eol_by_person_id IS NULL));
GRANT UPDATE (eol_at, eol_by_person_id) ON models TO revioten_app;

ALTER TABLE document_revisions
    ADD COLUMN ready_at timestamptz,
    ADD COLUMN ready_by_person_id uuid REFERENCES people (id),
    ADD CONSTRAINT document_revisions_ready_check CHECK ((ready_at IS NULL) = (ready_by_person_id IS NULL));
GRANT UPDATE (ready_at, ready_by_person_id) ON document_revisions TO revioten_app;

ALTER TABLE change_sources
    ADD COLUMN report_key text CHECK (report_key IN ('ecr', 'customer_claim', 'audit_finding', 'periodic_review'));
CREATE UNIQUE INDEX change_sources_report_key ON change_sources (report_key) WHERE report_key IS NOT NULL;
UPDATE change_sources SET report_key = 'ecr' WHERE name = 'ECR' AND mechanism IS NULL;
UPDATE change_sources SET report_key = 'customer_claim' WHERE name = 'Customer claim' AND mechanism IS NULL;
UPDATE change_sources SET report_key = 'audit_finding' WHERE name = 'Audit finding' AND mechanism IS NULL;
UPDATE change_sources SET report_key = 'periodic_review' WHERE name = 'Periodic review' AND mechanism IS NULL;

-- +goose Down
DROP INDEX change_sources_report_key;
ALTER TABLE change_sources DROP COLUMN report_key;
ALTER TABLE document_revisions DROP CONSTRAINT document_revisions_ready_check, DROP COLUMN ready_by_person_id, DROP COLUMN ready_at;
ALTER TABLE models DROP CONSTRAINT models_eol_check, DROP COLUMN eol_by_person_id, DROP COLUMN eol_at;
