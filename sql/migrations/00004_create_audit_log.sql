-- Audit log tahan manipulasi dan role database aplikasi (M1.3, R-SEC-03, E-06).
--
-- Dua role database (E-06):
--   - pemilik (POSTGRES_USER di .env) menjalankan migrasi dan memiliki semua tabel;
--   - revioten_app dipakai server untuk semua operasi, dengan hak sekecil
--     mungkin per tabel. Tabel audit_log hanya bisa dibaca dan ditambah.
--
-- Role berlaku untuk seluruh server PostgreSQL, bukan per database, jadi role
-- hanya dibuat bila belum ada. Passwordnya tidak ditulis di sini: program
-- memasangnya saat mulai dari APP_DB_PASSWORD di .env (internal/db/approle.go).

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'revioten_app') THEN
        CREATE ROLE revioten_app LOGIN;
    END IF;
END
$$;
-- +goose StatementEnd

-- Hak role aplikasi mengikuti operasi yang sudah ada. Migrasi berikutnya yang
-- membuat tabel atau operasi baru juga menulis GRANT-nya sendiri.
GRANT SELECT, INSERT ON customers TO revioten_app;
GRANT SELECT, INSERT ON people TO revioten_app;
-- Tanpa DELETE: akun tidak pernah dihapus, hanya dinonaktifkan (ASUMSI E-26).
GRANT SELECT, INSERT, UPDATE ON accounts TO revioten_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sessions TO revioten_app;
-- Katalog hak dan peran hanya dibaca; isinya diubah lewat migrasi sampai M8.1.
GRANT SELECT ON permissions, roles, role_permissions TO revioten_app;
GRANT SELECT, INSERT, DELETE ON account_roles TO revioten_app;

-- Setiap baris adalah satu perubahan data. Baris membentuk rantai hash: hash
-- baris dihitung dari isinya beserta hash baris sebelumnya (prev_hash), jadi
-- mengubah atau menghapus satu baris memutus rantai sesudahnya. Rumus hash ada
-- di internal/audit/hash.go.
CREATE TABLE audit_log (
    -- Nomor urut tanpa lubang. Dibagikan aplikasi saat memegang kunci rantai,
    -- bukan dari sequence, supaya lubang berarti ada baris yang hilang (E-06).
    seq bigint PRIMARY KEY CHECK (seq > 0),
    occurred_at timestamptz NOT NULL,
    -- Jenis aktor (R-API-03). human: akun yang login. system: perintah di
    -- komputer server, misalnya create-admin. anonymous: belum login, misalnya
    -- login gagal. ASUMSI E-27.
    actor_type text NOT NULL CHECK (actor_type IN ('human', 'system', 'anonymous')),
    actor_account_id uuid REFERENCES accounts (id),
    -- Orang pemegang akun saat itu, supaya log menampilkan akun dan orangnya (D-74).
    actor_person_id uuid REFERENCES people (id),
    request_id text,
    source_ip text,
    -- Kode peristiwa, misalnya account.created atau customer.created.
    action text NOT NULL CHECK (action ~ '^[a-z_]+\.[a-z_]+$'),
    target_type text,
    target_id uuid,
    -- Rincian sebagai teks JSON yang disusun aplikasi. Disimpan sebagai teks,
    -- bukan jsonb, supaya byte yang di-hash sama persis dengan yang disimpan.
    details text NOT NULL,
    prev_hash text NOT NULL CHECK (prev_hash ~ '^[0-9a-f]{64}$'),
    hash text NOT NULL UNIQUE CHECK (hash ~ '^[0-9a-f]{64}$'),
    -- Aktor human selalu punya akun dan orang; aktor lain tidak punya keduanya.
    CONSTRAINT audit_log_actor_check CHECK (
        (actor_type = 'human') = (actor_account_id IS NOT NULL)
        AND (actor_account_id IS NULL) = (actor_person_id IS NULL)
    ),
    CONSTRAINT audit_log_target_check CHECK ((target_type IS NULL) = (target_id IS NULL))
);

-- Trigger penolak: UPDATE, DELETE, dan TRUNCATE selalu gagal, juga untuk
-- pemilik tabel yang punya semua hak. Trigger tingkat statement berjalan
-- walau tidak ada baris yang cocok, jadi setiap percobaan ditolak.
-- +goose StatementBegin
CREATE FUNCTION audit_log_reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_log hanya bisa ditambah: % ditolak (R-SEC-03)', TG_OP;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER audit_log_append_only
    BEFORE UPDATE OR DELETE OR TRUNCATE ON audit_log
    FOR EACH STATEMENT EXECUTE FUNCTION audit_log_reject_change();

GRANT SELECT, INSERT ON audit_log TO revioten_app;

-- +goose Down
DROP TABLE audit_log;
DROP FUNCTION audit_log_reject_change();
REVOKE ALL ON customers, people, accounts, sessions, permissions, roles, role_permissions, account_roles
    FROM revioten_app;
-- Role revioten_app tidak dihapus, karena bisa dipakai database lain di server yang sama.
