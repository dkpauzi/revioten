-- Form set langkah B dan C (D-133, D-135, D-140): form PFMEA AIAG-VDA 2019
-- dan Control Plan AIAG 2024.
--
-- Yang baru di skema:
--   format_fields.key              key tetap sebuah field, misalnya failure_mode.
--                                  Dipakai untuk memetakan kolom antar form
--                                  (AIAG 4th ke AIAG-VDA) dan untuk aturan
--                                  yang menunjuk kolom tertentu. Label tetap
--                                  hanya tampilan. Field lama tidak punya key.
--   format_version_fields.group_label
--                                  judul grup di atas kolom, misalnya
--                                  "FAILURE ANALYSIS (STEP 4)".
--   jenis field 'ap'               Action Priority, dihitung dari tabel AP
--                                  versi itu (migrasi 00017); set 1 atau 2.
--   jenis field 'document_number'  nomor dokumen dari register, field header
--                                  yang tidak diketik.
--   metodologi 'aiag_cp_2024'      form Control Plan manual AIAG 2024.
--
-- Kedua form dibuat sebagai versi 1 berstatus draft, seperti form lama
-- (migrasi 00011), supaya admin bisa menyesuaikan labelnya sebelum terbit.

-- +goose Up
ALTER TABLE format_fields
    ADD COLUMN key text,
    ADD CONSTRAINT format_fields_key_check CHECK (key IS NULL OR key ~ '^[a-z][a-z0-9_]{0,49}$');
CREATE UNIQUE INDEX format_fields_key_key ON format_fields (template_id, area, key) WHERE key IS NOT NULL;

ALTER TABLE format_fields DROP CONSTRAINT format_fields_kind_check;
ALTER TABLE format_fields ADD CONSTRAINT format_fields_kind_check CHECK (
    kind IN ('text', 'date', 'select', 'symbol', 'rating', 'rpn', 'ap',
             'operation_number', 'process', 'process_flow', 'document_number')
);
ALTER TABLE format_fields DROP CONSTRAINT format_fields_rating_check;
ALTER TABLE format_fields ADD CONSTRAINT format_fields_rating_check CHECK (
    (kind = 'rating') = (rating_scale IS NOT NULL)
    AND (kind IN ('rating', 'rpn', 'ap')) = (rating_set IS NOT NULL)
    AND (rating_scale IS NULL OR rating_scale IN ('severity', 'occurrence', 'detection'))
    AND (rating_set IS NULL OR rating_set IN (1, 2))
    AND (kind NOT IN ('rating', 'rpn', 'ap') OR area = 'row')
);
ALTER TABLE format_fields DROP CONSTRAINT format_fields_filled_check;
ALTER TABLE format_fields ADD CONSTRAINT format_fields_filled_check CHECK (
    (kind NOT IN ('operation_number', 'process', 'process_flow') OR area = 'row')
    AND (kind <> 'document_number' OR area = 'header')
);

ALTER TABLE format_version_fields
    ADD COLUMN group_label text,
    ADD CONSTRAINT format_version_fields_group_check CHECK (
        group_label IS NULL OR (group_label <> '' AND group_label = btrim(group_label) AND char_length(group_label) <= 100)
    );

ALTER TABLE format_templates DROP CONSTRAINT format_templates_methodology_check;
ALTER TABLE format_templates ADD CONSTRAINT format_templates_methodology_check
    CHECK (methodology IN ('aiag_4th', 'aiag_vda', 'sqam', 'aiag_cp_2024'));

-- +goose StatementBegin
DO $$
DECLARE
    t_id uuid;
    v_id uuid;
    f_id uuid;
    f record;
