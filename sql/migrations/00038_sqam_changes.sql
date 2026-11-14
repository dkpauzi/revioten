-- M5.10 · Perubahan surut SQAM (R-CHG-01, R-TPL-07).
--
-- Perubahan bermekanisme sqam selalu punya rujukan revisi SQAM (di
-- source_reference) dan tanggal berlaku (effective_date, sejak 00036).
-- Perubahan ini menunjuk semua dokumen aktif satu customer; saat diterima,
-- setiap dokumen mendapat draft baru (M5.4), jadi revisi terbit tidak pernah
-- ditulis ulang.

-- +goose Up
ALTER TABLE changes ADD CONSTRAINT changes_sqam_check CHECK (
    mechanism <> 'sqam' OR (source_reference IS NOT NULL AND effective_date IS NOT NULL));

-- +goose Down
ALTER TABLE changes DROP CONSTRAINT changes_sqam_check;
