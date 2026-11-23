-- I1 (D-224, E-143 opsi b): akun hasil import memakai password sementara
-- bersama yang kedaluwarsa 7 hari bila tidak dipakai. Kolom ini kosong untuk
-- akun lain, dan dikosongkan lagi saat password diganti atau direset admin.

-- +goose Up
ALTER TABLE accounts ADD COLUMN temporary_password_expires_at timestamptz;

-- +goose Down
ALTER TABLE accounts DROP COLUMN temporary_password_expires_at;
