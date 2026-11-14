-- D-155 pola 2 · Bagian per PCB dalam satu dokumen (E-93, referensi D-159).
--
-- Satu model bisa terdiri dari beberapa PCB (contoh SC-KT-040-00: PCB A,
-- Heatsink Assy R/L, Base Assy R/L), masing-masing dengan alur proses sendiri,
-- tetapi satu nomor dokumen, satu revisi, dan satu persetujuan.
--
--   document_sections               bagian satu dokumen: nama dan urutan.
--                                   section_key sama untuk bagian yang sama di
--                                   semua dokumen satu set (PFD, PFMEA, CP),
--                                   supaya perakitan dan cek PFMEA-CP berjalan
--                                   per bagian (ASUMSI E-97).
--   document_section_part_numbers   part number setiap bagian, per section_key
--                                   (satu bagian bisa dua part number, R dan L).
--   document_rows.section_id        bagian pemilik baris; kosong untuk dokumen
--                                   tanpa bagian.
--
-- Kunci unik yang dulu per dokumen kini per dokumen dan bagian:
-- satu poin template paling banyak satu baris per bagian, poin yang
-- dikecualikan dicatat per bagian, dan tanda dicek PFMEA-CP per bagian.

-- +goose Up
CREATE TABLE document_sections (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    document_id uuid NOT NULL REFERENCES documents (id),
    section_key uuid NOT NULL,
    position integer NOT NULL CHECK (position >= 1),
    name text NOT NULL CHECK (name <> '' AND name = btrim(name) AND char_length(name) <= 200),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT document_sections_key_key UNIQUE (document_id, section_key),
    CONSTRAINT document_sections_position_key UNIQUE (document_id, position) DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT document_sections_name_key UNIQUE (document_id, name)
);
GRANT SELECT, INSERT, DELETE ON document_sections TO revioten_app;
GRANT UPDATE (name, position, updated_at) ON document_sections TO revioten_app;

CREATE TABLE document_section_part_numbers (
    section_key uuid NOT NULL,
    part_number_id uuid NOT NULL REFERENCES part_numbers (id),
    PRIMARY KEY (section_key, part_number_id)
);
GRANT SELECT, INSERT, DELETE ON document_section_part_numbers TO revioten_app;

ALTER TABLE document_rows ADD COLUMN section_id uuid REFERENCES document_sections (id);
GRANT UPDATE (section_id) ON document_rows TO revioten_app;

DROP INDEX document_rows_point_key;
CREATE UNIQUE INDEX document_rows_point_key
    ON document_rows (document_id, section_id, template_point_id) NULLS NOT DISTINCT
    WHERE template_point_id IS NOT NULL;

ALTER TABLE document_point_exclusions
    ADD COLUMN section_id uuid REFERENCES document_sections (id),
    DROP CONSTRAINT document_point_exclusions_pkey;
CREATE UNIQUE INDEX document_point_exclusions_key
    ON document_point_exclusions (document_id, section_id, point_id) NULLS NOT DISTINCT;

ALTER TABLE pfmea_cp_checks
    ADD COLUMN section_key uuid,
    DROP CONSTRAINT pfmea_cp_checks_pkey;
CREATE UNIQUE INDEX pfmea_cp_checks_key
    ON pfmea_cp_checks (pfmea_document_id, cp_document_id, section_key, process_id, pfmea_revision, cp_revision) NULLS NOT DISTINCT;

-- +goose Down
DROP INDEX pfmea_cp_checks_key;
ALTER TABLE pfmea_cp_checks
    DROP COLUMN section_key,
    ADD PRIMARY KEY (pfmea_document_id, cp_document_id, process_id, pfmea_revision, cp_revision);
DROP INDEX document_point_exclusions_key;
ALTER TABLE document_point_exclusions
    DROP COLUMN section_id,
    ADD PRIMARY KEY (document_id, point_id);
DROP INDEX document_rows_point_key;
CREATE UNIQUE INDEX document_rows_point_key ON document_rows (document_id, template_point_id) WHERE template_point_id IS NOT NULL;
REVOKE UPDATE (section_id) ON document_rows FROM revioten_app;
ALTER TABLE document_rows DROP COLUMN section_id;
DROP TABLE document_section_part_numbers;
DROP TABLE document_sections;
