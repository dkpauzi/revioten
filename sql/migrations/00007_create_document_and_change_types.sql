-- Master jenis dokumen dan jenis perubahan (M1.6, UC-05). Dua kelompok:
--   - bagian nomor dokumen menurut WI SW-ZZ-000 (E-12): kode pabrik, jenis
--     dokumen dengan hurufnya, dan kode khusus (R-NUM-01, R-NUM-02);
--   - daftar untuk perubahan: jenis perubahan (R-TPL-10), sumber perubahan
--     (R-CHG-06), dan kategori 4M (R-CHG-01).
-- Semua diubah Admin (A-2) dan tidak pernah dihapus, karena nomor, perubahan,
-- dan audit log menunjuk ID-nya. Entri yang dikenali kode program punya kolom
-- key tetap; namanya hanya tampilan.

-- +goose Up
-- Pengaturan nomor, satu baris saja. Kode pabrik adalah huruf pertama awalan
-- nomor, misalnya S di SH-KT-001 (E-12). Format nomor berjalan dan jumlah
-- digit revisi menyusul di M2.1.
CREATE TABLE number_settings (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    factory_code text NOT NULL,

    CONSTRAINT number_settings_factory_code_format CHECK (factory_code ~ '^[A-Z]$')
);
-- Indeks unik pada nilai yang selalu sama: baris kedua pasti ditolak.
CREATE UNIQUE INDEX number_settings_single_row ON number_settings ((true));
INSERT INTO number_settings (factory_code) VALUES ('S');

-- Jenis dokumen (huruf B di WI SW-ZZ-000). Awalan nomor = kode pabrik + huruf
-- jenis: SH untuk PFD, SF untuk PFMEA, dan SC untuk CP (R-NUM-01, E-12).
-- key diisi untuk dokumen yang dibuat sistem ini. Jenis lain dari WI (tanpa
-- key) boleh ditambah Admin, misalnya untuk dokumen terkait (R-DOC-11).
CREATE TABLE document_types (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    key text,
    name text NOT NULL,
    letter text NOT NULL,
    position smallint NOT NULL,

    CONSTRAINT document_types_key_key UNIQUE (key),
    CONSTRAINT document_types_key_check CHECK (key IN ('pfd', 'pfmea', 'cp')),
    CONSTRAINT document_types_letter_key UNIQUE (letter),
    CONSTRAINT document_types_letter_format CHECK (letter ~ '^[A-Z]$'),
    CONSTRAINT document_types_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);
-- Nama unik tanpa membedakan huruf besar kecil.
CREATE UNIQUE INDEX document_types_name_key ON document_types (lower(name));
INSERT INTO document_types (key, name, letter, position) VALUES
    ('pfd', 'PFD', 'H', 1),
    ('pfmea', 'PFMEA', 'F', 2),
    ('cp', 'CP', 'C', 3);

-- Kode khusus di posisi kode customer, untuk dokumen yang bukan milik satu
-- customer (E-12): ZZ untuk General/Common dan HSE untuk Health, Safety, and
-- Environment. Kode customer tidak boleh sama dengan kode khusus, dan
-- sebaliknya; aplikasi menjaganya dengan kunci advisory 11003 (E-37, D-114).
-- Kode tidak bisa diubah, hanya namanya.
CREATE TABLE special_codes (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    code text NOT NULL,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT special_codes_code_key UNIQUE (code),
    CONSTRAINT special_codes_code_format CHECK (code ~ '^[A-Z]{2,3}$'),
    CONSTRAINT special_codes_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);
INSERT INTO special_codes (code, name) VALUES
    ('ZZ', 'General/Common'),
    ('HSE', 'Health, Safety, and Environment');

-- Jenis perubahan (R-TPL-10): isi, tata letak saja, dan administratif. Jenis
-- ini menentukan jalur approval. Kebijakannya dibuat bersama rute approval di
-- M1.8; sampai itu semua jenis memakai alur yang sama (Q-15, E-35, D-112).
CREATE TABLE change_types (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    key text NOT NULL,
    name text NOT NULL,
    position smallint NOT NULL,

    CONSTRAINT change_types_key_key UNIQUE (key),
    CONSTRAINT change_types_key_check CHECK (key IN ('content', 'layout', 'administrative')),
    CONSTRAINT change_types_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);
CREATE UNIQUE INDEX change_types_name_key ON change_types (lower(name));
INSERT INTO change_types (key, name, position) VALUES
    ('content', 'Content', 1),
    ('layout', 'Layout only', 2),
    ('administrative', 'Administrative', 3);

-- Sumber perubahan (R-CHG-06): pemicu yang dipakai sekarang (PP, customer
-- claim, ECR, temuan audit) dan sumber dari mekanisme sistem (R-CHG-01).
-- mechanism menandai sumber yang dipakai mekanisme sistem: yokoten,
-- propagasi template, dan SQAM. Sumber itu selalu aktif (E-36, D-113).
CREATE TABLE change_sources (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    name text NOT NULL,
    mechanism text,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT change_sources_mechanism_key UNIQUE (mechanism),
    CONSTRAINT change_sources_mechanism_check CHECK (mechanism IN ('yokoten', 'template', 'sqam')),
    CONSTRAINT change_sources_system_active CHECK (mechanism IS NULL OR active),
    CONSTRAINT change_sources_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);
CREATE UNIQUE INDEX change_sources_name_key ON change_sources (lower(name));
INSERT INTO change_sources (name, mechanism) VALUES
    ('PP (pre-production)', NULL),
    ('Customer claim', NULL),
    ('ECR', NULL),
    ('Yokoten', 'yokoten'),
    ('Template propagation', 'template'),
    ('SQAM', 'sqam'),
    ('Audit finding', NULL);

-- Kategori 4M (brief bagian 3): Man, Machine, Material, Method. Kategori di
-- perubahan boleh kosong (R-CHG-01, Q-12).
CREATE TABLE four_m_categories (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    key text NOT NULL,
    name text NOT NULL,
    position smallint NOT NULL,

    CONSTRAINT four_m_categories_key_key UNIQUE (key),
    CONSTRAINT four_m_categories_key_check CHECK (key IN ('man', 'machine', 'material', 'method')),
    CONSTRAINT four_m_categories_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);
CREATE UNIQUE INDEX four_m_categories_name_key ON four_m_categories (lower(name));
INSERT INTO four_m_categories (key, name, position) VALUES
    ('man', 'Man', 1),
    ('machine', 'Machine', 2),
    ('material', 'Material', 3),
    ('method', 'Method', 4);

-- Hak role aplikasi (E-06): tanpa DELETE di semua tabel. Daftar tetap (jenis
-- dokumen sistem, jenis perubahan, 4M) hanya diganti namanya; jenis dokumen
-- lain, kode khusus, dan sumber perubahan bisa ditambah.
GRANT SELECT, UPDATE ON number_settings TO revioten_app;
GRANT SELECT, INSERT, UPDATE ON document_types TO revioten_app;
GRANT SELECT, INSERT, UPDATE ON special_codes TO revioten_app;
GRANT SELECT, UPDATE ON change_types TO revioten_app;
GRANT SELECT, INSERT, UPDATE ON change_sources TO revioten_app;
GRANT SELECT, UPDATE ON four_m_categories TO revioten_app;

-- +goose Down
DROP TABLE four_m_categories;
DROP TABLE change_sources;
DROP TABLE change_types;
DROP TABLE special_codes;
DROP TABLE document_types;
DROP TABLE number_settings;
