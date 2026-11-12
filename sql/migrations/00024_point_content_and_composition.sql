-- M4.5 · Isi poin per kolom (D-146) dan perakitan dokumen (R-CMP-01 sampai R-CMP-04).
--
--   process_template_points.content    isi poin per kolom format, dengan key
--   process_template_variants.content  tetap kolom sebagai kuncinya, misalnya
--                                      {"failure_mode": "...", "severity": 7}.
--                                      Kolom text tetap ada sebagai judul poin.
--   format_fields.key                  kolom form AIAG 4th (isi awal) diberi key,
--                                      sama dengan kolom sejenis di form AIAG-VDA,
--                                      supaya satu poin melayani kedua form set.
--   document_rows.template_point_id    baris hasil perakitan tertaut ke poin
--   document_rows.template_version_id  asalnya dan versi template yang dipakai.
--   document_point_exclusions          poin yang tidak dicentang, dengan alasan
--                                      opsional (R-CMP-03).
--
-- Menambah kolom dengan DEFAULT tidak menjalankan trigger baris, jadi trigger
-- penjaga versi terbit (migrasi 00016) tidak perlu dimatikan.

-- +goose Up
ALTER TABLE process_template_points
    ADD COLUMN content jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD CONSTRAINT process_template_points_content_check CHECK (jsonb_typeof(content) = 'object');
ALTER TABLE process_template_variants
    ADD COLUMN content jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD CONSTRAINT process_template_variants_content_check CHECK (jsonb_typeof(content) = 'object');
GRANT UPDATE (content) ON process_template_points TO revioten_app;
GRANT UPDATE (content) ON process_template_variants TO revioten_app;

-- Key kolom form isi awal. Kolom dicari lewat nama template dan label di
-- versi 1; template yang sudah diganti namanya dilewati (key tetap kosong).
-- ASUMSI E-92: pasangan kolom yang dianggap sama antara dua form set.
UPDATE format_fields f
SET key = m.key
FROM (VALUES
    ('PFMEA AIAG 4th', 'Requirement', 'requirement'),
    ('PFMEA AIAG 4th', 'Potential Failure Mode', 'failure_mode'),
    ('PFMEA AIAG 4th', 'Potential Effect(s) of Failure', 'failure_effect'),
    ('PFMEA AIAG 4th', 'Severity', 'severity'),
    ('PFMEA AIAG 4th', 'Class', 'special_characteristic'),
    ('PFMEA AIAG 4th', 'Potential Cause(s) / Mechanism(s) of Failure', 'failure_cause'),
    ('PFMEA AIAG 4th', 'Current Process Controls Prevention', 'prevention_control'),
    ('PFMEA AIAG 4th', 'Occurrence', 'occurrence'),
    ('PFMEA AIAG 4th', 'Current Process Controls Detection', 'detection_control'),
    ('PFMEA AIAG 4th', 'Detection', 'detection'),
    ('PFMEA AIAG 4th', 'Recommended Action', 'recommended_action'),
    ('PFMEA AIAG 4th', 'Responsibility & Target Completion Date', 'responsibility_target'),
    ('PFMEA AIAG 4th', 'Actions Taken & Completion Date', 'action_taken'),
    ('PFMEA AIAG 4th', 'Result Severity', 'severity_after'),
    ('PFMEA AIAG 4th', 'Result Occurrence', 'occurrence_after'),
    ('PFMEA AIAG 4th', 'Result Detection', 'detection_after'),
    ('CP', 'Machine, Equipment, Jigs, Fixture, Tools for Mfg', 'machine_tool'),
    ('CP', 'Product Characteristics', 'product_characteristic'),
    ('CP', 'Product Specification / Tolerance', 'product_specification'),
    ('CP', 'Process Characteristics', 'process_characteristic'),
    ('CP', 'Process Specification / Tolerance', 'process_specification'),
    ('CP', 'Special Characteristics', 'special_char_class'),
    ('CP', 'Equipment', 'evaluation_technique'),
    ('CP', 'Method', 'control_method'),
    ('CP', 'Frequency', 'sample_frequency'),
    ('CP', 'PIC', 'owner'),
    ('CP', 'Relevant Document', 'relevant_document'),
    ('CP', 'Pokayoke', 'pokayoke'),
    ('CP', 'Reaction Plan', 'reaction_plan'),
    ('PFD', 'Flow Symbol', 'flow_symbol'),
    ('PFD', 'NG Disposition', 'ng_disposition')
) AS m (template, label, key)
JOIN format_templates t ON t.name = m.template
JOIN format_template_versions v ON v.template_id = t.id AND v.version = 1
JOIN format_version_fields vf ON vf.version_id = v.id AND vf.label = m.label AND vf.area = 'row'
WHERE f.id = vf.field_id AND f.key IS NULL
  AND NOT EXISTS (SELECT 1 FROM format_fields o WHERE o.template_id = f.template_id AND o.area = f.area AND o.key = m.key);

ALTER TABLE document_rows
    ADD COLUMN template_point_id uuid REFERENCES template_points (id),
    ADD COLUMN template_version_id uuid REFERENCES process_template_versions (id),
    ADD CONSTRAINT document_rows_link_check CHECK ((template_point_id IS NULL) = (template_version_id IS NULL));
-- Satu poin paling banyak menjadi satu baris dalam satu dokumen.
CREATE UNIQUE INDEX document_rows_point_key ON document_rows (document_id, template_point_id) WHERE template_point_id IS NOT NULL;

CREATE TABLE document_point_exclusions (
    document_id uuid NOT NULL REFERENCES documents (id),
    point_id    uuid NOT NULL REFERENCES template_points (id),
    reason      text CHECK (reason IS NULL OR (reason <> '' AND reason = btrim(reason) AND char_length(reason) <= 500)),
    excluded_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (document_id, point_id)
);
GRANT SELECT, INSERT, UPDATE, DELETE ON document_point_exclusions TO revioten_app;

-- +goose Down
DROP TABLE document_point_exclusions;
DROP INDEX document_rows_point_key;
ALTER TABLE document_rows
    DROP CONSTRAINT document_rows_link_check,
    DROP COLUMN template_version_id,
    DROP COLUMN template_point_id;
UPDATE format_fields f SET key = NULL
FROM format_templates t
WHERE t.id = f.template_id AND t.name IN ('PFMEA AIAG 4th', 'CP', 'PFD');
REVOKE UPDATE (content) ON process_template_variants FROM revioten_app;
REVOKE UPDATE (content) ON process_template_points FROM revioten_app;
ALTER TABLE process_template_variants DROP CONSTRAINT process_template_variants_content_check, DROP COLUMN content;
ALTER TABLE process_template_points DROP CONSTRAINT process_template_points_content_check, DROP COLUMN content;
