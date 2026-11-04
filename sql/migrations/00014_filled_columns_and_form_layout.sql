-- Keputusan pemilik 4 November 2026, sesudah tinjauan desain UI:
--   D-122  kolom Process Flow di CP selalu tercetak; isinya dibentuk dari
--          langkah PFD (R-DOC-09), supaya mengubah alur, termasuk konektor di
--          perpindahan halaman, tidak lagi menggambar ulang di Excel;
--   D-123  nama proses dan nomor operasi selalu tampil dan tercetak di PFD,
--          PFMEA, dan CP; isinya diambil dari baris (ID proses master dan
--          nomor operasi), tidak diketik;
--   D-124  susunan kolom isi awal mengikuti form Excel asli (E-70).
--
-- Konsep: "kolom terisi otomatis". Seperti RPN yang dihitung, kolom ini ada di
-- format template supaya terlihat dan tercetak di tempatnya, tetapi isinya
-- tidak pernah diketik:
--   operation_number  nomor operasi baris, misalnya 5.2
--   process           nama proses baris dari master proses
--   process_flow      simbol alur langkah itu, dari PFD
--
-- Migrasi 00013 menyembunyikan Process Flow (CP) dan Process Step / Function
-- (PFMEA), karena isinya bukan ketikan. Migrasi ini memunculkannya lagi sebagai
-- kolom terisi otomatis. Jenis field-nya diubah di sini, sekali, sebagai
-- pemilik database: belum ada baris dokumen yang memakai field itu (dokumen
-- baru datang di M4.1). Role aplikasi tetap tidak bisa mengubah jenis field.

-- +goose Up
ALTER TABLE format_fields DROP CONSTRAINT format_fields_kind_check;
ALTER TABLE format_fields ADD CONSTRAINT format_fields_kind_check CHECK (
    kind IN ('text', 'date', 'select', 'symbol', 'rating', 'rpn', 'operation_number', 'process', 'process_flow')
);
-- Kolom terisi otomatis hanya untuk baris, bukan field header.
ALTER TABLE format_fields ADD CONSTRAINT format_fields_filled_check CHECK (
    kind NOT IN ('operation_number', 'process', 'process_flow') OR area = 'row'
);

-- Isi awal versi 1 yang masih draft. Template yang namanya sudah diganti
-- admin dilewati.
-- +goose StatementBegin
DO $$
DECLARE
    t_id uuid;
    v_id uuid;
    f_id uuid;
