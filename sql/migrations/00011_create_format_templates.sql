-- Format template (Tahap 3, M3.1, UC-20, R-TPL-01, R-TPL-02, R-TPL-11).
-- Konsep: struktur dokumen sebagai data yang berversi. Setiap field (kolom
-- baris dan field header) punya ID tetap; labelnya hanya tampilan. Isi baris
-- dokumen (M4.1) memakai ID itu, jadi label bisa diganti tanpa merusak data,
-- dan field yang tidak dipakai lagi disembunyikan, tidak dihapus (R-TPL-02).
--
--   format_templates          satu template: jenis dokumen, metodologi, customer (SQAM)
--   format_template_versions  versi template: metode risiko dan aturan sorot
--   format_fields             identitas field yang tetap lintas versi: area dan jenis
--   format_version_fields     label, urutan, wajib, tersembunyi, dan pilihan per versi
--   format_rating_criteria    tabel skala severity, occurrence, dan detection per versi
--   format_symbols            simbol karakteristik per versi

-- +goose Up
CREATE TABLE format_templates (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    name text NOT NULL,
    document_type_id uuid NOT NULL REFERENCES document_types (id),
    -- Metodologi PFMEA (R-TPL-02). Kosong untuk form PFD dan CP yang dipakai
    -- sekarang, karena keduanya bukan bagian buku AIAG FMEA (ASUMSI E-70).
    methodology text,
    -- Kosong untuk template umum; diisi untuk format SQAM satu customer.
    customer_id uuid REFERENCES customers (id),
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT format_templates_methodology_check CHECK (methodology IN ('aiag_4th', 'aiag_vda', 'sqam')),
    CONSTRAINT format_templates_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);
CREATE UNIQUE INDEX format_templates_name_key ON format_templates (lower(name));

-- Versi template. Versi draft bisa diubah; versi published (M3.4) tidak
-- bisa diubah lagi (R-TPL-05). Metode risiko tetap sejak versi dibuat
-- (ASUMSI E-70). Aturan sorot hanya memberi tanda, tidak memblokir
-- (R-TPL-11, R-DOC-06): RPN di atas nilai ini, atau severity paling sedikit
-- nilai ini.
CREATE TABLE format_template_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    template_id uuid NOT NULL REFERENCES format_templates (id),
    version integer NOT NULL,
    status text NOT NULL DEFAULT 'draft',
    risk_method text NOT NULL,
    highlight_rpn_above smallint,
    highlight_severity_at_least smallint,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT format_template_versions_version_key UNIQUE (template_id, version),
    -- Sasaran foreign key gabungan dari format_version_fields.
    CONSTRAINT format_template_versions_id_template_key UNIQUE (id, template_id),
    CONSTRAINT format_template_versions_version_check CHECK (version >= 1),
    CONSTRAINT format_template_versions_status_check CHECK (status IN ('draft', 'published')),
    CONSTRAINT format_template_versions_risk_method_check CHECK (risk_method IN ('rpn', 'ap', 'none')),
    CONSTRAINT format_template_versions_highlight_check CHECK (
        (highlight_rpn_above IS NULL OR (risk_method = 'rpn' AND highlight_rpn_above BETWEEN 1 AND 999))
        AND (highlight_severity_at_least IS NULL
             OR (risk_method IN ('rpn', 'ap') AND highlight_severity_at_least BETWEEN 1 AND 10))
    )
);

-- Identitas field, tetap lintas versi. Area dan jenis tidak bisa diubah,
-- karena mengubahnya merusak data yang sudah ada; role aplikasi tidak punya
-- UPDATE di tabel ini.
CREATE TABLE format_fields (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    template_id uuid NOT NULL REFERENCES format_templates (id),
    -- header: field header dokumen; row: kolom baris.
    area text NOT NULL,
    kind text NOT NULL,
    -- Untuk kind rating: severity, occurrence, atau detection.
    rating_scale text,
    -- Untuk kind rating dan rpn: 1 = nilai sekarang, 2 = hasil tindakan
    -- (kolom Action Results di PFMEA AIAG 4th). RPN dihitung dari severity,
    -- occurrence, dan detection dengan set yang sama.
    rating_set smallint,
    created_at timestamptz NOT NULL DEFAULT now(),

    -- Sasaran foreign key gabungan dari format_version_fields.
    CONSTRAINT format_fields_id_template_area_key UNIQUE (id, template_id, area),
    CONSTRAINT format_fields_area_check CHECK (area IN ('header', 'row')),
    CONSTRAINT format_fields_kind_check CHECK (kind IN ('text', 'date', 'select', 'symbol', 'rating', 'rpn')),
    CONSTRAINT format_fields_rating_check CHECK (
        (kind = 'rating') = (rating_scale IS NOT NULL)
        AND (kind IN ('rating', 'rpn')) = (rating_set IS NOT NULL)
        AND (rating_scale IS NULL OR rating_scale IN ('severity', 'occurrence', 'detection'))
        AND (rating_set IS NULL OR rating_set IN (1, 2))
        AND (kind NOT IN ('rating', 'rpn') OR area = 'row')
    )
);

