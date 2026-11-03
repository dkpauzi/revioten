-- Tinjauan desain UI dan UML draft 0.12 (3 November 2026): format template
-- M3.1 dibetulkan supaya sesuai aturan brief yang sudah ada. Desain UI
-- ("current form layout") memperlihatkan ketiganya.
--
--   1. Simbol Revision dipasang sistem pada baris yang berubah, dengan nomor
--      revisinya (R-REV-09: tanda berasal dari diff). Simbol ini tetap ada di
--      daftar simbol untuk legenda, tetapi tidak dipilih tangan di kolom simbol.
--      Kolom automatic menandainya.
--   2. Kolom Process Flow di CP dibentuk dari langkah PFD (R-DOC-09), jadi
--      bukan isian tangan.
--   3. Kolom Process Step / Function di PFMEA berisi nama proses baris itu.
--      Proses dan nomor langkah sudah menjadi atribut baris (ASUMSI E-70).
--
-- Nomor 2 dan 3 hanya menyembunyikan field di isi awal (versi 1 yang masih
-- draft), tidak menghapusnya (R-TPL-02). Admin tetap bisa memunculkannya lagi.

-- +goose Up
ALTER TABLE format_symbols ADD COLUMN automatic boolean NOT NULL DEFAULT false;
UPDATE format_symbols SET automatic = true WHERE key = 'revision';

UPDATE format_version_fields vf
SET hidden = true
FROM format_template_versions v
JOIN format_templates t ON t.id = v.template_id
WHERE vf.version_id = v.id
  AND v.version = 1 AND v.status = 'draft'
  AND vf.area = 'row' AND NOT vf.required
  AND ((t.name = 'CP' AND vf.label = 'Process Flow')
       OR (t.name = 'PFMEA AIAG 4th' AND vf.label = 'Process Step / Function'));

-- Hak role aplikasi tidak berubah. INSERT tingkat tabel ikut mencakup kolom
-- baru (salinan template membawa tandanya), dan UPDATE tetap hanya kolom
-- name, jadi aplikasi tidak bisa mengubah tanda automatic.

-- +goose Down
UPDATE format_version_fields vf
SET hidden = false
FROM format_template_versions v
JOIN format_templates t ON t.id = v.template_id
WHERE vf.version_id = v.id
  AND v.version = 1 AND v.status = 'draft'
  AND vf.area = 'row'
  AND ((t.name = 'CP' AND vf.label = 'Process Flow')
       OR (t.name = 'PFMEA AIAG 4th' AND vf.label = 'Process Step / Function'));

ALTER TABLE format_symbols DROP COLUMN automatic;