BEGIN
    -- PFD: nomor langkah dan proses di depan simbol alur, seperti daftar
    -- langkah PFD (Op no., Process, Symbol, NG goes to).
    SELECT v.id, v.template_id INTO v_id, t_id
    FROM format_template_versions v
    JOIN format_templates t ON t.id = v.template_id
    WHERE t.name = 'PFD' AND v.version = 1 AND v.status = 'draft';
    IF FOUND THEN
        UPDATE format_version_fields SET position = position + 2 WHERE version_id = v_id AND area = 'row';
        INSERT INTO format_fields (template_id, area, kind) VALUES (t_id, 'row', 'operation_number') RETURNING id INTO f_id;
        INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position)
        VALUES (v_id, f_id, t_id, 'row', 'No', 1);
        INSERT INTO format_fields (template_id, area, kind) VALUES (t_id, 'row', 'process') RETURNING id INTO f_id;
        INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position)
        VALUES (v_id, f_id, t_id, 'row', 'Process', 2);
    END IF;

    -- PFMEA: Process Item No. dan Process Step / Function di depan, lalu
    -- label mengikuti form (Requirement, Class, Potential Cause(s) /
    -- Mechanism(s) of Failure, Recommended Action).
    SELECT v.id, v.template_id INTO v_id, t_id
    FROM format_template_versions v
    JOIN format_templates t ON t.id = v.template_id
    WHERE t.name = 'PFMEA AIAG 4th' AND v.version = 1 AND v.status = 'draft';
    IF FOUND THEN
        UPDATE format_version_fields SET position = position + 1 WHERE version_id = v_id AND area = 'row';
        INSERT INTO format_fields (template_id, area, kind) VALUES (t_id, 'row', 'operation_number') RETURNING id INTO f_id;
        INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position)
        VALUES (v_id, f_id, t_id, 'row', 'Process Item No.', 1);
        UPDATE format_fields f
        SET kind = 'process'
        FROM format_version_fields vf
        WHERE vf.field_id = f.id AND vf.version_id = v_id AND vf.area = 'row'
          AND vf.label = 'Process Step / Function' AND f.kind = 'text';
        UPDATE format_version_fields vf
        SET label = m.new_label, hidden = false
        FROM (VALUES
            ('Process Step / Function', 'Process Step / Function'),
            ('Requirements', 'Requirement'),
            ('Classification', 'Class'),
            ('Potential Cause(s) of Failure', 'Potential Cause(s) / Mechanism(s) of Failure'),
            ('Recommended Action(s)', 'Recommended Action')
        ) AS m (old_label, new_label)
        WHERE vf.version_id = v_id AND vf.area = 'row' AND vf.label = m.old_label;
    END IF;

    -- CP: Process Flow, No, dan Process Name / Operation Description di
    -- depan; karakteristik produk dan proses masing-masing dengan
    -- Specification / Tolerance; label mengikuti form.
    SELECT v.id, v.template_id INTO v_id, t_id
    FROM format_template_versions v
    JOIN format_templates t ON t.id = v.template_id
    WHERE t.name = 'CP' AND v.version = 1 AND v.status = 'draft';
    IF FOUND THEN
        UPDATE format_fields f
        SET kind = 'process_flow'
        FROM format_version_fields vf
        WHERE vf.field_id = f.id AND vf.version_id = v_id AND vf.area = 'row'
          AND vf.label = 'Process Flow' AND f.kind = 'text';
        UPDATE format_version_fields vf
        SET label = m.new_label, position = m.pos, hidden = false
        FROM (VALUES
            ('Process Flow', 'Process Flow', 1),
            ('Machine and Tools', 'Machine, Equipment, Jigs, Fixture, Tools for Mfg', 4),
            ('Product Characteristic', 'Product Characteristics', 5),
            ('Specification', 'Product Specification / Tolerance', 6),
            ('Process Characteristic', 'Process Characteristics', 7),
            ('Special Characteristic', 'Special Characteristics', 9),
            ('Equipment', 'Equipment', 10),
            ('Method', 'Method', 11),
            ('Frequency', 'Frequency', 12),
            ('PIC', 'PIC', 13),
            ('Relevant Document', 'Relevant Document', 14),
            ('Pokayoke', 'Pokayoke', 15),
            ('Reaction Plan', 'Reaction Plan', 16)
        ) AS m (old_label, new_label, pos)
        WHERE vf.version_id = v_id AND vf.area = 'row' AND vf.label = m.old_label;
        INSERT INTO format_fields (template_id, area, kind) VALUES (t_id, 'row', 'operation_number') RETURNING id INTO f_id;
        INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position)
        VALUES (v_id, f_id, t_id, 'row', 'No', 2);
        INSERT INTO format_fields (template_id, area, kind) VALUES (t_id, 'row', 'process') RETURNING id INTO f_id;
        INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position)
        VALUES (v_id, f_id, t_id, 'row', 'Process Name / Operation Description', 3);
        INSERT INTO format_fields (template_id, area, kind) VALUES (t_id, 'row', 'text') RETURNING id INTO f_id;
        INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position)
        VALUES (v_id, f_id, t_id, 'row', 'Process Specification / Tolerance', 8);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- Mengembalikan isi awal ke keadaan sesudah migrasi 00013. Hanya untuk
-- database pengembangan: bila admin sudah membuat kolom dengan jenis baru,
-- langkah terakhir gagal, dan memang sebaiknya begitu.
-- +goose StatementBegin
DO $$
DECLARE
    v_id uuid;
