-- D-218: format template dan form set bisa disembunyikan (E-11).
--
-- Yang tersembunyi tidak ditawarkan untuk pilihan baru: form set customer,
-- sheet form set, dan sumber salinan. Daftar menampilkannya hanya bila
-- "Show hidden" dicentang. Dokumen yang sudah memakainya tidak berubah.
-- Set bawaan dan set yang dipilih customer tidak bisa disembunyikan (dijaga
-- layanan dan, untuk set bawaan, constraint di bawah).
--
-- Pekerjaan sekarang fokus ke form yang dipakai hari ini (tipe Tokairika),
-- jadi set AIAG-VDA beserta PFMEA AIAG-VDA 2019 dan Control Plan AIAG 2024
-- disembunyikan dulu. Format template juga diberi nama baru (nama hanya
-- tampilan; ID tetap, R-TPL-02).

-- +goose Up
ALTER TABLE format_templates ADD COLUMN hidden boolean NOT NULL DEFAULT false;
ALTER TABLE form_sets ADD COLUMN hidden boolean NOT NULL DEFAULT false;
ALTER TABLE form_sets ADD CONSTRAINT form_sets_default_visible CHECK (NOT (is_default AND hidden));

GRANT UPDATE (hidden) ON format_templates TO revioten_app;
GRANT UPDATE (hidden) ON form_sets TO revioten_app;

-- Nama baru untuk template bawaan. Template yang sudah diganti namanya
-- oleh pengguna tidak cocok dengan WHERE, jadi tidak disentuh.
UPDATE format_templates SET name = 'PFD v1' WHERE name = 'PFD' AND customer_id IS NULL;
UPDATE format_templates SET name = 'Control Plan v1' WHERE name = 'CP' AND customer_id IS NULL;
UPDATE format_templates SET name = 'Control Plan v2' WHERE name = 'Control Plan AIAG 2024' AND customer_id IS NULL;
UPDATE format_templates SET name = 'PFMEA v1' WHERE name = 'PFMEA AIAG 4th' AND customer_id IS NULL;
UPDATE format_templates SET name = 'PFMEA v2' WHERE name = 'PFMEA AIAG-VDA 2019' AND customer_id IS NULL;

UPDATE format_templates SET hidden = true WHERE name IN ('PFMEA v2', 'Control Plan v2') AND customer_id IS NULL;
-- Set AIAG-VDA hanya disembunyikan bila belum dipilih customer mana pun.
UPDATE form_sets SET hidden = true
WHERE key = 'aiag_vda'
  AND NOT EXISTS (SELECT 1 FROM customers c WHERE c.default_form_set_id = form_sets.id);

-- +goose Down
UPDATE format_templates SET name = 'PFD' WHERE name = 'PFD v1' AND customer_id IS NULL;
UPDATE format_templates SET name = 'CP' WHERE name = 'Control Plan v1' AND customer_id IS NULL;
UPDATE format_templates SET name = 'Control Plan AIAG 2024' WHERE name = 'Control Plan v2' AND customer_id IS NULL;
UPDATE format_templates SET name = 'PFMEA AIAG 4th' WHERE name = 'PFMEA v1' AND customer_id IS NULL;
UPDATE format_templates SET name = 'PFMEA AIAG-VDA 2019' WHERE name = 'PFMEA v2' AND customer_id IS NULL;
REVOKE UPDATE (hidden) ON form_sets FROM revioten_app;
REVOKE UPDATE (hidden) ON format_templates FROM revioten_app;
ALTER TABLE form_sets DROP CONSTRAINT form_sets_default_visible;
ALTER TABLE form_sets DROP COLUMN hidden;
ALTER TABLE format_templates DROP COLUMN hidden;