BEGIN
    -- PFMEA AIAG-VDA 2019 (Action Priority).
    INSERT INTO format_templates (name, document_type_id, methodology)
    SELECT 'PFMEA AIAG-VDA 2019', id, 'aiag_vda' FROM document_types WHERE key = 'pfmea'
    RETURNING id INTO t_id;
    INSERT INTO format_template_versions (template_id, version, risk_method, highlight_severity_at_least, ap_rules)
    VALUES (t_id, 1, 'ap', NULL, '[{"s":[9,10],"o":[8,10],"d":[7,10],"ap":"H"},{"s":[9,10],"o":[8,10],"d":[5,6],"ap":"H"},{"s":[9,10],"o":[8,10],"d":[2,4],"ap":"H"},{"s":[9,10],"o":[8,10],"d":[1,1],"ap":"H"},{"s":[9,10],"o":[6,7],"d":[7,10],"ap":"H"},{"s":[9,10],"o":[6,7],"d":[5,6],"ap":"H"},{"s":[9,10],"o":[6,7],"d":[2,4],"ap":"H"},{"s":[9,10],"o":[6,7],"d":[1,1],"ap":"H"},{"s":[9,10],"o":[4,5],"d":[7,10],"ap":"H"},{"s":[9,10],"o":[4,5],"d":[5,6],"ap":"H"},{"s":[9,10],"o":[4,5],"d":[2,4],"ap":"H"},{"s":[9,10],"o":[4,5],"d":[1,1],"ap":"M"},{"s":[9,10],"o":[2,3],"d":[7,10],"ap":"H"},{"s":[9,10],"o":[2,3],"d":[5,6],"ap":"M"},{"s":[9,10],"o":[2,3],"d":[2,4],"ap":"L"},{"s":[9,10],"o":[2,3],"d":[1,1],"ap":"L"},{"s":[9,10],"o":[1,1],"d":[7,10],"ap":"L"},{"s":[9,10],"o":[1,1],"d":[5,6],"ap":"L"},{"s":[9,10],"o":[1,1],"d":[2,4],"ap":"L"},{"s":[9,10],"o":[1,1],"d":[1,1],"ap":"L"},{"s":[7,8],"o":[8,10],"d":[7,10],"ap":"H"},{"s":[7,8],"o":[8,10],"d":[5,6],"ap":"H"},{"s":[7,8],"o":[8,10],"d":[2,4],"ap":"H"},{"s":[7,8],"o":[8,10],"d":[1,1],"ap":"H"},{"s":[7,8],"o":[6,7],"d":[7,10],"ap":"H"},{"s":[7,8],"o":[6,7],"d":[5,6],"ap":"H"},{"s":[7,8],"o":[6,7],"d":[2,4],"ap":"H"},{"s":[7,8],"o":[6,7],"d":[1,1],"ap":"M"},{"s":[7,8],"o":[4,5],"d":[7,10],"ap":"H"},{"s":[7,8],"o":[4,5],"d":[5,6],"ap":"M"},{"s":[7,8],"o":[4,5],"d":[2,4],"ap":"M"},{"s":[7,8],"o":[4,5],"d":[1,1],"ap":"M"},{"s":[7,8],"o":[2,3],"d":[7,10],"ap":"M"},{"s":[7,8],"o":[2,3],"d":[5,6],"ap":"M"},{"s":[7,8],"o":[2,3],"d":[2,4],"ap":"L"},{"s":[7,8],"o":[2,3],"d":[1,1],"ap":"L"},{"s":[7,8],"o":[1,1],"d":[7,10],"ap":"L"},{"s":[7,8],"o":[1,1],"d":[5,6],"ap":"L"},{"s":[7,8],"o":[1,1],"d":[2,4],"ap":"L"},{"s":[7,8],"o":[1,1],"d":[1,1],"ap":"L"},{"s":[4,6],"o":[8,10],"d":[7,10],"ap":"H"},{"s":[4,6],"o":[8,10],"d":[5,6],"ap":"H"},{"s":[4,6],"o":[8,10],"d":[2,4],"ap":"M"},{"s":[4,6],"o":[8,10],"d":[1,1],"ap":"M"},{"s":[4,6],"o":[6,7],"d":[7,10],"ap":"M"},{"s":[4,6],"o":[6,7],"d":[5,6],"ap":"M"},{"s":[4,6],"o":[6,7],"d":[2,4],"ap":"M"},{"s":[4,6],"o":[6,7],"d":[1,1],"ap":"L"},{"s":[4,6],"o":[4,5],"d":[7,10],"ap":"M"},{"s":[4,6],"o":[4,5],"d":[5,6],"ap":"L"},{"s":[4,6],"o":[4,5],"d":[2,4],"ap":"L"},{"s":[4,6],"o":[4,5],"d":[1,1],"ap":"L"},{"s":[4,6],"o":[2,3],"d":[7,10],"ap":"L"},{"s":[4,6],"o":[2,3],"d":[5,6],"ap":"L"},{"s":[4,6],"o":[2,3],"d":[2,4],"ap":"L"},{"s":[4,6],"o":[2,3],"d":[1,1],"ap":"L"},{"s":[4,6],"o":[1,1],"d":[7,10],"ap":"L"},{"s":[4,6],"o":[1,1],"d":[5,6],"ap":"L"},{"s":[4,6],"o":[1,1],"d":[2,4],"ap":"L"},{"s":[4,6],"o":[1,1],"d":[1,1],"ap":"L"},{"s":[2,3],"o":[8,10],"d":[7,10],"ap":"M"},{"s":[2,3],"o":[8,10],"d":[5,6],"ap":"M"},{"s":[2,3],"o":[8,10],"d":[2,4],"ap":"L"},{"s":[2,3],"o":[8,10],"d":[1,1],"ap":"L"},{"s":[2,3],"o":[6,7],"d":[7,10],"ap":"L"},{"s":[2,3],"o":[6,7],"d":[5,6],"ap":"L"},{"s":[2,3],"o":[6,7],"d":[2,4],"ap":"L"},{"s":[2,3],"o":[6,7],"d":[1,1],"ap":"L"},{"s":[2,3],"o":[4,5],"d":[7,10],"ap":"L"},{"s":[2,3],"o":[4,5],"d":[5,6],"ap":"L"},{"s":[2,3],"o":[4,5],"d":[2,4],"ap":"L"},{"s":[2,3],"o":[4,5],"d":[1,1],"ap":"L"},{"s":[2,3],"o":[2,3],"d":[7,10],"ap":"L"},{"s":[2,3],"o":[2,3],"d":[5,6],"ap":"L"},{"s":[2,3],"o":[2,3],"d":[2,4],"ap":"L"},{"s":[2,3],"o":[2,3],"d":[1,1],"ap":"L"},{"s":[2,3],"o":[1,1],"d":[7,10],"ap":"L"},{"s":[2,3],"o":[1,1],"d":[5,6],"ap":"L"},{"s":[2,3],"o":[1,1],"d":[2,4],"ap":"L"},{"s":[2,3],"o":[1,1],"d":[1,1],"ap":"L"},{"s":[1,1],"o":[1,10],"d":[7,10],"ap":"L"},{"s":[1,1],"o":[1,10],"d":[5,6],"ap":"L"},{"s":[1,1],"o":[1,10],"d":[2,4],"ap":"L"},{"s":[1,1],"o":[1,10],"d":[1,1],"ap":"L"}]'::jsonb)
    RETURNING id INTO v_id;
    FOR f IN
        SELECT * FROM (VALUES
            ('header', 1, 'part_name', 'text', NULL::text, NULL::smallint, 'Part Name', NULL::text, NULL::jsonb),
            ('header', 2, 'company_name', 'text', NULL, NULL, 'Company Name', NULL, NULL),
            ('header', 3, 'manufacturing_location', 'text', NULL, NULL, 'Manufacturing Location', NULL, NULL),
            ('header', 4, 'customer_name', 'text', NULL, NULL, 'Customer Name', NULL, NULL),
            ('header', 5, 'model_year_program', 'text', NULL, NULL, 'Model Year(s)/Program(s)', NULL, NULL),
            ('header', 6, 'part_no', 'text', NULL, NULL, 'Part No.', NULL, NULL),
            ('header', 7, 'pfmea_start_date', 'date', NULL, NULL, 'PFMEA Start Date', NULL, NULL),
            ('header', 8, 'pfmea_revision_date', 'date', NULL, NULL, 'PFMEA Revision Date', NULL, NULL),
            ('header', 9, 'cross_functional_team', 'text', NULL, NULL, 'Cross Functional Team', NULL, NULL),
            ('header', 10, 'pfmea_id_number', 'document_number', NULL, NULL, 'PFMEA ID Number', NULL, NULL),
            ('header', 11, 'process_responsibility', 'text', NULL, NULL, 'Process Responsibility', NULL, NULL),
            ('header', 12, 'confidentiality_level', 'text', NULL, NULL, 'Confidentiality Level', NULL, NULL),
            ('row', 1, 'change_history', 'text', NULL, NULL, 'History/change authorization (As applicable)', NULL, NULL),
            ('row', 2, 'process_item', 'text', NULL, NULL, '1. Process Item', 'STRUCTURE ANALYSIS (STEP 2)', NULL),
            ('row', 3, 'process_step', 'text', NULL, NULL, '2. Process Step', 'STRUCTURE ANALYSIS (STEP 2)', NULL),
            ('row', 4, 'work_element', 'select', NULL, NULL, '3. Process Work Element (4M)', 'STRUCTURE ANALYSIS (STEP 2)',
                '[{"key": "man", "label": "Man"}, {"key": "machine", "label": "Machine"}, {"key": "material", "label": "Material"},
                  {"key": "method", "label": "Method"}, {"key": "measurement", "label": "Measurement"}, {"key": "milieu", "label": "Milieu"}]'::jsonb),
            ('row', 5, 'function_process_item', 'text', NULL, NULL, '1. Function of the Process Item', 'FUNCTION ANALYSIS (STEP 3)', NULL),
            ('row', 6, 'function_process_step', 'text', NULL, NULL, '2. Function of the Process Step and Product Characteristic', 'FUNCTION ANALYSIS (STEP 3)', NULL),
            ('row', 7, 'function_work_element', 'text', NULL, NULL, '3. Function of the Work Element and Process Characteristic', 'FUNCTION ANALYSIS (STEP 3)', NULL),
            ('row', 8, 'failure_effect', 'text', NULL, NULL, '1. Failure Effects (FE)', 'FAILURE ANALYSIS (STEP 4)', NULL),
            ('row', 9, 'severity', 'rating', 'severity', 1, 'Severity (S) of FE', 'FAILURE ANALYSIS (STEP 4)', NULL),
            ('row', 10, 'failure_mode', 'text', NULL, NULL, '2. Failure Mode (FM)', 'FAILURE ANALYSIS (STEP 4)', NULL),
            ('row', 11, 'failure_cause', 'text', NULL, NULL, '3. Failure Cause (FC)', 'FAILURE ANALYSIS (STEP 4)', NULL),
            ('row', 12, 'prevention_control', 'text', NULL, NULL, 'Current Prevention Control (PC) of FC', 'RISK ANALYSIS (STEP 5)', NULL),
            ('row', 13, 'occurrence', 'rating', 'occurrence', 1, 'Occurrence (O) of FC', 'RISK ANALYSIS (STEP 5)', NULL),
            ('row', 14, 'detection_control', 'text', NULL, NULL, 'Current Detection Controls (DC) of FC or FM', 'RISK ANALYSIS (STEP 5)', NULL),
            ('row', 15, 'detection', 'rating', 'detection', 1, 'Detection (D) of FC/FM', 'RISK ANALYSIS (STEP 5)', NULL),
            ('row', 16, 'ap', 'ap', NULL, 1, 'PFMEA AP', 'RISK ANALYSIS (STEP 5)', NULL),
            ('row', 17, 'prevention_action', 'text', NULL, NULL, 'Prevention Action', 'OPTIMIZATION (STEP 6)', NULL),
            ('row', 18, 'detection_action', 'text', NULL, NULL, 'Detection Action', 'OPTIMIZATION (STEP 6)', NULL),
            ('row', 19, 'responsible_person', 'text', NULL, NULL, 'Responsible Person''s Name', 'OPTIMIZATION (STEP 6)', NULL),
            ('row', 20, 'target_completion_date', 'date', NULL, NULL, 'Target Completion Date', 'OPTIMIZATION (STEP 6)', NULL),
            ('row', 21, 'action_status', 'select', NULL, NULL, 'Status', 'OPTIMIZATION (STEP 6)',
                '[{"key": "open", "label": "Open"}, {"key": "completed", "label": "Completed"}, {"key": "discarded", "label": "Discarded"}]'::jsonb),
            ('row', 22, 'action_taken', 'text', NULL, NULL, 'Action Taken with Pointer to Evidence', 'OPTIMIZATION (STEP 6)', NULL),
            ('row', 23, 'completion_date', 'date', NULL, NULL, 'Completion Date', 'OPTIMIZATION (STEP 6)', NULL),
            ('row', 24, 'severity_after', 'rating', 'severity', 2, 'Severity (S)', 'OPTIMIZATION (STEP 6)', NULL),
            ('row', 25, 'occurrence_after', 'rating', 'occurrence', 2, 'Occurrence (O)', 'OPTIMIZATION (STEP 6)', NULL),
            ('row', 26, 'detection_after', 'rating', 'detection', 2, 'Detection (D)', 'OPTIMIZATION (STEP 6)', NULL),
            ('row', 27, 'special_characteristic', 'symbol', NULL, NULL, 'Special Characteristic', 'OPTIMIZATION (STEP 6)', NULL),
            ('row', 28, 'ap_after', 'ap', NULL, 2, 'PFMEA AP (after action)', 'OPTIMIZATION (STEP 6)', NULL),
            ('row', 29, 'note', 'text', NULL, NULL, 'Note', 'OPTIMIZATION (STEP 6)', NULL)
        ) AS v (area, position, key, kind, rating_scale, rating_set, label, group_label, options)
        ORDER BY area, position
    LOOP
        INSERT INTO format_fields (template_id, area, kind, rating_scale, rating_set, key)
        VALUES (t_id, f.area, f.kind, f.rating_scale, f.rating_set, f.key)
        RETURNING id INTO f_id;
        INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position, options, group_label)
        VALUES (v_id, f_id, t_id, f.area, f.label, f.position, f.options, f.group_label);
    END LOOP;
    INSERT INTO format_rating_criteria (version_id, scale, value)
    SELECT v_id, s.scale, n.value
    FROM (VALUES ('severity'), ('occurrence'), ('detection')) AS s (scale)
    CROSS JOIN generate_series(1, 10) AS n (value);
    INSERT INTO format_symbols (version_id, key, name, position, automatic) VALUES
        (v_id, 'critical_process', 'Critical Process', 1, false),
        (v_id, 'revision', 'Revision', 2, true),
        (v_id, 'important_character', 'Important Character', 3, false),
        (v_id, 'regulation', 'Regulation', 4, false);

    -- Control Plan AIAG 2024: ada fase (D-135) dan Owner, tanpa kolom Process Flow.
    INSERT INTO format_templates (name, document_type_id, methodology)
    SELECT 'Control Plan AIAG 2024', id, 'aiag_cp_2024' FROM document_types WHERE key = 'cp'
    RETURNING id INTO t_id;
    INSERT INTO format_template_versions (template_id, version, risk_method)
    VALUES (t_id, 1, 'none')
    RETURNING id INTO v_id;
    FOR f IN
        SELECT * FROM (VALUES
            ('header', 1, 'phase', 'select', 'Phase', NULL::text,
                '[{"key": "prototype", "label": "Prototype"}, {"key": "pre_launch", "label": "Pre-launch"},
                  {"key": "safe_launch", "label": "Safe Launch"}, {"key": "production", "label": "Production"}]'::jsonb),
            ('header', 2, 'control_plan_no', 'document_number', 'Control plan No.', NULL, NULL::jsonb),
            ('header', 3, 'key_contact', 'text', 'Key Contact', NULL, NULL),
            ('header', 4, 'date_orig', 'date', 'Date (orig)', NULL, NULL),
            ('header', 5, 'date_rev', 'date', 'Date (rev)', NULL, NULL),
            ('header', 6, 'customer_name', 'text', 'Customer Name', NULL, NULL),
            ('header', 7, 'core_team', 'text', 'Core Team', NULL, NULL),
            ('header', 8, 'customer_engineering_approval', 'text', 'Customer Engineering Approval/Date (if req''d)', NULL, NULL),
            ('header', 9, 'part_number', 'text', 'Part Number', NULL, NULL),
            ('header', 10, 'organization_plant_approval', 'text', 'Organization/Plant Approval/Date', NULL, NULL),
            ('header', 11, 'customer_quality_approval', 'text', 'Customer Quality Approval/Date (if req''d)', NULL, NULL),
            ('header', 12, 'part_name', 'text', 'Part Name', NULL, NULL),
            ('header', 13, 'other_approval_1', 'text', 'Other Approval/Date (if req''d) 1', NULL, NULL),
            ('header', 14, 'other_approval_2', 'text', 'Other Approval/Date (if req''d) 2', NULL, NULL),
            ('row', 1, 'process_no', 'operation_number', 'No', 'Part/Process Number', NULL),
            ('row', 2, 'process_description', 'process', 'Description', 'Part/Process Number', NULL),
            ('row', 3, 'owner', 'text', 'Owner', 'Part/Process Number', NULL),
            ('row', 4, 'machine_tool', 'text', 'Machine, device, jig or tool for MFG.', NULL, NULL),
            ('row', 5, 'characteristic_no', 'text', 'No.', 'Characteristics', NULL),
            ('row', 6, 'product_characteristic', 'text', 'Product control', 'Characteristics', NULL),
            ('row', 7, 'process_characteristic', 'text', 'Process control', 'Characteristics', NULL),
            ('row', 8, 'special_char_class', 'symbol', 'Special Char. (Class)', NULL, NULL),
            ('row', 9, 'specification_tolerance', 'text', 'Product/Process Specification/Tolerance', 'Methods', NULL),
            ('row', 10, 'evaluation_technique', 'text', 'Evaluation / Measurement technique', 'Methods', NULL),
            ('row', 11, 'sample_size', 'text', 'Sample Size', 'Methods', NULL),
            ('row', 12, 'sample_frequency', 'text', 'Sample Freq', 'Methods', NULL),
            ('row', 13, 'control_method', 'text', 'Control Method', 'Methods', NULL),
            ('row', 14, 'reaction_plan', 'text', 'Reaction Plan', NULL, NULL)
        ) AS v (area, position, key, kind, label, group_label, options)
        ORDER BY area, position
    LOOP
        INSERT INTO format_fields (template_id, area, kind, key)
        VALUES (t_id, f.area, f.kind, f.key)
        RETURNING id INTO f_id;
        INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position, options, group_label)
        VALUES (v_id, f_id, t_id, f.area, f.label, f.position, f.options, f.group_label);
    END LOOP;
    INSERT INTO format_symbols (version_id, key, name, position, automatic) VALUES
        (v_id, 'critical_process', 'Critical Process', 1, false),
        (v_id, 'revision', 'Revision', 2, true),
        (v_id, 'important_character', 'Important Character', 3, false),
        (v_id, 'regulation', 'Regulation', 4, false);