-- Field dalam satu versi: label tampilan, urutan dalam area, wajib atau
-- tidak, tersembunyi atau tidak, dan pilihan untuk kind select. Tidak ada
-- DELETE: field yang tidak dipakai disembunyikan, datanya tetap (R-TPL-02).
CREATE TABLE format_version_fields (
    version_id uuid NOT NULL,
    field_id uuid NOT NULL,
    template_id uuid NOT NULL,
    area text NOT NULL,
    label text NOT NULL,
    position smallint NOT NULL,
    required boolean NOT NULL DEFAULT false,
    hidden boolean NOT NULL DEFAULT false,
    -- Pilihan kind select, misalnya [{"key": "scrap", "label": "Scrap"}].
    -- Key tetap dan dipakai isi baris; label bisa diganti.
    options jsonb,

    PRIMARY KEY (version_id, field_id),
    -- Versi dan field harus milik template yang sama, dan area di sini sama
    -- dengan area identitas field.
    CONSTRAINT format_version_fields_version_fkey FOREIGN KEY (version_id, template_id)
        REFERENCES format_template_versions (id, template_id),
    CONSTRAINT format_version_fields_field_fkey FOREIGN KEY (field_id, template_id, area)
        REFERENCES format_fields (id, template_id, area),
    -- Urutan unik per area. Diperiksa di akhir transaksi (DEFERRABLE), supaya
    -- field bisa digeser tanpa bentrok di tengah jalan.
    CONSTRAINT format_version_fields_position_key UNIQUE (version_id, area, position)
        DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT format_version_fields_position_check CHECK (position >= 1),
    CONSTRAINT format_version_fields_label_check CHECK (
        label <> '' AND label = btrim(label) AND char_length(label) <= 100
    ),
    CONSTRAINT format_version_fields_options_check CHECK (options IS NULL OR jsonb_typeof(options) = 'array'),
    -- Field tersembunyi tidak bisa wajib.
    CONSTRAINT format_version_fields_hidden_check CHECK (NOT (hidden AND required))
);
CREATE UNIQUE INDEX format_version_fields_label_key ON format_version_fields (version_id, area, lower(label));

-- Tabel skala per versi (R-TPL-01): nilai 1 sampai 10 untuk severity,
-- occurrence, dan detection, dengan kriteria tertulis. Teks kriteria diisi
-- dari form yang dipakai; isi awalnya kosong (ASUMSI E-70).
CREATE TABLE format_rating_criteria (
    version_id uuid NOT NULL REFERENCES format_template_versions (id),
    scale text NOT NULL,
    value smallint NOT NULL,
    criteria text NOT NULL DEFAULT '',

    PRIMARY KEY (version_id, scale, value),
    CONSTRAINT format_rating_criteria_scale_check CHECK (scale IN ('severity', 'occurrence', 'detection')),
    CONSTRAINT format_rating_criteria_value_check CHECK (value BETWEEN 1 AND 10),
    CONSTRAINT format_rating_criteria_criteria_check CHECK (
        criteria = btrim(criteria) AND char_length(criteria) <= 500
    )
);

-- Simbol karakteristik per versi (R-TPL-11): key tetap dipakai isi baris,
-- nama hanya tampilan. Bentuk cetaknya ditentukan layout (Tahap 6).
CREATE TABLE format_symbols (
    version_id uuid NOT NULL REFERENCES format_template_versions (id),
    key text NOT NULL,
    name text NOT NULL,
    position smallint NOT NULL,

    PRIMARY KEY (version_id, key),
    CONSTRAINT format_symbols_position_key UNIQUE (version_id, position),
    CONSTRAINT format_symbols_key_check CHECK (key ~ '^[a-z][a-z0-9_]{0,49}$'),
    CONSTRAINT format_symbols_position_check CHECK (position >= 1),
    CONSTRAINT format_symbols_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 100
    )
);
CREATE UNIQUE INDEX format_symbols_name_key ON format_symbols (version_id, lower(name));

