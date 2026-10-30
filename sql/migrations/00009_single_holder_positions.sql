-- Jumlah pemegang jabatan (E-38 opsi b, D-115, 30 Oktober 2026). Jabatan
-- approver (NPI Manager, QA QC Manager, Advisor, Factory Manager) dan
-- APQP/PPAP Controller dipegang satu orang, karena satu slot mencetak satu
-- nama (D-76). Project Leader dan Project Engineer tetap satu per Project
-- Group (D-54, per_project_group), dan jabatan lain boleh punya beberapa
-- pemegang.
--
-- Aturan ini dijaga aplikasi: baris jabatan dikunci, lalu jumlah pemegangnya
-- diperiksa (internal/approval). Sengaja tanpa indeks unik di database: bila
-- database lama sudah punya dua pemegang untuk jabatan seperti ini, indeks
-- unik membuat migrasi gagal, dan server tidak bisa menyala untuk melepas
-- pemegang yang berlebih lewat layar Positions.

-- +goose Up
ALTER TABLE positions ADD COLUMN single_holder boolean NOT NULL DEFAULT false;
UPDATE positions SET single_holder = true
WHERE key IN ('apqp_ppap_controller', 'npi_manager', 'qa_qc_manager', 'advisor', 'factory_manager');
-- Satu jabatan tidak bisa sekaligus per Project Group dan satu pemegang.
ALTER TABLE positions ADD CONSTRAINT positions_holder_rule_check CHECK (NOT (per_project_group AND single_holder));

-- +goose Down
ALTER TABLE positions DROP CONSTRAINT positions_holder_rule_check;
ALTER TABLE positions DROP COLUMN single_holder;
