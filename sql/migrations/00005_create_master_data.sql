-- Master Project Group, kode customer dengan riwayat, group model, model, dan
-- part number (M1.4, UC-03). Semua master hanya diubah Admin (E-10) dan tidak
-- pernah dihapus: dokumen dan audit log menunjuk ID-nya (E-09).

-- +goose Up
-- Project Group (GR): kelompok customer yang dipegang satu Project Leader dan
-- satu Project Engineer (D-54). Pemegang jabatannya dicatat di M1.8 (E-05).
-- ASUMSI E-30: kode huruf besar dan angka, 1 sampai 10 karakter, misalnya GR1.
CREATE TABLE project_groups (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    code text NOT NULL,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT project_groups_code_key UNIQUE (code),
    CONSTRAINT project_groups_code_format CHECK (code ~ '^[A-Z0-9]{1,10}$'),
    CONSTRAINT project_groups_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);

-- Setiap customer dialokasikan ke satu Project Group (D-54). Kolomnya boleh
-- kosong hanya untuk customer yang dibuat sebelum M1.4; aplikasi mewajibkannya
-- untuk customer baru dan saat customer diubah (ASUMSI E-09).
-- Customer tidak dihapus, hanya dinonaktifkan (E-09).
ALTER TABLE customers
    ADD COLUMN project_group_id uuid REFERENCES project_groups (id),
    ADD COLUMN active boolean NOT NULL DEFAULT true;

-- Riwayat kode customer (E-11). customers.code adalah kode yang berlaku; tabel
-- ini menyimpan semua kode yang pernah berlaku beserta masanya, supaya nomor
-- lama yang memakai kode lama tetap bisa dicocokkan (R-NUM-02, R-POR-01).
-- Satu kode hanya pernah dipakai satu customer, jadi nomor lama tidak pernah
-- menunjuk dua customer (ASUMSI E-31).
CREATE TABLE customer_codes (
    code text PRIMARY KEY,
    customer_id uuid NOT NULL REFERENCES customers (id),
    valid_from timestamptz NOT NULL,
    -- Kosong untuk kode yang sedang berlaku.
    valid_until timestamptz,

    CONSTRAINT customer_codes_period CHECK (valid_until IS NULL OR valid_until >= valid_from)
);
-- Setiap customer tepat punya satu kode yang sedang berlaku.
CREATE UNIQUE INDEX customer_codes_current_key ON customer_codes (customer_id) WHERE valid_until IS NULL;

INSERT INTO customer_codes (code, customer_id, valid_from)
SELECT code, id, created_at FROM customers;

-- Group model: kumpulan model dalam satu customer yang berbagi isi template
-- (D-39). Satu group hanya milik satu customer (D-42).
-- ASUMSI E-30: group punya nama, unik dalam satu customer.
CREATE TABLE model_groups (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    customer_id uuid NOT NULL REFERENCES customers (id),
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT model_groups_name_key UNIQUE (customer_id, name),
    -- Sasaran foreign key gabungan di tabel models.
    CONSTRAINT model_groups_id_customer_key UNIQUE (id, customer_id),
    CONSTRAINT model_groups_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);

-- Model milik satu customer, dan paling banyak masuk satu group (D-42).
-- ASUMSI E-30: model punya nama (misalnya nama di kolom Model pada PFMEA),
-- unik dalam satu customer.
CREATE TABLE models (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    customer_id uuid NOT NULL REFERENCES customers (id),
    model_group_id uuid,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT models_name_key UNIQUE (customer_id, name),
    CONSTRAINT models_id_customer_key UNIQUE (id, customer_id),
    -- Foreign key gabungan: group yang dipilih harus milik customer yang sama
    -- dengan modelnya (D-42). Bila model_group_id kosong, batasan ini tidak dicek.
    CONSTRAINT models_group_same_customer FOREIGN KEY (model_group_id, customer_id)
        REFERENCES model_groups (id, customer_id),
    CONSTRAINT models_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);

-- Part number milik satu model, dan unik dalam satu customer.
-- ASUMSI E-30: nomor part wajib (misalnya nomor part customer), nama part boleh kosong.
CREATE TABLE part_numbers (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    customer_id uuid NOT NULL,
    model_id uuid NOT NULL,
    number text NOT NULL,
    name text,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT part_numbers_number_key UNIQUE (customer_id, number),
    -- Customer part number selalu sama dengan customer modelnya.
    CONSTRAINT part_numbers_model_same_customer FOREIGN KEY (model_id, customer_id)
        REFERENCES models (id, customer_id),
    CONSTRAINT part_numbers_number_check CHECK (
        number <> '' AND number = btrim(number) AND char_length(number) <= 100
    ),
    CONSTRAINT part_numbers_name_check CHECK (
        name IS NULL OR (name <> '' AND name = btrim(name) AND char_length(name) <= 200)
    )
);

-- Hak role aplikasi (E-06): tanpa DELETE, karena master tidak pernah dihapus.
GRANT SELECT, INSERT, UPDATE ON project_groups TO revioten_app;
GRANT UPDATE ON customers TO revioten_app;
GRANT SELECT, INSERT, UPDATE ON customer_codes TO revioten_app;
GRANT SELECT, INSERT, UPDATE ON model_groups TO revioten_app;
GRANT SELECT, INSERT, UPDATE ON models TO revioten_app;
GRANT SELECT, INSERT, UPDATE ON part_numbers TO revioten_app;

-- +goose Down
DROP TABLE part_numbers;
DROP TABLE models;
DROP TABLE model_groups;
DROP TABLE customer_codes;
REVOKE UPDATE ON customers FROM revioten_app;
ALTER TABLE customers DROP COLUMN active, DROP COLUMN project_group_id;
DROP TABLE project_groups;
