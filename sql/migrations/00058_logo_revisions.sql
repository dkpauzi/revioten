-- D-223 (E-145 opsi a): ganti logo seperti revisi dokumen.
--
--   company_logos  setiap logo yang pernah dipakai, menurut SHA-256-nya.
--                  Revisi terbit menyimpan hash logo di snapshot, jadi
--                  cetakan revisi itu selalu memakai logonya sendiri. Logo
--                  bawaan (dibundel di binary) tidak disimpan di sini.
--   logo_changes   setiap penggantian logo, dengan jawaban admin apakah
--                  dokumen direvisi. Hash kosong berarti kembali ke logo
--                  bawaan.
--   changes        mekanisme baru "logo": satu perubahan per customer per
--                  penggantian logo (logo_change_id), mencakup semua dokumen
--                  aktif customer, seperti perubahan surut SQAM (M5.10).

-- +goose Up
CREATE TABLE company_logos (
    sha256 text PRIMARY KEY CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    png bytea NOT NULL CHECK (octet_length(png) <= 1048576),
    created_at timestamptz NOT NULL DEFAULT now(),
    created_by_person_id uuid REFERENCES people (id)
);
GRANT SELECT, INSERT ON company_logos TO revioten_app;

-- Logo unggahan yang sedang dipakai ikut masuk riwayat.
INSERT INTO company_logos (sha256, png, created_by_person_id)
SELECT company_logo_sha256, company_logo, company_logo_updated_by_person_id
FROM system_settings WHERE company_logo IS NOT NULL;

CREATE TABLE logo_changes (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    logo_sha256 text REFERENCES company_logos (sha256),
    revise boolean NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    created_by_person_id uuid REFERENCES people (id)
);
GRANT SELECT, INSERT ON logo_changes TO revioten_app;

ALTER TABLE changes DROP CONSTRAINT changes_mechanism_check;
ALTER TABLE changes ADD CONSTRAINT changes_mechanism_check
    CHECK (mechanism IN ('manual', 'yokoten', 'template', 'sqam', 'integration', 'logo'));
ALTER TABLE changes ADD COLUMN logo_change_id uuid REFERENCES logo_changes (id),
    ADD CONSTRAINT changes_logo_check CHECK ((mechanism = 'logo') = (logo_change_id IS NOT NULL));
-- Satu perubahan logo per customer per penggantian logo.
CREATE UNIQUE INDEX changes_logo_customer_key ON changes (logo_change_id, customer_id) WHERE logo_change_id IS NOT NULL;

-- +goose Down
DROP INDEX changes_logo_customer_key;
ALTER TABLE changes DROP CONSTRAINT changes_logo_check, DROP COLUMN logo_change_id;
ALTER TABLE changes DROP CONSTRAINT changes_mechanism_check;
ALTER TABLE changes ADD CONSTRAINT changes_mechanism_check
    CHECK (mechanism IN ('manual', 'yokoten', 'template', 'sqam', 'integration')) NOT VALID;
DROP TABLE logo_changes;
DROP TABLE company_logos;
