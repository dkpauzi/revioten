-- Tabel customer pertama (M0.5). Kode dengan riwayat (E-11), Project Group
-- (D-54), dan status aktif menyusul di M1.4.

-- +goose Up
CREATE TABLE customers (
    -- uuidv7() tersedia sejak PostgreSQL 18. Nilainya urut menurut waktu
    -- pembuatan, dan dipakai sebagai ID tetap yang tidak berarti apa-apa (R-API-13).
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    -- Kode customer di nomor dokumen, misalnya SC-AB-001 (E-12).
    code text NOT NULL,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT customers_code_key UNIQUE (code),
    -- Aturan yang sama juga diperiksa di internal/customer. Batasan di sini
    -- adalah pengaman terakhir bila ada kode lain yang menulis langsung.
    CONSTRAINT customers_code_format CHECK (code ~ '^[A-Z]{2}$'),
    -- E-12 (D-89): ZZ dipakai untuk dokumen General/Common, bukan customer.
    CONSTRAINT customers_code_not_zz CHECK (code <> 'ZZ'),
    -- E-21 (D-100): nama wajib, paling panjang 200 karakter, tanpa spasi di tepi.
    CONSTRAINT customers_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);

-- +goose Down
DROP TABLE customers;
