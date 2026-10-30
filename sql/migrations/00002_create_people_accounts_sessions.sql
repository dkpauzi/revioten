-- Orang, akun, dan sesi login (M1.1).
-- E-02 (diputuskan): orang dan akun dipisah. Satu orang punya paling banyak
-- satu akun kerja dan satu akun admin (D-72). Pemisahan tugas nanti
-- membandingkan orang, bukan akun.

-- +goose Up
CREATE TABLE people (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    name text NOT NULL,
    -- Nomor karyawan boleh kosong. Brief bagian 8: identitas nanti diikat ke
    -- nomor karyawan (LDAP), jadi kolomnya disiapkan sekarang.
    employee_number text,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT people_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    ),
    CONSTRAINT people_employee_number_key UNIQUE (employee_number)
);

CREATE TABLE accounts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    person_id uuid NOT NULL REFERENCES people (id),
    -- work: akun kerja. admin: akun admin terpisah (D-72, R-SEC-13).
    kind text NOT NULL,
    -- Nama login selalu huruf kecil, jadi UNIQUE di bawah sekaligus berarti
    -- tidak membedakan huruf besar dan kecil. E-08 (D-85): 3 sampai 64 karakter.
    login text NOT NULL,
    -- Hash argon2id dalam format PHC, misalnya $argon2id$v=19$m=19456,t=2,p=1$...
    -- Password asli tidak pernah disimpan.
    password_hash text NOT NULL,
    -- true bila admin mereset password: pemilik akun wajib menggantinya (E-08).
    must_change_password boolean NOT NULL DEFAULT false,
    -- Jumlah gagal login berturut-turut, dan kunci sementara (E-08).
    failed_logins integer NOT NULL DEFAULT 0,
    locked_until timestamptz,
    -- Akun yang dinonaktifkan tidak bisa login (ASVS 7.4.2).
    disabled_at timestamptz,
    -- Disiapkan untuk login LDAP nanti (brief bagian 10, DEFERRED).
    external_id text,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT accounts_kind_check CHECK (kind IN ('work', 'admin')),
    CONSTRAINT accounts_login_key UNIQUE (login),
    CONSTRAINT accounts_login_format CHECK (login ~ '^[a-z0-9._-]{3,64}$'),
    CONSTRAINT accounts_person_kind_key UNIQUE (person_id, kind),
    CONSTRAINT accounts_external_id_key UNIQUE (external_id)
);

CREATE TABLE sessions (
    -- Yang disimpan hanya hash SHA-256 dari token sesi. Token aslinya hanya ada
    -- di cookie browser, jadi isi tabel ini tidak bisa dipakai untuk login.
    token_hash bytea PRIMARY KEY,
    account_id uuid NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL,
    -- Batas mutlak umur sesi, dihitung dari created_at.
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_account_id_idx ON sessions (account_id);

-- +goose Down
DROP TABLE sessions;
DROP TABLE accounts;
DROP TABLE people;
