-- Form set langkah D (D-133, D-139, D-140), dan keputusan E-89 (D-144).
--
--   form_sets              satu paket form: nama, key, dan tanda set bawaan
--   form_set_versions      versi paket: menunjuk versi format template untuk
--                          PFD, PFMEA, dan Control Plan (Cover menyusul saat
--                          form Cover dibangun). Draft lalu terbit; versi
--                          terbit tidak berubah, seperti versi template.
--   customers.default_form_set_id
--                          form set untuk dokumen baru customer itu. Kosong
--                          berarti set bawaan (AIAG 4th).
--
-- Hak baru V-3 "setujui form set dan tabel AP" dan peran baru form_approver
-- (Penyetuju form), sesuai D-139: untuk saat ini dipegang PPAP Supervisor,
-- dan admin bisa memberikannya ke akun lain tanpa perubahan kode (D-66).
-- Penyetuju tidak boleh orang yang menyusun drafnya, jadi orang penyusun
-- dicatat di versinya.

-- +goose Up

-- D-144 (keputusan E-89): di PFMEA AIAG-VDA, "2. Process Step" diisi dari
-- master proses lewat baris, tidak diketik (sama dengan D-123). Jenis field
-- adalah bagian identitas field; belum ada dokumen yang memakai form ini.
UPDATE format_fields f
SET kind = 'process'
FROM format_templates t
WHERE t.id = f.template_id AND t.methodology = 'aiag_vda'
  AND f.key = 'process_step' AND f.area = 'row' AND f.kind = 'text';

-- Hak dan peran baru. Urutan tampil: V-3 sesudah V-2, peran sesudah Verifikator.
UPDATE permissions SET sort_order = sort_order + 1 WHERE sort_order >= 17;
INSERT INTO permissions (code, level, sort_order) VALUES ('V-3', 'verify', 17);
UPDATE roles SET sort_order = sort_order + 1 WHERE sort_order >= 5;
INSERT INTO roles (id, level, sort_order) VALUES ('form_approver', 'verify', 5);
INSERT INTO role_permissions (role_id, permission_code) VALUES
    ('form_approver', 'L-1'), ('form_approver', 'L-2'), ('form_approver', 'L-3'), ('form_approver', 'L-4'),
    ('form_approver', 'V-3');

-- Orang yang terakhir mengubah isi versi format (dipakai untuk tabel AP).
ALTER TABLE format_template_versions ADD COLUMN changed_by_person_id uuid REFERENCES people (id);
GRANT UPDATE (changed_by_person_id) ON format_template_versions TO revioten_app;

CREATE TABLE form_sets (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    key text NOT NULL,
    name text NOT NULL,
    -- Set bawaan dipakai customer yang belum memilih form set. Hanya satu.
    is_default boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT form_sets_key_key UNIQUE (key),
    CONSTRAINT form_sets_key_check CHECK (key ~ '^[a-z][a-z0-9_]{0,49}$'),
    CONSTRAINT form_sets_name_check CHECK (name <> '' AND name = btrim(name) AND char_length(name) <= 200)
);
CREATE UNIQUE INDEX form_sets_name_key ON form_sets (lower(name));
CREATE UNIQUE INDEX form_sets_default_key ON form_sets (is_default) WHERE is_default;

CREATE TABLE form_set_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    form_set_id uuid NOT NULL REFERENCES form_sets (id),
    version integer NOT NULL,
    status text NOT NULL DEFAULT 'draft',
    published_at timestamptz,
    -- Cover belum punya format template; kolomnya disiapkan dan boleh kosong.
    cover_format_version_id uuid REFERENCES format_template_versions (id),
    pfd_format_version_id uuid NOT NULL REFERENCES format_template_versions (id),
    pfmea_format_version_id uuid NOT NULL REFERENCES format_template_versions (id),
    cp_format_version_id uuid NOT NULL REFERENCES format_template_versions (id),
    prepared_by_person_id uuid REFERENCES people (id),
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT form_set_versions_version_key UNIQUE (form_set_id, version),
    CONSTRAINT form_set_versions_version_check CHECK (version >= 1),
    CONSTRAINT form_set_versions_status_check CHECK (status IN ('draft', 'published')),
    CONSTRAINT form_set_versions_published_check CHECK ((status = 'published') = (published_at IS NOT NULL))
);
CREATE UNIQUE INDEX form_set_versions_draft_key ON form_set_versions (form_set_id) WHERE status = 'draft';

