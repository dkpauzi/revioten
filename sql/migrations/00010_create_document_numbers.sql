-- Register nomor dokumen (Tahap 2: M2.1 sampai M2.5, UC-10 sampai UC-14).
-- Skema seluruh Tahap 2 dibuat sekaligus di M2.1, supaya bentuk register
-- tidak berubah-ubah antar milestone; fiturnya dibangun per milestone.
--
-- Aturan utama:
--   - nomor disimpan sebagai teks dan tidak pernah dihitung ulang (R-NUM-02);
--   - nomor dialokasikan server dengan baris penghitung di dalam transaksi,
--     bukan sequence PostgreSQL, dengan UNIQUE pada teks nomor dan kunci
--     idempoten (R-NUM-03);
--   - nomor tidak pernah dihapus atau dipakai ulang (R-NUM-04): role aplikasi
--     tanpa DELETE, dan hanya kolom status yang boleh diubah;
--   - kode customer di nomor adalah kode customer itu, sekarang atau dulu
--     (R-POR-01, E-11), dijaga foreign key gabungan.

-- +goose Up
-- Pola nomor, bentuk nomor berjalan, dan jumlah digit revisi sebagai master
-- data (R-NUM-02, E-12, D-89). Satu pola untuk semua jenis dokumen dan
-- customer (ASUMSI E-61); pola per customer menunggu Q-05.
ALTER TABLE number_settings
    ADD COLUMN pattern text NOT NULL DEFAULT '{prefix}-{customer}-{seq}',
    ADD COLUMN sequence_format text NOT NULL DEFAULT 'letter_block',
    ADD COLUMN revision_digits smallint NOT NULL DEFAULT 2,
    ADD CONSTRAINT number_settings_pattern_check CHECK (char_length(pattern) BETWEEN 1 AND 50),
    ADD CONSTRAINT number_settings_sequence_format_check CHECK (sequence_format IN ('letter_block')),
    ADD CONSTRAINT number_settings_revision_digits_check CHECK (revision_digits BETWEEN 1 AND 3);

-- Sasaran foreign key gabungan (kode, customer) di document_numbers.
ALTER TABLE customer_codes ADD CONSTRAINT customer_codes_code_customer_key UNIQUE (code, customer_id);

-- Penghitung per teks kode customer dan jenis dokumen (R-NUM-03, D-93): kode
-- baru setelah penggantian kode mulai lagi dari nomor pertama, dan kode lama
-- tidak pernah menghasilkan nomor ganda. last_value adalah nomor berjalan
-- terakhir yang sudah dipakai.
CREATE TABLE number_counters (
    customer_code text NOT NULL REFERENCES customer_codes (code),
    document_type_id uuid NOT NULL REFERENCES document_types (id),
    last_value integer NOT NULL,

    PRIMARY KEY (customer_code, document_type_id),
    CONSTRAINT number_counters_last_value_check CHECK (last_value >= 0)
);

-- Satu permintaan yang menghasilkan nomor: pendaftaran satu nomor (single),
-- pendaftaran sebagai set (set, M2.4), atau impor nomor lama (import, M2.5).
-- Kunci idempoten (R-NUM-03) berlaku per akun: permintaan yang diulang dengan
-- kunci yang sama mendapat nomor yang sama, bukan nomor baru (ASUMSI E-63).
CREATE TABLE number_batches (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    kind text NOT NULL,
    customer_id uuid NOT NULL REFERENCES customers (id),
    account_id uuid NOT NULL REFERENCES accounts (id),
    person_id uuid NOT NULL REFERENCES people (id),
    idempotency_key text,
    -- SHA-256 dari isi permintaan, untuk mengenali kunci yang dipakai ulang
    -- dengan isi berbeda.
    request_hash text,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT number_batches_kind_check CHECK (kind IN ('single', 'set', 'import')),
    CONSTRAINT number_batches_idempotency_key UNIQUE (account_id, idempotency_key),
    -- Pendaftaran selalu punya kunci idempoten; impor tidak.
    CONSTRAINT number_batches_key_check CHECK ((kind = 'import') = (idempotency_key IS NULL)),
    CONSTRAINT number_batches_key_format CHECK (
        idempotency_key IS NULL OR (char_length(idempotency_key) BETWEEN 1 AND 100 AND request_hash IS NOT NULL)
    )
);

-- Register nomor (R-NUM-04). sequence adalah nilai nomor berjalan (1 untuk
-- 001, 1000 untuk A001), dipakai untuk penghitung dan laporan lubang (M2.5).
-- Status: Reserved, In use, Released, Obsolete, Void (R-NUM-04), ditambah
-- Skipped untuk nomor yang dilewati saat daftar sebagai set (ASUMSI E-64).
CREATE TABLE document_numbers (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    number text NOT NULL,
    customer_id uuid NOT NULL,
    customer_code text NOT NULL,
    document_type_id uuid NOT NULL REFERENCES document_types (id),
    sequence integer NOT NULL,
    status text NOT NULL,
    -- Revisi awal dokumen yang memakai nomor ini, misalnya 4 untuk dokumen
    -- lama yang masuk di revisi -04 (R-REV-01, M2.4).
    initial_revision smallint NOT NULL DEFAULT 0,
    batch_id uuid NOT NULL REFERENCES number_batches (id),
    -- Alasan Void atau Obsolete (M2.3).
    status_reason text,
    status_changed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT document_numbers_number_key UNIQUE (number),
    -- Satu nomor berjalan hanya sekali per kode customer dan jenis dokumen,
    -- juga bila pola nomor berubah (tidak pernah dipakai ulang, R-NUM-04).
    CONSTRAINT document_numbers_sequence_key UNIQUE (customer_code, document_type_id, sequence),
    -- Kode di nomor harus kode customer itu, sekarang atau dulu (R-POR-01, E-11).
    CONSTRAINT document_numbers_code_fkey FOREIGN KEY (customer_code, customer_id)
        REFERENCES customer_codes (code, customer_id),
    CONSTRAINT document_numbers_sequence_check CHECK (sequence >= 0),
    CONSTRAINT document_numbers_status_check CHECK (
        status IN ('reserved', 'in_use', 'released', 'obsolete', 'void', 'skipped')
    ),
    CONSTRAINT document_numbers_initial_revision_check CHECK (initial_revision >= 0),
    CONSTRAINT document_numbers_reason_check CHECK (
        status_reason IS NULL OR (status_reason <> '' AND status_reason = btrim(status_reason) AND char_length(status_reason) <= 500)
    )
);
CREATE INDEX document_numbers_customer_idx ON document_numbers (customer_id, document_type_id, sequence);
CREATE INDEX document_numbers_batch_idx ON document_numbers (batch_id);

-- Hak role aplikasi (E-06). Tanpa DELETE di semua tabel register: nomor tidak
-- pernah dihapus. Di document_numbers hanya kolom status yang boleh diubah,
-- jadi teks nomor tidak bisa diganti, juga oleh kode program yang salah.
GRANT SELECT, INSERT, UPDATE ON number_counters TO revioten_app;
GRANT SELECT, INSERT ON number_batches TO revioten_app;
GRANT SELECT, INSERT ON document_numbers TO revioten_app;
GRANT UPDATE (status, status_reason, status_changed_at) ON document_numbers TO revioten_app;

-- +goose Down
DROP TABLE document_numbers;
DROP TABLE number_batches;
DROP TABLE number_counters;
ALTER TABLE customer_codes DROP CONSTRAINT customer_codes_code_customer_key;
ALTER TABLE number_settings
    DROP COLUMN revision_digits,
    DROP COLUMN sequence_format,
    DROP COLUMN pattern;