END
$$;
-- +goose StatementEnd

-- Hak role aplikasi: INSERT tingkat tabel sudah mencakup kolom baru. Judul
-- grup belum bisa diubah lewat aplikasi (tidak ada UPDATE untuk group_label).

-- +goose Down
-- Hanya untuk pengembangan: kedua form baru beserta isinya ikut terhapus.
ALTER TABLE format_version_fields DISABLE TRIGGER format_version_fields_published;
ALTER TABLE format_rating_criteria DISABLE TRIGGER format_rating_criteria_published;
ALTER TABLE format_symbols DISABLE TRIGGER format_symbols_published;
ALTER TABLE format_template_versions DISABLE TRIGGER format_template_versions_published;
DELETE FROM format_version_fields WHERE template_id IN (SELECT id FROM format_templates WHERE methodology = 'aiag_cp_2024' OR name = 'PFMEA AIAG-VDA 2019');
DELETE FROM format_rating_criteria WHERE version_id IN (SELECT v.id FROM format_template_versions v JOIN format_templates t ON t.id = v.template_id WHERE t.methodology = 'aiag_cp_2024' OR t.name = 'PFMEA AIAG-VDA 2019');
DELETE FROM format_symbols WHERE version_id IN (SELECT v.id FROM format_template_versions v JOIN format_templates t ON t.id = v.template_id WHERE t.methodology = 'aiag_cp_2024' OR t.name = 'PFMEA AIAG-VDA 2019');
DELETE FROM format_fields WHERE template_id IN (SELECT id FROM format_templates WHERE methodology = 'aiag_cp_2024' OR name = 'PFMEA AIAG-VDA 2019');
DELETE FROM format_template_versions WHERE template_id IN (SELECT id FROM format_templates WHERE methodology = 'aiag_cp_2024' OR name = 'PFMEA AIAG-VDA 2019');
DELETE FROM format_templates WHERE methodology = 'aiag_cp_2024' OR name = 'PFMEA AIAG-VDA 2019';
ALTER TABLE format_template_versions ENABLE TRIGGER format_template_versions_published;
ALTER TABLE format_symbols ENABLE TRIGGER format_symbols_published;
ALTER TABLE format_rating_criteria ENABLE TRIGGER format_rating_criteria_published;
ALTER TABLE format_version_fields ENABLE TRIGGER format_version_fields_published;

