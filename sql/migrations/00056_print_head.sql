-- Bagian C (E-144): cover dan kepala lembar seperti dokumen Tokairika TR-020.
--
-- 1. Nama perusahaan dan logo untuk cetakan, diatur admin (Admin → Settings).
--    Logo kosong berarti logo bawaan (logo SIIX dari Excel referensi, dibundel
--    di binary). Isinya PNG; layanan memeriksa tanda tangan file dan ukurannya.
-- 2. Field header bawaan diberi key tetap, supaya cetakan menemukan isinya
--    walau labelnya diganti: process_line, core_team, process_responsibility,
--    key_date. Label tetap hanya tampilan (R-TPL-02).
-- 3. Field header Remark untuk PFMEA v1 dan Control Plan v1, dan judul grup
--    kolom seperti referensi (PROCESS, CHARACTERISTICS ( Key Parameter ),
--    CONTROL METHOD; Current Process, Action Results). Hanya versi draft yang
--    diubah: versi terbit tidak bisa diubah (M3.4). Template yang versinya
--    sudah terbit mendapat isian ini lewat versi baru di Format templates.

-- +goose Up
ALTER TABLE system_settings
    ADD COLUMN company_name text NOT NULL DEFAULT 'PT. SIIX EMS INDONESIA'
        CHECK (company_name = btrim(company_name) AND char_length(company_name) <= 200),
    ADD COLUMN company_logo bytea CHECK (company_logo IS NULL OR octet_length(company_logo) <= 1048576),
    ADD COLUMN company_logo_sha256 text,
    ADD COLUMN company_logo_updated_at timestamptz,
    ADD COLUMN company_logo_updated_by_person_id uuid REFERENCES people (id),
    ADD CONSTRAINT system_settings_logo_hash CHECK ((company_logo IS NULL) = (company_logo_sha256 IS NULL));
GRANT UPDATE (company_name, company_logo, company_logo_sha256, company_logo_updated_at, company_logo_updated_by_person_id)
    ON system_settings TO revioten_app;

-- Key field header bawaan.
UPDATE format_fields f
SET key = m.key
FROM (VALUES
    ('PFD v1', 'Process or Line', 'process_line'),
    ('PFMEA v1', 'Process or Line', 'process_line'),
    ('PFMEA v1', 'Process Responsibility', 'process_responsibility'),
    ('PFMEA v1', 'Key Date', 'key_date'),
    ('PFMEA v1', 'Core Team', 'core_team'),
    ('Control Plan v1', 'Process or Line', 'process_line'),
    ('Control Plan v1', 'Core Team', 'core_team')
) AS m (template, label, key)
JOIN format_templates t ON t.name = m.template AND t.customer_id IS NULL
JOIN format_template_versions v ON v.template_id = t.id AND v.version = 1
JOIN format_version_fields vf ON vf.version_id = v.id AND vf.label = m.label AND vf.area = 'header'
WHERE f.id = vf.field_id AND f.key IS NULL
  AND NOT EXISTS (SELECT 1 FROM format_fields o WHERE o.template_id = f.template_id AND o.area = f.area AND o.key = m.key);

-- +goose StatementBegin
DO $$
DECLARE
    v record;
    f_id uuid;
BEGIN
    -- Field Remark di versi draft terakhir PFMEA v1 dan Control Plan v1.
    FOR v IN
        SELECT tv.id, tv.template_id
        FROM format_template_versions tv
        JOIN format_templates t ON t.id = tv.template_id
        WHERE t.name IN ('PFMEA v1', 'Control Plan v1') AND t.customer_id IS NULL AND tv.status = 'draft'
          AND NOT EXISTS (SELECT 1 FROM format_fields o WHERE o.template_id = t.id AND o.area = 'header' AND o.key = 'remark')
    LOOP
        INSERT INTO format_fields (template_id, area, kind, key)
        VALUES (v.template_id, 'header', 'text', 'remark')
        RETURNING id INTO f_id;
        INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position)
        SELECT v.id, f_id, v.template_id, 'header', 'Remark', COALESCE(max(position), 0) + 1
        FROM format_version_fields WHERE version_id = v.id AND area = 'header';
    END LOOP;
END $$;
-- +goose StatementEnd

-- Judul grup kolom seperti SC-TR-020-04 dan SF-TR-020-03 (versi draft saja).
UPDATE format_version_fields vf
SET group_label = m.grp
FROM (VALUES
    ('Control Plan v1', 'operation_number', NULL, 'PROCESS'),
    ('Control Plan v1', 'process', NULL, 'PROCESS'),
    ('Control Plan v1', NULL, 'product_characteristic', 'CHARACTERISTICS ( Key Parameter )'),
    ('Control Plan v1', NULL, 'product_specification', 'CHARACTERISTICS ( Key Parameter )'),
    ('Control Plan v1', NULL, 'process_characteristic', 'CHARACTERISTICS ( Key Parameter )'),
    ('Control Plan v1', NULL, 'process_specification', 'CHARACTERISTICS ( Key Parameter )'),
    ('Control Plan v1', NULL, 'evaluation_technique', 'CONTROL METHOD'),
    ('Control Plan v1', NULL, 'control_method', 'CONTROL METHOD'),
    ('Control Plan v1', NULL, 'sample_frequency', 'CONTROL METHOD'),
    ('Control Plan v1', NULL, 'owner', 'CONTROL METHOD'),
    ('Control Plan v1', NULL, 'relevant_document', 'CONTROL METHOD'),
    ('PFMEA v1', NULL, 'prevention_control', 'Current Process'),
    ('PFMEA v1', NULL, 'occurrence', 'Current Process'),
    ('PFMEA v1', NULL, 'detection_control', 'Current Process'),
    ('PFMEA v1', NULL, 'detection', 'Current Process'),
    ('PFMEA v1', NULL, 'action_taken', 'Action Results'),
    ('PFMEA v1', NULL, 'severity_after', 'Action Results'),
    ('PFMEA v1', NULL, 'occurrence_after', 'Action Results'),
    ('PFMEA v1', NULL, 'detection_after', 'Action Results')
) AS m (template, kind, key, grp)
JOIN format_templates t ON t.name = m.template AND t.customer_id IS NULL
JOIN format_template_versions tv ON tv.template_id = t.id AND tv.status = 'draft'
JOIN format_fields f ON f.template_id = t.id AND f.area = 'row' AND (f.key = m.key OR (m.key IS NULL AND f.kind = m.kind))
WHERE vf.version_id = tv.id AND vf.field_id = f.id AND vf.group_label IS NULL;

-- Result RPN ikut grup Action Results (kind rpn, set hasil).
UPDATE format_version_fields vf
SET group_label = 'Action Results'
FROM format_templates t, format_template_versions tv, format_fields f
WHERE t.name = 'PFMEA v1' AND t.customer_id IS NULL AND tv.template_id = t.id AND tv.status = 'draft'
  AND f.template_id = t.id AND f.kind = 'rpn' AND f.rating_set = 2
  AND vf.version_id = tv.id AND vf.field_id = f.id AND vf.group_label IS NULL;

-- +goose Down
REVOKE UPDATE (company_name, company_logo, company_logo_sha256, company_logo_updated_at, company_logo_updated_by_person_id)
    ON system_settings FROM revioten_app;
ALTER TABLE system_settings
    DROP CONSTRAINT system_settings_logo_hash,
    DROP COLUMN company_logo_updated_by_person_id,
    DROP COLUMN company_logo_updated_at,
    DROP COLUMN company_logo_sha256,
    DROP COLUMN company_logo,
    DROP COLUMN company_name;