-- Isi awal (usulan M3.1): kolom form yang dipakai sekarang, menurut temuan
-- dokumen referensi di brief bagian 21. PFD dari F-09 dan R-DOC-09, PFMEA
-- dari F-10 dengan kolom standar AIAG 4th, dan CP dari F-11. Proses dan nomor
-- langkah bukan field template, karena keduanya atribut baris (R-DOC-04,
-- R-CMP-01). Semua field tidak wajib; pemilik yang menentukan (ASUMSI E-70).
-- +goose StatementBegin
DO $$
DECLARE
    t_id uuid;
    v_id uuid;
    f_id uuid;
    f record;
BEGIN
    -- PFD: simbol alur dan disposisi NG.
    INSERT INTO format_templates (name, document_type_id)
    SELECT 'PFD', id FROM document_types WHERE key = 'pfd'
    RETURNING id INTO t_id;
    INSERT INTO format_template_versions (template_id, version, risk_method)
    VALUES (t_id, 1, 'none')
    RETURNING id INTO v_id;
    FOR f IN
        SELECT * FROM (VALUES
            ('header', 1, 'text', 'Process or Line', NULL::jsonb),
            ('row', 1, 'select', 'Flow Symbol', '[{"key": "process", "label": "Process"},
                {"key": "alternate_process", "label": "Alternate Process"}, {"key": "decision", "label": "Decision"},
                {"key": "connector", "label": "Connector"}, {"key": "merge", "label": "Merge"},
                {"key": "document", "label": "Document"}]'::jsonb),
            ('row', 2, 'select', 'NG Disposition', '[{"key": "scrap", "label": "Scrap"},
                {"key": "ncp", "label": "NCP"}, {"key": "rework", "label": "Rework"}]'::jsonb)
        ) AS v (area, position, kind, label, options)
        ORDER BY area, position
    LOOP
        INSERT INTO format_fields (template_id, area, kind)
        VALUES (t_id, f.area, f.kind)
        RETURNING id INTO f_id;
        INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position, options)
        VALUES (v_id, f_id, t_id, f.area, f.label, f.position, f.options);
    END LOOP;

    -- PFMEA AIAG 4th: header F-10, kolom standar AIAG 4th dengan RPN dan
    -- hasil tindakan, aturan sorot dari form (RPN di atas 100 atau severity 8).
    INSERT INTO format_templates (name, document_type_id, methodology)
    SELECT 'PFMEA AIAG 4th', id, 'aiag_4th' FROM document_types WHERE key = 'pfmea'
    RETURNING id INTO t_id;
    INSERT INTO format_template_versions (template_id, version, risk_method, highlight_rpn_above, highlight_severity_at_least)
    VALUES (t_id, 1, 'rpn', 100, 8)
    RETURNING id INTO v_id;
    FOR f IN
        SELECT * FROM (VALUES
            ('header', 1, 'text', NULL::text, NULL::smallint, 'Process or Line'),
            ('header', 2, 'text', NULL, NULL, 'Process Responsibility'),
            ('header', 3, 'date', NULL, NULL, 'Key Date'),
            ('header', 4, 'text', NULL, NULL, 'Core Team'),
            ('row', 1, 'text', NULL, NULL, 'Process Step / Function'),
            ('row', 2, 'text', NULL, NULL, 'Requirements'),
            ('row', 3, 'text', NULL, NULL, 'Potential Failure Mode'),
            ('row', 4, 'text', NULL, NULL, 'Potential Effect(s) of Failure'),
            ('row', 5, 'rating', 'severity', 1, 'Severity'),
            ('row', 6, 'symbol', NULL, NULL, 'Classification'),
            ('row', 7, 'text', NULL, NULL, 'Potential Cause(s) of Failure'),
            ('row', 8, 'text', NULL, NULL, 'Current Process Controls Prevention'),
            ('row', 9, 'rating', 'occurrence', 1, 'Occurrence'),
            ('row', 10, 'text', NULL, NULL, 'Current Process Controls Detection'),
            ('row', 11, 'rating', 'detection', 1, 'Detection'),
            ('row', 12, 'rpn', NULL, 1, 'RPN'),
            ('row', 13, 'text', NULL, NULL, 'Recommended Action(s)'),
            ('row', 14, 'text', NULL, NULL, 'Responsibility & Target Completion Date'),
            ('row', 15, 'text', NULL, NULL, 'Actions Taken & Completion Date'),
            ('row', 16, 'rating', 'severity', 2, 'Result Severity'),
            ('row', 17, 'rating', 'occurrence', 2, 'Result Occurrence'),
            ('row', 18, 'rating', 'detection', 2, 'Result Detection'),
            ('row', 19, 'rpn', NULL, 2, 'Result RPN')
        ) AS v (area, position, kind, rating_scale, rating_set, label)
        ORDER BY area, position
    LOOP
        INSERT INTO format_fields (template_id, area, kind, rating_scale, rating_set)
        VALUES (t_id, f.area, f.kind, f.rating_scale, f.rating_set)
        RETURNING id INTO f_id;
        INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position)
        VALUES (v_id, f_id, t_id, f.area, f.label, f.position);
    END LOOP;
    INSERT INTO format_rating_criteria (version_id, scale, value)
    SELECT v_id, s.scale, n
    FROM unnest(ARRAY['severity', 'occurrence', 'detection']) AS s (scale)
    CROSS JOIN generate_series(1, 10) AS n;
    INSERT INTO format_symbols (version_id, key, name, position) VALUES
        (v_id, 'critical_process', 'Critical Process', 1),
        (v_id, 'revision', 'Revision', 2),
        (v_id, 'important_character', 'Important Character', 3),
        (v_id, 'regulation', 'Regulation', 4);

    -- CP: header dan kolom form CP yang dipakai sekarang (F-11).
    INSERT INTO format_templates (name, document_type_id)
    SELECT 'CP', id FROM document_types WHERE key = 'cp'
    RETURNING id INTO t_id;
    INSERT INTO format_template_versions (template_id, version, risk_method)
    VALUES (t_id, 1, 'none')
    RETURNING id INTO v_id;
    FOR f IN
        SELECT * FROM (VALUES
            ('header', 1, 'text', 'Process or Line'),
            ('header', 2, 'text', 'Core Team'),
            ('row', 1, 'text', 'Process Flow'),
            ('row', 2, 'text', 'Machine and Tools'),
            ('row', 3, 'text', 'Product Characteristic'),
            ('row', 4, 'text', 'Process Characteristic'),
            ('row', 5, 'text', 'Specification'),
            ('row', 6, 'symbol', 'Special Characteristic'),
            ('row', 7, 'text', 'Equipment'),
            ('row', 8, 'text', 'Method'),
            ('row', 9, 'text', 'Frequency'),
            ('row', 10, 'text', 'PIC'),
            ('row', 11, 'text', 'Relevant Document'),
            ('row', 12, 'text', 'Pokayoke'),
            ('row', 13, 'text', 'Reaction Plan')
        ) AS v (area, position, kind, label)
        ORDER BY area, position
    LOOP
        INSERT INTO format_fields (template_id, area, kind)
        VALUES (t_id, f.area, f.kind)
        RETURNING id INTO f_id;
        INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position)
        VALUES (v_id, f_id, t_id, f.area, f.label, f.position);
    END LOOP;
    INSERT INTO format_symbols (version_id, key, name, position) VALUES
        (v_id, 'critical_process', 'Critical Process', 1),
        (v_id, 'revision', 'Revision', 2),
        (v_id, 'important_character', 'Important Character', 3),
        (v_id, 'regulation', 'Regulation', 4);