-- Versi form set yang sudah terbit tidak bisa diubah, dengan penjaga yang
-- sama seperti versi template (migrasi 00016).
CREATE TRIGGER form_set_versions_published
    BEFORE UPDATE OR DELETE ON form_set_versions
    FOR EACH ROW EXECUTE FUNCTION template_version_reject_change();
CREATE TRIGGER form_set_versions_truncate
    BEFORE TRUNCATE ON form_set_versions FOR EACH STATEMENT EXECUTE FUNCTION template_reject_truncate();

ALTER TABLE customers ADD COLUMN default_form_set_id uuid REFERENCES form_sets (id);

-- Dua set bawaan. Format dicari menurut jenis dokumen dan metodologi, bukan
-- nama, karena nama template boleh diganti admin. Yang dipakai template
-- tertua dengan ciri itu, yaitu form isi awal.
-- +goose StatementBegin
DO $$
DECLARE
    pfd uuid;
    pfmea_4th uuid;
    cp_old uuid;
    pfmea_vda uuid;
    cp_2024 uuid;
    s_id uuid;
BEGIN
    SELECT v.id INTO pfd FROM format_template_versions v
    JOIN format_templates t ON t.id = v.template_id JOIN document_types d ON d.id = t.document_type_id
    WHERE d.key = 'pfd' ORDER BY t.created_at, t.id, v.version DESC LIMIT 1;
    SELECT v.id INTO pfmea_4th FROM format_template_versions v
    JOIN format_templates t ON t.id = v.template_id
    WHERE t.methodology = 'aiag_4th' ORDER BY t.created_at, t.id, v.version DESC LIMIT 1;
    SELECT v.id INTO cp_old FROM format_template_versions v
    JOIN format_templates t ON t.id = v.template_id JOIN document_types d ON d.id = t.document_type_id
    WHERE d.key = 'cp' AND t.methodology IS NULL ORDER BY t.created_at, t.id, v.version DESC LIMIT 1;
    SELECT v.id INTO pfmea_vda FROM format_template_versions v
    JOIN format_templates t ON t.id = v.template_id JOIN document_types d ON d.id = t.document_type_id
    WHERE d.key = 'pfmea' AND t.methodology = 'aiag_vda' ORDER BY t.created_at, t.id, v.version DESC LIMIT 1;
    SELECT v.id INTO cp_2024 FROM format_template_versions v
    JOIN format_templates t ON t.id = v.template_id
    WHERE t.methodology = 'aiag_cp_2024' ORDER BY t.created_at, t.id, v.version DESC LIMIT 1;

    INSERT INTO form_sets (key, name, is_default) VALUES ('aiag_4th', 'AIAG 4th set', true) RETURNING id INTO s_id;
    INSERT INTO form_set_versions (form_set_id, version, pfd_format_version_id, pfmea_format_version_id, cp_format_version_id)
    VALUES (s_id, 1, pfd, pfmea_4th, cp_old);

    INSERT INTO form_sets (key, name) VALUES ('aiag_vda', 'AIAG-VDA set') RETURNING id INTO s_id;
    INSERT INTO form_set_versions (form_set_id, version, pfd_format_version_id, pfmea_format_version_id, cp_format_version_id)
    VALUES (s_id, 1, pfd, pfmea_vda, cp_2024);
END
$$;
-- +goose StatementEnd

-- Hak sekecil mungkin (E-06). Form set tidak pernah dihapus.
GRANT SELECT, INSERT ON form_sets TO revioten_app;
GRANT SELECT, INSERT ON form_set_versions TO revioten_app;
GRANT UPDATE (status, published_at, pfd_format_version_id, pfmea_format_version_id, cp_format_version_id, prepared_by_person_id)
    ON form_set_versions TO revioten_app;

-- +goose Down
ALTER TABLE customers DROP COLUMN default_form_set_id;
DROP TABLE form_set_versions;
DROP TABLE form_sets;
ALTER TABLE format_template_versions DROP COLUMN changed_by_person_id;
DELETE FROM account_roles WHERE role_id = 'form_approver';
DELETE FROM role_permissions WHERE role_id = 'form_approver';
DELETE FROM roles WHERE id = 'form_approver';
UPDATE roles SET sort_order = sort_order - 1 WHERE sort_order >= 6;
DELETE FROM permissions WHERE code = 'V-3';
UPDATE permissions SET sort_order = sort_order - 1 WHERE sort_order >= 18;
UPDATE format_fields f
SET kind = 'text'
FROM format_templates t
WHERE t.id = f.template_id AND t.methodology = 'aiag_vda'
  AND f.key = 'process_step' AND f.area = 'row' AND f.kind = 'process';
