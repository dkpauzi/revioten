-- Form set langkah A: mesin AP (D-134, D-140).
--
-- Tabel Action Priority disimpan sebagai data di versi format template,
-- bukan sebagai konstanta di kode, supaya bisa dikoreksi lewat versi baru.
-- Bentuknya array JSON; setiap aturan memberi satu AP untuk satu kotak
-- rentang S, O, dan D:
--   {"s": [9, 10], "o": [4, 5], "d": [7, 10], "ap": "H"}
-- Layanan memeriksa bahwa tabel menutup 1.000 kombinasi S, O, D tepat satu
-- kali (internal/template/ap.go). Isi awalnya tabel PFMEA AIAG & VDA FMEA
-- Handbook 2019 dari paket desain pemilik (belum dicocokkan Claude Code
-- dengan bukunya).
--
-- Kolom ini bagian dari baris versi, jadi trigger migrasi 00016 sudah
-- menolak perubahannya setelah versi terbit.

-- +goose Up
ALTER TABLE format_template_versions
    ADD COLUMN ap_rules jsonb,
    ADD CONSTRAINT format_template_versions_ap_rules_check
        CHECK (ap_rules IS NULL OR (risk_method = 'ap' AND jsonb_typeof(ap_rules) = 'array'));

-- Versi dengan metode risiko AP yang sudah ada mendapat tabel bawaan. Versi
-- yang sudah terbit ikut diisi, karena sebelum migrasi ini versi itu belum
-- punya tabel sama sekali; trigger penolak dimatikan sebentar untuk itu
-- (lihat catatan di kepala migrasi 00016).
ALTER TABLE format_template_versions DISABLE TRIGGER format_template_versions_published;
UPDATE format_template_versions
SET ap_rules = '[{"s":[9,10],"o":[8,10],"d":[7,10],"ap":"H"},{"s":[9,10],"o":[8,10],"d":[5,6],"ap":"H"},{"s":[9,10],"o":[8,10],"d":[2,4],"ap":"H"},{"s":[9,10],"o":[8,10],"d":[1,1],"ap":"H"},{"s":[9,10],"o":[6,7],"d":[7,10],"ap":"H"},{"s":[9,10],"o":[6,7],"d":[5,6],"ap":"H"},{"s":[9,10],"o":[6,7],"d":[2,4],"ap":"H"},{"s":[9,10],"o":[6,7],"d":[1,1],"ap":"H"},{"s":[9,10],"o":[4,5],"d":[7,10],"ap":"H"},{"s":[9,10],"o":[4,5],"d":[5,6],"ap":"H"},{"s":[9,10],"o":[4,5],"d":[2,4],"ap":"H"},{"s":[9,10],"o":[4,5],"d":[1,1],"ap":"M"},{"s":[9,10],"o":[2,3],"d":[7,10],"ap":"H"},{"s":[9,10],"o":[2,3],"d":[5,6],"ap":"M"},{"s":[9,10],"o":[2,3],"d":[2,4],"ap":"L"},{"s":[9,10],"o":[2,3],"d":[1,1],"ap":"L"},{"s":[9,10],"o":[1,1],"d":[7,10],"ap":"L"},{"s":[9,10],"o":[1,1],"d":[5,6],"ap":"L"},{"s":[9,10],"o":[1,1],"d":[2,4],"ap":"L"},{"s":[9,10],"o":[1,1],"d":[1,1],"ap":"L"},{"s":[7,8],"o":[8,10],"d":[7,10],"ap":"H"},{"s":[7,8],"o":[8,10],"d":[5,6],"ap":"H"},{"s":[7,8],"o":[8,10],"d":[2,4],"ap":"H"},{"s":[7,8],"o":[8,10],"d":[1,1],"ap":"H"},{"s":[7,8],"o":[6,7],"d":[7,10],"ap":"H"},{"s":[7,8],"o":[6,7],"d":[5,6],"ap":"H"},{"s":[7,8],"o":[6,7],"d":[2,4],"ap":"H"},{"s":[7,8],"o":[6,7],"d":[1,1],"ap":"M"},{"s":[7,8],"o":[4,5],"d":[7,10],"ap":"H"},{"s":[7,8],"o":[4,5],"d":[5,6],"ap":"M"},{"s":[7,8],"o":[4,5],"d":[2,4],"ap":"M"},{"s":[7,8],"o":[4,5],"d":[1,1],"ap":"M"},{"s":[7,8],"o":[2,3],"d":[7,10],"ap":"M"},{"s":[7,8],"o":[2,3],"d":[5,6],"ap":"M"},{"s":[7,8],"o":[2,3],"d":[2,4],"ap":"L"},{"s":[7,8],"o":[2,3],"d":[1,1],"ap":"L"},{"s":[7,8],"o":[1,1],"d":[7,10],"ap":"L"},{"s":[7,8],"o":[1,1],"d":[5,6],"ap":"L"},{"s":[7,8],"o":[1,1],"d":[2,4],"ap":"L"},{"s":[7,8],"o":[1,1],"d":[1,1],"ap":"L"},{"s":[4,6],"o":[8,10],"d":[7,10],"ap":"H"},{"s":[4,6],"o":[8,10],"d":[5,6],"ap":"H"},{"s":[4,6],"o":[8,10],"d":[2,4],"ap":"M"},{"s":[4,6],"o":[8,10],"d":[1,1],"ap":"M"},{"s":[4,6],"o":[6,7],"d":[7,10],"ap":"M"},{"s":[4,6],"o":[6,7],"d":[5,6],"ap":"M"},{"s":[4,6],"o":[6,7],"d":[2,4],"ap":"M"},{"s":[4,6],"o":[6,7],"d":[1,1],"ap":"L"},{"s":[4,6],"o":[4,5],"d":[7,10],"ap":"M"},{"s":[4,6],"o":[4,5],"d":[5,6],"ap":"L"},{"s":[4,6],"o":[4,5],"d":[2,4],"ap":"L"},{"s":[4,6],"o":[4,5],"d":[1,1],"ap":"L"},{"s":[4,6],"o":[2,3],"d":[7,10],"ap":"L"},{"s":[4,6],"o":[2,3],"d":[5,6],"ap":"L"},{"s":[4,6],"o":[2,3],"d":[2,4],"ap":"L"},{"s":[4,6],"o":[2,3],"d":[1,1],"ap":"L"},{"s":[4,6],"o":[1,1],"d":[7,10],"ap":"L"},{"s":[4,6],"o":[1,1],"d":[5,6],"ap":"L"},{"s":[4,6],"o":[1,1],"d":[2,4],"ap":"L"},{"s":[4,6],"o":[1,1],"d":[1,1],"ap":"L"},{"s":[2,3],"o":[8,10],"d":[7,10],"ap":"M"},{"s":[2,3],"o":[8,10],"d":[5,6],"ap":"M"},{"s":[2,3],"o":[8,10],"d":[2,4],"ap":"L"},{"s":[2,3],"o":[8,10],"d":[1,1],"ap":"L"},{"s":[2,3],"o":[6,7],"d":[7,10],"ap":"L"},{"s":[2,3],"o":[6,7],"d":[5,6],"ap":"L"},{"s":[2,3],"o":[6,7],"d":[2,4],"ap":"L"},{"s":[2,3],"o":[6,7],"d":[1,1],"ap":"L"},{"s":[2,3],"o":[4,5],"d":[7,10],"ap":"L"},{"s":[2,3],"o":[4,5],"d":[5,6],"ap":"L"},{"s":[2,3],"o":[4,5],"d":[2,4],"ap":"L"},{"s":[2,3],"o":[4,5],"d":[1,1],"ap":"L"},{"s":[2,3],"o":[2,3],"d":[7,10],"ap":"L"},{"s":[2,3],"o":[2,3],"d":[5,6],"ap":"L"},{"s":[2,3],"o":[2,3],"d":[2,4],"ap":"L"},{"s":[2,3],"o":[2,3],"d":[1,1],"ap":"L"},{"s":[2,3],"o":[1,1],"d":[7,10],"ap":"L"},{"s":[2,3],"o":[1,1],"d":[5,6],"ap":"L"},{"s":[2,3],"o":[1,1],"d":[2,4],"ap":"L"},{"s":[2,3],"o":[1,1],"d":[1,1],"ap":"L"},{"s":[1,1],"o":[1,10],"d":[7,10],"ap":"L"},{"s":[1,1],"o":[1,10],"d":[5,6],"ap":"L"},{"s":[1,1],"o":[1,10],"d":[2,4],"ap":"L"},{"s":[1,1],"o":[1,10],"d":[1,1],"ap":"L"}]'::jsonb
WHERE risk_method = 'ap';
ALTER TABLE format_template_versions ENABLE TRIGGER format_template_versions_published;

GRANT UPDATE (ap_rules) ON format_template_versions TO revioten_app;

-- +goose Down
REVOKE UPDATE (ap_rules) ON format_template_versions FROM revioten_app;
ALTER TABLE format_template_versions DISABLE TRIGGER format_template_versions_published;
ALTER TABLE format_template_versions DROP CONSTRAINT format_template_versions_ap_rules_check;
ALTER TABLE format_template_versions DROP COLUMN ap_rules;
ALTER TABLE format_template_versions ENABLE TRIGGER format_template_versions_published;