BEGIN
    SELECT v.id INTO v_id
    FROM format_template_versions v
    JOIN format_templates t ON t.id = v.template_id
    WHERE t.name = 'PFD' AND v.version = 1 AND v.status = 'draft';
    IF FOUND THEN
        WITH gone AS (
            DELETE FROM format_version_fields
            WHERE version_id = v_id AND area = 'row' AND label IN ('No', 'Process')
            RETURNING field_id
        )
        DELETE FROM format_fields WHERE id IN (SELECT field_id FROM gone);
        UPDATE format_version_fields SET position = position - 2 WHERE version_id = v_id AND area = 'row';
    END IF;

    SELECT v.id INTO v_id
    FROM format_template_versions v
    JOIN format_templates t ON t.id = v.template_id
    WHERE t.name = 'PFMEA AIAG 4th' AND v.version = 1 AND v.status = 'draft';
    IF FOUND THEN
        WITH gone AS (
            DELETE FROM format_version_fields
            WHERE version_id = v_id AND area = 'row' AND label = 'Process Item No.'
            RETURNING field_id
        )
        DELETE FROM format_fields WHERE id IN (SELECT field_id FROM gone);
        UPDATE format_version_fields SET position = position - 1 WHERE version_id = v_id AND area = 'row';
        UPDATE format_fields f
        SET kind = 'text'
        FROM format_version_fields vf
        WHERE vf.field_id = f.id AND vf.version_id = v_id AND vf.label = 'Process Step / Function' AND f.kind = 'process';
        UPDATE format_version_fields vf
        SET label = m.old_label, hidden = (m.old_label = 'Process Step / Function')
        FROM (VALUES
            ('Process Step / Function', 'Process Step / Function'),
            ('Requirements', 'Requirement'),
            ('Classification', 'Class'),
            ('Potential Cause(s) of Failure', 'Potential Cause(s) / Mechanism(s) of Failure'),
            ('Recommended Action(s)', 'Recommended Action')
        ) AS m (old_label, new_label)
        WHERE vf.version_id = v_id AND vf.area = 'row' AND vf.label = m.new_label;
    END IF;

    SELECT v.id INTO v_id
    FROM format_template_versions v
    JOIN format_templates t ON t.id = v.template_id
    WHERE t.name = 'CP' AND v.version = 1 AND v.status = 'draft';
    IF FOUND THEN
        WITH gone AS (
            DELETE FROM format_version_fields
            WHERE version_id = v_id AND area = 'row'
              AND label IN ('No', 'Process Name / Operation Description', 'Process Specification / Tolerance')
            RETURNING field_id
        )
        DELETE FROM format_fields WHERE id IN (SELECT field_id FROM gone);
        UPDATE format_fields f
        SET kind = 'text'
        FROM format_version_fields vf
        WHERE vf.field_id = f.id AND vf.version_id = v_id AND vf.label = 'Process Flow' AND f.kind = 'process_flow';
        UPDATE format_version_fields vf
        SET label = m.old_label, position = m.pos, hidden = (m.old_label = 'Process Flow')
        FROM (VALUES
            ('Process Flow', 'Process Flow', 1),
            ('Machine and Tools', 'Machine, Equipment, Jigs, Fixture, Tools for Mfg', 2),
            ('Product Characteristic', 'Product Characteristics', 3),
            ('Process Characteristic', 'Process Characteristics', 4),
            ('Specification', 'Product Specification / Tolerance', 5),
            ('Special Characteristic', 'Special Characteristics', 6),
            ('Equipment', 'Equipment', 7),
            ('Method', 'Method', 8),
            ('Frequency', 'Frequency', 9),
            ('PIC', 'PIC', 10),
            ('Relevant Document', 'Relevant Document', 11),
            ('Pokayoke', 'Pokayoke', 12),
            ('Reaction Plan', 'Reaction Plan', 13)
        ) AS m (old_label, new_label, pos)
        WHERE vf.version_id = v_id AND vf.area = 'row' AND vf.label = m.new_label;
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE format_fields DROP CONSTRAINT format_fields_filled_check;
ALTER TABLE format_fields DROP CONSTRAINT format_fields_kind_check;
ALTER TABLE format_fields ADD CONSTRAINT format_fields_kind_check CHECK (
    kind IN ('text', 'date', 'select', 'symbol', 'rating', 'rpn')
);