END
$$;
-- +goose StatementEnd

-- Hak role aplikasi (E-06), sekecil mungkin. Tidak ada DELETE di semua tabel
-- format template, dan identitas field (format_fields) tidak bisa diubah.
GRANT SELECT, INSERT ON format_templates TO revioten_app;
GRANT UPDATE (name) ON format_templates TO revioten_app;
GRANT SELECT, INSERT ON format_template_versions TO revioten_app;
GRANT UPDATE (highlight_rpn_above, highlight_severity_at_least) ON format_template_versions TO revioten_app;
GRANT SELECT, INSERT ON format_fields TO revioten_app;
GRANT SELECT, INSERT ON format_version_fields TO revioten_app;
GRANT UPDATE (label, position, required, hidden, options) ON format_version_fields TO revioten_app;
GRANT SELECT, INSERT ON format_rating_criteria TO revioten_app;
GRANT UPDATE (criteria) ON format_rating_criteria TO revioten_app;
GRANT SELECT, INSERT ON format_symbols TO revioten_app;
GRANT UPDATE (name) ON format_symbols TO revioten_app;

-- +goose Down
DROP TABLE format_symbols;
DROP TABLE format_rating_criteria;
DROP TABLE format_version_fields;
DROP TABLE format_fields;
DROP TABLE format_template_versions;
DROP TABLE format_templates;