ALTER TABLE format_templates DROP CONSTRAINT format_templates_methodology_check;
ALTER TABLE format_templates ADD CONSTRAINT format_templates_methodology_check
    CHECK (methodology IN ('aiag_4th', 'aiag_vda', 'sqam'));
ALTER TABLE format_version_fields DROP COLUMN group_label;
ALTER TABLE format_fields DROP CONSTRAINT format_fields_filled_check;
ALTER TABLE format_fields ADD CONSTRAINT format_fields_filled_check CHECK (
    kind NOT IN ('operation_number', 'process', 'process_flow') OR area = 'row'
);
ALTER TABLE format_fields DROP CONSTRAINT format_fields_rating_check;
ALTER TABLE format_fields ADD CONSTRAINT format_fields_rating_check CHECK (
    (kind = 'rating') = (rating_scale IS NOT NULL)
    AND (kind IN ('rating', 'rpn')) = (rating_set IS NOT NULL)
    AND (rating_scale IS NULL OR rating_scale IN ('severity', 'occurrence', 'detection'))
    AND (rating_set IS NULL OR rating_set IN (1, 2))
    AND (kind NOT IN ('rating', 'rpn') OR area = 'row')
);
ALTER TABLE format_fields DROP CONSTRAINT format_fields_kind_check;
ALTER TABLE format_fields ADD CONSTRAINT format_fields_kind_check CHECK (
    kind IN ('text', 'date', 'select', 'symbol', 'rating', 'rpn', 'operation_number', 'process', 'process_flow')
);
DROP INDEX format_fields_key_key;
ALTER TABLE format_fields DROP COLUMN key;